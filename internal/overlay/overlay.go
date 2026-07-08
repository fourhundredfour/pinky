// Package overlay owns the layered, click-through window that pinky draws
// the processed taskbar image onto.
package overlay

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/fourhundredfour/pinky/internal/win32"
)

const className = "PinkyOverlayWindow"

// errClassAlreadyExists is what RegisterClassExW's underlying syscall
// returns when the class was already registered by a previous Overlay - one
// per connected monitor is expected, so this is not a real error.
const errClassAlreadyExists = syscall.Errno(1410)

var (
	registerOnce sync.Once
	registerErr  error
)

// Overlay is a WS_EX_LAYERED window that sits on top of the real taskbar.
//
//   - WS_EX_TRANSPARENT makes it click/hover-through, so the real taskbar
//     underneath keeps receiving mouse input normally.
//   - WS_EX_TOPMOST + reasserting HWND_TOPMOST on every update keeps it
//     above the taskbar even if the taskbar itself is briefly topmost.
//   - SetWindowDisplayAffinity(WDA_EXCLUDEFROMCAPTURE) removes it from every
//     capture pipeline (BitBlt, PrintWindow, WGC), which is what prevents
//     our own layer from ever being captured back into itself.
type Overlay struct {
	hwnd     win32.HWND
	screenDC win32.HDC
	memDC    win32.HDC
	bitmap   win32.HBITMAP
	oldBmp   win32.HGDIOBJ
	bits     unsafe.Pointer
	width    int32
	height   int32
	visible  bool
}

// New creates a hidden layered window, one per taskbar to be colorized
// (primary and/or each secondary monitor's). Overlay windows never need
// custom message handling - all app logic (timers, tray icon) lives on a
// separate controller window - so they always use DefWindowProc.
func New() (*Overlay, error) {
	instance := win32.GetModuleHandleW()

	classNamePtr, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return nil, fmt.Errorf("overlay: encoding class name: %w", err)
	}
	cursor := win32.LoadCursorW(0, win32.IDCArrow)

	registerOnce.Do(func() {
		defWndProc := windows.NewCallback(func(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
			return win32.DefWindowProcW(hwnd, msg, wParam, lParam)
		})
		wc := win32.WNDCLASSEXW{
			WndProc:   defWndProc,
			Instance:  instance,
			Cursor:    cursor,
			ClassName: classNamePtr,
		}
		wc.Size = uint32(unsafe.Sizeof(wc))
		if _, regErr := win32.RegisterClassExW(&wc); regErr != nil && !errors.Is(regErr, errClassAlreadyExists) {
			registerErr = fmt.Errorf("overlay: RegisterClassExW: %w", regErr)
		}
	})
	if registerErr != nil {
		return nil, registerErr
	}

	exStyle := uint32(win32.WSExLayered | win32.WSExTransparent | win32.WSExTopMost | win32.WSExToolWindow | win32.WSExNoActivate)
	hwnd, err := win32.CreateWindowExW(
		exStyle,
		classNamePtr, nil,
		win32.WSPopup,
		0, 0, 0, 0,
		0, 0, instance, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("overlay: CreateWindowExW: %w", err)
	}

	if !win32.SetWindowDisplayAffinity(hwnd, win32.WDAExcludeFromCapture) {
		return nil, fmt.Errorf("overlay: SetWindowDisplayAffinity failed (requires Windows 10 2004+/Windows 11)")
	}

	screenDC := win32.GetDC(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("overlay: GetDC(desktop) failed")
	}
	memDC := win32.CreateCompatibleDC(screenDC)
	if memDC == 0 {
		win32.ReleaseDC(0, screenDC)
		return nil, fmt.Errorf("overlay: CreateCompatibleDC failed")
	}

	return &Overlay{hwnd: hwnd, screenDC: screenDC, memDC: memDC}, nil
}

// Hwnd returns the overlay's window handle.
func (o *Overlay) Hwnd() win32.HWND { return o.hwnd }

func (o *Overlay) ensureSize(width, height int32) error {
	if o.bitmap != 0 && o.width == width && o.height == height {
		return nil
	}
	if o.bitmap != 0 {
		win32.SelectObject(o.memDC, o.oldBmp)
		win32.DeleteObject(win32.HGDIOBJ(o.bitmap))
		o.bitmap = 0
	}
	bmp, bits, err := win32.CreateDIBSection(o.screenDC, width, height)
	if err != nil {
		return fmt.Errorf("overlay: CreateDIBSection: %w", err)
	}
	o.bitmap = bmp
	o.bits = bits
	o.width = width
	o.height = height
	o.oldBmp = win32.SelectObject(o.memDC, win32.HGDIOBJ(bmp))
	return nil
}

// Update draws pix (a BGRA buffer matching rect's size) onto the overlay and
// (re)positions it to cover rect on screen, reasserting topmost z-order.
func (o *Overlay) Update(rect win32.RECT, pix []byte) error {
	width, height := rect.Width(), rect.Height()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("overlay: empty rect %+v", rect)
	}
	if err := o.ensureSize(width, height); err != nil {
		return err
	}

	n := int(width) * int(height) * 4
	if len(pix) < n {
		return fmt.Errorf("overlay: pixel buffer too small: got %d want %d", len(pix), n)
	}
	dst := unsafe.Slice((*byte)(o.bits), n)
	copy(dst, pix[:n])

	pos := win32.POINT{X: rect.Left, Y: rect.Top}
	size := win32.SIZE{CX: width, CY: height}
	srcPos := win32.POINT{X: 0, Y: 0}
	blend := win32.BLENDFUNCTION{
		BlendOp:             win32.ACSrcOver,
		SourceConstantAlpha: 255,
		AlphaFormat:         win32.ACSrcAlpha,
	}

	if !win32.UpdateLayeredWindow(o.hwnd, o.screenDC, &pos, &size, o.memDC, &srcPos, 0, &blend, win32.ULWAlpha) {
		return fmt.Errorf("overlay: UpdateLayeredWindow failed")
	}

	win32.SetWindowPos(o.hwnd, win32.HWNDTopMost, rect.Left, rect.Top, width, height,
		win32.SWPNoActivate|win32.SWPShowWindow)
	o.visible = true
	return nil
}

// Hide removes the overlay from the screen without destroying it, e.g. when
// the effect is disabled in config or the taskbar is temporarily unavailable.
func (o *Overlay) Hide() {
	if o.visible {
		win32.ShowWindow(o.hwnd, win32.SWHide)
		o.visible = false
	}
}

// Close releases all GDI/window resources held by the overlay.
func (o *Overlay) Close() {
	if o.bitmap != 0 {
		win32.SelectObject(o.memDC, o.oldBmp)
		win32.DeleteObject(win32.HGDIOBJ(o.bitmap))
		o.bitmap = 0
	}
	if o.memDC != 0 {
		win32.DeleteDC(o.memDC)
		o.memDC = 0
	}
	if o.screenDC != 0 {
		win32.ReleaseDC(0, o.screenDC)
		o.screenDC = 0
	}
	if o.hwnd != 0 {
		win32.DestroyWindow(o.hwnd)
		o.hwnd = 0
	}
}
