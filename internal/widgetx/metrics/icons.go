package metrics

import "ryanmarsh.net/rmgo/internal/icons"

// IconRune resolves a glyph from icon name and/or hex icon_code via the
// shared icon map (see assets/fonts/icons/icon-map.json).
func IconRune(name, codeHex string) rune {
	return icons.Rune(name, codeHex)
}
