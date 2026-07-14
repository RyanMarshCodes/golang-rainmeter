//go:build windows && cgo

package wasapi

// #include "c/helpers.h"
import "C"
import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var vtableSlotSize = unsafe.Sizeof(uintptr(0))

// HresultError represents a failed HRESULT from a WASAPI/COM call.
type HresultError struct {
	Op   string
	Code int32
}

func (e *HresultError) Error() string {
	return fmt.Sprintf("wasapi: %s: HRESULT 0x%08X", e.Op, uint32(e.Code))
}

func hresultError(op string, hr C.HRESULT) error {
	return hresultErrorInt(op, int32(hr))
}

func hresultErrorInt(op string, hr int32) error {
	if hr >= 0 {
		return nil
	}
	return &HresultError{Op: op, Code: hr}
}

// Init initializes COM on the current thread for WASAPI use.
func Init() error {
	hr := C.CoInitializeEx(nil, C.COINIT_MULTITHREADED)
	if hr < 0 && uint32(hr) != 0x80010106 {
		return hresultError("CoInitializeEx(MTA)", hr)
	}
	return nil
}

// Uninit uninitializes COM on the current thread.
func Uninit() {
	C.CoUninitialize()
}

// RunOnLockedThread runs fn on a locked OS thread with COM initialized.
func RunOnLockedThread(fn func() error) error {
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if err := Init(); err != nil {
			done <- err
			return
		}
		defer Uninit()
		done <- fn()
	}()
	return <-done
}

func CoCreateInstance(clsid *GUID, iid *GUID) (unsafe.Pointer, error) {
	var result unsafe.Pointer
	hr := C.CoCreateInstance(
		(*C.IID)(unsafe.Pointer(clsid)),
		nil,
		C.DWORD(clsctxAll),
		(*C.IID)(unsafe.Pointer(iid)),
		(*C.LPVOID)(unsafe.Pointer(&result)),
	)
	if hr < 0 {
		return nil, hresultError("CoCreateInstance", hr)
	}
	return result, nil
}

func Release(obj unsafe.Pointer) uint32 {
	if obj == nil {
		return 0
	}
	fn := vtableFn(obj, 2)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj))
	return uint32(r1)
}

func vtableFn(obj unsafe.Pointer, slot int) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	slotAddr := unsafe.Add(vtbl, int(vtableSlotSize)*slot)
	if vtableSlotSize == 8 {
		return uintptr(*(*uint64)(slotAddr))
	}
	return uintptr(*(*uint32)(slotAddr))
}

func vtableNoArgs(obj unsafe.Pointer, slot int, op string) error {
	fn := vtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj))
	if int32(r1) < 0 {
		return hresultErrorInt(op, int32(r1))
	}
	return nil
}

func coTaskMemFree(ptr unsafe.Pointer) {
	if ptr != nil {
		C.CoTaskMemFree(C.LPVOID(ptr))
	}
}
