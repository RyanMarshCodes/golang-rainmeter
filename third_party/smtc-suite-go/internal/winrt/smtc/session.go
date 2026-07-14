//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSession vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6]  GetSourceAppUserModelId(out *HSTRING) HRESULT
//	[7]  TryGetMediaPropertiesAsync(out *IAsyncOperation<IMediaProperties*>) HRESULT
//	[8]  GetTimelineProperties(out *ITimelineProperties*) HRESULT
//	[9]  GetPlaybackInfo(out *IPlaybackInfo*) HRESULT
//	[10] TryPlayAsync(out *IAsyncOperation<bool>) HRESULT
//	[11] TryPauseAsync(out *IAsyncOperation<bool>) HRESULT
//	[12] TryStopAsync(out *IAsyncOperation<bool>) HRESULT
//	[13] TryRecordAsync(out *IAsyncOperation<bool>) HRESULT
//	[14] TryFastForwardAsync(out *IAsyncOperation<bool>) HRESULT
//	[15] TryRewindAsync(out *IAsyncOperation<bool>) HRESULT
//	[16] TrySkipNextAsync(out *IAsyncOperation<bool>) HRESULT
//	[17] TrySkipPreviousAsync(out *IAsyncOperation<bool>) HRESULT
//	[18] TryChangeChannelUpAsync(out *IAsyncOperation<bool>) HRESULT
//	[19] TryChangeChannelDownAsync(out *IAsyncOperation<bool>) HRESULT
//	[20] TryTogglePlayPauseAsync(out *IAsyncOperation<bool>) HRESULT
//	[21] TryChangeAutoRepeatModeAsync(mode i32, out *IAsyncOperation<bool>) HRESULT
//	[22] TryChangePlaybackRateAsync(rate f64, out *IAsyncOperation<bool>) HRESULT
//	[23] TryChangeShuffleActiveAsync(active bool, out *IAsyncOperation<bool>) HRESULT
//	[24] TryChangePlaybackPositionAsync(position i64, out *IAsyncOperation<bool>) HRESULT
//	[25] add_TimelinePropertiesChanged(handler, out token) HRESULT
//	[26] remove_TimelinePropertiesChanged(token) HRESULT
//	[27] add_PlaybackInfoChanged(handler, out token) HRESULT
//	[28] remove_PlaybackInfoChanged(token) HRESULT
//	[29] add_MediaPropertiesChanged(handler, out token) HRESULT
//	[30] remove_MediaPropertiesChanged(token) HRESULT
//
// Source: windows-rs crate, authoritative from WinMD metadata.

const (
	Slot_Session_GetSourceAppUserModelId   = 6
	Slot_Session_TryGetMediaPropertiesAsync = 7
	Slot_Session_GetTimelineProperties     = 8
	Slot_Session_GetPlaybackInfo           = 9
	Slot_Session_TryPlayAsync              = 10
	Slot_Session_TryPauseAsync             = 11
	Slot_Session_TryStopAsync              = 12
	Slot_Session_TryRecordAsync            = 13
	Slot_Session_TryFastForwardAsync       = 14
	Slot_Session_TryRewindAsync             = 15
	Slot_Session_TrySkipNextAsync          = 16
	Slot_Session_TrySkipPreviousAsync      = 17
	Slot_Session_TryChangeChannelUpAsync   = 18
	Slot_Session_TryChangeChannelDownAsync = 19
	Slot_Session_TryTogglePlayPauseAsync   = 20
	Slot_Session_TryChangeAutoRepeatModeAsync = 21
	Slot_Session_TryChangePlaybackRateAsync   = 22
	Slot_Session_TryChangeShuffleActiveAsync  = 23
	Slot_Session_TryChangePlaybackPositionAsync = 24
	Slot_Session_add_TimelinePropertiesChanged    = 25
	Slot_Session_remove_TimelinePropertiesChanged = 26
	Slot_Session_add_PlaybackInfoChanged          = 27
	Slot_Session_remove_PlaybackInfoChanged       = 28
	Slot_Session_add_MediaPropertiesChanged       = 29
	Slot_Session_remove_MediaPropertiesChanged    = 30
)

// ---- MediaPlaybackAutoRepeatMode enum values ----

type AutoRepeatMode int32

const (
	AutoRepeatModeNone  AutoRepeatMode = 0
	AutoRepeatModeTrack AutoRepeatMode = 1
	AutoRepeatModeList  AutoRepeatMode = 2
)
