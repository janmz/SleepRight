//
// SleepRight: Windows Sleep Management Tool
//
// description: Configure Windows 11 PCs for reliable sleep mode and fix sleep-related issues
//
// Version: 1.0.0.0 (in version.go zu ändern)
//
// ChangeLog:
// 12.01.25	1.0.0	Initial version
//
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	infoFlag      bool
	configureFlag bool
	waitMinutes   int
	verboseFlag   bool
	versionFlag   bool
)

func main() {
	// Display version on startup
	fmt.Printf("SleepRight v%s (Build: %s)\n", Version, BuildTime)

	// Parse command line flags
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

