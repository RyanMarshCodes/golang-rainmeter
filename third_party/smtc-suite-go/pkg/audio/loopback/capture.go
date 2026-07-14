//go:build windows && cgo

package loopback

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/internal/wasapi"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
)

const (
	defaultFrameBuffer = 32
	defaultErrorBuffer = 8
	defaultPollDelay   = 10 * time.Millisecond
)

var (
	ErrClosed       = errors.New("loopback: capturer is closed")
	ErrAlreadyStart = errors.New("loopback: capturer is already running")
)

// Config configures a WASAPI loopback capturer.
type Config struct {
	// DeviceID is reserved for a future explicit-device selection pass. Leave it
	// empty to capture the default Windows render device.
	DeviceID string

	// BufferDuration requests the shared-mode WASAPI buffer duration. Zero uses a
	// conservative default chosen by the internal WASAPI layer.
	BufferDuration time.Duration

	// EventBuffer controls the capacity of the Frames channel. If zero or
	// negative, a small default buffer is used. Frames are dropped when the channel
	// is full so the capture thread is not blocked by user code.
	EventBuffer int
}

// Frame contains one captured PCM packet.
type Frame struct {
	Data      []byte
	Format    audio.Format
	Frames    int
	Timestamp time.Time
	Silent    bool
}

// Capturer owns a WASAPI loopback capture stream for the default render device.
//
// All WASAPI objects live on one locked OS thread. Public methods communicate
// with that thread through an internal command queue.
type Capturer struct {
	cmd     chan command
	frames  chan Frame
	errors  chan error
	done    chan struct{}
	format  audio.Format
	closeMu sync.Mutex
	closed  bool
}

type commandKind int

const (
	commandStart commandKind = iota
	commandStop
	commandClose
)

type command struct {
	kind commandKind
	resp chan error
}

type initResult struct {
	format audio.Format
	err    error
}

// New creates a capturer for the default Windows output mix.
func New(cfg *Config) (*Capturer, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.DeviceID != "" {
		return nil, fmt.Errorf("loopback: explicit DeviceID is not implemented yet")
	}
	bufferDuration100ns := durationToReferenceTime(cfg.BufferDuration)
	frameBuffer := cfg.EventBuffer
	if frameBuffer <= 0 {
		frameBuffer = defaultFrameBuffer
	}

	c := &Capturer{
		cmd:    make(chan command),
		frames: make(chan Frame, frameBuffer),
		errors: make(chan error, defaultErrorBuffer),
		done:   make(chan struct{}),
	}
	initCh := make(chan initResult, 1)
	go c.worker(bufferDuration100ns, initCh)

	init := <-initCh
	if init.err != nil {
		return nil, init.err
	}
	c.format = init.format
	return c, nil
}

// Start starts reading PCM packets from the loopback stream.
func (c *Capturer) Start() error {
	return c.send(commandStart)
}

// Frames returns captured PCM packets. The channel closes when the capturer is
// closed or when initialization fails.
func (c *Capturer) Frames() <-chan Frame { return c.frames }

// Errors returns asynchronous capture-loop errors. The channel closes when the
// capturer is closed.
func (c *Capturer) Errors() <-chan error { return c.errors }

// Stop stops capture. The capturer can be started again after Stop succeeds.
func (c *Capturer) Stop() error {
	return c.send(commandStop)
}

// Close stops capture, releases WASAPI objects on their owning thread, and
// closes the frame and error channels. Close is idempotent.
func (c *Capturer) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	c.closeMu.Unlock()

	err := c.send(commandClose)
	<-c.done
	return err
}

// Format returns the WASAPI mix format used by the capturer.
func (c *Capturer) Format() audio.Format {
	if c == nil {
		return audio.Format{}
	}
	return c.format
}

func (c *Capturer) send(kind commandKind) error {
	if c == nil {
		return ErrClosed
	}
	select {
	case <-c.done:
		return ErrClosed
	default:
	}
	resp := make(chan error, 1)
	cmd := command{kind: kind, resp: resp}
	select {
	case c.cmd <- cmd:
		return <-resp
	case <-c.done:
		return ErrClosed
	}
}

func (c *Capturer) worker(bufferDuration100ns int64, initCh chan<- initResult) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.done)
	defer close(c.frames)
	defer close(c.errors)

	if err := wasapi.Init(); err != nil {
		initCh <- initResult{err: err}
		return
	}
	defer wasapi.Uninit()

	loopback, err := wasapi.OpenDefaultLoopback(bufferDuration100ns)
	if err != nil {
		initCh <- initResult{err: err}
		return
	}
	defer loopback.Close()

	initCh <- initResult{format: loopback.Format().Format}
	c.commandLoop(loopback)
}

func (c *Capturer) commandLoop(loopback *wasapi.LoopbackCapture) {
	running := false
	for {
		if !running {
			cmd := <-c.cmd
			switch cmd.kind {
			case commandStart:
				err := loopback.Start()
				if err == nil {
					running = true
				}
				cmd.resp <- err
			case commandStop:
				cmd.resp <- nil
			case commandClose:
				cmd.resp <- nil
				return
			}
			continue
		}

		select {
		case cmd := <-c.cmd:
			switch cmd.kind {
			case commandStart:
				cmd.resp <- ErrAlreadyStart
			case commandStop:
				cmd.resp <- loopback.Stop()
				running = false
			case commandClose:
				cmd.resp <- loopback.Stop()
				return
			}
		default:
			packet, ok, err := loopback.ReadPacket()
			if err != nil {
				c.sendError(err)
				_ = loopback.Stop()
				running = false
				continue
			}
			if !ok {
				time.Sleep(defaultPollDelay)
				continue
			}
			c.sendFrame(packet, loopback.Format().Format)
		}
	}
}

func (c *Capturer) sendFrame(packet wasapi.Packet, format audio.Format) {
	frame := Frame{
		Data:      packet.Data,
		Format:    format,
		Frames:    packet.Frames,
		Timestamp: packet.Timestamp,
		Silent:    packet.Silent(),
	}
	select {
	case c.frames <- frame:
	default:
	}
}

func (c *Capturer) sendError(err error) {
	select {
	case c.errors <- err:
	default:
	}
}

func durationToReferenceTime(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64(d / (100 * time.Nanosecond))
}
