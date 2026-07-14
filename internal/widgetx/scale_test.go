package widgetx

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
)

func TestRampScaleAtDesignSize(t *testing.T) {
	r := NewRamp(420, 780)
	got := r.Scale(fyne.NewSize(420, 780))
	if got != 1 {
		t.Fatalf("Scale at design size = %v, want 1", got)
	}
}

func TestRampScaleUniformMin(t *testing.T) {
	r := NewRamp(420, 780)
	// Width would be 2x but height is 1x — uniform min keeps 1x.
	got := r.Scale(fyne.NewSize(840, 780))
	if got != 1 {
		t.Fatalf("Scale wider only = %v, want 1", got)
	}
	got = r.Scale(fyne.NewSize(840, 1560))
	if got != 2 {
		t.Fatalf("Scale both axes 2x = %v, want 2", got)
	}
}

func TestRampScaleDown(t *testing.T) {
	r := NewRamp(420, 780)
	got := r.Scale(fyne.NewSize(315, 585))
	if got != 0.75 {
		t.Fatalf("Scale 75%% = %v, want 0.75", got)
	}
}

func TestRampScaleClamp(t *testing.T) {
	r := NewRamp(420, 780)
	got := r.Scale(fyne.NewSize(42, 78))
	if got != DefaultMinScale {
		t.Fatalf("Scale tiny = %v, want min %v", got, DefaultMinScale)
	}
	got = r.Scale(fyne.NewSize(4200, 7800))
	if got != DefaultMaxScale {
		t.Fatalf("Scale huge = %v, want max %v", got, DefaultMaxScale)
	}
}

func TestRampPxRounds(t *testing.T) {
	r := NewRamp(420, 780)
	got := r.Px(15, fyne.NewSize(420, 780))
	if got != 15 {
		t.Fatalf("Px at design = %v, want 15", got)
	}
	got = r.Px(15, fyne.NewSize(840, 1560))
	if got != 30 {
		t.Fatalf("Px 2x = %v, want 30", got)
	}
}

func TestNewRampDefaults(t *testing.T) {
	r := NewRamp(0, 0)
	if r.DesignW != DefaultDesignWidth || r.DesignH != DefaultDesignHeight {
		t.Fatalf("defaults: %v x %v", r.DesignW, r.DesignH)
	}
}

func TestRampFromConfigUsesBandHeight(t *testing.T) {
	r := RampFromConfig(config.WidgetConfig{
		DesignWidth:      420,
		DesignHeight:     780,
		DesignBandHeight: 210,
		Height:           210,
	})
	if r.DesignH != 210 {
		t.Fatalf("DesignH = %v, want 210", r.DesignH)
	}
	got := r.Scale(fyne.NewSize(840, 420)) // 2x width, 2x band height
	if got != 2 {
		t.Fatalf("Scale 2x band = %v, want 2", got)
	}
}

func TestRampFromConfigWidthOnly(t *testing.T) {
	r := RampFromConfig(config.WidgetConfig{
		DesignWidth:      420,
		DesignBandHeight: 210,
	})
	got := r.Scale(fyne.NewSize(840, 210)) // 2x width only
	if got != 1 {
		t.Fatalf("width-only at band height = %v, want 1 (height-limited)", got)
	}
}
