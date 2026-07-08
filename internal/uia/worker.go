//go:build windows

package uia

import (
	"runtime"
	"sync"
	"time"

	"github.com/fourhundredfour/pinky/internal/win32"
)

// DefaultPollInterval is how often the taskbar icon rectangles are
// re-enumerated. UIA queries are comparatively expensive and can briefly
// stall if Explorer is busy, so this is deliberately slow relative to the
// render tick; the render loop only ever reads the last published snapshot.
const DefaultPollInterval = 300 * time.Millisecond

// Worker polls taskbar icon rectangles on a dedicated COM-initialized OS
// thread and publishes them behind a mutex. It implements the rectangle
// lookup the render loop needs without ever blocking that loop: a slow or
// hung UIA call only stalls this background goroutine, leaving the last
// known rectangles in place.
type Worker struct {
	hwndSource func() []win32.HWND
	interval   time.Duration

	mu    sync.RWMutex
	rects map[win32.HWND][]win32.RECT

	stop chan struct{}
	done chan struct{}
}

// NewWorker creates a worker. hwndSource is called each poll to get the
// current set of taskbar windows (primary + any secondary monitors), so the
// worker automatically tracks monitors coming and going.
func NewWorker(hwndSource func() []win32.HWND) *Worker {
	return &Worker{
		hwndSource: hwndSource,
		interval:   DefaultPollInterval,
		rects:      map[win32.HWND][]win32.RECT{},
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Start launches the background polling goroutine.
func (w *Worker) Start() { go w.run() }

// Stop signals the goroutine to exit and waits for it to finish (releasing
// its COM resources).
func (w *Worker) Stop() {
	select {
	case <-w.stop:
		// already stopped
	default:
		close(w.stop)
	}
	<-w.done
}

// RectsFor returns the most recently enumerated button rectangles for hwnd,
// or nil if none are known yet. The returned slice must not be mutated.
func (w *Worker) RectsFor(hwnd win32.HWND) []win32.RECT {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rects[hwnd]
}

func (w *Worker) run() {
	defer close(w.done)

	// COM and the IUIAutomation instance are thread-affine; pin this
	// goroutine to one OS thread for its whole life.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := win32.CoInitializeEx(win32.COINITMultiThreaded); err != nil {
		return
	}
	defer win32.CoUninitialize()

	client, _ := NewClient()
	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	w.pollOnce(client)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			if client == nil {
				client, _ = NewClient()
			}
			if client != nil && !w.pollOnce(client) {
				// A COM error likely left the client in a bad state; drop it
				// and rebuild on the next tick.
				client.Close()
				client = nil
			}
		}
	}
}

// pollOnce enumerates rects for every current taskbar window and atomically
// swaps in the new snapshot. It returns false if any enumeration failed, so
// the caller can decide to recreate the client.
func (w *Worker) pollOnce(client *Client) bool {
	if client == nil {
		return false
	}
	hwnds := w.hwndSource()
	result := make(map[win32.HWND][]win32.RECT, len(hwnds))
	ok := true
	for _, h := range hwnds {
		rects, err := client.EnumButtonRects(h)
		if err != nil {
			ok = false
			continue
		}
		result[h] = rects
	}

	w.mu.Lock()
	w.rects = result
	w.mu.Unlock()
	return ok
}
