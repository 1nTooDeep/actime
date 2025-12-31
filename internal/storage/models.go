package storage

import "time"

// Session represents a usage session in the database
type Session struct {
	ID              int64     `db:"id"`
	AppName         string    `db:"app_name"`
	WindowTitle     string    `db:"window_title"`
	StartTime       time.Time `db:"start_time"`
	EndTime         time.Time `db:"end_time"`
	DurationSeconds int64     `db:"duration_seconds"`
	CreatedAt       time.Time `db:"created_at"`
}

// DailyStats represents daily usage statistics in the database
type DailyStats struct {
	ID           int64     `db:"id"`
	AppName      string    `db:"app_name"`
	Date         time.Time `db:"date"`
	TotalSeconds int64     `db:"total_seconds"`
}

// StatsQuery represents parameters for querying statistics
type StatsQuery struct {
	AppName string
	StartDate time.Time
	EndDate time.Time
	Limit int
}

// ExportData represents data for export
type ExportData struct {
	AppName      string
	TotalSeconds int64
	Sessions     []Session
}