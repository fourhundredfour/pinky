package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procRegisterClassExW          = user32.NewProc("RegisterClassExW")
	procCreateWindowExW           = user32.NewProc("CreateWindowExW")
	procDefWindowProcW            = user32.NewProc("DefWindowProcW")
	procDestroyWindow             = user32.NewProc("DestroyWindow")
	procShowWindow                = user32.NewProc("ShowWindow")
	procGetMessageW               = user32.NewProc("GetMessageW")
	procTranslateMessage          = user32.NewProc("TranslateMessage")
	procDispatchMessageW          = user32.NewProc("DispatchMessageW")
	procPostQuitMessage           = user32.NewProc("PostQuitMessage")
	procSetWindowPos              = user32.NewProc("SetWindowPos")
	procSetTimer                  = user32.NewProc("SetTimer")
	procKillTimer                 = user32.NewProc("KillTimer")
	procFindWindowW               = user32.NewProc("FindWindowW")
	procFindWindowExW             = user32.NewProc("FindWindowExW")
	procGetWindowRect             = user32.NewProc("GetWindowRect")
	procGetDC                     = user32.NewProc("GetDC")
	procReleaseDC                 = user32.NewProc("ReleaseDC")
	procUpdateLayeredWindow       = user32.NewProc("UpdateLayeredWindow")
	procSetWindowDisplayAffinity  = user32.NewProc("SetWindowDisplayAffinity")
	procSetProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
	procLoadCursorW               = user32.NewProc("LoadCursorW")
	procLoadIconW                 = user32.NewProc("LoadIconW")
	procIsWindow                  = user32.NewProc("IsWindow")
	procIsWindowVisible           = user32.NewProc("IsWindowVisible")
	procCreatePopupMenu           = user32.NewProc("CreatePopupMenu")
	procAppendMenuW               = user32.NewProc("AppendMenuW")
	procTrackPopupMenu            = user32.NewProc("TrackPopupMenu")
	procDestroyMenu               = user32.NewProc("DestroyMenu")
	procSetForegroundWindow       = user32.NewProc("SetForegroundWindow")
	procGetCursorPos              = user32.NewProc("GetCursorPos")
)

func RegisterClassExW(wc *WNDCLASSEXW) (ATOM, error) {
	r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r == 0 {
		return 0, err
	}
	return ATOM(r), nil
}

func CreateWindowExW(exStyle uint32, className, windowName *uint16, style uint32,
	x, y, w, h int32, parent HWND, menu uintptr, instance HMODULE, param unsafe.Pointer) (HWND, error) {
	r, _, err := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(parent),
		menu,
		uintptr(instance),
		uintptr(param),
	)
	if r == 0 {
		return 0, err
	}
	return HWND(r), nil
}

func DefWindowProcW(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func DestroyWindow(hwnd HWND) bool {
	r, _, _ := procDestroyWindow.Call(uintptr(hwnd))
	return r != 0
}

func ShowWindow(hwnd HWND, cmdShow int32) bool {
	r, _, _ := procShowWindow.Call(uintptr(hwnd), uintptr(cmdShow))
	return r != 0
}

// GetMessageW returns >0 for a normal message, 0 for WM_QUIT, and a negative
// value (with the OS error set) on failure - matching the Win32 contract.
func GetMessageW(msg *MSG, hwnd HWND, msgFilterMin, msgFilterMax uint32) int32 {
	r, _, _ := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax),
	)
	return int32(r)
}

func TranslateMessage(msg *MSG) bool {
	r, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return r != 0
}

func DispatchMessageW(msg *MSG) uintptr {
	r, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
	return r
}

func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func SetWindowPos(hwnd, hwndInsertAfter HWND, x, y, cx, cy int32, flags uint32) bool {
	r, _, _ := procSetWindowPos.Call(
		uintptr(hwnd), uintptr(hwndInsertAfter),
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy),
		uintptr(flags),
	)
	return r != 0
}

func SetTimer(hwnd HWND, id uintptr, elapseMs uint32) (uintptr, error) {
	r, _, err := procSetTimer.Call(uintptr(hwnd), id, uintptr(elapseMs), 0)
	if r == 0 {
		return 0, err
	}
	return r, nil
}

func KillTimer(hwnd HWND, id uintptr) bool {
	r, _, _ := procKillTimer.Call(uintptr(hwnd), id)
	return r != 0
}

func FindWindowW(className, windowName string) HWND {
	var cls, win *uint16
	if className != "" {
		cls, _ = windows.UTF16PtrFromString(className)
	}
	if windowName != "" {
		win, _ = windows.UTF16PtrFromString(windowName)
	}
	r, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(win)))
	return HWND(r)
}

// FindWindowExW searches child windows of parent for a window with the given
// class name (and optional window name), starting after childAfter (0 to
// start from the first child).
func FindWindowExW(parent, childAfter HWND, className, windowName string) HWND {
	var cls, win *uint16
	if className != "" {
		cls, _ = windows.UTF16PtrFromString(className)
	}
	if windowName != "" {
		win, _ = windows.UTF16PtrFromString(windowName)
	}
	r, _, _ := procFindWindowExW.Call(
		uintptr(parent), uintptr(childAfter),
		uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(win)),
	)
	return HWND(r)
}

func GetWindowRect(hwnd HWND) (RECT, bool) {
	var rect RECT
	r, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	return rect, r != 0
}

func GetDC(hwnd HWND) HDC {
	r, _, _ := procGetDC.Call(uintptr(hwnd))
	return HDC(r)
}

func ReleaseDC(hwnd HWND, hdc HDC) bool {
	r, _, _ := procReleaseDC.Call(uintptr(hwnd), uintptr(hdc))
	return r != 0
}

// UpdateLayeredWindow pushes a fully rendered ARGB bitmap (selected into
// srcDC) onto a WS_EX_LAYERED window at the given screen position/size.
func UpdateLayeredWindow(hwnd HWND, hdcDst HDC, pos *POINT, size *SIZE, hdcSrc HDC,
	srcPos *POINT, colorKey uint32, blend *BLENDFUNCTION, flags uint32) bool {
	r, _, _ := procUpdateLayeredWindow.Call(
		uintptr(hwnd),
		uintptr(hdcDst),
		uintptr(unsafe.Pointer(pos)),
		uintptr(unsafe.Pointer(size)),
		uintptr(hdcSrc),
		uintptr(unsafe.Pointer(srcPos)),
		uintptr(colorKey),
		uintptr(unsafe.Pointer(blend)),
		uintptr(flags),
	)
	return r != 0
}

// SetWindowDisplayAffinity, when passed WDAExcludeFromCapture, removes hwnd
// from every screen-capture pipeline (BitBlt, PrintWindow, WGC, ...). This is
// what lets us safely capture the real taskbar without ever capturing our own
// overlay back into itself.
func SetWindowDisplayAffinity(hwnd HWND, affinity uint32) bool {
	r, _, _ := procSetWindowDisplayAffinity.Call(uintptr(hwnd), uintptr(affinity))
	return r != 0
}

// SetProcessDpiAwarenessContext makes the whole process Per-Monitor-V2 DPI
// aware so that all screen/window rects we read are in real physical pixels.
func SetProcessDpiAwarenessContext(context uintptr) bool {
	r, _, _ := procSetProcessDpiAwarenessCtx.Call(context)
	return r != 0
}

func LoadCursorW(instance HMODULE, cursorName uintptr) HCURSOR {
	r, _, _ := procLoadCursorW.Call(uintptr(instance), cursorName)
	return HCURSOR(r)
}

func IsWindow(hwnd HWND) bool {
	r, _, _ := procIsWindow.Call(uintptr(hwnd))
	return r != 0
}

func IsWindowVisible(hwnd HWND) bool {
	r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r != 0
}

// LoadIconW(0, id) loads one of the builtin system icons (e.g. IDIApplication).
func LoadIconW(instance HMODULE, iconName uintptr) HICON {
	r, _, _ := procLoadIconW.Call(uintptr(instance), iconName)
	return HICON(r)
}

func CreatePopupMenu() HMENU {
	r, _, _ := procCreatePopupMenu.Call()
	return HMENU(r)
}

func AppendMenuW(menu HMENU, flags uint32, idNewItem uintptr, item string) bool {
	var ptr *uint16
	if flags&MFSeparator == 0 {
		ptr, _ = windows.UTF16PtrFromString(item)
	}
	r, _, _ := procAppendMenuW.Call(uintptr(menu), uintptr(flags), idNewItem, uintptr(unsafe.Pointer(ptr)))
	return r != 0
}

// TrackPopupMenu displays the popup menu at (x,y) and, with TPMReturnCmd set
// in flags, blocks until the user picks an item or dismisses the menu,
// returning the chosen item's ID (or 0 if dismissed).
func TrackPopupMenu(menu HMENU, flags uint32, x, y int32, hwnd HWND) int32 {
	r, _, _ := procTrackPopupMenu.Call(
		uintptr(menu), uintptr(flags), uintptr(x), uintptr(y), 0, uintptr(hwnd), 0,
	)
	return int32(r)
}

func DestroyMenu(menu HMENU) bool {
	r, _, _ := procDestroyMenu.Call(uintptr(menu))
	return r != 0
}

// SetForegroundWindow must be called before TrackPopupMenu so the menu
// reliably closes when the user clicks away from it.
func SetForegroundWindow(hwnd HWND) bool {
	r, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return r != 0
}

func GetCursorPos() (POINT, bool) {
	var p POINT
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p, r != 0
}
