# rmgo

Windows desktop overlay inspired by Rainmeter. One transparent Fyne splash window stacks widgets (audio visualizer + SMTC now-playing, clock, weather via Open-Meteo, system metrics). A system tray controls show/hide, per-widget toggles, and edit mode for positioning.

**Platform:** Windows 10/11 only for full features (WASAPI visualizer, SMTC media, layered / click-through windows). Module: `github.com/RyanMarshCodes/golang-rainmeter`.

## Table of contents

- [Prerequisites](#prerequisites)
- [Clone and run](#clone-and-run)
- [Config](#config)
- [Fonts and icons](#fonts-and-icons)
- [Media / visualizer allowlist](#media--visualizer-allowlist)
- [Project layout](#project-layout)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Prerequisites

- **Go 1.26+** (see `go.mod`)
- **Windows 10 or 11**
- **CGO_ENABLED=1** and a C toolchain (MSVC or MinGW) — SMTC uses cgo / WinRT
- **Git**

For installing Go and a C compiler on Windows (and a minimal Fyne sanity check), follow the official [Fyne Quick Start](https://docs.fyne.io/started/quick/).

## Clone and run

```powershell
git clone https://github.com/RyanMarshCodes/golang-rainmeter.git
cd golang-rainmeter

# Optional: seed local config (first run also auto-copies if missing)
Copy-Item config\config.example.yml config\config.yml
# cmd.exe: copy config\config.example.yml config\config.yml

# Required: install an icon font (see Fonts and icons)
go run ./cmd/rmgo
```

### Build and run (tray / double-click)

`go run` is fine for development. For day-to-day use, build a GUI binary (no console window) and launch it like a normal app:

```powershell
$env:CGO_ENABLED = "1"
# -H windowsgui = no black console; app lives in the system tray
go build -ldflags="-H windowsgui" -o rmgo.exe ./cmd/rmgo
```

Then either:

- Double-click `rmgo.exe` in Explorer, or
- Pin it / shortcut to Startup if you want it at login

Look for the **rmgo** tray icon (notification area). Right-click for reload, edit mode, widget toggles, and Quit. Logs from `log.Printf` will not appear in a terminal with the GUI build — use a normal `go build` (omit `-H windowsgui`) when you need console output while debugging.

Keep `config\config.yml` (and `assets\`) next to the exe, or run from the repo root so those paths resolve. Optional: pass a config path as the first argument (`.\rmgo.exe path\to\config.yml`).

## Config

- `config/config.yml` is **gitignored** (machine-local).
- Template: [`config/config.example.yml`](config/config.example.yml).
- First run copies the example to `config.yml` if the file is missing.
- Resolution prefers `config/config.yml` next to the executable, then cwd `config/config.yml`, then `config.yml`.

Key sections:

| Section | Role |
| --- | --- |
| `shell` | Overlay position/size, widget `order`, transparency, `click_through` |
| `widgets` | Per-widget type, fonts, sizes, enabled flag |
| `icon_map` | Path (under `assets/`) to name→hex JSON |
| `media_apps` | SMTC allowlist substrings on the visualizer widget |

**Weather:** set `place` to a city name (e.g. `Toronto`) or a US ZIP. Canadian postals often fail — use a city instead. Units default to `f` (`c` also supported). Uses [Open-Meteo](https://open-meteo.com/) — no API key. Legacy `zip:` still works as an alias for `place`.

**Hot-reload:** editing `config.yml` reloads the UI (fsnotify). Tray → **Reload config** does the same.

### Edit mode / tray

Tray menu (`rmgo`):

- **Reload config** — reload `config.yml` from disk
- **Enter / Exit edit mode** — turns off click-through, enables native Windows title bar + resize so you can move/size the overlay; geometry is saved back to `config.yml`
- **Show overlay** — hide/show the shell
- Per-widget checked items — enable/disable (saved to config)
- **Quit**

## Fonts and icons

### Text fonts (bundled)

Montserrat **Medium** and **SemiBold** under `assets/fonts/montserrat/` (SIL Open Font License — see `montserrat/OFL.txt`). Paths in config are relative to `assets/`, e.g. `fonts/montserrat/static/Montserrat-Medium.ttf`.

### Icon fonts (not bundled)

Icon fonts are **not** shipped (licensing). Widgets draw icons as glyphs from whatever file you set as `icon_font`. Details: [`assets/fonts/icons/README.md`](assets/fonts/icons/README.md).

1. Download [Font Awesome Free](https://fontawesome.com/download) (or any icon `.ttf` / `.otf`).
2. Save it, e.g. `assets/fonts/icons/fa-solid.otf` (icon binaries under `icons/` are gitignored).
3. Set each widget’s `icon_font` (example config already points at `fonts/icons/fa-solid.otf`).
4. Optionally edit [`assets/fonts/icons/icon-map.json`](assets/fonts/icons/icon-map.json) so logical names (`music`, `cloud`, `gpu`, …) map to your font’s hex codepoints.
5. Per-measure `icon_code` (or music `music_icon_code`) overrides the map for one glyph.

Local proprietary Font Awesome Pro belongs in gitignored `assets/fonts/fontawesome/` — do **not** commit those binaries.

## Media / visualizer allowlist

Now-playing comes from Windows SMTC via a local fork of [smtc-suite-go](https://github.com/xiaowumin-mark/smtc-suite-go) (`third_party/smtc-suite-go`). SMTC cannot filter by browser tab URL; Chromium tabs typically share one AppID.

- Allowlist with `media_apps` on the visualizer widget: case-insensitive substrings matched against AppID and common metadata (title/artist/album…). Empty list = accept any session (still subject to optional `media_ignore` denylist).
- A PWA install (e.g. YouTube Music → **Install app**) gets a distinct AppID you can pin.
- Watch the log for `media: SMTC source "…"` once per AppID to discover values.

## Privacy

- **Visualizer** — reads system audio via WASAPI loopback locally; nothing is sent over the network.
- **Now-playing** — reads Windows SMTC metadata locally (title, artist, etc.).
- **Weather** — sends your configured place name or ZIP to [Open-Meteo](https://open-meteo.com/) for geocoding and forecast.

## Project layout

```
cmd/rmgo/           # main entry
internal/
  app/              # tray, edit mode, config watch, geometry save
  config/           # YAML load/save + fsnotify
  icons/            # icon-map resolution
  media/            # SMTC now-playing (cgo/WinRT on Windows)
  audio/            # WASAPI loopback visualizer bands
  weather/          # Open-Meteo client
  sysinfo/          # CPU / GPU / RAM / net / disk
  widgetx/          # shell + clock, weather, metrics, visualizer
  winutil/          # layered window, click-through, native chrome
assets/             # fonts (paths relative to assets/)
config/             # config.example.yml (+ local config.yml)
third_party/        # vendored/patched deps (glfw, smtc-suite-go) + their licenses
```

## Troubleshooting

| Symptom | Likely fix |
| --- | --- |
| Blank / missing icons | No icon font installed, wrong `icon_font` path, or codepoints don’t match your font — see [Fonts and icons](#fonts-and-icons) |
| Config not found | Run from repo root, or place `config/config.yml` next to `rmgo.exe`; optional path arg |
| Build / SMTC fails | Ensure `CGO_ENABLED=1` and a working C toolchain |
| GPU metric shows `—` | Works without MSI Afterburner (Windows PDH fallback). Afterburner can improve accuracy when running with GPU usage monitoring enabled |
| Can’t click widgets | `shell.click_through: true` passes clicks through; use tray → edit mode (disables click-through) or set `click_through: false` |

## License

[MIT](LICENSE) — see root `LICENSE` for rmgo application code.

- **Montserrat** — SIL Open Font License (`assets/fonts/montserrat/OFL.txt`)
- **Icon fonts** — your responsibility; do not commit proprietary binaries
- **Third-party** — see license files under `third_party/` (e.g. glfw, smtc-suite-go)
