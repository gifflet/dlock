package dlock

import "sync"

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

// IncrementSuccess safely increments the success counter
func (ps *ProcessingStats) IncrementSuccess() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.successCount++
}

// AddFailedDevice safely adds a device to the failed list
func (ps *ProcessingStats) AddFailedDevice(deviceSerial string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.failedDevices = append(ps.failedDevices, deviceSerial)
}

// GetStats safely retrieves current statistics
func (ps *ProcessingStats) GetStats() (int, []string, int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	failedCopy := make([]string, len(ps.failedDevices))
	copy(failedCopy, ps.failedDevices)
	return ps.successCount, failedCopy, ps.totalDevices
}

// NewProcessingStats creates a new ProcessingStats instance
func NewProcessingStats(totalDevices int) *ProcessingStats {
	return &ProcessingStats{
		totalDevices: totalDevices,
	}
}
