package dlock

import (
	"fmt"
	"strings"
	"time"
)

// CheckDevicePermissions checks if device has necessary permissions for lock screen modifications
func (a *AndroidLockScreenDisabler) CheckDevicePermissions(deviceSerial string) bool {
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

// CheckExistingLockScreen checks if device has any lock screen configured
func (a *AndroidLockScreenDisabler) CheckExistingLockScreen(deviceSerial string) (bool, string) {
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

// CheckLockScreenStatus checks if device is showing lock screen
func (a *AndroidLockScreenDisabler) CheckLockScreenStatus(deviceSerial string) (bool, error) {
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

// ValidateLockScreenRemoval validates that lock screen has been successfully removed after reboot
func (a *AndroidLockScreenDisabler) ValidateLockScreenRemoval(deviceSerial string) bool {
	a.log(fmt.Sprintf("Validating lock screen removal on device %s...", deviceSerial), "🔍")

	// Wait a moment for UI to stabilize
	time.Sleep(3 * time.Second)

	// Check lock screen status
	isLocked, err := a.CheckLockScreenStatus(deviceSerial)

	if err != nil {
		a.log(fmt.Sprintf("Warning: Could not definitively determine lock screen status on device %s: %v",
			deviceSerial, err), "⚠️")
		// Try to wake up the device and check again
		a.runADBCommand("shell input keyevent KEYCODE_WAKEUP", deviceSerial)
		time.Sleep(2 * time.Second)

		isLocked, err = a.CheckLockScreenStatus(deviceSerial)
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
