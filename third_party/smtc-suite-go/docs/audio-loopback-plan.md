# Audio Loopback Capture Plan

This note records the idea for a future audio-capture module after the initial
SMTC release. The goal is to complement SMTC metadata/control with access to the
actual audio stream currently rendered by Windows.

## Goal

Add an experimental module that captures the system output mix with WASAPI
loopback capture. The first version should focus on the default render device
and expose PCM frames to Go callers.

This is separate from SMTC:

- SMTC reports media sessions, metadata, playback state, timelines, and control
  capabilities.
- WASAPI loopback captures the real audio samples being sent to the output
  device.
- The two modules can be combined by examples, but should remain separate APIs.

## Authoritative Source Notes

Research against Microsoft Learn confirms the following constraints for the first
implementation:

- Loopback recording is done on a rendering endpoint with a shared-mode stream.
- `IAudioClient::Initialize` must use `AUDCLNT_SHAREMODE_SHARED` together with
  `AUDCLNT_STREAMFLAGS_LOOPBACK` for the system mix path.
- `IAudioClient::GetService` is used to obtain `IAudioCaptureClient` from the
  initialized render stream.
- `IAudioCaptureClient::GetBuffer` data remains valid only until the matching
  `ReleaseBuffer` call, so packets must be copied into Go-owned memory first.
- `GetBuffer` and `ReleaseBuffer` must be paired on the same thread.
- Loopback capture is supported on Windows 10 and later with event-driven
  buffering; older versions need a render-stream workaround.
- `GetMixFormat` returns the device mix format, which may be `WAVEFORMATEX` or
  `WAVEFORMATEXTENSIBLE`.
- `WAVEFORMATEX.nBlockAlign` defines the frame size in bytes and should be used
  when calculating packet sizes and WAV output.

These constraints come directly from the official loopback, audio client,
capture-client, activation, and format documentation on Microsoft Learn.

## Positioning

This should be introduced as a new experimental module rather than as part of
`pkg/smtc`:

- Suggested public package: `pkg/audio/loopback`.
- Suggested internal package: `internal/wasapi`.
- Suggested release target: the next minor release, for example `v0.2.0`.
- First scope: default output device system mix only.

The feature should be documented as experimental until it has been manually
verified across common devices such as built-in speakers, Bluetooth output,
virtual audio devices, and browser/music-player playback.

## Suggested Package Layout

```text
pkg/audio/
  types.go

pkg/audio/loopback/
  capture.go
  capture_stub.go

internal/wasapi/
  device.go
  audio_client.go
  capture_client.go
  format.go
  com.go
  c/helpers.h

examples/audio-capture/
  main.go

examples/audio-meter/
  main.go
```

Keep the implementation out of `internal/winrt` because WASAPI is classic COM,
not WinRT.

## Public API Sketch

```go
package audio

type SampleFormat int

const (
    SampleFormatUnknown SampleFormat = iota
    SampleFormatFloat32
    SampleFormatInt16
    SampleFormatInt24
    SampleFormatInt32
)

type Format struct {
    SampleRate    int
    Channels      int
    BitsPerSample int
    SampleFormat  SampleFormat
    BlockAlign    int
}
```

```go
package loopback

type Config struct {
    DeviceID       string
    BufferDuration time.Duration
    EventBuffer    int
}

type Frame struct {
    Data      []byte
    Format    audio.Format
    Frames    int
    Timestamp time.Time
    Silent    bool
}

func New(cfg *Config) (*Capturer, error)
func (c *Capturer) Start() error
func (c *Capturer) Frames() <-chan Frame
func (c *Capturer) Errors() <-chan error
func (c *Capturer) Stop() error
func (c *Capturer) Close() error
func (c *Capturer) Format() audio.Format
```

First version should return the device mix format as-is. Resampling and format
conversion can be added later if needed.

Expected first-version behavior:

- `New(nil)` opens the default output device.
- `Start` launches the capture worker.
- `Frames` returns Go-owned PCM frame data.
- `Stop` stops capture without releasing the capturer.
- `Close` stops capture, releases COM/WASAPI resources, and closes channels.
- Unsupported builds expose stubs that return the same unsupported error style
  used by the existing public packages.

## WASAPI Flow

```text
CoInitializeEx(MTA)
CoCreateInstance(MMDeviceEnumerator)
IMMDeviceEnumerator.GetDefaultAudioEndpoint(eRender, eConsole)
IMMDevice.Activate(IAudioClient)
IAudioClient.GetMixFormat()
IAudioClient.Initialize(AUDCLNT_SHAREMODE_SHARED, AUDCLNT_STREAMFLAGS_LOOPBACK, ...)
IAudioClient.GetService(IAudioCaptureClient)
IAudioClient.Start()
loop:
  IAudioCaptureClient.GetNextPacketSize()
  IAudioCaptureClient.GetBuffer()
  copy PCM bytes into Go-owned memory
  IAudioCaptureClient.ReleaseBuffer()
IAudioClient.Stop()
release COM objects
```

Start with a polling capture loop because it is easier to verify. Event-driven
capture with `AUDCLNT_STREAMFLAGS_EVENTCALLBACK` and `SetEventHandle` can be a
later optimization.

Implementation details to verify carefully:

- Every vtable slot must be checked against the official WASAPI interface order.
- `IAudioCaptureClient.GetBuffer` data must be copied into Go-owned memory
  before `ReleaseBuffer` is called.
- `GetMixFormat` returns memory allocated by COM and must be released with
  `CoTaskMemFree`.
- `WAVEFORMATEX` and `WAVEFORMATEXTENSIBLE` must be parsed conservatively.
- Silent packets should set `Frame.Silent` when `AUDCLNT_BUFFERFLAGS_SILENT` is
  present.

## Threading Model

Use a dedicated goroutine locked to one OS thread for capture:

```go
runtime.LockOSThread()
defer runtime.UnlockOSThread()

CoInitializeEx(nil, COINIT_MULTITHREADED)
defer CoUninitialize()
```

Keep `IAudioClient` and `IAudioCaptureClient` operations on that thread. This
makes COM lifetime and future event-handle waiting easier to reason about.

Public methods should communicate with the capture worker by channels or a small
command queue. User code must never run on the WASAPI thread. If callers consume
frames too slowly, the first implementation should prefer dropping frames over
blocking the capture thread, and that behavior should be documented.

Apartment choice remains a validation point during implementation. Microsoft
documents a Windows 8 note that the first use of `IAudioClient` should be on an
STA thread, so the first prototype should confirm whether the planned MTA worker
is stable on the project's Windows 10+ target. If MTA proves fragile, the
capturer should move to a dedicated STA worker instead of forcing the API shape
to change later.

## Detailed Roadmap

### Phase 0: Document and Freeze Scope

Purpose: make the feature boundaries explicit before implementation begins.

Deliverables:

- Keep this design note up to date with API shape, implementation sequence, and
  limitations.
- Add README status once implementation starts.
- Add manual verification steps before the feature is released.
- Record authoritative documentation findings from Microsoft Learn before any
  implementation work lands.
- Decide whether the first capture worker should stay MTA or move to STA after a
  minimal prototype validates the audio-client initialization path.

Decisions:

- Capture system output mix only.
- Do not capture microphone input.
- Do not capture a single application or process.
- Do not resample or convert formats in the first version.
- Do not bypass DRM or protected-content restrictions.
- Do not handle output-device switching automatically in the first version.

Done when:

- The plan includes API, implementation flow, risks, examples, tests, and
  explicit non-goals.
- `pkg/audio` contains shared format types that future examples and loopback API
  can use.
- `pkg/audio/loopback` contains unsupported stubs so package paths and API shape
  can compile before the WASAPI implementation starts.

### Phase 1: Build `internal/wasapi` Minimal Loopback

Purpose: prove that the project can open the default output device, initialize
WASAPI loopback, and copy real PCM packets.

Planned files:

```text
internal/wasapi/com.go
internal/wasapi/guid.go
internal/wasapi/device.go
internal/wasapi/audio_client.go
internal/wasapi/capture_client.go
internal/wasapi/format.go
internal/wasapi/c/helpers.h
```

Required pieces:

- COM helpers for `CoInitializeEx`, `CoUninitialize`, `CoCreateInstance`,
  `Release`, HRESULT errors, and vtable dispatch.
- GUID constants for `CLSID_MMDeviceEnumerator`, `IID_IMMDeviceEnumerator`,
  `IID_IMMDevice`, `IID_IAudioClient`, and `IID_IAudioCaptureClient`.
- WASAPI constants for `eRender`, `eConsole`, `AUDCLNT_SHAREMODE_SHARED`,
  `AUDCLNT_STREAMFLAGS_LOOPBACK`, and `AUDCLNT_BUFFERFLAGS_SILENT`.
- Device helpers for default render endpoint lookup and audio-client activation.
- Format helpers for `GetMixFormat`, `WAVEFORMATEX`, `WAVEFORMATEXTENSIBLE`,
  sample-format detection, and `CoTaskMemFree`.
- Capture helpers for `Initialize`, `GetService`, `Start`, `Stop`,
  `GetNextPacketSize`, `GetBuffer`, and `ReleaseBuffer`.

Done when:

- A temporary/internal smoke path can print the mix format.
- Playing audio produces non-empty packet reads.
- Silent packets are identified correctly.
- Packet data is copied before releasing the WASAPI buffer.
- COM objects are released on the owning thread.

Implementation note: the current prototype uses `examples/audio-smoke` as a
manual verification entry point. It opens the default render endpoint, prints
the mix format, starts loopback capture, and reports packet/byte counts.
This stays internal to the repository and is not part of the public audio API.

Key risks:

- Wrong vtable slot can crash the process.
- Wrong format parsing can corrupt WAV output or meter calculations.
- MinGW/Windows SDK header availability may require local C declarations.

### Phase 2: Add Public `pkg/audio/loopback`

Purpose: expose a small Go API that hides COM details and keeps unsafe pointers
out of user code.

Planned files:

```text
pkg/audio/types.go
pkg/audio/loopback/capture.go
pkg/audio/loopback/capture_stub.go
```

Implementation requirements:

- `Capturer` owns one capture worker locked to an OS thread.
- Public lifecycle supports `New`, `Start`, `Stop`, `Close`, `Frames`, `Errors`,
  and `Format`.
- State transitions reject invalid operations such as starting a closed capturer.
- `Close` is idempotent and must not deadlock if called while capture is active.
- Frame data must be Go-owned and safe after the next capture iteration.
- Slow consumers should not block the WASAPI worker indefinitely.
- Non-Windows or non-CGo stubs should compile and return unsupported errors.

Done when:

- Public API compiles on Windows/CGo.
- Stub API compiles with `CGO_ENABLED=0`.
- Basic lifecycle tests cover repeated start/stop/close behavior where possible.
- No public type exposes `unsafe.Pointer` or COM-specific details.

Implementation note: the first public wrapper uses one locked OS-thread worker
with a command queue. Captured frames are sent through `Frames`; if the caller is
too slow, frames are dropped instead of blocking the WASAPI thread. `Errors`
reports asynchronous capture-loop failures.

### Phase 3: Add WAV Capture Example

Purpose: verify that captured bytes represent usable audio.

Planned file:

```text
examples/audio-capture/main.go
```

Suggested command:

```powershell
go run ./examples/audio-capture -duration 10s -out testdata/loopback.wav
```

Example behavior:

- Opens the default output device.
- Captures for the requested duration.
- Writes a WAV header and captured data.
- Rewrites the final RIFF/data sizes before exit.
- Prints sample rate, channels, sample format, frame count, and byte count.

Format handling:

- Support IEEE float WAV for float32 mix formats.
- Support PCM WAV for int16 mix formats.
- If another format appears, either write only when safe or report a clear
  unsupported-for-WAV error while keeping the capture API format-neutral.

Done when:

- The generated WAV opens in common players.
- Recorded duration is close to the requested duration.
- Output contains audible system playback when music/video is playing.

Implementation note: `examples/audio-capture` now records through the public
`pkg/audio/loopback` API and writes RIFF/WAVE output for PCM integer formats and
IEEE float32 mix formats. Silent packets are written as zero-filled audio so the
output duration remains continuous.

### Phase 4: Add Realtime Meter Example

Purpose: verify realtime consumption and provide a simple starting point for
visualizers or audio analysis tools.

Planned file:

```text
examples/audio-meter/main.go
```

Suggested command:

```powershell
go run ./examples/audio-meter
```

Example behavior:

- Prints sample rate, channel count, sample format, and current level.
- Computes peak or RMS for float32 and int16 frames.
- Displays silent packets clearly.
- Refreshes at a human-readable interval such as around 100 ms.

Done when:

- Levels rise and fall with real playback.
- Pausing playback or muting output is reflected in the meter.
- CPU use remains low and the frame channel does not grow without bound.

Implementation note: `examples/audio-meter` now consumes public loopback frames,
computes peak and RMS values for float32 and integer PCM formats, and refreshes a
single-line console meter at a configurable interval.

### Phase 5: Documentation and Release Preparation

Purpose: make the experimental module discoverable and safe to use.

Documentation updates:

- Add `pkg/audio/loopback` to the README module status table.
- Add example commands for `audio-capture` and `audio-meter`.
- Add known limitations for WASAPI loopback.
- Add an Audio Loopback section to `docs/manual-test.md`.
- Add release notes to `CHANGELOG.md` when the feature is ready to ship.

Manual verification checklist:

```text
1. Start music/video playback.
2. Run audio-meter and confirm levels change with playback volume.
3. Pause playback and confirm levels drop or packets become silent.
4. Run audio-capture for 10 seconds.
5. Confirm the WAV file is playable and contains the system output.
6. Try built-in speakers/headphones and note behavior for Bluetooth/virtual devices.
7. Switch output devices and confirm the first-version limitation is documented.
```

Release checks:

```powershell
go test ./...
$env:CGO_ENABLED=0; go test ./pkg/smtc/... ./pkg/audio/...
go run ./examples/audio-meter -duration 10s
go run ./examples/audio-capture -duration 10s -out testdata/loopback.wav
```

Done when:

- Automated checks pass.
- Manual Windows smoke tests pass.
- README and manual-test docs describe experimental status and limitations.

Implementation note: Phase 5 finalized the experimental documentation and made
`audio-meter` support `-duration` so automated smoke runs terminate cleanly.

### Phase 6: Follow-up Enhancements

Possible future work after the first usable version:

1. Event-driven capture with `AUDCLNT_STREAMFLAGS_EVENTCALLBACK`,
   `IAudioClient.SetEventHandle`, and `WaitForSingleObject`.
2. Device enumeration and explicit device selection by `DeviceID`.
3. Output-device change detection and optional capturer recreation.
4. Research process-specific loopback capture on supported Windows versions as a
   separate experimental path.

Implemented in Phase 6:

- `pkg/audio.ConvertToFloat32` and `CanConvertToFloat32` convert supported PCM
  frame bytes to normalized float32 samples.
- `examples/audio-meter` uses the shared conversion helper instead of local PCM
  decoding code.
- `examples/now-playing-meter` combines SMTC current-session metadata with the
  realtime loopback level meter.

## Milestones

1. Document scope, API sketch, and roadmap.
2. Build `internal/wasapi` enough to open the default output device, read the
   mix format, start loopback capture, and copy non-empty PCM packets.
3. Add `pkg/audio/loopback` as the public wrapper with Windows/CGo build tags
   and unsupported stubs for other builds.
4. Add `examples/audio-capture` to write a short WAV file.
5. Add `examples/audio-meter` to print a realtime level meter.
6. Document the module as experimental in the README, manual test notes, and
   changelog.

## First Release Acceptance Criteria

- `loopback.New(nil)` opens the default output device on Windows with CGo.
- `Start` produces continuous `Frame` values while system audio is playing.
- `Frame.Data` is a Go-owned copy independent of WASAPI buffer lifetime.
- `Frame.Format` correctly describes sample rate, channels, sample format, and
  block alignment.
- `Stop` and `Close` do not deadlock, panic, or obviously leak COM resources.
- Unsupported stubs compile for non-Windows or non-CGo builds.
- `examples/audio-capture` can produce a playable WAV file.
- `examples/audio-meter` shows realtime level changes.
- Documentation clearly states experimental status and limitations.

## Initial Non-Goals

- Microphone input capture.
- Per-application or per-process capture.
- Audio playback, transcoding, or resampling.
- DRM/protected-content bypass.
- Automatic device-switch handling.
- Cross-platform abstraction.

## Known Limitations

- Loopback captures the system output mix, not individual SMTC sessions.
- Some protected content may be silent or unavailable.
- Exclusive-mode render streams and unusual devices may behave differently.
- Bluetooth, virtual audio devices, and spatial-audio paths may expose different
  formats and latencies.
- Device switching should initially require recreating the capturer.

## Example Combinations With SMTC

Future examples can combine the existing SMTC monitor with audio frames:

- Show current title/artist plus live output level.
- Record audio while naming files from the current SMTC metadata.
- Build a simple visualizer that uses SMTC artwork and WASAPI PCM samples.
