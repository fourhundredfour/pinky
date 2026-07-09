//go:build windows

// Package explorer hides and restores Windows' own taskbar (Explorer's
// Shell_TrayWnd/Shell_SecondaryTrayWnd windows) while pinky stands in for
// it, and hands back the reclaimed screen edge via the normal DWM work-area
// recalculation that follows a taskbar window becoming invisible.
package explorer

import (
	"sync"

	"github.com/fourhundredfour/pinky/internal/win32"
)

const (
	primaryClassName   = "Shell_TrayWnd"
	secondaryClassName = "Shell_SecondaryTrayWnd"
)

// Controller tracks which real taskbar windows pinky has hidden so it can
// restore them later, even if new secondary taskbars appear or disappear
// (monitors being connected/disconnected) while pinky is running.
type Controller struct {
	mu     sync.Mutex
	active bool
	known  map[win32.HWND]bool
}

// New creates a controller that has not hidden anything yet.
func New() *Controller {
	return &Controller{known: make(map[win32.HWND]bool)}
}

// Hide locates every currently open real taskbar window (primary plus any
// per-monitor secondary bars) and hides it. Safe to call repeatedly, e.g.
// from a periodic watchdog, to re-hide bars Explorer decided to re-show on
// its own (which it occasionally does after certain shell notifications).
func (c *Controller) Hide() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = true
	for _, hwnd := range findAll() {
		if !win32.IsWindow(hwnd) {
			continue
		}
		c.known[hwnd] = true
		if win32.IsWindowVisible(hwnd) {
			win32.ShowWindow(hwnd, win32.SWHide)
		}
	}
}

// Show restores every taskbar window this controller previously hid, then
// stops tracking them. Intended for use on shutdown, or when the user
// disables hide_real_taskbar in the config.
func (c *Controller) Show() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = false
	for hwnd := range c.known {
		if win32.IsWindow(hwnd) {
			win32.ShowWindow(hwnd, win32.SWShow)
		}
	}
	c.known = make(map[win32.HWND]bool)
}

// Reassert re-applies Hide() if the controller is active. Call this on a
// short interval (e.g. every couple of seconds) and whenever
// appbar.Callbacks.OnTaskbarRecreated fires, since Explorer restarting
// creates a brand-new, visible Shell_TrayWnd that needs to be re-hidden.
func (c *Controller) Reassert() {
	c.mu.Lock()
	active := c.active
	c.mu.Unlock()
	if active {
		c.Hide()
	}
}

// ForceShow unconditionally restores every real taskbar window (primary plus
// secondary bars) by making them visible, independent of any Controller
// instance state.
func ForceShow() {
	for _, hwnd := range findAll() {
		if win32.IsWindow(hwnd) {
			win32.ShowWindow(hwnd, win32.SWShow)
		}
	}
}

// findAll enumerates the primary taskbar plus every currently open
// secondary (per-monitor) taskbar window.
func findAll() []win32.HWND {
	var out []win32.HWND
	if hwnd := win32.FindWindowW(primaryClassName, ""); hwnd != 0 {
		out = append(out, hwnd)
	}
	var after win32.HWND
	for {
		hwnd := win32.FindWindowExW(0, after, secondaryClassName, "")
		if hwnd == 0 {
			break
		}
		out = append(out, hwnd)
		after = hwnd
	}
	return out
}
