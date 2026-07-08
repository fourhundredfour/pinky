// Package tasks tracks the set of "open programs" that should appear on
// the bar, mirroring the classic Windows taskbar's window-listing rules.
//
// The selection rule (shouldList) is deliberately kept free of any Win32
// dependency so it can be unit-tested on any platform; the Windows-only
// half of this package (tasks_windows.go) is responsible for gathering the
// raw window attributes and turning the result into icons/actions.
package tasks

// Window is the data sent to the frontend for one open, listable window.
type Window struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	Active    bool   `json:"active"`
	Minimized bool   `json:"minimized"`
}

// The extended-style bits this package cares about, duplicated from the
// Win32 ABI (rather than importing internal/win32, which is Windows-only)
// so shouldList stays portable and testable.
const (
	exStyleToolWindow = 0x00000080
	exStyleAppWindow  = 0x00040000
)

// candidate holds the raw attributes of one top-level window, as gathered
// by the platform-specific enumerator.
type candidate struct {
	Title    string
	Visible  bool
	Cloaked  bool
	HasOwner bool
	ExStyle  uint32
}

// shouldList applies the standard Windows taskbar heuristic: a window
// appears if it's visible, has a title, isn't cloaked (suspended/hidden by
// DWM), and is either explicitly flagged WS_EX_APPWINDOW or is an
// unowned, non-tool window.
func shouldList(c candidate) bool {
	if !c.Visible || c.Cloaked {
		return false
	}
	if c.Title == "" {
		return false
	}
	if c.ExStyle&exStyleAppWindow != 0 {
		return true
	}
	if c.ExStyle&exStyleToolWindow != 0 {
		return false
	}
	return !c.HasOwner
}
