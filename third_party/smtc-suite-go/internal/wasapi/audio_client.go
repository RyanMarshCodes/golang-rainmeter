//go:build windows && cgo

package wasapi

import (
	"fmt"
	"syscall"
	"unsafe"
)

type AudioClient struct {
	ptr unsafe.Pointer
}

func (c *AudioClient) Close() {
	if c != nil && c.ptr != nil {
		Release(c.ptr)
		c.ptr = nil
	}
}

func (c *AudioClient) GetMixFormat() (Format, error) {
	if c == nil || c.ptr == nil {
		return Format{}, fmt.Errorf("audio client is closed")
	}
	fn := vtableFn(c.ptr, slotIAudioClientGetMixFormat)
	var formatPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(unsafe.Pointer(&formatPtr)),
	)
	if int32(r1) < 0 {
		return Format{}, hresultErrorInt("IAudioClient.GetMixFormat", int32(r1))
	}
	defer coTaskMemFree(formatPtr)
	return parseWaveFormat(formatPtr)
}

func (c *AudioClient) InitializeLoopback(format Format, bufferDuration100ns int64) error {
	if c == nil || c.ptr == nil {
		return fmt.Errorf("audio client is closed")
	}
	if bufferDuration100ns < 0 {
		return fmt.Errorf("buffer duration must be non-negative")
	}
	formatPtr, err := c.rawMixFormat()
	if err != nil {
		return err
	}
	defer coTaskMemFree(formatPtr)

	fn := vtableFn(c.ptr, slotIAudioClientInitialize)
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(audioClientShareModeShared),
		uintptr(audioClientStreamFlagsLoopback),
		uintptr(bufferDuration100ns),
		0,
		uintptr(formatPtr),
		0,
	)
	if int32(r1) < 0 {
		return hresultErrorInt("IAudioClient.Initialize(loopback)", int32(r1))
	}
	_ = format
	return nil
}

func (c *AudioClient) rawMixFormat() (unsafe.Pointer, error) {
	fn := vtableFn(c.ptr, slotIAudioClientGetMixFormat)
	var formatPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(unsafe.Pointer(&formatPtr)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IAudioClient.GetMixFormat", int32(r1))
	}
	return formatPtr, nil
}

func (c *AudioClient) GetCaptureClient() (*CaptureClient, error) {
	if c == nil || c.ptr == nil {
		return nil, fmt.Errorf("audio client is closed")
	}
	fn := vtableFn(c.ptr, slotIAudioClientGetService)
	var capture unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(unsafe.Pointer(IID_IAudioCaptureClient)),
		uintptr(unsafe.Pointer(&capture)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IAudioClient.GetService(IAudioCaptureClient)", int32(r1))
	}
	return &CaptureClient{ptr: capture}, nil
}

func (c *AudioClient) Start() error {
	if c == nil || c.ptr == nil {
		return fmt.Errorf("audio client is closed")
	}
	return vtableNoArgs(c.ptr, slotIAudioClientStart, "IAudioClient.Start")
}

func (c *AudioClient) Stop() error {
	if c == nil || c.ptr == nil {
		return fmt.Errorf("audio client is closed")
	}
	return vtableNoArgs(c.ptr, slotIAudioClientStop, "IAudioClient.Stop")
}
