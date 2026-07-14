//go:build windows && cgo

package wasapi

import (
	"fmt"
	"time"
)

type LoopbackCapture struct {
	enumerator *DeviceEnumerator
	device     *Device
	audio      *AudioClient
	capture    *CaptureClient
	format     Format
}

func OpenDefaultLoopback(bufferDuration100ns int64) (*LoopbackCapture, error) {
	if bufferDuration100ns == 0 {
		bufferDuration100ns = defaultBufferDuration100ns
	}

	enumerator, err := NewDeviceEnumerator()
	if err != nil {
		return nil, err
	}
	l := &LoopbackCapture{enumerator: enumerator}
	defer func() {
		if err != nil {
			l.Close()
		}
	}()

	l.device, err = enumerator.DefaultRenderDevice()
	if err != nil {
		return nil, err
	}
	l.audio, err = l.device.ActivateAudioClient()
	if err != nil {
		return nil, err
	}
	l.format, err = l.audio.GetMixFormat()
	if err != nil {
		return nil, err
	}
	if err = l.audio.InitializeLoopback(l.format, bufferDuration100ns); err != nil {
		return nil, err
	}
	l.capture, err = l.audio.GetCaptureClient()
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *LoopbackCapture) Format() Format {
	if l == nil {
		return Format{}
	}
	return l.format
}

func (l *LoopbackCapture) Start() error {
	if l == nil || l.audio == nil {
		return fmt.Errorf("loopback capture is closed")
	}
	return l.audio.Start()
}

func (l *LoopbackCapture) Stop() error {
	if l == nil || l.audio == nil {
		return fmt.Errorf("loopback capture is closed")
	}
	return l.audio.Stop()
}

func (l *LoopbackCapture) ReadPacket() (Packet, bool, error) {
	if l == nil || l.capture == nil {
		return Packet{}, false, fmt.Errorf("loopback capture is closed")
	}
	return l.capture.ReadPacket(l.format)
}

func (l *LoopbackCapture) ReadPacketsFor(duration time.Duration, sleep time.Duration, onPacket func(Packet) error) error {
	if sleep <= 0 {
		sleep = 10 * time.Millisecond
	}
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		packet, ok, err := l.ReadPacket()
		if err != nil {
			return err
		}
		if ok {
			if err := onPacket(packet); err != nil {
				return err
			}
			continue
		}
		time.Sleep(sleep)
	}
	return nil
}

func (l *LoopbackCapture) Close() {
	if l == nil {
		return
	}
	if l.capture != nil {
		l.capture.Close()
		l.capture = nil
	}
	if l.audio != nil {
		l.audio.Close()
		l.audio = nil
	}
	if l.device != nil {
		l.device.Close()
		l.device = nil
	}
	if l.enumerator != nil {
		l.enumerator.Close()
		l.enumerator = nil
	}
}
