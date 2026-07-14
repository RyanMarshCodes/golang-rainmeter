//go:build !windows || !cgo

package monitor

import "github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"

// Monitor is a stub on unsupported platforms.
type Monitor struct{}

// Config configures the Monitor on Windows builds.
type Config struct {
	ManagerEventBuffer int
	WorkBuffer         int
}

// New returns smtc.ErrUnsupported on unsupported platforms.
func New(cfg *Config) (*Monitor, error) {
	return nil, smtc.ErrUnsupported
}

// Sessions returns nil on unsupported platforms.
func (m *Monitor) Sessions() []smtc.SessionInfo { return nil }

// CurrentSession returns nil on unsupported platforms.
func (m *Monitor) CurrentSession() *smtc.SessionInfo { return nil }

// Events returns a closed channel on unsupported platforms.
func (m *Monitor) Events() <-chan ManagerEvent {
	ch := make(chan ManagerEvent)
	close(ch)
	return ch
}

// Close is a no-op on unsupported platforms.
func (m *Monitor) Close() error { return nil }

// ManagerEventType classifies monitor events.
type ManagerEventType int

const (
	ManagerEventSessionsChanged ManagerEventType = iota
	ManagerEventCurrentSessionChanged
	ManagerEventSessionPlaybackChanged
	ManagerEventSessionTimelineChanged
	ManagerEventSessionMediaChanged
)

// ManagerEvent is a union type for manager and session events.
type ManagerEvent struct {
	Type             ManagerEventType
	Sessions         []smtc.SessionInfo
	CurrentSessionID string
	SessionID        string
	Session          *smtc.SessionInfo
}
