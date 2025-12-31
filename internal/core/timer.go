package core

import "time"

// Timer manages activity timing
type Timer struct {
	activityWindow time.Duration
	lastActive     time.Time
	isActive       bool
}

// NewTimer creates a new timer
func NewTimer(activityWindow time.Duration) *Timer {
	return &Timer{
		activityWindow: activityWindow,
		lastActive:     time.Now(),
		isActive:       true,
	}
}

// Update updates the timer with the current idle time
func (t *Timer) Update(idleTime time.Duration) {
	if idleTime < t.activityWindow {
		t.lastActive = time.Now()
		t.isActive = true
	} else {
		t.isActive = false
	}
}

// IsActive returns true if currently active
func (t *Timer) IsActive() bool {
	return t.isActive
}

// GetActiveDuration returns the duration since last active
func (t *Timer) GetActiveDuration() time.Duration {
	if t.isActive {
		return time.Since(t.lastActive)
	}
	return 0
}