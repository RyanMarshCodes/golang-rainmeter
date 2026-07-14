# smtc-suite-go

Go library for Windows System Media Transport Controls (SMTC) via CGo + raw COM interop.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  Examples (examples/)                                 │
│  monitor/  control/  create/  debug/                  │
├──────────────────────────────────────────────────────┤
│  Public API (pkg/smtc/)                               │
│  types.go  ← 共享类型                                 │
│  monitor/  ← Phase 1: 监听                            │
│  control/  ← Phase 2: 控制                            │
│  create/   ← Phase 3: 创建 (WIP)                      │
├──────────────────────────────────────────────────────┤
│  WinRT Interop (internal/winrt/)                      │
│  init.go       RoInitialize / COM init                │
│  hstring.go    HSTRING management                     │
│  activation.go RoGetActivationFactory                 │
│  com.go        COM vtable dispatch helpers            │
│  async.go      IAsyncOperation<T> Wait (event-driven) │
│  event.go      ITypedEventHandler bridge (skeleton)   │
│  iids.go       All interface GUIDs                    │
│  smtc/         SMTC-specific vtable slot constants    │
├──────────────────────────────────────────────────────┤
│  C Helpers (internal/winrt/c/helpers.h)               │
│  WinRT type declarations (MinGW lacks WinRT headers)  │
├──────────────────────────────────────────────────────┤
│  Windows Runtime (COM)                                │
│  runtimeobject.dll, combase.dll, kernel32.dll         │
└──────────────────────────────────────────────────────┘
```

## Key Technical Decisions

| Decision | Choice | Reason |
|---|---|---|
| Interop approach | CGo + raw COM vtable | Windows-only, no C++/WinRT bridge DLL needed |
| Event waiting | `put_Completed` + Go channel callback | async never completes without proper handler IID |
| Handler allocation | `HeapAlloc` (process heap) | WinRT may validate handler provenance |
| Handler vtable | `syscall.NewCallback` | Go-callable C function pointers, go-libnp pattern |
| Thread model (Monitor/Control) | MTA (`CoInitializeEx`) | Multi-threaded, no message pump required |
| Thread model (Create) | STA + dedicated OS thread + message pump | Required by `ISystemMediaTransportControlsInterop::GetForWindow` |
| Platform restriction | `//go:build windows && cgo` | Windows-only; stubs for other platforms |

## Module Progress

### Phase 1 — Monitor (DONE)

**Package**: `pkg/smtc/monitor/`

Monitor system-wide SMTC sessions via `GlobalSystemMediaTransportControlsSessionManager`.

```
mgr, _ := monitor.New(nil)
sessions := mgr.Sessions()
for _, s := range sessions {
    fmt.Println(s.MediaInfo.Title, s.MediaInfo.Artist, s.PlaybackStatus)
}
```

Working features:
- Enumerate all sessions (app ID, playback status, timeline)
- Read media properties (title, artist, album, track number)
- Read timeline (position, duration)
- Current session detection

Verified: Apple Music, Spotify, browsers all detected.

Key bugs fixed:
- `IAsyncOperation` vtable layout: uses go-libnp layout (PutCompleted=6, GetCompleted=7, GetResults=8), NOT standard IInspectable-based. Confirmed by runtime vtable scan.
- Handler IID: must respond with parameterized GUID to `put_Completed` QI. IIDs sourced from go-libnp: `{10F0074E-...}` (SessionManager), `{84593A3D-...}` (MediaProperties). Pure Go handler uses IID capture for arbitrary types.
- `IVectorView` vtable layout: GetAt=6, get_Size=7 (from runtime scan).

### Phase 2 — Control (DONE)

**Package**: `pkg/smtc/control/`

Control remote SMTC sessions.

```
ctrl, _ := control.New("")  // current session
ctrl.Play()
ctrl.Pause()
ctrl.Next()
ctrl.Seek(30 * time.Second)
```

Working operations:
- Play, Pause, Stop, TogglePlayPause
- Next, Previous
- Seek (change playback position)
- FastForward, Rewind
- SetShuffle, SetRepeatMode

Known limitation:
- `SetPlaybackRate(float64)`: `syscall.SyscallN` passes float in integer registers (not XMM), so rate parameter is corrupted. Requires CGo wrapper.

### Phase 3 — Create (EXPERIMENTAL)

**Package**: `pkg/smtc/create/`

Create a custom SMTC session visible in Windows media overlay.

What works:
- Modern `Windows.Media.Playback.MediaPlayer` activation
- `IMediaPlayer2::get_SystemMediaTransportControls` returns modern SMTC
- Enable/disable SMTC and common transport buttons
- Publish playback status, music title/artist, timeline, and thumbnail references
- Receive `ButtonPressed` events for enabled common buttons

Current limitations:
- `PlaybackRate` is not written yet where the WinRT setter takes a `double` unless a CGo wrapper is used.
- Advanced buttons (fast-forward, rewind, record, channel up/down) are intentionally rejected until their behavior is verified.
- Windows Shell may ignore local file artwork for MediaPlayer-backed sessions; URI artwork is more reliable.

## Root Cause of Earlier Create Failure

Microsoft deprecated the legacy COM `ISystemMediaTransportControls` (from `Windows.Media.SystemMediaTransportControls.h`) in favor of the modern WinRT `SystemMediaTransportControls` class (`Windows.Media.SystemMediaTransportControls`).

The modern approach for desktop apps uses:
1. `RoActivateInstance("Windows.Media.Playback.MediaPlayer")` → get `MediaPlayer`
2. QueryInterface for `IMediaPlayer2`
3. `get_SystemMediaTransportControls` → returns modern WinRT version

This modern SMTC object integrates with the Windows 10/11 media overlay. The legacy COM interface does not.

## Build Requirements

| Requirement | Details |
|---|---|
| Go | 1.25+ |
| OS | Windows 10 17763+ |
| C Compiler | MinGW-w64 (GCC) |
| CGo | `CGO_ENABLED=1` |
| Linker flags | `-lole32 -lruntimeobject` |

## Key Lessons Learned

1. **Don't assume IInspectable**: `IAsyncOperation<T>` vtable on Win11 26200 includes IInspectable, but method order differs from WinMD documentation. Always verify with runtime vtable scan.

2. **Parameterized IIDs**: `IAsyncOperationCompletedHandler<T>` has a unique IID per T. Must either capture at runtime or source from reference implementations (go-libnp).

3. **Legacy COM ≠ Modern WinRT**: `ISystemMediaTransportControls` (C++) is the old interface. Modern SMTC uses `SystemMediaTransportControls` (WinRT). They share similar names but different vtable layouts and system integration.

4. **Vtable slot discovery is dangerous**: Wrong slot → wrong function → wrong args → SIGSEGV. Cannot recover in Go. Need authoritative IDL source or reference implementation.

5. **Go channel > C blocking**: For async wait, blocking on a Go channel (go-libnp pattern) keeps the Go runtime active and allows CGo callbacks to be processed. Blocking on C calls like `WaitForSingleObject` prevents callback delivery.

## Next Steps for Create Module

1. Add CGo wrappers for methods that take `double` arguments.
2. Verify and enable advanced button capability slots.
3. Expand display metadata beyond title/artist once slot behavior is verified.
4. Add integration tests or manual test scripts for Windows overlay behavior.
