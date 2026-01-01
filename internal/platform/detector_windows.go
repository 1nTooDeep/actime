//go:build windows

package platform

import (
	"fmt"
	"runtime"
)

// NewDetector creates a new platform-specific detector based on the operating system
func NewDetector() (Detector, error) {
	switch runtime.GOOS {
	case "windows":
		detector := NewWindowsDetector()
		if err := detector.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize Windows detector: %w", err)
		}
		return detector, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// InitializePlatformDetector initializes the global platform detector
func InitializePlatformDetector() error {
	detector, err := NewDetector()
	if err != nil {
		return err
	}
	PlatformDetector = detector
	return nil
}

// ClosePlatformDetector closes the global platform detector
func ClosePlatformDetector() error {
	if PlatformDetector != nil {
		return PlatformDetector.Close()
	}
	return nil
}