//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionPlaybackControls vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
// All "get_Is*Enabled" methods return bool* (C boolean) via an out parameter.

const (
	Slot_Controls_IsPlayEnabled              = 6
	Slot_Controls_IsPauseEnabled             = 7
	Slot_Controls_IsStopEnabled              = 8
	Slot_Controls_IsRecordEnabled            = 9
	Slot_Controls_IsFastForwardEnabled       = 10
	Slot_Controls_IsRewindEnabled            = 11
	Slot_Controls_IsNextEnabled              = 12
	Slot_Controls_IsPreviousEnabled          = 13
	Slot_Controls_IsChannelUpEnabled         = 14
	Slot_Controls_IsChannelDownEnabled       = 15
	Slot_Controls_IsPlayPauseToggleEnabled   = 16
	Slot_Controls_IsShuffleEnabled           = 17
	Slot_Controls_IsRepeatEnabled            = 18
	Slot_Controls_IsPlaybackRateEnabled      = 19
	Slot_Controls_IsPlaybackPositionEnabled  = 20
)
