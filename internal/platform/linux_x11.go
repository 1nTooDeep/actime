//go:build linux

package platform

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/screensaver"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xprop"
)

// X11Detector implements Detector for Linux using X11
type X11Detector struct {
	X               *xgb.Conn
	XUtil           *xgbutil.XUtil
	initialized     bool
	display         string
	screenSaverExt  bool
}

// NewX11Detector creates a new X11 detector
func NewX11Detector() *X11Detector {
	return &X11Detector{}
}

// Initialize initializes the X11 connection
func (d *X11Detector) Initialize() error {
	// Get DISPLAY environment variable
	d.display = os.Getenv("DISPLAY")
	if d.display == "" {
		d.display = ":0"
	}

	// Connect to X server
	X, err := xgb.NewConnDisplay(d.display)
	if err != nil {
		return fmt.Errorf("failed to connect to X server: %w", err)
	}
	d.X = X

	// Initialize XUtil for xgbutil functions
	d.XUtil, err = xgbutil.NewConn()
	if err != nil {
		return fmt.Errorf("failed to initialize XUtil: %w", err)
	}

	// Check for ScreenSaver extension
	err = screensaver.Init(d.X)
	if err != nil {
		d.screenSaverExt = false
	} else {
		d.screenSaverExt = true
	}

	d.initialized = true
	return nil
}

// GetActiveWindow returns the active window information
func (d *X11Detector) GetActiveWindow() (*WindowInfo, error) {
	if !d.initialized {
		return nil, fmt.Errorf("detector not initialized")
	}

	// Get the active window using EWMH
	activeWin, err := ewmh.ActiveWindowGet(d.XUtil)
	if err != nil {
		return nil, fmt.Errorf("failed to get active window: %w", err)
	}

	if activeWin == 0 {
		return nil, fmt.Errorf("no active window")
	}

	// Get window name (WM_NAME)
	wmName, err := ewmh.WmNameGet(d.XUtil, activeWin)
	if err != nil {
		// Fallback to classic WM_NAME
		name, err := xprop.PropValStr(xprop.GetProperty(d.XUtil, activeWin, "WM_NAME"))
		if err != nil {
			// Another fallback to _NET_WM_NAME
			name2, err := xprop.PropValStr(xprop.GetProperty(d.XUtil, activeWin, "_NET_WM_NAME"))
			if err != nil {
				wmName = "Unknown"
			} else {
				wmName = name2
			}
		} else {
			wmName = name
		}
	}

	// Get application name (WM_CLASS)
	wmClass, err := xprop.PropValStr(xprop.GetProperty(d.XUtil, activeWin, "WM_CLASS"))
	if err != nil {
		wmClass = "Unknown"
	}

	// Clean up WM_CLASS: it may contain null bytes and multiple parts
	// WM_CLASS format is typically "instance\0class\0" or "instance class"
	wmClass = strings.ReplaceAll(wmClass, "\x00", " ")
	wmClass = strings.TrimSpace(wmClass)
	parts := strings.Fields(wmClass)
	if len(parts) > 0 {
		// Use the first non-empty part
		wmClass = parts[0]
	}

	// Get window PID
	pid, err := ewmh.WmPidGet(d.XUtil, activeWin)
	if err != nil {
		pid = 0
	}

	return &WindowInfo{
		AppName:     wmClass,
		WindowTitle: wmName,
		PID:         int32(pid),
	}, nil
}

// GetIdleTime returns the idle time using XScreenSaver
func (d *X11Detector) GetIdleTime() (time.Duration, error) {
	if !d.initialized {
		return 0, fmt.Errorf("detector not initialized")
	}

	if !d.screenSaverExt {
		// ScreenSaver extension not available, return 0
		return 0, nil
	}

	// Query screen saver info
	screen := xproto.Setup(d.X).DefaultScreen(d.X)
	reply, err := screensaver.QueryInfo(d.X, xproto.Drawable(screen.Root)).Reply()
	if err != nil {
		return 0, fmt.Errorf("failed to query screen saver info: %w", err)
	}

	// Idle time is in milliseconds
	idleTime := time.Duration(reply.MsSinceUserInput) * time.Millisecond
	return idleTime, nil
}

// IsScreenLocked returns true if the screen is locked
func (d *X11Detector) IsScreenLocked() (bool, error) {
	if !d.initialized {
		return false, fmt.Errorf("detector not initialized")
	}

	// Try to detect screen lock by checking if there's a screensaver window active
	// This is a simplified detection method
	activeWin, err := ewmh.ActiveWindowGet(d.XUtil)
	if err != nil {
		return false, fmt.Errorf("failed to get active window: %w", err)
	}

	if activeWin == 0 {
		return false, nil
	}

	// Get window class
	wmClass, err := xprop.PropValStr(xprop.GetProperty(d.XUtil, activeWin, "WM_CLASS"))
	if err != nil {
		return false, nil
	}

	// Check for common screen locker window classes
	screenLockers := []string{
		"xscreensaver",
		"gnome-screensaver",
		"kscreenlocker",
		"lightdm-gtk-greeter",
		"light-locker",
	}

	for _, locker := range screenLockers {
		if wmClass == locker {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the X11 connection
func (d *X11Detector) Close() error {
	if d.XUtil != nil {
		d.XUtil.Conn().Close()
		d.XUtil = nil
	}
	if d.X != nil {
		d.X.Close()
		d.X = nil
	}
	d.initialized = false
	return nil
}