//go:build windows

// pinky is a Windows 11 taskbar replacement: a Wails v3 (WebView2) window
// docked to a screen edge via the Win32 AppBar API, backed by Go services
// for open windows, the clock, and system indicators. See
// internal/appbar, internal/explorer and internal/widget for the pieces
// this file wires together.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/fourhundredfour/pinky/internal/appbar"
	"github.com/fourhundredfour/pinky/internal/applog"
	"github.com/fourhundredfour/pinky/internal/clock"
	"github.com/fourhundredfour/pinky/internal/config"
	"github.com/fourhundredfour/pinky/internal/explorer"
	"github.com/fourhundredfour/pinky/internal/indicators"
	"github.com/fourhundredfour/pinky/internal/tasks"
	"github.com/fourhundredfour/pinky/internal/trayicon"
	"github.com/fourhundredfour/pinky/internal/widget"
	"github.com/fourhundredfour/pinky/internal/win32"
)

//go:embed frontend/dist
var distFS embed.FS

// reassertInterval is how often the real Explorer taskbar is re-hidden in
// case it decided to show itself again (observed after certain shell
// notifications) - see internal/explorer's package doc.
const reassertInterval = 3 * time.Second

func init() {
	// Set Go heap memory limit to 32MB to aggressively reclaim memory
	// and keep the RAM footprint extremely low.
	debug.SetMemoryLimit(32 * 1024 * 1024)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL PANIC in main: %v\nStack trace:\n%s", r, string(debug.Stack()))
			explorer.ForceShow()
			os.Exit(1)
		}
	}()

	// Per-Monitor-V2 DPI awareness keeps every rect this program computes
	// or receives (monitor bounds, AppBar rects, window bounds) in the
	// same physical-pixel space, with no scale-factor surprises.
	win32.SetProcessDpiAwarenessContext(win32.DPIAwarenessContextPerMonitorAwareV2)

	configPath, err := defaultConfigPath()
	if err != nil {
		log.Fatalf("pinky: could not determine config path: %v", err)
	}

	logPath := filepath.Join(filepath.Dir(configPath), "pinky.log")
	if err := applog.Init(logPath); err != nil {
		log.Printf("pinky: failed to initialize logging: %v", err)
	}

	log.Printf("pinky: starting up...")
	log.Printf("pinky: config path: %s", configPath)
	log.Printf("pinky: log path: %s", logPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("pinky: could not load config %q: %v", configPath, err)
	}

	assets, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		log.Fatalf("pinky: embedded frontend assets missing (did you run `npm run build` in frontend/?): %v", err)
	}

	iconHandler := &IconImageHandler{
		defaultHandler: application.BundledAssetFileServer(assets),
	}

	app := application.New(application.Options{
		Name:        "pinky",
		Description: "A customizable Windows 11 taskbar replacement",
		Icon:        trayicon.Icon(),
		Assets: application.AssetOptions{
			Handler: iconHandler,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:           "pinky-bar",
		URL:            "/",
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		BackgroundType: application.BackgroundTypeTransparent,
		// The real bar geometry is only known once the AppBar handshake
		// completes (needs the native HWND, so it happens in the
		// WindowShow hook below); this is just a reasonable placeholder
		// so there is no zero-sized window before then.
		Width:  1280,
		Height: cfg.Size,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
			DisableIcon:     true,
		},
		DefaultContextMenuDisabled: true,
	})

	explorerCtl := explorer.New()
	registry := widget.NewRegistry()

	clockSvc := clock.NewService(app, clock.Format{Time: cfg.ClockFormat, Date: cfg.DateFormat})
	tasksSvc := tasks.NewService(app, 0, time.Duration(cfg.TasksPollIntervalMs)*time.Millisecond)
	iconHandler.tasksSvc = tasksSvc
	indicatorsSvc := indicators.NewService(app, time.Duration(cfg.IndicatorsPollIntervalMs)*time.Millisecond)
	registry.Register(clockSvc)
	registry.Register(tasksSvc)
	registry.Register(indicatorsSvc)

	var (
		barMu sync.Mutex
		bar   *appbar.Bar
	)

	// applyRuntimeConfig pushes everything about cfg that lives outside
	// the frontend's CSS-variable theming (which config.Service's
	// EventUpdate already drives - see frontend/src/App.svelte) into the
	// native side: corner rounding, AppBar edge/size, and whether the
	// real Explorer taskbar stays hidden. Used both on startup and on
	// every subsequent config reload/save.
	applyRuntimeConfig := func(c *config.Config) {
		hwnd := selfHWND(window)
		if hwnd != 0 {
			pref := int32(win32.DWMWCPRound)
			if c.Shape == config.ShapeSquare {
				pref = win32.DWMWCPDoNotRound
			}
			win32.DwmSetWindowCornerPreference(hwnd, pref)
		}

		barMu.Lock()
		b := bar
		barMu.Unlock()
		if b != nil {
			if monitor, ok := win32.PrimaryMonitorRect(); ok {
				if rect, err := b.Reposition(c.Edge, int32(c.Size), monitor); err == nil {
					go window.SetPhysicalBounds(rectToWails(rect))
				}
			}
		}

		if c.HideRealTaskbar && b != nil {
			explorerCtl.Hide()
		} else {
			explorerCtl.Show()
		}

		clockSvc.SetFormat(clock.Format{Time: c.ClockFormat, Date: c.DateFormat})
	}

	configSvc := config.NewService(app, configPath, cfg, applyRuntimeConfig)

	app.RegisterService(application.NewService(configSvc))
	app.RegisterService(application.NewService(clockSvc))
	app.RegisterService(application.NewService(tasksSvc))
	app.RegisterService(application.NewService(indicatorsSvc))

	var (
		readyOnce      sync.Once
		watcher        *config.Watcher
		reassertTicker *time.Ticker
		stopReassert   = make(chan struct{})
	)

	ready := func() {
		readyOnce.Do(func() {
			hwnd := selfHWND(window)
			if hwnd == 0 {
				log.Printf("pinky: native window handle unavailable, cannot register as AppBar")
				return
			}
			tasksSvc.SetSelfHWND(hwnd)

			monitor, ok := win32.PrimaryMonitorRect()
			if !ok {
				log.Printf("pinky: could not determine primary monitor rect, AppBar geometry may be wrong")
			}

			b, rect, err := appbar.Register(hwnd, cfg.Edge, int32(cfg.Size), monitor, appbar.Callbacks{
				OnPosChanged: func(rect win32.RECT) {
					go window.SetPhysicalBounds(rectToWails(rect))
				},
				OnFullScreenApp: func(fullscreen bool) {
					go func() {
						if fullscreen {
							window.Hide()
						} else {
							window.Show()
						}
					}()
				},
				OnTaskbarRecreated: func() {
					explorerCtl.Reassert()
				},
			})
			if err != nil {
				log.Printf("pinky: could not register AppBar: %v", err)
			} else {
				barMu.Lock()
				bar = b
				barMu.Unlock()
				go window.SetPhysicalBounds(rectToWails(rect))
			}

			applyRuntimeConfig(cfg)

			if err := registry.StartAll(); err != nil {
				log.Printf("pinky: failed to start widgets: %v", err)
			}

			if w, err := config.NewWatcher(configPath, configSvc.Apply); err != nil {
				log.Printf("pinky: could not watch config file for changes: %v", err)
			} else {
				watcher = w
			}

			reassertTicker = time.NewTicker(reassertInterval)
			applog.Go("reassert-ticker", func() {
				for {
					select {
					case <-reassertTicker.C:
						explorerCtl.Reassert()
					case <-stopReassert:
						return
					}
				}
			})

			trayicon.Setup(app, trayicon.Options{
				ConfigPath: configPath,
				OnReload: func() {
					if c, err := config.Load(configPath); err == nil {
						configSvc.Apply(c)
					} else {
						log.Printf("pinky: manual config reload failed: %v", err)
					}
				},
				OnQuit: app.Quit,
			})

			window.Show()
		})
	}
	window.OnWindowEvent(events.Common.WindowShow, func(_ *application.WindowEvent) { ready() })

	app.OnShutdown(func() {
		close(stopReassert)
		if reassertTicker != nil {
			reassertTicker.Stop()
		}
		if watcher != nil {
			watcher.Close()
		}
		registry.StopAll()
		barMu.Lock()
		if bar != nil {
			bar.Unregister()
		}
		barMu.Unlock()
		explorerCtl.Show()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// selfHWND returns pinky's own native window handle, or 0 before the
// underlying OS window has been created (NativeWindow is only valid once
// Wails' own message loop has run at least once - see the WindowShow hook
// in main above).
func selfHWND(window *application.WebviewWindow) win32.HWND {
	return win32.HWND(uintptr(window.NativeWindow()))
}

// rectToWails converts a physical-pixel win32.RECT (as returned by the
// AppBar handshake) into the application.Rect SetPhysicalBounds expects.
func rectToWails(r win32.RECT) application.Rect {
	return application.Rect{
		X:      int(r.Left),
		Y:      int(r.Top),
		Width:  int(r.Width()),
		Height: int(r.Height()),
	}
}

// defaultConfigPath returns "<user config dir>/pinky/config.toml",
// falling back to a config.toml next to the executable if the user config
// directory cannot be determined (e.g. a locked-down environment).
func defaultConfigPath() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "pinky", "config.toml"), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "config.toml"), nil
}

// IconImageHandler intercepts requests to /icon and serves raw PNG bytes
// directly from the tasks service cache, bypassing base64 and JSON overhead.
type IconImageHandler struct {
	defaultHandler http.Handler
	tasksSvc       *tasks.Service
}

func (h *IconImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/icon" {
		hiconStr := r.URL.Query().Get("hicon")
		hiconVal, err := strconv.ParseUint(hiconStr, 16, 64)
		if err == nil && hiconVal != 0 {
			pngBytes, ok := h.tasksSvc.GetIconPNGBytes(uintptr(hiconVal))
			if ok && len(pngBytes) > 0 {
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache aggressively
				w.Write(pngBytes)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	h.defaultHandler.ServeHTTP(w, r)
}
