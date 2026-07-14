//go:build windows && cgo

package smtc

// ISystemMediaTransportControls vtable layout for the modern WinRT
// Windows.Media.SystemMediaTransportControls interface.
//
// Inherits from IInspectable (slots 0-5). Methods in WinRT metadata order:
//
//	[6]  get_PlaybackStatus(out *MediaPlaybackStatus)
//	[7]  put_PlaybackStatus(in MediaPlaybackStatus)
//	[8]  get_DisplayUpdater(out *ISystemMediaTransportControlsDisplayUpdater*)
//	[9]  get_SoundLevel(out *SoundLevel)
//	[10] get_IsEnabled(out *bool)
//	[11] put_IsEnabled(in bool)
//	[12] get_IsPlayEnabled(out *bool)
//	[13] put_IsPlayEnabled(in bool)
//	[14] get_IsStopEnabled(out *bool)
//	[15] put_IsStopEnabled(in bool)
//	[16] get_IsPauseEnabled(out *bool)
//	[17] put_IsPauseEnabled(in bool)
//	[18] get_IsRecordEnabled(out *bool)
//	[19] put_IsRecordEnabled(in bool)
//	[20] get_IsFastForwardEnabled(out *bool)
//	[21] put_IsFastForwardEnabled(in bool)
//	[22] get_IsRewindEnabled(out *bool)
//	[23] put_IsRewindEnabled(in bool)
//	[24] get_IsPreviousEnabled(out *bool)
//	[25] put_IsPreviousEnabled(in bool)
//	[26] get_IsNextEnabled(out *bool)
//	[27] put_IsNextEnabled(in bool)
//	[28] get_IsChannelUpEnabled(out *bool)
//	[29] put_IsChannelUpEnabled(in bool)
//	[30] get_IsChannelDownEnabled(out *bool)
//	[31] put_IsChannelDownEnabled(in bool)
//	[32] add_ButtonPressed(...)
//	[33] remove_ButtonPressed(...)
//	[34] add_PropertyChanged(...)
//	[35] remove_PropertyChanged(...)
//
// Source: Windows.Media.SystemMediaTransportControls WinRT metadata.

const (
	Slot_Controls_get_DisplayUpdater    = 8
	Slot_Controls_get_IsEnabled         = 10
	Slot_Controls_get_PlaybackStatus    = 6
	Slot_Controls_put_IsChannelDown     = 31
	Slot_Controls_put_IsChannelUp       = 29
	Slot_Controls_put_IsEnabled         = 11
	Slot_Controls_put_IsFastForward     = 21
	Slot_Controls_put_IsNextEnabled     = 27
	Slot_Controls_put_IsPauseEnabled    = 17
	Slot_Controls_put_IsPlayEnabled     = 13
	Slot_Controls_put_IsPreviousEnabled = 25
	Slot_Controls_put_IsRecordEnabled   = 19
	Slot_Controls_put_IsRewindEnabled   = 23
	Slot_Controls_put_IsStopEnabled     = 15
	Slot_Controls_put_PlaybackStatus    = 7
	Slot_Controls_add_ButtonPressed     = 32
	Slot_Controls_remove_ButtonPressed  = 33
	Slot_ButtonPressedArgs_get_Button   = 6
)

type MediaPlaybackStatus int32

const (
	MediaPlaybackStatusClosed   MediaPlaybackStatus = 0
	MediaPlaybackStatusChanging MediaPlaybackStatus = 1
	MediaPlaybackStatusStopped  MediaPlaybackStatus = 2
	MediaPlaybackStatusPlaying  MediaPlaybackStatus = 3
	MediaPlaybackStatusPaused   MediaPlaybackStatus = 4
)

type MediaPlaybackType int32

const (
	MediaPlaybackTypeUnknown MediaPlaybackType = 0
	MediaPlaybackTypeMusic   MediaPlaybackType = 1
	MediaPlaybackTypeVideo   MediaPlaybackType = 2
	MediaPlaybackTypeImage   MediaPlaybackType = 3
)

// ISystemMediaTransportControls2 vtable layout.
//
// Inherits from IInspectable (slots 0-5):
//
//	[6]  get_AutoRepeatMode(out *MediaPlaybackAutoRepeatMode)
//	[7]  put_AutoRepeatMode(in MediaPlaybackAutoRepeatMode)
//	[8]  get_ShuffleEnabled(out *bool)
//	[9]  put_ShuffleEnabled(in bool)
//	[10] get_PlaybackRate(out *double)
//	[11] put_PlaybackRate(in double)
//	[12] UpdateTimelineProperties(in ISystemMediaTransportControlsTimelineProperties*) HRESULT
//
// The PlaybackRate setter takes a double and must be called through the CGo ABI
// helper in internal/winrt because float arguments require XMM registers on
// Windows x64.
const (
	Slot_Controls2_put_PlaybackRate         = 11
	Slot_Controls2_UpdateTimelineProperties = 12
)

// SystemMediaTransportControlsTimelineProperties vtable layout.
//
// Inherits from IInspectable (slots 0-5):
//
//	[6]  get_StartTime(out *TimeSpan)
//	[7]  put_StartTime(in TimeSpan)
//	[8]  get_EndTime(out *TimeSpan)
//	[9]  put_EndTime(in TimeSpan)
//	[10] get_MinSeekTime(out *TimeSpan)
//	[11] put_MinSeekTime(in TimeSpan)
//	[12] get_MaxSeekTime(out *TimeSpan)
//	[13] put_MaxSeekTime(in TimeSpan)
//	[14] get_Position(out *TimeSpan)
//	[15] put_Position(in TimeSpan)
const (
	RuntimeClass_SystemMediaTransportControlsTimelineProperties = "Windows.Media.SystemMediaTransportControlsTimelineProperties"

	Slot_TimelineProps_put_StartTime   = 7
	Slot_TimelineProps_put_EndTime     = 9
	Slot_TimelineProps_put_MinSeekTime = 11
	Slot_TimelineProps_put_MaxSeekTime = 13
	Slot_TimelineProps_put_Position    = 15
)
