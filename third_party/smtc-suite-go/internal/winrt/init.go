//go:build windows && cgo

package winrt

// #include "c/helpers.h"
import "C"
import (
	"fmt"
	"runtime"
	"sync/atomic"
)

// ---- MTA Initialization (for Monitor and Control modules) ----

var (
	mtaInits atomic.Int32
	roInits  atomic.Int32
	coInits  atomic.Int32
)

// InitMTA initializes COM/WinRT on the calling thread in MTA mode.
//
// Tries CoInitializeEx first (like go-libnp), then RoInitialize as fallback.
func InitMTA() error {
	// Try classic COM init first (COINIT_MULTITHREADED = 0)
	hr := C.CoInitializeEx(nil, 0)
	if hr >= 0 {
		coInits.Add(1)
		mtaInits.Add(1)
		return nil
	}
	// Fall back to RoInitialize. RPC_E_CHANGED_MODE means this thread is already
	// initialized for another apartment; WinRT activation can still be used from
	// that thread, so treat it as an attach that needs no uninitialize call here.
	if uint32(hr) == 0x80010106 {
		mtaInits.Add(1)
		return nil
	}

	roHR := C.RoInitialize(C.RO_INIT_MULTITHREADED)
	if roHR >= 0 {
		roInits.Add(1)
		mtaInits.Add(1)
		return nil
	}
	if uint32(roHR) == 0x80010106 {
		mtaInits.Add(1)
		return nil
	}
	return hresultError("COM/WinRT init (MTA)", roHR)
}

// UninitMTA uninitializes COM/WinRT on the calling thread.
func UninitMTA() {
	if mtaInits.Load() > 0 {
		mtaInits.Add(-1)
	}
	if coInits.Load() > 0 {
		coInits.Add(-1)
		C.CoUninitialize()
		return
	}
	if roInits.Load() > 0 {
		roInits.Add(-1)
		C.RoUninitialize()
	}
}

// ---- STA Initialization (for Create module) ----

// InitSTA initializes the Windows Runtime on the calling thread in
// Single-Threaded Apartment (STA) mode. The caller MUST lock the current
// goroutine to an OS thread via runtime.LockOSThread() before calling this.
//
// Each call to InitSTA must be paired with a call to UninitSTA.
func InitSTA() error {
	hr := C.RoInitialize(C.RO_INIT_SINGLETHREADED)
	if hr < 0 {
		return hresultError("RoInitialize(STA)", hr)
	}
	runtime.LockOSThread()
	return nil
}

// UninitSTA uninitializes the Windows Runtime (STA mode).
// The caller may then call runtime.UnlockOSThread().
func UninitSTA() {
	C.RoUninitialize()
}

// hresultError creates an error from an HRESULT value.
func hresultError(op string, hr C.HRESULT) error {
	return hresultErrorInt(op, int32(hr))
}

// hresultErrorInt creates an error from an HRESULT value (Go int32).
func hresultErrorInt(op string, hr int32) error {
	if hr >= 0 {
		return nil
	}
	return &HresultError{Op: op, Code: hr}
}

// HresultError represents a failed HRESULT from a COM/WinRT call.
type HresultError struct {
	Op   string
	Code int32
}

func (e *HresultError) Error() string {
	return fmt.Sprintf("winrt: %s: HRESULT 0x%08X", e.Op, uint32(e.Code))
}
