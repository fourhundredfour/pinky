//go:build windows

package tasks

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/fourhundredfour/pinky/internal/widget"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// EventUpdate is the Wails event name emitted whenever the open-window
// list changes.
const EventUpdate = "tasks:update"

// Service is bound to the frontend via application.NewService, exposing
// List/Focus/Minimize/Close, and separately drives EventUpdate pushes on
// window create/destroy/activate. It also implements widget.Widget so
// main.go can start/stop it uniformly alongside the other bar widgets.
type Service struct {
	app          *application.App
	selfHWND     win32.HWND
	pollInterval time.Duration

	lister  *Lister
	watcher *Watcher

	mu   sync.Mutex
	last []Window
}

// NewService creates the task-list service. selfHWND (pinky's own bar
// window) is excluded from its own listing; pass 0 if it is not known yet
// and call SetSelfHWND once it is, before Start. Call Start once the app's
// native window handle is available, and Stop on shutdown.
func NewService(app *application.App, selfHWND win32.HWND, pollInterval time.Duration) *Service {
	return &Service{app: app, selfHWND: selfHWND, pollInterval: pollInterval}
}

// SetSelfHWND updates the window excluded from the listing. Must be called
// before Start; services are registered with Wails (and therefore
// constructed) before pinky's own native window handle exists, so main.go
// fills it in once NativeWindow() becomes valid.
func (s *Service) SetSelfHWND(hwnd win32.HWND) {
	s.selfHWND = hwnd
}

// ID identifies this widget for the registry/plugin namespace.
func (s *Service) ID() string { return "tasks" }

// Zone places the open-window list in the bar's flexible zone.
func (s *Service) Zone() widget.Zone { return widget.ZoneTasks }

// Start begins watching selfHWND for shell-hook notifications and polls
// every pollInterval as a fallback.
func (s *Service) Start() error {
	s.lister = NewLister(s.selfHWND)
	s.refresh()
	s.watcher = NewWatcher(s.selfHWND, s.pollInterval, s.refresh)
	return nil
}

// Stop releases the shell hook subscription and stops polling.
func (s *Service) Stop() {
	if s.watcher != nil {
		s.watcher.Close()
	}
}

func (s *Service) refresh() {
	list := s.lister.List()
	s.mu.Lock()
	s.last = list
	s.mu.Unlock()
	if s.app != nil {
		s.app.Event.Emit(EventUpdate, list)
	}
}

// List returns the current open-window list. Bound to the frontend so it
// can fetch the initial state before the first EventUpdate arrives.
func (s *Service) List() []Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Window, len(s.last))
	copy(out, s.last)
	return out
}

// Focus brings the window identified by id to the foreground.
func (s *Service) Focus(id string) error {
	return Focus(id)
}

// Minimize minimizes the window identified by id.
func (s *Service) Minimize(id string) error {
	return Minimize(id)
}

// Close asks the window identified by id to close.
func (s *Service) Close(id string) error {
	return Close(id)
}

// GetIconPNGBytes returns the cached raw PNG bytes for the given HICON.
func (s *Service) GetIconPNGBytes(hicon uintptr) ([]byte, bool) {
	if s.lister == nil {
		return nil, false
	}
	return s.lister.GetIconPNGBytes(hicon)
}
