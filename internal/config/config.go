// Package config loads pinky's YAML configuration from disk and supports
// polling it for changes so the running overlay can hot-reload without a
// restart.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Mode selects the graphic-design blend mode applied to the captured
// taskbar image before it is drawn back as the overlay layer.
type Mode string

const (
	// ModeMonochrome converts the source to grayscale, then multiplies it by
	// the configured color - a classic "monochrome tint".
	ModeMonochrome Mode = "monochrome"
	// ModeTint is a flat "Normal" blend: the layer is just the solid color,
	// and Opacity controls how strongly it replaces the source.
	ModeTint Mode = "tint"
	// ModeMultiply reproduces Photoshop's Multiply blend mode.
	ModeMultiply Mode = "multiply"
	// ModeColor reproduces Photoshop's Color blend mode (keeps the source's
	// luminance, takes hue+saturation from the configured color).
	ModeColor Mode = "color"
)

func (m Mode) valid() bool {
	switch m {
	case ModeMonochrome, ModeTint, ModeMultiply, ModeColor:
		return true
	default:
		return false
	}
}

// Config is the on-disk (YAML) configuration for pinky.
type Config struct {
	// Enabled toggles the whole overlay on/off without closing the app.
	Enabled bool `yaml:"enabled"`
	// Color is the layer color as a "#RRGGBB" hex string.
	Color string `yaml:"color"`
	// Opacity is the layer strength in [0,1]; 0 shows the real taskbar
	// untouched, 1 shows the fully blended result.
	Opacity float64 `yaml:"opacity"`
	// Mode selects the blend algorithm. See the Mode* constants.
	Mode Mode `yaml:"mode"`
	// FPS controls how often the taskbar is re-captured and re-drawn.
	FPS int `yaml:"fps"`
	// IncludeTray, when true (default), colorizes the entire taskbar strip
	// including the system tray/clock. When false, the system tray/clock
	// area is left uncolored so only the running-app icon band is affected.
	IncludeTray bool `yaml:"include_tray"`
}

// RGB is a parsed 8-bit-per-channel color.
type RGB struct {
	R, G, B uint8
}

// Default returns the built-in defaults, used both as the starting point
// before a YAML file is parsed and as the fallback if no file exists yet.
func Default() *Config {
	return &Config{
		Enabled:     true,
		Color:       "#FF33AA",
		Opacity:     0.8,
		Mode:        ModeMonochrome,
		FPS:         30,
		IncludeTray: true,
	}
}

// Load reads and parses the YAML file at path, applying defaults for any
// field left unset and validating the result. If the file does not exist,
// it is created with the built-in defaults so the user has something to
// edit.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := Default()
		if writeErr := Save(path, cfg); writeErr != nil {
			return nil, fmt.Errorf("config file %q does not exist and could not be created: %w", path, writeErr)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}
	cfg.applyBoundsAndDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to path as YAML, creating parent directories as needed.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// applyBoundsAndDefaults fixes up zero-value / out-of-range fields that
// would otherwise slip through YAML unmarshalling (e.g. a config file that
// only sets "color:" and leaves everything else absent).
func (c *Config) applyBoundsAndDefaults() {
	if c.Color == "" {
		c.Color = Default().Color
	}
	if c.Mode == "" {
		c.Mode = Default().Mode
	}
	if c.FPS <= 0 {
		c.FPS = Default().FPS
	}
	if c.FPS > 144 {
		c.FPS = 144
	}
	if c.Opacity < 0 {
		c.Opacity = 0
	}
	if c.Opacity > 1 {
		c.Opacity = 1
	}
}

// Validate reports an error for settings that cannot be fixed up
// automatically (currently just an unknown blend mode or bad color).
func (c *Config) Validate() error {
	if !c.Mode.valid() {
		return fmt.Errorf("unknown mode %q (want one of monochrome, tint, multiply, color)", c.Mode)
	}
	if _, err := ParseColor(c.Color); err != nil {
		return fmt.Errorf("color: %w", err)
	}
	return nil
}

// ParseColor parses a "#RRGGBB" or "RRGGBB" hex string.
func ParseColor(hex string) (RGB, error) {
	s := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(s) != 6 {
		return RGB{}, fmt.Errorf("color %q must be in #RRGGBB format", hex)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("color %q is not valid hex: %w", hex, err)
	}
	return RGB{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
	}, nil
}

// Watcher polls a config file's modification time and re-loads it when it
// changes, so edits take effect without restarting the app.
type Watcher struct {
	path    string
	modTime time.Time
}

// NewWatcher creates a watcher for path. It does not load the file itself;
// call Poll (or use config.Load separately) to get the initial value.
func NewWatcher(path string) *Watcher {
	return &Watcher{path: path}
}

// Poll checks whether the config file has changed on disk since the last
// call. If it has (or this is the first call), it reloads and returns the
// new config with changed=true. If the file is unchanged, or reloading it
// fails (e.g. mid-write, transiently invalid YAML), it returns
// changed=false and no error so the caller keeps running with the last-good
// config.
func (w *Watcher) Poll() (cfg *Config, changed bool, err error) {
	info, statErr := os.Stat(w.path)
	if statErr != nil {
		return nil, false, nil
	}
	if !info.ModTime().After(w.modTime) && !w.modTime.IsZero() {
		return nil, false, nil
	}

	loaded, loadErr := Load(w.path)
	if loadErr != nil {
		// Keep the previous modTime so we retry on the next poll instead of
		// silently giving up on a transiently-invalid file.
		return nil, false, loadErr
	}
	w.modTime = info.ModTime()
	return loaded, true, nil
}
