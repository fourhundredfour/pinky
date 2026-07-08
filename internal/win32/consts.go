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
	WSExNoActivate  = 0x08000000
)

// ShowWindow commands.
const (
	SWHide           = 0
	SWShowNoActivate = 4
)

// SetWindowPos flags.
const (
	SWPNoActivate = 0x0010
	SWPShowWindow = 0x0040
	SWPNoSize     = 0x0001
	SWPNoMove     = 0x0002
)

// HWND_TOPMOST as defined by Win32 (cast of -1).
var HWNDTopMost = HWND(^uintptr(0))

// Window messages.
const (
	WMDestroy     = 0x0002
	WMTimer       = 0x0113
	WMNCDestroy   = 0x0082
	WMApp         = 0x8000
	WMLButtonUp   = 0x0202
	WMRButtonUp   = 0x0205
	WMContextMenu = 0x007B
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

// Standard cursor/icon resource identifiers for LoadCursorW/LoadIconW.
const (
	IDCArrow       = 32512
	IDIApplication = 32512
)
