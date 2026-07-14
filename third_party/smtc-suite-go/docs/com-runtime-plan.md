# COM Apartment Runtime Plan

The current implementation initializes COM/WinRT from the goroutine that creates
`Monitor`, `Controller`, or `Creator`. This works for simple command-line use,
but COM initialization is thread-affine and Go goroutines can move between OS
threads. A dedicated runtime should make this explicit and reliable.

## Goals

- Keep all WinRT calls for an object on a known initialized OS thread.
- Pair COM initialization and uninitialization on the same thread.
- Preserve the public APIs for monitor, control, and create where possible.
- Make event callbacks safe during close and shutdown.

## Proposed Shape

1. Add an internal `Apartment` type that owns one locked OS thread.
2. Start one MTA apartment for monitor/control objects.
3. Keep a separate STA apartment with message pump only for APIs that require it.
4. Expose `Do(func() error) error` and `DoValue[T](func() (T, error))` helpers internally.
5. Store COM pointers with their owning apartment and release them on that apartment.
6. Route event callbacks to lightweight Go channels, then process object updates on the owner apartment.
7. Shut down by unregistering events, releasing COM pointers, uninitializing COM, and stopping the thread in that order.

## Migration Steps

1. Introduce `internal/winrt/apartment.go` with a minimal MTA worker.
2. Move `control.Controller` calls onto the MTA worker first; it has the smallest state surface.
3. Move `monitor.Monitor` session refresh and event update work onto the same model.
4. Decide whether `create.Creator` stays MTA/MediaPlayer-backed or uses an STA worker for future APIs.
5. Remove global COM init counters once all public objects own an apartment.
6. Add close-race tests around event unregister and async cancellation paths.

## Open Questions

- Whether a process-wide shared MTA worker is enough, or each object should own its own apartment.
- Whether monitor event callbacks can safely call `AddRef` on callback threads or should only enqueue opaque ids.
- How to surface context cancellation and operation timeouts in public APIs.
