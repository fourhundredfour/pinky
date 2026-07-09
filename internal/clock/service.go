//go:build windows

package clock

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/fourhundredfour/pinky/internal/applog"
	"github.com/fourhundredfour/pinky/internal/widget"
)

// EventTick is the Wails event name emitted once a second with the current
// Tick payload.
const EventTick = "clock:tick"

// Service is bound to the frontend via application.NewService, exposing
// Now() for the initial fetch, and separately drives EventTick pushes. It
// also implements widget.Widget so main.go can start/stop it uniformly
// alongside the other bar widgets.
type Service struct {
	app *application.App

	mu     sync.Mutex
	format Format
	stop   chan struct{}
}

// NewService creates the clock service with the given initial format.
func NewService(app *application.App, format Format) *Service {
	return &Service{app: app, format: format}
}

// ID identifies this widget for the registry/plugin namespace.
func (s *Service) ID() string { return "clock" }

// Zone places the clock in the bar's fixed system cluster.
func (s *Service) Zone() widget.Zone { return widget.ZoneSystem }

// Start begins the once-a-second tick loop. Call Stop on shutdown.
func (s *Service) Start() error {
	s.stop = make(chan struct{})
	applog.Go("clock-tick", func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		s.emit()
		for {
			select {
			case <-ticker.C:
				s.emit()
			case <-s.stop:
				return
			}
		}
	})
	return nil
}

// Stop halts the tick loop.
func (s *Service) Stop() {
	if s.stop != nil {
		close(s.stop)
	}
}

// SetFormat updates the render format, e.g. after a config reload.
func (s *Service) SetFormat(format Format) {
	s.mu.Lock()
	s.format = format
	s.mu.Unlock()
	s.emit()
}

func (s *Service) emit() {
	if s.app != nil {
		s.app.Event.Emit(EventTick, s.Now())
	}
}

// Now renders the current time. Bound to the frontend so it can fetch the
// initial state before the first EventTick arrives.
func (s *Service) Now() Tick {
	s.mu.Lock()
	format := s.format
	s.mu.Unlock()
	return Render(time.Now(), format)
}
