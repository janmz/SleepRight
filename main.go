// SleepRight: Windows Sleep Management Tool
//
// description: Configure Windows 11 PCs for reliable sleep mode and fix sleep-related issues
//
// Version: 1.0.1.2 (in version.go zu ändern)
//
// ChangeLog:
// 19.12.25	1.0.1	Include automatic elevation
// 12.01.25	1.0.0	Initial version
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	infoFlag      bool
	configureFlag bool
	waitMinutes   int
	verboseFlag   bool
	versionFlag   bool
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
	flag.Parse()

	// Request administrator privileges if needed (for configure or info operations)
	if configureFlag || infoFlag {
		if !isAdmin() {
			if err := runAsAdmin(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to request administrator privileges: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please run this program as administrator.\n")
				os.Exit(1)
			}
			// If runAsAdmin succeeded, the current process will exit
			// and a new elevated process will start
			return
		}
	}

	// Display version on startup
	fmt.Printf("SleepRight v%s (Build: %s)\n", Version, BuildTime)

	// Handle version flag
	if versionFlag {
		fmt.Printf("Version: %s\nBuildTime: %s\n", Version, BuildTime)
		os.Exit(0)
	}

	// If no flags specified, show usage
	if !infoFlag && !configureFlag && waitMinutes == 0 {
		showUsage()
		os.Exit(0)
	}

	// Execute requested actions
	if infoFlag {
		if err := showInfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing info: %v\n", err)
			os.Exit(1)
		}
	}

	if configureFlag {
		if err := configurePowerSettings(waitMinutes); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring power settings: %v\n", err)
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

// runAsAdmin restarts the program with administrator privileges
func runAsAdmin() error {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		// Fallback to os.Args[0]
		exe = os.Args[0]
		// Try to make it absolute
		if !filepath.IsAbs(exe) {
			if absPath, err := filepath.Abs(exe); err == nil {
				exe = absPath
			}
		}
	}

	// Get absolute path
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Build command line arguments (preserve all original arguments)
	args := os.Args[1:]

	// Use ShellExecute to run as administrator
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	var argsPtr *uint16
	if len(args) > 0 {
		argsStr := ""
		for i, arg := range args {
			if i > 0 {
				argsStr += " "
			}
			// Quote arguments that contain spaces
			if containsSpace(arg) {
				argsStr += `"` + arg + `"`
			} else {
				argsStr += arg
			}
		}
		argsPtr, _ = syscall.UTF16PtrFromString(argsStr)
	}

	var showCmd int32 = 1 // SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, argsPtr, nil, showCmd)
	if err != nil {
		return fmt.Errorf("failed to execute as administrator: %w", err)
	}

	// Exit current process - the elevated process will continue
	os.Exit(0)
	return nil
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
