//go:build windows

// Package uia is a tiny, purpose-built UI Automation client used to locate
// the bounding rectangles of the icon/button cells on the Windows 11
// taskbar. The Win11 taskbar is a XAML surface, so the legacy toolbar rect
// messages no longer work and UI Automation is the supported way to read
// per-icon positions.
//
// Only the handful of COM methods pinky needs are bound, directly against
// the interface vtables (no cgo). Method indices below are the zero-based
// vtable slots from the Windows SDK (UIAutomationClient.h); the first three
// slots of every interface are IUnknown's QueryInterface/AddRef/Release.
package uia

import (
	"fmt"
	"unsafe"

	"github.com/fourhundredfour/pinky/internal/win32"
)

var (
	clsidCUIAutomation = mustGUID("{ff48dba4-60ef-4201-aa87-54103eef594e}")
	iidIUIAutomation   = mustGUID("{30cbe57d-d9d0-452a-ab13-7ac5ac4825ee}")
)

// UIA constants.
const (
	treeScopeDescendants     = 0x4
	uiaControlTypePropertyID = 30003
	uiaButtonControlTypeID   = 50000
	vtI4                     = 3
)

// Vtable slot indices.
const (
	idxAutomationElementFromHandle       = 6
	idxAutomationCreatePropertyCondition = 23

	idxElementFindAll           = 6
	idxElementBoundingRectangle = 44

	idxArrayLength     = 3
	idxArrayGetElement = 4
)

// variant is a minimal VARIANT holding a VT_I4 value. On x64 a 16-byte
// VARIANT passed "by value" is actually passed by reference per the calling
// convention, so we hand CreatePropertyCondition a pointer to this struct.
type variant struct {
	vt   uint16
	pad1 uint16
	pad2 uint16
	pad3 uint16
	val  int64 // union field; VT_I4 uses the low 4 bytes (little-endian)
}

// Client wraps an IUIAutomation instance plus a cached "control type is
// Button" condition. It is NOT safe for concurrent use and must be created
// and used on a single COM-initialized OS thread (see Worker).
type Client struct {
	automation uintptr
	buttonCond uintptr
}

// NewClient creates the IUIAutomation COM object. COM must already be
// initialized on the calling thread.
func NewClient() (*Client, error) {
	p, err := win32.CoCreateInstance(&clsidCUIAutomation, win32.CLSCTXInprocServer, &iidIUIAutomation)
	if err != nil {
		return nil, fmt.Errorf("uia: create IUIAutomation: %w", err)
	}
	c := &Client{automation: p}

	cond, err := c.createButtonCondition()
	if err != nil {
		c.Close()
		return nil, err
	}
	c.buttonCond = cond
	return c, nil
}

func (c *Client) createButtonCondition() (uintptr, error) {
	v := variant{vt: vtI4, val: int64(uiaButtonControlTypeID)}
	var cond uintptr
	hr := win32.ComCall(c.automation, idxAutomationCreatePropertyCondition,
		uintptr(uiaControlTypePropertyID),
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&cond)),
	)
	if win32.Failed(hr) || cond == 0 {
		return 0, fmt.Errorf("uia: CreatePropertyCondition failed: 0x%08x", uint32(hr))
	}
	return cond, nil
}

// Close releases the COM objects held by the client.
func (c *Client) Close() {
	if c.buttonCond != 0 {
		win32.ComRelease(c.buttonCond)
		c.buttonCond = 0
	}
	if c.automation != 0 {
		win32.ComRelease(c.automation)
		c.automation = 0
	}
}

// EnumButtonRects returns the screen-space bounding rectangles of every
// button-type element under the given taskbar window (app icons, Start,
// search, widgets, and system tray icons are all button controls). Rects are
// in physical pixels, matching pinky's DPI-aware capture coordinates.
func (c *Client) EnumButtonRects(hwnd win32.HWND) ([]win32.RECT, error) {
	var element uintptr
	hr := win32.ComCall(c.automation, idxAutomationElementFromHandle,
		uintptr(hwnd), uintptr(unsafe.Pointer(&element)))
	if win32.Failed(hr) || element == 0 {
		return nil, fmt.Errorf("uia: ElementFromHandle failed: 0x%08x", uint32(hr))
	}
	defer win32.ComRelease(element)

	var array uintptr
	hr = win32.ComCall(element, idxElementFindAll,
		uintptr(treeScopeDescendants), c.buttonCond, uintptr(unsafe.Pointer(&array)))
	if win32.Failed(hr) || array == 0 {
		return nil, fmt.Errorf("uia: FindAll failed: 0x%08x", uint32(hr))
	}
	defer win32.ComRelease(array)

	var length int32
	hr = win32.ComCall(array, idxArrayLength, uintptr(unsafe.Pointer(&length)))
	if win32.Failed(hr) {
		return nil, fmt.Errorf("uia: get_Length failed: 0x%08x", uint32(hr))
	}

	rects := make([]win32.RECT, 0, length)
	for i := int32(0); i < length; i++ {
		var el uintptr
		hr = win32.ComCall(array, idxArrayGetElement, uintptr(i), uintptr(unsafe.Pointer(&el)))
		if win32.Failed(hr) || el == 0 {
			continue
		}
		var r win32.RECT
		hr = win32.ComCall(el, idxElementBoundingRectangle, uintptr(unsafe.Pointer(&r)))
		win32.ComRelease(el)
		if win32.Failed(hr) {
			continue
		}
		if r.Right > r.Left && r.Bottom > r.Top {
			rects = append(rects, r)
		}
	}
	return rects, nil
}

func mustGUID(s string) win32.GUID {
	g, err := win32.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}
