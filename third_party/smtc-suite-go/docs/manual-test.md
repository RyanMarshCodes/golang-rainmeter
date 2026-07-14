# Manual Verification

These checks exercise the real Windows SMTC/WinRT path. Run them on Windows with
`CGO_ENABLED=1` and a working C compiler.

## Monitor

1. Start a media app such as Apple Music, Spotify, or a browser playing media.
2. Run:

   ```powershell
   go run ./examples/monitor -duration 60s
   ```

3. Change tracks, pause/play, seek, and toggle shuffle/repeat in the media app.
4. Confirm the example prints session, playback, timeline, and media events.
5. Confirm cover files appear in `testdata/` when artwork changes.

Useful options:

```powershell
go run ./examples/monitor -duration 2m -covers-dir testdata
go run ./examples/monitor -no-cover-save
```

## Control

Start media playback, then run commands against the current session:

```powershell
go run ./examples/control info
go run ./examples/control toggle
go run ./examples/control seek 30s
go run ./examples/control next
go run ./examples/control shuffle on
go run ./examples/control repeat track
go run ./examples/control rate 1.25
```

If multiple sessions are active, list them and target one explicitly:

```powershell
go run ./examples/control sessions
go run ./examples/control -session <app-user-model-id> pause
```

Some commands may return `operation rejected` when the target app does not
support that operation.

## Create

Run the create example and open the Windows media overlay:

```powershell
go run ./examples/create
```

Verify:

- The custom session appears in the Windows media overlay.
- Title, artist, playback status, and timeline update.
- Play, pause, stop, next, and previous buttons produce console events.
- Passing an artwork URL displays cover art more reliably than local files:

  ```powershell
  go run ./examples/create https://example.com/cover.jpg
  ```

## Audio Loopback (Experimental)

The public loopback package is available starting in Phase 2. The WAV capture
example is available starting in Phase 3. The realtime meter is available
starting in Phase 4.

During Phase 1, use the internal smoke helper instead:

```powershell
go run ./examples/audio-smoke -duration 5s
```

During Phase 2, `pkg/audio/loopback` can be used by custom callers. During Phase
3, use `audio-capture` to verify WAV output. During Phase 4, use `audio-meter`
to verify realtime level changes.

1. Start music or video playback through the default Windows output device.
2. Run the realtime meter:

   ```powershell
   go run ./examples/audio-meter -duration 10s
   ```

3. Confirm the level changes with playback volume and drops when playback is
   paused or muted.
4. Record a short WAV file:

   ```powershell
   go run ./examples/audio-capture -duration 10s -out testdata/loopback.wav
   ```

5. Confirm the WAV file is playable, has roughly the requested duration, and
   contains the system output.
6. Run the combined SMTC/audio meter:

   ```powershell
   go run ./examples/now-playing-meter -duration 10s
   ```

7. Confirm it prints current SMTC metadata alongside changing audio levels.
8. Repeat with built-in output and, when available, Bluetooth or virtual audio
   devices.
9. Switch the default output device while capturing and confirm the documented
   first-version limitation: recreate the capturer after device changes.

## Release Checklist

Before tagging a release, run:

```powershell
go test ./...
$env:CGO_ENABLED=0; go test ./pkg/smtc/... ./pkg/audio/...
```

Then repeat the monitor/control/create smoke checks on a real Windows desktop.
Once audio loopback is implemented, include the audio-meter and audio-capture
smoke checks as well.

Recommended release steps:

1. Confirm `git status --short` contains only intended changes.
2. Confirm `CHANGELOG.md` has the correct version and date.
3. Run the automated checks above.
4. Run the monitor, control, and create smoke checks above.
5. Tag the release after all checks pass.
