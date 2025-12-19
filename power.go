package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yusufpapurcu/wmi"
)

func showPowerSettings(full bool) error {
	// Get current power scheme
	outputStr, err := runCommandWithEncoding("powercfg", "/getactivescheme")
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen des aktiven Energieschemas: %w", err)
	}
	printUTF8ln("Aktives Energieschema:")
	printUTF8ln(outputStr)

	// Get sleep timeout settings
	if err := showSleepSettings(); err != nil {
		return fmt.Errorf("Fehler beim Anzeigen der Ruhezustand-Einstellungen: %w", err)
	}

	// Get hibernate timeout settings
	if err := showHibernateSettings(); err != nil {
		return fmt.Errorf("Fehler beim Anzeigen der Ruhezustand-Einstellungen: %w", err)
	}

	// Get wake device settings
	if err := showWakeDeviceSettings(full); err != nil {
		return fmt.Errorf("Fehler beim Anzeigen der Aufweck-Geräte-Einstellungen: %w", err)
	}

	return nil
}

func showSleepSettings() error {
	outputStr, err := runCommandWithEncoding("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", "STANDBYIDLE")
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen der Ruhezustand-Einstellungen: %w", err)
	}

	printUTF8ln("\nRuhezustand-Einstellungen (Netzbetrieb):")
	parsePowerSetting(outputStr, "AC Setting Index")

	printUTF8ln("\nRuhezustand-Einstellungen (Batterie):")
	parsePowerSetting(outputStr, "DC Setting Index")

	return nil
}

func showHibernateSettings() error {
	outputStr, err := runCommandWithEncoding("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", "HIBERNATEIDLE")
	if err != nil {
		return fmt.Errorf("Fehler beim Abrufen der Ruhezustand-Einstellungen: %w", err)
	}

	printUTF8ln("\nRuhezustand-Einstellungen:")
	parsePowerSetting(outputStr, "Setting Index")
	return nil
}

// WMINetworkWakeInfo represents WMI information about network wake settings
// Field names must match WMI property names exactly (case-sensitive)
type WMINetworkWakeInfo struct {
	InstanceName                string `wmi:"InstanceName"`
	Active                      bool   `wmi:"Active"`
	EnableWakeOnMagicPacketOnly bool   `wmi:"EnableWakeOnMagicPacketOnly"`
}

func showWakeDeviceSettings(full bool) error {
	// Get all devices that are currently wake-armed
	armedOutput, err := runCommandWithEncoding("powercfg", "/devicequery", "wake_armed")
	if err != nil {
		return fmt.Errorf("failed to get wake-armed devices: %w", err)
	}

	// Get all devices that are wake-programmable (can potentially wake)
	programmableOutput, err := runCommandWithEncoding("powercfg", "/devicequery", "wake_programmable")
	if err != nil {
		return fmt.Errorf("failed to get wake-programmable devices: %w", err)
	}

	// Parse wake-armed devices into a map for quick lookup
	armedDevices := make(map[string]bool)
	armedLines := strings.Split(armedOutput, "\n")
	for _, line := range armedLines {
		line = strings.TrimSpace(line)
		if line != "" {
			armedDevices[line] = true
		}
	}

	// Parse wake-programmable devices
	programmableLines := strings.Split(programmableOutput, "\n")
	var programmableDevices []string
	for _, line := range programmableLines {
		line = strings.TrimSpace(line)
		if line != "" {
			programmableDevices = append(programmableDevices, line)
		}
	}

	// Query WMI for network wake information (all devices, not just active ones)
	var networkWakeInfo []WMINetworkWakeInfo
	const NameSpace = "root\\wmi"

	err = wmi.QueryNamespace("SELECT InstanceName, Active, EnableWakeOnMagicPacketOnly FROM MSNdis_DeviceWakeOnMagicPacketOnly", &networkWakeInfo, NameSpace)
	if err != nil {
		// WMI query failed, continue without WMI info
		if verboseFlag {
			printUTF8ln("Note: Could not query WMI for network wake info: %v", err)
		}
	}

	type Win32_NetworkAdapter struct {
		Name        string
		PNPDeviceID string
	}
	var adapters []Win32_NetworkAdapter
	idToName := make(map[string]string)

	// Win32_NetworkAdapter liegt im Standard-Namespace root\cimv2
	err = wmi.Query("SELECT Name, PNPDeviceID FROM Win32_NetworkAdapter", &adapters)
	if err == nil {
		for _, a := range adapters {
			idToName[strings.ToLower(a.PNPDeviceID)] = a.Name
		}
	}
	// Create a map of network device names from WMI
	networkWakeMap := make(map[string]WMINetworkWakeInfo)
	for _, info := range networkWakeInfo {
		pnpID := strings.ToLower(info.InstanceName)
		if lastUnderscore := strings.LastIndex(pnpID, "_"); lastUnderscore > -1 {
			pnpID = pnpID[:lastUnderscore]
		}

		// Suche den Anzeigenamen basierend auf der ID
		friendlyName, found := idToName[pnpID]
		if !found {
			// Fallback: Falls keine exakte ID-Übereinstimmung, suche per Teilstring in der ID-Map
			for id, name := range idToName {
				if strings.Contains(pnpID, id) || strings.Contains(id, pnpID) {
					friendlyName = name
					found = true
					break
				}
			}
		}

		if found {
			// Speichere unter dem Namen, den powercfg verwendet (normalisiert)
			cleanName := strings.ToLower(strings.TrimSpace(friendlyName))
			networkWakeMap[cleanName] = info

			if verboseFlag {
				printUTF8ln("Mapped WMI ID to Name: %s", friendlyName)
			}
		}
	}

	// Helper function to find WMI info for a device
	findWMIInfo := func(deviceName string) (WMINetworkWakeInfo, bool) {
		// Try exact match first
		if info, found := networkWakeMap[deviceName]; found {
			return info, true
		}
		// Try partial matching
		deviceLower := strings.ToLower(deviceName)
		for wmiName, wmiInfo := range networkWakeMap {
			if strings.Contains(deviceLower, strings.ToLower(wmiName)) ||
				strings.Contains(strings.ToLower(wmiName), deviceLower) {
				return wmiInfo, true
			}
		}
		return WMINetworkWakeInfo{}, false
	}

	// Separate devices into enabled and disabled
	var enabledDevices []string
	var disabledDevices []string
	for _, device := range programmableDevices {
		if armedDevices[device] {
			enabledDevices = append(enabledDevices, device)
		} else {
			disabledDevices = append(disabledDevices, device)
		}
	}

	// Display results
	printUTF8ln("\n=== Aufweck-Geräte-Analyse ===")
	if full {
		printUTF8ln("\nGesamt aufweck-programmierbare Geräte: %d", len(programmableDevices))
		printUTF8ln("Aktuell aufweck-aktivierte Geräte: %d\n", len(armedDevices))
	} else {
		printUTF8ln("\nAktuell aufweck-aktivierte Geräte: %d\n", len(armedDevices))
	}

	// Display enabled devices first
	if len(enabledDevices) > 0 {
		printUTF8ln("Aktivierte aufweck-programmierbare Geräte:")
		for i, device := range enabledDevices {
			printUTF8("  %d. %s", i+1, device)

			// Check if this is a network device with WMI info
			if wmiInfo, found := findWMIInfo(device); found {
				if wmiInfo.EnableWakeOnMagicPacketOnly {
					printUTF8(" - Magic-Packet: Aktiviert")
				} else {
					printUTF8(" - Magic-Packet: Deaktiviert")
				}
			}
			printUTF8ln("")
		}
	}

	// Display disabled devices only in full mode
	if full && len(disabledDevices) > 0 {
		if len(enabledDevices) > 0 {
			printUTF8ln("")
		}
		printUTF8ln("Deaktivierte aufweck-programmierbare Geräte:")
		for i, device := range disabledDevices {
			printUTF8("  %d. %s", i+1, device)

			// Always show Magic-Packet status for disabled network devices
			if wmiInfo, found := findWMIInfo(device); found {
				if wmiInfo.EnableWakeOnMagicPacketOnly {
					printUTF8(" - Magic-Packet: Aktiviert")
				} else {
					printUTF8(" - Magic-Packet: Deaktiviert")
				}
			}
			printUTF8ln("")
		}
	}

	if len(programmableDevices) == 0 {
		printUTF8ln("Keine aufweck-programmierbaren Geräte gefunden.")
	}

	return nil
}

func parsePowerSetting(output, searchKey string) {
	lines := strings.Split(output, "\n")

	// Determine if we're looking for AC or DC setting based on searchKey
	isAC := strings.Contains(searchKey, "AC") || strings.Contains(strings.ToUpper(searchKey), "AC")
	isDC := strings.Contains(searchKey, "DC") || strings.Contains(strings.ToUpper(searchKey), "DC")

	// German patterns: "Wechselstromeinstellung" (AC) and "Gleichstromeinstellung" (DC)
	// English patterns: "AC Setting Index" and "DC Setting Index"

	// Look for the German pattern first: "Index der aktuellen Wechselstromeinstellung" or "Index der aktuellen Gleichstromeinstellung"
	var targetPattern string
	if isAC {
		targetPattern = "Wechselstromeinstellung"
	} else if isDC {
		targetPattern = "Gleichstromeinstellung"
	} else {
		// For hibernate, look for any "Index der aktuellen" line
		targetPattern = "Index der aktuellen"
	}

	// Search for the pattern and extract hex value
	for i, line := range lines {
		if strings.Contains(line, targetPattern) {
			// Look for hex value in this line or next few lines
			for j := i; j < len(lines) && j < i+3; j++ {
				currentLine := lines[j]

				// Look for hex pattern: "0x00003840" (without parentheses in German output)
				re := regexp.MustCompile(`0x([0-9a-fA-F]+)`)
				matches := re.FindStringSubmatch(currentLine)
				if len(matches) > 1 {
					// Parse hex value
					value, err := strconv.ParseInt(matches[1], 16, 64)
					if err == nil {
						// Value is in seconds, use formatDuration for consistent formatting
						seconds := int(value)
						if seconds == 0 {
							printUTF8ln("  Timeout: Deaktiviert")
						} else {
							duration := time.Duration(seconds) * time.Second
							timeoutStr := formatDuration(duration)
							printUTF8ln("  Timeout: %s", timeoutStr)
						}
						return
					}
				}

				// Also try pattern with parentheses: "0x00000708 (1800)"
				re2 := regexp.MustCompile(`0x[0-9a-fA-F]+\s*\((\d+)\)`)
				matches2 := re2.FindStringSubmatch(currentLine)
				if len(matches2) > 1 {
					value, err := strconv.Atoi(matches2[1])
					if err == nil {
						if value == 0 {
							printUTF8ln("  Timeout: Deaktiviert")
						} else {
							duration := time.Duration(value) * time.Second
							timeoutStr := formatDuration(duration)
							printUTF8ln("  Timeout: %s", timeoutStr)
						}
						return
					}
				}
			}
		}
	}

	// Fallback: Try English patterns
	searchPatterns := []string{
		"Current AC Power Setting Index",
		"Current DC Power Setting Index",
		"AC Setting Index",
		"DC Setting Index",
		"Setting Index",
	}

	for _, pattern := range searchPatterns {
		if isAC && strings.Contains(pattern, "DC") {
			continue
		}
		if isDC && strings.Contains(pattern, "AC") {
			continue
		}

		for i, line := range lines {
			if strings.Contains(line, pattern) {
				for j := i; j < len(lines) && j < i+10; j++ {
					currentLine := lines[j]

					// Look for hex pattern
					re := regexp.MustCompile(`0x([0-9a-fA-F]+)`)
					matches := re.FindStringSubmatch(currentLine)
					if len(matches) > 1 {
						value, err := strconv.ParseInt(matches[1], 16, 64)
						if err == nil {
							seconds := int(value)
							if seconds == 0 {
								printUTF8ln("  Timeout: Deaktiviert")
							} else {
								duration := time.Duration(seconds) * time.Second
								timeoutStr := formatDuration(duration)
								printUTF8ln("  Timeout: %s", timeoutStr)
							}
							return
						}
					}

					// Try pattern with parentheses
					re2 := regexp.MustCompile(`0x[0-9a-fA-F]+\s*\((\d+)\)`)
					matches2 := re2.FindStringSubmatch(currentLine)
					if len(matches2) > 1 {
						value, err := strconv.Atoi(matches2[1])
						if err == nil && value >= 0 && value <= 86400 {
							if value == 0 {
								printUTF8ln("  Timeout: Deaktiviert")
							} else {
								duration := time.Duration(value) * time.Second
								timeoutStr := formatDuration(duration)
								printUTF8ln("  Timeout: %s", timeoutStr)
							}
							return
						}
					}
				}
			}
		}
	}

	// If still not found, show debug info or default message
	if debugFlag {
		printUTF8ln("  Einstellung nicht gefunden (Suchschlüssel: %s)", searchKey)
		printUTF8ln("  Erste 30 Zeilen der Ausgabe zum Debuggen:")
		for i, line := range lines {
			if i >= 30 {
				break
			}
			printUTF8ln("    %d: %s", i, strings.TrimSpace(line))
		}
	} else {
		printUTF8ln("  Timeout: Nicht konfiguriert oder nicht verfügbar")
	}
}

func configureWakeDevices() error {
	fmt.Println("Configuring wake devices...")

	// First, disable all wake devices
	cmd := exec.Command("powercfg", "/devicequery", "wake_armed")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				// Disable wake for this device
				cmd := exec.Command("powercfg", "/devicedisablewake", line)
				if err := cmd.Run(); err != nil {
					if verboseFlag {
						fmt.Printf("  Warning: Could not disable wake for %s: %v\n", line, err)
					}
				} else {
					if verboseFlag {
						fmt.Printf("  Disabled wake for: %s\n", line)
					}
				}
			}
		}
	}

	// Enable wake for keyboard
	fmt.Println("  Enabling wake for keyboard...")
	cmd = exec.Command("powercfg", "/deviceenablewake", "\"HID Keyboard Device\"")
	if err := cmd.Run(); err != nil {
		// Try alternative names
		cmd = exec.Command("powercfg", "/deviceenablewake", "\"Standard PS/2 Keyboard\"")
		if err := cmd.Run(); err != nil {
			// Try to find keyboard device
			cmd = exec.Command("powercfg", "/devicequery", "wake_programmable")
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.Contains(strings.ToLower(line), "keyboard") {
						cmd = exec.Command("powercfg", "/deviceenablewake", line)
						cmd.Run()
						break
					}
				}
			}
		}
	}

	// Enable wake for Ethernet adapter
	fmt.Println("  Enabling wake for Ethernet adapter...")
	cmd = exec.Command("powercfg", "/devicequery", "wake_programmable")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(strings.ToLower(line), "ethernet") ||
				strings.Contains(strings.ToLower(line), "network") ||
				strings.Contains(strings.ToLower(line), "realtek") ||
				strings.Contains(strings.ToLower(line), "intel") {
				cmd = exec.Command("powercfg", "/deviceenablewake", line)
				if err := cmd.Run(); err != nil {
					if verboseFlag {
						fmt.Printf("    Warning: Could not enable wake for %s: %v\n", line, err)
					}
				} else {
					fmt.Printf("    Enabled wake for: %s\n", line)
				}
			}
		}
	}

	fmt.Println("  Wake device configuration completed.")
	return nil
}

func configureSleepTimeout() error {
	fmt.Println("Configuring sleep timeout to 30 minutes...")

	// Set sleep timeout to 30 minutes (1800 seconds) for AC power
	cmd := exec.Command("powercfg", "/change", "standby-timeout-ac", "30")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set AC sleep timeout: %w", err)
	}

	// Set sleep timeout to 30 minutes for battery
	cmd = exec.Command("powercfg", "/change", "standby-timeout-dc", "30")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set battery sleep timeout: %w", err)
	}

	fmt.Println("  Sleep timeout set to 30 minutes.")
	return nil
}

func configurePowerScheme() error {
	fmt.Println("Configuring power scheme to Balanced...")

	// Get available power schemes
	cmd := exec.Command("powercfg", "/list")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list power schemes: %w", err)
	}

	// Find Balanced scheme GUID
	lines := strings.Split(string(output), "\n")
	var balancedGUID string
	for _, line := range lines {
		if strings.Contains(line, "Balanced") {
			// Extract GUID (format: * GUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
			re := regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				balancedGUID = matches[1]
				break
			}
		}
	}

	if balancedGUID == "" {
		return fmt.Errorf("could not find Balanced power scheme")
	}

	// Set active scheme to Balanced
	cmd = exec.Command("powercfg", "/setactive", balancedGUID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set active power scheme: %w", err)
	}

	fmt.Printf("  Power scheme set to Balanced (%s).\n", balancedGUID)
	return nil
}

func configureHibernateTimeout(minutes int) error {
	fmt.Printf("Configuring hibernate timeout to %d minutes...\n", minutes)

	// Set hibernate timeout for AC power (powercfg expects minutes)
	cmd := exec.Command("powercfg", "/change", "hibernate-timeout-ac", strconv.Itoa(minutes))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set AC hibernate timeout: %w", err)
	}

	// Set hibernate timeout for battery (powercfg expects minutes)
	cmd = exec.Command("powercfg", "/change", "hibernate-timeout-dc", strconv.Itoa(minutes))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set battery hibernate timeout: %w", err)
	}

	fmt.Printf("  Hibernate timeout set to %d minutes.\n", minutes)
	return nil
}

// configureWakeTimers disables wake timers to prevent scheduled tasks from waking the system
func configureWakeTimers() error {
	fmt.Println("Configuring wake timers (disabling scheduled task wake-ups)...")

	// Get current power scheme GUID
	cmd := exec.Command("powercfg", "/getactivescheme")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active power scheme: %w", err)
	}

	// Extract GUID from output
	re := regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return fmt.Errorf("could not extract power scheme GUID")
	}
	schemeGUID := matches[1]

	// Disable wake timers for AC power
	// Setting ID: 238c9fa8-0aad-41ed-83f4-97be242c8f20 (Sleep)
	// Subgroup ID: 29f6c1db-86da-48c5-9fdb-f2b67b1f44da (Sleep)
	// Setting ID: bd3b718a-0680-4d9d-8ab2-e1d2b4ac806d (Allow wake timers)
	// Value: 0 = Disabled, 1 = Enabled (AC), 2 = Enabled (DC), 3 = Enabled (Both)

	// For AC power: set to 0 (Disabled)
	cmd = exec.Command("powercfg", "/setacvalueindex", schemeGUID,
		"238c9fa8-0aad-41ed-83f4-97be242c8f20",
		"29f6c1db-86da-48c5-9fdb-f2b67b1f44da",
		"bd3b718a-0680-4d9d-8ab2-e1d2b4ac806d", "0")
	if err := cmd.Run(); err != nil {
		if verboseFlag {
			fmt.Printf("  Warning: Could not disable AC wake timers: %v\n", err)
		}
	}

	// For DC (battery) power: set to 0 (Disabled)
	cmd = exec.Command("powercfg", "/setdcvalueindex", schemeGUID,
		"238c9fa8-0aad-41ed-83f4-97be242c8f20",
		"29f6c1db-86da-48c5-9fdb-f2b67b1f44da",
		"bd3b718a-0680-4d9d-8ab2-e1d2b4ac806d", "0")
	if err := cmd.Run(); err != nil {
		if verboseFlag {
			fmt.Printf("  Warning: Could not disable DC wake timers: %v\n", err)
		}
	}

	// Apply the changes
	cmd = exec.Command("powercfg", "/setactive", schemeGUID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply wake timer settings: %w", err)
	}

	fmt.Println("  Wake timers disabled (scheduled tasks will not wake the system).")
	return nil
}
