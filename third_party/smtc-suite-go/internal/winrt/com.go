//go:build windows && cgo

package winrt

// #include "c/helpers.h"
import "C"
import (
	"syscall"
	"unsafe"
)

// COM vtable dispatch utilities.
//
// At the binary level, a COM/WinRT object is a pointer to a pointer to a
// vtable (array of function pointers):
//
//   obj -> *vtable_ptr -> [fn0, fn1, fn2, ...]
//
// IUnknown occupies slots 0-2: QueryInterface, AddRef, Release
// IInspectable adds slots 3-5: GetIids, GetRuntimeClassName, GetTrustLevel
// WinRT interface methods start at slot 6.

var vtableSlotSize = unsafe.Sizeof(uintptr(0))

// VtableFn returns the function pointer at the given vtable slot index.
// Exported for debug purposes.
func VtableFn(obj unsafe.Pointer, slot int) uintptr {
	return vtableFn(obj, slot)
}

// vtableFn returns the function pointer at the given vtable slot index.
func vtableFn(obj unsafe.Pointer, slot int) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	slotAddr := unsafe.Add(vtbl, int(vtableSlotSize)*slot)
	if vtableSlotSize == 8 {
		return uintptr(*(*uint64)(slotAddr))
	}
	return uintptr(*(*uint32)(slotAddr))
}

// AddRef increments the COM object's reference count.
func AddRef(obj unsafe.Pointer) uint32 {
	fn := vtableFn(obj, 1)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj))
	return uint32(r1)
}

// Release decrements the COM object's reference count.
func Release(obj unsafe.Pointer) uint32 {
	fn := vtableFn(obj, 2)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj))
	return uint32(r1)
}

// QueryInterface queries a COM object for a specific interface.
func QueryInterface(obj unsafe.Pointer, iid *GUID) (unsafe.Pointer, error) {
	fn := vtableFn(obj, 0)
	var result unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("QueryInterface", int32(r1))
	}
	return result, nil
}

// ---- Exported vtable call helpers ----

// VtableGetPtr calls a vtable method that returns a COM pointer via an out param.
// slot = vtable slot index (0-based).
func VtableGetPtr(obj unsafe.Pointer, slot int) (unsafe.Pointer, error) {
	fn := vtableFn(obj, slot)
	var result unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("VtableGetPtr", int32(r1))
	}
	return result, nil
}

// VtableGetPtrWithArg calls a vtable method that takes one uintptr argument
// and returns a COM pointer via an out param. Used for IVectorView::GetAt(index).
func VtableGetPtrWithArg(obj unsafe.Pointer, slot int, arg uintptr) (unsafe.Pointer, error) {
	fn := vtableFn(obj, slot)
	var result unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		arg,
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("VtableGetPtrWithArg", int32(r1))
	}
	return result, nil
}

// VtableGetHSTRING calls a vtable method that returns an HSTRING via out param.
func VtableGetHSTRING(obj unsafe.Pointer, slot int) (*HSTRING, error) {
	fn := vtableFn(obj, slot)
	hstr := &HSTRING{}
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&hstr.h)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("VtableGetHSTRING", int32(r1))
	}
	return hstr, nil
}

// VtableGetBool calls a vtable method that returns a C boolean via out param.
func VtableGetBool(obj unsafe.Pointer, slot int) (bool, error) {
	fn := vtableFn(obj, slot)
	var result C.boolean
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return false, hresultErrorInt("VtableGetBool", int32(r1))
	}
	return result != 0, nil
}

// VtableGetI32 calls a vtable method that returns an int32 via out param.
func VtableGetI32(obj unsafe.Pointer, slot int) (int32, error) {
	fn := vtableFn(obj, slot)
	var result int32
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("VtableGetI32", int32(r1))
	}
	return result, nil
}

// VtableGetU32 calls a vtable method that returns a uint32 via out param.
func VtableGetU32(obj unsafe.Pointer, slot int) (uint32, error) {
	fn := vtableFn(obj, slot)
	var result uint32
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("VtableGetU32", int32(r1))
	}
	return result, nil
}

// VtableGetF64 calls a vtable method that returns a float64 via out param.
func VtableGetF64(obj unsafe.Pointer, slot int) (float64, error) {
	fn := vtableFn(obj, slot)
	var result float64
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("VtableGetF64", int32(r1))
	}
	return result, nil
}

// VtableGetTicks calls a vtable method that returns int64 (TimeSpan in 100ns ticks).
func VtableGetTicks(obj unsafe.Pointer, slot int) (int64, error) {
	fn := vtableFn(obj, slot)
	var result int64
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(obj),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("VtableGetTicks", int32(r1))
	}
	return result, nil
}

// ReferenceGetI32 reads the Value property from an IReference<int32-like> object.
func ReferenceGetI32(obj unsafe.Pointer) (int32, error) {
	return VtableGetI32(obj, 6)
}

// ReferenceGetBool reads the Value property from an IReference<bool> object.
func ReferenceGetBool(obj unsafe.Pointer) (bool, error) {
	return VtableGetBool(obj, 6)
}

// ReferenceGetF64 reads the Value property from an IReference<double> object.
func ReferenceGetF64(obj unsafe.Pointer) (float64, error) {
	return VtableGetF64(obj, 6)
}

// VtablePutBool calls a vtable put_* method with a boolean value.
func VtablePutBool(obj unsafe.Pointer, slot int, value bool) error {
	fn := vtableFn(obj, slot)
	v := uintptr(0)
	if value {
		v = 1
	}
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), v)
	if int32(r1) < 0 {
		return hresultErrorInt("VtablePutBool", int32(r1))
	}
	return nil
}

// VtablePutI32 calls a vtable put_* method with an int32 value.
func VtablePutI32(obj unsafe.Pointer, slot int, value int32) error {
	fn := vtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), uintptr(value))
	if int32(r1) < 0 {
		return hresultErrorInt("VtablePutI32", int32(r1))
	}
	return nil
}

// VtablePutTicks calls a vtable put_* method with a TimeSpan value.
//
// WinRT TimeSpan is represented as a signed 64-bit count of 100ns ticks and is
// passed by value in the Windows x64 integer ABI.
func VtablePutTicks(obj unsafe.Pointer, slot int, value int64) error {
	fn := vtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), uintptr(value))
	if int32(r1) < 0 {
		return hresultErrorInt("VtablePutTicks", int32(r1))
	}
	return nil
}

// VtablePutF64 calls a vtable put_* method with a float64 value.
func VtablePutF64(obj unsafe.Pointer, slot int, value float64) error {
	fn := vtableFn(obj, slot)
	hr := C.smtcVtablePutF64(
		obj,
		unsafe.Pointer(fn),
		C.double(value),
	)
	if hr < 0 {
		return hresultError("VtablePutF64", hr)
	}
	return nil
}

// VtableAsyncBoolWithF64 calls a method that takes a float64 and returns
// IAsyncOperation<bool> through an out parameter.
func VtableAsyncBoolWithF64(obj unsafe.Pointer, slot int, value float64) (unsafe.Pointer, error) {
	fn := vtableFn(obj, slot)
	var asyncPtr unsafe.Pointer
	hr := C.smtcAsyncF64(
		obj,
		unsafe.Pointer(fn),
		C.double(value),
		&asyncPtr,
	)
	if hr < 0 {
		return nil, hresultError("VtableAsyncBoolWithF64", hr)
	}
	return asyncPtr, nil
}

// VtablePutHSTRING calls a vtable put_* method with an HSTRING value.
func VtablePutHSTRING(obj unsafe.Pointer, slot int, hstr *HSTRING) error {
	fn := vtableFn(obj, slot)
	var raw C.HSTRING
	if hstr != nil {
		raw = hstr.h
	}
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), uintptr(unsafe.Pointer(raw)))
	if int32(r1) < 0 {
		return hresultErrorInt("VtablePutHSTRING", int32(r1))
	}
	return nil
}

// ---- Internal (unexported) helpers ----

// VtableCall3 calls a vtable method (slot) with object + 1 arg, returning HRESULT.
func VtableCall3(obj unsafe.Pointer, slot int, arg1 uintptr) error {
	fn := vtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), arg1)
	if int32(r1) < 0 {
		return hresultErrorInt("vtableCall", int32(r1))
	}
	return nil
}

// vtableCall4 calls a vtable method with object + 2 args, returning HRESULT.
func vtableCall4(obj unsafe.Pointer, slot int, arg1, arg2 uintptr) error {
	fn := vtableFn(obj, slot)
	r1, _, _ := syscall.SyscallN(fn, uintptr(obj), arg1, arg2)
	if int32(r1) < 0 {
		return hresultErrorInt("vtableCall", int32(r1))
	}
	return nil
}
