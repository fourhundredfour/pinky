package win32

import "unsafe"

import "golang.org/x/sys/windows"

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
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
