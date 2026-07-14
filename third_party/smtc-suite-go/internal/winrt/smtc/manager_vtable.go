//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionManager vtable layout.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6]  GetCurrentSession(out *IGSMTCSession*) HRESULT
//	[7]  GetSessions(out *IVectorView<IGSMTCSession*>) HRESULT
//	[8]  add_CurrentSessionChanged(handler, out token) HRESULT
//	[9]  remove_CurrentSessionChanged(token) HRESULT
//	[10] add_SessionsChanged(handler, out token) HRESULT
//	[11] remove_SessionsChanged(token) HRESULT

const (
	Slot_Manager_GetCurrentSession       = 6
	Slot_Manager_GetSessions             = 7
	Slot_Manager_add_CurrentSessionChanged    = 8
	Slot_Manager_remove_CurrentSessionChanged = 9
	Slot_Manager_add_SessionsChanged          = 10
	Slot_Manager_remove_SessionsChanged       = 11
)
