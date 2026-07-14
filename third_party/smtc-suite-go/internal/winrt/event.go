//go:build windows && cgo

package winrt

// #include "c/helpers.h"
import "C"
import (
	"fmt"
	"runtime"
	"runtime/cgo"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// EventHandler wraps a WinRT ITypedEventHandler<TSender, TArgs> COM object
// bridging WinRT events to Go callbacks.
//
// WinRT events follow this pattern:
//  1. Event source has add_EventName(slot N) and remove_EventName(slot N+1) methods
//  2. Create an ITypedEventHandler COM object and pass it to add_EventName
//  3. When the event fires, WinRT calls Invoke(sender, args) on the handler
//  4. add_EventName returns an EventRegistrationToken for unsubscribe
//
// Thread safety: Invoke is called on arbitrary WinRT threads. The callback
// should do minimal work and hand off to a Go channel or goroutine quickly.
//
// Object layout (heap-allocated COM identity):
//
//	[0]  vtable*
//	[8]  callback cgo.Handle
//	[16] object cgo.Handle (*EventHandler)
//	[24] refCount uint32
//	[32] captured IID (GUID, 16 bytes)

const (
	eventOffVtbl     = 0
	eventOffCallback = 8
	eventOffObject   = 16
	eventOffRefCount = 24
	eventOffIID      = 32
	eventObjSize     = 48
)

// EventHandler manages a WinRT event subscription.
type EventHandler struct {
	mu           sync.Mutex
	obj          unsafe.Pointer // C-allocated COM object (ITypedEventHandler)
	vtbl         unsafe.Pointer
	token        int64
	source       unsafe.Pointer // event source COM object
	addSlot      int            // vtable slot for add_EventName
	removeSlot   int            // vtable slot for remove_EventName
	registered   bool
	accepted     []*GUID
	objectHandle cgo.Handle // handle to EventHandler
	handle       cgo.Handle // handle to Go callback
	closed       atomic.Bool
	inflight     atomic.Int32
}

var (
	_eventQICB      = syscall.NewCallback(eventQueryInterface)
	_eventAddRefCB  = syscall.NewCallback(eventAddRef)
	_eventReleaseCB = syscall.NewCallback(eventRelease)
	_eventInvokeCB  = syscall.NewCallback(eventInvoke)
)

// NewEventHandler creates a new EventHandler.
// The callback is invoked on the WinRT event thread and should do minimal work.
func NewEventHandler(callback func(sender, args unsafe.Pointer)) *EventHandler {
	return NewTypedEventHandler(nil, callback)
}

// NewTypedEventHandler creates a new EventHandler that explicitly accepts typed
// event handler IIDs in QueryInterface.
func NewTypedEventHandler(accepted []*GUID, callback func(sender, args unsafe.Pointer)) *EventHandler {
	h := cgo.NewHandle(callback)
	ev := &EventHandler{
		accepted: accepted,
		handle:   h,
	}
	ev.objectHandle = cgo.NewHandle(ev)
	ev.obj, ev.vtbl = newEventHandlerObject(uintptr(h), uintptr(ev.objectHandle))
	return ev
}

// Register subscribes the handler to an event on the source object.
func (h *EventHandler) Register(source unsafe.Pointer, addSlot, removeSlot int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed.Load() {
		return fmt.Errorf("winrt: event handler is closed")
	}
	if h.registered {
		return nil
	}
	if h.obj == nil {
		return fmt.Errorf("winrt: event handler allocation failed")
	}

	h.source = source
	h.addSlot = addSlot
	h.removeSlot = removeSlot

	fn := vtableFn(source, addSlot)
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(source),
		uintptr(h.obj),
		uintptr(unsafe.Pointer(&h.token)),
	)
	if int32(r1) < 0 {
		return hresultErrorInt("add_EventHandler", int32(r1))
	}

	h.registered = true
	return nil
}

// Unregister removes the event subscription.
func (h *EventHandler) Unregister() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.unregisterLocked()
}

func (h *EventHandler) unregisterLocked() error {
	if !h.registered {
		return nil
	}

	fn := vtableFn(h.source, h.removeSlot)
	var token = h.token
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(h.source),
		uintptr(unsafe.Pointer(&token)),
	)
	if int32(r1) < 0 {
		return hresultErrorInt("remove_EventHandler", int32(r1))
	}

	h.registered = false
	return nil
}

// Close releases the event handler and its resources.
func (h *EventHandler) Close() error {
	h.mu.Lock()
	if h.closed.Load() {
		h.mu.Unlock()
		return nil
	}
	if err := h.unregisterLocked(); err != nil {
		h.mu.Unlock()
		return err
	}
	h.closed.Store(true)

	// Disarm Invoke immediately so in-flight / late callbacks skip Value().
	if h.obj != nil {
		*(*uintptr)(unsafe.Add(h.obj, eventOffCallback)) = 0
	}
	if h.handle > 0 {
		h.handle.Delete()
		h.handle = 0
	}

	obj := h.obj
	h.obj = nil
	h.vtbl = nil
	h.objectHandle = 0 // COM Release owns deleting the object handle
	h.mu.Unlock()

	// Wait briefly for Invoke/QI that already entered before freeref.
	deadline := time.Now().Add(2 * time.Second)
	for h.inflight.Load() > 0 && time.Now().Before(deadline) {
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}

	if obj != nil {
		eventRelease(uintptr(obj))
	}
	return nil
}

// Token returns the EventRegistrationToken.
func (h *EventHandler) Token() int64 {
	return h.token
}

func newEventHandlerObject(handle, objectHandle uintptr) (unsafe.Pointer, unsafe.Pointer) {
	obj := heapAlloc(eventObjSize)
	if obj == nil {
		return nil, nil
	}
	vtbl := heapAlloc(4 * unsafe.Sizeof(uintptr(0)))
	if vtbl == nil {
		heapFree(obj)
		return nil, nil
	}

	*(*uintptr)(unsafe.Add(vtbl, 0)) = _eventQICB
	*(*uintptr)(unsafe.Add(vtbl, 8)) = _eventAddRefCB
	*(*uintptr)(unsafe.Add(vtbl, 16)) = _eventReleaseCB
	*(*uintptr)(unsafe.Add(vtbl, 24)) = _eventInvokeCB

	*(*uintptr)(obj) = uintptr(vtbl)
	*(*uintptr)(unsafe.Add(obj, eventOffCallback)) = handle
	*(*uintptr)(unsafe.Add(obj, eventOffObject)) = objectHandle
	*(*uint32)(unsafe.Add(obj, eventOffRefCount)) = 1 // Go EventHandler owns one ref
	return obj, vtbl
}

func eventQueryInterface(self uintptr, riid uintptr, ppv uintptr) (hr uintptr) {
	defer func() {
		if recover() != nil {
			if ppv != 0 {
				*(*uintptr)(unsafe.Pointer(ppv)) = 0
			}
			hr = 0x80004002 // E_NOINTERFACE
		}
	}()

	if riid == 0 || ppv == 0 {
		return 0x80004003 // E_POINTER
	}
	g := (*GUID)(unsafe.Pointer(riid))
	if isSameGUID(g, IID_IUnknown) || isSameGUID(g, IID_IAgileObject) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		eventAddRef(self)
		return 0
	}

	objectHandle := *(*uintptr)(unsafe.Pointer(self + eventOffObject))
	if objectHandle != 0 {
		if h, ok := cgo.Handle(objectHandle).Value().(*EventHandler); ok {
			h.inflight.Add(1)
			defer h.inflight.Add(-1)
			for _, accepted := range h.accepted {
				if isSameGUID(g, accepted) {
					*(*uintptr)(unsafe.Pointer(ppv)) = self
					eventAddRef(self)
					return 0
				}
			}
			if len(h.accepted) > 0 {
				*(*uintptr)(unsafe.Pointer(ppv)) = 0
				return 0x80004002 // E_NOINTERFACE
			}
		}
	}

	capturedPtr := (*GUID)(unsafe.Pointer(self + eventOffIID))
	if capturedPtr.Data1 == 0 && capturedPtr.Data2 == 0 && capturedPtr.Data3 == 0 {
		*capturedPtr = *g
	}
	if isSameGUID(g, capturedPtr) {
		*(*uintptr)(unsafe.Pointer(ppv)) = self
		eventAddRef(self)
		return 0
	}

	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	return 0x80004002 // E_NOINTERFACE
}

func eventAddRef(self uintptr) uintptr {
	if self == 0 {
		return 0
	}
	n := atomic.AddUint32((*uint32)(unsafe.Pointer(self+eventOffRefCount)), 1)
	return uintptr(n)
}

func eventRelease(self uintptr) uintptr {
	if self == 0 {
		return 0
	}
	n := atomic.AddUint32((*uint32)(unsafe.Pointer(self+eventOffRefCount)), ^uint32(0))
	if n != 0 {
		return uintptr(n)
	}

	// Last reference: disarm handles and free C memory.
	if cb := *(*uintptr)(unsafe.Pointer(self + eventOffCallback)); cb != 0 {
		*(*uintptr)(unsafe.Pointer(self + eventOffCallback)) = 0
		cgo.Handle(cb).Delete()
	}
	if oh := *(*uintptr)(unsafe.Pointer(self + eventOffObject)); oh != 0 {
		*(*uintptr)(unsafe.Pointer(self + eventOffObject)) = 0
		cgo.Handle(oh).Delete()
	}
	vtbl := *(*uintptr)(unsafe.Pointer(self + eventOffVtbl))
	heapFree(unsafe.Pointer(self))
	if vtbl != 0 {
		heapFree(unsafe.Pointer(vtbl))
	}
	return 0
}

func eventInvoke(self uintptr, sender uintptr, args uintptr) (hr uintptr) {
	defer func() {
		if recover() != nil {
			hr = 0
		}
	}()

	if self == 0 {
		return 0
	}
	eventAddRef(self)
	defer eventRelease(self)

	handle := *(*uintptr)(unsafe.Pointer(self + eventOffCallback))
	if handle == 0 {
		return 0
	}
	objectHandle := *(*uintptr)(unsafe.Pointer(self + eventOffObject))
	if objectHandle != 0 {
		if h, ok := cgo.Handle(objectHandle).Value().(*EventHandler); ok {
			if h.closed.Load() {
				return 0
			}
			h.inflight.Add(1)
			defer h.inflight.Add(-1)
		}
	}

	value := cgo.Handle(handle).Value()
	callback, ok := value.(func(sender, args unsafe.Pointer))
	if !ok || callback == nil {
		return 0
	}
	callback(unsafe.Pointer(sender), unsafe.Pointer(args))
	return 0
}
