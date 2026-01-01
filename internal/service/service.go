package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/weii/actime/internal/core"
	"github.com/weii/actime/internal/platform"
	"github.com/weii/actime/internal/storage"
	"github.com/weii/actime/pkg/logger"
)

// Service represents the main service
type Service struct {
	config          *core.Config
	db              *storage.DB
	tracker         *core.Tracker
	ctx             context.Context
	cancel          context.CancelFunc
	running         bool
	sessionBuffer   []*storage.Session
	sessionMutex    sync.Mutex
	batchInterval   time.Duration
	batchTicker     *time.Ticker
}

// NewService creates a new service instance
func NewService(cfg *core.Config) (*Service, error) {
	// Initialize logger
	if err := logger.Init(
		cfg.Logging.Level,
		cfg.Logging.File,
		cfg.Logging.MaxSizeMB,
		cfg.Logging.MaxBackups,
		cfg.Logging.MaxAgeDays,
	); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize database
	db, err := storage.NewDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize platform detector
	if err := platform.InitializePlatformDetector(); err != nil {
		return nil, fmt.Errorf("failed to initialize platform detector: %w", err)
	}

	// Initialize tracker
	tracker := core.NewTracker(cfg, platform.PlatformDetector)

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:        cfg,
		db:            db,
		tracker:       tracker,
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		sessionBuffer: make([]*storage.Session, 0),
		batchInterval: 60 * time.Second, // Batch write every 60 seconds
	}, nil
}

// Start starts the service
func (s *Service) Start() error {
	if s.running {
		return fmt.Errorf("service is already running")
	}

	log := logger.GetLogger()
	log.Info("Starting Actime service")

	// Check and lock PID file
	if err := CheckAndLockPIDFile(PIDFile); err != nil {
		return fmt.Errorf("failed to acquire PID lock: %w", err)
	}

	// Start tracker
	if err := s.tracker.Start(); err != nil {
		RemovePIDFile(PIDFile)
		return fmt.Errorf("failed to start tracker: %w", err)
	}

	s.running = true

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring loop
	go s.monitorLoop()

	// Start batch write loop
	s.batchTicker = time.NewTicker(s.batchInterval)
	go s.batchWriteLoop()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Received shutdown signal")

	s.Stop()
	return nil
}

// Stop stops the service
func (s *Service) Stop() error {
	if !s.running {
		return fmt.Errorf("service is not running")
	}

	log := logger.GetLogger()
	log.Info("Stopping Actime service")

	s.running = false
	s.cancel()

	// Stop tracker
	if err := s.tracker.Stop(); err != nil {
		log.Error("Failed to stop tracker", "error", err)
	}

	// Stop batch ticker
	if s.batchTicker != nil {
		s.batchTicker.Stop()
	}

	// Flush remaining sessions
	if err := s.flushSessions(); err != nil {
		log.Error("Failed to flush sessions", "error", err)
	}

	// Close platform detector
	if err := platform.ClosePlatformDetector(); err != nil {
		log.Error("Failed to close platform detector", "error", err)
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Error("Failed to close database", "error", err)
		}
	}

	// Close logger
	if err := logger.Close(); err != nil {
		log.Error("Failed to close logger", "error", err)
	}

	// Remove PID file
	if err := RemovePIDFile(PIDFile); err != nil {
		log.Error("Failed to remove PID file", "error", err)
	}

	log.Info("Service stopped")
	return nil
}

// monitorLoop is the main monitoring loop
func (s *Service) monitorLoop() {
	log := logger.GetLogger()
	ticker := time.NewTicker(s.config.Monitor.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Check for session to buffer
			currentSession := s.tracker.GetCurrentSession()
			if currentSession != nil {
				s.bufferSession(currentSession)
			}
			log.Debug("Monitoring tick")
		}
	}
}

// batchWriteLoop performs periodic batch writes
func (s *Service) batchWriteLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.batchTicker.C:
			if err := s.flushSessions(); err != nil {
				logger.GetLogger().Error("Failed to flush sessions", "error", err)
			}
		}
	}
}

// bufferSession adds a session to the buffer
func (s *Service) bufferSession(session *core.Session) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	// Convert core.Session to storage.Session
	storageSession := &storage.Session{
		AppName:         session.AppName,
		WindowTitle:     session.WindowTitle,
		StartTime:       session.StartTime,
		EndTime:         session.EndTime,
		DurationSeconds: session.DurationSeconds,
	}

	// Check if we already have a session for this app/window in the buffer
	// If yes, update it instead of adding a new one
	for i, bufSession := range s.sessionBuffer {
		if bufSession.AppName == storageSession.AppName &&
			bufSession.WindowTitle == storageSession.WindowTitle &&
			bufSession.EndTime.Equal(storageSession.StartTime) {
			// Update existing session
			s.sessionBuffer[i] = storageSession
			return
		}
	}

	// Add new session to buffer
	s.sessionBuffer = append(s.sessionBuffer, storageSession)
}

// flushSessions writes all buffered sessions to the database
func (s *Service) flushSessions() error {
	s.sessionMutex.Lock()
	sessions := make([]*storage.Session, len(s.sessionBuffer))
	copy(sessions, s.sessionBuffer)
	s.sessionBuffer = s.sessionBuffer[:0] // Clear buffer
	s.sessionMutex.Unlock()

	if len(sessions) == 0 {
		return nil
	}

	log := logger.GetLogger()
	log.Info("Flushing sessions to database", "count", len(sessions))

	// Batch insert sessions
	if err := s.db.BatchInsertSessions(sessions); err != nil {
		return fmt.Errorf("failed to batch insert sessions: %w", err)
	}

	// Update daily statistics
	if err := s.db.UpdateDailyStatsBatch(sessions); err != nil {
		return fmt.Errorf("failed to update daily stats: %w", err)
	}

	return nil
}

// IsRunning returns true if the service is running
func (s *Service) IsRunning() bool {
	return s.running
}