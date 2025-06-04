package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DeviceInfo holds information about an Android device
type DeviceInfo struct {
	Model          string
	Manufacturer   string
	AndroidVersion string
	APILevel       string
}

// ProcessingStats holds the statistics for device processing
type ProcessingStats struct {
	mu            sync.Mutex
	successCount  int
	failedDevices []string
	totalDevices  int
}

// incrementSuccess safely increments the success counter
func (ps *ProcessingStats) incrementSuccess() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.successCount++
}

// addFailedDevice safely adds a device to the failed list
func (ps *ProcessingStats) addFailedDevice(deviceSerial string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.failedDevices = append(ps.failedDevices, deviceSerial)
}

// getStats safely retrieves current statistics
func (ps *ProcessingStats) getStats() (int, []string, int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	failedCopy := make([]string, len(ps.failedDevices))
	copy(failedCopy, ps.failedDevices)
	return ps.successCount, failedCopy, ps.totalDevices
}

// AndroidLockScreenDisabler handles the lock screen disabling process
type AndroidLockScreenDisabler struct {
	connectedDevices []string
	targetDevices    []string // New field for target UDIDs
	logMutex         sync.Mutex
}

// NewAndroidLockScreenDisabler creates a new instance of the disabler
func NewAndroidLockScreenDisabler(targetDevices []string) *AndroidLockScreenDisabler {
	return &AndroidLockScreenDisabler{
		connectedDevices: make([]string, 0),
		targetDevices:    targetDevices,
	}
}

// log prints formatted log messages with emojis (thread-safe)
func (a *AndroidLockScreenDisabler) log(message, emoji string) {
	if emoji == "" {
		emoji = "ℹ️"
	}

	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	fmt.Printf("%s %s\n", emoji, message)
}

// runADBCommand executes an ADB command and returns success, output, and error
func (a *AndroidLockScreenDisabler) runADBCommand(command string, deviceSerial string) (bool, string, string) {
	var fullCommand string
	if deviceSerial != "" {
		fullCommand = fmt.Sprintf("adb -s %s %s", deviceSerial, command)
	} else {
		fullCommand = fmt.Sprintf("adb %s", command)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	// Use appropriate shell based on operating system
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", fullCommand)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", fullCommand)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "", "Command timed out"
		}
		return false, "", err.Error()
	}

	return cmd.ProcessState.ExitCode() == 0, strings.TrimSpace(string(output)), ""
}

// checkADBAvailability checks if ADB is available in the system
func (a *AndroidLockScreenDisabler) checkADBAvailability() bool {
	a.log("Checking ADB availability...", "🔍")
	success, _, errorMsg := a.runADBCommand("version", "")

	if success {
		a.log("ADB is available and working!", "✅")
		return true
	}

	a.log("ADB is not available or not working properly!", "❌")
	a.log(fmt.Sprintf("Error: %s", errorMsg), "⚠️")
	return false
}

// getConnectedDevices gets list of connected Android devices
func (a *AndroidLockScreenDisabler) getConnectedDevices() []string {
	a.log("Scanning for connected Android devices...", "📱")
	success, output, _ := a.runADBCommand("devices", "")

	if !success {
		a.log("Failed to get device list!", "❌")
		return []string{}
	}

	allDevices := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	firstLine := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if firstLine {
			firstLine = false
			continue // Skip the header line
		}

		if line != "" && strings.Contains(line, "\tdevice") {
			parts := strings.Split(line, "\t")
			if len(parts) > 0 {
				allDevices = append(allDevices, parts[0])
			}
		}
	}

	// Filter devices based on target UDIDs if specified
	var devices []string
	if len(a.targetDevices) > 0 {
		a.log(fmt.Sprintf("Filtering devices based on specified UDIDs: %s", strings.Join(a.targetDevices, ", ")), "🎯")

		deviceMap := make(map[string]bool)
		for _, device := range allDevices {
			deviceMap[device] = true
		}

		for _, targetDevice := range a.targetDevices {
			if deviceMap[targetDevice] {
				devices = append(devices, targetDevice)
			} else {
				a.log(fmt.Sprintf("Warning: Device %s not found in connected devices", targetDevice), "⚠️")
			}
		}
	} else {
		devices = allDevices
	}

	if len(devices) > 0 {
		a.log(fmt.Sprintf("Found %d device(s) to process: %s", len(devices), strings.Join(devices, ", ")), "🎯")
		if len(a.targetDevices) > 0 {
			a.log(fmt.Sprintf("Total connected devices: %d, Processing: %d", len(allDevices), len(devices)), "ℹ️")
		}
	} else {
		if len(a.targetDevices) > 0 {
			a.log("None of the specified devices are connected!", "❌")
		} else {
			a.log("No connected devices found!", "❌")
		}
	}

	a.connectedDevices = devices
	return devices
}

// getDeviceInfo gets device information
func (a *AndroidLockScreenDisabler) getDeviceInfo(deviceSerial string) DeviceInfo {
	info := DeviceInfo{
		Model:          "Unknown",
		Manufacturer:   "Unknown",
		AndroidVersion: "Unknown",
		APILevel:       "Unknown",
	}

	// Get device model
	if success, output, _ := a.runADBCommand("shell getprop ro.product.model", deviceSerial); success && output != "" {
		info.Model = output
	}

	// Get manufacturer
	if success, output, _ := a.runADBCommand("shell getprop ro.product.manufacturer", deviceSerial); success && output != "" {
		info.Manufacturer = output
	}

	// Get Android version
	if success, output, _ := a.runADBCommand("shell getprop ro.build.version.release", deviceSerial); success && output != "" {
		info.AndroidVersion = output
	}

	// Get API level
	if success, output, _ := a.runADBCommand("shell getprop ro.build.version.sdk", deviceSerial); success && output != "" {
		info.APILevel = output
	}

	return info
}

// checkDevicePermissions checks if device has necessary permissions for lock screen modifications
func (a *AndroidLockScreenDisabler) checkDevicePermissions(deviceSerial string) bool {
	a.log(fmt.Sprintf("Checking permissions for device %s...", deviceSerial), "🔐")

	// Test basic shell access
	success, _, _ := a.runADBCommand("shell echo 'test'", deviceSerial)
	if !success {
		a.log(fmt.Sprintf("No shell access to device %s", deviceSerial), "❌")
		return false
	}

	// Check if we can access settings (get just the list without head command)
	success, output, _ := a.runADBCommand("shell settings list secure", deviceSerial)
	if !success || output == "" {
		a.log(fmt.Sprintf("Cannot access settings on device %s", deviceSerial), "❌")
		return false
	}

	a.log(fmt.Sprintf("Device %s has necessary permissions", deviceSerial), "✅")
	return true
}

// disableLockscreenMethod1 uses locksettings command (Most compatible)
func (a *AndroidLockScreenDisabler) disableLockscreenMethod1(deviceSerial string) bool {
	a.log(fmt.Sprintf("Trying Method 1 (locksettings) on device %s...", deviceSerial), "🔑")

	// First try to clear any existing lock
	if success, _, _ := a.runADBCommand("shell locksettings clear", deviceSerial); success {
		a.log(fmt.Sprintf("Cleared existing lock settings on %s", deviceSerial), "🧹")
	}

	// Set lockscreen as disabled
	success, _, errorMsg := a.runADBCommand("shell locksettings set-disabled true", deviceSerial)

	if success {
		a.log(fmt.Sprintf("Method 1 succeeded on device %s!", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Method 1 failed on device %s: %s", deviceSerial, errorMsg), "❌")
	return false
}

// disableLockscreenMethod2 uses settings secure (Alternative approach)
func (a *AndroidLockScreenDisabler) disableLockscreenMethod2(deviceSerial string) bool {
	a.log(fmt.Sprintf("Trying Method 2 (settings secure) on device %s...", deviceSerial), "⚙️")

	// Set lockscreen.disabled to 1
	success, _, errorMsg := a.runADBCommand("shell settings put secure lockscreen.disabled 1", deviceSerial)

	if success {
		a.log(fmt.Sprintf("Method 2 succeeded on device %s!", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Method 2 failed on device %s: %s", deviceSerial, errorMsg), "❌")
	return false
}

// disableLockscreenMethod3 uses system settings (Legacy compatibility)
func (a *AndroidLockScreenDisabler) disableLockscreenMethod3(deviceSerial string) bool {
	a.log(fmt.Sprintf("Trying Method 3 (system settings) on device %s...", deviceSerial), "🔧")

	// Set lockscreen_disabled in system settings
	success, _, errorMsg := a.runADBCommand("shell settings put system lockscreen_disabled 1", deviceSerial)

	if success {
		a.log(fmt.Sprintf("Method 3 succeeded on device %s!", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Method 3 failed on device %s: %s", deviceSerial, errorMsg), "❌")
	return false
}

// disableLockscreenMethod4 uses global settings approach
func (a *AndroidLockScreenDisabler) disableLockscreenMethod4(deviceSerial string) bool {
	a.log(fmt.Sprintf("Trying Method 4 (global settings) on device %s...", deviceSerial), "🌐")

	// Set device_provisioned and user_setup_complete
	commands := []string{
		"shell settings put global device_provisioned 1",
		"shell settings put secure user_setup_complete 1",
	}

	successCount := 0
	for _, cmd := range commands {
		if success, _, _ := a.runADBCommand(cmd, deviceSerial); success {
			successCount++
		}
	}

	if successCount > 0 {
		a.log(fmt.Sprintf("Method 4 partially succeeded on device %s!", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Method 4 failed on device %s", deviceSerial), "❌")
	return false
}

// rebootDevice reboots the Android device
func (a *AndroidLockScreenDisabler) rebootDevice(deviceSerial string) bool {
	a.log(fmt.Sprintf("Rebooting device %s...", deviceSerial), "🔄")

	success, _, errorMsg := a.runADBCommand("reboot", deviceSerial)

	if success {
		a.log(fmt.Sprintf("Reboot command sent to device %s", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Failed to reboot device %s: %s", deviceSerial, errorMsg), "❌")
	return false
}

// waitForDeviceReady waits for device to be ready after reboot
func (a *AndroidLockScreenDisabler) waitForDeviceReady(deviceSerial string, maxWaitMinutes int) bool {
	a.log(fmt.Sprintf("Waiting for device %s to be ready after reboot...", deviceSerial), "⏳")

	maxAttempts := maxWaitMinutes * 12 // Check every 5 seconds
	attempt := 0

	for attempt < maxAttempts {
		// First check if device appears in device list
		success, _, _ := a.runADBCommand("get-state", deviceSerial)
		if success {
			// Wait a bit more for system to fully boot
			a.log(fmt.Sprintf("Device %s detected, waiting for system to fully boot...", deviceSerial), "⏱️")
			time.Sleep(10 * time.Second)

			// Test if we can execute shell commands
			success, _, _ := a.runADBCommand("shell echo 'test'", deviceSerial)
			if success {
				a.log(fmt.Sprintf("Device %s is ready!", deviceSerial), "✅")
				return true
			}
		}

		attempt++
		if attempt%6 == 0 { // Log every 30 seconds
			minutesWaited := attempt / 12
			a.log(fmt.Sprintf("Still waiting for device %s... (%d/%d minutes)",
				deviceSerial, minutesWaited, maxWaitMinutes), "⌛")
		}
		time.Sleep(5 * time.Second)
	}

	a.log(fmt.Sprintf("Timeout waiting for device %s to be ready after %d minutes",
		deviceSerial, maxWaitMinutes), "⏰")
	return false
}

// checkExistingLockScreen checks if device has any lock screen configured
func (a *AndroidLockScreenDisabler) checkExistingLockScreen(deviceSerial string) (bool, string) {
	a.log(fmt.Sprintf("Checking if device %s has existing lock screen configured...", deviceSerial), "🔍")

	// Method 1: Check keyguard state
	success, output, _ := a.runADBCommand("shell dumpsys trust", deviceSerial)
	if success && output != "" {
		if strings.Contains(strings.ToLower(output), "isdevicesecure=true") ||
			strings.Contains(strings.ToLower(output), "iskeyguardsecure=true") {
			return true, "Device has secure lock screen (detected via trust manager)"
		}
	}

	// Method 2: Check lock pattern/PIN/password settings
	lockScreenDisabledLockSettingsMethod := false
	success, output, _ = a.runADBCommand("shell locksettings get-disabled", deviceSerial)
	if success {
		lockScreenDisabledLockSettingsMethod = strings.Contains(strings.ToLower(output), "true")
		if !lockScreenDisabledLockSettingsMethod {
			return true, "Device has lock configured (detected via locksettings)"
		}
	}

	// Method 3: Check keyguard manager
	success, output, _ = a.runADBCommand("shell dumpsys activity services KeyguardService", deviceSerial)
	if success && output != "" {
		if strings.Contains(strings.ToLower(output), "secure=true") ||
			strings.Contains(strings.ToLower(output), "enabled=true") {
			return true, "Device has keyguard enabled (detected via KeyguardService)"
		}
	}

	// Method 4: Check lock settings in secure database
	lockMethods := []string{
		"shell settings get secure lock_pattern_enabled",
		"shell settings get secure lockscreen.password_type",
		"shell settings get secure lockscreen.disabled",
	}

	for _, method := range lockMethods {
		success, output, _ := a.runADBCommand(method, deviceSerial)
		if success && output != "" && output != "null" {
			if strings.Contains(method, "lock_pattern_enabled") && output == "1" {
				return true, "Device has lock pattern enabled"
			}
			if strings.Contains(method, "password_type") && output != "0" {
				return true, fmt.Sprintf("Device has password type configured (type: %s)", output)
			}
			if strings.Contains(method, "lockscreen.disabled") && output == "0" && !lockScreenDisabledLockSettingsMethod {
				return true, "Lock screen is explicitly enabled in settings"
			}
		}
	}

	// Method 5: Check device policy manager for admin locks
	success, output, _ = a.runADBCommand("shell dumpsys device_policy", deviceSerial)
	if success && output != "" {
		if strings.Contains(strings.ToLower(output), "passwordquality") ||
			strings.Contains(strings.ToLower(output), "minimumpasswordlength") {
			return true, "Device has admin-enforced password policy"
		}
	}

	return false, "No lock screen detected"
}

// checkLockScreenStatus checks if device is showing lock screen
func (a *AndroidLockScreenDisabler) checkLockScreenStatus(deviceSerial string) (bool, error) {
	a.log(fmt.Sprintf("Checking lock screen status on device %s...", deviceSerial), "🔍")

	// Method 1: Check if keyguard is showing
	success, output, _ := a.runADBCommand("shell dumpsys window", deviceSerial)
	if success && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "mdreaminglockscreen") ||
				strings.Contains(lowerLine, "mshowingleckscreen") ||
				strings.Contains(lowerLine, "keyguardcontroller") {
				if strings.Contains(lowerLine, "mshowingleckscreen=true") ||
					strings.Contains(lowerLine, "mdreaminglockscreen=true") ||
					strings.Contains(lowerLine, "keyguardshowing=true") {
					return true, nil // Lock screen is showing
				}
			}
		}
	}

	// Method 2: Check power manager state
	success, output, _ = a.runADBCommand("shell dumpsys power", deviceSerial)
	if success && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "mwakefulness") || strings.Contains(lowerLine, "display power") {
				if strings.Contains(lowerLine, "asleep") || strings.Contains(lowerLine, "dozing") {
					return true, nil // Device is locked/sleeping
				}
			}
		}
	}

	// Method 3: Try to get current activity (may fail if locked)
	success, output, _ = a.runADBCommand("shell dumpsys activity activities", deviceSerial)
	if success && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "mresumedactivity") || strings.Contains(lowerLine, "mfocusedactivity") {
				// If we can get activity info and it's not a lock screen activity, likely unlocked
				if !strings.Contains(lowerLine, "keyguard") &&
					!strings.Contains(lowerLine, "lockscreen") &&
					!strings.Contains(lowerLine, "bouncer") {
					return false, nil // No lock screen detected
				}
			}
		}
	}

	// Method 4: Check settings values
	success, output, _ = a.runADBCommand("shell settings get secure lockscreen.disabled", deviceSerial)
	if success && output == "1" {
		return false, nil // Lock screen is disabled in settings
	}

	success, output, _ = a.runADBCommand("shell locksettings get-disabled", deviceSerial)
	if success && strings.Contains(strings.ToLower(output), "true") {
		return false, nil // Lock screen is disabled via locksettings
	}

	// If we can't determine definitively, assume locked for safety
	return true, fmt.Errorf("unable to determine lock screen status definitively")
}

// validateLockScreenRemoval validates that lock screen has been successfully removed after reboot
func (a *AndroidLockScreenDisabler) validateLockScreenRemoval(deviceSerial string) bool {
	a.log(fmt.Sprintf("Validating lock screen removal on device %s...", deviceSerial), "🔍")

	// Wait a moment for UI to stabilize
	time.Sleep(3 * time.Second)

	// Check lock screen status
	isLocked, err := a.checkLockScreenStatus(deviceSerial)

	if err != nil {
		a.log(fmt.Sprintf("Warning: Could not definitively determine lock screen status on device %s: %v",
			deviceSerial, err), "⚠️")
		// Try to wake up the device and check again
		a.runADBCommand("shell input keyevent KEYCODE_WAKEUP", deviceSerial)
		time.Sleep(2 * time.Second)

		isLocked, err = a.checkLockScreenStatus(deviceSerial)
		if err != nil {
			a.log(fmt.Sprintf("Still unable to determine lock screen status on device %s", deviceSerial), "⚠️")
			return false
		}
	}

	if !isLocked {
		a.log(fmt.Sprintf("✅ Lock screen successfully removed on device %s!", deviceSerial), "🎉")
		return true
	} else {
		a.log(fmt.Sprintf("❌ Lock screen is still present on device %s", deviceSerial), "😞")
		return false
	}
}

// disableLockscreenOnDeviceAsync processes a single device asynchronously
func (a *AndroidLockScreenDisabler) disableLockscreenOnDeviceAsync(deviceSerial string, stats *ProcessingStats, wg *sync.WaitGroup) {
	defer wg.Done()

	// Add device identifier to logs for better tracking in concurrent execution
	deviceTag := fmt.Sprintf("[%s]", deviceSerial)

	a.log(fmt.Sprintf("%s Starting lock screen disable process", deviceTag), "🚀")

	// Get device info
	deviceInfo := a.getDeviceInfo(deviceSerial)
	a.log(fmt.Sprintf("%s Device: %s %s (Android %s, API %s)", deviceTag,
		deviceInfo.Manufacturer, deviceInfo.Model, deviceInfo.AndroidVersion, deviceInfo.APILevel), "📋")

	// Check permissions
	if !a.checkDevicePermissions(deviceSerial) {
		a.log(fmt.Sprintf("%s Insufficient permissions. "+
			"Make sure USB debugging is enabled and device is authorized.", deviceTag), "❌")
		stats.addFailedDevice(deviceSerial)
		return
	}

	// Check if device has existing lock screen configured
	hasLock, lockType := a.checkExistingLockScreen(deviceSerial)
	if !hasLock {
		a.log(fmt.Sprintf("%s No lock screen detected on device. Skipping lock screen disable process.", deviceTag), "ℹ️")
		a.log(fmt.Sprintf("%s Device is already unlocked or has no lock configured", deviceTag), "✅")
		stats.incrementSuccess()
		return
	}

	a.log(fmt.Sprintf("%s Lock screen detected: %s", deviceTag, lockType), "🔒")
	a.log(fmt.Sprintf("%s Proceeding with lock screen disable process...", deviceTag), "🚀")

	// Try each method until one succeeds
	methods := []func(string) bool{
		a.disableLockscreenMethod1,
		a.disableLockscreenMethod2,
		a.disableLockscreenMethod3,
		a.disableLockscreenMethod4,
	}

	success := false
	for i, method := range methods {
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.log(fmt.Sprintf("%s Method %d crashed: %v", deviceTag, i+1, r), "💥")
				}
			}()

			if method(deviceSerial) {
				success = true
				return
			}
			time.Sleep(1 * time.Second) // Brief pause between methods
		}()

		if success {
			break
		}
	}

	if !success {
		a.log(fmt.Sprintf("%s All methods failed", deviceTag), "😞")
		stats.addFailedDevice(deviceSerial)
		return
	}

	// Wait a moment for settings to take effect
	time.Sleep(2 * time.Second)

	// Reboot the device to apply changes
	a.log(fmt.Sprintf("%s Rebooting device to apply lock screen changes...", deviceTag), "🔄")

	if !a.rebootDevice(deviceSerial) {
		a.log(fmt.Sprintf("%s Failed to reboot device, but lock screen settings were applied", deviceTag), "⚠️")
		stats.incrementSuccess()
		return
	}

	// Wait for device to be ready after reboot (max 5 minutes)
	a.log(fmt.Sprintf("%s Waiting for device to be ready after reboot (up to 5 minutes)...", deviceTag), "⏳")
	if !a.waitForDeviceReady(deviceSerial, 5) {
		a.log(fmt.Sprintf("%s Device did not become ready within 5 minutes after reboot", deviceTag), "⏰")
		stats.addFailedDevice(deviceSerial)
		return
	}

	// Validate that lock screen has been removed
	if a.validateLockScreenRemoval(deviceSerial) {
		a.log(fmt.Sprintf("%s Successfully disabled and validated lock screen removal! 🎉", deviceTag), "🎊")
		stats.incrementSuccess()
	} else {
		a.log(fmt.Sprintf("%s Lock screen settings were applied, but validation failed after reboot", deviceTag), "⚠️")
		// Still count as success since we successfully applied the settings
		stats.incrementSuccess()
	}
}

// run is the main execution method
func (a *AndroidLockScreenDisabler) run() {
	a.log("Android Lock Screen Disabler Starting...", "🚀")
	a.log(strings.Repeat("=", 50), "")

	// Check ADB availability
	if !a.checkADBAvailability() {
		a.log("Please install ADB and ensure it's in your PATH.", "💡")
		os.Exit(1)
	}

	// Get connected devices
	devices := a.getConnectedDevices()
	if len(devices) == 0 {
		a.log("Please connect at least one Android device with USB debugging enabled.", "💡")
		os.Exit(1)
	}

	// Process each device concurrently
	stats := &ProcessingStats{
		totalDevices: len(devices),
	}

	var wg sync.WaitGroup

	a.log(fmt.Sprintf("Processing %d device(s) concurrently...", len(devices)), "🚀")
	a.log(strings.Repeat("-", 50), "")

	// Start processing all devices in parallel
	for _, device := range devices {
		wg.Add(1)
		go a.disableLockscreenOnDeviceAsync(device, stats, &wg)
	}

	// Wait for all goroutines to complete
	a.log("Waiting for all devices to complete processing...", "⏳")
	wg.Wait()

	// Get final statistics
	successCount, failedDevices, totalDevices := stats.getStats()

	// Summary
	a.log("\n"+strings.Repeat("=", 50), "")
	a.log("EXECUTION SUMMARY", "📊")
	a.log(strings.Repeat("=", 50), "")
	a.log(fmt.Sprintf("Total devices processed: %d", totalDevices), "📱")
	a.log(fmt.Sprintf("Successfully disabled: %d", successCount), "✅")
	a.log(fmt.Sprintf("Failed: %d", len(failedDevices)), "❌")

	if len(failedDevices) > 0 {
		a.log(fmt.Sprintf("Failed devices: %s", strings.Join(failedDevices, ", ")), "⚠️")
		a.log("\nTroubleshooting tips for failed devices:", "💡")
		a.log("• Ensure USB debugging is enabled", "")
		a.log("• Check if device requires authorization", "")
		a.log("• Try enabling 'Settings > Developer Options > Disable permission monitoring'", "")
		a.log("• Some devices may have policy restrictions", "")
	}

	if successCount > 0 {
		a.log(fmt.Sprintf("\n🎉 Successfully processed %d device(s)!", successCount), "🎊")
	}

	a.log("\nScript completed!", "🏁")
}

func main() {
	// Handle Ctrl+C gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n\n⛔ Script interrupted by user. Exiting...")
		os.Exit(0)
	}()

	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n💥 Unexpected error: %v\n", r)
			os.Exit(1)
		}
	}()

	// Parse command line arguments
	var devicesFlag = flag.String("devices", "", "Space-separated list of device UDIDs to process (optional). If not specified, all connected devices will be processed.")
	var helpFlag = flag.Bool("help", false, "Show help information")
	flag.Parse()

	// Show help if requested
	if *helpFlag {
		fmt.Println("Android Lock Screen Disabler")
		fmt.Println("============================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run disable-lock-screen.go [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -devices string")
		fmt.Println("        Space-separated list of device UDIDs to process (optional)")
		fmt.Println("        Example: -devices \"device1 device2 device3\"")
		fmt.Println("  -help")
		fmt.Println("        Show this help information")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Process all connected devices:")
		fmt.Println("  go run disable-lock-screen.go")
		fmt.Println()
		fmt.Println("  # Process specific devices:")
		fmt.Println("  go run disable-lock-screen.go -devices \"ABC123DEF456 789GHI012JKL\"")
		fmt.Println()
		fmt.Println("  # List connected devices to get their UDIDs:")
		fmt.Println("  adb devices")
		return
	}

	// Parse target devices from command line argument
	var targetDevices []string
	if *devicesFlag != "" {
		targetDevices = strings.Fields(*devicesFlag)
		fmt.Printf("🎯 Target devices specified: %s\n", strings.Join(targetDevices, ", "))
	}

	// Create and run the disabler
	disabler := NewAndroidLockScreenDisabler(targetDevices)
	disabler.run()
}
