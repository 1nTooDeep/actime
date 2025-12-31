package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	// Default logger instance
	defaultLogger *slog.Logger
)

// Init initializes the logger with the specified configuration
func Init(level string, file string, maxSizeMB int, maxBackups int, maxAgeDays int) error {
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
	logFile, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create handler with both file and stdout
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: logLevel,
	})

	// Set default logger
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	return nil
}

// GetLogger returns the default logger instance
func GetLogger() *slog.Logger {
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
	// TODO: Implement log rotation and cleanup
	return nil
}