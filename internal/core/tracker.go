package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/weii/actime/internal/platform"
	"github.com/weii/actime/pkg/logger"
)

// Tracker tracks application usage
type Tracker struct {
	config          *Config
	detector        platform.Detector
	timer           *Timer
	session         *Session
	sessionMutex    sync.RWMutex
	running         bool
	stopChan        chan struct{}
	checkInterval   time.Duration
	activityWindow  time.Duration
}

// NewTracker creates a new tracker
func NewTracker(cfg *Config, detector platform.Detector) *Tracker {
	return &Tracker{
		config:         cfg,
		detector:       detector,
		timer:          NewTimer(cfg.Monitor.ActivityWindow),
		checkInterval:  cfg.Monitor.CheckInterval,
		activityWindow: cfg.Monitor.ActivityWindow,
		stopChan:       make(chan struct{}),
	}
}

// Start starts tracking
func (t *Tracker) Start() error {
	if t.running {
		return fmt.Errorf("tracker is already running")
	}

	log := logger.GetLogger()
	log.Info("Starting tracker")

	t.running = true

	// Start tracking loop
	go t.trackLoop()

	return nil
}

// Stop stops tracking
func (t *Tracker) Stop() error {
	if !t.running {
		return fmt.Errorf("tracker is not running")
	}

	log := logger.GetLogger()
	log.Info("Stopping tracker")

	t.running = false
	close(t.stopChan)

	// Finalize current session
	t.sessionMutex.Lock()
	if t.session != nil {
		t.session.EndTime = time.Now()
		log.Info("Finalizing session",
			"app", t.session.AppName,
			"duration", t.session.DurationSeconds)
		t.session = nil
	}
	t.sessionMutex.Unlock()

	return nil
}

// trackLoop is the main tracking loop
func (t *Tracker) trackLoop() {
	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.tick()
		}
	}
}

// tick performs a single tracking check
func (t *Tracker) tick() {
	// Check if screen is locked
	locked, err := t.detector.IsScreenLocked()
	if err != nil {
		logger.GetLogger().Error("Failed to check screen lock status", "error", err)
		return
	}

	if locked {
		logger.GetLogger().Debug("Screen is locked, pausing tracking")
		t.pauseSession()
		return
	}

	// Get idle time
	idleTime, err := t.detector.GetIdleTime()
	if err != nil {
		logger.GetLogger().Error("Failed to get idle time", "error", err)
		return
	}

	// Update timer
	t.timer.Update(idleTime)

	// Check if system is active
	if !t.timer.IsActive() {
		logger.GetLogger().Debug("System is idle, pausing tracking", "idle_time", idleTime)
		t.pauseSession()
		return
	}

	// Get active window
	window, err := t.detector.GetActiveWindow()
	if err != nil {
		logger.GetLogger().Error("Failed to get active window", "error", err)
		return
	}

	// Update session
	t.updateSession(window)
}

// updateSession updates the current session based on the active window
func (t *Tracker) updateSession(window *platform.WindowInfo) {
	t.sessionMutex.Lock()
	defer t.sessionMutex.Unlock()

	now := time.Now()

	// Check if we need to start a new session
	if t.session == nil {
		// Start new session
		t.session = &Session{
			AppName:     window.AppName,
			WindowTitle: window.WindowTitle,
			StartTime:   now,
			EndTime:     now,
		}
		logger.GetLogger().Info("Started new session",
			"app", window.AppName,
			"title", window.WindowTitle)
	} else {
		// Check if window changed
		if t.session.AppName != window.AppName || t.session.WindowTitle != window.WindowTitle {
			// Finalize current session
			t.session.EndTime = now
			logger.GetLogger().Info("Ended session",
				"app", t.session.AppName,
				"duration", t.session.DurationSeconds)

			// Start new session
			t.session = &Session{
				AppName:     window.AppName,
				WindowTitle: window.WindowTitle,
				StartTime:   now,
				EndTime:     now,
			}
			logger.GetLogger().Info("Started new session",
				"app", window.AppName,
				"title", window.WindowTitle)
		} else {
			// Update existing session
			t.session.EndTime = now
			t.session.DurationSeconds++
		}
	}
}

// pauseSession pauses the current session
func (t *Tracker) pauseSession() {
	t.sessionMutex.Lock()
	defer t.sessionMutex.Unlock()

	if t.session != nil {
		t.session.EndTime = time.Now()
		logger.GetLogger().Info("Paused session",
			"app", t.session.AppName,
			"duration", t.session.DurationSeconds)
		t.session = nil
	}
}

// GetCurrentSession returns the current session (if any)
func (t *Tracker) GetCurrentSession() *Session {
	t.sessionMutex.RLock()
	defer t.sessionMutex.RUnlock()

	if t.session == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	sessionCopy := *t.session
	return &sessionCopy
}

// IsRunning returns true if the tracker is running
func (t *Tracker) IsRunning() bool {
	return t.running
}