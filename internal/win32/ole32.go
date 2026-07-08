package win32

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32 = windows.NewLazySystemDLL("ole32.dll")

	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
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

// GUID is defined in shell32.go; CLSIDs and IIDs use the same layout.

// GUIDFromString parses a GUID in the canonical registry form, with or
// without surrounding braces, e.g. "{ff48dba4-60ef-4201-aa87-54103eef594e}".
func GUIDFromString(s string) (GUID, error) {
	str := s
	if len(str) >= 2 && str[0] == '{' && str[len(str)-1] == '}' {
		str = str[1 : len(str)-1]
	}
	// Expected layout: 8-4-4-4-12 hex digits separated by hyphens.
	if len(str) != 36 || str[8] != '-' || str[13] != '-' || str[18] != '-' || str[23] != '-' {
		return GUID{}, fmt.Errorf("win32: invalid GUID %q", s)
	}

	var g GUID
	var err error
	parse := func(sub string, bits int) uint64 {
		if err != nil {
			return 0
		}
		var v uint64
		v, err = parseHex(sub, bits)
		return v
	}

	g.Data1 = uint32(parse(str[0:8], 32))
	g.Data2 = uint16(parse(str[9:13], 16))
	g.Data3 = uint16(parse(str[14:18], 16))

	// Data4 holds the last two groups (2 bytes then 6 bytes), stored as raw
	// bytes in order.
	hi := parse(str[19:23], 16)
	g.Data4[0] = byte(hi >> 8)
	g.Data4[1] = byte(hi)
	last := parse(str[24:36], 48)
	for i := 0; i < 6; i++ {
		g.Data4[2+i] = byte(last >> (uint(5-i) * 8))
	}
	if err != nil {
		return GUID{}, fmt.Errorf("win32: invalid GUID %q: %w", s, err)
	}
	return g, nil
}

func parseHex(s string, bits int) (uint64, error) {
	var v uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d uint64
		switch {
		case c >= '0' && c <= '9':
			d = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint64(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q", string(c))
		}
		v = v<<4 | d
	}
	_ = bits
	return v, nil
}

// CoInitializeEx initializes COM for the calling thread with the given
// concurrency model. S_FALSE (already initialized) is treated as success.
func CoInitializeEx(coInit uint32) error {
	r, _, _ := procCoInitializeEx.Call(0, uintptr(coInit))
	// S_OK (0) and S_FALSE (1) both mean COM is usable on this thread.
	if int32(r) < 0 {
		return fmt.Errorf("win32: CoInitializeEx failed: 0x%08x", uint32(r))
	}
	return nil
}

// CoUninitialize releases the COM library on the calling thread.
func CoUninitialize() {
	procCoUninitialize.Call()
}

// CoCreateInstance creates a single uninitialized COM object of the class
// clsid, returning the requested interface pointer iid.
func CoCreateInstance(clsid *GUID, clsctx uint32, iid *GUID) (uintptr, error) {
	var ptr uintptr
	r, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(clsid)),
		0,
		uintptr(clsctx),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&ptr)),
	)
	if int32(r) < 0 {
		return 0, fmt.Errorf("win32: CoCreateInstance failed: 0x%08x", uint32(r))
	}
	if ptr == 0 {
		return 0, fmt.Errorf("win32: CoCreateInstance returned nil interface")
	}
	return ptr, nil
}

// ComCall invokes the method at the given vtable slot index on a COM
// interface pointer. index is the zero-based position in the vtable; the
// first three slots (0,1,2) are always IUnknown's QueryInterface, AddRef and
// Release, so a first custom interface method is typically index 3.
//
// args must NOT include the `this` pointer; ComCall prepends it, matching the
// thiscall/stdcall convention that COM methods receive the interface pointer
// as their first argument.
func ComCall(this uintptr, index int, args ...uintptr) uintptr {
	// The vtable is the first pointer-sized field of the object; it points at
	// an array of function pointers.
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(uintptr(0))))

	full := make([]uintptr, 0, len(args)+1)
	full = append(full, this)
	full = append(full, args...)
	r, _, _ := syscall.SyscallN(fn, full...)
	return r
}

// ComRelease calls IUnknown::Release (vtable index 2) on a COM object.
func ComRelease(this uintptr) {
	if this != 0 {
		ComCall(this, 2)
	}
}

// Failed reports whether an HRESULT-style return value indicates failure.
func Failed(hr uintptr) bool { return int32(hr) < 0 }
