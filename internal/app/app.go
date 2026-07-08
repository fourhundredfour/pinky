// Package app wires config, taskbar geometry, capture, blending and the
// overlay window(s) together into the render loop.
package app

import (
	"log"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/fourhundredfour/pinky/internal/blend"
	"github.com/fourhundredfour/pinky/internal/capture"
	"github.com/fourhundredfour/pinky/internal/config"
	"github.com/fourhundredfour/pinky/internal/mask"
	"github.com/fourhundredfour/pinky/internal/overlay"
	"github.com/fourhundredfour/pinky/internal/taskbar"
	"github.com/fourhundredfour/pinky/internal/uia"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// rectProvider supplies the current icon/button cell rectangles (screen
// coordinates) for a taskbar window. Implemented by the UIA worker; an
// interface so the render logic can be exercised with a fake in tests.
type rectProvider interface {
	RectsFor(hwnd win32.HWND) []win32.RECT
}

const (
	timerRenderID uintptr = 1
	timerConfigID uintptr = 2
	timerRescanID uintptr = 3

	configPollMs uint32 = 1000
	rescanMs     uint32 = 2000

	controllerClassName = "PinkyControllerWindow"
)

// monitorTarget bundles one taskbar (primary or secondary) with its own
// capture and overlay resources, since each monitor's taskbar occupies a
// different screen region and needs its own GDI buffers and layered window.
type monitorTarget struct {
	bar      *taskbar.Bar
	capturer *capture.Capturer
	ov       *overlay.Overlay
}

func (t *monitorTarget) close() {
	t.ov.Close()
	t.capturer.Close()
}

// App owns every long-lived resource pinky needs and drives the render loop
// entirely from Win32 timer messages delivered to a single hidden
// controller window, so all GDI/User32 calls happen on the one OS thread
// that owns it - no locking required.
type App struct {
	cfgPath string
	cfg     *config.Config
	watcher *config.Watcher

	controller win32.HWND
	targets    []*monitorTarget

	// rects provides icon cell rectangles; uiaWorker is the concrete
	// implementation (kept separately so Run can start/stop it).
	rects     rectProvider
	uiaWorker *uia.Worker

	// hwndList is a concurrency-safe snapshot of the taskbar window handles,
	// read by the UIA worker goroutine and written on the message thread
	// whenever the set of targets changes.
	hwndMu   sync.Mutex
	hwndList []win32.HWND

	lastFPS int
}

// New loads the config and locates the primary taskbar, but does not create
// any windows yet - call Run for that.
func New(cfgPath string) (*App, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	bars, err := taskbar.FindAll()
	if err != nil {
		return nil, err
	}

	a := &App{
		cfgPath: cfgPath,
		cfg:     cfg,
		watcher: config.NewWatcher(cfgPath),
		lastFPS: cfg.FPS,
	}
	for _, bar := range bars {
		target, err := newMonitorTarget(bar)
		if err != nil {
			a.closeTargets()
			return nil, err
		}
		a.targets = append(a.targets, target)
	}
	a.syncHWNDs()
	return a, nil
}

// syncHWNDs republishes the taskbar handle list for the UIA worker. Called
// on the message thread whenever a.targets changes.
func (a *App) syncHWNDs() {
	list := make([]win32.HWND, 0, len(a.targets))
	for _, t := range a.targets {
		list = append(list, t.bar.HWND())
	}
	a.hwndMu.Lock()
	a.hwndList = list
	a.hwndMu.Unlock()
}

// currentHWNDs returns a copy of the tracked taskbar handles, safe to call
// from the UIA worker goroutine.
func (a *App) currentHWNDs() []win32.HWND {
	a.hwndMu.Lock()
	defer a.hwndMu.Unlock()
	out := make([]win32.HWND, len(a.hwndList))
	copy(out, a.hwndList)
	return out
}

func newMonitorTarget(bar *taskbar.Bar) (*monitorTarget, error) {
	capturer, err := capture.New()
	if err != nil {
		return nil, err
	}
	ov, err := overlay.New()
	if err != nil {
		capturer.Close()
		return nil, err
	}
	return &monitorTarget{bar: bar, capturer: capturer, ov: ov}, nil
}

func (a *App) closeTargets() {
	for _, t := range a.targets {
		t.close()
	}
	a.targets = nil
}

// Run creates the controller window and blocks on the Win32 message loop
// until the process is asked to exit (e.g. WM_DESTROY, or Quit from the
// tray menu). Must be called from the goroutine that is meant to own the
// window (main should runtime.LockOSThread first).
func (a *App) Run() error {
	hwnd, err := a.createController()
	if err != nil {
		return err
	}
	a.controller = hwnd
	defer a.closeTargets()
	defer a.removeTrayIcon()

	a.uiaWorker = uia.NewWorker(a.currentHWNDs)
	a.rects = a.uiaWorker
	a.uiaWorker.Start()
	defer a.uiaWorker.Stop()

	if _, err := win32.SetTimer(hwnd, timerRenderID, renderIntervalMs(a.cfg.FPS)); err != nil {
		return err
	}
	if _, err := win32.SetTimer(hwnd, timerConfigID, configPollMs); err != nil {
		return err
	}
	if _, err := win32.SetTimer(hwnd, timerRescanID, rescanMs); err != nil {
		return err
	}

	a.setupTrayIcon()

	log.Printf("pinky running (config: %s, mode: %s, color: %s, opacity: %.2f, fps: %d, include_tray: %v, icon_sensitivity: %.2f, monitors: %d)",
		a.cfgPath, a.cfg.Mode, a.cfg.Color, a.cfg.Opacity, a.cfg.FPS, a.cfg.IncludeTray, a.cfg.IconSensitivity, len(a.targets))

	var msg win32.MSG
	for {
		r := win32.GetMessageW(&msg, 0, 0, 0)
		if r <= 0 {
			break
		}
		win32.TranslateMessage(&msg)
		win32.DispatchMessageW(&msg)
	}
	return nil
}

// createController registers and creates a hidden, zero-size window whose
// sole purpose is to receive timer ticks and the tray icon's callback
// messages. It never renders anything itself.
func (a *App) createController() (win32.HWND, error) {
	instance := win32.GetModuleHandleW()
	classNamePtr, err := windows.UTF16PtrFromString(controllerClassName)
	if err != nil {
		return 0, err
	}

	wndProc := windows.NewCallback(a.wndProc)
	wc := win32.WNDCLASSEXW{
		WndProc:   wndProc,
		Instance:  instance,
		Cursor:    win32.LoadCursorW(0, win32.IDCArrow),
		ClassName: classNamePtr,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if _, err := win32.RegisterClassExW(&wc); err != nil {
		return 0, err
	}

	hwnd, err := win32.CreateWindowExW(0, classNamePtr, nil, win32.WSPopup, 0, 0, 0, 0, 0, 0, instance, nil)
	if err != nil {
		return 0, err
	}
	return hwnd, nil
}

func renderIntervalMs(fps int) uint32 {
	if fps <= 0 {
		fps = 30
	}
	return uint32(1000 / fps)
}

func (a *App) wndProc(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win32.WMTimer:
		switch wParam {
		case timerRenderID:
			a.onRenderTick()
		case timerConfigID:
			a.onConfigTick()
		case timerRescanID:
			a.onRescanTick()
		}
		return 0
	case trayCallbackMessage:
		a.onTrayCallback(lParam)
		return 0
	case win32.WMDestroy:
		win32.PostQuitMessage(0)
		return 0
	}
	return win32.DefWindowProcW(hwnd, msg, wParam, lParam)
}

func (a *App) onConfigTick() {
	cfg, changed, err := a.watcher.Poll()
	if err != nil {
		log.Printf("config: reload failed, keeping previous settings: %v", err)
		return
	}
	if !changed {
		return
	}

	log.Printf("config: reloaded (mode: %s, color: %s, opacity: %.2f, fps: %d, enabled: %v, include_tray: %v)",
		cfg.Mode, cfg.Color, cfg.Opacity, cfg.FPS, cfg.Enabled, cfg.IncludeTray)
	a.cfg = cfg

	if cfg.FPS != a.lastFPS {
		win32.KillTimer(a.controller, timerRenderID)
		if _, err := win32.SetTimer(a.controller, timerRenderID, renderIntervalMs(cfg.FPS)); err != nil {
			log.Printf("config: failed to apply new fps: %v", err)
		}
		a.lastFPS = cfg.FPS
	}
	if !cfg.Enabled {
		a.hideAll()
	}
	a.updateTrayTooltip()
}

// onRescanTick re-enumerates taskbar windows so secondary monitors that get
// connected/disconnected (or a taskbar recreated by an Explorer restart) are
// picked up without restarting pinky.
func (a *App) onRescanTick() {
	bars, err := taskbar.FindAll()
	if err != nil {
		// Primary taskbar missing entirely (Explorer likely restarting);
		// leave existing targets as-is, they'll self-heal via Bar.Refresh.
		return
	}

	known := make(map[win32.HWND]bool, len(a.targets))
	for _, t := range a.targets {
		known[t.bar.HWND()] = true
	}

	for _, bar := range bars {
		if known[bar.HWND()] {
			continue
		}
		target, err := newMonitorTarget(bar)
		if err != nil {
			log.Printf("rescan: failed to attach new monitor: %v", err)
			continue
		}
		log.Printf("rescan: attached new taskbar window")
		a.targets = append(a.targets, target)
	}

	seen := make(map[win32.HWND]bool, len(bars))
	for _, bar := range bars {
		seen[bar.HWND()] = true
	}
	kept := a.targets[:0]
	for _, t := range a.targets {
		if seen[t.bar.HWND()] || t.bar.HWND() == 0 {
			kept = append(kept, t)
			continue
		}
		log.Printf("rescan: detached a taskbar window that no longer exists")
		t.close()
	}
	a.targets = kept
	a.syncHWNDs()
}

func (a *App) hideAll() {
	for _, t := range a.targets {
		t.ov.Hide()
	}
}

func (a *App) onRenderTick() {
	cfg := a.cfg
	if !cfg.Enabled || cfg.Opacity <= 0 {
		a.hideAll()
		return
	}

	color, err := config.ParseColor(cfg.Color)
	if err != nil {
		// Already validated at load time; should not happen at runtime.
		log.Printf("invalid color %q: %v", cfg.Color, err)
		return
	}

	for _, t := range a.targets {
		if err := t.bar.Refresh(); err != nil {
			t.ov.Hide()
			continue
		}
		if !t.bar.Visible() {
			t.ov.Hide()
			continue
		}

		// Capture the full taskbar strip; include_tray now only decides which
		// icon cells get colored, not how much of the bar is captured.
		rect := t.bar.Rect()
		if rect.Empty() {
			t.ov.Hide()
			continue
		}

		frame, err := t.capturer.Capture(rect)
		if err != nil {
			log.Printf("capture failed: %v", err)
			continue
		}

		cells := a.cellsFor(t.bar, rect, cfg.IncludeTray)
		m := mask.Compute(frame.Pix, int(frame.Width), int(frame.Height), cells, cfg.IconSensitivity)
		blend.Apply(frame.Pix, m, blend.Params{Mode: cfg.Mode, Color: color, Opacity: cfg.Opacity})

		if err := t.ov.Update(rect, frame.Pix); err != nil {
			log.Printf("overlay update failed: %v", err)
		}
	}
}

// cellsFor maps the UIA-provided screen-space icon rectangles for a taskbar
// into strip-local coordinates for the mask, filtering out the system tray
// band when include_tray is false.
//
// It returns nil when no UIA rectangles are available yet, which signals the
// mask to fall back to whole-strip content detection. When UIA data exists
// but every cell is filtered out, it returns a non-nil empty slice so the
// mask stays fully transparent (no fallback).
func (a *App) cellsFor(bar *taskbar.Bar, capRect win32.RECT, includeTray bool) []mask.Rect {
	if a.rects == nil {
		return nil
	}
	screen := a.rects.RectsFor(bar.HWND())
	if len(screen) == 0 {
		return nil
	}
	trayRect, hasTray := bar.TrayRect()
	return mapCells(screen, capRect, trayRect, hasTray, includeTray)
}

// mapCells converts screen-space icon rectangles to strip-local mask cells,
// dropping tray-band cells when they should be excluded. It is a pure helper
// (no taskbar/UIA dependency) so the selection logic is unit-testable. The
// returned slice is always non-nil (possibly empty) so the caller can
// distinguish "UIA selected nothing" from "no UIA data".
func mapCells(screen []win32.RECT, capRect, trayRect win32.RECT, hasTray, includeTray bool) []mask.Rect {
	cells := make([]mask.Rect, 0, len(screen))
	for _, sc := range screen {
		if !includeTray && hasTray && centerIn(sc, trayRect) {
			continue
		}
		cells = append(cells, mask.Rect{
			Left:   int(sc.Left - capRect.Left),
			Top:    int(sc.Top - capRect.Top),
			Right:  int(sc.Right - capRect.Left),
			Bottom: int(sc.Bottom - capRect.Top),
		})
	}
	return cells
}

// centerIn reports whether the center of r lies within region.
func centerIn(r, region win32.RECT) bool {
	cx := (r.Left + r.Right) / 2
	cy := (r.Top + r.Bottom) / 2
	return cx >= region.Left && cx < region.Right && cy >= region.Top && cy < region.Bottom
}
