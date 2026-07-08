//go:build windows

package app

import (
	"testing"

	"github.com/fourhundredfour/pinky/internal/mask"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// fakeProvider is a stand-in for the UIA worker, letting the render-selection
// logic be tested without a live taskbar.
type fakeProvider struct {
	rects map[win32.HWND][]win32.RECT
}

func (f fakeProvider) RectsFor(hwnd win32.HWND) []win32.RECT { return f.rects[hwnd] }

func rect(l, t, r, b int32) win32.RECT {
	return win32.RECT{Left: l, Top: t, Right: r, Bottom: b}
}

func TestMapCellsToLocalCoords(t *testing.T) {
	cap := rect(100, 10, 500, 58) // capture origin at (100,10)
	icon := rect(150, 20, 190, 50)

	cells := mapCells([]win32.RECT{icon}, cap, win32.RECT{}, false, true)
	if len(cells) != 1 {
		t.Fatalf("got %d cells, want 1", len(cells))
	}
	want := mask.Rect{Left: 50, Top: 10, Right: 90, Bottom: 40}
	if cells[0] != want {
		t.Fatalf("mapped cell = %+v, want %+v", cells[0], want)
	}
}

func TestMapCellsExcludesTrayWhenDisabled(t *testing.T) {
	cap := rect(0, 0, 500, 48)
	tray := rect(400, 0, 500, 48)
	appIcon := rect(50, 5, 90, 43)
	trayIcon := rect(430, 5, 460, 43)

	// include_tray = false -> tray icon dropped.
	cells := mapCells([]win32.RECT{appIcon, trayIcon}, cap, tray, true, false)
	if len(cells) != 1 {
		t.Fatalf("got %d cells, want 1 (tray excluded)", len(cells))
	}
	if cells[0].Left != 50 {
		t.Fatalf("kept the wrong cell: %+v", cells[0])
	}

	// include_tray = true -> both kept.
	cells = mapCells([]win32.RECT{appIcon, trayIcon}, cap, tray, true, true)
	if len(cells) != 2 {
		t.Fatalf("got %d cells, want 2 (tray included)", len(cells))
	}
}

func TestMapCellsEmptyIsNonNil(t *testing.T) {
	cells := mapCells(nil, rect(0, 0, 10, 10), win32.RECT{}, false, true)
	if cells == nil {
		t.Fatalf("mapCells should return a non-nil slice so the mask does not fall back")
	}
	if len(cells) != 0 {
		t.Fatalf("expected empty slice, got %d", len(cells))
	}
}

func TestFakeProviderSatisfiesInterface(t *testing.T) {
	var p rectProvider = fakeProvider{rects: map[win32.HWND][]win32.RECT{
		win32.HWND(1): {rect(0, 0, 10, 10)},
	}}
	if got := p.RectsFor(win32.HWND(1)); len(got) != 1 {
		t.Fatalf("fake provider returned %d rects, want 1", len(got))
	}
	if got := p.RectsFor(win32.HWND(2)); got != nil {
		t.Fatalf("unknown hwnd should return nil, got %v", got)
	}
}

func TestCenterIn(t *testing.T) {
	region := rect(100, 0, 200, 50)
	if !centerIn(rect(140, 10, 160, 40), region) {
		t.Errorf("rect centered inside region should be detected as inside")
	}
	if centerIn(rect(0, 10, 40, 40), region) {
		t.Errorf("rect outside region should not be detected as inside")
	}
}
