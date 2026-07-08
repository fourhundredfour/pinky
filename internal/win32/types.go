//go:build windows

// Package win32 provides the minimal set of raw Win32 API bindings pinky
// needs to register itself as an AppBar, hide the real Explorer taskbar,
// enumerate/control open windows, extract icons, and read system state
// (battery/volume/network).
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

// BITMAPINFOHEADER mirrors the Win32 BITMAPINFOHEADER struct.
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

// BITMAPINFO mirrors BITMAPINFO for a 32bpp BI_RGB bitmap, which never needs
// a color table, so no trailing RGBQUAD array is declared.
type BITMAPINFO struct {
	Header BITMAPINFOHEADER
}

// BITMAP mirrors the Win32 BITMAP struct, used to query an icon's mask/color
// bitmap dimensions before calling GetDIBits.
type BITMAP struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

// ICONINFO mirrors the Win32 ICONINFO struct returned by GetIconInfo.
type ICONINFO struct {
	FIcon    int32 // BOOL
	XHotspot uint32
	YHotspot uint32
	HbmMask  HBITMAP
	HbmColor HBITMAP
}

// BLENDFUNCTION mirrors the Win32 BLENDFUNCTION struct used by
// UpdateLayeredWindow.
type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

// GUID mirrors the Win32 GUID struct.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// APPBARDATA mirrors the Win32 APPBARDATA struct used by SHAppBarMessage.
// Field order/types matter: Go lays out uintptr-sized fields on natural
// 8-byte boundaries the same way the C compiler does, so this is ABI
// compatible without manual padding.
type APPBARDATA struct {
	CbSize           uint32
	Hwnd             HWND
	UCallbackMessage uint32
	UEdge            uint32
	Rc               RECT
	LParam           uintptr
}

// MONITORINFO mirrors the Win32 MONITORINFO struct returned by
// GetMonitorInfoW, used to find the full screen rect (RcMonitor) for AppBar
// placement, as opposed to the work area already shrunk by an existing
// taskbar.
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// SYSTEMPOWERSTATUS mirrors the Win32 SYSTEM_POWER_STATUS struct returned by
// GetSystemPowerStatus.
type SYSTEMPOWERSTATUS struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}
