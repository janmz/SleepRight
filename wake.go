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

func showWakeEvents(full bool) error {
	// Try to get wake events from powercfg
	outputStr, err := runCommandWithEncoding("powercfg", "/lastwake")
	if err != nil {
		return fmt.Errorf("Fehler beim Ausführen von powercfg /lastwake: %w", err)
	}

	printUTF8ln("Letztes Aufweck-Ereignis:")
	printUTF8ln(outputStr)

	// Check if lastwake shows no results (common Windows 11 issue)
	if strings.Contains(outputStr, "Wake History Count - 0") ||
		strings.Contains(outputStr, "Wake Source Count - 0") {
		if full {
			printUTF8ln("Hinweis: Keine Aufweckquelle identifiziert. Dies ist ein bekanntes Windows 11 Problem. Das System wurde möglicherweise durch Hardware oder internen Timer aufgeweckt.")
		}
	}

	// Show available sleep states (important for Modern Standby detection)
	if err := showAvailableSleepStates(full); err != nil {
		if verboseFlag {
			printUTF8ln("Hinweis: Konnte verfügbare Standby-Zustände nicht abrufen: %v", err)
		}
	}

	// Show wake timers (scheduled tasks that can wake the system)
	if err := showWakeTimers(full); err != nil {
		if verboseFlag {
			printUTF8ln("Hinweis: Konnte Aufweck-Zeitgeber nicht abrufen: %v", err)
		}
	}

	// Show power requests (what prevents sleep)
	if err := showPowerRequests(full); err != nil {
		if verboseFlag {
			printUTF8ln("Hinweis: Konnte Energieanfragen nicht abrufen: %v", err)
		}
	}

	// Read Windows Event Log for Power-Troubleshooter events
	if err := showEventLogWakeEvents(full); err != nil {
		if verboseFlag {
			printUTF8ln("Hinweis: Konnte Ereignisprotokoll nicht lesen: %v", err)
		}
	}

	// Try to get sleep study (requires admin)
	sleepStudyOutput, err := runCommandWithEncoding("powercfg", "/sleepstudy")
	if err != nil {
		if verboseFlag {
			fmt.Printf("\nNote: Could not retrieve sleep study (may require admin rights): %v\n", err)
		}
	} else {
		// Parse sleep study output
		lines := strings.Split(sleepStudyOutput, "\n")
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
	uptimeOutput, err := runCommandWithEncoding("net", "stats", "srv")
	if err == nil {
		// Parse uptime from net stats output
		if strings.Contains(uptimeOutput, "Statistics since") {
			lines := strings.Split(uptimeOutput, "\n")
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
func showAvailableSleepStates(full bool) error {
	outputStr, err := runCommandWithEncoding("powercfg", "/a")
	if err != nil {
		return fmt.Errorf("Fehler beim Ausführen von powercfg /a: %w", err)
	}

	printUTF8ln("\n=== Verfügbare Standby-Zustände ===")
	
	// Filter output based on full flag
	lines := strings.Split(outputStr, "\n")
	var outputLines []string
	inUnavailable := false
	
	for _, line := range lines {
		// Check if we're entering unavailable section
		if strings.Contains(line, "nicht verfügbar") || strings.Contains(line, "not available") {
			if !full {
				break // Stop here if not full mode
			}
			inUnavailable = true
		}
		
		// Check if we're in available section
		if strings.Contains(line, "verfügbar") || strings.Contains(line, "available") {
			inUnavailable = false
		}
		
		if !inUnavailable || full {
			outputLines = append(outputLines, line)
		}
	}
	
	printUTF8ln(strings.Join(outputLines, "\n"))

	// Check for Modern Standby (S0 Low Power Idle)
	if strings.Contains(outputStr, "S0 Low Power Idle") || strings.Contains(outputStr, "S0 Niedriger Energieverbrauch") {
		printUTF8ln("Warnung: Modern Standby (S0 Low Power Idle) ist aktiv. Der PC schläft möglicherweise nie vollständig und kann unerwartet aufwachen.")
	}

	return nil
}

// showWakeTimers shows scheduled tasks/timers that can wake the system
func showWakeTimers(full bool) error {
	outputStr, err := runCommandWithEncoding("powercfg", "/waketimers")
	if err != nil {
		return fmt.Errorf("Fehler beim Ausführen von powercfg /waketimers: %w", err)
	}

	printUTF8ln("\n=== Aufweck-Zeitgeber (Geplante Aufgaben) ===")
	printUTF8ln(outputStr)

	// Check if there are active wake timers
	hasTimers := strings.Contains(outputStr, "Ursache:") || strings.Contains(outputStr, "Cause:")
	if hasTimers {
		printUTF8ln("Warnung: Aktive Aufweck-Zeitgeber gefunden! Diese geplanten Aufgaben können den PC aus dem Ruhemodus wecken.")
	} else if full {
		printUTF8ln("Keine aktiven Aufweck-Zeitgeber gefunden.")
	}

	return nil
}

// showPowerRequests shows what prevents the system from entering sleep
func showPowerRequests(full bool) error {
	outputStr, err := runCommandWithEncoding("powercfg", "/requests")
	if err != nil {
		return fmt.Errorf("Fehler beim Ausführen von powercfg /requests: %w", err)
	}

	// Parse and filter output
	lines := strings.Split(outputStr, "\n")
	var filteredLines []string
	hasRequests := false
	requestTypes := []string{"DISPLAY", "SYSTEM", "AWAYMODE", "AUSFÜHRUNG", "PERFBOOST", "ACTIVELOCKSCREEN"}
	
	currentRequestType := ""
	
	for i, line := range lines {
		lineTrimmed := strings.TrimSpace(line)
		
		// Check if this is a request type header
		isRequestType := false
		for _, reqType := range requestTypes {
			if strings.HasPrefix(lineTrimmed, reqType+":") || strings.HasPrefix(lineTrimmed, reqType+" ") {
				currentRequestType = reqType
				isRequestType = true
				break
			}
		}
		
		if isRequestType {
			// Check next few lines for actual requests
			hasContent := false
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				// Check if it's another request type
				isNextRequestType := false
				for _, rt := range requestTypes {
					if strings.HasPrefix(nextLine, rt+":") || strings.HasPrefix(nextLine, rt+" ") {
						isNextRequestType = true
						break
					}
				}
				if isNextRequestType {
					break
				}
				// If not "None" or "Keine", it's actual content
				if !strings.Contains(nextLine, "None") && !strings.Contains(nextLine, "Keine") && nextLine != "" {
					hasContent = true
					hasRequests = true
					break
				}
			}
			
			// Only include this section if it has content or we're in full mode
			if hasContent || full {
				filteredLines = append(filteredLines, line)
			}
		} else if currentRequestType != "" {
			// This is content for the current request type
			// Only include if we're including this request type
			if full || (!strings.Contains(lineTrimmed, "None") && !strings.Contains(lineTrimmed, "Keine") && lineTrimmed != "") {
				filteredLines = append(filteredLines, line)
			}
		} else {
			// Header or other content
			filteredLines = append(filteredLines, line)
		}
	}
	
	if len(filteredLines) > 0 || full {
		printUTF8ln("\n=== Energieanfragen (Was verhindert Ruhezustand) ===")
		if len(filteredLines) > 0 {
			printUTF8ln(strings.Join(filteredLines, "\n"))
		}
		
		if hasRequests {
			printUTF8ln("Warnung: Aktive Energieanfragen gefunden! Diese Anwendungen oder Treiber verhindern den Ruhezustand.")
		} else if full {
			printUTF8ln("Keine aktiven Energieanfragen gefunden. Das System kann normal in den Ruhezustand gehen.")
		}
	}

	return nil
}

// showEventLogWakeEvents reads Windows Event Log for Power-Troubleshooter events
func showEventLogWakeEvents(full bool) error {
	printUTF8ln("\n=== Ereignisprotokoll-Analyse (Power-Troubleshooter) ===")

	// Query Event Log for Power-Troubleshooter events (Event ID 1)
	// This shows wake source information
	// Increase limit to get more events for better sorting
	cmd := exec.Command("wevtutil", "qe", "System", "/q:*[System[Provider[@Name='Microsoft-Windows-Power-Troubleshooter']]]", "/f:text", "/c:20", "/rd:true")
	output, err := cmd.Output()
	if err != nil {
		// If wevtutil fails, try alternative method
		return showEventLogAlternative()
	}

	outputStr := string(output) // Keep in Windows codepage, do not convert

	// Parse and display relevant wake events in a clean format
	lines := strings.Split(outputStr, "\n")
	type WakeEvent struct {
		sleepTime time.Time // Zeit im Energiesparmodus
		wakeTime  time.Time // Reaktivierungszeit
		source    string    // Reaktivierungsquelle
	}
	var events []WakeEvent

	// Parse events - look for both sleep time and wake time in the same event block
	// Events are separated by empty lines or "Event[" markers
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		
		// Look for "Reaktivierungszeit:" or "Wake Time:" to identify a wake event
		if strings.Contains(line, "Reaktivierungszeit:") || strings.Contains(line, "Wake Time:") {
			var sleepTime time.Time
			var wakeTime time.Time
			sourceStr := "Unknown"
			
			// Find the start of this event block (look for "Event[" or empty line before)
			eventStart := i
			for k := i; k >= 0 && k > i-30; k-- {
				if strings.Contains(lines[k], "Event[") || (k < i && strings.TrimSpace(lines[k]) == "") {
					eventStart = k + 1
					break
				}
			}
			
			// Find the end of this event block (look for next "Event[" or empty line)
			eventEnd := i + 30
			if eventEnd > len(lines) {
				eventEnd = len(lines)
			}
			for k := i + 1; k < len(lines) && k < i+30; k++ {
				if strings.Contains(lines[k], "Event[") || (k > i && strings.TrimSpace(lines[k]) == "" && strings.TrimSpace(lines[k-1]) != "") {
					eventEnd = k
					break
				}
			}
			
			// Parse this event block - look for sleep time and wake time
			for j := eventStart; j < eventEnd; j++ {
				currentLine := strings.TrimSpace(lines[j])
				
				// Look for "Zeit im Energiesparmodus:" or "Sleep Time:"
				if strings.Contains(currentLine, "Zeit im Energiesparmodus:") || strings.Contains(currentLine, "Sleep Time:") {
					timeStr := ""
					if idx := strings.Index(currentLine, ":"); idx >= 0 {
						timeStr = strings.TrimSpace(currentLine[idx+1:])
						// Remove special characters that might interfere
						timeStr = strings.ReplaceAll(timeStr, "?", "")
						timeStr = strings.ReplaceAll(timeStr, "‎", "") // Remove left-to-right mark
						timeStr = strings.TrimSpace(timeStr)
						
						// Try to parse the time - handle UTC times
						if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
							sleepTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z", timeStr); err == nil {
							sleepTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
							sleepTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
							// If no timezone, assume UTC
							sleepTime = t.UTC()
						}
					}
				}
				
				// Look for "Reaktivierungszeit:" or "Wake Time:"
				if strings.Contains(currentLine, "Reaktivierungszeit:") || strings.Contains(currentLine, "Wake Time:") {
					timeStr := ""
					if idx := strings.Index(currentLine, ":"); idx >= 0 {
						timeStr = strings.TrimSpace(currentLine[idx+1:])
						// Remove special characters that might interfere
						timeStr = strings.ReplaceAll(timeStr, "?", "")
						timeStr = strings.ReplaceAll(timeStr, "‎", "") // Remove left-to-right mark
						timeStr = strings.TrimSpace(timeStr)
						
						// Try to parse the time - handle UTC times
						if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
							wakeTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05.999999999Z", timeStr); err == nil {
							wakeTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
							wakeTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
							// If no timezone, assume UTC
							wakeTime = t.UTC()
						}
					}
				}
				
				// Look for "Reaktivierungsquelle:" or "Wake Source:"
				if strings.Contains(currentLine, "Reaktivierungsquelle:") || strings.Contains(currentLine, "Wake Source:") {
					if idx := strings.Index(currentLine, ":"); idx >= 0 {
						sourceStr = strings.TrimSpace(currentLine[idx+1:])
						sourceStr = strings.ReplaceAll(sourceStr, "?", "")
						sourceStr = strings.TrimSpace(sourceStr)
					}
				}
			}
			
			// Only add event if we have both sleep time and wake time from the same event
			if !sleepTime.IsZero() && !wakeTime.IsZero() {
				// Verify that wake time is after sleep time (sanity check)
				if wakeTime.After(sleepTime) {
					events = append(events, WakeEvent{
						sleepTime: sleepTime,
						wakeTime:  wakeTime,
						source:    sourceStr,
					})
				}
			}
		}
	}

	// Sort events by wake time (descending - newest first)
	// Simple bubble sort: if event[i] wake time is older than event[j] wake time, swap them
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i].wakeTime.Before(events[j].wakeTime) {
				// i is older than j, so swap to put j (newer) first
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// Filter events by time (only last 24 hours if not full)
	now := time.Now()
	var filteredEvents []WakeEvent
	for _, event := range events {
		if full || event.wakeTime.After(now.Add(-24*time.Hour)) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Display events in clean format with sleep duration
	if len(filteredEvents) > 0 {
		printUTF8ln("Aufweck-Ereignisse aus dem Ereignisprotokoll (neueste zuerst):")
		maxEvents := 10
		if len(filteredEvents) < maxEvents {
			maxEvents = len(filteredEvents)
		}
		for i := 0; i < maxEvents; i++ {
			event := filteredEvents[i]
			// Convert UTC times to local time for display
			wakeTimeLocal := event.wakeTime.Local()
			sleepTimeLocal := event.sleepTime.Local()
			wakeTimeFormatted := wakeTimeLocal.Format("02.01.2006 15:04:05")
			sleepTimeFormatted := sleepTimeLocal.Format("02.01.2006 15:04:05")
			
			printUTF8ln("  %d. Aufwachzeit: %s", i+1, wakeTimeFormatted)
			printUTF8ln("     Schlafbeginn: %s", sleepTimeFormatted)
			printUTF8ln("     Quelle: %s", event.source)
			
			// Calculate sleep duration (difference between wake time and sleep time in the same event)
			duration := event.wakeTime.Sub(event.sleepTime)
			sleepDuration := formatDuration(duration)
			printUTF8ln("     Schlafdauer: %s", sleepDuration)
			
			if i < maxEvents-1 {
				printUTF8ln("")
			}
		}
	} else {
		if full {
			printUTF8ln("Keine Power-Troubleshooter-Ereignisse im Ereignisprotokoll gefunden.")
		}
	}

	return nil
}

// formatDuration formats a duration in a human-readable format (German)
// Shows days, hours, minutes, and seconds as appropriate
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	
	if seconds < 60 {
		return fmt.Sprintf("%d Sekunden", seconds)
	}
	
	minutes := seconds / 60
	remainingSeconds := seconds % 60
	
	if minutes < 60 {
		if remainingSeconds > 0 {
			return fmt.Sprintf("%d Minuten %d Sekunden", minutes, remainingSeconds)
		}
		return fmt.Sprintf("%d Minuten", minutes)
	}
	
	hours := minutes / 60
	remainingMinutes := minutes % 60
	
	if hours < 24 {
		if remainingMinutes > 0 {
			return fmt.Sprintf("%d Stunden %d Minuten", hours, remainingMinutes)
		}
		return fmt.Sprintf("%d Stunden", hours)
	}
	
	days := hours / 24
	remainingHours := hours % 24
	
	if remainingHours > 0 {
		return fmt.Sprintf("%d Tage %d Stunden", days, remainingHours)
	}
	return fmt.Sprintf("%d Tage", days)
}

// showEventLogAlternative tries alternative method to read event log
func showEventLogAlternative() error {
	// Try using PowerShell to read Event Log
	psScript := `Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-Power-Troubleshooter'} -MaxEvents 10 -ErrorAction SilentlyContinue | Select-Object -First 5 TimeCreated, Message | Format-List`
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not read Event Log: %w", err)
	}

	outputStr := string(output) // Keep in Windows codepage, do not convert
	if strings.TrimSpace(outputStr) != "" {
		fmt.Println(outputStr)
	} else {
		fmt.Println("No Power-Troubleshooter events found.")
	}

	return nil
}
