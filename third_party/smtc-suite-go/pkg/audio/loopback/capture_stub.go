//go:build !windows || !cgo

package loopback

import (
	"errors"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
)

var (
	ErrClosed       = errors.New("loopback: capturer is closed")
	ErrAlreadyStart = errors.New("loopback: capturer is already running")
)

// Config configures a WASAPI loopback capturer on Windows builds.
type Config struct {
	DeviceID       string
	BufferDuration time.Duration
	EventBuffer    int
}

// Frame contains one captured PCM packet.
type Frame struct {
	Data      []byte
	Format    audio.Format
	Frames    int
	Timestamp time.Time
	Silent    bool
}

// Capturer is a stub on unsupported platforms.
type Capturer struct{}

// New returns smtc.ErrUnsupported on unsupported platforms.
func New(cfg *Config) (*Capturer, error) { return nil, smtc.ErrUnsupported }

// Start returns smtc.ErrUnsupported on unsupported platforms.
func (c *Capturer) Start() error { return smtc.ErrUnsupported }

// Frames returns a closed channel on unsupported platforms.
func (c *Capturer) Frames() <-chan Frame {
	ch := make(chan Frame)
	close(ch)
	return ch
}

// Errors returns a closed channel on unsupported platforms.
func (c *Capturer) Errors() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

// Stop returns smtc.ErrUnsupported on unsupported platforms.
func (c *Capturer) Stop() error { return smtc.ErrUnsupported }

// Close is a no-op on unsupported platforms.
func (c *Capturer) Close() error { return nil }

// Format returns an empty format on unsupported platforms.
func (c *Capturer) Format() audio.Format { return audio.Format{} }
