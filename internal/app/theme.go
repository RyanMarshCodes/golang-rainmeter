package app

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// skinTheme uses a fully transparent clear so GLFW framebuffer alpha can
// composite widgets onto the desktop without color-key fringing.
type skinTheme struct {
	base fyne.Theme
}

func newSkinTheme() fyne.Theme {
	return &skinTheme{base: theme.DefaultTheme()}
}

func (t *skinTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground, theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground,
		theme.ColorNameInputBackground, theme.ColorNameButton, theme.ColorNameHeaderBackground,
		theme.ColorNameShadow, theme.ColorNameSeparator:
		return color.NRGBA{A: 0}
	default:
		return t.base.Color(name, variant)
	}
}

func (t *skinTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *skinTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *skinTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}
