// Package taskbar locates the Windows taskbar window(s) and computes the
// screen rectangle(s) that should be colorized.
package taskbar

import (
	"errors"

	"github.com/fourhundredfour/pinky/internal/win32"
)

// PrimaryClassName is the window class of the primary taskbar on both
// Windows 10 and Windows 11 (the Win11 taskbar is rendered with
// XAML/DirectComposition internally, but the top-level host window is still
// Shell_TrayWnd). There is always exactly one of these.
const PrimaryClassName = "Shell_TrayWnd"

// SecondaryClassName is the window class used for the taskbar Windows shows
// on each additional monitor (when "Show taskbar on all displays" is
// enabled). There can be zero or more of these, one per extra monitor.
const SecondaryClassName = "Shell_SecondaryTrayWnd"

// trayNotifyClassName is the child window hosting the system tray icons and
// clock, docked at the end of the taskbar regardless of whether the running
// app icons are left-aligned or centered (Windows 11 default). Secondary
// taskbars do not have one.
const trayNotifyClassName = "TrayNotifyWnd"

// ErrNotFound is returned when a taskbar window cannot be located, e.g.
// because Explorer is restarting or a secondary monitor was disconnected.
var ErrNotFound = errors.New("taskbar: window not found")

// Bar tracks one taskbar window's handle and geometry, refreshed once per
// render tick since the taskbar can move, resize, or auto-hide.
type Bar struct {
	className string
	hwnd      win32.HWND
	rect      win32.RECT
	trayRect  win32.RECT
	hasTray   bool
}

// Find locates the primary taskbar window. It does not read geometry yet;
// call Refresh to populate it.
func Find() (*Bar, error) {
	hwnd := win32.FindWindowW(PrimaryClassName, "")
	if hwnd == 0 {
		return nil, ErrNotFound
	}
	return &Bar{className: PrimaryClassName, hwnd: hwnd}, nil
}

// FindSecondary enumerates every currently-open secondary taskbar window
// (one per extra monitor showing a taskbar). It never errors: zero results
// just means there is nothing extra to colorize right now.
func FindSecondary() []*Bar {
	var bars []*Bar
	var after win32.HWND
	for {
		hwnd := win32.FindWindowExW(0, after, SecondaryClassName, "")
		if hwnd == 0 {
			break
		}
		bars = append(bars, &Bar{className: SecondaryClassName, hwnd: hwnd})
		after = hwnd
	}
	return bars
}

// FindAll returns the primary taskbar plus every currently visible
// secondary taskbar.
func FindAll() ([]*Bar, error) {
	primary, err := Find()
	if err != nil {
		return nil, err
	}
	return append([]*Bar{primary}, FindSecondary()...), nil
}

// HWND returns the taskbar window's current handle, primarily so callers can
// tell two Bar values for the same underlying window apart.
func (b *Bar) HWND() win32.HWND { return b.hwnd }

// Refresh re-validates the taskbar's handle and re-reads its current screen
// rectangle. For the primary taskbar (of which there is always exactly one)
// it will re-find the window by class if Explorer has restarted. Secondary
// taskbars are not re-acquired this way, since with multiple monitors a
// class-name lookup could silently latch onto the wrong monitor's bar;
// instead the caller is expected to periodically call FindAll again and
// reconcile its set of tracked Bars.
func (b *Bar) Refresh() error {
	if b.hwnd == 0 || !win32.IsWindow(b.hwnd) {
		if b.className != PrimaryClassName {
			return ErrNotFound
		}
		hwnd := win32.FindWindowW(PrimaryClassName, "")
		if hwnd == 0 {
			return ErrNotFound
		}
		b.hwnd = hwnd
	}

	rect, ok := win32.GetWindowRect(b.hwnd)
	if !ok {
		return ErrNotFound
	}
	b.rect = rect

	if tray := win32.FindWindowExW(b.hwnd, 0, trayNotifyClassName, ""); tray != 0 {
		if trayRect, ok := win32.GetWindowRect(tray); ok {
			b.trayRect = trayRect
			b.hasTray = true
		} else {
			b.hasTray = false
		}
	} else {
		b.hasTray = false
	}
	return nil
}

// Visible reports whether the taskbar currently occupies a non-empty screen
// rectangle (it can be zero-sized while auto-hidden and off-screen).
func (b *Bar) Visible() bool {
	return win32.IsWindow(b.hwnd) && win32.IsWindowVisible(b.hwnd) && !b.rect.Empty()
}

// Rect returns the taskbar's full screen rectangle as of the last Refresh.
func (b *Bar) Rect() win32.RECT {
	return b.rect
}

// TargetRect returns the rectangle that should be colorized: the full
// taskbar rect when includeTray is true, or that rect with the system
// tray/clock band trimmed off the end when includeTray is false.
//
// Trimming is alignment-agnostic: it works the same whether the running-app
// icons are left-aligned or centered (Windows 11's default), because it is
// computed from TrayNotifyWnd's actual position each frame rather than an
// assumed icon origin. Precisely isolating only the running-app icon
// cluster (excluding the Start button, search, task view and widgets) would
// require UI Automation and is not attempted here - see the README for
// details. Secondary taskbars have no TrayNotifyWnd, so includeTray has no
// effect on them.
func (b *Bar) TargetRect(includeTray bool) win32.RECT {
	if includeTray || !b.hasTray {
		return b.rect
	}

	r := b.rect
	horizontal := r.Width() >= r.Height()
	if horizontal {
		// The tray is docked at whichever horizontal end has less space on
		// its far side; on the vast majority of setups that's the right
		// edge, but this still holds if the taskbar is dragged to a monitor
		// where the tray sits closer to the left.
		if b.trayRect.Left-r.Left > r.Right-b.trayRect.Right {
			r.Right = b.trayRect.Left
		} else {
			r.Left = b.trayRect.Right
		}
	} else {
		if b.trayRect.Top-r.Top > r.Bottom-b.trayRect.Bottom {
			r.Bottom = b.trayRect.Top
		} else {
			r.Top = b.trayRect.Bottom
		}
	}
	return r
}
