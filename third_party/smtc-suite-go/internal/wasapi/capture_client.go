//go:build windows && cgo

package wasapi

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type CaptureClient struct {
	ptr unsafe.Pointer
}

type Packet struct {
	Data           []byte
	Frames         int
	Flags          uint32
	DevicePosition uint64
	QPCPosition    uint64
	Timestamp      time.Time
}

func (p Packet) Silent() bool {
	return p.Flags&audioClientBufferFlagsSilent != 0
}

func (p Packet) DataDiscontinuity() bool {
	return p.Flags&audioClientBufferFlagsDataDiscontinuity != 0
}

func (p Packet) TimestampError() bool {
	return p.Flags&audioClientBufferFlagsTimestampError != 0
}

func (c *CaptureClient) Close() {
	if c != nil && c.ptr != nil {
		Release(c.ptr)
		c.ptr = nil
	}
}

func (c *CaptureClient) NextPacketSize() (uint32, error) {
	if c == nil || c.ptr == nil {
		return 0, fmt.Errorf("capture client is closed")
	}
	fn := vtableFn(c.ptr, slotIAudioCaptureClientGetNextPacketSize)
	var frames uint32
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(unsafe.Pointer(&frames)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("IAudioCaptureClient.GetNextPacketSize", int32(r1))
	}
	return frames, nil
}

func (c *CaptureClient) ReadPacket(format Format) (Packet, bool, error) {
	if c == nil || c.ptr == nil {
		return Packet{}, false, fmt.Errorf("capture client is closed")
	}
	packetSize, err := c.NextPacketSize()
	if err != nil {
		return Packet{}, false, err
	}
	if packetSize == 0 {
		return Packet{}, false, nil
	}

	fn := vtableFn(c.ptr, slotIAudioCaptureClientGetBuffer)
	var dataPtr unsafe.Pointer
	var frames uint32
	var flags uint32
	var devicePosition uint64
	var qpcPosition uint64
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(unsafe.Pointer(&dataPtr)),
		uintptr(unsafe.Pointer(&frames)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&devicePosition)),
		uintptr(unsafe.Pointer(&qpcPosition)),
	)
	if int32(r1) < 0 {
		return Packet{}, false, hresultErrorInt("IAudioCaptureClient.GetBuffer", int32(r1))
	}

	packet := Packet{
		Frames:         int(frames),
		Flags:          flags,
		DevicePosition: devicePosition,
		QPCPosition:    qpcPosition,
		Timestamp:      time.Now(),
	}
	if frames > 0 && dataPtr != nil && !packet.Silent() {
		bytes := int(frames) * format.BlockAlign
		packet.Data = append([]byte(nil), unsafe.Slice((*byte)(dataPtr), bytes)...)
	}

	if err := c.ReleaseBuffer(frames); err != nil {
		return Packet{}, false, err
	}
	return packet, true, nil
}

func (c *CaptureClient) ReleaseBuffer(frames uint32) error {
	fn := vtableFn(c.ptr, slotIAudioCaptureClientReleaseBuffer)
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(c.ptr),
		uintptr(frames),
	)
	if int32(r1) < 0 {
		return hresultErrorInt("IAudioCaptureClient.ReleaseBuffer", int32(r1))
	}
	return nil
}
