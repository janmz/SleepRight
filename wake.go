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

	// Try to get sleep study (requires admin)
	cmd = exec.Command("powercfg", "/sleepstudy")
	output, err = cmd.Output()
	if err != nil {
		if verboseFlag {
			fmt.Printf("Note: Could not retrieve sleep study (may require admin rights): %v\n", err)
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
					fmt.Printf("System Statistics: %s\n", strings.TrimSpace(line))
					break
				}
			}
		}
	}

	return nil
}

