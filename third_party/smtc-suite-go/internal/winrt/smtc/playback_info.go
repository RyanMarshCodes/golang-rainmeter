//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionPlaybackInfo vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6]  get_Controls(out *IPlaybackControls*) HRESULT
//	[7]  get_PlaybackStatus(out *i32) HRESULT
//	[8]  get_PlaybackType(out *IReference<MediaPlaybackType>*) HRESULT (nullable)
//	[9]  get_AutoRepeatMode(out *IReference<AutoRepeatMode>*) HRESULT (nullable)
//	[10] get_PlaybackRate(out *IReference<f64>*) HRESULT (nullable)
//	[11] get_IsShuffleActive(out *IReference<bool>*) HRESULT (nullable)

const (
	Slot_PlaybackInfo_Controls        = 6
	Slot_PlaybackInfo_PlaybackStatus  = 7
	Slot_PlaybackInfo_PlaybackType    = 8
	Slot_PlaybackInfo_AutoRepeatMode  = 9
	Slot_PlaybackInfo_PlaybackRate    = 10
	Slot_PlaybackInfo_IsShuffleActive = 11
)
