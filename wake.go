package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// WakeEvent represents a wake event from Windows
type WakeEvent struct {
	Timestamp time.Time
	Reason    string
	Device    string
	Source    string
}

func showWakeEvents() error {
	// Try to get wake events from powercfg
	cmd := exec.Command("powercfg", "/lastwake")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute powercfg /lastwake: %w", err)
	}

	outputStr := string(output)
	fmt.Println("Last Wake Event:")
	fmt.Println(outputStr)

	// Check if lastwake shows no results (common Windows 11 issue)
	if strings.Contains(outputStr, "Wake History Count - 0") ||
		strings.Contains(outputStr, "Wake Source Count - 0") {
		fmt.Println("\n⚠ Note: No wake source identified. This is a known Windows 11 issue.")
		fmt.Println("   The system may have been woken by hardware or internal timer.")
		fmt.Println("   Check the detailed diagnostics below for more information.")
	}

	// Show available sleep states (important for Modern Standby detection)
	if err := showAvailableSleepStates(); err != nil {
		if verboseFlag {
			fmt.Printf("Note: Could not retrieve available sleep states: %v\n", err)
		}
	}

	// Show wake timers (scheduled tasks that can wake the system)
	if err := showWakeTimers(); err != nil {
		if verboseFlag {
			fmt.Printf("Note: Could not retrieve wake timers: %v\n", err)
		}
	}

	// Show power requests (what prevents sleep)
	if err := showPowerRequests(); err != nil {
		if verboseFlag {
			fmt.Printf("Note: Could not retrieve power requests: %v\n", err)
		}
	}

	// Try to get sleep study (requires admin)
	cmd = exec.Command("powercfg", "/sleepstudy")
	output, err = cmd.Output()
	if err != nil {
		if verboseFlag {
			fmt.Printf("\nNote: Could not retrieve sleep study (may require admin rights): %v\n", err)
		}
	} else {
		// Parse sleep study output
		lines := strings.Split(string(output), "\n")
		eventCount := 0
		for _, line := range lines {
			if strings.Contains(line, "Wake Source") || strings.Contains(line, "Wake Time") {
				if eventCount < 5 { // Show last 5 events
					fmt.Println(line)
					eventCount++
				}
			}
		}
	}

	// Try to get system uptime
	cmd = exec.Command("net", "stats", "srv")
	output, err = cmd.Output()
	if err == nil {
		outputStr = string(output)
		// Parse uptime from net stats output
		if strings.Contains(outputStr, "Statistics since") {
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Statistics since") {
					fmt.Printf("\nSystem Statistics: %s\n", strings.TrimSpace(line))
					break
				}
			}
		}
	}

	return nil
}

// showAvailableSleepStates shows available sleep states (important for Modern Standby detection)
func showAvailableSleepStates() error {
	cmd := exec.Command("powercfg", "/a")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute powercfg /a: %w", err)
	}

	outputStr := string(output)
	fmt.Println("\n=== Available Sleep States ===")
	fmt.Println(outputStr)

	// Check for Modern Standby (S0 Low Power Idle)
	if strings.Contains(outputStr, "S0 Low Power Idle") || strings.Contains(outputStr, "S0 Niedriger Energieverbrauch") {
		fmt.Println("\n⚠ Warning: Modern Standby (S0 Low Power Idle) is active.")
		fmt.Println("   Your PC may never fully sleep and can wake unexpectedly.")
		fmt.Println("   This is common on modern laptops and can cause battery drain.")
	}

	return nil
}

// showWakeTimers shows scheduled tasks/timers that can wake the system
func showWakeTimers() error {
	cmd := exec.Command("powercfg", "/waketimers")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute powercfg /waketimers: %w", err)
	}

	outputStr := string(output)
	fmt.Println("\n=== Wake Timers (Scheduled Tasks) ===")
	fmt.Println(outputStr)

	// Check if there are active wake timers
	lines := strings.Split(outputStr, "\n")
	hasTimers := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "Timer set") &&
			!strings.Contains(line, "There are no active wake timers") &&
			!strings.Contains(line, "Es sind keine aktiven") {
			// Check if line contains a timer entry (usually has a timestamp or process name)
			if strings.Contains(line, ":") || strings.Contains(line, ".exe") {
				hasTimers = true
				break
			}
		}
	}

	if hasTimers {
		fmt.Println("\n⚠ Warning: Active wake timers found!")
		fmt.Println("   These scheduled tasks can wake your PC from sleep.")
		fmt.Println("   Common culprits: Windows Update (MoUsoCoreWorker.exe), maintenance tasks.")
		fmt.Println("   Solution: Disable 'Allow wake timers' in Power Options -> Advanced settings.")
	} else if strings.Contains(outputStr, "no active wake timers") ||
		strings.Contains(outputStr, "keine aktiven") {
		fmt.Println("✓ No active wake timers found.")
	}

	return nil
}

// showPowerRequests shows what prevents the system from entering sleep
func showPowerRequests() error {
	cmd := exec.Command("powercfg", "/requests")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute powercfg /requests: %w", err)
	}

	outputStr := string(output)
	fmt.Println("\n=== Power Requests (What Prevents Sleep) ===")
	fmt.Println(outputStr)

	// Check if there are active requests
	lines := strings.Split(outputStr, "\n")
	hasRequests := false
	requestTypes := []string{"DISPLAY", "SYSTEM", "AWAYMODE"}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, reqType := range requestTypes {
			if strings.Contains(line, reqType) &&
				!strings.Contains(line, "None") &&
				!strings.Contains(line, "Keine") &&
				line != "" {
				hasRequests = true
				break
			}
		}
		if hasRequests {
			break
		}
	}

	if hasRequests {
		fmt.Println("\n⚠ Warning: Active power requests found!")
		fmt.Println("   These applications or drivers are preventing sleep.")
		fmt.Println("   Common causes: Video streams, background processes, drivers.")
		fmt.Println("   Check the output above to identify the blocking application.")
	} else {
		fmt.Println("✓ No active power requests found. System can enter sleep normally.")
	}

	return nil
}
