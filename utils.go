package main

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// runCommandWithEncoding runs a command and returns output in Windows codepage (CP1252)
// DO NOT convert to UTF-8 - Windows console expects CP1252
func runCommandWithEncoding(name string, args ...string) (string, error) {
	if debugFlag {
		fmt.Println("------------------ Externer Aufruf ------------------------")
		fmt.Printf("Aufruf: %s", name)
		for _, arg := range args {
			fmt.Printf(" %s", arg)
		}
		fmt.Println()
	}

	cmd := exec.Command(name, args...)
	output, err := cmd.Output()

	if debugFlag {
		fmt.Println("Ausgabe:")
		if err != nil {
			fmt.Printf("Fehler: %v\n", err)
		} else {
			// Show raw output in Windows codepage
			fmt.Print(string(output))
		}
		fmt.Println("-------------------------------------------------")
	}

	if err != nil {
		return "", err
	}
	// Return output as-is in Windows codepage (CP1252), do NOT convert to UTF-8
	return string(output), nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// printUTF8 converts UTF-8 string to Windows codepage (CP1252) and prints it
func printUTF8(format string, args ...interface{}) {
	// Format the string first
	utf8Str := fmt.Sprintf(format, args...)
	// Convert UTF-8 to Windows CP1252
	encoder := charmap.Windows1252.NewEncoder()
	cp1252Bytes, err := encoder.String(utf8Str)
	if err != nil {
		// If conversion fails, print as-is
		fmt.Print(utf8Str)
		return
	}
	fmt.Print(cp1252Bytes)
}

// printUTF8ln converts UTF-8 string to Windows codepage (CP1252) and prints it with newline
func printUTF8ln(format string, args ...interface{}) {
	printUTF8(format, args...)
	fmt.Println()
}
