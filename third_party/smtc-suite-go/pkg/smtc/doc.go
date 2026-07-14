// Package smtc provides Go bindings for Windows System Media Transport Controls (SMTC).
//
// Three modules are available:
//
//   - monitor: Watch system-wide media sessions (track changes, playback status, timeline)
//   - control: Control remote media sessions (play/pause, skip, seek)
//   - create:  Publish a custom media session to the Windows system UI
//
// All modules require Windows 10+ and CGo at runtime. Unsupported builds expose
// stub packages that return ErrUnsupported.
package smtc

import "errors"

// ErrUnsupported is returned by public stubs on platforms where Windows SMTC is
// unavailable.
var ErrUnsupported = errors.New("smtc-suite-go: Windows SMTC requires Windows 10+ with CGO_ENABLED=1")
