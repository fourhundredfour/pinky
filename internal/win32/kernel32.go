package win32

import "golang.org/x/sys/windows"

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

// GetModuleHandleW(nil) returns the calling process's own HINSTANCE, which
// is what window classes should be registered under.
func GetModuleHandleW() HMODULE {
	r, _, _ := procGetModuleHandleW.Call(0)
	return HMODULE(r)
}
