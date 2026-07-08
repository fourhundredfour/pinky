//go:build windows

package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procShellNotifyIconW        = shell32.NewProc("Shell_NotifyIconW")
	procSHAppBarMessage         = shell32.NewProc("SHAppBarMessage")
	procRegisterShellHookWindow = shell32.NewProc("RegisterShellHookWindow")
	procDeregisterShellHookWindow = shell32.NewProc("DeregisterShellHookWindow")
)

// NOTIFYICONDATAW mirrors the modern (Vista+) NOTIFYICONDATAW struct used by
// Shell_NotifyIconW to add/update/remove a system tray icon.
type NOTIFYICONDATAW struct {
	CbSize            uint32
	Hwnd              HWND
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             HICON
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UVersionOrTimeout uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          GUID
	HBalloonIcon      HICON
}

// Shell_NotifyIconW messages.
const (
	NIMAdd    = 0x00000000
	NIMModify = 0x00000001
	NIMDelete = 0x00000002
)

// NOTIFYICONDATAW.UFlags bits.
const (
	NIFMessage = 0x00000001
	NIFIcon    = 0x00000002
	NIFTip     = 0x00000004
)

func ShellNotifyIconW(message uint32, data *NOTIFYICONDATAW) bool {
	r, _, _ := procShellNotifyIconW.Call(uintptr(message), uintptr(unsafe.Pointer(data)))
	return r != 0
}

// SHAppBarMessage sends an AppBar registration/positioning message to the
// system; data is updated in place (e.g. ABM_QUERYPOS/ABM_SETPOS return the
// system-approved rectangle in data.Rc).
func SHAppBarMessage(message uint32, data *APPBARDATA) uintptr {
	r, _, _ := procSHAppBarMessage.Call(uintptr(message), uintptr(unsafe.Pointer(data)))
	return r
}

// RegisterShellHookWindow subscribes hwnd to shell hook notifications
// (window created/destroyed/activated/flashed, ...), delivered via the
// message returned by RegisterWindowMessageW("SHELLHOOK").
func RegisterShellHookWindow(hwnd HWND) bool {
	r, _, _ := procRegisterShellHookWindow.Call(uintptr(hwnd))
	return r != 0
}

func DeregisterShellHookWindow(hwnd HWND) bool {
	r, _, _ := procDeregisterShellHookWindow.Call(uintptr(hwnd))
	return r != 0
}

// SetUTF16 copies s (truncated if needed) null-terminated into dst.
func SetUTF16(dst []uint16, s string) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return
	}
	n := copy(dst, u)
	if n == len(dst) {
		dst[len(dst)-1] = 0
	}
}
