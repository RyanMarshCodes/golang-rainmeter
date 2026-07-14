package smtc

import "time"

// PlaybackStatus represents the playback state of a media session.
//
// Source: GlobalSystemMediaTransportControlsSessionPlaybackStatus (Windows.Media.Control).
// Note: This is a 6-member enum, different from the older 5-member
// MediaPlaybackStatus in Windows.Media.
type PlaybackStatus int32

const (
	PlaybackStatusClosed   PlaybackStatus = 0
	PlaybackStatusOpened   PlaybackStatus = 1
	PlaybackStatusChanging PlaybackStatus = 2
	PlaybackStatusStopped  PlaybackStatus = 3
	PlaybackStatusPlaying  PlaybackStatus = 4
	PlaybackStatusPaused   PlaybackStatus = 5
)

func (s PlaybackStatus) String() string {
	switch s {
	case PlaybackStatusClosed:
		return "Closed"
	case PlaybackStatusOpened:
		return "Opened"
	case PlaybackStatusChanging:
		return "Changing"
	case PlaybackStatusStopped:
		return "Stopped"
	case PlaybackStatusPlaying:
		return "Playing"
	case PlaybackStatusPaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

// PlaybackType indicates the type of media being played.
type PlaybackType int32

const (
	PlaybackTypeUnknown PlaybackType = 0
	PlaybackTypeMusic   PlaybackType = 1
	PlaybackTypeVideo   PlaybackType = 2
	PlaybackTypeImage   PlaybackType = 3
)

// AutoRepeatMode controls the repeat behavior.
type AutoRepeatMode int32

const (
	AutoRepeatNone  AutoRepeatMode = 0
	AutoRepeatTrack AutoRepeatMode = 1
	AutoRepeatList  AutoRepeatMode = 2
)

func (m AutoRepeatMode) String() string {
	switch m {
	case AutoRepeatNone:
		return "None"
	case AutoRepeatTrack:
		return "Track"
	case AutoRepeatList:
		return "List"
	default:
		return "Unknown"
	}
}

// Button identifies a system media transport button.
type Button int32

const (
	ButtonPlay        Button = 0
	ButtonPause       Button = 1
	ButtonStop        Button = 2
	ButtonRecord      Button = 3
	ButtonFastForward Button = 4
	ButtonRewind      Button = 5
	ButtonNext        Button = 6
	ButtonPrevious    Button = 7
	ButtonChannelUp   Button = 8
	ButtonChannelDown Button = 9
)

// String returns the Windows media transport button name.
func (b Button) String() string {
	switch b {
	case ButtonPlay:
		return "Play"
	case ButtonPause:
		return "Pause"
	case ButtonStop:
		return "Stop"
	case ButtonRecord:
		return "Record"
	case ButtonFastForward:
		return "FastForward"
	case ButtonRewind:
		return "Rewind"
	case ButtonNext:
		return "Next"
	case ButtonPrevious:
		return "Previous"
	case ButtonChannelUp:
		return "ChannelUp"
	case ButtonChannelDown:
		return "ChannelDown"
	default:
		return "Unknown"
	}
}

// MediaInfo holds track/album metadata for a media session.
type MediaInfo struct {
	Title           string
	Subtitle        string
	Artist          string
	AlbumArtist     string
	AlbumTitle      string
	TrackNumber     int32
	AlbumTrackCount int32
	Genres          []string
	PlaybackType    PlaybackType
	// ThumbnailAvailable reports whether the session exposed cover artwork.
	ThumbnailAvailable bool
	ThumbnailData      []byte
	ThumbnailHash      string
}

// TimelineInfo holds playback position and timing information.
// All time values are relative to the media content.
type TimelineInfo struct {
	StartTime    time.Duration
	EndTime      time.Duration
	MinSeekTime  time.Duration
	MaxSeekTime  time.Duration
	Position     time.Duration
	PlaybackRate float64
}

// TicksToDuration converts WinRT TimeSpan (100-nanosecond ticks) to time.Duration.
func TicksToDuration(ticks int64) time.Duration {
	return time.Duration(ticks * 100)
}

// DurationToTicks converts time.Duration to WinRT TimeSpan (100-nanosecond ticks).
func DurationToTicks(d time.Duration) int64 {
	return int64(d / 100)
}

// SessionInfo holds information about a media session.
type SessionInfo struct {
	SessionID            string
	SourceAppUserModelID string
	MediaInfo            MediaInfo
	PlaybackStatus       PlaybackStatus
	PlaybackType         PlaybackType
	AutoRepeatMode       AutoRepeatMode
	PlaybackRate         float64
	IsShuffleActive      bool
	PlaybackControls     PlaybackControls
	TimelineInfo         TimelineInfo
}

// PlaybackControls reports which transport operations a session says it supports.
type PlaybackControls struct {
	Play             bool
	Pause            bool
	Stop             bool
	Record           bool
	FastForward      bool
	Rewind           bool
	Next             bool
	Previous         bool
	ChannelUp        bool
	ChannelDown      bool
	PlayPauseToggle  bool
	Shuffle          bool
	Repeat           bool
	PlaybackRate     bool
	PlaybackPosition bool
}
