package dlock

import (
	"fmt"
	"time"
)

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

// DisableLockScreen attempts to disable lock screen using all available methods
func (a *AndroidLockScreenDisabler) DisableLockScreen(deviceSerial string) bool {
	// Try each method until one succeeds
	methods := []func(string) bool{
		a.disableLockscreenMethod1,
		a.disableLockscreenMethod2,
		a.disableLockscreenMethod3,
		a.disableLockscreenMethod4,
	}

	for i, method := range methods {
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.log(fmt.Sprintf("Method %d crashed: %v", i+1, r), "💥")
				}
			}()

			if method(deviceSerial) {
				return
			}
			time.Sleep(1 * time.Second) // Brief pause between methods
		}()
	}

	return false
}
