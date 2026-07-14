//go:build !windows

// Package winrt provides Windows Runtime COM interop primitives.
// This file provides stub implementations for non-Windows platforms.
package winrt

import "errors"

// ErrUnsupported is returned when SMTC functionality is used on a non-Windows platform.
var ErrUnsupported = errors.New("smtc-suite-go: Windows SMTC is only supported on Windows 10+")
