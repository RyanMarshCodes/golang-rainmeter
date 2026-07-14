//go:build windows && cgo

// Package winrt provides low-level Windows Runtime COM interop primitives
// used by the smtc-suite-go library. It wraps RoInitialize, RoGetActivationFactory,
// HSTRING management, WinRT COM vtable dispatch, async operations, and typed event handlers.
//
// This package is internal and not intended for direct use.
package winrt

// #cgo LDFLAGS: -lole32 -lruntimeobject
import "C"
