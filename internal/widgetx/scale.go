package widgetx

import (
	"fyne.io/fyne/v2"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
)

// Defaults match config.example.yml design viewport.
const (
	DefaultDesignWidth  float32 = 420
	DefaultDesignHeight float32 = 780
	DefaultMinScale     float32 = 0.75
	DefaultMaxScale     float32 = 2.0
)

// Ramp scales design-token sizes from a reference viewport to a live container.
type Ramp struct {
	DesignW  float32
	DesignH  float32
	MinScale float32
	MaxScale float32
}

// NewRamp returns a ramp with defaults filled in for zero fields.
func NewRamp(designW, designH float32) Ramp {
	r := Ramp{
		DesignW:  designW,
		DesignH:  designH,
		MinScale: DefaultMinScale,
		MaxScale: DefaultMaxScale,
	}
	if r.DesignW <= 0 {
		r.DesignW = DefaultDesignWidth
	}
	if r.DesignH <= 0 {
		r.DesignH = DefaultDesignHeight
	}
	return r
}

// Scale returns uniform min(wScale, hScale) with clamping.
func (r Ramp) Scale(container fyne.Size) float32 {
	wScale := container.Width / r.DesignW
	hScale := container.Height / r.DesignH
	s := min(wScale, hScale)
	return max(r.MinScale, min(s, r.MaxScale))
}

// Px scales a baseline by the container scale and rounds to whole pixels.
func (r Ramp) Px(base float32, container fyne.Size) float32 {
	if base <= 0 {
		return 0
	}
	v := base * r.Scale(container)
	if v < 1 {
		return 1
	}
	return float32(int(v + 0.5))
}

// Text scales a caption/body baseline.
func (r Ramp) Text(base float32, container fyne.Size) float32 {
	return r.Px(base, container)
}

// Icon scales an icon baseline.
func (r Ramp) Icon(base float32, container fyne.Size) float32 {
	return r.Px(base, container)
}

// RampFromConfig builds a ramp from widget runtime design fields.
// DesignH uses the widget's configured band height so scale compares like-with-like
// (widget container height vs its design height), not vs the full shell stack.
func RampFromConfig(cfg config.WidgetConfig) Ramp {
	designH := cfg.DesignBandHeight
	if designH <= 0 {
		designH = cfg.Height
	}
	if designH <= 0 {
		designH = cfg.DesignHeight
	}
	return NewRamp(cfg.DesignWidth, designH)
}
