package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/weii/actime/internal/config"
	"github.com/weii/actime/internal/core"
	"github.com/weii/actime/internal/storage"
)

const (
	Version = "0.1.0"
)

// Global configuration
var appConfig *core.Config

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
	case "visualize":
		if err := visualizeData(); err != nil {
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
	fmt.Println("  stats     Show usage statistics")
	fmt.Println("  export    Export data to CSV or JSON")
	fmt.Println("  visualize Generate HTML visualization report")
	fmt.Println("  config    Show configuration")
	fmt.Println("  version   Show version information")
	fmt.Println("  help      Show this help message")
	fmt.Println()
	fmt.Println("Stats Options:")
	fmt.Println("  --days <n>       Number of days to show (default: 1 for today)")
	fmt.Println("  --start <date>   Start date (format: YYYY-MM-DD)")
	fmt.Println("  --end <date>     End date (format: YYYY-MM-DD)")
	fmt.Println("  --top <n>        Show top N applications only")
	fmt.Println()
	fmt.Println("Visualize Options:")
	fmt.Println("  --output <file>  Output HTML file (default: actime_visualization.html)")
	fmt.Println("  --days <n>       Number of days to visualize (default: 7)")
	fmt.Println("  --start <date>   Start date (format: YYYY-MM-DD)")
	fmt.Println("  --end <date>     End date (format: YYYY-MM-DD)")
	fmt.Println("  --open           Open report in browser after generation")
	fmt.Println()
	fmt.Println("Export Options:")
	fmt.Println("  --format <fmt>   Output format: csv or json (default: csv)")
	fmt.Println("  --output <file>  Output file (default: actime_export.csv/.json)")
	fmt.Println("  --start <date>   Start date (format: YYYY-MM-DD)")
	fmt.Println("  --end <date>     End date (format: YYYY-MM-DD)")
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

func visualizeData() error {
	// Parse command line arguments
	outputFile := "actime_visualization.html"
	startDate := ""
	endDate := ""
	days := 7 // Default: last 7 days
	openBrowser := false

	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
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
		case "--days":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &days)
				i++
			}
		case "--open":
			openBrowser = true
		}
	}

	fmt.Printf("Generating visualization report to %s...\n", outputFile)

	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	appConfig = cfg

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
	} else {
		// Default to last N days
		end = time.Now().Truncate(24 * time.Hour)
		start = end.AddDate(0, 0, -days)
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

	// Get sessions for heatmap
	sessions, err := db.GetSessions(start, end)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	if len(stats) == 0 && len(sessions) == 0 {
		fmt.Println("No data found for the specified date range")
		return nil
	}

	// Generate HTML report
	if err := generateHTMLReport(stats, sessions, outputFile, start, end); err != nil {
		return err
	}

	fmt.Printf("Visualization report generated successfully: %s\n", outputFile)

	// Open in browser if requested
	if openBrowser {
		if err := openFile(outputFile); err != nil {
			fmt.Printf("Warning: failed to open browser: %v\n", err)
		}
	}

	return nil
}

func generateHTMLReport(stats []*storage.DailyStats, sessions []*storage.Session, outputFile string, start, end time.Time) error {
	// Validate input data
	if len(stats) == 0 && len(sessions) == 0 {
		return fmt.Errorf("no data provided")
	}
	if start.After(end) {
		return fmt.Errorf("start date must be before or equal to end date")
	}

	// Generate charts
	barChart := createBarChart(stats)
	pieChart := createPieChart(stats)
	lineChart := createLineChart(stats, start, end)
	heatMap := createHeatMap(sessions, start, end)
	treeMap := createTreeMap(stats)

	// Format date range
	dateRange := fmt.Sprintf("%s 至 %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	totalDays := int(end.Sub(start).Hours()/24) + 1

	// Build HTML content with structured layout
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Actime 使用时间分析报告</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            padding: 20px;
            margin: 0;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .report-header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 40px 60px;
            text-align: center;
        }
        .report-header h1 {
            margin: 0 0 10px 0;
            font-size: 32px;
            font-weight: 700;
        }
        .report-header .subtitle {
            font-size: 16px;
            opacity: 0.9;
            margin: 0;
        }
        .report-header .date-range {
            margin-top: 15px;
            font-size: 14px;
            opacity: 0.8;
            background: rgba(255,255,255,0.2);
            display: inline-block;
            padding: 8px 20px;
            border-radius: 20px;
        }
        .section {
            padding: 40px 60px;
            border-bottom: 1px solid #f0f0f0;
        }
        .section:last-child {
            border-bottom: none;
        }
        .section-header {
            margin-bottom: 30px;
        }
        .section-title {
            font-size: 24px;
            font-weight: 600;
            color: #2c3e50;
            margin: 0 0 8px 0;
        }
        .section-description {
            font-size: 14px;
            color: #7f8c8d;
            margin: 0;
            line-height: 1.6;
        }
        .charts-row {
            display: flex;
            gap: 30px;
            margin-bottom: 30px;
        }
        .charts-row:last-child {
            margin-bottom: 0;
        }
        .charts-row .item {
            flex: 1;
            min-height: 400px;
        }
        .charts-row.single-chart .item {
            flex: 1;
        }
        .charts-row.two-charts .item {
            flex: 1;
        }
        .charts-row.two-charts .item:first-child {
            flex: 1.2;
        }
        .charts-row.two-charts .item:last-child {
            flex: 0.8;
        }
        @media (max-width: 1200px) {
            .charts-row.two-charts {
                flex-direction: column;
            }
            .charts-row.two-charts .item {
                flex: 1;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="report-header">
            <h1>Actime 使用时间分析报告</h1>
            <p class="subtitle">了解您的应用使用习惯，优化时间分配</p>
            <div class="date-range">
                📅 统计周期：%s（共 %d 天）
            </div>
        </div>
        
        <div class="section">
            <div class="section-header">
                <h2 class="section-title">📊 总览：应用使用排行</h2>
                <p class="section-description">
                    查看您使用时间最长的应用排行，了解主要时间分配。左侧柱状图展示 Top 10 应用的具体时长，
                    右侧饼图展示各应用的时间占比，帮助您快速识别重点应用。
                </p>
            </div>
            <div class="charts-row two-charts">
                <div class="item" id="bar-chart"></div>
                <div class="item" id="pie-chart"></div>
            </div>
        </div>
        
        <div class="section">
            <div class="section-header">
                <h2 class="section-title">📈 趋势：每日使用变化</h2>
                <p class="section-description">
                    观察应用使用时间随日期的变化趋势，识别使用模式和异常波动。
                    折线图展示各应用在统计周期内的每日使用时长，帮助您了解使用习惯的稳定性。
                </p>
            </div>
            <div class="charts-row single-chart">
                <div class="item" id="line-chart"></div>
            </div>
        </div>
        
        <div class="section">
            <div class="section-header">
                <h2 class="section-title">⏰ 节律：时间段分布</h2>
                <p class="section-description">
                    了解您在不同时间段的应用使用密度，识别活跃时段和休息时间。
                    热力图展示每天 24 小时的使用时长分布，颜色越深表示使用时间越长。
                </p>
            </div>
            <div class="charts-row single-chart">
                <div class="item" id="heatmap-chart"></div>
            </div>
        </div>
        
        <div class="section">
            <div class="section-header">
                <h2 class="section-title">🧩 构成：应用时间结构</h2>
                <p class="section-description">
                    以矩形面积展示各应用的时间占比，直观比较不同应用的相对重要性。
                    矩形面积越大表示使用时间越长，帮助您快速识别时间分配的结构特征。
                </p>
            </div>
            <div class="charts-row single-chart">
                <div class="item" id="treemap-chart"></div>
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
    <script>
        // Render Bar Chart
        var barChart = echarts.init(document.getElementById('bar-chart'), "macarons");
        barChart.setOption(%s);
        
        // Render Pie Chart
        var pieChart = echarts.init(document.getElementById('pie-chart'), "macarons");
        pieChart.setOption(%s);
        
        // Render Line Chart
        var lineChart = echarts.init(document.getElementById('line-chart'), "macarons");
        lineChart.setOption(%s);
        
        // Render HeatMap
        var heatMapChart = echarts.init(document.getElementById('heatmap-chart'), "macarons");
        heatMapChart.setOption(%s);
        
        // Render TreeMap
        var treeMapChart = echarts.init(document.getElementById('treemap-chart'), "macarons");
        treeMapChart.setOption(%s);
        
        // Responsive resize
        window.addEventListener('resize', function() {
            barChart.resize();
            pieChart.resize();
            lineChart.resize();
            heatMapChart.resize();
            treeMapChart.resize();
        });
    </script>
</body>
</html>
`, dateRange, totalDays,
		getChartJSON(barChart),
		getChartJSON(pieChart),
		getChartJSON(lineChart),
		getChartJSON(heatMap),
		getChartJSON(treeMap))

	// Write to file
	if err := os.WriteFile(outputFile, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func createBarChart(stats []*storage.DailyStats) *charts.Bar {
	// Aggregate data by application
	appMinutes := make(map[string]int)

	for _, stat := range stats {
		appName := cleanAppName(stat.AppName)
		minutes := int(math.Ceil(float64(stat.TotalSeconds) / 60))
		appMinutes[appName] += minutes
	}

	// Sort by value
	type appStat struct {
		name    string
		minutes int
	}

	var sorted []appStat
	for name, m := range appMinutes {
		sorted = append(sorted, appStat{name: name, minutes: m})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].minutes > sorted[j].minutes
	})

	// Top 10
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}

	// Prepare data
	var appNames []string
	var values []opts.BarData

	for _, s := range sorted {
		appNames = append(appNames, s.name)
		values = append(values, opts.BarData{
			Value: s.minutes,
		})
	}

	// Create bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "macarons",
			Width:  "1200px",
			Height: "500px",
		}),
		charts.WithTitleOpts(opts.Title{
			Title: "应用使用时长排行",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:            opts.Bool(true),
			Trigger:         "axis",
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#CCCCCC",
			Formatter: opts.FuncOpts(`
				function (params) {
					var p = params[0];
					var m = p.value;
					var h = Math.floor(m / 60);
					var mm = m % 60;
					if (h > 0) {
						return p.name + ': ' + h + 'h ' + mm + 'm';
					}
					return p.name + ': ' + m + 'm';
				}
			`),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "应用",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "使用时长（分钟）",
			AxisLabel: &opts.AxisLabel{
				Formatter: opts.FuncOpts(`
					function (v) {
						if (v >= 60) {
							return Math.floor(v / 60) + 'h';
						}
						return v + 'm';
					}
				`),
			},
		}),
	)

	bar.SetXAxis(appNames).
		AddSeries("使用时长", values).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show: opts.Bool(false),
			}),
		)

	return bar
}

func createPieChart(stats []*storage.DailyStats) *charts.Pie {
	// 1. 聚合应用使用时长
	appStats := make(map[string]int64)
	var totalSeconds int64

	for _, stat := range stats {
		appName := cleanAppName(stat.AppName)
		appStats[appName] += stat.TotalSeconds
		totalSeconds += stat.TotalSeconds
	}

	// 2. 构造 PieData
	var items []opts.PieData
	for name, t := range appStats {
		items = append(items, opts.PieData{
			Name:  name,
			Value: t,
		})
	}

	// 3. 按使用时长排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value.(int64) > items[j].Value.(int64)
	})

	// 4. 合并占比 < 3% 的应用为「其他」
	var finalItems []opts.PieData
	var others int64

	for _, item := range items {
		v := item.Value.(int64)
		if float64(v)/float64(totalSeconds) < 0.03 {
			others += v
		} else {
			finalItems = append(finalItems, item)
		}
	}

	if others > 0 {
		finalItems = append(finalItems, opts.PieData{
			Name:  "其他",
			Value: others,
		})
	}

	// 5. 创建 Pie 图
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "应用使用时长分布",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "item",
			Formatter: opts.FuncOpts(`function(params) {
				var s = params.value;
				var h = Math.floor(s / 3600);
				var m = Math.floor((s % 3600) / 60);
				var t = h > 0 ? h + '小时 ' + m + '分钟' : m + '分钟';
				return params.name + '<br/>' + t + '<br/>' + params.percent + '%';
			}`),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "macarons",
			Width:  "1200px",
			Height: "600px",
		}),
	)

	// 6. 关键修复：Pie 的空间与 Label 策略
	pie.AddSeries("使用时长", finalItems).
		SetSeriesOptions(
			charts.WithPieChartOpts(opts.PieChart{
				Radius: []string{"38%", "68%"},
				Center: []string{"55%", "50%"},
			}),
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}\n{d}%",
				FontSize:  11,
			}),
			charts.WithLabelLineOpts(opts.LabelLine{
				Show:    opts.Bool(true),
				Length2: 10,
				Smooth:  opts.Bool(false),
			}),
		)

	return pie
}

func createLineChart(stats []*storage.DailyStats, start, end time.Time) *charts.Line {
	// Aggregate data by date
	dateStats := make(map[string]map[string]int64)
	for _, stat := range stats {
		date := stat.Date.Format("2006-01-02")
		if dateStats[date] == nil {
			dateStats[date] = make(map[string]int64)
		}
		appName := cleanAppName(stat.AppName)
		dateStats[date][appName] += stat.TotalSeconds
	}

	// Get all dates in range (including end date)
	var dates []string
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	// Get top 5 apps
	appTotals := make(map[string]int64)
	for _, apps := range dateStats {
		for app, time := range apps {
			appTotals[app] += time
		}
	}

	type appTotal struct {
		name string
		time int64
	}
	var sortedApps []appTotal
	for name, time := range appTotals {
		sortedApps = append(sortedApps, appTotal{name, time})
	}
	sort.Slice(sortedApps, func(i, j int) bool {
		return sortedApps[i].time > sortedApps[j].time
	})

	if len(sortedApps) > 5 {
		sortedApps = sortedApps[:5]
	}

	// Create line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "每日使用时长趋势",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:            opts.Bool(true),
			Trigger:         "axis",
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#CCCCCC",
			Formatter: opts.FuncOpts(`function(params) {
				var result = params[0].name + '<br/>';
				params.forEach(function(item) {
					var seconds = item.value;
					var hours = Math.floor(seconds / 3600);
					var minutes = Math.floor((seconds % 3600) / 60);
					var timeStr;
					if (hours > 0) {
						timeStr = hours + 'h ' + minutes + 'm';
					} else {
						timeStr = minutes + 'm';
					}
					result += item.seriesName + ': ' + timeStr + '<br/>';
				});
				return result;
			}`),
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "日期",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "时长",
			AxisLabel: &opts.AxisLabel{
				Formatter: opts.FuncOpts(`function(value) {
					var hours = Math.floor(value / 3600);
					var minutes = Math.floor((value % 3600) / 60);
					if (hours > 0) {
						return hours + 'h';
					} else {
						return minutes + 'm';
					}
				}`),
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "macarons",
			Width:  "1200px",
			Height: "600px",
		}),
	)

	// Set X axis and add all series
	line.SetXAxis(dates)
	for _, app := range sortedApps {
		var values []opts.LineData
		for _, date := range dates {
			if time, exists := dateStats[date][app.name]; exists {
				values = append(values, opts.LineData{Value: time})
			} else {
				values = append(values, opts.LineData{Value: 0})
			}
		}
		line.AddSeries(app.name, values).
			SetSeriesOptions(
				charts.WithLabelOpts(opts.Label{
					Show: opts.Bool(false),
				}),
			)
	}

	return line
}

func createHeatMap(sessions []*storage.Session, start, end time.Time) *charts.HeatMap {
	// Create a map to store hourly usage: date -> hour -> seconds
	hourlyUsage := make(map[string]map[int]int64)

	// Initialize all dates and hours
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		hourlyUsage[dateStr] = make(map[int]int64)
		for hour := 0; hour < 24; hour++ {
			hourlyUsage[dateStr][hour] = 0
		}
	}

	// Aggregate sessions by hour
	for _, session := range sessions {
		dateStr := session.StartTime.Format("2006-01-02")
		hour := session.StartTime.Hour()

		// Skip if date is outside our range
		if _, exists := hourlyUsage[dateStr]; !exists {
			continue
		}

		// Add duration to the hour
		hourlyUsage[dateStr][hour] += session.DurationSeconds
	}

	// Prepare data for heatmap
	var items []opts.HeatMapData
	dates := make([]string, 0)

	// Get sorted dates
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		dates = append(dates, dateStr)
	}

	// Create heatmap data
	for _, dateStr := range dates {
		for hour := 0; hour < 24; hour++ {
			seconds := hourlyUsage[dateStr][hour]
			items = append(items, opts.HeatMapData{
				Value: []interface{}{hour, dateStr, seconds},
			})
		}
	}

	// Create heatmap
	heatMap := charts.NewHeatMap()
	heatMap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "每日使用时间分布（按小时）",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:            opts.Bool(true),
			Trigger:         "item",
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#CCCCCC",
			Formatter: opts.FuncOpts(`function(params) {
				var hour = params.value[0];
				var date = params.value[1];
				var seconds = params.value[2];
				var hours = Math.floor(seconds / 3600);
				var minutes = Math.floor((seconds % 3600) / 60);
				var timeStr;
				if (hours > 0) {
					timeStr = hours + '小时 ' + minutes + '分钟';
				} else if (minutes > 0) {
					timeStr = minutes + '分钟';
				} else {
					timeStr = '0分钟';
				}
				return date + ' ' + hour + ':00<br/>' + timeStr;
			}`),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "时间（小时）",
			Type: "category",
			Data: func() []string {
				var hours []string
				for i := 0; i < 24; i++ {
					hours = append(hours, fmt.Sprintf("%d:00", i))
				}
				return hours
			}(),
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "日期",
			Type: "category",
			Data: dates,
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(true),
			Min:        0,
			Max: func() float32 {
				maxSeconds := int64(0)
				for _, item := range items {
					if val, ok := item.Value.([]interface{}); ok && len(val) >= 3 {
						if seconds, ok := val[2].(int64); ok && seconds > maxSeconds {
							maxSeconds = seconds
						}
					}
				}
				return float32(maxSeconds)
			}(),
			InRange: &opts.VisualMapInRange{
				Color: []string{"#50a3ba", "#eac736", "#d94e5d"},
			},
			Text: []string{"高", "低"},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "macarons",
			Width:  "1200px",
			Height: "600px",
		}),
	)

	heatMap.AddSeries("使用时长", items)

	return heatMap
}

func createTreeMap(stats []*storage.DailyStats) *charts.TreeMap {
	// Aggregate data by application
	appStats := make(map[string]int64)
	for _, stat := range stats {
		appName := cleanAppName(stat.AppName)
		appStats[appName] += stat.TotalSeconds
	}

	// Prepare data
	var items []opts.TreeMapNode
	for name, time := range appStats {
		items = append(items, opts.TreeMapNode{
			Name:  name,
			Value: int(time),
		})
	}

	// Sort by value
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})

	// Create treemap
	treeMap := charts.NewTreeMap()
	treeMap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "应用使用时长分布（树状图）",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:            opts.Bool(true),
			Trigger:         "item",
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#CCCCCC",
			Formatter:       "{b}<br/>{c}秒",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  "macarons",
			Width:  "1200px",
			Height: "600px",
		}),
	)

	treeMap.AddSeries("使用时长", items).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Position:  "inside",
				Formatter: "{b}\n{c}s",
			}),
		)

	return treeMap
}

func getChartJSON(chart interface{}) string {
	// Create a temporary buffer to render the chart
	var buf strings.Builder

	// Create a minimal page with just this chart
	page := components.NewPage()

	// Type assertion to Charter
	if charter, ok := chart.(components.Charter); ok {
		page.AddCharts(charter)
	} else {
		return "{}"
	}

	// Render to buffer
	if err := page.Render(&buf); err != nil {
		return "{}"
	}

	// Extract the JSON option from the rendered HTML
	html := buf.String()

	// Find the JSON option
	startIdx := strings.Index(html, "let option_")
	if startIdx == -1 {
		return "{}"
	}

	// Find the = sign
	eqIdx := strings.Index(html[startIdx:], "=")
	if eqIdx == -1 {
		return "{}"
	}

	// Find the opening {
	openBraceIdx := strings.Index(html[startIdx+eqIdx:], "{")
	if openBraceIdx == -1 {
		return "{}"
	}

	// Find the matching closing }
	braceCount := 0
	startPos := startIdx + eqIdx + openBraceIdx
	var endPos int

	for i := startPos; i < len(html); i++ {
		if html[i] == '{' {
			braceCount++
		} else if html[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endPos = i + 1
				break
			}
		}
	}

	if endPos == 0 {
		return "{}"
	}

	return html[startPos:endPos]
}

func cleanAppName(appName string) string {
	// Remove null bytes and extra whitespace
	cleanName := strings.ReplaceAll(appName, "\x00", " ")
	cleanName = strings.TrimSpace(cleanName)

	// Apply process name mapping if config is available
	if appConfig != nil && appConfig.AppMapping.ProcessNames != nil {
		// Try to find a match (case-insensitive)
		lowerName := strings.ToLower(cleanName)
		if mappedName, exists := appConfig.AppMapping.ProcessNames[lowerName]; exists {
			return mappedName
		}
	}

	return cleanName
}

func openFile(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // linux and others
		cmd = "xdg-open"
	}
	args = append(args, path)
	return exec.Command(cmd, args...).Start()
}
