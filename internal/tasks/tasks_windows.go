//go:build windows

package tasks

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"

	"github.com/fourhundredfour/pinky/internal/applog"
	"github.com/fourhundredfour/pinky/internal/win32"
)

// Lister enumerates and filters top-level windows into the taskbar's
// listable set, extracting and caching icons along the way.
type Lister struct {
	selfHWND win32.HWND

	mu        sync.Mutex
	iconCache map[uintptr][]byte // keyed by HICON value, stores raw PNG bytes
}

// NewLister creates a Lister that always excludes selfHWND (pinky's own
// bar window) from its results.
func NewLister(selfHWND win32.HWND) *Lister {
	return &Lister{selfHWND: selfHWND, iconCache: make(map[uintptr][]byte)}
}

// List enumerates every currently listable top-level window.
func (l *Lister) List() []Window {
	foreground := win32.GetForegroundWindow()

	var out []Window
	win32.EnumWindows(func(hwnd win32.HWND) bool {
		if hwnd == l.selfHWND {
			return true
		}
		c := candidate{
			Title:    win32.GetWindowTextW(hwnd),
			Visible:  win32.IsWindowVisible(hwnd),
			Cloaked:  win32.DwmIsCloaked(hwnd),
			HasOwner: win32.GetWindow(hwnd, win32.GWOwner) != 0,
			ExStyle:  uint32(win32.GetWindowLongPtrW(hwnd, win32.GWLExStyle)),
		}
		if !shouldList(c) {
			return true
		}
		out = append(out, Window{
			ID:        strconv.FormatUint(uint64(hwnd), 10),
			Title:     c.Title,
			Icon:      l.iconFor(hwnd),
			Active:    hwnd == foreground,
			Minimized: win32.IsIconic(hwnd),
		})
		return true
	})
	return out
}

var (
	pixelBufferPool = sync.Pool{
		New: func() any {
			return make([]byte, 262144) // 256KB is enough for up to 256x256 icons
		},
	}
	bytesBufferPool = sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}
)

// GetIconPNGBytes returns the cached raw PNG bytes for the given HICON.
func (l *Lister) GetIconPNGBytes(hicon uintptr) ([]byte, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	bytes, ok := l.iconCache[hicon]
	return bytes, ok
}

// iconFor returns a "/icon?id=<hwnd>&hicon=<hicon_hex>" URL for hwnd's icon,
// caching by the underlying HICON value so repeated polls of an unchanged
// window are free.
func (l *Lister) iconFor(hwnd win32.HWND) string {
	hicon := windowIcon(hwnd)
	if hicon == 0 {
		return ""
	}

	l.mu.Lock()
	_, exists := l.iconCache[uintptr(hicon)]
	l.mu.Unlock()

	if !exists {
		pngBytes, err := iconToPNGBytes(hicon)
		if err == nil && len(pngBytes) > 0 {
			l.mu.Lock()
			l.iconCache[uintptr(hicon)] = pngBytes
			l.mu.Unlock()
		} else {
			return ""
		}
	}

	return fmt.Sprintf("/icon?id=%d&hicon=%x", hwnd, hicon)
}

// windowIcon asks the window for its icon (large, then small), falling
// back to the window class's icon. The returned HICON is owned by the
// target window/class - callers must not DestroyIcon it.
func windowIcon(hwnd win32.HWND) win32.HICON {
	const timeoutMs = 100
	for _, size := range []uintptr{win32.ICONBig, win32.ICONSmall2, win32.ICONSmall} {
		if r, ok := win32.SendMessageTimeoutW(hwnd, win32.WMGetIcon, size, 0, win32.SMTOAbortIfHung, timeoutMs); ok && r != 0 {
			return win32.HICON(r)
		}
	}
	if r := win32.GetClassLongPtrW(hwnd, win32.GCLPHIcon); r != 0 {
		return win32.HICON(r)
	}
	if r := win32.GetClassLongPtrW(hwnd, win32.GCLPHIconSm); r != 0 {
		return win32.HICON(r)
	}
	return 0
}

// iconToPNGBytes rasterizes hicon (via GDI) into a PNG using a zero-allocation pipeline.
func iconToPNGBytes(hicon win32.HICON) ([]byte, error) {
	info, ok := win32.GetIconInfo(hicon)
	if !ok {
		return nil, fmt.Errorf("tasks: GetIconInfo failed")
	}
	defer win32.DeleteObject(win32.HGDIOBJ(info.HbmMask))
	defer win32.DeleteObject(win32.HGDIOBJ(info.HbmColor))

	if info.HbmColor == 0 {
		return nil, fmt.Errorf("tasks: monochrome icons not supported")
	}

	width, height, ok := win32.GetBitmapDimensions(info.HbmColor)
	if !ok || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("tasks: GetBitmapDimensions failed")
	}

	hdc := win32.GetDC(0)
	defer win32.ReleaseDC(0, hdc)

	neededSize := int(width) * int(height) * 4
	var pixels []byte
	var pooledPixels []byte
	if neededSize <= 262144 {
		pooledPixels = pixelBufferPool.Get().([]byte)
		pixels = pooledPixels[:neededSize]
	} else {
		pixels = make([]byte, neededSize)
	}
	if pooledPixels != nil {
		defer pixelBufferPool.Put(pooledPixels)
	}

	ok = win32.GetDIBitsRGBABuf(hdc, info.HbmColor, width, height, pixels)
	if !ok {
		return nil, fmt.Errorf("tasks: GetDIBits failed")
	}

	// In-place BGRA-to-RGBA conversion
	hasAlpha := false
	for i := 0; i < len(pixels); i += 4 {
		b, r := pixels[i], pixels[i+2]
		pixels[i] = r   // Red
		pixels[i+2] = b // Blue
		if pixels[i+3] != 0 {
			hasAlpha = true
		}
	}
	if !hasAlpha {
		for i := 3; i < len(pixels); i += 4 {
			pixels[i] = 255
		}
	}

	img := image.NRGBA{
		Rect:   image.Rect(0, 0, int(width), int(height)),
		Stride: int(width) * 4,
		Pix:    pixels,
	}

	buf := bytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bytesBufferPool.Put(buf)

	if err := png.Encode(buf, &img); err != nil {
		return nil, err
	}

	pngBytes := make([]byte, buf.Len())
	copy(pngBytes, buf.Bytes())
	return pngBytes, nil
}

// Focus brings hwnd to the foreground, restoring it first if minimized.
func Focus(id string) error {
	hwnd, err := parseHWND(id)
	if err != nil {
		return err
	}
	if win32.IsIconic(hwnd) {
		win32.ShowWindow(hwnd, win32.SWRestore)
	}
	win32.SetForegroundWindow(hwnd)
	return nil
}

// Minimize minimizes hwnd.
func Minimize(id string) error {
	hwnd, err := parseHWND(id)
	if err != nil {
		return err
	}
	win32.ShowWindow(hwnd, win32.SWMinimize)
	return nil
}

// Close asks hwnd to close via WM_CLOSE, the same way clicking its title
// bar's close button would.
func Close(id string) error {
	hwnd, err := parseHWND(id)
	if err != nil {
		return err
	}
	win32.PostMessageW(hwnd, win32.WMClose, 0, 0)
	return nil
}

func parseHWND(id string) (win32.HWND, error) {
	v, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("tasks: invalid window id %q: %w", id, err)
	}
	return win32.HWND(v), nil
}

// Watcher subclasses hwnd to receive shell-hook notifications (window
// created/destroyed/activated/...), falling back to a plain poll timer as
// a safety net, and calls onChange whenever the open-window set might have
// changed.
type Watcher struct {
	hwnd         win32.HWND
	shellHookMsg uint32
	prevWndProc  uintptr

	stop chan struct{}

	mu       sync.Mutex
	timer    *time.Timer
	onChange func()
	closed   atomic.Bool
}

// NewWatcher installs the shell hook and starts the fallback poll ticker.
// hwnd must be a top-level window belonging to this process (pinky's own
// bar window).
func NewWatcher(hwnd win32.HWND, pollInterval time.Duration, onChange func()) *Watcher {
	w := &Watcher{
		hwnd:     hwnd,
		stop:     make(chan struct{}),
		onChange: onChange,
	}
	w.shellHookMsg = win32.RegisterWindowMessageW("SHELLHOOK")
	win32.RegisterShellHookWindow(hwnd)

	// Fetch the previous window procedure BEFORE subclassing to avoid race conditions
	// where messages are dispatched to wndProc before SetWindowLongPtrW returns.
	w.prevWndProc = win32.GetWindowLongPtrW(hwnd, win32.GWLPWndProc)

	proc := windows.NewCallback(func(h win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		defer applog.RecoverAndLog("tasks-watcher-wndproc")
		if w.shellHookMsg != 0 && msg == w.shellHookMsg {
			switch wParam & 0x7FFF {
			case win32.HSHELLWindowCreated, win32.HSHELLWindowDestroyed,
				win32.HSHELLWindowActivated, win32.HSHELLRedraw:
				w.trigger()
			}
		}
		if w.prevWndProc == 0 {
			return win32.DefWindowProcW(h, msg, wParam, lParam)
		}
		return win32.CallWindowProcW(w.prevWndProc, h, msg, wParam, lParam)
	})
	win32.SetWindowLongPtrW(hwnd, win32.GWLPWndProc, proc)

	if pollInterval > 0 {
		applog.Go("tasks-watcher-poll", func() {
			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					w.trigger()
				case <-w.stop:
					return
				}
			}
		})
	}
	return w
}

func (w *Watcher) trigger() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed.Load() {
		return
	}

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(50*time.Millisecond, func() {
		defer applog.RecoverAndLog("tasks-watcher-debounce")
		if w.closed.Load() {
			return
		}
		w.onChange()
	})
}

// Close stops the poll ticker, deregisters the shell hook and restores the
// original window procedure.
func (w *Watcher) Close() {
	w.closed.Store(true)
	close(w.stop)
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	win32.DeregisterShellHookWindow(w.hwnd)
	if w.prevWndProc != 0 {
		win32.SetWindowLongPtrW(w.hwnd, win32.GWLPWndProc, w.prevWndProc)
		w.prevWndProc = 0
	}
}
