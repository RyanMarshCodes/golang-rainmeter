//go:build windows && cgo

package smtc

// IMediaPlayer2 vtable layout.
//
// Inherits from IInspectable (slots 0-5):
//
//	[6] get_SystemMediaTransportControls(out *ISystemMediaTransportControls*) HRESULT
//
// Source: Windows.Media.Playback.IMediaPlayer2 WinRT metadata.
const (
	Slot_MediaPlayer2_get_SystemMediaTransportControls = 6
)

const (
	Slot_MediaPlayer_put_AutoPlay = 7
	Slot_MediaPlayer_Play         = 45
	Slot_MediaPlayer_Pause        = 46
)

const (
	RuntimeClass_MediaSource       = "Windows.Media.Core.MediaSource"
	RuntimeClass_MediaPlaybackItem = "Windows.Media.Playback.MediaPlaybackItem"

	Slot_MediaPlayerSource2_put_Source        = 7
	Slot_MediaSourceStatics_CreateFromStorage = 10
	Slot_MediaPlaybackItemFactory_Create      = 6
	Slot_MediaPlaybackItem2_GetDisplayProps   = 11
	Slot_MediaPlaybackItem2_ApplyDisplayProps = 12
)

const (
	Slot_MediaItemDisplayProps_put_Type       = 7
	Slot_MediaItemDisplayProps_get_MusicProps = 8
	Slot_MediaItemDisplayProps_get_VideoProps = 9
	Slot_MediaItemDisplayProps_get_Thumbnail  = 10
	Slot_MediaItemDisplayProps_put_Thumbnail  = 11
	Slot_MediaItemDisplayProps_ClearAll       = 12
)
