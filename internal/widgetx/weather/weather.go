package weather

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/icons"
	wx "github.com/RyanMarshCodes/golang-rainmeter/internal/weather"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/winutil"
)

func init() {
	widgetx.Register("weather", New)
}

const (
	defaultIconFont  = "fonts/icons/fa-solid.otf"
	defaultLabelFont = widgetx.CaptionFont
	defaultBoldFont  = "fonts/montserrat/static/Montserrat-SemiBold.ttf"
	defaultInterval  = 600_000 // 10 minutes
	forecastDays     = 4
)

var (
	colorWhite    = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	defaultAccent = color.NRGBA{R: 0x7E, G: 0xC8, B: 0xE3, A: 255}
)

// Weather is a XENIUM-style current + 4-day forecast panel.
type Weather struct {
	id      string
	win     fyne.Window
	surface *surface
	client  *wx.Client
	kick    chan struct{}

	mu     sync.Mutex
	cfg    config.WidgetConfig
	cancel context.CancelFunc
}

// New creates the weather widget panel (no personal window).
func New(_ fyne.App, cfg config.WidgetConfig) (widgetx.Instance, error) {
	return &Weather{
		id:      cfg.ID,
		surface: newSurface(),
		client:  wx.NewClient(),
		cfg:     cfg,
		kick:    make(chan struct{}, 1),
	}, nil
}

func (w *Weather) ID() string   { return w.id }
func (w *Weather) Type() string { return "weather" }

func (w *Weather) Content() fyne.CanvasObject { return w.surface }

func (w *Weather) SetHost(win fyne.Window) {
	w.win = win
	w.surface.drag.Win = win
}

func (w *Weather) FlexWeight() float32 {
	w.mu.Lock()
	h := w.cfg.Height
	w.mu.Unlock()
	if h <= 0 {
		return 210
	}
	return h
}

func (w *Weather) MinSize() fyne.Size { return w.surface.MinSize() }

func (w *Weather) Start(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.startPollLocked(ctx)
}

func (w *Weather) Close() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}

func (w *Weather) Apply(cfg config.WidgetConfig) error {
	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()

	w.surface.SetDraggable(cfg.EditMode)

	if err := w.surface.applyStyle(cfg); err != nil {
		log.Printf("weather %q style: %v", cfg.ID, err)
	}
	w.surface.layoutAll(w.surface.Size())
	w.requestPoll()
	return nil
}

func (w *Weather) requestPoll() {
	select {
	case w.kick <- struct{}{}:
	default:
	}
}

func (w *Weather) startPollLocked(parent context.Context) {
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	go w.pollLoop(ctx)
}

func (w *Weather) pollLoop(ctx context.Context) {
	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	resetTicker := func() {
		w.mu.Lock()
		ms := w.cfg.IntervalMS
		w.mu.Unlock()
		if ms <= 0 {
			ms = defaultInterval
		}
		if ticker != nil {
			ticker.Stop()
		}
		ticker = time.NewTicker(time.Duration(ms) * time.Millisecond)
	}
	resetTicker()

	// First fetch comes from Apply kick (or first ticker). Avoid double Start+Apply poll.
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pollOnce()
		case <-w.kick:
			w.pollOnce()
			resetTicker()
		}
	}
}

func (w *Weather) pollOnce() {
	w.mu.Lock()
	cfg := w.cfg
	w.mu.Unlock()

	units := wx.UnitsF
	if strings.EqualFold(cfg.Units, "c") || strings.EqualFold(cfg.Units, "celsius") {
		units = wx.UnitsC
	}
	snap := w.client.Fetch(cfg.WeatherPlace(), units)
	metric := units == wx.UnitsC
	fyne.Do(func() {
		w.surface.setSnapshot(snap, metric)
	})
}

type dayCol struct {
	icon  *canvas.Text
	temps *canvas.Text
	name  *canvas.Text
}

type surface struct {
	widget.BaseWidget
	drag widgetx.Drag

	rootBG *canvas.Rectangle
	panel  *canvas.Rectangle
	accent *canvas.Rectangle
	rule   *canvas.Rectangle

	bigIcon   *canvas.Text
	temp      *canvas.Text
	condition *canvas.Text
	detailKey [3]*canvas.Text
	detailVal [3]*canvas.Text
	status    *canvas.Text
	days      [forecastDays]dayCol

	fg     color.Color
	dim    color.Color
	iconSz float32
	textSz float32
	tempSz float32
}

func newSurface() *surface {
	s := &surface{
		drag:    widgetx.Drag{},
		rootBG:  canvas.NewRectangle(winutil.ClearColor()),
		panel:   canvas.NewRectangle(winutil.ClearColor()),
		accent:  canvas.NewRectangle(defaultAccent),
		rule:    canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 180}),
		fg:      colorWhite,
		dim:     color.NRGBA{R: 255, G: 255, B: 255, A: 200},
		iconSz:  42,
		textSz:  widgetx.CaptionSize,
		tempSz:  36,
		bigIcon: &canvas.Text{
			Text:      string(rune(0xf185)),
			Color:     colorWhite,
			TextSize:  42,
			Alignment: fyne.TextAlignCenter,
		},
		temp: &canvas.Text{
			Color:     colorWhite,
			TextSize:  36,
			Alignment: fyne.TextAlignLeading,
		},
		condition: &canvas.Text{
			Color:     colorWhite,
			TextSize:  13,
			Alignment: fyne.TextAlignLeading,
		},
		status: &canvas.Text{
			Color:     color.NRGBA{R: 255, G: 180, B: 80, A: 255},
			TextSize:  11,
			Alignment: fyne.TextAlignCenter,
		},
	}
	keys := []string{"Feels Like:", "Humidity:", "Wind:"}
	for i := 0; i < 3; i++ {
		s.detailKey[i] = &canvas.Text{
			Text:      keys[i],
			Color:     s.dim,
			TextSize:  11,
			Alignment: fyne.TextAlignLeading,
		}
		s.detailVal[i] = &canvas.Text{
			Color:     colorWhite,
			TextSize:  11,
			Alignment: fyne.TextAlignTrailing,
		}
	}
	for i := 0; i < forecastDays; i++ {
		s.days[i] = dayCol{
			icon: &canvas.Text{
				Color:     colorWhite,
				TextSize:  18,
				Alignment: fyne.TextAlignCenter,
			},
			temps: &canvas.Text{
				Color:     colorWhite,
				TextSize:  11,
				Alignment: fyne.TextAlignCenter,
			},
			name: &canvas.Text{
				Color:     s.dim,
				TextSize:  11,
				Alignment: fyne.TextAlignCenter,
			},
		}
	}
	s.status.Hide()
	s.accent.Hide()
	s.rule.Hide()
	s.ExtendBaseWidget(s)
	return s
}

func (s *surface) SetDraggable(on bool) { s.drag.Enabled = on }

func (s *surface) setSnapshot(snap wx.Snapshot, metric bool) {
	if !snap.OK {
		msg := snap.Err
		if msg == "" {
			msg = "weather unavailable"
		}
		s.status.Text = msg
		s.status.Show()
		s.bigIcon.Hide()
		s.temp.Hide()
		s.condition.Hide()
		for i := 0; i < 3; i++ {
			s.detailKey[i].Hide()
			s.detailVal[i].Hide()
		}
		for i := 0; i < forecastDays; i++ {
			s.days[i].icon.Hide()
			s.days[i].temps.Hide()
			s.days[i].name.Hide()
		}
		s.layoutAll(s.Size())
		s.refreshAll()
		return
	}

	s.status.Hide()
	s.bigIcon.Show()
	s.temp.Show()
	s.condition.Show()
	for i := 0; i < 3; i++ {
		s.detailKey[i].Show()
		s.detailVal[i].Show()
	}

	s.bigIcon.Text = string(icons.Rune(snap.Current.Icon, ""))
	s.temp.Text = wx.FormatTemp(snap.Current.TempC)
	s.condition.Text = snap.Current.Label
	s.detailVal[0].Text = wx.FormatTemp(snap.Current.FeelsC)
	s.detailVal[1].Text = fmt.Sprintf("%d%%", snap.Current.Humidity)
	s.detailVal[2].Text = wx.FormatWind(snap.Current.WindDeg, snap.Current.WindKmh, metric)

	for i := 0; i < forecastDays; i++ {
		if i >= len(snap.Forecast) {
			s.days[i].icon.Hide()
			s.days[i].temps.Hide()
			s.days[i].name.Hide()
			continue
		}
		d := snap.Forecast[i]
		s.days[i].icon.Show()
		s.days[i].temps.Show()
		s.days[i].name.Show()
		s.days[i].icon.Text = string(icons.Rune(d.Icon, ""))
		s.days[i].temps.Text = wx.FormatRange(d.HighC, d.LowC)
		s.days[i].name.Text = d.Label
	}

	s.layoutAll(s.Size())
	s.refreshAll()
}

func (s *surface) refreshAll() {
	canvas.Refresh(s.panel)
	canvas.Refresh(s.accent)
	canvas.Refresh(s.rule)
	canvas.Refresh(s.bigIcon)
	canvas.Refresh(s.temp)
	canvas.Refresh(s.condition)
	canvas.Refresh(s.status)
	for i := 0; i < 3; i++ {
		canvas.Refresh(s.detailKey[i])
		canvas.Refresh(s.detailVal[i])
	}
	for i := 0; i < forecastDays; i++ {
		canvas.Refresh(s.days[i].icon)
		canvas.Refresh(s.days[i].temps)
		canvas.Refresh(s.days[i].name)
	}
}

func (s *surface) applyStyle(cfg config.WidgetConfig) error {
	col, err := config.ParseColor(cfg.Color, colorWhite)
	if err != nil {
		return fmt.Errorf("color: %w", err)
	}
	if c, ok := col.(color.NRGBA); ok {
		c.A = 255
		col = c
	}
	if _, err := config.ParseColor(cfg.AccentColor, defaultAccent); err != nil {
		return fmt.Errorf("accent_color: %w", err)
	}
	panel := winutil.ClearColor()
	if strings.TrimSpace(cfg.PanelColor) != "" {
		panel, err = config.ParseColor(cfg.PanelColor, panel)
		if err != nil {
			return fmt.Errorf("panel_color: %w", err)
		}
	}

	s.fg = col
	if c, ok := col.(color.NRGBA); ok {
		s.dim = color.NRGBA{R: c.R, G: c.G, B: c.B, A: 200}
	} else {
		s.dim = color.NRGBA{R: 255, G: 255, B: 255, A: 200}
	}
	s.iconSz = cfg.IconSize
	if s.iconSz <= 0 {
		s.iconSz = 42
	}
	s.textSz = cfg.TextSize
	if s.textSz <= 0 {
		s.textSz = widgetx.CaptionSize
	}
	s.tempSz = s.textSz * 3
	if s.tempSz < 28 {
		s.tempSz = 28
	}
	detailSz := float32(int(s.textSz)) // avoid fractional glyph sizes
	if detailSz < widgetx.CaptionSize {
		detailSz = widgetx.CaptionSize
	}
	forecastSz := detailSz

	iconPath := cfg.IconFont
	if iconPath == "" {
		iconPath = defaultIconFont
	}
	labelPath := cfg.LabelFont
	if labelPath == "" {
		labelPath = defaultLabelFont
	}
	boldPath := cfg.WeekdayFont
	if boldPath == "" {
		boldPath = defaultBoldFont
	}
	iconRes, err := widgetx.LoadFont(cfg.AssetsDir, iconPath)
	if err != nil {
		return fmt.Errorf("icon_font: %w", err)
	}
	labelRes, err := widgetx.LoadFont(cfg.AssetsDir, labelPath)
	if err != nil {
		return fmt.Errorf("label_font: %w", err)
	}
	boldRes, err := widgetx.LoadFont(cfg.AssetsDir, boldPath)
	if err != nil {
		return fmt.Errorf("weekday_font: %w", err)
	}

	s.panel.FillColor = panel
	s.accent.Hide()
	s.rule.Hide()
	s.rootBG.FillColor = winutil.ClearColor()

	s.bigIcon.Color = col
	s.bigIcon.TextSize = s.iconSz
	s.bigIcon.FontSource = iconRes

	s.temp.Color = col
	s.temp.TextSize = s.tempSz
	s.temp.FontSource = boldRes

	s.condition.Color = s.dim
	s.condition.TextSize = detailSz
	s.condition.FontSource = labelRes
	s.condition.TextStyle = fyne.TextStyle{}

	for i := 0; i < 3; i++ {
		s.detailKey[i].Color = s.dim
		s.detailKey[i].TextSize = detailSz
		s.detailKey[i].FontSource = labelRes
		s.detailKey[i].TextStyle = fyne.TextStyle{}
		s.detailVal[i].Color = col
		s.detailVal[i].TextSize = detailSz
		s.detailVal[i].FontSource = labelRes
		s.detailVal[i].TextStyle = fyne.TextStyle{}
	}
	s.status.FontSource = labelRes
	s.status.TextSize = detailSz
	s.status.TextStyle = fyne.TextStyle{}

	for i := 0; i < forecastDays; i++ {
		s.days[i].icon.Color = col
		s.days[i].icon.TextSize = float32(int(s.iconSz * 0.5))
		if s.days[i].icon.TextSize < 18 {
			s.days[i].icon.TextSize = 18
		}
		s.days[i].icon.FontSource = iconRes
		s.days[i].temps.Color = col
		s.days[i].temps.TextSize = forecastSz
		s.days[i].temps.FontSource = labelRes
		s.days[i].temps.TextStyle = fyne.TextStyle{}
		s.days[i].name.Color = s.dim
		s.days[i].name.TextSize = forecastSz
		s.days[i].name.FontSource = labelRes
		s.days[i].name.TextStyle = fyne.TextStyle{}
	}
	return nil
}

func (s *surface) layoutAll(size fyne.Size) {
	s.rootBG.Resize(size)
	s.rootBG.Move(fyne.NewPos(0, 0))

	pad := widgetx.Pad
	s.panel.Move(fyne.NewPos(0, 0))
	s.panel.Resize(size)

	innerW := size.Width - pad*2
	if innerW < 40 {
		innerW = size.Width
		pad = 0
	}

	usableH := size.Height - widgetx.PadY*2
	if usableH < 80 {
		usableH = size.Height
	}
	originY := (size.Height - usableH) / 2

	topH := usableH * 0.55
	if topH < 90 {
		topH = 90
		if topH > usableH*0.7 {
			topH = usableH * 0.55
		}
	}
	splitY := originY + topH

	if !s.status.Hidden {
		s.status.Move(fyne.NewPos(pad, size.Height/2-8))
		s.status.Resize(fyne.NewSize(innerW, 20))
		return
	}

	contentTop := originY + widgetx.PadY
	rowH := splitY - contentTop - 2

	detailW := float32(132)
	if innerW < 300 {
		detailW = 110
	}

	iconW := s.iconSz + 8
	iconH := s.iconSz + 4
	const minIconTextGap float32 = 14
	const stackGap float32 = 2
	tempMin := s.temp.MinSize()
	condMin := s.condition.MinSize()
	textColW := tempMin.Width
	if condMin.Width > textColW {
		textColW = condMin.Width
	}
	if textColW < 48 {
		textColW = 48
	}
	detailX := pad + innerW - detailW
	// Leave a gutter before the detail column, then center temp/condition
	// in the remaining span so icon + big weather fill the mid gap.
	const midGutter float32 = 16
	midLeft := pad + iconW
	midRight := detailX - midGutter
	midW := midRight - midLeft
	if midW < textColW+minIconTextGap {
		midW = textColW + minIconTextGap
	}
	if textColW > midW-minIconTextGap && midW-minIconTextGap > 40 {
		textColW = midW - minIconTextGap
	}
	stackH := s.tempSz + stackGap + s.textSz
	groupH := stackH
	if iconH > groupH {
		groupH = iconH
	}
	groupTop := contentTop + (rowH-groupH)/2
	if groupTop < contentTop {
		groupTop = contentTop
	}
	groupLeft := pad

	s.bigIcon.Move(fyne.NewPos(groupLeft, groupTop+(groupH-iconH)/2))
	s.bigIcon.Resize(fyne.NewSize(iconW, iconH))

	// Bias toward a comfortable gap; when extra room remains, park the
	// stack near the middle of icon→details rather than hugging the icon.
	textX := midLeft + (midW-textColW)/2
	if textX < midLeft+minIconTextGap {
		textX = midLeft + minIconTextGap
	}
	stackTop := groupTop + (groupH-stackH)/2
	s.temp.Alignment = fyne.TextAlignLeading
	s.condition.Alignment = fyne.TextAlignLeading
	s.temp.Move(fyne.NewPos(textX, stackTop))
	s.temp.Resize(fyne.NewSize(textColW, s.tempSz+2))
	s.condition.Move(fyne.NewPos(textX, stackTop+s.tempSz+stackGap))
	s.condition.Resize(fyne.NewSize(textColW, s.textSz+2))

	lineH := s.textSz + 4
	detailBlockH := lineH * 3
	detailStartY := contentTop + (rowH-detailBlockH)/2
	for i := 0; i < 3; i++ {
		y := detailStartY + float32(i)*lineH
		keyW := detailW * 0.58
		s.detailKey[i].Move(fyne.NewPos(detailX, y))
		s.detailKey[i].Resize(fyne.NewSize(keyW, lineH))
		s.detailVal[i].Move(fyne.NewPos(detailX+keyW, y))
		s.detailVal[i].Resize(fyne.NewSize(detailW-keyW, lineH))
	}

	botTop := splitY + widgetx.RowGap/2
	botH := originY + usableH - botTop - widgetx.PadY
	colW := innerW / float32(forecastDays)
	for i := 0; i < forecastDays; i++ {
		x := pad + float32(i)*colW
		dayIconH := s.days[i].icon.TextSize + 2
		tempsH := s.textSz + 2
		nameH := s.textSz + 2
		const fGap float32 = 2
		blockH := dayIconH + fGap + tempsH + fGap + nameH
		blockTop := botTop + (botH-blockH)/2
		if blockTop < botTop {
			blockTop = botTop
		}
		s.days[i].icon.Move(fyne.NewPos(x, blockTop))
		s.days[i].icon.Resize(fyne.NewSize(colW, dayIconH))
		s.days[i].temps.Move(fyne.NewPos(x, blockTop+dayIconH+fGap))
		s.days[i].temps.Resize(fyne.NewSize(colW, tempsH))
		s.days[i].name.Move(fyne.NewPos(x, blockTop+dayIconH+fGap+tempsH+fGap))
		s.days[i].name.Resize(fyne.NewSize(colW, nameH))
	}
}
func (s *surface) CreateRenderer() fyne.WidgetRenderer {
	return &renderer{surface: s}
}

func (s *surface) MinSize() fyne.Size { return fyne.NewSize(120, 120) }

type renderer struct{ surface *surface }

func (r *renderer) Layout(size fyne.Size) { r.surface.layoutAll(size) }
func (r *renderer) MinSize() fyne.Size    { return r.surface.MinSize() }
func (r *renderer) Refresh() {
	s := r.surface
	s.rootBG.Refresh()
	for _, o := range r.Objects() {
		o.Refresh()
	}
}
func (r *renderer) Destroy() {}
func (r *renderer) Objects() []fyne.CanvasObject {
	s := r.surface
	objs := []fyne.CanvasObject{
		s.rootBG, s.panel, s.accent, s.rule,
		s.bigIcon, s.temp, s.condition, s.status,
	}
	for i := 0; i < 3; i++ {
		objs = append(objs, s.detailKey[i], s.detailVal[i])
	}
	for i := 0; i < forecastDays; i++ {
		objs = append(objs, s.days[i].icon, s.days[i].temps, s.days[i].name)
	}
	return objs
}

func (s *surface) MouseDown(e *desktop.MouseEvent) { s.drag.MouseDown(e) }
func (s *surface) MouseUp(e *desktop.MouseEvent)   { s.drag.MouseUp(e) }
func (s *surface) MouseMoved(e *desktop.MouseEvent) {
	s.drag.MouseMoved(e)
}
func (s *surface) MouseIn(e *desktop.MouseEvent) { s.drag.MouseIn(e) }
func (s *surface) MouseOut()                     { s.drag.MouseOut() }

var _ desktop.Mouseable = (*surface)(nil)
var _ desktop.Hoverable = (*surface)(nil)
