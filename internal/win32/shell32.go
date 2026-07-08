package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

// GUID mirrors the Win32 GUID struct.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

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
