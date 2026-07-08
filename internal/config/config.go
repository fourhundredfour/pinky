// Package config loads and validates pinky's TOML configuration and lets
// callers watch it for changes so the running taskbar can hot-reload
// (appearance, layout, indicators, ...) without a restart.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Edge selects which screen edge the bar is docked to.
type Edge string

const (
	EdgeBottom Edge = "bottom"
	EdgeTop    Edge = "top"
	EdgeLeft   Edge = "left"
	EdgeRight  Edge = "right"
)

func (e Edge) valid() bool {
	switch e {
	case EdgeBottom, EdgeTop, EdgeLeft, EdgeRight:
		return true
	default:
		return false
	}
}

// Horizontal reports whether this edge runs along the top/bottom of the
// screen (as opposed to a vertical, left/right dock).
func (e Edge) Horizontal() bool { return e == EdgeTop || e == EdgeBottom }

// Alignment selects where the "open windows" zone sits within the bar.
type Alignment string

const (
	AlignLeft   Alignment = "left"
	AlignCenter Alignment = "center"
	AlignRight  Alignment = "right"
)

func (a Alignment) valid() bool {
	switch a {
	case AlignLeft, AlignCenter, AlignRight:
		return true
	default:
		return false
	}
}

// Shape selects the bar's corner style.
type Shape string

const (
	ShapeSquare  Shape = "square"
	ShapeRounded Shape = "rounded"
)

func (s Shape) valid() bool {
	switch s {
	case ShapeSquare, ShapeRounded:
		return true
	default:
		return false
	}
}

// Config is the on-disk (TOML) configuration for pinky.
type Config struct {
	// Edge is the screen edge the bar docks to.
	Edge Edge `toml:"edge"`
	// Size is the bar's thickness in pixels (height if docked to top/bottom,
	// width if docked to left/right).
	Size int `toml:"size"`
	// Alignment controls where the open-window icons sit within the bar;
	// the clock and system indicators always stay grouped at the far end.
	Alignment Alignment `toml:"alignment"`

	// Shape selects square or rounded corners.
	Shape Shape `toml:"shape"`
	// BackgroundColor is the bar's background as "#RRGGBB".
	BackgroundColor string `toml:"background_color"`
	// BackgroundOpacity in [0,1]; 0 is fully transparent (click-through look
	// against the desktop/other windows), 1 fully opaque.
	BackgroundOpacity float64 `toml:"background_opacity"`
	// AccentColor highlights the active window and hover states.
	AccentColor string `toml:"accent_color"`
	// MonochromeIcons desaturates running-app icons for a flatter look.
	MonochromeIcons bool `toml:"monochrome_icons"`

	// ClockFormat is a Go time layout string, e.g. "15:04" or "03:04 PM".
	ClockFormat string `toml:"clock_format"`
	// DateFormat is a Go time layout string used for the clock's tooltip.
	DateFormat string `toml:"date_format"`

	// ShowTasks/ShowClock/ShowBattery/ShowNetwork/ShowVolume toggle
	// individual widgets off without removing their config.
	ShowTasks   bool `toml:"show_tasks"`
	ShowClock   bool `toml:"show_clock"`
	ShowBattery bool `toml:"show_battery"`
	ShowNetwork bool `toml:"show_network"`
	ShowVolume  bool `toml:"show_volume"`

	// HideRealTaskbar controls whether Explorer's own taskbar is hidden
	// while pinky runs. Turn this off to run pinky side-by-side with the
	// real taskbar (e.g. while developing).
	HideRealTaskbar bool `toml:"hide_real_taskbar"`

	// TasksPollIntervalMs is the fallback poll interval for the open-window
	// list, used in addition to shell-hook notifications.
	TasksPollIntervalMs int `toml:"tasks_poll_interval_ms"`
	// IndicatorsPollIntervalMs controls how often battery/network/volume
	// are re-read.
	IndicatorsPollIntervalMs int `toml:"indicators_poll_interval_ms"`

	// Monitor selects which display the bar appears on. Only "primary" is
	// supported for now; multi-monitor support is a planned follow-up.
	Monitor string `toml:"monitor"`
}

// RGB is a parsed 8-bit-per-channel color.
type RGB struct {
	R, G, B uint8
}

// Default returns the built-in defaults, used both as the starting point
// before a TOML file is parsed and as the fallback if no file exists yet.
func Default() *Config {
	return &Config{
		Edge:              EdgeBottom,
		Size:              48,
		Alignment:         AlignCenter,
		Shape:             ShapeRounded,
		BackgroundColor:   "#101014",
		BackgroundOpacity: 0.85,
		AccentColor:       "#3AA0FF",
		MonochromeIcons:   false,
		ClockFormat:       "15:04",
		DateFormat:        "Monday, 02 January 2006",
		ShowTasks:         true,
		ShowClock:         true,
		ShowBattery:       true,
		ShowNetwork:       true,
		ShowVolume:        true,
		HideRealTaskbar:   true,

		TasksPollIntervalMs:      1500,
		IndicatorsPollIntervalMs: 2000,

		Monitor: "primary",
	}
}

// Load reads and parses the TOML file at path, applying defaults for any
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
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}
	cfg.applyBoundsAndDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to path as TOML, creating parent directories as needed.
func Save(path string, cfg *Config) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	// Rename-over-existing gives watchers (fsnotify) a single atomic change
	// instead of a truncate-then-write they could observe mid-write.
	return os.Rename(tmp, path)
}

// applyBoundsAndDefaults fixes up zero-value / out-of-range fields that
// would otherwise slip through TOML unmarshalling (e.g. a config file that
// only sets one field and leaves everything else absent/zero).
func (c *Config) applyBoundsAndDefaults() {
	def := Default()
	if c.Edge == "" {
		c.Edge = def.Edge
	}
	if c.Size <= 0 {
		c.Size = def.Size
	}
	if c.Size > 200 {
		c.Size = 200
	}
	if c.Alignment == "" {
		c.Alignment = def.Alignment
	}
	if c.Shape == "" {
		c.Shape = def.Shape
	}
	if c.BackgroundColor == "" {
		c.BackgroundColor = def.BackgroundColor
	}
	if c.BackgroundOpacity < 0 {
		c.BackgroundOpacity = 0
	}
	if c.BackgroundOpacity > 1 {
		c.BackgroundOpacity = 1
	}
	if c.AccentColor == "" {
		c.AccentColor = def.AccentColor
	}
	if c.ClockFormat == "" {
		c.ClockFormat = def.ClockFormat
	}
	if c.DateFormat == "" {
		c.DateFormat = def.DateFormat
	}
	if c.TasksPollIntervalMs <= 0 {
		c.TasksPollIntervalMs = def.TasksPollIntervalMs
	}
	if c.IndicatorsPollIntervalMs <= 0 {
		c.IndicatorsPollIntervalMs = def.IndicatorsPollIntervalMs
	}
	if c.Monitor == "" {
		c.Monitor = def.Monitor
	}
}

// Validate reports an error for settings that cannot be fixed up
// automatically (currently just unknown enum values or bad colors).
func (c *Config) Validate() error {
	if !c.Edge.valid() {
		return fmt.Errorf("unknown edge %q (want one of top, bottom, left, right)", c.Edge)
	}
	if !c.Alignment.valid() {
		return fmt.Errorf("unknown alignment %q (want one of left, center, right)", c.Alignment)
	}
	if !c.Shape.valid() {
		return fmt.Errorf("unknown shape %q (want one of square, rounded)", c.Shape)
	}
	if _, err := ParseColor(c.BackgroundColor); err != nil {
		return fmt.Errorf("background_color: %w", err)
	}
	if _, err := ParseColor(c.AccentColor); err != nil {
		return fmt.Errorf("accent_color: %w", err)
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
