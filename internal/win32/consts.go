//go:build windows

package win32

// Window styles / extended styles.
const (
	WSPopup    = 0x80000000
	WSVisible  = 0x10000000
	WSDisabled = 0x08000000

	WSExLayered     = 0x00080000
	WSExTransparent = 0x00000020
	WSExTopMost     = 0x00000008
	WSExToolWindow  = 0x00000080
	WSExAppWindow   = 0x00040000
	WSExNoActivate  = 0x08000000

	WSMinimize = 0x20000000
)

// GetWindowLong(Ptr) indices.
const (
	GWLStyle   = -16
	GWLExStyle = -20
	GWLPWndProc = -4
)

// GetClassLongPtr indices.
const (
	GCLPHIcon   = -14
	GCLPHIconSm = -34
)

// GetWindow commands.
const (
	GWOwner = 4
)

// ShowWindow commands.
const (
	SWHide           = 0
	SWShowNormal     = 1
	SWShowMinimized  = 2
	SWShowMaximized  = 3
	SWShowNoActivate = 4
	SWShow           = 5
	SWMinimize       = 6
	SWRestore        = 9
)

// SetWindowPos flags.
const (
	SWPNoActivate  = 0x0010
	SWPShowWindow  = 0x0040
	SWPHideWindow  = 0x0080
	SWPNoSize      = 0x0001
	SWPNoMove      = 0x0002
	SWPNoZOrder    = 0x0004
	SWPFrameChanged = 0x0020
)

// HWND_TOPMOST / HWND_NOTOPMOST as defined by Win32 (cast of -1 / -2).
var (
	HWNDTopMost   = HWND(^uintptr(0))
	HWNDNoTopMost = HWND(^uintptr(1))
)

// Window messages.
const (
	WMDestroy       = 0x0002
	WMSize          = 0x0005
	WMClose         = 0x0010
	WMTimer         = 0x0113
	WMNCDestroy     = 0x0082
	WMApp           = 0x8000
	WMLButtonUp     = 0x0202
	WMRButtonUp     = 0x0205
	WMContextMenu   = 0x007B
	WMSettingChange = 0x001A
	WMDisplayChange = 0x007E
	WMDPIChanged    = 0x02E0
	WMGetIcon       = 0x007F
)

// GetIcon / SendMessage WM_GETICON parameters.
const (
	ICONSmall  = 0
	ICONBig    = 1
	ICONSmall2 = 2
)

// SendMessageTimeout flags.
const (
	SMTOAbortIfHung = 0x0002
	SMTONormal      = 0x0000
)

// Popup menu flags/commands.
const (
	MFString       = 0x00000000
	MFSeparator    = 0x00000800
	TPMRightButton = 0x0002
	TPMReturnCmd   = 0x0100
	TPMNonNotify   = 0x0080
)

// PeekMessage flags.
const (
	PMRemove = 0x0001
)

// UpdateLayeredWindow flags.
const (
	ULWAlpha = 0x00000002
)

// BLENDFUNCTION constants.
const (
	ACSrcOver  = 0x00
	ACSrcAlpha = 0x01
)

// Bitmap constants.
const (
	BIRGB        = 0
	DIBRGBColors = 0
)

// BitBlt raster operation.
const (
	SrcCopy = 0x00CC0020
)

// SetWindowDisplayAffinity values.
const (
	WDANone               = 0x00000000
	WDAMonitor            = 0x00000001
	WDAExcludeFromCapture = 0x00000011
)

// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, expressed the same way the
// Windows SDK does: ((DPI_AWARENESS_CONTEXT)-4).
var DPIAwarenessContextPerMonitorAwareV2 = ^uintptr(3)

// MonitorFromWindow/MonitorFromRect flags.
const (
	MonitorDefaultToNull      = 0x00000000
	MonitorDefaultToPrimary   = 0x00000001
	MonitorDefaultToNearest   = 0x00000002
)

// Standard cursor/icon resource identifiers for LoadCursorW/LoadIconW.
const (
	IDCArrow       = 32512
	IDIApplication = 32512
)

// SHAppBarMessage message identifiers (dwMessage).
const (
	ABMNew             = 0x00000000
	ABMRemove          = 0x00000001
	ABMQueryPos        = 0x00000002
	ABMSetPos          = 0x00000003
	ABMGetState        = 0x00000004
	ABMGetTaskbarPos   = 0x00000005
	ABMActivate        = 0x00000006
	ABMGetAutoHideBar  = 0x00000007
	ABMSetAutoHideBar  = 0x00000008
	ABMWindowPosChanged = 0x00000009
	ABMSetState        = 0x0000000A
)

// Appbar screen edges (uEdge).
const (
	ABELeft   = 0
	ABETop    = 1
	ABERight  = 2
	ABEBottom = 3
)

// Appbar state flags (ABM_GETSTATE / ABM_SETSTATE, lParam).
const (
	ABSAutoHide    = 0x0000001
	ABSAlwaysOnTop = 0x0000002
)

// Appbar notification codes, delivered via the app-defined
// uCallbackMessage registered with ABM_NEW.
const (
	ABNStateChange    = 0x0000000
	ABNPosChanged     = 0x0000001
	ABNFullScreenApp  = 0x0000002
	ABNWindowArrange  = 0x0000003
)

// DWM window attributes.
const (
	DWMWACloaked            = 14
	DWMWAWindowCornerPref   = 33
)

// DWM_WINDOW_CORNER_PREFERENCE values.
const (
	DWMWCPDefault    = 0
	DWMWCPDoNotRound = 1
	DWMWCPRound      = 2
	DWMWCPRoundSmall = 3
)

// Shell hook notification codes (delivered via the registered "SHELLHOOK"
// message after RegisterShellHookWindow).
const (
	HSHELLWindowCreated       = 1
	HSHELLWindowDestroyed     = 2
	HSHELLActivateShellWindow = 3
	HSHELLWindowActivated     = 4
	HSHELLRedraw              = 6
	HSHELLFlash               = 0x8006
	HSHELLRudeAppActivated    = 0x8004
)

// COINIT values for CoInitializeEx.
const (
	COINITApartmentThreaded = 0x2
	COINITMultiThreaded     = 0x0
)

// CLSCTX values for CoCreateInstance.
const (
	CLSCTXInprocServer = 0x1
	CLSCTXLocalServer  = 0x4
	CLSCTXAll          = CLSCTXInprocServer | 0x2 | CLSCTXLocalServer | 0x10
)

// VARIANT type tags used by the tiny VARIANT helper in ole32.go.
const (
	VTEmpty = 0
	VTI4    = 3
	VTBool  = 11
)
