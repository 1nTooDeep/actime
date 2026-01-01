//go:build windows

package platform

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetLastInputInfo    = user32.NewProc("GetLastInputInfo")
	procOpenProcess         = user32.NewProc("OpenProcess")
	procCloseHandle         = user32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = user32.NewProc("QueryFullProcessImageNameW")
	procGetClassNameW       = user32.NewProc("GetClassNameW")

	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procWTSGetSessionConsoleSessionId = kernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQuerySessionInformationW   = kernel32.NewLazySystemDLL("wtsapi32.dll").NewProc("WTSQuerySessionInformationW")
)

// LASTINPUTINFO contains information about the last input event
type LASTINPUTINFO struct {
	CBSize uint32
	DwTime uint32
}

// WindowsDetector implements Detector for Windows
type WindowsDetector struct {
	initialized bool
}

// NewWindowsDetector creates a new Windows detector
func NewWindowsDetector() *WindowsDetector {
	return &WindowsDetector{}
}

// Initialize initializes the Windows detector
func (d *WindowsDetector) Initialize() error {
	d.initialized = true
	return nil
}

// GetActiveWindow returns the active window information
func (d *WindowsDetector) GetActiveWindow() (*WindowInfo, error) {
	if !d.initialized {
		return nil, fmt.Errorf("detector not initialized")
	}

	// Get foreground window handle
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return nil, fmt.Errorf("no foreground window")
	}

	// Get window title
	var titleBuf [512]uint16
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
	windowTitle := syscall.UTF16ToString(titleBuf[:])

	// Get process ID
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	// Get process name
	appName := "Unknown"
	if pid != 0 {
		// Open process
		hProcess, _, err := procOpenProcess.Call(
			windows.PROCESS_QUERY_LIMITED_INFORMATION,
			0,
			uintptr(pid),
		)
		if hProcess != 0 {
			defer procCloseHandle.Call(hProcess)

			// Get full process image name
			var nameBuf [260]uint16
			var size uint32 = uint32(len(nameBuf))
			ret, _, _ := procQueryFullProcessImageNameW.Call(
				hProcess,
				0,
				uintptr(unsafe.Pointer(&nameBuf[0])),
				uintptr(unsafe.Pointer(&size)),
			)
			if ret != 0 {
				fullPath := syscall.UTF16ToString(nameBuf[:size])
				// Extract just the filename
				parts := strings.Split(fullPath, "\\")
				if len(parts) > 0 {
					appName = parts[len(parts)-1]
				}
			}
		}
	}

	// Get window class name
	var classNameBuf [256]uint16
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&classNameBuf[0])), uintptr(len(classNameBuf)))
	className := syscall.UTF16ToString(classNameBuf[:])

	// Use class name as fallback if app name is unknown
	if appName == "Unknown" && className != "" {
		appName = className
	}

	return &WindowInfo{
		AppName:     appName,
		WindowTitle: windowTitle,
		PID:         int32(pid),
	}, nil
}

// GetIdleTime returns the idle time using GetLastInputInfo
func (d *WindowsDetector) GetIdleTime() (time.Duration, error) {
	if !d.initialized {
		return 0, fmt.Errorf("detector not initialized")
	}

	var lii LASTINPUTINFO
	lii.CBSize = uint32(unsafe.Sizeof(lii))

	ret, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0, fmt.Errorf("failed to get last input info")
	}

	// Calculate idle time
	now := uint32(time.Now().Unix() * 1000) // milliseconds
	idleTime := time.Duration(now-lii.DwTime) * time.Millisecond

	return idleTime, nil
}

// IsScreenLocked returns true if the screen is locked
func (d *WindowsDetector) IsScreenLocked() (bool, error) {
	if !d.initialized {
		return false, fmt.Errorf("detector not initialized")
	}

	// Check if the workstation is locked
	// This is done by checking if the foreground window is the lock screen
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return false, nil
	}

	// Get window class name
	var classNameBuf [256]uint16
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&classNameBuf[0])), uintptr(len(classNameBuf)))
	className := syscall.UTF16ToString(classNameBuf[:])

	// Check for common lock screen window classes
	lockScreenClasses := []string{
		"LockScreen",
		"Windows.UI.Core.CoreWindow",
		"ApplicationFrameWindow",
	}

	for _, lockClass := range lockScreenClasses {
		if strings.Contains(className, lockClass) {
			return true, nil
		}
	}

	return false, nil
}

// Close cleans up Windows resources
func (d *WindowsDetector) Close() error {
	d.initialized = false
	return nil
}