package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/weii/actime/internal/config"
	"github.com/weii/actime/internal/service"
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

	// Check for -h flag
	if hasHelpFlag(os.Args) {
		printCommandHelp(command)
		os.Exit(0)
	}

	switch command {
	case "start":
		if err := startService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Actime daemon started successfully")
	case "daemon":
		// This is the actual daemon process, runs in background
		if err := runDaemon(); err != nil {
			fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	case "stop":
		if err := stopService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Actime daemon stopped successfully")
	case "restart":
		if err := restartService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Actime daemon restarted successfully")
	case "status":
		if err := statusService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "log":
		follow := false
		if len(os.Args) > 2 && os.Args[2] == "-f" {
			follow = true
		}
		if err := showLog(follow); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("Actime Daemon v%s\n", Version)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func printCommandHelp(command string) {
	switch command {
	case "start":
		fmt.Println("Start the Actime daemon")
		fmt.Println()
		fmt.Println("Usage: actimed start")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Starts the Actime daemon in the background. The daemon will")
		fmt.Println("  track application usage time automatically.")
		fmt.Println()
		fmt.Println("Exit codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Failed to start (service already running or error)")
	case "stop":
		fmt.Println("Stop the Actime daemon")
		fmt.Println()
		fmt.Println("Usage: actimed stop")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Stops the running Actime daemon gracefully. All pending data")
		fmt.Println("  will be saved before shutdown.")
		fmt.Println()
		fmt.Println("Exit codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Failed to stop (service not running or error)")
	case "restart":
		fmt.Println("Restart the Actime daemon")
		fmt.Println()
		fmt.Println("Usage: actimed restart")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Restarts the Actime daemon by stopping it and then starting")
		fmt.Println("  it again. All pending data will be saved before restart.")
		fmt.Println()
		fmt.Println("Exit codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Failed to restart")
	case "status":
		fmt.Println("Show the status of the Actime daemon")
		fmt.Println()
		fmt.Println("Usage: actimed status")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Displays the current status of the Actime daemon, including")
		fmt.Println("  whether it is running and its process ID.")
		fmt.Println()
		fmt.Println("Exit codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Failed to get status")
	case "log":
		fmt.Println("Show the recent log entries")
		fmt.Println()
		fmt.Println("Usage: actimed log [-f]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -f  Follow log output (like tail -f)")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Displays the last 50 log entries from the Actime log file.")
		fmt.Println("  With -f flag, it will continuously display new log entries.")
		fmt.Println()
		fmt.Println("Exit codes:")
		fmt.Println("  0 - Success")
		fmt.Println("  1 - Failed to read log file")
	case "version":
		fmt.Println("Show version information")
		fmt.Println()
		fmt.Println("Usage: actimed version")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Displays the current version of Actime daemon.")
	case "daemon":
		fmt.Println("Run Actime as daemon (internal command)")
		fmt.Println()
		fmt.Println("Usage: actimed daemon")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  This is an internal command used by the 'start' command.")
		fmt.Println("  Users should not call this directly.")
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Printf("Actime Daemon v%s\n\n", Version)
	fmt.Println("Usage: actimed <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start    Start the Actime daemon")
	fmt.Println("  stop     Stop the Actime daemon")
	fmt.Println("  restart  Restart the Actime daemon")
	fmt.Println("  status   Show the status of the Actime daemon")
	fmt.Println("  log [-f] Show the recent log entries [-f: follow log output]")
	fmt.Println("  version  Show version information")
	fmt.Println("  help     Show this help message")
}

func startService() error {
	// Check if service is already running
	if isRunning() {
		return fmt.Errorf("service is already running")
	}

	fmt.Println("Starting Actime daemon...")

	// Load configuration first (just to validate it)
	_, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// On Windows, we need to use a different approach to create a detached process
	// We'll use STARTUPINFO to hide the console window
	args := []string{"daemon"}
	cmd := exec.Command(os.Args[0], args...)

	// Hide the console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	// Redirect output to avoid blocking
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the daemon process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Verify it's still running
	if cmd.ProcessState == nil {
		// Process is still running (ProcessState is nil until the process exits)
		fmt.Printf("Daemon started with PID: %d\n", cmd.Process.Pid)
		return nil
	}

	// Process has exited
	return fmt.Errorf("daemon process exited immediately")
}

func stopService() error {
	fmt.Println("Stopping Actime daemon...")

	// Check if PID file exists
	if _, err := os.Stat(service.PIDFile); os.IsNotExist(err) {
		return fmt.Errorf("service is not running")
	}

	// Read PID file
	pid, err := service.ReadPIDFile(service.PIDFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Check if process is running
	if !service.IsProcessRunning(pid) {
		// Process is not running, remove stale PID file
		service.RemovePIDFile(service.PIDFile)
		return fmt.Errorf("service is not running")
	}

	// Kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	return nil
}

func restartService() error {
	fmt.Println("Restarting Actime daemon...")

	// Stop the service if it's running
	if isRunning() {
		if err := stopService(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		// Wait a bit for the service to stop
		time.Sleep(1 * time.Second)
	}

	// Start the service
	if err := startService(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

func statusService() error {
	fmt.Println("Actime daemon status:")

	// Check if service is running
	if isRunning() {
		pid, err := service.ReadPIDFile(service.PIDFile)
		if err != nil {
			fmt.Println("  Status: Running (PID: unknown)")
			return nil
		}
		fmt.Printf("  Status: Running (PID: %d)\n", pid)

		// Get process information
		if err := printProcessInfo(pid); err != nil {
			fmt.Printf("  Process info: Unable to retrieve (%v)\n", err)
		}
	} else {
		fmt.Println("  Status: Stopped")
	}

	return nil
}

func printProcessInfo(pid int) error {
	// Use platform-specific method to get process info
	if runtime.GOOS == "windows" {
		return printProcessInfoWindows(pid)
	}
	return printProcessInfoUnix(pid)
}

func printProcessInfoWindows(pid int) error {
	// Use tasklist to get process information on Windows
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get process info: %w", err)
	}

	// Parse CSV output: "Image Name","PID","Session Name","Session#","Mem Usage","Status","User Name","CPU Time","Window Title"
	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("no process info found")
	}

	// Parse the CSV line
	fields := strings.Split(lines[0], "\",\"")
	if len(fields) < 9 {
		return fmt.Errorf("invalid process info format")
	}

	// Extract fields
	imageName := strings.Trim(fields[0], "\"")
	memUsage := strings.Trim(fields[4], "\"")
	cpuTime := strings.Trim(fields[7], "\"")

	// Print process info
	fmt.Printf("  Process: %s\n", imageName)
	fmt.Printf("  Memory: %s\n", memUsage)
	fmt.Printf("  CPU Time: %s\n", cpuTime)

	return nil
}

func printProcessInfoUnix(pid int) error {
	// Read /proc/[pid]/stat for process information
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := os.ReadFile(statPath)
	if err != nil {
		return fmt.Errorf("failed to read stat file: %w", err)
	}

	// Parse stat file
	// Format: pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime priority nice num_threads itrealvalue starttime vsize rss rsslim startcode endcode startstack kstkesp kstkeip signal blocked sigignore sigcatch wchan nswap cnswap exit_signal processor rt_priority policy delayacct_blkio_ticks guest_time cguest_time
	fields := strings.Fields(string(statData))
	if len(fields) < 24 {
		return fmt.Errorf("invalid stat file format")
	}

	// Extract relevant fields
	// Field 22: vsize (virtual memory size in bytes)
	vsize, _ := strconv.ParseInt(fields[22], 10, 64)
	// Field 23: rss (resident set size in pages)
	rss, _ := strconv.ParseInt(fields[23], 10, 64)
	// Field 13: utime (user mode time in clock ticks)
	utime, _ := strconv.ParseInt(fields[13], 10, 64)
	// Field 14: stime (kernel mode time in clock ticks)
	stime, _ := strconv.ParseInt(fields[14], 10, 64)
	// Field 21: starttime (process start time in clock ticks)
	starttime, _ := strconv.ParseInt(fields[21], 10, 64)

	// Get system clock ticks per second
	clockTicks := int64(100) // Default on most systems

	// Calculate memory usage
	rssBytes := rss * 4096 // Page size is typically 4096 bytes
	vsizeMB := float64(vsize) / 1024 / 1024
	rssMB := float64(rssBytes) / 1024 / 1024

	// Calculate CPU time
	totalTime := (utime + stime) / clockTicks // seconds
	uptime := getSystemUptime()
	if uptime > 0 {
		elapsed := uptime - (float64(starttime) / float64(clockTicks)) // uptime - process start time
		if elapsed > 0 {
			cpuPercent := float64(totalTime) / elapsed * 100
			fmt.Printf("  CPU: %.2f%%\n", cpuPercent)
		} else {
			fmt.Printf("  CPU Time: %.2fs\n", float64(totalTime))
		}
	} else {
		fmt.Printf("  CPU Time: %.2fs\n", float64(totalTime))
	}

	// Print memory info
	fmt.Printf("  Memory: %.2f MB RSS, %.2f MB VIRT\n", rssMB, vsizeMB)

	// Get uptime
	uptimeSeconds := float64(starttime) / float64(clockTicks)
	fmt.Printf("  Uptime: %s\n", fmtDuration(int(uptimeSeconds)))

	return nil
}

func getSystemUptime() float64 {
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(uptimeData))
	if len(fields) < 1 {
		return 0
	}
	uptime, _ := strconv.ParseFloat(fields[0], 64)
	return uptime
}

func fmtDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, seconds%60)
	}
	hours := minutes / 60
	minutes = minutes % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

func showLog(follow bool) error {
	// Load configuration to get log file path
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Open log file
	file, err := os.Open(cfg.Logging.File)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read last 50 lines
	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Show last 50 lines
	start := 0
	if len(lines) > 50 {
		start = len(lines) - 50
	}

	fmt.Printf("Last %d log entries from %s:\n", len(lines)-start, cfg.Logging.File)
	fmt.Println(strings.Repeat("-", 80))
	for i := start; i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	fmt.Println(strings.Repeat("-", 80))

	// If follow mode, tail the file
	if follow {
		fmt.Println("Following log output (press Ctrl+C to stop)...")
		fmt.Println(strings.Repeat("-", 80))

		// Seek to end of file
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}
		_, err = file.Seek(fileInfo.Size(), 0)
		if err != nil {
			return fmt.Errorf("failed to seek file: %w", err)
		}

		// Read new lines as they are written
		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				// Wait a bit and try again
				time.Sleep(100 * time.Millisecond)
				continue
			}
			fmt.Print(line)
		}
	}

	return nil
}

func isRunning() bool {
	// Check if PID file exists
	if _, err := os.Stat(service.PIDFile); err != nil {
		return false
	}

	// Read PID
	pid, err := service.ReadPIDFile(service.PIDFile)
	if err != nil {
		return false
	}

	// Check if process is running
	return service.IsProcessRunning(pid)
}

func runDaemon() error {
	// Load configuration
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create service
	svc, err := service.NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Start service (this will block)
	return svc.Start()
}
