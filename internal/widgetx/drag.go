package widgetx

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"ryanmarsh.net/rmgo/internal/winutil"
)

// Drag tracks edit-mode window dragging for a splash surface.
type Drag struct {
	Win     fyne.Window
	Enabled bool

	dragging           bool
	grabX, grabY       int
	originX, originY   int
}

func (d *Drag) MouseDown(e *desktop.MouseEvent) {
	if d.Win == nil || !d.Enabled || e.Button != desktop.MouseButtonPrimary {
		return
	}
	cx, cy, ok := winutil.CursorPos()
	if !ok {
		return
	}
	x, y, _, _, ok := winutil.Bounds(d.Win)
	if !ok {
		return
	}
	d.dragging = true
	d.grabX, d.grabY = cx, cy
	d.originX, d.originY = x, y
}

func (d *Drag) MouseUp(*desktop.MouseEvent) { d.dragging = false }

func (d *Drag) MouseMoved(*desktop.MouseEvent) {
	if !d.dragging || !d.Enabled {
		return
	}
	cx, cy, ok := winutil.CursorPos()
	if !ok {
		return
	}
	winutil.SetPosition(d.Win, d.originX+(cx-d.grabX), d.originY+(cy-d.grabY))
}

func (d *Drag) MouseIn(*desktop.MouseEvent) {}
func (d *Drag) MouseOut()                   {}
