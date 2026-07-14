package config

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// ParseColor parses #RGB, #RRGGBB, #RRGGBBAA, or "r,g,b[,a]" (0–255).
// Empty string returns the provided default.
func ParseColor(s string, fallback color.Color) (color.Color, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback, nil
	}
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s)
	}
	return parseCSVColor(s)
}

func parseHexColor(s string) (color.Color, error) {
	h := strings.TrimPrefix(s, "#")
	var r, g, b, a uint8
	a = 255
	switch len(h) {
	case 3: // RGB
		ri, err1 := strconv.ParseUint(h[0:1]+h[0:1], 16, 8)
		gi, err2 := strconv.ParseUint(h[1:2]+h[1:2], 16, 8)
		bi, err3 := strconv.ParseUint(h[2:3]+h[2:3], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		r, g, b = uint8(ri), uint8(gi), uint8(bi)
	case 6: // RRGGBB
		ri, err1 := strconv.ParseUint(h[0:2], 16, 8)
		gi, err2 := strconv.ParseUint(h[2:4], 16, 8)
		bi, err3 := strconv.ParseUint(h[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		r, g, b = uint8(ri), uint8(gi), uint8(bi)
	case 8: // RRGGBBAA
		ri, err1 := strconv.ParseUint(h[0:2], 16, 8)
		gi, err2 := strconv.ParseUint(h[2:4], 16, 8)
		bi, err3 := strconv.ParseUint(h[4:6], 16, 8)
		ai, err4 := strconv.ParseUint(h[6:8], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return nil, fmt.Errorf("invalid hex color %q", s)
		}
		r, g, b, a = uint8(ri), uint8(gi), uint8(bi), uint8(ai)
	default:
		return nil, fmt.Errorf("invalid hex color %q (use #RGB, #RRGGBB, or #RRGGBBAA)", s)
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

func parseCSVColor(s string) (color.Color, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 && len(parts) != 4 {
		return nil, fmt.Errorf("invalid rgba %q (use r,g,b or r,g,b,a)", s)
	}
	vals := make([]uint8, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 || n > 255 {
			return nil, fmt.Errorf("invalid rgba component %q in %q", p, s)
		}
		vals[i] = uint8(n)
	}
	a := uint8(255)
	if len(vals) == 4 {
		a = vals[3]
	}
	return color.NRGBA{R: vals[0], G: vals[1], B: vals[2], A: a}, nil
}
