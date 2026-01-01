package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	PIDFile = "/tmp/actime.pid"
)

// WritePIDFile writes the current process ID to the PID file
func WritePIDFile(pidFile string) error {
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPIDFile reads the process ID from the PID file
func ReadPIDFile(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// RemovePIDFile removes the PID file
func RemovePIDFile(pidFile string) error {
	return os.Remove(pidFile)
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}

	return true
}

// CheckAndLockPIDFile checks if the service is already running and locks the PID file
func CheckAndLockPIDFile(pidFile string) error {
	// Check if PID file exists
	if _, err := os.Stat(pidFile); err == nil {
		// PID file exists, read it
		pid, err := ReadPIDFile(pidFile)
		if err != nil {
			return fmt.Errorf("failed to read PID file: %w", err)
		}

		// Check if the process is still running
		if IsProcessRunning(pid) {
			return fmt.Errorf("service is already running (PID: %d)", pid)
		}

		// Process is not running, remove stale PID file
		if err := RemovePIDFile(pidFile); err != nil {
			return fmt.Errorf("failed to remove stale PID file: %w", err)
		}
	}

	// Write new PID file
	if err := WritePIDFile(pidFile); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}