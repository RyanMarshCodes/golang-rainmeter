package widgetx

// Shared layout / type ramp for splash widgets.
const (
	Pad    float32 = 4 // horizontal inset from window edge
	PadY   float32 = 4 // vertical inset from window edge
	RowGap float32 = 6 // gap between stacked rows inside a widget

	// CaptionSize + CaptionFont: secondary labels (weather details/forecast,
	// metrics values, media artist + progress times).
	CaptionSize float32 = 15
	CaptionFont         = "fonts/montserrat/static/Montserrat-Medium.ttf"

	// DisplayFont: hero/emphasis (song title, large temps use SemiBold via config).
	DisplayFont = "fonts/montserrat/static/Montserrat-SemiBold.ttf"
)
