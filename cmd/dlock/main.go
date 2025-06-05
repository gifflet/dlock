package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gifflet/dlock/pkg/dlock"
)

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
		fmt.Println("  dlock [options]")
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
		fmt.Println("  dlock")
		fmt.Println()
		fmt.Println("  # Process specific devices:")
		fmt.Println("  dlock -devices \"ABC123DEF456 789GHI012JKL\"")
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
	disabler := dlock.NewAndroidLockScreenDisabler(targetDevices)
	disabler.Run()
}
