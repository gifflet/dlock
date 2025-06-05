package dlock

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

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

// CheckADBAvailability checks if ADB is available in the system
func (a *AndroidLockScreenDisabler) CheckADBAvailability() bool {
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

// GetConnectedDevices gets list of connected Android devices
func (a *AndroidLockScreenDisabler) GetConnectedDevices() []string {
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

// GetDeviceInfo gets device information
func (a *AndroidLockScreenDisabler) GetDeviceInfo(deviceSerial string) DeviceInfo {
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

// RebootDevice reboots the Android device
func (a *AndroidLockScreenDisabler) RebootDevice(deviceSerial string) bool {
	a.log(fmt.Sprintf("Rebooting device %s...", deviceSerial), "🔄")

	success, _, errorMsg := a.runADBCommand("reboot", deviceSerial)

	if success {
		a.log(fmt.Sprintf("Reboot command sent to device %s", deviceSerial), "✅")
		return true
	}

	a.log(fmt.Sprintf("Failed to reboot device %s: %s", deviceSerial, errorMsg), "❌")
	return false
}

// WaitForDeviceReady waits for device to be ready after reboot
func (a *AndroidLockScreenDisabler) WaitForDeviceReady(deviceSerial string, maxWaitMinutes int) bool {
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
