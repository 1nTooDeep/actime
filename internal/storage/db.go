package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
	path string
}

// NewDB creates a new database connection
func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: path,
	}

	// Initialize database schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema creates the database tables if they don't exist
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_name TEXT NOT NULL,
		window_title TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		duration_seconds INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(app_name, window_title, start_time)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_app_name ON sessions(app_name);
	CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time);

	CREATE TABLE IF NOT EXISTS daily_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_name TEXT NOT NULL,
		date DATE NOT NULL,
		total_seconds INTEGER NOT NULL,
		UNIQUE(app_name, date)
	);

	CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// InsertSession inserts a new session into the database
func (db *DB) InsertSession(session *Session) error {
	query := `
	INSERT INTO sessions (app_name, window_title, start_time, end_time, duration_seconds)
	VALUES (?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		session.AppName,
		session.WindowTitle,
		session.StartTime,
		session.EndTime,
		session.DurationSeconds,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	session.ID = id
	return nil
}

// GetDailyStats retrieves daily statistics for the given date range
func (db *DB) GetDailyStats(query *StatsQuery) ([]*DailyStats, error) {
	sqlQuery := `
	SELECT app_name, date, SUM(total_seconds) as total_seconds
	FROM daily_stats
	WHERE 1=1
	`
	args := []interface{}{}

	if !query.StartDate.IsZero() {
		sqlQuery += " AND date >= ?"
		args = append(args, query.StartDate)
	}

	if !query.EndDate.IsZero() {
		sqlQuery += " AND date <= ?"
		args = append(args, query.EndDate)
	}

	if query.AppName != "" {
		sqlQuery += " AND app_name = ?"
		args = append(args, query.AppName)
	}

	sqlQuery += " GROUP BY app_name, date ORDER BY date DESC"

	if query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)
	}

	rows, err := db.conn.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*DailyStats
	for rows.Next() {
		var stat DailyStats
		if err := rows.Scan(&stat.AppName, &stat.Date, &stat.TotalSeconds); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		stats = append(stats, &stat)
	}

	return stats, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// UpdateDailyStats updates or inserts daily statistics
func (db *DB) UpdateDailyStats(appName string, date time.Time, seconds int64) error {
	query := `
	INSERT INTO daily_stats (app_name, date, total_seconds)
	VALUES (?, ?, ?)
	ON CONFLICT(app_name, date) DO UPDATE SET
	total_seconds = total_seconds + ?
	`

	_, err := db.conn.Exec(query, appName, date, seconds, seconds)
	if err != nil {
		return fmt.Errorf("failed to update daily stats: %w", err)
	}

	return nil
}

// BatchInsertSessions inserts or replaces multiple sessions in a single transaction
// Uses INSERT OR REPLACE to avoid duplicate sessions for the same app/window/start_time
func (db *DB) BatchInsertSessions(sessions []*Session) error {
	if len(sessions) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Use INSERT OR REPLACE to avoid duplicates
	// The combination of app_name, window_title, and start_time should be unique
	query := `
	INSERT OR REPLACE INTO sessions (app_name, window_title, start_time, end_time, duration_seconds)
	VALUES (?, ?, ?, ?, ?)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, session := range sessions {
		_, err = stmt.Exec(
			session.AppName,
			session.WindowTitle,
			session.StartTime,
			session.EndTime,
			session.DurationSeconds,
		)
		if err != nil {
			return fmt.Errorf("failed to insert session: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateDailyStatsBatch updates daily statistics for multiple sessions
func (db *DB) UpdateDailyStatsBatch(sessions []*Session) error {
	if len(sessions) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
	INSERT INTO daily_stats (app_name, date, total_seconds)
	VALUES (?, ?, ?)
	ON CONFLICT(app_name, date) DO UPDATE SET
	total_seconds = total_seconds + ?
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, session := range sessions {
		date := session.StartTime.Format("2006-01-02")
		_, err = stmt.Exec(
			session.AppName,
			date,
			session.DurationSeconds,
			session.DurationSeconds,
		)
		if err != nil {
			return fmt.Errorf("failed to update daily stats: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSessions retrieves all sessions within the specified date range
func (db *DB) GetSessions(startDate, endDate time.Time) ([]*Session, error) {
	query := `
	SELECT id, app_name, window_title, start_time, end_time, duration_seconds, created_at
	FROM sessions
	WHERE start_time >= ? AND start_time < ?
	ORDER BY start_time ASC
	`

	rows, err := db.conn.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(
			&session.ID,
			&session.AppName,
			&session.WindowTitle,
			&session.StartTime,
			&session.EndTime,
			&session.DurationSeconds,
			&session.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}