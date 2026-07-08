//go:build windows

package indicators

import (
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"github.com/fourhundredfour/pinky/internal/win32"
)

// Well-known CoreAudio CLSID/IIDs (mmdeviceapi.h / endpointvolume.h).
var (
	clsidMMDeviceEnumerator = win32.MustGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidIMMDeviceEnumerator  = win32.MustGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidIAudioEndpointVolume = win32.MustGUID("{5CDF2C82-841E-4546-9722-0CF74078229A}")
)

// EDataFlow / ERole values used by GetDefaultAudioEndpoint.
const (
	eRender  = 0
	eConsole = 0
)

// Vtable slot indices (zero-based, IUnknown occupies 0-2).
const (
	idxEnumGetDefaultAudioEndpoint = 4 // IMMDeviceEnumerator
	idxDeviceActivate              = 3 // IMMDevice
	idxAEVSetMasterVolumeLevel     = 6 // IAudioEndpointVolume (unused, kept for reference)
	idxAEVSetVolumeScalar          = 7
	idxAEVGetVolumeScalar          = 9
	idxAEVSetMute                  = 14
	idxAEVGetMute                  = 15
)

// Volume is the payload sent to the frontend for the volume indicator.
type Volume struct {
	Level float64 `json:"level"`
	Muted bool    `json:"muted"`
}

// volumeController owns a single OS thread that COM is initialized on and
// through which every IAudioEndpointVolume call is funneled - required
// because apartment-threaded COM objects are thread-affine and Go
// goroutines otherwise migrate between OS threads freely.
type volumeController struct {
	reqCh    chan func()
	endpoint uintptr
}

func newVolumeController() *volumeController {
	vc := &volumeController{reqCh: make(chan func(), 4)}
	go vc.run()
	return vc
}

func (vc *volumeController) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := win32.CoInitializeEx(win32.COINITApartmentThreaded); err != nil {
		// Nothing else can run without COM; drain requests with an error
		// forever rather than leaving callers blocked.
		for fn := range vc.reqCh {
			fn()
		}
		return
	}
	defer win32.CoUninitialize()
	defer vc.release()

	for fn := range vc.reqCh {
		fn()
	}
}

func (vc *volumeController) release() {
	if vc.endpoint != 0 {
		win32.ComRelease(vc.endpoint)
		vc.endpoint = 0
	}
}

// do runs fn on the COM thread and blocks until it completes.
func (vc *volumeController) do(fn func()) {
	done := make(chan struct{})
	vc.reqCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// ensureEndpoint lazily activates the default render endpoint's
// IAudioEndpointVolume interface. Must only be called from the COM thread.
func (vc *volumeController) ensureEndpoint() error {
	if vc.endpoint != 0 {
		return nil
	}
	enumerator, err := win32.CoCreateInstance(&clsidMMDeviceEnumerator, win32.CLSCTXInprocServer, &iidIMMDeviceEnumerator)
	if err != nil {
		return err
	}
	defer win32.ComRelease(enumerator)

	var device uintptr
	hr := win32.ComCall(enumerator, idxEnumGetDefaultAudioEndpoint,
		uintptr(eRender), uintptr(eConsole), uintptr(unsafe.Pointer(&device)))
	if win32.Failed(hr) || device == 0 {
		return fmt.Errorf("indicators: GetDefaultAudioEndpoint failed: 0x%08x", uint32(hr))
	}
	defer win32.ComRelease(device)

	var endpoint uintptr
	hr = win32.ComCall(device, idxDeviceActivate,
		uintptr(unsafe.Pointer(&iidIAudioEndpointVolume)), uintptr(win32.CLSCTXInprocServer), 0,
		uintptr(unsafe.Pointer(&endpoint)))
	if win32.Failed(hr) || endpoint == 0 {
		return fmt.Errorf("indicators: IMMDevice.Activate(IAudioEndpointVolume) failed: 0x%08x", uint32(hr))
	}
	vc.endpoint = endpoint
	return nil
}

// Get reads the current master volume level (0.0-1.0) and mute state.
func (vc *volumeController) Get() (Volume, error) {
	var result Volume
	var outErr error
	vc.do(func() {
		if err := vc.ensureEndpoint(); err != nil {
			outErr = err
			return
		}
		var level float32
		win32.ComCall(vc.endpoint, idxAEVGetVolumeScalar, uintptr(unsafe.Pointer(&level)))
		var muted int32
		win32.ComCall(vc.endpoint, idxAEVGetMute, uintptr(unsafe.Pointer(&muted)))
		result = Volume{Level: float64(level), Muted: muted != 0}
	})
	return result, outErr
}

// Set changes the master volume level (clamped to [0,1]).
func (vc *volumeController) Set(level float64) error {
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	var outErr error
	vc.do(func() {
		if err := vc.ensureEndpoint(); err != nil {
			outErr = err
			return
		}
		// SetMasterVolumeLevelScalar takes its level argument by value as a
		// 32-bit float. Go's syscall trampoline for windows/amd64 mirrors
		// the first four integer-register argument slots into XMM0-XMM3 as
		// well, so placing the float's raw bits in the low 32 bits of the
		// uintptr lets the callee's `movss` pick them up correctly. This
		// does not hold on windows/arm64.
		bits := uintptr(math.Float32bits(float32(level)))
		win32.ComCall(vc.endpoint, idxAEVSetVolumeScalar, bits, 0)
	})
	return outErr
}

// ToggleMute flips the current mute state and returns the new value.
func (vc *volumeController) ToggleMute() (bool, error) {
	var newMuted bool
	var outErr error
	vc.do(func() {
		if err := vc.ensureEndpoint(); err != nil {
			outErr = err
			return
		}
		var muted int32
		win32.ComCall(vc.endpoint, idxAEVGetMute, uintptr(unsafe.Pointer(&muted)))
		next := int32(0)
		if muted == 0 {
			next = 1
		}
		win32.ComCall(vc.endpoint, idxAEVSetMute, uintptr(next), 0)
		newMuted = next != 0
	})
	return newMuted, outErr
}
