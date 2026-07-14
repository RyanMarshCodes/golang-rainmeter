package widgetx

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// NewEditBorder returns a white stroke frame for edit-mode widget bounds.
func NewEditBorder() *canvas.Rectangle {
	r := &canvas.Rectangle{
		StrokeColor: color.NRGBA{R: 255, G: 255, B: 255, A: 230},
		StrokeWidth: 2,
		FillColor:   color.Transparent,
	}
	r.Hide()
	return r
}

// LayoutEditBorder sizes the frame to the surface and toggles visibility.
func LayoutEditBorder(b *canvas.Rectangle, size fyne.Size, on bool) {
	if b == nil {
		return
	}
	b.Move(fyne.NewPos(0, 0))
	b.Resize(size)
	if on {
		b.Show()
	} else {
		b.Hide()
	}
}
