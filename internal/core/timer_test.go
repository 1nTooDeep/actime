package core

import (
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	activityWindow := 5 * time.Minute
	timer := NewTimer(activityWindow)

	if timer == nil {
		t.Fatal("NewTimer returned nil")
	}

	if timer.activityWindow != activityWindow {
		t.Errorf("Expected activityWindow %v, got %v", activityWindow, timer.activityWindow)
	}

	if !timer.isActive {
		t.Error("Expected timer to be active initially")
	}
}

func TestTimerUpdate(t *testing.T) {
	timer := NewTimer(5 * time.Minute)

	// Test with idle time less than activity window
	timer.Update(1 * time.Minute)
	if !timer.IsActive() {
		t.Error("Expected timer to be active with idle time < activity window")
	}

	// Test with idle time greater than activity window
	timer.Update(6 * time.Minute)
	if timer.IsActive() {
		t.Error("Expected timer to be inactive with idle time > activity window")
	}

	// Test with idle time equal to activity window
	timer.Update(5 * time.Minute)
	// Should be inactive (idle time >= activity window)
	if timer.IsActive() {
		t.Error("Expected timer to be inactive with idle time >= activity window")
	}
}

func TestTimerIsActive(t *testing.T) {
	timer := NewTimer(5 * time.Minute)

	if !timer.IsActive() {
		t.Error("Expected timer to be active initially")
	}

	timer.Update(10 * time.Minute)
	if timer.IsActive() {
		t.Error("Expected timer to be inactive after long idle time")
	}
}

func TestTimerGetActiveDuration(t *testing.T) {
	timer := NewTimer(5 * time.Minute)

	// Update with short idle time
	timer.Update(1 * time.Minute)
	duration := timer.GetActiveDuration()

	if duration == 0 {
		t.Error("Expected non-zero active duration")
	}

	// After long idle time, duration should be 0
	timer.Update(10 * time.Minute)
	duration = timer.GetActiveDuration()

	if duration != 0 {
		t.Error("Expected zero active duration after long idle time")
	}
}