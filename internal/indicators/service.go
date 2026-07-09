//go:build windows

package indicators

import (
	"os/exec"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/fourhundredfour/pinky/internal/applog"
	"github.com/fourhundredfour/pinky/internal/widget"
)

// EventUpdate is the Wails event name emitted whenever any indicator
// changes.
const EventUpdate = "indicators:update"

// Update is the combined payload pushed to the frontend.
type Update struct {
	Battery Battery `json:"battery"`
	Network Network `json:"network"`
	Volume  Volume  `json:"volume"`
}

// Service is bound to the frontend via application.NewService, exposing
// SetVolume/ToggleMute, and separately polls battery/network/volume on an
// interval, pushing EventUpdate whenever it does. It also implements
// widget.Widget so main.go can start/stop it uniformly alongside the other
// bar widgets.
type Service struct {
	app      *application.App
	volume   *volumeController
	interval time.Duration

	mu   sync.Mutex
	last Update
	stop chan struct{}
}

// NewService creates the indicators service, polling every interval.
func NewService(app *application.App, interval time.Duration) *Service {
	return &Service{app: app, volume: newVolumeController(), interval: interval}
}

// ID identifies this widget for the registry/plugin namespace.
func (s *Service) ID() string { return "indicators" }

// Zone places battery/network/volume in the bar's fixed system cluster.
func (s *Service) Zone() widget.Zone { return widget.ZoneSystem }

// Start begins polling every interval. Call Stop on shutdown.
func (s *Service) Start() error {
	s.stop = make(chan struct{})
	applog.Go("indicators-poll", func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		s.refresh()
		for {
			select {
			case <-ticker.C:
				s.refresh()
			case <-s.stop:
				return
			}
		}
	})
	return nil
}

// Stop halts polling.
func (s *Service) Stop() {
	if s.stop != nil {
		close(s.stop)
	}
}

func (s *Service) refresh() {
	vol, _ := s.volume.Get()
	update := Update{
		Battery: readBattery(),
		Network: readNetwork(),
		Volume:  vol,
	}

	s.mu.Lock()
	s.last = update
	s.mu.Unlock()

	if s.app != nil {
		s.app.Event.Emit(EventUpdate, update)
	}
}

// Snapshot returns the most recently polled state. Bound to the frontend
// so it can fetch the initial state before the first EventUpdate arrives.
func (s *Service) Snapshot() Update {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

// SetVolume sets the master volume level (0.0-1.0).
func (s *Service) SetVolume(level float64) error {
	if err := s.volume.Set(level); err != nil {
		return err
	}
	s.refresh()
	return nil
}

// ToggleMute flips the master mute state.
func (s *Service) ToggleMute() error {
	if _, err := s.volume.ToggleMute(); err != nil {
		return err
	}
	s.refresh()
	return nil
}

// OpenNetworkFlyout opens Windows' own Quick Settings/network flyout,
// since pinky does not (in v1) reimplement the network picker itself.
func (s *Service) OpenNetworkFlyout() error {
	return exec.Command("explorer.exe", "ms-actioncenter:controlcenter/networkflyout").Start()
}

// OpenSoundFlyout opens Windows' own Quick Settings/volume flyout.
func (s *Service) OpenSoundFlyout() error {
	return exec.Command("explorer.exe", "ms-actioncenter:controlcenter/audioflyout").Start()
}

// OpenActionCenter opens the full Windows 11 Quick Settings panel.
func (s *Service) OpenActionCenter() error {
	return exec.Command("explorer.exe", "ms-actioncenter:controlcenter").Start()
}
