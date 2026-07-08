//go:build windows

// Package trayicon sets up pinky's own system tray icon and right-click
// menu (Settings/Reload/Quit) using Wails' native SystemTray manager.
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os/exec"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Options configures the tray menu's actions.
type Options struct {
	// ConfigPath is opened in the system default editor by "Open config".
	ConfigPath string
	// OnReload is invoked by "Reload config".
	OnReload func()
	// OnQuit is invoked by "Quit pinky"; defaults to app.Quit() if nil.
	OnQuit func()
}

// Setup creates and shows pinky's tray icon with its menu, returning it so
// the caller can update the tooltip later (e.g. to reflect live state).
func Setup(app *application.App, opts Options) *application.SystemTray {
	tray := app.SystemTray.New()
	tray.SetLabel("pinky")
	tray.SetTooltip("pinky - Windows 11 taskbar replacement")
	tray.SetIcon(Icon())

	menu := application.NewMenu()
	menu.Add("Open config.toml").OnClick(func(_ *application.Context) {
		_ = exec.Command("cmd", "/c", "start", "", opts.ConfigPath).Start()
	})
	menu.Add("Reload config").OnClick(func(_ *application.Context) {
		if opts.OnReload != nil {
			opts.OnReload()
		}
	})
	menu.AddSeparator()
	menu.Add("Quit pinky").OnClick(func(_ *application.Context) {
		if opts.OnQuit != nil {
			opts.OnQuit()
			return
		}
		app.Quit()
	})
	tray.SetMenu(menu)
	tray.Show()
	return tray
}

// Icon renders a small, distinctive placeholder tray/app icon at runtime
// (a filled circle in pinky's default accent color) so the project does
// not need to ship a binary icon asset in the repository.
func Icon() []byte {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill := color.RGBA{R: 0x3A, G: 0xA0, B: 0xFF, A: 0xFF}

	cx, cy, r := float64(size)/2, float64(size)/2, float64(size)/2-1
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r*r {
				img.SetRGBA(x, y, fill)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
