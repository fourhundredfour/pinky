package config

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventUpdate is the Wails event name emitted whenever the effective
// config changes, whether from an on-disk edit (via Watcher) or a Save
// call triggered by the frontend.
const EventUpdate = "config:update"

// Service is bound to the frontend via application.NewService, exposing
// Get/Save, and is also the single funnel both the on-disk Watcher and
// frontend-initiated saves go through to notify the rest of the app of a
// new config.
type Service struct {
	app  *application.App
	path string

	mu      sync.Mutex
	current *Config
	onApply func(*Config)
}

// NewService creates the config service. onApply is called (in addition
// to emitting EventUpdate to the frontend) every time the effective config
// changes, so main.go can re-wire the appbar/explorer/widgets.
func NewService(app *application.App, path string, initial *Config, onApply func(*Config)) *Service {
	return &Service{app: app, path: path, current: initial, onApply: onApply}
}

// Get returns the current effective config. Bound to the frontend for its
// initial fetch before the first EventUpdate arrives.
func (s *Service) Get() *Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := *s.current
	return &cfg
}

// Save validates, persists, and applies cfg. Bound to the frontend for a
// future settings UI; also used internally by anything that programmatically
// changes settings.
func (s *Service) Save(cfg *Config) error {
	cfg.applyBoundsAndDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := Save(s.path, cfg); err != nil {
		return err
	}
	s.Apply(cfg)
	return nil
}

// Apply updates the in-memory current config and notifies both the
// frontend (EventUpdate) and the backend (onApply) without touching disk -
// used by the fsnotify Watcher's onChange callback, since the file is
// already saved at that point.
func (s *Service) Apply(cfg *Config) {
	s.mu.Lock()
	s.current = cfg
	s.mu.Unlock()

	if s.app != nil {
		s.app.Event.Emit(EventUpdate, cfg)
	}
	if s.onApply != nil {
		s.onApply(cfg)
	}
}
