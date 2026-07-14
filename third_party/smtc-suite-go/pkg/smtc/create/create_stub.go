//go:build !windows || !cgo

package create

import "github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"

// Creator is a stub on unsupported platforms.
type Creator struct{}

// Config describes the initial state of a Creator on Windows builds.
type Config struct {
	MediaInfo         smtc.MediaInfo
	AppMediaID        string
	PlaybackStatus    smtc.PlaybackStatus
	Buttons           *Buttons
	TimelineInfo      smtc.TimelineInfo
	ThumbnailPath     string
	ButtonEventBuffer int
}

// Buttons describes which Windows media transport buttons are enabled.
type Buttons struct {
	Play        bool
	Pause       bool
	Stop        bool
	Next        bool
	Previous    bool
	FastForward bool
	Rewind      bool
	Record      bool
	ChannelUp   bool
	ChannelDown bool
}

func DefaultConfig() *Config { return &Config{} }

func DefaultButtons() Buttons {
	return Buttons{Play: true, Pause: true, Stop: true, Next: true, Previous: true}
}

// New returns smtc.ErrUnsupported on unsupported platforms.
func New(cfg *Config) (*Creator, error) { return nil, smtc.ErrUnsupported }

func (c *Creator) SetThumbnailFromFile(path string) error { return smtc.ErrUnsupported }
func (c *Creator) SetThumbnailFromURI(uri string) error   { return smtc.ErrUnsupported }
func (c *Creator) SetTimelineInfo(info smtc.TimelineInfo) error {
	return smtc.ErrUnsupported
}
func (c *Creator) ButtonEvents() <-chan smtc.Button {
	ch := make(chan smtc.Button)
	close(ch)
	return ch
}
func (c *Creator) SetEnabled(enabled bool) error { return smtc.ErrUnsupported }
func (c *Creator) SetEnabledButtons(buttons Buttons) error {
	return smtc.ErrUnsupported
}
func (c *Creator) SetPlaybackStatus(status smtc.PlaybackStatus) error {
	return smtc.ErrUnsupported
}
func (c *Creator) SetMediaInfo(info smtc.MediaInfo) error { return smtc.ErrUnsupported }
func (c *Creator) Close() error                           { return nil }
