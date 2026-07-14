package clock

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

	"ryanmarsh.net/rmgo/internal/config"
	"ryanmarsh.net/rmgo/internal/widgetx"
	"ryanmarsh.net/rmgo/internal/winutil"
)

func init() {
	widgetx.Register("clock", New)
}

var (
	colorWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colorInk   = color.NRGBA{R: 20, G: 20, B: 20, A: 255}
)

// Clock is weekday (top bordered box) over a solid bar with time/date.
type Clock struct {
	id         string
	win        fyne.Window
	surface    *dragSurface
	weekdayFmt string

	mu     sync.Mutex
	format string
	cfg    config.WidgetConfig

	lastWeekday string
}

// New creates a datetime widget panel (no personal window).
func New(_ fyne.App, cfg config.WidgetConfig) (widgetx.Instance, error) {
	surface := newDragSurface()
	c := &Clock{
		id:         cfg.ID,
		surface:    surface,
		format:     defaultFormat(cfg.Format),
		weekdayFmt: "Monday",
		cfg:        cfg,
	}
	return c, nil
}

func defaultFormat(format string) string {
	if format == "" {
		return "3:04 PM | January 2"
	}
	return format
}

func (c *Clock) ID() string   { return c.id }
func (c *Clock) Type() string { return "clock" }

func (c *Clock) Content() fyne.CanvasObject { return c.surface }

func (c *Clock) SetHost(win fyne.Window) {
	c.win = win
	c.surface.drag.Win = win
}

func (c *Clock) FlexWeight() float32 {
	c.mu.Lock()
	h := c.cfg.Height
	c.mu.Unlock()
	if h <= 0 {
		return 160
	}
	return h
}

func (c *Clock) MinSize() fyne.Size { return c.surface.MinSize() }

func (c *Clock) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		c.tick()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c.tick()
			}
		}
	}()
}

func (c *Clock) tick() {
	c.mu.Lock()
	format := c.format
	weekdayFmt := c.weekdayFmt
	c.mu.Unlock()

	now := time.Now()
	weekday := strings.ToUpper(now.Format(weekdayFmt))
	bottom := now.Format(format)
	fyne.Do(func() {
		weekdayChanged := weekday != c.lastWeekday
		if weekdayChanged {
			c.lastWeekday = weekday
			c.surface.weekday.Text = weekday
			c.surface.weekday.Refresh()
		}
		c.surface.detail.Text = bottom
		c.surface.detail.Refresh()
	})
}

func (c *Clock) Apply(cfg config.WidgetConfig) error {
	c.mu.Lock()
	c.cfg = cfg
	c.format = defaultFormat(cfg.Format)
	c.mu.Unlock()

	c.surface.SetDraggable(cfg.EditMode)

	if err := c.surface.applyFonts(cfg); err != nil {
		log.Printf("clock %q fonts: %v", cfg.ID, err)
	}
	if err := c.surface.applyColors(cfg); err != nil {
		log.Printf("clock %q colors: %v", cfg.ID, err)
	}
	c.lastWeekday = "" // force weekday refresh after style change
	c.tick()
	return nil
}

func (c *Clock) Close() {}

type dragSurface struct {
	widget.BaseWidget
	drag widgetx.Drag

	rootBG    *canvas.Rectangle
	dayFill   *canvas.Rectangle
	dayBorder *canvas.Rectangle
	weekday   *canvas.Text
	barFill   *canvas.Rectangle
	detail    *canvas.Text
}

func newDragSurface() *dragSurface {
	key := winutil.ClearColor()
	s := &dragSurface{
		drag:      widgetx.Drag{},
		rootBG:    canvas.NewRectangle(key),
		dayFill:   canvas.NewRectangle(key),
		dayBorder: &canvas.Rectangle{StrokeColor: colorWhite, StrokeWidth: 2, FillColor: color.Transparent},
		weekday: &canvas.Text{
			Color:     colorWhite,
			TextSize:  42,
			TextStyle: fyne.TextStyle{},
			Alignment: fyne.TextAlignCenter,
		},
		barFill: canvas.NewRectangle(colorWhite),
		detail: &canvas.Text{
			Color:     colorInk,
			TextSize:  18,
			TextStyle: fyne.TextStyle{},
			Alignment: fyne.TextAlignCenter,
		},
	}
	s.ExtendBaseWidget(s)
	return s
}

func (s *dragSurface) SetDraggable(on bool) { s.drag.Enabled = on }

func (s *dragSurface) applyFonts(cfg config.WidgetConfig) error {
	weekdayFont, err := widgetx.LoadFont(cfg.AssetsDir, cfg.WeekdayFont)
	if err != nil {
		return fmt.Errorf("weekday_font: %w", err)
	}
	detailFont, err := widgetx.LoadFont(cfg.AssetsDir, cfg.DetailFont)
	if err != nil {
		return fmt.Errorf("detail_font: %w", err)
	}
	s.weekday.FontSource = weekdayFont
	s.detail.FontSource = detailFont
	s.weekday.Refresh()
	s.detail.Refresh()
	return nil
}

func (s *dragSurface) applyColors(cfg config.WidgetConfig) error {
	weekdayCol, err := config.ParseColor(cfg.TimeColor, colorWhite)
	if err != nil {
		return fmt.Errorf("time_color: %w", err)
	}
	detailCol, err := config.ParseColor(cfg.DateColor, colorInk)
	if err != nil {
		return fmt.Errorf("date_color: %w", err)
	}
	borderCol, err := config.ParseColor(cfg.RuleColor, colorWhite)
	if err != nil {
		return fmt.Errorf("rule_color: %w", err)
	}

	s.weekday.Color = weekdayCol
	s.detail.Color = detailCol
	s.dayBorder.StrokeColor = borderCol
	s.barFill.FillColor = colorWhite
	s.weekday.Refresh()
	s.detail.Refresh()
	s.dayBorder.Refresh()
	s.barFill.Refresh()
	return nil
}

func (s *dragSurface) CreateRenderer() fyne.WidgetRenderer {
	return &dragRenderer{surface: s}
}

func (s *dragSurface) MinSize() fyne.Size {
	return fyne.NewSize(120, 100)
}

type dragRenderer struct {
	surface *dragSurface
}

func (r *dragRenderer) Layout(size fyne.Size) {
	s := r.surface
	s.rootBG.Move(fyne.NewPos(0, 0))
	s.rootBG.Resize(size)

	pad := widgetx.Pad
	gap := widgetx.RowGap
	innerW := size.Width - pad*2
	if innerW < 40 {
		innerW = size.Width
		pad = 0
	}

	usableH := size.Height - widgetx.PadY*2
	if usableH < 60 {
		usableH = size.Height
	}
	originY := (size.Height - usableH) / 2

	barH := usableH * 0.28
	if barH < 28 {
		barH = 28
	}
	if barH > 44 {
		barH = 44
	}
	dayH := usableH - barH - gap
	if dayH < 40 {
		dayH = usableH * 0.62
		barH = usableH - dayH - gap
	}

	s.dayFill.Move(fyne.NewPos(pad, originY))
	s.dayFill.Resize(fyne.NewSize(innerW, dayH))
	s.dayBorder.Move(fyne.NewPos(pad, originY))
	s.dayBorder.Resize(fyne.NewSize(innerW, dayH))

	s.weekday.TextSize = dayH * 0.45
	if s.weekday.TextSize < 22 {
		s.weekday.TextSize = 22
	}
	s.weekday.Move(fyne.NewPos(pad, originY))
	s.weekday.Resize(fyne.NewSize(innerW, dayH))

	barY := originY + dayH + gap
	s.barFill.Move(fyne.NewPos(pad, barY))
	s.barFill.Resize(fyne.NewSize(innerW, barH))

	s.detail.TextSize = barH * 0.42
	if s.detail.TextSize < 12 {
		s.detail.TextSize = 12
	}
	s.detail.Move(fyne.NewPos(pad, barY))
	s.detail.Resize(fyne.NewSize(innerW, barH))
}

func (r *dragRenderer) MinSize() fyne.Size { return r.surface.MinSize() }

func (r *dragRenderer) Objects() []fyne.CanvasObject {
	s := r.surface
	return []fyne.CanvasObject{s.rootBG, s.dayFill, s.dayBorder, s.weekday, s.barFill, s.detail}
}

func (r *dragRenderer) Destroy() {}

func (r *dragRenderer) Refresh() {
	key := winutil.ClearColor()
	s := r.surface
	s.rootBG.FillColor = key
	s.dayFill.FillColor = key
	s.dayBorder.FillColor = color.Transparent
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

func (s *dragSurface) MouseDown(e *desktop.MouseEvent) { s.drag.MouseDown(e) }
func (s *dragSurface) MouseUp(e *desktop.MouseEvent)   { s.drag.MouseUp(e) }
func (s *dragSurface) MouseMoved(e *desktop.MouseEvent) {
	s.drag.MouseMoved(e)
}
func (s *dragSurface) MouseIn(e *desktop.MouseEvent) { s.drag.MouseIn(e) }
func (s *dragSurface) MouseOut()                     { s.drag.MouseOut() }

var _ desktop.Mouseable = (*dragSurface)(nil)
var _ desktop.Hoverable = (*dragSurface)(nil)
