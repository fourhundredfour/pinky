// Package win32 provides the minimal set of raw Win32 API bindings needed to
// capture, transform and re-draw the Windows taskbar as a layered overlay.
//
// lxn/win (a common Go Win32 wrapper) does not expose UpdateLayeredWindow or
// SetWindowDisplayAffinity, which are the two calls this project is built
// around, so we bind directly against golang.org/x/sys/windows instead of
// pulling in a large wrapper library that would still need to be extended.
package win32

// Handle types. All are plain integers under the hood, matching the Win32
// convention of opaque HANDLE-like values.
type (
	HWND    uintptr
	HDC     uintptr
	HBITMAP uintptr
	HGDIOBJ uintptr
	HICON   uintptr
	HCURSOR uintptr
	HBRUSH  uintptr
	HMODULE uintptr
	HMENU   uintptr
	ATOM    uint16
)

// RECT mirrors the Win32 RECT struct (all screen coordinates in this project
// are physical pixels since the process is Per-Monitor-V2 DPI aware).
type RECT struct {
	Left, Top, Right, Bottom int32
}

func (r RECT) Width() int32  { return r.Right - r.Left }
func (r RECT) Height() int32 { return r.Bottom - r.Top }
func (r RECT) Empty() bool   { return r.Width() <= 0 || r.Height() <= 0 }
func (r RECT) Equal(o RECT) bool {
	return r.Left == o.Left && r.Top == o.Top && r.Right == o.Right && r.Bottom == o.Bottom
}

// POINT mirrors the Win32 POINT struct.
type POINT struct {
	X, Y int32
}

// SIZE mirrors the Win32 SIZE struct.
type SIZE struct {
	CX, CY int32
}

// MSG mirrors the Win32 MSG struct used by the message loop.
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// WNDCLASSEXW mirrors the Win32 WNDCLASSEXW struct.
type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   HMODULE
	Icon       HICON
	Cursor     HCURSOR
	Background HBRUSH
	MenuName   *uint16
	ClassName  *uint16
	IconSm     HICON
}

// BITMAPINFOHEADER mirrors the Win32 BITMAPINFOHEADER struct. For our use
// case (32bpp, BI_RGB) no color table follows the header, so we never need
// the full BITMAPINFO wrapper struct.
type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// BLENDFUNCTION mirrors the Win32 BLENDFUNCTION struct used by
// UpdateLayeredWindow.
type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}
