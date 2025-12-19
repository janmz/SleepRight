# SleepRight

SleepRight is a Windows sleep management tool that helps configure Windows 11 PCs for reliable sleep mode and troubleshoot sleep-related issues.

## Features

- **Wake Event Analysis**: Reads and displays information about the last wake events from Windows Event Log
- **Power Settings Overview**: Shows current power scheme settings, sleep timeouts, wake device configuration, and hibernate settings
- **Power Configuration**: Configures the PC so that only keyboard and Ethernet can wake it, with 30 minutes of inactivity before sleep mode
- **Hibernate Timeout**: Configurable hibernate timeout via command-line parameter

## Installation

### Build from Source

```bash
git clone https://github.com/janmz/SleepRight.git
cd SleepRight
go build
```

Or use prepareBuild to build with version information:

```bash
prepareBuild
```

## Usage

### Show Information

Display wake events and current power settings:

```bash
SleepRight -info
# or
SleepRight -i
```

### Configure Power Settings

Configure power settings with default values (30 minutes sleep timeout, Balanced power scheme):

```bash
SleepRight -configure
# or
SleepRight -c
```

### Configure with Custom Hibernate Timeout

Configure power settings with a custom hibernate timeout (in minutes):

```bash
SleepRight -configure -wait 60
# or
SleepRight -c -w 60
```

### Verbose Output

Enable verbose output for detailed information:

```bash
SleepRight -info -verbose
# or
SleepRight -i -v
```

### Show Version

Display version and build time:

```bash
SleepRight --version
```

## Command Line Options

- `-info`, `-i` - Show wake events and current power settings
- `-configure`, `-c` - Configure power settings
- `-wait`, `-w <minutes>` - Set hibernate timeout in minutes (e.g., `-w 60` for 60 minutes)
- `-verbose`, `-v` - Verbose output
- `--version` - Show version and exit

## What SleepRight Configures

When you run `SleepRight -configure`, it will:

1. **Wake Devices**: Enable wake only for keyboard and Ethernet adapter, disable all other wake devices
2. **Sleep Timeout**: Set sleep timeout to 30 minutes of inactivity (both AC and battery)
3. **Power Scheme**: Set power scheme to "Balanced"
4. **Hibernate Timeout**: Configure hibernate timeout if `-wait` parameter is provided

## Requirements

- Windows 11 (may work on Windows 10)
- Administrator rights required for power configuration
- Go 1.23 or later (for building from source)

## Examples

### Example 1: Check Current Settings

```bash
SleepRight -info
```

Output:
```
SleepRight v1.0.0.0 (Build: 2025-01-12 00:00:00)
=== Wake Events ===
Last Wake Event:
Wake History Count - 1
Wake History [0]
  Wake Source Count - 0

=== Power Settings ===
Active Power Scheme:
Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)

Sleep Settings (AC Power):
  Timeout: 30 minutes

Hibernate Settings:
  Timeout: 0 seconds (disabled)

Devices that can wake the computer:
  - HID Keyboard Device
  - Realtek PCIe GbE Family Controller
```

### Example 2: Configure with 60 Minute Hibernate

```bash
SleepRight -configure -w 60
```

This will configure all power settings and set hibernate timeout to 60 minutes.

## Troubleshooting

### Administrator Rights Required

Some operations require administrator rights. If you see errors about permissions, run the program as administrator:

1. Right-click on `SleepRight.exe`
2. Select "Run as administrator"

### Wake Devices Not Found

If SleepRight cannot find your keyboard or Ethernet adapter, you can manually enable wake for specific devices:

```bash
powercfg /deviceenablewake "Device Name"
```

To see all available devices:

```bash
powercfg /devicequery wake_programmable
```

## License

MIT License with attribution. Donationware for CFI Kinderhilfe.

Copyright (c) 2024-2025 VAYA Consulting

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and feature requests, please use the [GitHub Issues](https://github.com/janmz/SleepRight/issues) page.

