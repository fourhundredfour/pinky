// Package capture grabs a screen region (the taskbar rect) into an in-memory
// pixel buffer via GDI BitBlt.
//
// BitBlt against the screen DC is used deliberately instead of PrintWindow:
// PrintWindow frequently returns a black image for the Windows 11 taskbar
// because it is rendered via XAML/DirectComposition, whereas BitBlt reads
// straight from the DWM-composited framebuffer and picks it up correctly.
package capture

import (
	"fmt"
	"unsafe"

	"github.com/fourhundredfour/pinky/internal/win32"
)

// Frame is a top-down 32bpp BGRA pixel buffer.
type Frame struct {
	Width, Height int32
	// Pix holds Height rows of Width*4 bytes each, in B,G,R,A byte order,
	// matching the native layout of a GDI 32bpp DIB section.
	Pix []byte
}

func (f *Frame) Stride() int32 { return f.Width * 4 }

// Capturer owns the GDI resources used to repeatedly capture arbitrary
// screen rectangles. It keeps a single DIB section alive and only
// reallocates it when the requested size changes, since the taskbar rect is
// typically stable frame to frame.
type Capturer struct {
	screenDC win32.HDC
	memDC    win32.HDC
	bitmap   win32.HBITMAP
	oldBmp   win32.HGDIOBJ
	bits     unsafe.Pointer
	width    int32
	height   int32
}

// New creates a Capturer. Close must be called to release GDI resources.
func New() (*Capturer, error) {
	screenDC := win32.GetDC(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("capture: GetDC(desktop) failed")
	}
	memDC := win32.CreateCompatibleDC(screenDC)
	if memDC == 0 {
		win32.ReleaseDC(0, screenDC)
		return nil, fmt.Errorf("capture: CreateCompatibleDC failed")
	}
	return &Capturer{screenDC: screenDC, memDC: memDC}, nil
}

// Close releases the GDI resources held by the Capturer.
func (c *Capturer) Close() {
	if c.bitmap != 0 {
		win32.SelectObject(c.memDC, c.oldBmp)
		win32.DeleteObject(win32.HGDIOBJ(c.bitmap))
		c.bitmap = 0
	}
	if c.memDC != 0 {
		win32.DeleteDC(c.memDC)
		c.memDC = 0
	}
	if c.screenDC != 0 {
		win32.ReleaseDC(0, c.screenDC)
		c.screenDC = 0
	}
}

func (c *Capturer) ensureSize(width, height int32) error {
	if c.bitmap != 0 && c.width == width && c.height == height {
		return nil
	}
	if c.bitmap != 0 {
		win32.SelectObject(c.memDC, c.oldBmp)
		win32.DeleteObject(win32.HGDIOBJ(c.bitmap))
		c.bitmap = 0
	}

	bmp, bits, err := win32.CreateDIBSection(c.screenDC, width, height)
	if err != nil {
		return fmt.Errorf("capture: CreateDIBSection: %w", err)
	}
	c.bitmap = bmp
	c.bits = bits
	c.width = width
	c.height = height
	c.oldBmp = win32.SelectObject(c.memDC, win32.HGDIOBJ(bmp))
	return nil
}

// Capture grabs the given screen rectangle and returns a copy of its pixels.
// A copy is returned (rather than a view into the DIB) so callers can freely
// mutate it (e.g. to apply a blend) without racing the next capture.
func (c *Capturer) Capture(rect win32.RECT) (*Frame, error) {
	width, height := rect.Width(), rect.Height()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("capture: empty rect %+v", rect)
	}
	if err := c.ensureSize(width, height); err != nil {
		return nil, err
	}

	if !win32.BitBlt(c.memDC, 0, 0, width, height, c.screenDC, rect.Left, rect.Top, win32.SrcCopy) {
		return nil, fmt.Errorf("capture: BitBlt failed")
	}

	n := int(width) * int(height) * 4
	pix := make([]byte, n)
	src := unsafe.Slice((*byte)(c.bits), n)
	copy(pix, src)

	return &Frame{Width: width, Height: height, Pix: pix}, nil
}
