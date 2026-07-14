//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionTimelineProperties vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6]  get_StartTime(out *i64) HRESULT           (TimeSpan = 100ns ticks)
//	[7]  get_EndTime(out *i64) HRESULT             (TimeSpan)
//	[8]  get_MinSeekTime(out *i64) HRESULT         (TimeSpan)
//	[9]  get_MaxSeekTime(out *i64) HRESULT         (TimeSpan)
//	[10] get_Position(out *i64) HRESULT            (TimeSpan)
//	[11] get_LastUpdatedTime(out *i64) HRESULT     (DateTime)

const (
	Slot_Timeline_StartTime       = 6
	Slot_Timeline_EndTime         = 7
	Slot_Timeline_MinSeekTime     = 8
	Slot_Timeline_MaxSeekTime     = 9
	Slot_Timeline_Position        = 10
	Slot_Timeline_LastUpdatedTime = 11
)
