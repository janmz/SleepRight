package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func showPowerSettings() error {
	// Get current power scheme
	cmd := exec.Command("powercfg", "/getactivescheme")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active power scheme: %w", err)
	}
	fmt.Printf("Active Power Scheme:\n%s\n", string(output))

	// Get sleep timeout settings
	if err := showSleepSettings(); err != nil {
		return fmt.Errorf("failed to show sleep settings: %w", err)
	}

	// Get hibernate timeout settings
	if err := showHibernateSettings(); err != nil {
		return fmt.Errorf("failed to show hibernate settings: %w", err)
	}

	// Get wake device settings
	if err := showWakeDeviceSettings(); err != nil {
		return fmt.Errorf("failed to show wake device settings: %w", err)
	}

	return nil
}

func showSleepSettings() error {
	cmd := exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", "STANDBYIDLE")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get sleep settings: %w", err)
	}

	fmt.Println("\nSleep Settings (AC Power):")
	parsePowerSetting(string(output), "AC Setting Index")

	cmd = exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", "STANDBYIDLE")
	output, err = cmd.Output()
	if err == nil {
		fmt.Println("\nSleep Settings (Battery):")
		parsePowerSetting(string(output), "DC Setting Index")
	}

	return nil
}

func showHibernateSettings() error {
	cmd := exec.Command("powercfg", "/query", "SCHEME_CURRENT", "SUB_SLEEP", "HIBERNATEIDLE")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get hibernate settings: %w", err)
	}

	fmt.Println("\nHibernate Settings:")
	parsePowerSetting(string(output), "Setting Index")
	return nil
}

func showWakeDeviceSettings() error {
	cmd := exec.Command("powercfg", "/devicequery", "wake_armed")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get wake device settings: %w", err)
	}

	fmt.Println("\nDevices that can wake the computer:")
	lines := strings.Split(string(output), "\n")
	deviceCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Printf("  - %s\n", line)
			deviceCount++
		}
	}
	if deviceCount == 0 {
		fmt.Println("  (none)")
	}

	return nil
}

func parsePowerSetting(output, searchKey string) {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, searchKey) {
			// Try to extract the value
			re := regexp.MustCompile(`(\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				value, err := strconv.Atoi(matches[1])
				if err == nil {
					// Convert to minutes (value is in seconds)
					minutes := value / 60
					if minutes > 0 {
						fmt.Printf("  Timeout: %d minutes\n", minutes)
					} else {
						fmt.Printf("  Timeout: %d seconds (disabled)\n", value)
					}
				}
			}
			// Show next few lines for context
			for j := i + 1; j < len(lines) && j < i+3; j++ {
				if strings.TrimSpace(lines[j]) != "" {
					fmt.Printf("  %s\n", strings.TrimSpace(lines[j]))
				}
			}
			break
		}
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

