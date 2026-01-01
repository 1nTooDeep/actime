package core

import "time"

// Session represents a usage session for an application
type Session struct {
	ID              int64
	AppName         string
	WindowTitle     string
	StartTime       time.Time
	EndTime         time.Time
	DurationSeconds int64
	CreatedAt       time.Time
}

// DailyStats represents daily usage statistics for an application
type DailyStats struct {
	ID           int64
	AppName      string
	Date         time.Time
	TotalSeconds int64
}

// WindowInfo contains information about the active window
type WindowInfo struct {
	AppName     string
	WindowTitle string
	PID         int32
}

// ActivityStatus represents the current activity status
type ActivityStatus struct {
	IsActive      bool
	IdleTime      time.Duration
	CurrentWindow *WindowInfo
	LastActive    time.Time
}

// Config represents the application configuration
type Config struct {
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Monitor struct {
		CheckInterval  time.Duration `yaml:"check_interval"`
		ActivityWindow time.Duration `yaml:"activity_window"`
		IdleTimeout    time.Duration `yaml:"idle_timeout"`
	} `yaml:"monitor"`

	Logging struct {
		Level       string `yaml:"level"`
		File        string `yaml:"file"`
		MaxSizeMB   int    `yaml:"max_size_mb"`
		MaxBackups  int    `yaml:"max_backups"`
		MaxAgeDays  int    `yaml:"max_age_days"`
	} `yaml:"logging"`

	Export struct {
		OutputDir     string `yaml:"output_dir"`
		DefaultFormat string `yaml:"default_format"`
	} `yaml:"export"`

	AppMapping struct {
		ProcessNames map[string]string `yaml:"process_names"` // Map process name to display name
		Browsers     []string          `yaml:"browsers"`      // Browser process names
	} `yaml:"app_mapping"`
}

// AppCategory represents application category
type AppCategory string

const (
	CategoryBrowser     AppCategory = "browser"
	CategoryCommunication AppCategory = "communication"
	CategoryDevelopment  AppCategory = "development"
	CategoryOffice       AppCategory = "office"
	CategoryMedia        AppCategory = "media"
	CategorySystem       AppCategory = "system"
	CategoryGame         AppCategory = "game"
	CategoryOther        AppCategory = "other"
)