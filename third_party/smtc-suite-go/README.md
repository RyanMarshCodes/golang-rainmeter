# smtc-suite-go

Go bindings for Windows System Media Transport Controls (SMTC), implemented with CGo and raw WinRT COM interop.

Current release: `v0.2.0`. The monitor and control modules are usable; create
and audio loopback are available as experimental modules.

## Module Status

| Module | Status | Notes |
|---|---|---|
| `pkg/smtc/monitor` | Usable | Enumerates system media sessions, reads metadata/timeline/playback state, exposes manager and session events. |
| `pkg/smtc/control` | Usable | Controls existing media sessions with play/pause/skip/seek/shuffle/repeat. Playback rate needs a CGo ABI helper. |
| `pkg/smtc/create` | Experimental | Publishes a MediaPlayer-backed SMTC session with metadata, timeline, artwork, enabled buttons, and button events. |
| `pkg/audio/loopback` | Experimental | Captures the default Windows output mix with WASAPI loopback and exposes realtime PCM frames. |

## Feature Matrix

| Area | Monitor | Control | Create |
|---|---:|---:|---:|
| Enumerate sessions | Yes | Yes, for lookup | N/A |
| Current session | Yes | Default target | N/A |
| Media title/artist/album | Yes | Read current target | Publish title/artist |
| Cover artwork | Read bytes/hash | Read bytes/hash | Publish file or URI reference |
| Playback status | Yes | Via commands | Publish |
| Timeline position/duration | Yes | Seek | Publish |
| Playback controls/capabilities | Yes | Not exposed yet | Common buttons |
| Shuffle/repeat/playback rate state | Yes | Set | Playback rate publish |
| Button events | N/A | N/A | Yes |

## Audio Capture

| Area | Loopback |
|---|---:|
| Default output mix capture | Experimental |
| PCM frame stream | Experimental |
| WAV capture example | Experimental |
| Realtime level meter | Experimental |
| SMTC + audio meter example | Experimental |
| Per-app capture | Not planned yet |

## Known Limitations

- This project uses raw COM vtable slots. Wrong slots can crash the process, so new WinRT APIs should be added conservatively.
- COM/WinRT initialization is still thread-sensitive. A future runtime should centralize calls on dedicated apartment threads.
- Monitor metadata is best-effort. Some apps do not expose artwork, playback rate, repeat, or shuffle state.
- Control operations can be rejected by the target media app even when the async call completes successfully.
- Create is experimental. Windows Shell behavior varies by Windows version, especially for local file artwork.
- Create currently enables only common buttons: play, pause, stop, next, and previous.
- Audio loopback is experimental, captures the default output mix only, and does
  not yet support explicit device selection or automatic device-switch handling.
- There are no automated WinRT integration tests yet; examples are the primary runtime verification path.
- See `docs/com-runtime-plan.md` for the planned dedicated apartment runtime refactor.
- See `docs/audio-loopback-plan.md` for the planned experimental WASAPI system
  audio capture module.

## Verified Runtime

The monitor, control, and create examples have been manually verified on a real
Windows desktop after the current raw COM vtable and CGo ABI changes. For the
full smoke-test checklist, see `docs/manual-test.md`.

See `CHANGELOG.md` for the `v0.2.0` release notes.

## Requirements

| Requirement | Details |
|---|---|
| OS | Windows 10 build 17763+ |
| Go | 1.25+ |
| C compiler | MinGW-w64 (GCC) or MSVC-compatible CGo toolchain |
| Windows SDK | 10.0.17763+ |
| CGo | `CGO_ENABLED=1` |

Non-Windows and non-CGo builds expose stub packages that return `smtc.ErrUnsupported`.

The audio loopback module exposes experimental WASAPI capture on Windows/CGo and
unsupported stubs on other builds.

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

func main() {
    mgr, err := monitor.New(nil)
    if err != nil {
        panic(err)
    }
    defer mgr.Close()

    for _, s := range mgr.Sessions() {
        fmt.Println(s.SourceAppUserModelID, s.MediaInfo.Title, s.PlaybackStatus)
    }
}
```

## Examples

```powershell
go run ./examples/monitor
go run ./examples/control info
go run ./examples/control toggle
go run ./examples/control seek 30s
go run ./examples/create
go run ./examples/audio-meter -duration 10s
go run ./examples/audio-capture -duration 10s -out testdata/loopback.wav
go run ./examples/now-playing-meter -duration 10s
```

More manual verification commands are documented in `docs/manual-test.md`.

## Implementation Notes

- The project is Windows-only at runtime and uses raw COM vtable calls instead of a C++/WinRT bridge DLL.
- Monitor and control use MTA initialization; create uses the modern `Windows.Media.Playback.MediaPlayer` SMTC path.
- WinRT async operations are completed with `put_Completed` handlers and Go channels.
- Some advanced SMTC methods use floating-point parameters and require CGo wrappers to pass values through XMM registers on Windows x64.
- Audio loopback uses WASAPI shared-mode loopback on the default render endpoint
  and sends Go-owned PCM frame copies through `pkg/audio/loopback` channels.

## License

MIT
