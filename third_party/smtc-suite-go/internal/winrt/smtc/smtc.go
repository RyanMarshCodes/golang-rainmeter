//go:build windows && cgo

// Package smtc provides raw COM vtable definitions for WinRT SMTC interfaces
// in the Windows.Media.Control namespace, plus Windows.Media interfaces for
// the Create module.
//
// All vtables start at slot 6 (after IUnknown 0-2 and IInspectable 3-5).
package smtc

// Base vtable slot offsets shared across all WinRT interfaces.
// Slots 0-2: IUnknown (QueryInterface, AddRef, Release)
// Slots 3-5: IInspectable (GetIids, GetRuntimeClassName, GetTrustLevel)
// Slots 6+:  Interface-specific methods
const (
	SlotUnknownStart  = 0
	SlotInspectStart  = 3
	SlotInterfaceStart = 6
)

// Event add/remove slots come in pairs: add at slot N, remove at slot N+1.
// The remove slot is always add slot + 1.
