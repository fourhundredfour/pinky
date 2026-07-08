//go:build windows

package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procGetObjectW         = gdi32.NewProc("GetObjectW")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
)

func CreateCompatibleDC(hdc HDC) HDC {
	r, _, _ := procCreateCompatibleDC.Call(uintptr(hdc))
	return HDC(r)
}

// CreateDIBSection allocates a top-down, 32bpp BGRA DIB of the given size and
// returns both the bitmap handle and a pointer to its pixel buffer, which the
// caller can write to directly.
func CreateDIBSection(hdc HDC, width, height int32) (HBITMAP, unsafe.Pointer, error) {
	header := BITMAPINFOHEADER{
		Width:       width,
		Height:      -height, // negative height => top-down DIB
		Planes:      1,
		BitCount:    32,
		Compression: BIRGB,
	}
	header.Size = uint32(unsafe.Sizeof(header))

	var bits unsafe.Pointer
	r, _, err := procCreateDIBSection.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&header)),
		uintptr(DIBRGBColors),
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if r == 0 {
		return 0, nil, err
	}
	return HBITMAP(r), bits, nil
}

func SelectObject(hdc HDC, obj HGDIOBJ) HGDIOBJ {
	r, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(obj))
	return HGDIOBJ(r)
}

func DeleteObject(obj HGDIOBJ) bool {
	r, _, _ := procDeleteObject.Call(uintptr(obj))
	return r != 0
}

func DeleteDC(hdc HDC) bool {
	r, _, _ := procDeleteDC.Call(uintptr(hdc))
	return r != 0
}

func BitBlt(dstDC HDC, x, y, w, h int32, srcDC HDC, srcX, srcY int32, rop uint32) bool {
	r, _, _ := procBitBlt.Call(
		uintptr(dstDC), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(srcDC), uintptr(srcX), uintptr(srcY), uintptr(rop),
	)
	return r != 0
}

// GetBitmapDimensions returns the width/height of a bitmap via GetObjectW.
func GetBitmapDimensions(bmp HBITMAP) (width, height int32, ok bool) {
	var bm BITMAP
	r, _, _ := procGetObjectW.Call(uintptr(bmp), unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))
	if r == 0 {
		return 0, 0, false
	}
	return bm.Width, bm.Height, true
}

// GetDIBitsRGBA reads bmp's pixels into a top-down 32bpp BGRA buffer of size
// width*height*4, using hdc as the reference device context.
func GetDIBitsRGBA(hdc HDC, bmp HBITMAP, width, height int32) ([]byte, bool) {
	buf := make([]byte, int(width)*int(height)*4)
	ok := GetDIBitsRGBABuf(hdc, bmp, width, height, buf)
	return buf, ok
}

// GetDIBitsRGBABuf reads bmp's pixels into a pre-allocated top-down 32bpp BGRA buffer,
// using hdc as the reference device context. The buffer must be at least width*height*4 bytes.
func GetDIBitsRGBABuf(hdc HDC, bmp HBITMAP, width, height int32, buf []byte) bool {
	if len(buf) < int(width)*int(height)*4 {
		return false
	}
	info := BITMAPINFO{Header: BITMAPINFOHEADER{
		Width:       width,
		Height:      -height,
		Planes:      1,
		BitCount:    32,
		Compression: BIRGB,
	}}
	info.Header.Size = uint32(unsafe.Sizeof(info.Header))

	r, _, _ := procGetDIBits.Call(
		uintptr(hdc), uintptr(bmp), 0, uintptr(height),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&info)),
		uintptr(DIBRGBColors),
	)
	return r != 0
}
