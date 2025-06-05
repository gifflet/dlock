package main

import (
	"fmt"
	"log"

	"github.com/gifflet/dlock/pkg/dlock"
)

func main() {
	// Example 1: Process all connected devices
	fmt.Println("=== Example 1: Process all connected devices ===")

	disabler := dlock.NewAndroidLockScreenDisabler(nil)

	// Check ADB availability first
	if !disabler.CheckADBAvailability() {
		log.Fatal("ADB is not available")
	}

	// Get connected devices
	devices := disabler.GetConnectedDevices()
	if len(devices) == 0 {
		log.Fatal("No devices connected")
	}

	fmt.Printf("Found %d devices: %v\n", len(devices), devices)

	// Process all devices
	successCount, failedDevices, totalDevices := disabler.ProcessDevices(devices)

	fmt.Printf("Results: %d/%d successful, failed: %v\n", successCount, totalDevices, failedDevices)

	// Example 2: Process specific devices
	fmt.Println("\n=== Example 2: Process specific devices ===")

	targetDevices := []string{"device_serial_1", "device_serial_2"}
	specificDisabler := dlock.NewAndroidLockScreenDisabler(targetDevices)

	// Disable logging for cleaner output
	specificDisabler.SetLogging(false)

	devices = specificDisabler.GetConnectedDevices()
	successCount, failedDevices, totalDevices = specificDisabler.ProcessDevices(devices)

	fmt.Printf("Targeted processing results: %d/%d successful, failed: %v\n",
		successCount, totalDevices, failedDevices)

	// Example 3: Process single device
	fmt.Println("\n=== Example 3: Process single device ===")

	if len(devices) > 0 {
		singleDisabler := dlock.NewAndroidLockScreenDisabler(nil)
		success := singleDisabler.ProcessSingleDevice(devices[0])
		fmt.Printf("Single device processing successful: %t\n", success)
	}

	// Example 4: Get device information
	fmt.Println("\n=== Example 4: Get device information ===")

	if len(devices) > 0 {
		infoDisabler := dlock.NewAndroidLockScreenDisabler(nil)
		deviceInfo := infoDisabler.GetDeviceInfo(devices[0])

		fmt.Printf("Device Information:\n")
		fmt.Printf("  Model: %s\n", deviceInfo.Model)
		fmt.Printf("  Manufacturer: %s\n", deviceInfo.Manufacturer)
		fmt.Printf("  Android Version: %s\n", deviceInfo.AndroidVersion)
		fmt.Printf("  API Level: %s\n", deviceInfo.APILevel)

		// Check if device has lock screen
		hasLock, lockType := infoDisabler.CheckExistingLockScreen(devices[0])
		fmt.Printf("  Has Lock Screen: %t (%s)\n", hasLock, lockType)
	}
}
