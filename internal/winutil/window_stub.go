//go:build !windows

package winutil

import (
	"image/color"

	"fyne.io/fyne/v2"
)

// ClearColor matches the Windows clear fill (fully transparent).
func ClearColor() color.Color {
	return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
}

// ChromaKeyColor is a legacy alias for ClearColor.
func ChromaKeyColor() color.Color { return ClearColor() }

// ApplyDesktopProps is a no-op on non-Windows platforms.
func ApplyDesktopProps(w fyne.Window, x, y int, width, height int, alwaysOnTop, transparent, clickThrough bool, opacity float32) bool {
	return true
}

// Bounds returns false on non-Windows platforms.
func Bounds(w fyne.Window) (x, y, width, height int, ok bool) {
	return 0, 0, 0, 0, false
}

// ClientBounds returns false on non-Windows platforms.
func ClientBounds(w fyne.Window) (x, y, width, height int, ok bool) {
	return 0, 0, 0, 0, false
}

// CursorPos returns false on non-Windows platforms.
func CursorPos() (x, y int, ok bool) {
	return 0, 0, false
}

// SetPosition is a no-op on non-Windows platforms.
func SetPosition(w fyne.Window, x, y int) {}

// SetBounds is a no-op on non-Windows platforms.
func SetBounds(w fyne.Window, x, y, width, height int) {}

// SetClientBounds is a no-op on non-Windows platforms.
func SetClientBounds(w fyne.Window, x, y, width, height int) {}

// SetNativeChrome is a no-op on non-Windows platforms.
func SetNativeChrome(w fyne.Window, enabled bool) {}

// IsVisible returns false on non-Windows platforms.
func IsVisible(w fyne.Window) bool { return false }
