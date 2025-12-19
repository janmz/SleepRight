// SleepRight: Windows Sleep Management Tool
//
// description: Configure Windows 11 PCs for reliable sleep mode and fix sleep-related issues
//
// Version: 1.0.3.4 (in version.go zu ändern)
//
// ChangeLog:
// 19.12.25	1.0.3	Use namend pipes to get output from elevated instance
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
	"runtime"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

var (
	infoFlag      bool
	configureFlag bool
	waitMinutes   int
	verboseFlag   bool
	versionFlag   bool
	childModeFlag string // Pipe name for child mode (elevated instance)
	stdOutWriter  *os.File
	stdErrWriter  *os.File
	childPipe     io.Writer // Pipe connection in child mode
	childExitCode int       // Exit code for child mode
)

func main() {
	// Check if running on Windows
	if runtime.GOOS != "windows" {
		fmt.Fprintf(os.Stderr, "Error: SleepRight is only supported on Windows\n")
		os.Exit(1)
	}

	// Parse command line flags first (before elevation check)
	flag.BoolVar(&infoFlag, "info", false, "Show wake events and current power settings")
	flag.BoolVar(&infoFlag, "i", false, "Show wake events and current power settings (short)")
	flag.BoolVar(&configureFlag, "configure", false, "Configure power settings")
	flag.BoolVar(&configureFlag, "c", false, "Configure power settings (short)")
	flag.IntVar(&waitMinutes, "wait", 0, "Set hibernate timeout in minutes")
	flag.IntVar(&waitMinutes, "w", 0, "Set hibernate timeout in minutes (short)")
	flag.BoolVar(&verboseFlag, "verbose", false, "Verbose output")
	flag.BoolVar(&verboseFlag, "v", false, "Verbose output (short)")
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
	if configureFlag || infoFlag {
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
	if !infoFlag && !configureFlag && waitMinutes == 0 {
		showUsage()
		os.Exit(0)
	}

	// Execute requested actions
	var exitCode int = 0
	if infoFlag {
		if err := showInfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing info: %v\n", err)
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
	defer pipe.Close()

	// Create a multi-writer that writes to both pipe and original stdout/stderr
	// This allows output to be captured by parent while still visible in child
	pipeWriter := io.MultiWriter(pipe, os.Stdout)
	pipeErrorWriter := io.MultiWriter(pipe, os.Stderr)

	// Temporarily replace stdout/stderr with our pipe writers
	// We'll use a goroutine to copy output
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes to intercept output
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()

	// Copy stdout to both pipe and original stdout
	go func() {
		io.Copy(pipeWriter, stdoutR)
	}()
	go func() {
		io.Copy(originalStdout, stdoutR)
	}()

	// Copy stderr to both pipe and original stderr
	go func() {
		io.Copy(pipeErrorWriter, stderrR)
	}()
	go func() {
		io.Copy(originalStderr, stderrR)
	}()

	// Replace stdout/stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW
	stdOutWriter = stdoutW
	stdErrWriter = stderrW
	childPipe = pipe

	return nil
}

func CloseChildMode() {
	if stdOutWriter == nil || stdErrWriter == nil || childPipe == nil {
		return
	}

	// Close the write ends to signal completion
	stdOutWriter.Close()
	stdErrWriter.Close()

	// Wait a bit for output to be copied
	time.Sleep(100 * time.Millisecond)

	// Send exit code through pipe (as a special marker)
	fmt.Fprintf(childPipe, "\n[EXIT_CODE:%d]\n", childExitCode)

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

		// Read all output from the pipe
		buffer := make([]byte, 4096)
		var output []byte

		for {
			n, err := conn.Read(buffer)
			if n > 0 {
				output = append(output, buffer[:n]...)
				// Check for exit code marker
				outputStr := string(output)
				if idx := len(outputStr) - 200; idx > 0 {
					// Check last 200 chars for exit code
					lastPart := outputStr[len(outputStr)-200:]
					var exitCode int
					if _, err := fmt.Sscanf(lastPart, "[EXIT_CODE:%d]", &exitCode); err == nil {
						// Remove exit code marker from output
						outputStr = outputStr[:len(outputStr)-len(fmt.Sprintf("[EXIT_CODE:%d]", exitCode))-1]
						output = []byte(outputStr)
						exitCodeChan <- exitCode
						break
					}
				}
			}
			if err == io.EOF {
				// Check for exit code in remaining output
				outputStr := string(output)
				var exitCode int
				if _, err := fmt.Sscanf(outputStr, "[EXIT_CODE:%d]", &exitCode); err == nil {
					outputStr = outputStr[:len(outputStr)-len(fmt.Sprintf("[EXIT_CODE:%d]", exitCode))-1]
					output = []byte(outputStr)
					exitCodeChan <- exitCode
				} else {
					exitCodeChan <- 0
				}
				break
			}
			if err != nil {
				outputChan <- fmt.Sprintf("Error reading from pipe: %v\n", err)
				exitCodeChan <- 1
				break
			}
		}

		// Send output in chunks
		if len(output) > 0 {
			outputChan <- string(output)
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

	showCmd := int32(0) // SW_HIDE - hide the window

	err = windows.ShellExecute(0, verbPtr, exePtr, argsPtr, nil, showCmd)
	if err != nil {
		return fmt.Errorf("failed to execute as administrator: %w", err)
	}

	// Wait a bit for the connection
	time.Sleep(100 * time.Millisecond)

	// Read and display output from the pipe
	timeout := time.After(30 * time.Second)
	var exitCode int = 1

	for {
		select {
		case output := <-outputChan:
			fmt.Print(output)
		case exitCode = <-exitCodeChan:
			// All output received, exit
			os.Exit(exitCode)
		case <-timeout:
			fmt.Fprintf(os.Stderr, "Timeout waiting for elevated process\n")
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
	fmt.Fprintf(os.Stderr, "  SleepRight -configure -w 60          # Configure with 60 min hibernate\n")
}

func showInfo() error {
	fmt.Println("=== Wake Events ===")
	if err := showWakeEvents(); err != nil {
		return fmt.Errorf("failed to show wake events: %w", err)
	}

	fmt.Println("\n=== Power Settings ===")
	if err := showPowerSettings(); err != nil {
		return fmt.Errorf("failed to show power settings: %w", err)
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

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr)))
}
