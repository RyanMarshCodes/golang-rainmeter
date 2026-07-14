# Changelog

All notable changes to this project are documented in this file.

## v0.2.0 - 2026-05-17

Experimental Windows audio loopback release.

### Added

- Planned `pkg/audio/loopback` API skeleton for future WASAPI system-output
  capture, including shared `pkg/audio` format types and unsupported stubs.
- Detailed audio loopback roadmap in `docs/audio-loopback-plan.md`.
- Internal WASAPI loopback smoke helper at `examples/audio-smoke` for Phase 1
  verification.
- Experimental Windows/CGo implementation of `pkg/audio/loopback` with
  `New`, `Start`, `Frames`, `Errors`, `Stop`, `Close`, and `Format`.
- `examples/audio-capture` for recording the default output mix to WAV.
- `examples/audio-meter` for realtime peak/RMS output-level visualization.
- `examples/audio-meter -duration` option for bounded smoke-test runs.
- `pkg/audio.ConvertToFloat32` and `CanConvertToFloat32` helpers for normalized
  PCM sample conversion.
- `examples/now-playing-meter` combining SMTC current-session metadata with
  realtime loopback audio levels.

### Changed

- README and manual-test documentation now describe audio loopback as an
  experimental Windows/CGo module instead of a planned feature.

## v0.1.0 - 2026-05-15

Initial usable Windows SMTC release.

### Added

- Monitor API for enumerating system SMTC sessions, reading metadata, timeline,
  playback status, playback controls, shuffle/repeat/playback rate state, and
  cover artwork bytes/hash.
- Control API for targeting the current or named session and sending playback,
  seek, shuffle, repeat, and playback-rate commands.
- Create API for publishing a MediaPlayer-backed SMTC session with metadata,
  timeline, artwork, common transport buttons, playback status, and button
  events.
- Non-Windows and non-CGo public stubs that return `smtc.ErrUnsupported`.
- Examples for monitor, control, and create workflows.
- Manual verification checklist in `docs/manual-test.md`.
- COM apartment runtime refactor plan in `docs/com-runtime-plan.md`.

### Fixed

- Async completion handlers now keep per-handler state instead of a global
  singleton, allowing concurrent waits.
- Event handlers now release C heap allocations and cgo handles during close.
- Control `Seek` no longer deadlocks on the controller mutex.
- Control `Try*Async` calls consistently read and validate `IAsyncOperation<bool>`
  results.
- Create `Close` now attempts to release all owned COM resources before
  returning combined cleanup errors.
- Float64 COM calls use CGo ABI helpers so `double` arguments are passed through
  the Windows x64 floating-point registers.

### Known Limitations

- Runtime support requires Windows 10 build 17763+ with `CGO_ENABLED=1`.
- Create remains experimental because Windows Shell behavior can vary across
  Windows versions, especially for local artwork files.
- Advanced create buttons are intentionally disabled until their slot behavior is
  verified.
- COM initialization is still thread-sensitive; see `docs/com-runtime-plan.md`.
