//go:build !windows || !cgo

package control

import (
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
)

// Controller is a stub on unsupported platforms.
type Controller struct{}

// New returns smtc.ErrUnsupported on unsupported platforms.
func New(sessionID string) (*Controller, error) { return nil, smtc.ErrUnsupported }

func (c *Controller) Play() error                        { return smtc.ErrUnsupported }
func (c *Controller) Pause() error                       { return smtc.ErrUnsupported }
func (c *Controller) TogglePlayPause() error             { return smtc.ErrUnsupported }
func (c *Controller) Stop() error                        { return smtc.ErrUnsupported }
func (c *Controller) Next() error                        { return smtc.ErrUnsupported }
func (c *Controller) Previous() error                    { return smtc.ErrUnsupported }
func (c *Controller) FastForward() error                 { return smtc.ErrUnsupported }
func (c *Controller) Rewind() error                      { return smtc.ErrUnsupported }
func (c *Controller) Seek(position time.Duration) error  { return smtc.ErrUnsupported }
func (c *Controller) SetPlaybackRate(rate float64) error { return smtc.ErrUnsupported }
func (c *Controller) SetShuffle(active bool) error       { return smtc.ErrUnsupported }
func (c *Controller) SetRepeatMode(mode smtc.AutoRepeatMode) error {
	return smtc.ErrUnsupported
}
func (c *Controller) MediaInfo() (smtc.MediaInfo, error) {
	return smtc.MediaInfo{}, smtc.ErrUnsupported
}
func (c *Controller) Close() error { return nil }
