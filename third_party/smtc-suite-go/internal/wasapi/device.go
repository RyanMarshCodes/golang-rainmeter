//go:build windows && cgo

package wasapi

import (
	"fmt"
	"syscall"
	"unsafe"
)

type DeviceEnumerator struct {
	ptr unsafe.Pointer
}

type Device struct {
	ptr unsafe.Pointer
}

func NewDeviceEnumerator() (*DeviceEnumerator, error) {
	ptr, err := CoCreateInstance(CLSID_MMDeviceEnumerator, IID_IMMDeviceEnumerator)
	if err != nil {
		return nil, fmt.Errorf("MMDeviceEnumerator: %w", err)
	}
	return &DeviceEnumerator{ptr: ptr}, nil
}

func (e *DeviceEnumerator) Close() {
	if e != nil && e.ptr != nil {
		Release(e.ptr)
		e.ptr = nil
	}
}

func (e *DeviceEnumerator) DefaultRenderDevice() (*Device, error) {
	if e == nil || e.ptr == nil {
		return nil, fmt.Errorf("device enumerator is closed")
	}
	fn := vtableFn(e.ptr, slotIMMDeviceEnumeratorGetDefaultAudioEndpoint)
	var device unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(e.ptr),
		uintptr(eRender),
		uintptr(eConsole),
		uintptr(unsafe.Pointer(&device)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IMMDeviceEnumerator.GetDefaultAudioEndpoint", int32(r1))
	}
	return &Device{ptr: device}, nil
}

func (d *Device) Close() {
	if d != nil && d.ptr != nil {
		Release(d.ptr)
		d.ptr = nil
	}
}

func (d *Device) ActivateAudioClient() (*AudioClient, error) {
	if d == nil || d.ptr == nil {
		return nil, fmt.Errorf("device is closed")
	}
	fn := vtableFn(d.ptr, slotIMMDeviceActivate)
	var client unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(d.ptr),
		uintptr(unsafe.Pointer(IID_IAudioClient)),
		uintptr(clsctxAll),
		0,
		uintptr(unsafe.Pointer(&client)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IMMDevice.Activate(IAudioClient)", int32(r1))
	}
	return &AudioClient{ptr: client}, nil
}
