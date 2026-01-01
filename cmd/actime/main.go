package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/weii/actime/internal/config"
	"github.com/weii/actime/internal/storage"
)

const (
	Version = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "stats":
		if err := showStats(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "export":
		if err := exportData(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "config":
		if err := showConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("Actime CLI v%s\n", Version)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Actime CLI v%s\n\n", Version)
	fmt.Println("Usage: actime <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  stats    Show usage statistics")
	fmt.Println("  export   Export data to CSV or JSON")
	fmt.Println("  config   Show configuration")
	fmt.Println("  version  Show version information")
	fmt.Println("  help     Show this help message")
}

func showStats() error {
	fmt.Println("Usage Statistics:")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Open database
	db, err := storage.NewDB(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get today's stats
	today := time.Now().Format("2006-01-02")
	startDate, _ := time.Parse("2006-01-02", today)
	endDate := startDate.Add(24 * time.Hour)

	query := &storage.StatsQuery{
		StartDate: startDate,
		EndDate:   endDate,
	}

	stats, err := db.GetDailyStats(query)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	if len(stats) == 0 {
		fmt.Println("  No data for today")
		return nil
	}

	// Calculate total
	var totalSeconds int64
	for _, stat := range stats {
		totalSeconds += stat.TotalSeconds
	}

	fmt.Printf("  Total time: %s\n", formatDuration(totalSeconds))
	fmt.Println()
	fmt.Println("  By application:")

	for _, stat := range stats {
		fmt.Printf("    %s: %s\n", stat.AppName, formatDuration(stat.TotalSeconds))
	}

	return nil
}

func exportData() error {
	// Parse command line arguments
	format := "csv"
	outputFile := "actime_export.csv"
	startDate := ""
	endDate := ""

	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--format":
			if i+1 < len(os.Args) {
				format = os.Args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(os.Args) {
				outputFile = os.Args[i+1]
				i++
			}
		case "--start":
			if i+1 < len(os.Args) {
				startDate = os.Args[i+1]
				i++
			}
		case "--end":
			if i+1 < len(os.Args) {
				endDate = os.Args[i+1]
				i++
			}
		}
	}

	fmt.Printf("Exporting data to %s (format: %s)...\n", outputFile, format)

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Open database
	db, err := storage.NewDB(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Parse date range
	var start, end time.Time
	if startDate != "" {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return fmt.Errorf("invalid start date format: %w", err)
		}
	}
	if endDate != "" {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return fmt.Errorf("invalid end date format: %w", err)
		}
	}

	// Get statistics
	query := &storage.StatsQuery{
		StartDate: start,
		EndDate:   end,
	}

	stats, err := db.GetDailyStats(query)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	// Export based on format
	switch format {
	case "csv":
		if err := exportToCSV(stats, outputFile); err != nil {
			return err
		}
	case "json":
		if err := exportToJSON(stats, outputFile); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	fmt.Printf("Data exported successfully to %s\n", outputFile)
	return nil
}

func exportToCSV(stats []*storage.DailyStats, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Date", "Application", "Total Seconds", "Formatted Duration"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	for _, stat := range stats {
		// Clean up AppName: remove null bytes and extra whitespace
		cleanAppName := strings.ReplaceAll(stat.AppName, "\x00", " ")
		cleanAppName = strings.TrimSpace(cleanAppName)

		if err := writer.Write([]string{
			stat.Date.Format("2006-01-02"),
			cleanAppName,
			fmt.Sprintf("%d", stat.TotalSeconds),
			formatDuration(stat.TotalSeconds),
		}); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

func exportToJSON(stats []*storage.DailyStats, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func showConfig() error {
	fmt.Println("Current Configuration:")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Printf("  Database Path: %s\n", cfg.Database.Path)
	fmt.Printf("  Check Interval: %s\n", cfg.Monitor.CheckInterval)
	fmt.Printf("  Activity Window: %s\n", cfg.Monitor.ActivityWindow)
	fmt.Printf("  Log Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  Log File: %s\n", cfg.Logging.File)
	fmt.Printf("  Export Directory: %s\n", cfg.Export.OutputDir)

	return nil
}

func formatDuration(seconds int64) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		return fmt.Sprintf("%ds", secs)
	}
}