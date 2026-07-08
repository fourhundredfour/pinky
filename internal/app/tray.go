package app

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/fourhundredfour/pinky/internal/config"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// trayCallbackMessage is the app-defined message Shell_NotifyIconW sends to
// the controller window on mouse activity over the tray icon.
const trayCallbackMessage = win32.WMApp + 1

const trayIconUID = 1

const (
	menuIDToggle = 1
	menuIDReload = 2
	menuIDQuit   = 3
)

func (a *App) setupTrayIcon() {
	data := a.trayIconData()
	data.UFlags = win32.NIFMessage | win32.NIFIcon | win32.NIFTip
	if !win32.ShellNotifyIconW(win32.NIMAdd, &data) {
		log.Printf("tray: failed to create tray icon (continuing without one)")
	}
}

func (a *App) removeTrayIcon() {
	data := a.trayIconData()
	win32.ShellNotifyIconW(win32.NIMDelete, &data)
}

func (a *App) updateTrayTooltip() {
	data := a.trayIconData()
	data.UFlags = win32.NIFTip
	win32.ShellNotifyIconW(win32.NIMModify, &data)
}

func (a *App) trayIconData() win32.NOTIFYICONDATAW {
	var data win32.NOTIFYICONDATAW
	data.CbSize = uint32(unsafe.Sizeof(data))
	data.Hwnd = a.controller
	data.UID = trayIconUID
	data.UCallbackMessage = trayCallbackMessage
	data.HIcon = win32.LoadIconW(0, win32.IDIApplication)

	status := "off"
	if a.cfg.Enabled {
		status = "on"
	}
	win32.SetUTF16(data.SzTip[:], fmt.Sprintf("pinky (%s) - %s %.0f%%", status, a.cfg.Mode, a.cfg.Opacity*100))
	return data
}

// onTrayCallback handles mouse activity forwarded from Shell_NotifyIconW.
// lParam carries the original mouse message (WM_LBUTTONUP / WM_RBUTTONUP);
// either click opens the same menu, matching how most tray apps behave when
// they only have one thing to show.
func (a *App) onTrayCallback(lParam uintptr) {
	switch uint32(lParam) {
	case win32.WMLButtonUp, win32.WMRButtonUp, win32.WMContextMenu:
		a.showTrayMenu()
	}
}

func (a *App) showTrayMenu() {
	menu := win32.CreatePopupMenu()
	if menu == 0 {
		return
	}
	defer win32.DestroyMenu(menu)

	toggleLabel := "Enable"
	if a.cfg.Enabled {
		toggleLabel = "Disable"
	}
	win32.AppendMenuW(menu, win32.MFString, menuIDToggle, toggleLabel)
	win32.AppendMenuW(menu, win32.MFString, menuIDReload, "Reload config")
	win32.AppendMenuW(menu, win32.MFSeparator, 0, "")
	win32.AppendMenuW(menu, win32.MFString, menuIDQuit, "Quit pinky")

	pos, _ := win32.GetCursorPos()
	win32.SetForegroundWindow(a.controller)
	cmd := win32.TrackPopupMenu(menu, win32.TPMRightButton|win32.TPMReturnCmd, pos.X, pos.Y, a.controller)

	switch cmd {
	case menuIDToggle:
		a.toggleEnabled()
	case menuIDReload:
		a.forceReloadConfig()
	case menuIDQuit:
		win32.DestroyWindow(a.controller)
	}
}

func (a *App) toggleEnabled() {
	cfg := *a.cfg
	cfg.Enabled = !cfg.Enabled
	a.cfg = &cfg
	if !cfg.Enabled {
		a.hideAll()
	}
	if err := config.Save(a.cfgPath, &cfg); err != nil {
		log.Printf("tray: failed to persist enabled=%v to config: %v", cfg.Enabled, err)
	}
	a.updateTrayTooltip()
}

func (a *App) forceReloadConfig() {
	cfg, err := config.Load(a.cfgPath)
	if err != nil {
		log.Printf("tray: reload failed, keeping previous settings: %v", err)
		return
	}
	a.cfg = cfg
	if cfg.FPS != a.lastFPS {
		win32.KillTimer(a.controller, timerRenderID)
		if _, err := win32.SetTimer(a.controller, timerRenderID, renderIntervalMs(cfg.FPS)); err != nil {
			log.Printf("tray: failed to apply new fps: %v", err)
		}
		a.lastFPS = cfg.FPS
	}
	if !cfg.Enabled {
		a.hideAll()
	}
	a.updateTrayTooltip()
	log.Printf("tray: config reloaded manually")
}
