// SleepRight: Windows Sleep Management Tool
//
// description: Configure Windows 11 PCs for reliable sleep mode and fix sleep-related issues
//
// Version: 1.0.4.17 (in version.go zu ändern)
//
// ChangeLog:
// 19.12.25	1.0.4	Use wmi to aquire detailed information, implement -info-full and -info to have to different levels of details
// 19.12.25	1.0.2	Include timer analysis and detect modern Standby
// 19.12.25	1.0.1	Include automatic elevation
// 12.01.25	1.0.0	Initial version
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

var (
	infoFlag      bool
	infoFullFlag  bool
	configureFlag bool
	waitMinutes   int
	verboseFlag   bool
	debugFlag     bool
	versionFlag   bool
	childModeFlag string // Pipe name for child mode (elevated instance)
	stdOutWriter  *os.File
	stdErrWriter  *os.File
	childPipe     io.WriteCloser // Pipe connection in child mode (must be WriteCloser for Close())
	childExitCode int            // Exit code for child mode
)

func main() {
	// Check if running on Windows
	if runtime.GOOS != "windows" {
		fmt.Fprintf(os.Stderr, "Error: SleepRight is only supported on Windows\n")
		os.Exit(1)
	}

	// Parse command line flags first (before elevation check)
	flag.BoolVar(&infoFlag, "info", false, "Show wake events and current power settings (summary)")
	flag.BoolVar(&infoFlag, "i", false, "Show wake events and current power settings (summary, short)")
	flag.BoolVar(&infoFullFlag, "info-full", false, "Show wake events and current power settings (full details)")
	flag.BoolVar(&configureFlag, "configure", false, "Configure power settings")
	flag.BoolVar(&configureFlag, "c", false, "Configure power settings (short)")
	flag.IntVar(&waitMinutes, "wait", 0, "Set hibernate timeout in minutes")
	flag.IntVar(&waitMinutes, "w", 0, "Set hibernate timeout in minutes (short)")
	flag.BoolVar(&verboseFlag, "verbose", false, "Verbose output")
	flag.BoolVar(&verboseFlag, "v", false, "Verbose output (short)")
	flag.BoolVar(&debugFlag, "debug", false, "Debug mode: show all external command calls")
	flag.BoolVar(&versionFlag, "version", false, "Show version and exit")
	flag.StringVar(&childModeFlag, "child-mode", "", "Internal flag: pipe name for elevated instance")
	flag.Parse()

	// Handle child mode (elevated instance) - redirect output to pipe
	if childModeFlag != "" {
		if err := runAsChild(childModeFlag); err != nil {
			os.Exit(1)
		}
		defer CloseChildMode()
	}

	// Request administrator privileges if needed (for configure or info operations)
	if configureFlag || infoFlag || infoFullFlag {
		if !isAdmin() {
			if err := runAsAdminWithPipe(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to request administrator privileges: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please run this program as administrator.\n")
				os.Exit(1)
			}
			// Never reached as runAsAdminWithPipe will exit the current process
			os.Exit(0)
		}
	}

	// Display version on startup
	fmt.Printf("SleepRight v%s (Build: %s)\n", Version, BuildTime)

	// Handle version flag
	if versionFlag {
		os.Exit(0)
	}

	// If no flags specified, show usage
	if !infoFlag && !infoFullFlag && !configureFlag && waitMinutes == 0 {
		showUsage()
		os.Exit(0)
	}

	// Execute requested actions
	var exitCode int = 0
	if infoFlag || infoFullFlag {
		if err := showInfo(infoFullFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Fehler beim Anzeigen der Informationen: %v\n", err)
			exitCode = 1
		}
	}

	if configureFlag {
		if err := configurePowerSettings(waitMinutes); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring power settings: %v\n", err)
			exitCode = 1
		}
	}

	// If in child mode, store exit code and let CloseChildMode handle exit
	if childModeFlag != "" {
		childExitCode = exitCode
		return
	}

	os.Exit(exitCode)
}

// runAsChild runs in child mode (elevated instance) and redirects output to pipe
func runAsChild(pipeName string) error {
	// Connect to the named pipe created by the parent process
	pipe, err := winio.DialPipe(pipeName, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to pipe: %w", err)
	}
	// DO NOT close pipe here - it must stay open for output
	// It will be closed in CloseChildMode()

	// Store original stdout/stderr for fallback
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create a multi-writer that writes to both pipe and original stdout/stderr
	// This allows output to be captured by parent while still visible in child
	pipeWriter := io.MultiWriter(pipe, originalStdout)
	pipeErrorWriter := io.MultiWriter(pipe, originalStderr)

	// Create pipes to intercept output
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		pipe.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		pipe.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Copy stdout to both pipe and original stdout
	go func() {
		defer stdoutR.Close()
		io.Copy(pipeWriter, stdoutR)
	}()

	// Copy stderr to both pipe and original stderr
	go func() {
		defer stderrR.Close()
		io.Copy(pipeErrorWriter, stderrR)
	}()

	// Replace stdout/stderr with our pipe writers
	os.Stdout = stdoutW
	os.Stderr = stderrW
	stdOutWriter = stdoutW
	stdErrWriter = stderrW
	childPipe = pipe

	// Test: Write a message directly to pipe to verify connection
	// This should appear in parent's output
	fmt.Fprintf(pipe, "[PIPE_TEST] Child process connected to pipe successfully\n")
	// Also write to stdout to test the pipe mechanism
	fmt.Println("[STDOUT_TEST] This should appear in pipe")

	return nil
}

func CloseChildMode() {
	// Flush any remaining output
	if stdOutWriter != nil {
		stdOutWriter.Sync()
	}
	if stdErrWriter != nil {
		stdErrWriter.Sync()
	}

	// Wait a bit for output to be copied to pipe
	time.Sleep(300 * time.Millisecond)

	// Send exit code through pipe (as a special marker)
	if childPipe != nil {
		fmt.Fprintf(childPipe, "\n[EXIT_CODE:%d]\n", childExitCode)
		// Flush the pipe connection
		if conn, ok := childPipe.(interface{ Flush() error }); ok {
			conn.Flush()
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Close the write ends to signal completion
	if stdOutWriter != nil {
		stdOutWriter.Close()
	}
	if stdErrWriter != nil {
		stdErrWriter.Close()
	}

	// Wait a bit more for final output to be copied
	time.Sleep(300 * time.Millisecond)

	// Close the pipe connection
	if childPipe != nil {
		childPipe.Close()
	}

	os.Exit(childExitCode)
}

// runAsAdminWithPipe starts the program with admin rights and uses a named pipe for output
func runAsAdminWithPipe() error {
	// Create a unique pipe name
	pipeName := fmt.Sprintf(`\\.\pipe\SleepRight_%d`, os.Getpid())

	// Create the named pipe
	listener, err := winio.ListenPipe(pipeName, nil)
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer listener.Close()

	// Start goroutine to accept connection and read output
	outputChan := make(chan string, 100)
	exitCodeChan := make(chan int, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			outputChan <- fmt.Sprintf("Error accepting pipe connection: %v\n", err)
			exitCodeChan <- 1
			return
		}
		defer conn.Close()

		// Read output from the pipe in real-time and send immediately
		buffer := make([]byte, 4096)
		var allOutput []byte

		for {
			n, err := conn.Read(buffer)
			if n > 0 {
				// Add chunk to accumulated output
				chunk := buffer[:n]
				allOutput = append(allOutput, chunk...)
				chunkStr := string(chunk)

				// Filter out exit code marker from chunk before sending
				re := regexp.MustCompile(`\[EXIT_CODE:\d+\]`)
				if re.MatchString(chunkStr) {
					// Remove the marker from chunk
					chunkStr = re.ReplaceAllString(chunkStr, "")
					// Also remove surrounding newlines if they're part of the marker
					chunkStr = strings.ReplaceAll(chunkStr, "\n\n", "\n")
					chunkStr = strings.TrimPrefix(chunkStr, "\n")
					chunkStr = strings.TrimSuffix(chunkStr, "\n")
				}

				// Check if we have a complete exit code marker in accumulated output
				allOutputStr := string(allOutput)
				re2 := regexp.MustCompile(`\[EXIT_CODE:(\d+)\]`)
				matches := re2.FindStringSubmatch(allOutputStr)
				if len(matches) > 1 {
					// Found exit code, extract it
					var exitCode int
					if _, err := fmt.Sscanf(matches[1], "%d", &exitCode); err == nil {
						// Remove marker from all output
						marker := matches[0]
						allOutputStr = strings.ReplaceAll(allOutputStr, "\n"+marker+"\n", "")
						allOutputStr = strings.ReplaceAll(allOutputStr, marker+"\n", "")
						allOutputStr = strings.ReplaceAll(allOutputStr, "\n"+marker, "")
						allOutputStr = strings.ReplaceAll(allOutputStr, marker, "")

						// Send remaining chunk if it doesn't contain the marker
						if !strings.Contains(chunkStr, marker) && chunkStr != "" {
							outputChan <- chunkStr
						}

						exitCodeChan <- exitCode
						return
					}
				}

				// No exit code found yet, send chunk normally (after filtering)
				if chunkStr != "" {
					outputChan <- chunkStr
				}
			}
			if err == io.EOF {
				// Check for exit code in remaining output
				outputStr := string(allOutput)
				var exitCode int
				if _, err := fmt.Sscanf(outputStr, "[EXIT_CODE:%d]", &exitCode); err == nil {
					// Remove exit code marker from output
					marker := fmt.Sprintf("\n[EXIT_CODE:%d]\n", exitCode)
					marker2 := fmt.Sprintf("[EXIT_CODE:%d]\n", exitCode)
					if strings.HasSuffix(outputStr, marker) {
						allOutput = allOutput[:len(allOutput)-len(marker)]
					} else if strings.HasSuffix(outputStr, marker2) {
						allOutput = allOutput[:len(allOutput)-len(marker2)]
					}
					// Send remaining output without marker
					if len(allOutput) > 0 {
						outputChan <- string(allOutput)
					}
					exitCodeChan <- exitCode
				} else {
					// No exit code found, send all output
					if len(allOutput) > 0 {
						outputChan <- string(allOutput)
					}
					exitCodeChan <- 0
				}
				return
			}
			if err != nil {
				outputChan <- fmt.Sprintf("Error reading from pipe: %v\n", err)
				exitCodeChan <- 1
				return
			}
		}
	}()

	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
		if !filepath.IsAbs(exe) {
			if absPath, err := filepath.Abs(exe); err == nil {
				exe = absPath
			}
		}
	}

	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Build command line arguments (preserve all original arguments and add -child-mode)
	args := os.Args[1:]

	// Remove -child-mode if present (shouldn't be, but just in case)
	filteredArgs := []string{}
	for _, arg := range args {
		if arg != "-child-mode" && !contains(arg, "-child-mode=") {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// Add -child-mode with pipe name
	filteredArgs = append(filteredArgs, "-child-mode", pipeName)

	// Build argument string
	argsStr := ""
	for i, arg := range filteredArgs {
		if i > 0 {
			argsStr += " "
		}
		if containsSpace(arg) {
			argsStr += `"` + arg + `"`
		} else {
			argsStr += arg
		}
	}

	// Use ShellExecute to run as administrator (hidden window)
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	argsPtr, _ := syscall.UTF16PtrFromString(argsStr)

	// showCmd := int32(0) // SW_HIDE - hide the window
	showCmd := int32(1) // SW_NORMAL - show window for debugging

	err = windows.ShellExecute(0, verbPtr, exePtr, argsPtr, nil, showCmd)
	if err != nil {
		return fmt.Errorf("failed to execute as administrator: %w", err)
	}

	// Wait a bit for the connection
	time.Sleep(500 * time.Millisecond)

	// Read and display output from the pipe
	timeout := time.After(60 * time.Second)
	var exitCode int = 1
	outputReceived := false

	for {
		select {
		case output := <-outputChan:
			if output != "" {
				outputReceived = true
				// Output is already in Windows codepage, print directly
				fmt.Print(output)
			}
		case exitCode = <-exitCodeChan:
			// All output received, exit
			if !outputReceived {
				fmt.Fprintf(os.Stderr, "Warning: No output received from elevated process\n")
			}
			os.Exit(exitCode)
		case <-timeout:
			if !outputReceived {
				fmt.Fprintf(os.Stderr, "Timeout waiting for elevated process (no output received)\n")
			} else {
				fmt.Fprintf(os.Stderr, "Timeout waiting for elevated process to complete\n")
			}
			os.Exit(1)
		}
	}
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "Usage: SleepRight [OPTIONS]\n\n")
	fmt.Fprintf(os.Stderr, "OPTIONS:\n")
	fmt.Fprintf(os.Stderr, "  -info, -i              Show wake events and current power settings\n")
	fmt.Fprintf(os.Stderr, "  -configure, -c         Configure power settings\n")
	fmt.Fprintf(os.Stderr, "  -wait, -w <minutes>    Set hibernate timeout in minutes\n")
	fmt.Fprintf(os.Stderr, "  -verbose, -v           Verbose output\n")
	fmt.Fprintf(os.Stderr, "  --version              Show version and exit\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  SleepRight -info                    # Show current settings\n")
	fmt.Fprintf(os.Stderr, "  SleepRight -configure               # Configure power settings\n")
	fmt.Fprintf(os.Stderr, "  SleepRight -configure -w 60         # Configure with 60 min before hibernate\n")
}

func showInfo(full bool) error {
	printUTF8ln("=== Aufweck-Ereignisse ===")
	if err := showWakeEvents(full); err != nil {
		return fmt.Errorf("Fehler beim Anzeigen der Aufweck-Ereignisse: %w", err)
	}

	printUTF8ln("\n=== Energieeinstellungen ===")
	if err := showPowerSettings(full); err != nil {
		return fmt.Errorf("Fehler beim Anzeigen der Energieeinstellungen: %w", err)
	}

	return nil
}

func configurePowerSettings(hibernateMinutes int) error {
	fmt.Println("=== Configuring Power Settings ===")

	if err := configureWakeDevices(); err != nil {
		return fmt.Errorf("failed to configure wake devices: %w", err)
	}

	if err := configureSleepTimeout(); err != nil {
		return fmt.Errorf("failed to configure sleep timeout: %w", err)
	}

	if err := configurePowerScheme(); err != nil {
		return fmt.Errorf("failed to configure power scheme: %w", err)
	}

	if err := configureWakeTimers(); err != nil {
		return fmt.Errorf("failed to configure wake timers: %w", err)
	}

	if hibernateMinutes > 0 {
		if err := configureHibernateTimeout(hibernateMinutes); err != nil {
			return fmt.Errorf("failed to configure hibernate timeout: %w", err)
		}
	}

	fmt.Println("\nConfiguration completed successfully!")
	return nil
}

// isAdmin checks if the current process is running with administrator privileges
func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// containsSpace checks if a string contains spaces
func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}
