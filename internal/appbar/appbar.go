//go:build windows

// Package appbar registers a window as a Windows "Application Desktop
// Toolbar" (AppBar) - the same documented mechanism Explorer's own taskbar
// uses to reserve a strip of the screen along one edge, so maximized
// windows and the desktop work area make room for it.
//
// See https://learn.microsoft.com/windows/win32/shell/application-desktop-toolbars.
package appbar

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/fourhundredfour/pinky/internal/applog"
	"github.com/fourhundredfour/pinky/internal/config"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// sizeofAppBarData is the ABI size the shell expects in APPBARDATA.CbSize.
var sizeofAppBarData = uint32(unsafe.Sizeof(win32.APPBARDATA{}))

// Edge re-exports config.Edge so callers only need to import one package
// for the enum, while keeping appbar decoupled from config's other fields.
type Edge = config.Edge

// Callbacks lets the owner react to appbar notifications delivered on the
// window's own message queue (i.e. these fire on whatever thread pumps
// hwnd's messages - the same thread Wails' webview runs its message loop
// on).
type Callbacks struct {
	// OnPosChanged fires when the system asks every appbar to re-check its
	// position (another appbar registered/changed, display settings
	// changed, ...). The bar has already been re-queried/re-set by the time
	// this fires; the new rect is passed so the caller can move its window.
	OnPosChanged func(rect win32.RECT)
	// OnFullScreenApp fires when a full-screen app is activated/deactivated;
	// arg is true when a full-screen app just went foreground (the bar
	// should get out of the way) and false when it's safe to come back.
	OnFullScreenApp func(fullscreen bool)
	// OnTaskbarRecreated fires when Explorer (re)creates its own taskbar,
	// most commonly after `explorer.exe` restarts/crashes. The caller
	// should re-hide the real taskbar and re-register the appbar.
	OnTaskbarRecreated func()
}

// taskbarCreatedMsg is the well-known systemwide message broadcast by
// Explorer whenever it (re)creates its taskbar window.
var taskbarCreatedMsg = win32.RegisterWindowMessageW("TaskbarCreated")

// appBarCallbackMsg is pinky's own app-defined message identifier that the
// system uses to deliver appbar notifications (ABN_*) to Bar's subclass
// proc, registered once per process.
const appBarCallbackMsg = win32.WMApp + 100

// Bar is a registered AppBar bound to an existing top-level window (in this
// project, the Wails webview window's native HWND).
type Bar struct {
	hwnd      win32.HWND
	edge      uint32
	thickness int32
	monitor   win32.RECT

	prevWndProc uintptr
	callbacks   Callbacks
}

// Register turns hwnd into an AppBar docked to edge with the given
// thickness (in physical pixels) on the given monitor rect, and subclasses
// hwnd's window procedure so appbar/shell notifications reach callbacks.
// It returns the system-approved rectangle the caller should move/resize
// hwnd to.
func Register(hwnd win32.HWND, edge Edge, thickness int32, monitor win32.RECT, callbacks Callbacks) (*Bar, win32.RECT, error) {
	b := &Bar{
		hwnd:      hwnd,
		edge:      edgeConst(edge),
		thickness: thickness,
		monitor:   monitor,
		callbacks: callbacks,
	}

	data := win32.APPBARDATA{
		Hwnd:             hwnd,
		UCallbackMessage: appBarCallbackMsg,
	}
	data.CbSize = sizeofAppBarData
	if win32.SHAppBarMessage(win32.ABMNew, &data) == 0 {
		return nil, win32.RECT{}, fmt.Errorf("appbar: ABM_NEW failed (already registered?)")
	}

	b.subclass()

	rect, err := b.setPos()
	if err != nil {
		win32.SHAppBarMessage(win32.ABMRemove, &data)
		b.unsubclass()
		return nil, win32.RECT{}, err
	}
	return b, rect, nil
}

// Reposition re-queries and re-commits the bar's rectangle for a new edge
// and/or thickness (e.g. after a config reload), returning the
// system-approved rect.
func (b *Bar) Reposition(edge Edge, thickness int32, monitor win32.RECT) (win32.RECT, error) {
	b.edge = edgeConst(edge)
	b.thickness = thickness
	b.monitor = monitor
	return b.setPos()
}

// setPos runs the documented ABM_QUERYPOS -> ABM_SETPOS handshake and
// returns the rectangle the system approved.
func (b *Bar) setPos() (win32.RECT, error) {
	wanted := rectForEdge(b.edge, b.thickness, b.monitor)

	query := win32.APPBARDATA{Hwnd: b.hwnd, UEdge: b.edge, Rc: wanted}
	query.CbSize = sizeofAppBarData
	win32.SHAppBarMessage(win32.ABMQueryPos, &query)

	// Re-derive the correct thickness from the (possibly adjusted) rect
	// instead of trusting our own request, then commit it.
	approved := clampToThickness(b.edge, query.Rc, b.thickness)

	set := win32.APPBARDATA{Hwnd: b.hwnd, UEdge: b.edge, Rc: approved}
	set.CbSize = sizeofAppBarData
	win32.SHAppBarMessage(win32.ABMSetPos, &set)

	return set.Rc, nil
}

// Unregister removes the appbar registration and restores hwnd's original
// window procedure. Safe to call at most once.
func (b *Bar) Unregister() {
	data := win32.APPBARDATA{Hwnd: b.hwnd}
	data.CbSize = sizeofAppBarData
	win32.SHAppBarMessage(win32.ABMRemove, &data)
	b.unsubclass()
}

func (b *Bar) subclass() {
	proc := windows.NewCallback(b.wndProc)
	b.prevWndProc = win32.SetWindowLongPtrW(b.hwnd, win32.GWLPWndProc, proc)
}

func (b *Bar) unsubclass() {
	if b.prevWndProc != 0 {
		win32.SetWindowLongPtrW(b.hwnd, win32.GWLPWndProc, b.prevWndProc)
		b.prevWndProc = 0
	}
}

// wndProc is installed as hwnd's window procedure (subclassing it "in
// front of" whatever Wails installed there) so pinky can observe appbar
// and shell notifications without interfering with Wails' own message
// handling - every message it does not specifically act on is forwarded
// unchanged to the previous proc via CallWindowProcW.
func (b *Bar) wndProc(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	defer applog.RecoverAndLog("appbar-wndproc")
	switch {
	case msg == appBarCallbackMsg:
		b.onAppBarNotify(wParam, lParam)
	case taskbarCreatedMsg != 0 && msg == taskbarCreatedMsg:
		if b.callbacks.OnTaskbarRecreated != nil {
			b.callbacks.OnTaskbarRecreated()
		}
	case msg == win32.WMDisplayChange || msg == win32.WMSettingChange:
		if rect, err := b.setPos(); err == nil && b.callbacks.OnPosChanged != nil {
			b.callbacks.OnPosChanged(rect)
		}
	}
	return win32.CallWindowProcW(b.prevWndProc, hwnd, msg, wParam, lParam)
}

func (b *Bar) onAppBarNotify(notification, lParam uintptr) {
	switch notification {
	case win32.ABNPosChanged:
		if rect, err := b.setPos(); err == nil && b.callbacks.OnPosChanged != nil {
			b.callbacks.OnPosChanged(rect)
		}
	case win32.ABNFullScreenApp:
		if b.callbacks.OnFullScreenApp != nil {
			b.callbacks.OnFullScreenApp(lParam != 0)
		}
	}
}

func edgeConst(e Edge) uint32 {
	switch e {
	case config.EdgeLeft:
		return win32.ABELeft
	case config.EdgeTop:
		return win32.ABETop
	case config.EdgeRight:
		return win32.ABERight
	default:
		return win32.ABEBottom
	}
}

func rectForEdge(edge uint32, thickness int32, monitor win32.RECT) win32.RECT {
	switch edge {
	case win32.ABELeft:
		return win32.RECT{Left: monitor.Left, Top: monitor.Top, Right: monitor.Left + thickness, Bottom: monitor.Bottom}
	case win32.ABERight:
		return win32.RECT{Left: monitor.Right - thickness, Top: monitor.Top, Right: monitor.Right, Bottom: monitor.Bottom}
	case win32.ABETop:
		return win32.RECT{Left: monitor.Left, Top: monitor.Top, Right: monitor.Right, Bottom: monitor.Top + thickness}
	default: // ABEBottom
		return win32.RECT{Left: monitor.Left, Top: monitor.Bottom - thickness, Right: monitor.Right, Bottom: monitor.Bottom}
	}
}

// clampToThickness re-applies the requested thickness along the docked
// axis to whatever rect the system returned from ABM_QUERYPOS, which may
// have shrunk the cross-axis span (e.g. to avoid another appbar) but
// should not be trusted to preserve our exact thickness.
func clampToThickness(edge uint32, rect win32.RECT, thickness int32) win32.RECT {
	switch edge {
	case win32.ABELeft:
		rect.Right = rect.Left + thickness
	case win32.ABERight:
		rect.Left = rect.Right - thickness
	case win32.ABETop:
		rect.Bottom = rect.Top + thickness
	default: // ABEBottom
		rect.Top = rect.Bottom - thickness
	}
	return rect
}
