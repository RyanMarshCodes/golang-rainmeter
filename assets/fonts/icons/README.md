# Icon fonts

Widgets render icons as glyphs from whatever font you set as `icon_font`.
Logical names (`music`, `cloud`, `gpu`, …) are resolved through `icon-map.json`.

## Quick start (Font Awesome Free)

1. Download [Font Awesome Free](https://fontawesome.com/download) (or use any icon TTF/OTF).
2. Copy the Solid (or Thin) binary into this folder, e.g.:

   `assets/fonts/icons/fa-solid.otf`

3. Point config at it:

```yaml
icon_map: fonts/icons/icon-map.json   # optional; this is the default
widgets:
  - type: metrics
    icon_font: fonts/icons/fa-solid.otf
    measures:
      - kind: cpu
        icon: computer          # name from icon-map.json
      - kind: gpu
        icon: gpu               # mapped to free microchip by default
        # icon_code: f2db       # optional: skip the map, use hex directly
```

## Using a different icon pack

1. Drop your `.ttf` / `.otf` under `assets/fonts/icons/` (or anywhere under `assets/`).
2. Edit `icon-map.json` so each logical name’s hex matches **that** font’s codepoints.
3. Set `icon_font` to your file.

`icon_code` in config always wins over the map, so you can override one glyph without editing JSON.

## Note on Font Awesome Pro

Pro fonts are licensed and must not be committed. Keep them outside the repo or under the gitignored `assets/fonts/fontawesome/` path if you use them locally.
