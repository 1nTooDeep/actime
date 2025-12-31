package platform

import "time"

// Detector defines the interface for platform-specific detection
type Detector interface {
	// GetActiveWindow returns information about the currently active window
	GetActiveWindow() (*WindowInfo, error)

	// GetIdleTime returns the time since last user input
	GetIdleTime() (time.Duration, error)

	// Initialize initializes the detector
	Initialize() error

	// Close cleans up resources
	Close() error

	// IsScreenLocked returns true if the screen is locked
	IsScreenLocked() (bool, error)
}

// WindowInfo contains information about a window
type WindowInfo struct {
	AppName     string
	WindowTitle string
	PID         int32
}

// PlatformDetector is the global detector instance
var PlatformDetector Detector