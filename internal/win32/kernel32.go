//go:build windows

package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
	procGetSystemPowerStatus  = kernel32.NewProc("GetSystemPowerStatus")
)

// GetModuleHandleW(nil) returns the calling process's own HINSTANCE, which
// is what window classes should be registered under.
func GetModuleHandleW() HMODULE {
	r, _, _ := procGetModuleHandleW.Call(0)
	return HMODULE(r)
}

// GetSystemPowerStatus reports the current battery/AC state.
func GetSystemPowerStatus() (SYSTEMPOWERSTATUS, bool) {
	var status SYSTEMPOWERSTATUS
	r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	return status, r != 0
}
