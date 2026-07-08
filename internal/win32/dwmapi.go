//go:build windows

package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	dwmapi = windows.NewLazySystemDLL("dwmapi.dll")

	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

// DwmIsCloaked reports whether hwnd is currently cloaked (hidden from view
// by DWM even though it may report itself as visible - common for
// suspended UWP apps and virtual-desktop-hidden windows). Cloaked windows
// should not appear in a task list.
func DwmIsCloaked(hwnd HWND) bool {
	var cloaked uint32
	r, _, _ := procDwmGetWindowAttribute.Call(
		uintptr(hwnd), uintptr(DWMWACloaked),
		uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked),
	)
	return r == 0 && cloaked != 0
}

// DwmSetWindowCornerPreference requests rounded (or square) corners for
// hwnd on Windows 11; a no-op (returns false) on older systems.
func DwmSetWindowCornerPreference(hwnd HWND, pref int32) bool {
	r, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(hwnd), uintptr(DWMWAWindowCornerPref),
		uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref),
	)
	return r == 0
}
