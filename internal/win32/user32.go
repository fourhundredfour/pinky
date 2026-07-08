//go:build windows

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
	procPeekMessageW              = user32.NewProc("PeekMessageW")
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
	procIsIconic                  = user32.NewProc("IsIconic")
	procCreatePopupMenu           = user32.NewProc("CreatePopupMenu")
	procAppendMenuW               = user32.NewProc("AppendMenuW")
	procTrackPopupMenu            = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                = user32.NewProc("DestroyMenu")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procGetCursorPos                = user32.NewProc("GetCursorPos")
	procEnumWindows                 = user32.NewProc("EnumWindows")
	procGetWindowTextW               = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW         = user32.NewProc("GetWindowTextLengthW")
	procGetClassNameW                = user32.NewProc("GetClassNameW")
	procGetWindowLongPtrW            = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW            = user32.NewProc("SetWindowLongPtrW")
	procGetClassLongPtrW             = user32.NewProc("GetClassLongPtrW")
	procCallWindowProcW              = user32.NewProc("CallWindowProcW")
	procGetWindow                    = user32.NewProc("GetWindow")
	procGetWindowThreadProcessId     = user32.NewProc("GetWindowThreadProcessId")
	procSendMessageTimeoutW          = user32.NewProc("SendMessageTimeoutW")
	procPostMessageW                 = user32.NewProc("PostMessageW")
	procRegisterWindowMessageW       = user32.NewProc("RegisterWindowMessageW")
	procDestroyIcon                  = user32.NewProc("DestroyIcon")
	procGetIconInfo                  = user32.NewProc("GetIconInfo")
	procMonitorFromWindow            = user32.NewProc("MonitorFromWindow")
	procGetForegroundWindow          = user32.NewProc("GetForegroundWindow")
	procGetMonitorInfoW              = user32.NewProc("GetMonitorInfoW")
)

func GetForegroundWindow() HWND {
	r, _, _ := procGetForegroundWindow.Call()
	return HWND(r)
}

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

// PeekMessageW is a non-blocking poll for a pending message, used by
// components that piggyback on someone else's message loop (e.g. Wails')
// and cannot call the blocking GetMessageW themselves.
func PeekMessageW(msg *MSG, hwnd HWND, msgFilterMin, msgFilterMax, removeMsg uint32) bool {
	r, _, _ := procPeekMessageW.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax),
		uintptr(removeMsg),
	)
	return r != 0
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
// from every screen-capture pipeline (BitBlt, PrintWindow, WGC, ...).
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

func IsIconic(hwnd HWND) bool {
	r, _, _ := procIsIconic.Call(uintptr(hwnd))
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

// EnumWindowsProc is the callback signature expected by EnumWindows: return
// false to stop enumeration early, true to continue.
type EnumWindowsProc func(hwnd HWND) bool

// EnumWindows enumerates all top-level windows, invoking fn for each one
// until it returns false or every window has been visited.
func EnumWindows(fn EnumWindowsProc) {
	cb := windows.NewCallback(func(hwnd HWND, _ uintptr) uintptr {
		if fn(hwnd) {
			return 1
		}
		return 0
	})
	procEnumWindows.Call(cb, 0)
}

// GetWindowTextW returns the window's title bar text (empty if it has none).
func GetWindowTextW(hwnd HWND) string {
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}

// GetClassNameW returns the window class name of hwnd.
func GetClassNameW(hwnd HWND) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

// GetWindowLongPtrW reads an extended window attribute (GWL_STYLE,
// GWL_EXSTYLE, GWLP_WNDPROC, ...).
func GetWindowLongPtrW(hwnd HWND, index int32) uintptr {
	r, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(int64(index)))
	return r
}

// SetWindowLongPtrW sets an extended window attribute and returns the
// previous value, most commonly used here to subclass a window by
// overwriting GWLP_WNDPROC and chaining to the old proc via
// CallWindowProcW.
func SetWindowLongPtrW(hwnd HWND, index int32, newLong uintptr) uintptr {
	r, _, _ := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(int64(index)), newLong)
	return r
}

// GetClassLongPtrW reads a window class attribute (GCLP_HICON, ...).
func GetClassLongPtrW(hwnd HWND, index int32) uintptr {
	r, _, _ := procGetClassLongPtrW.Call(uintptr(hwnd), uintptr(int64(index)))
	return r
}

// CallWindowProcW invokes a (previous) window procedure directly, used when
// chaining a subclass proc to the original one.
func CallWindowProcW(prevWndProc uintptr, hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallWindowProcW.Call(prevWndProc, uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

// GetWindow retrieves a related window handle, e.g. GWOwner for the owner
// window (used to filter out owned utility/tool windows from the task list).
func GetWindow(hwnd HWND, cmd uint32) HWND {
	r, _, _ := procGetWindow.Call(uintptr(hwnd), uintptr(cmd))
	return HWND(r)
}

// GetWindowThreadProcessId returns the process ID that owns hwnd.
func GetWindowThreadProcessId(hwnd HWND) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

// SendMessageTimeoutW sends msg to hwnd but gives up after timeoutMs instead
// of blocking forever on a hung window - important when polling WM_GETICON
// across every open window on a timer tick.
func SendMessageTimeoutW(hwnd HWND, msg uint32, wParam, lParam uintptr, flags uint32, timeoutMs uint32) (result uintptr, ok bool) {
	var res uintptr
	r, _, _ := procSendMessageTimeoutW.Call(
		uintptr(hwnd), uintptr(msg), wParam, lParam,
		uintptr(flags), uintptr(timeoutMs), uintptr(unsafe.Pointer(&res)),
	)
	return res, r != 0
}

// PostMessageW posts msg to hwnd's queue without waiting for it to be
// processed (used to send WM_CLOSE to another process's window).
func PostMessageW(hwnd HWND, msg uint32, wParam, lParam uintptr) bool {
	r, _, _ := procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r != 0
}

// RegisterWindowMessageW allocates (or looks up) a systemwide message
// identifier for name, used for cross-process notifications like
// "TaskbarCreated" and "SHELLHOOK".
func RegisterWindowMessageW(name string) uint32 {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(ptr)))
	return uint32(r)
}

func DestroyIcon(hicon HICON) bool {
	r, _, _ := procDestroyIcon.Call(uintptr(hicon))
	return r != 0
}

// GetIconInfo retrieves the mask/color bitmaps that make up hicon. The
// caller owns (and must DeleteObject) the returned bitmaps.
func GetIconInfo(hicon HICON) (ICONINFO, bool) {
	var info ICONINFO
	r, _, _ := procGetIconInfo.Call(uintptr(hicon), uintptr(unsafe.Pointer(&info)))
	return info, r != 0
}

// MonitorFromWindow returns the handle of the display monitor with the
// largest overlap with hwnd; flags is typically MONITOR_DEFAULTTONEAREST.
func MonitorFromWindow(hwnd HWND, flags uint32) uintptr {
	r, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), uintptr(flags))
	return r
}

// GetMonitorInfoW returns the full monitor and work-area rects for the
// monitor handle hMonitor (e.g. from MonitorFromWindow).
func GetMonitorInfoW(hMonitor uintptr) (MONITORINFO, bool) {
	var info MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	r, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&info)))
	return info, r != 0
}

// PrimaryMonitorRect returns the full-screen rect (not work area) of the
// primary display, used by appbar.Register to compute the bar's edge
// placement independent of any already-reserved taskbar work area.
func PrimaryMonitorRect() (RECT, bool) {
	h := MonitorFromWindow(0, MonitorDefaultToPrimary)
	if h == 0 {
		return RECT{}, false
	}
	info, ok := GetMonitorInfoW(h)
	if !ok {
		return RECT{}, false
	}
	return info.RcMonitor, true
}
