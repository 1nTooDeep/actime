package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/weii/actime/internal/config"
	"github.com/weii/actime/internal/storage"
	"github.com/weii/actime/pkg/logger"
)

// Service represents the main service
type Service struct {
	config     *config.Config
	db         *storage.DB
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewService creates a new service instance
func NewService(cfg *config.Config) (*Service, error) {
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

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:  cfg,
		db:      db,
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}, nil
}

// Start starts the service
func (s *Service) Start() error {
	if s.running {
		return fmt.Errorf("service is already running")
	}

	log := logger.GetLogger()
	log.Info("Starting Actime service")

	s.running = true

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring loop
	go s.monitorLoop()

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
			// TODO: Implement monitoring logic
			// 1. Get active window
			// 2. Get idle time
			// 3. Update session tracking
			// 4. Batch write to database
			log.Debug("Monitoring tick")
		}
	}
}

// IsRunning returns true if the service is running
func (s *Service) IsRunning() bool {
	return s.running
}