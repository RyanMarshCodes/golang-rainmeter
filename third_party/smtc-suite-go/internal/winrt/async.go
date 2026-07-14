//go:build windows && cgo

package winrt

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// IAsyncOperation vtable slots:
//
//	IUnknown 0-2, IInspectable 3-5, PutCompleted 6, GetCompleted 7, GetResults 8
const (
	asyncSlotPutCompleted = 6
	asyncSlotGetCompleted = 7
	asyncSlotGetResults   = 8
)

// IAsyncOperationWithProgress<TResult, TProgress> vtable slots:
//
//	IUnknown 0-2, IInspectable 3-5, PutProgress 6, GetProgress 7,
//	PutCompleted 8, GetCompleted 9, GetResults 10
const (
	asyncProgressSlotPutCompleted = 8
	asyncProgressSlotGetResults   = 10
)

type asyncStatus int32

const (
	asyncStatusStarted   asyncStatus = 0
	asyncStatusCompleted asyncStatus = 1
	asyncStatusCanceled  asyncStatus = 2
	asyncStatusError     asyncStatus = 3
)

const defaultAsyncTimeout = 30 * time.Second

// ---- CompletionHandler built with Go callbacks (go-libnp pattern) ----

// completionHandlerVtbl: IUnknown(3) + Invoke(1) = 4 slots, using syscall.NewCallback
type completionHandler struct {
	obj uintptr // raw COM object pointer (HeapAlloc'd)
	ch  chan asyncStatus
}

// Global Go-callable functions for the handler vtable (used via syscall.NewCallback)
var (
	_qicb   = syscall.NewCallback(handlerQueryInterface)
	_addrcb = syscall.NewCallback(handlerAddRef)
	_relcb  = syscall.NewCallback(handlerRelease)
	_invcb  = syscall.NewCallback(handlerInvoke)
)

func handlerQueryInterface(self uintptr, riid uintptr, ppv uintptr) uintptr {
	if riid == 0 || ppv == 0 {
		return 0x80004003 // E_POINTER
	}
	g := (*GUID)(unsafe.Pointer(riid))

	// IUnknown: {00000000-0000-0000-C000-000000000046}
	unknownIID := GUID{0, 0, 0, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	if isSameGUID(g, &unknownIID) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		return 0
	}
	// IInspectable: {AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}
	inspectIID := GUID{0xAF86E2E0, 0xB12D, 0x4C6A, [8]byte{0x9C, 0x5A, 0xD7, 0xAA, 0x65, 0x10, 0x1E, 0x90}}
	if isSameGUID(g, &inspectIID) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		return 0
	}
	if isSameGUID(g, IID_IAgileObject) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		return 0
	}

	// Store the first non-standard IID per handler instance, then accept
	// subsequent queries for that same parameterized handler IID.
	capturedPtr := (*GUID)(unsafe.Pointer(self + 16))
	if capturedPtr.Data1 == 0 && capturedPtr.Data2 == 0 && capturedPtr.Data3 == 0 {
		// First non-standard IID — capture it
		*capturedPtr = *g
	}
	if isSameGUID(g, capturedPtr) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		return 0
	}

	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	return 0x80004002 // E_NOINTERFACE
}

func isSameGUID(a, b *GUID) bool {
	if a.Data1 != b.Data1 {
		return false
	}
	if a.Data2 != b.Data2 {
		return false
	}
	if a.Data3 != b.Data3 {
		return false
	}
	for i := 0; i < 8; i++ {
		if a.Data4[i] != b.Data4[i] {
			return false
		}
	}
	return true
}

func handlerAddRef(self uintptr) uintptr {
	return 1 // dummy
}

func handlerRelease(self uintptr) uintptr {
	return 1 // dummy
}

func handlerInvoke(self uintptr, operation uintptr, status int32) uintptr {
	handle := *(*uintptr)(unsafe.Pointer(self + unsafe.Sizeof(uintptr(0)) + 8 + 16))
	if handle != 0 {
		h := (*completionHandler)(unsafe.Pointer(handle))
		select {
		case h.ch <- asyncStatus(status):
		default:
		}
	}
	return 0
}

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procHeapAlloc      = modkernel32.NewProc("HeapAlloc")
	procHeapFree       = modkernel32.NewProc("HeapFree")
	procGetProcessHeap = modkernel32.NewProc("GetProcessHeap")
)

func heapAlloc(size uintptr) unsafe.Pointer {
	heap, _, _ := procGetProcessHeap.Call()
	r, _, _ := procHeapAlloc.Call(heap, 8, size) // HEAP_ZERO_MEMORY=8
	return unsafe.Pointer(r)
}

func heapFree(ptr unsafe.Pointer) {
	heap, _, _ := procGetProcessHeap.Call()
	procHeapFree.Call(heap, 0, uintptr(ptr))
}

func newCompletionHandler() *completionHandler {
	h := &completionHandler{
		ch: make(chan asyncStatus, 1),
	}

	// Allocate vtable + object from process heap
	// Object layout: [vtable_ptr] [refCount|padding] [capturedIID] [go handler pointer]
	objSize := unsafe.Sizeof(uintptr(0)) + 8 + 16 + unsafe.Sizeof(uintptr(0))
	objPtr := heapAlloc(objSize)
	if objPtr == nil {
		return nil
	}

	// Vtable: 4 function pointers (QI, AddRef, Release, Invoke)
	vtSize := 4 * unsafe.Sizeof(uintptr(0))
	vtPtr := heapAlloc(vtSize)
	if vtPtr == nil {
		heapFree(objPtr)
		return nil
	}

	*(*uintptr)(unsafe.Add(vtPtr, 0)) = _qicb
	*(*uintptr)(unsafe.Add(vtPtr, 8)) = _addrcb
	*(*uintptr)(unsafe.Add(vtPtr, 16)) = _relcb
	*(*uintptr)(unsafe.Add(vtPtr, 24)) = _invcb

	// Object points to vtable
	*(*uintptr)(objPtr) = uintptr(vtPtr)
	*(*uintptr)(unsafe.Add(objPtr, unsafe.Sizeof(uintptr(0))+8+16)) = uintptr(unsafe.Pointer(h))

	h.obj = uintptr(objPtr)

	return h
}

func (h *completionHandler) ptr() uintptr {
	return h.obj
}

func (h *completionHandler) close() {
	if h.obj != 0 {
		vtPtr := *(*uintptr)(unsafe.Pointer(h.obj))
		if vtPtr != 0 {
			heapFree(unsafe.Pointer(vtPtr))
		}
		heapFree(unsafe.Pointer(h.obj))
		h.obj = 0
	}
}

// ---- AsyncOperation ----

type AsyncOperation struct {
	Ptr unsafe.Pointer
}

func NewAsyncOperation(p unsafe.Pointer) *AsyncOperation {
	return &AsyncOperation{Ptr: p}
}

func (a *AsyncOperation) Wait() (unsafe.Pointer, error) {
	return a.WaitTimeout(defaultAsyncTimeout)
}

func (a *AsyncOperation) WaitTimeout(timeout time.Duration) (unsafe.Pointer, error) {
	if a.Ptr == nil {
		return nil, fmt.Errorf("winrt: async operation is nil")
	}

	h := newCompletionHandler()
	if h == nil {
		return nil, fmt.Errorf("winrt: failed to create completion handler")
	}
	defer h.close()

	fn := vtableFn(a.Ptr, asyncSlotPutCompleted)
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(a.Ptr),
		h.ptr(),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("put_Completed", int32(r1))
	}

	select {
	case s := <-h.ch:
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		switch s {
		case asyncStatusCompleted:
			return a.GetResults()
		case asyncStatusError:
			return nil, fmt.Errorf("winrt: async operation failed (status=Error)")
		case asyncStatusCanceled:
			return nil, fmt.Errorf("winrt: async operation was canceled")
		default:
			return nil, fmt.Errorf("winrt: async operation status %d", s)
		}
	case <-time.After(timeout):
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		return nil, fmt.Errorf("winrt: async operation timed out after %v", timeout)
	}
}

func (a *AsyncOperation) GetResults() (unsafe.Pointer, error) {
	return VtableGetPtr(a.Ptr, asyncSlotGetResults)
}

func (a *AsyncOperation) Close() error { return nil }

type AsyncOperationWithProgress struct {
	Ptr unsafe.Pointer
}

func NewAsyncOperationWithProgress(p unsafe.Pointer) *AsyncOperationWithProgress {
	return &AsyncOperationWithProgress{Ptr: p}
}

func (a *AsyncOperationWithProgress) Wait() (unsafe.Pointer, error) {
	return a.WaitTimeout(defaultAsyncTimeout)
}

func (a *AsyncOperationWithProgress) WaitTimeout(timeout time.Duration) (unsafe.Pointer, error) {
	if a.Ptr == nil {
		return nil, fmt.Errorf("winrt: async operation with progress is nil")
	}

	h := newCompletionHandler()
	if h == nil {
		return nil, fmt.Errorf("winrt: failed to create completion handler")
	}
	defer h.close()

	fn := vtableFn(a.Ptr, asyncProgressSlotPutCompleted)
	r1, _, _ := syscall.SyscallN(fn, uintptr(a.Ptr), h.ptr())
	if int32(r1) < 0 {
		return nil, hresultErrorInt("put_Completed", int32(r1))
	}

	select {
	case s := <-h.ch:
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		switch s {
		case asyncStatusCompleted:
			return VtableGetPtr(a.Ptr, asyncProgressSlotGetResults)
		case asyncStatusError:
			return nil, fmt.Errorf("winrt: async operation failed (status=Error)")
		case asyncStatusCanceled:
			return nil, fmt.Errorf("winrt: async operation was canceled")
		default:
			return nil, fmt.Errorf("winrt: async operation status %d", s)
		}
	case <-time.After(timeout):
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		return nil, fmt.Errorf("winrt: async operation timed out after %v", timeout)
	}
}

func (a *AsyncOperationWithProgress) Release() {
	if a.Ptr != nil {
		Release(a.Ptr)
		a.Ptr = nil
	}
}

func (a *AsyncOperation) Release() {
	if a.Ptr != nil {
		Release(a.Ptr)
		a.Ptr = nil
	}
}

// ---- AsyncOperationBool ----

type AsyncOperationBool struct {
	Ptr unsafe.Pointer
}

func NewAsyncOperationBool(p unsafe.Pointer) *AsyncOperationBool {
	return &AsyncOperationBool{Ptr: p}
}

func (a *AsyncOperationBool) Wait() (bool, error) {
	return a.WaitTimeout(defaultAsyncTimeout)
}

func (a *AsyncOperationBool) WaitTimeout(timeout time.Duration) (bool, error) {
	if a.Ptr == nil {
		return false, fmt.Errorf("winrt: async operation is nil")
	}

	h := newCompletionHandler()
	if h == nil {
		return false, fmt.Errorf("winrt: failed to create completion handler")
	}
	defer h.close()

	fn := vtableFn(a.Ptr, asyncSlotPutCompleted)
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(a.Ptr),
		h.ptr(),
	)
	if int32(r1) < 0 {
		return false, hresultErrorInt("put_Completed", int32(r1))
	}

	select {
	case s := <-h.ch:
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		switch s {
		case asyncStatusCompleted:
			return getAsyncBoolResult(a.Ptr)
		case asyncStatusError:
			return false, fmt.Errorf("winrt: async operation failed (status=Error)")
		case asyncStatusCanceled:
			return false, fmt.Errorf("winrt: async operation was canceled")
		default:
			return false, fmt.Errorf("winrt: async operation status %d", s)
		}
	case <-time.After(timeout):
		syscall.SyscallN(fn, uintptr(a.Ptr), 0)
		return false, fmt.Errorf("winrt: async operation timed out after %v", timeout)
	}
}

func (a *AsyncOperationBool) GetResults() (bool, error) {
	return getAsyncBoolResult(a.Ptr)
}
func (a *AsyncOperationBool) Close() error { return nil }
func (a *AsyncOperationBool) Release() {
	if a.Ptr != nil {
		Release(a.Ptr)
		a.Ptr = nil
	}
}

func getAsyncBoolResult(ptr unsafe.Pointer) (bool, error) {
	fn := vtableFn(ptr, asyncSlotGetResults)
	var result byte
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(ptr),
		uintptr(unsafe.Pointer(&result)),
	)
	if int32(r1) < 0 {
		return false, hresultErrorInt("GetResults", int32(r1))
	}
	return result != 0, nil
}
