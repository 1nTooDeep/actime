package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/weii/actime/internal/core"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigPath is the default configuration file path
	DefaultConfigPath = "~/.actime/config.yaml"
)

// Load loads configuration from the specified path
// If the file doesn't exist, returns default configuration
func Load(path string) (*core.Config, error) {
	// Expand ~ to home directory
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		// Return default config
		return getDefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg core.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate and set defaults
	if err := validateAndSetDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}

// Save saves configuration to the specified path
func Save(cfg *core.Config, path string) error {
	// Expand ~ to home directory
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *core.Config {
	homeDir, _ := os.UserHomeDir()

	return &core.Config{
		Database: struct {
			Path string `yaml:"path"`
		}{
			Path: filepath.Join(homeDir, ".actime", "actime.db"),
		},
		Monitor: struct {
			CheckInterval  time.Duration `yaml:"check_interval"`
			ActivityWindow time.Duration `yaml:"activity_window"`
			IdleTimeout    time.Duration `yaml:"idle_timeout"`
		}{
			CheckInterval:  1 * time.Second,
			ActivityWindow: 5 * time.Minute,
			IdleTimeout:    10 * time.Minute,
		},
		Logging: struct {
			Level      string `yaml:"level"`
			File       string `yaml:"file"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
		}{
			Level:      "info",
			File:       filepath.Join(homeDir, ".actime", "actime.log"),
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
		},
		Export: struct {
			OutputDir     string `yaml:"output_dir"`
			DefaultFormat string `yaml:"default_format"`
		}{
			OutputDir:     filepath.Join(homeDir, ".actime", "exports"),
			DefaultFormat: "csv",
		},
	}
}

// validateAndSetDefaults validates configuration and sets defaults
func validateAndSetDefaults(cfg *core.Config) error {
	homeDir, _ := os.UserHomeDir()

	// Validate database path
	if cfg.Database.Path == "" {
		cfg.Database.Path = filepath.Join(homeDir, ".actime", "actime.db")
	}

	// Validate monitor settings
	if cfg.Monitor.CheckInterval == 0 {
		cfg.Monitor.CheckInterval = 1 * time.Second
	}
	if cfg.Monitor.ActivityWindow == 0 {
		cfg.Monitor.ActivityWindow = 5 * time.Minute
	}
	if cfg.Monitor.IdleTimeout == 0 {
		cfg.Monitor.IdleTimeout = 10 * time.Minute
	}

	// Validate logging settings
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.File == "" {
		cfg.Logging.File = filepath.Join(homeDir, ".actime", "actime.log")
	}
	if cfg.Logging.MaxSizeMB == 0 {
		cfg.Logging.MaxSizeMB = 100
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 3
	}
	if cfg.Logging.MaxAgeDays == 0 {
		cfg.Logging.MaxAgeDays = 30
	}

	// Validate export settings
	if cfg.Export.OutputDir == "" {
		cfg.Export.OutputDir = filepath.Join(homeDir, ".actime", "exports")
	}
	if cfg.Export.DefaultFormat == "" {
		cfg.Export.DefaultFormat = "csv"
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, path[1:]), nil
	}
	return path, nil
}