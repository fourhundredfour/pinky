// Package widget defines the minimal interface and registry pinky's
// internal bar widgets (tasks, clock, indicators, ...) implement.
//
// It is deliberately small today - just enough to start/stop every widget
// uniformly from main.go - but shaped so a future scripting layer (e.g. a
// Lua VM exposing a Neovim-style plugin API) could register additional
// widgets through the exact same Registry.Register seam, without any
// changes to this package or to how built-in widgets work.
package widget

// Zone is where on the bar a widget's frontend counterpart renders.
type Zone string

const (
	// ZoneTasks is the flexible, alignment-configurable open-windows region.
	ZoneTasks Zone = "tasks"
	// ZoneSystem is the fixed cluster at the far end of the bar (clock,
	// battery, network, volume, ...).
	ZoneSystem Zone = "system"
)

// Widget is anything that contributes backend state/events to the bar.
// Implementations are expected to also register a Wails service (via
// application.NewService) for whatever RPC surface their frontend
// counterpart needs; Widget itself only tracks lifecycle and placement so
// the registry can start/stop everything uniformly, regardless of what
// each widget's constructor needed.
type Widget interface {
	// ID uniquely identifies the widget. Used as its event/service
	// namespace today (e.g. "tasks", "clock") and, in the future, would
	// double as its plugin manifest name.
	ID() string
	// Zone reports where the widget's frontend counterpart belongs.
	Zone() Zone
	// Start begins whatever background work the widget needs (tickers,
	// watchers, ...). Called once, after every widget has been registered.
	Start() error
	// Stop releases any resources acquired by Start. Called on shutdown, in
	// reverse registration order.
	Stop()
}

// Registry tracks the set of active widgets. Built-in widgets register
// themselves at startup in main.go; a future plugin loader (e.g. Lua
// scripts invoking an exported "register widget" API) would use this same
// Register call.
type Registry struct {
	widgets []Widget
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a widget. Order matters only for shutdown, which happens
// in reverse.
func (r *Registry) Register(w Widget) {
	r.widgets = append(r.widgets, w)
}

// All returns a snapshot of the registered widgets.
func (r *Registry) All() []Widget {
	return append([]Widget(nil), r.widgets...)
}

// StartAll starts every registered widget in registration order. If one
// fails, every widget already started is stopped (in reverse) before the
// error is returned.
func (r *Registry) StartAll() error {
	for i, w := range r.widgets {
		if err := w.Start(); err != nil {
			for j := i - 1; j >= 0; j-- {
				r.widgets[j].Stop()
			}
			return err
		}
	}
	return nil
}

// StopAll stops every registered widget in reverse registration order.
func (r *Registry) StopAll() {
	for i := len(r.widgets) - 1; i >= 0; i-- {
		r.widgets[i].Stop()
	}
}
