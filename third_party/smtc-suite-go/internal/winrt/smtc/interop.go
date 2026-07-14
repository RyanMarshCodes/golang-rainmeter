//go:build windows && cgo

package smtc

// ISystemMediaTransportControlsInterop vtable layout.
//
// This COM interop interface allows desktop apps to obtain an
// ISystemMediaTransportControls instance bound to a window.
//
// Vtable slots (after IInspectable at 0-5):
//
//	[6] GetForWindow(HWND appWindow, REFIID riid, void **mediaTransportControl) HRESULT
const (
	Slot_Interop_GetForWindow = 6
)
