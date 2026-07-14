//go:build windows && cgo

package smtc

// IGlobalSystemMediaTransportControlsSessionManagerStatics vtable layout.
//
// This is the statics (factory) interface for GlobalSystemMediaTransportControlsSessionManager.
// Obtained by calling RoGetActivationFactory with the runtime class name
// "Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager"
// and IID_IGSMTCSessionManagerStatics.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6] RequestAsync(out *IAsyncOperation<IGSMTCSessionManager*>) HRESULT
const (
	Slot_ManagerStatics_RequestAsync = 6
)

const (
	// Runtime class name for activation.
	RuntimeClass_GlobalSystemMediaTransportControlsSessionManager = "Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager"
)
