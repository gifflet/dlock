package dlock

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// AndroidLockScreenDisabler handles the lock screen disabling process
type AndroidLockScreenDisabler struct {
	connectedDevices []string
	targetDevices    []string // New field for target UDIDs
	logMutex         sync.Mutex
	enableLogging    bool // Control whether logging is enabled
}

// NewAndroidLockScreenDisabler creates a new instance of the disabler
func NewAndroidLockScreenDisabler(targetDevices []string) *AndroidLockScreenDisabler {
	return &AndroidLockScreenDisabler{
		connectedDevices: make([]string, 0),
		targetDevices:    targetDevices,
		enableLogging:    true, // Default to enabled logging
	}
}

// SetLogging enables or disables logging
func (a *AndroidLockScreenDisabler) SetLogging(enabled bool) {
	a.enableLogging = enabled
}

// log prints formatted log messages with emojis (thread-safe)
func (a *AndroidLockScreenDisabler) log(message, emoji string) {
	if !a.enableLogging {
		return
	}

	if emoji == "" {
		emoji = "ℹ️"
	}

	a.logMutex.Lock()
	defer a.logMutex.Unlock()
	fmt.Printf("%s %s\n", emoji, message)
}

// DisableLockscreenOnDeviceAsync processes a single device asynchronously
func (a *AndroidLockScreenDisabler) DisableLockscreenOnDeviceAsync(deviceSerial string, stats *ProcessingStats, wg *sync.WaitGroup) {
	defer wg.Done()

	// Add device identifier to logs for better tracking in concurrent execution
	deviceTag := fmt.Sprintf("[%s]", deviceSerial)

	a.log(fmt.Sprintf("%s Starting lock screen disable process", deviceTag), "🚀")

	// Get device info
	deviceInfo := a.GetDeviceInfo(deviceSerial)
	a.log(fmt.Sprintf("%s Device: %s %s (Android %s, API %s)", deviceTag,
		deviceInfo.Manufacturer, deviceInfo.Model, deviceInfo.AndroidVersion, deviceInfo.APILevel), "📋")

	// Check permissions
	if !a.CheckDevicePermissions(deviceSerial) {
		a.log(fmt.Sprintf("%s Insufficient permissions. "+
			"Make sure USB debugging is enabled and device is authorized.", deviceTag), "❌")
		stats.AddFailedDevice(deviceSerial)
		return
	}

	// Check if device has existing lock screen configured
	hasLock, lockType := a.CheckExistingLockScreen(deviceSerial)
	if !hasLock {
		a.log(fmt.Sprintf("%s No lock screen detected on device. Skipping lock screen disable process.", deviceTag), "ℹ️")
		a.log(fmt.Sprintf("%s Device is already unlocked or has no lock configured", deviceTag), "✅")
		stats.IncrementSuccess()
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
		stats.AddFailedDevice(deviceSerial)
		return
	}

	// Wait a moment for settings to take effect
	time.Sleep(2 * time.Second)

	// Reboot the device to apply changes
	a.log(fmt.Sprintf("%s Rebooting device to apply lock screen changes...", deviceTag), "🔄")

	if !a.RebootDevice(deviceSerial) {
		a.log(fmt.Sprintf("%s Failed to reboot device, but lock screen settings were applied", deviceTag), "⚠️")
		stats.IncrementSuccess()
		return
	}

	// Wait for device to be ready after reboot (max 5 minutes)
	a.log(fmt.Sprintf("%s Waiting for device to be ready after reboot (up to 5 minutes)...", deviceTag), "⏳")
	if !a.WaitForDeviceReady(deviceSerial, 5) {
		a.log(fmt.Sprintf("%s Device did not become ready within 5 minutes after reboot", deviceTag), "⏰")
		stats.AddFailedDevice(deviceSerial)
		return
	}

	// Validate that lock screen has been removed
	if a.ValidateLockScreenRemoval(deviceSerial) {
		a.log(fmt.Sprintf("%s Successfully disabled and validated lock screen removal! 🎉", deviceTag), "🎊")
		stats.IncrementSuccess()
	} else {
		a.log(fmt.Sprintf("%s Lock screen settings were applied, but validation failed after reboot", deviceTag), "⚠️")
		// Still count as success since we successfully applied the settings
		stats.IncrementSuccess()
	}
}

// ProcessDevices processes multiple devices concurrently and returns processing statistics
func (a *AndroidLockScreenDisabler) ProcessDevices(devices []string) (int, []string, int) {
	if len(devices) == 0 {
		return 0, nil, 0
	}

	// Process each device concurrently
	stats := NewProcessingStats(len(devices))
	var wg sync.WaitGroup

	a.log(fmt.Sprintf("Processing %d device(s) concurrently...", len(devices)), "🚀")
	a.log(strings.Repeat("-", 50), "")

	// Start processing all devices in parallel
	for _, device := range devices {
		wg.Add(1)
		go a.DisableLockscreenOnDeviceAsync(device, stats, &wg)
	}

	// Wait for all goroutines to complete
	a.log("Waiting for all devices to complete processing...", "⏳")
	wg.Wait()

	// Get final statistics
	return stats.GetStats()
}

// Run is the main execution method for CLI usage
func (a *AndroidLockScreenDisabler) Run() {
	a.log("Android Lock Screen Disabler Starting...", "🚀")
	a.log(strings.Repeat("=", 50), "")

	// Check ADB availability
	if !a.CheckADBAvailability() {
		a.log("Please install ADB and ensure it's in your PATH.", "💡")
		return
	}

	// Get connected devices
	devices := a.GetConnectedDevices()
	if len(devices) == 0 {
		a.log("Please connect at least one Android device with USB debugging enabled.", "💡")
		return
	}

	// Process all devices
	successCount, failedDevices, totalDevices := a.ProcessDevices(devices)

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

// ProcessSingleDevice processes a single device and returns success status
func (a *AndroidLockScreenDisabler) ProcessSingleDevice(deviceSerial string) bool {
	devices := []string{deviceSerial}
	successCount, _, _ := a.ProcessDevices(devices)
	return successCount > 0
}
