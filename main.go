// Command pinky colorizes the Windows 11 taskbar icons with a configurable
// graphic-design blend layer (monochrome, tint, multiply or color), read
// from a YAML config file and hot-reloaded while running.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fourhundredfour/pinky/internal/app"
	"github.com/fourhundredfour/pinky/internal/win32"
)

func main() {
	// All Win32 window/GDI calls in this program must happen on the same OS
	// thread that created the window and runs its message loop.
	runtime.LockOSThread()

	cfgPath := flag.String("config", defaultConfigPath(), "path to the YAML config file")
	flag.Parse()

	// Per-Monitor-V2 DPI awareness so every rect we read (taskbar, tray) and
	// draw (overlay) is in real physical pixels.
	if !win32.SetProcessDpiAwarenessContext(win32.DPIAwarenessContextPerMonitorAwareV2) {
		log.Println("warning: could not set Per-Monitor-V2 DPI awareness; overlay may misalign on scaled displays")
	}

	a, err := app.New(*cfgPath)
	if err != nil {
		log.Fatalf("pinky: %v", err)
	}
	if err := a.Run(); err != nil {
		log.Fatalf("pinky: %v", err)
	}
}

func defaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(filepath.Dir(exe), "config.yaml")
}
