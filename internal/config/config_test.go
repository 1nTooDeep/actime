package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/weii/actime/internal/core"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  path: /tmp/test.db
monitor:
  check_interval: 2s
  activity_window: 10m
  idle_timeout: 15m
logging:
  level: debug
  file: /tmp/test.log
  max_size_mb: 50
  max_backups: 5
  max_age_days: 60
export:
  output_dir: /tmp/exports
  default_format: json
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Expected database path /tmp/test.db, got %s", cfg.Database.Path)
	}

	if cfg.Monitor.CheckInterval != 2*time.Second {
		t.Errorf("Expected check interval 2s, got %v", cfg.Monitor.CheckInterval)
	}

	if cfg.Monitor.ActivityWindow != 10*time.Minute {
		t.Errorf("Expected activity window 10m, got %v", cfg.Monitor.ActivityWindow)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level debug, got %s", cfg.Logging.Level)
	}

	if cfg.Export.DefaultFormat != "json" {
		t.Errorf("Expected default format json, got %s", cfg.Export.DefaultFormat)
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Try to load non-existent config file
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Should return default config
	if cfg == nil {
		t.Fatal("Expected default config, got nil")
	}

	if cfg.Database.Path == "" {
		t.Error("Expected default database path to be set")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := getDefaultConfig()
	cfg.Database.Path = "/tmp/custom.db"

	// Save config
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load and verify
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Database.Path != "/tmp/custom.db" {
		t.Errorf("Expected database path /tmp/custom.db, got %s", loadedCfg.Database.Path)
	}
}

func TestValidateAndSetDefaults(t *testing.T) {
	cfg := &core.Config{}

	// Validate and set defaults
	if err := validateAndSetDefaults(cfg); err != nil {
		t.Fatalf("Failed to validate and set defaults: %v", err)
	}

	// Verify defaults are set
	if cfg.Database.Path == "" {
		t.Error("Expected database path to be set")
	}

	if cfg.Monitor.CheckInterval == 0 {
		t.Error("Expected check interval to be set")
	}

	if cfg.Logging.Level == "" {
		t.Error("Expected log level to be set")
	}
}