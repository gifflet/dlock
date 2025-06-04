# dLock

**Disable the Android lock screen using ADB — powered by Go.**

`dlock` is a powerful CLI tool that automates the process of disabling the lock screen on Android devices connected via ADB. Designed for developers, testers, and automation workflows with support for multiple devices and concurrent processing.

## Features

- 🚀 No root required
- 🔍 Automatic device detection
- 📱 Multi-device support with concurrent processing
- 🎯 Target specific devices by UDID
- 🔐 Multiple disabling strategies for maximum compatibility
- ✅ Verification of successful lock screen removal
- 📊 Detailed execution logs and status reporting
- ⚠️ Comprehensive error handling and troubleshooting

## Quick Start

### Prerequisites

- ADB (Android Debug Bridge) installed and available in your system PATH
- USB debugging enabled on your Android device(s)

### Installation

1. **Download the latest binary** from the [releases page](https://github.com/gifflet/dlock/releases)

2. **Install ADB** on your system:
   - **macOS**: `brew install android-platform-tools`
   - **Linux**: `sudo apt-get install android-tools-adb`
   - **Windows**: Download from [Android Platform Tools](https://developer.android.com/tools/releases/platform-tools)

3. **Enable USB debugging** on your Android device:
   - Go to Settings > About Phone
   - Tap "Build Number" 7 times to enable Developer Options
   - Go to Settings > Developer Options
   - Enable "USB debugging"

### Usage

1. **Connect your Android device(s)** via USB

2. **Run dlock:**
   ```bash
   # Process all connected devices
   ./dlock
   
   # Process specific devices by UDID
   ./dlock -devices "ABC123DEF456 789GHI012JKL"
   
   # Show help
   ./dlock -help
   ```

3. **Get device UDIDs** (if needed):
   ```bash
   adb devices
   ```

### How It Works

The tool automatically:
1. Detects connected Android devices
2. Tries multiple methods to disable the lock screen
3. Reboots the device to apply changes
4. Validates that the lock screen has been removed

## Troubleshooting

If the script fails to disable the lock screen:

- Ensure USB debugging is enabled on your device
- Check if your device requires USB debugging authorization
- Try enabling 'Settings > Developer Options > Disable permission monitoring'
- Some devices may have policy restrictions preventing lock screen modifications
- Make sure ADB is properly installed and accessible from the command line
- Check USB connection and try a different USB cable if necessary

## Security Considerations

- Only use this tool on devices you own or have permission to modify
- Be aware that disabling the lock screen may reduce device security
- Some enterprise policies or device administrators may prevent lock screen modifications
- Re-enabling the lock screen may require a device reset or manual configuration

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 