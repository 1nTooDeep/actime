package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	// Default logger instance
	defaultLogger *slog.Logger
	// Log file for cleanup
	logFile *os.File
	// Mutex for thread-safe operations
	mu sync.RWMutex
)

// Init initializes the logger with the specified configuration
func Init(level string, file string, maxSizeMB int, maxBackups int, maxAgeDays int) error {
	mu.Lock()
	defer mu.Unlock()

	// Parse log level
	logLevel, err := parseLogLevel(level)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(file)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err = os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer for both file and stdout
	multiWriter := io.MultiWriter(logFile, os.Stdout)

	// Create handler with both file and stdout
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: logLevel,
	})

	// Set default logger
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	return nil
}

// GetLogger returns the default logger instance
func GetLogger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if defaultLogger == nil {
		// Initialize with default settings if not initialized
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		defaultLogger = slog.New(handler)
		slog.SetDefault(defaultLogger)
	}
	return defaultLogger
}

// parseLogLevel parses log level string
func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}

// Close closes the logger and flushes any buffered logs
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		// Sync and close the log file
		if err := logFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync log file: %w", err)
		}
		if err := logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		logFile = nil
	}

	return nil
}

// CheckLogRotation checks if log rotation is needed
// This is a simplified version - for production, consider using a proper log rotation library
func CheckLogRotation(filePath string, maxSizeMB int) error {
	mu.Lock()
	defer mu.Unlock()

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get log file info: %w", err)
	}

	// Check if file size exceeds max size
	maxSize := int64(maxSizeMB) * 1024 * 1024
	if info.Size() < maxSize {
		return nil
	}

	// Rotate log file
	if err := rotateLogFile(filePath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	return nil
}

// rotateLogFile rotates the log file by renaming it and creating a new one
func rotateLogFile(filePath string) error {
	// Close current log file
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			return err
		}
		logFile = nil
	}

	// Find next available backup number
	for i := 1; i <= 100; i++ {
		backupPath := fmt.Sprintf("%s.%d", filePath, i)
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			// Rename current file to backup
			if err := os.Rename(filePath, backupPath); err != nil {
				return err
			}
			break
		}
	}

	// Reopen log file
	var err error
	logFile, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen log file: %w", err)
	}

	return nil
}