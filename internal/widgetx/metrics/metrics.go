package metrics

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/sysinfo"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/winutil"
)

func init() {
	widgetx.Register("metrics", New)
}

const (
	defaultIconFont  = "fonts/icons/fa-solid.otf"
	defaultLabelFont = widgetx.CaptionFont
	defaultIconSize  = float32(28)
	defaultTextSize  = widgetx.CaptionSize
	defaultGapX      = float32(28)
	defaultGapY      = float32(12)
	defaultInterval  = 1000
)

// Metrics is a transparent grid of icon + system measure text cells.
type Metrics struct {
	id      string
	win     fyne.Window
	surface *gridSurface
	kick    chan struct{}

	mu     sync.Mutex
	cfg    config.WidgetConfig
	cancel context.CancelFunc
}

// New creates a metrics widget panel (no personal window).
func New(_ fyne.App, cfg config.WidgetConfig) (widgetx.Instance, error) {
	return &Metrics{
		id:      cfg.ID,
		surface: newGridSurface(),
		cfg:     cfg,
		kick:    make(chan struct{}, 1),
	}, nil
}

func (m *Metrics) ID() string   { return m.id }
func (m *Metrics) Type() string { return "metrics" }

func (m *Metrics) Content() fyne.CanvasObject { return m.surface }

func (m *Metrics) SetHost(win fyne.Window) {
	m.win = win
	m.surface.drag.Win = win
}

func (m *Metrics) FlexWeight() float32 {
	m.mu.Lock()
	h := m.cfg.Height
	m.mu.Unlock()
	if h <= 0 {
		return 160
	}
	return h
}

func (m *Metrics) MinSize() fyne.Size { return m.surface.MinSize() }

func (m *Metrics) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startPollLocked(ctx)
}

func (m *Metrics) Close() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.mu.Unlock()
}

func (m *Metrics) Apply(cfg config.WidgetConfig) error {
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()

	m.surface.SetDraggable(cfg.EditMode)

	if !measuresEqual(m.surface.measures, cfg.Measures) {
		m.surface.rebuild(cfg.Measures)
	}
	if err := m.surface.applyStyle(cfg); err != nil {
		log.Printf("metrics %q style: %v", cfg.ID, err)
	}
	m.surface.layoutCells(m.surface.Size())
	for _, c := range m.surface.cells {
		canvas.Refresh(c.icon)
		canvas.Refresh(c.label)
		canvas.Refresh(c.label2)
	}

	m.requestPoll()
	return nil
}

func measuresEqual(a, b []config.MeasureConfig) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *Metrics) requestPoll() {
	select {
	case m.kick <- struct{}{}:
	default:
	}
}

func (m *Metrics) startPollLocked(parent context.Context) {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	go m.pollLoop(ctx)
}

func (m *Metrics) pollLoop(ctx context.Context) {
	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	resetTicker := func() {
		m.mu.Lock()
		ms := m.cfg.IntervalMS
		m.mu.Unlock()
		if ms <= 0 {
			ms = defaultInterval
		}
		if ticker != nil {
			ticker.Stop()
		}
		ticker = time.NewTicker(time.Duration(ms) * time.Millisecond)
	}
	resetTicker()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollOnce()
		case <-m.kick:
			m.pollOnce()
			resetTicker()
		}
	}
}

func (m *Metrics) pollOnce() {
	m.mu.Lock()
	cfg := m.cfg
	m.mu.Unlock()

	lines := make([]measureLines, len(cfg.Measures))
	for i, mc := range cfg.Measures {
		lines[i] = formatMeasure(mc)
	}
	fyne.Do(func() {
		m.surface.setLines(lines)
	})
}

type measureLines struct {
	primary   string
	secondary string
}

func formatMeasure(mc config.MeasureConfig) measureLines {
	label := mc.Label
	switch mc.Kind {
	case "cpu":
		if label == "" {
			label = "CPU"
		}
		u := sysinfo.CPUPercent(0.2)
		return measureLines{primary: sysinfo.FormatUsageLine(label, u.Percent, u.OK)}
	case "gpu":
		if label == "" {
			label = "GPU"
		}
		u := sysinfo.GPUPercentIndex(mc.GPU)
		return measureLines{primary: sysinfo.FormatUsageLine(label, u.Percent, u.OK)}
	case "memory":
		if label == "" {
			label = "RAM"
		}
		u := sysinfo.MemoryPercent()
		return measureLines{primary: sysinfo.FormatUsageLine(label, u.Percent, u.OK)}
	case "storage":
		if label == "" {
			label = mc.Device
		}
		s := sysinfo.DiskUsage(mc.Device)
		p, sec := sysinfo.FormatStorageLines(label, s.UsedBytes, s.TotalBytes, s.OK)
		return measureLines{primary: p, secondary: sec}
	case "network":
		up, down, ok := sysinfo.NetworkRates()
		p, sec := sysinfo.FormatNetworkLines(up, down, ok)
		return measureLines{primary: p, secondary: sec}
	default:
		return measureLines{primary: fmt.Sprintf("%s - —", label)}
	}
}

type measureCell struct {
	icon     *canvas.Text
	label    *canvas.Text
	label2   *canvas.Text
	capacity bool // storage / inventory row
}

type gridSurface struct {
	widget.BaseWidget
	drag widgetx.Drag

	rootBG     *canvas.Rectangle
	editBorder *canvas.Rectangle
	cells      []measureCell
	measures   []config.MeasureConfig
	editMode   bool

	columns int
	gapX    float32
	gapY    float32
	iconSz  float32
	textSz  float32
	fg      color.Color
}

func newGridSurface() *gridSurface {
	s := &gridSurface{
		drag:       widgetx.Drag{},
		rootBG:     canvas.NewRectangle(winutil.ClearColor()),
		editBorder: widgetx.NewEditBorder(),
		columns:    3,
		gapX:       defaultGapX,
		gapY:       defaultGapY,
		iconSz:     defaultIconSize,
		textSz:     defaultTextSize,
		fg:         color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
	s.ExtendBaseWidget(s)
	return s
}

func (s *gridSurface) SetDraggable(on bool) { s.drag.Enabled = on }

func (s *gridSurface) SetEditMode(on bool) {
	s.editMode = on
	widgetx.LayoutEditBorder(s.editBorder, s.Size(), on)
	canvas.Refresh(s.editBorder)
}

func (s *gridSurface) applyStyle(cfg config.WidgetConfig) error {
	col, err := config.ParseColor(cfg.Color, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return fmt.Errorf("color: %w", err)
	}
	// Keep ink fully opaque so AA composites cleanly on the transparent framebuffer.
	if n, ok := col.(color.NRGBA); ok {
		n.A = 255
		col = n
	}
	s.fg = col
	s.columns = cfg.Columns
	s.gapX = cfg.GapX
	if s.gapX <= 0 {
		s.gapX = defaultGapX
	}
	s.gapY = cfg.GapY
	if s.gapY <= 0 {
		s.gapY = defaultGapY
	}
	s.iconSz = cfg.IconSize
	if s.iconSz <= 0 {
		s.iconSz = defaultIconSize
	}
	s.textSz = cfg.TextSize
	if s.textSz <= 0 {
		s.textSz = defaultTextSize
	}

	iconFontPath := cfg.IconFont
	if iconFontPath == "" {
		iconFontPath = defaultIconFont
	}
	labelFontPath := cfg.LabelFont
	if labelFontPath == "" {
		labelFontPath = defaultLabelFont
	}
	iconRes, err := widgetx.LoadFont(cfg.AssetsDir, iconFontPath)
	if err != nil {
		return fmt.Errorf("icon_font: %w", err)
	}
	labelRes, err := widgetx.LoadFont(cfg.AssetsDir, labelFontPath)
	if err != nil {
		return fmt.Errorf("label_font: %w", err)
	}

	for i := range s.cells {
		s.cells[i].icon.Color = col
		s.cells[i].icon.TextSize = s.iconSz
		s.cells[i].icon.FontSource = iconRes
		s.cells[i].label.Color = col
		s.cells[i].label.TextSize = s.textSz
		s.cells[i].label.FontSource = labelRes
		s.cells[i].label.TextStyle = fyne.TextStyle{}
		s.cells[i].label2.Color = col
		s.cells[i].label2.TextSize = s.textSz
		s.cells[i].label2.FontSource = labelRes
		s.cells[i].label2.TextStyle = fyne.TextStyle{}
	}
	return nil
}

func (s *gridSurface) rebuild(measures []config.MeasureConfig) {
	s.measures = append([]config.MeasureConfig(nil), measures...)
	s.cells = make([]measureCell, len(measures))
	for i, m := range measures {
		r := IconRune(m.Icon, m.IconCode)
		iconText := ""
		if r != 0 {
			iconText = string(r)
		}
		s.cells[i] = measureCell{
			capacity: m.Kind == "storage",
			icon: &canvas.Text{
				Text:      iconText,
				Color:     s.fg,
				TextSize:  s.iconSz,
				Alignment: fyne.TextAlignCenter,
			},
			label: &canvas.Text{
				Text:      "",
				Color:     s.fg,
				TextSize:  s.textSz,
				Alignment: fyne.TextAlignCenter,
			},
			label2: &canvas.Text{
				Text:      "",
				Color:     s.fg,
				TextSize:  s.textSz,
				Alignment: fyne.TextAlignCenter,
			},
		}
		s.cells[i].label2.Hide()
	}
	s.Refresh()
}

func (s *gridSurface) setLines(lines []measureLines) {
	n := len(s.cells)
	if len(lines) < n {
		n = len(lines)
	}
	changed := false
	for i := 0; i < n; i++ {
		c := &s.cells[i]
		if c.label.Text != lines[i].primary {
			c.label.Text = lines[i].primary
			changed = true
		}
		sec := lines[i].secondary
		if sec == "" {
			if !c.label2.Hidden || c.label2.Text != "" {
				c.label2.Text = ""
				c.label2.Hide()
				changed = true
			}
		} else {
			if c.label2.Hidden || c.label2.Text != sec {
				c.label2.Text = sec
				c.label2.Show()
				changed = true
			}
		}
	}
	if !changed {
		return
	}
	s.layoutCells(s.Size())
	for _, c := range s.cells {
		canvas.Refresh(c.icon)
		canvas.Refresh(c.label)
		canvas.Refresh(c.label2)
	}
}

func (s *gridSurface) layoutCells(size fyne.Size) {
	s.rootBG.Move(fyne.NewPos(0, 0))
	s.rootBG.Resize(size)
	if len(s.cells) == 0 {
		return
	}
	cols := s.columns
	if cols <= 0 {
		cols = len(s.cells)
	}
	if cols < 1 {
		cols = 1
	}

	pad := widgetx.Pad
	innerW := size.Width - pad*2
	innerH := size.Height - widgetx.PadY*2
	if innerW < 40 {
		innerW = size.Width
		pad = 0
	}
	if innerH < 40 {
		innerH = size.Height
	}
	originY := (size.Height - innerH) / 2

	liveIdx := make([]int, 0, len(s.cells))
	capIdx := make([]int, 0, len(s.cells))
	for i, c := range s.cells {
		if c.capacity {
			capIdx = append(capIdx, i)
		} else {
			liveIdx = append(liveIdx, i)
		}
	}

	// Split live (rates) above capacity (disks) when both are present.
	if len(liveIdx) > 0 && len(capIdx) > 0 {
		sectionGap := s.gapY
		if sectionGap < widgetx.RowGap {
			sectionGap = widgetx.RowGap
		}
		liveRows := (len(liveIdx) + cols - 1) / cols
		capRows := (len(capIdx) + cols - 1) / cols
		liveWeight := float32(liveRows)
		capWeight := float32(capRows)
		bandTotal := liveWeight + capWeight
		liveBandH := (innerH - sectionGap) * (liveWeight / bandTotal)
		capBandH := innerH - sectionGap - liveBandH
		s.layoutCellGroup(liveIdx, pad, originY, innerW, liveBandH, cols)
		s.layoutCellGroup(capIdx, pad, originY+liveBandH+sectionGap, innerW, capBandH, cols)
		return
	}

	all := make([]int, len(s.cells))
	for i := range all {
		all[i] = i
	}
	s.layoutCellGroup(all, pad, originY, innerW, innerH, cols)
}

func (s *gridSurface) layoutCellGroup(idxs []int, pad, originY, innerW, bandH float32, cols int) {
	if len(idxs) == 0 || bandH < 1 {
		return
	}
	rows := (len(idxs) + cols - 1) / cols
	cellW := innerW / float32(cols)
	cellH := bandH / float32(rows)
	iconH := s.iconSz
	labelH := s.textSz + 2
	lineGap := float32(1)
	stackGap := float32(4)

	// Keep icon baselines aligned when any cell in the group has two caption lines.
	groupTwoLine := false
	for _, i := range idxs {
		if !s.cells[i].label2.Hidden && s.cells[i].label2.Text != "" {
			groupTwoLine = true
			break
		}
	}
	stackH := iconH + stackGap + labelH
	if groupTwoLine {
		stackH += lineGap + labelH
	}

	for n, i := range idxs {
		c := s.cells[i]
		row := n / cols
		col := n % cols
		rowStart := row * cols
		rowCount := cols
		if rowStart+rowCount > len(idxs) {
			rowCount = len(idxs) - rowStart
		}
		rowOffset := (float32(cols-rowCount) * cellW) / 2

		x := pad + rowOffset + float32(col)*cellW
		y := originY + float32(row)*cellH
		top := y + (cellH-stackH)/2
		if top < y {
			top = y
		}

		c.icon.Alignment = fyne.TextAlignCenter
		c.label.Alignment = fyne.TextAlignCenter
		c.label2.Alignment = fyne.TextAlignCenter
		c.icon.TextSize = s.iconSz
		c.label.TextSize = s.textSz
		c.label2.TextSize = s.textSz
		c.icon.Move(fyne.NewPos(x, top))
		c.icon.Resize(fyne.NewSize(cellW, iconH))
		ly := top + iconH + stackGap
		c.label.Move(fyne.NewPos(x, ly))
		c.label.Resize(fyne.NewSize(cellW, labelH))
		if !c.label2.Hidden && c.label2.Text != "" {
			c.label2.Move(fyne.NewPos(x, ly+labelH+lineGap))
			c.label2.Resize(fyne.NewSize(cellW, labelH))
		} else {
			c.label2.Move(fyne.NewPos(x, ly))
			c.label2.Resize(fyne.NewSize(cellW, 0))
		}
	}
}

func (s *gridSurface) CreateRenderer() fyne.WidgetRenderer {
	return &gridRenderer{surface: s}
}

func (s *gridSurface) MinSize() fyne.Size { return fyne.NewSize(120, 64) }

type gridRenderer struct {
	surface *gridSurface
	objs    []fyne.CanvasObject
}

func (r *gridRenderer) Layout(size fyne.Size) {
	r.surface.layoutCells(size)
}

func (r *gridRenderer) MinSize() fyne.Size { return r.surface.MinSize() }

func (r *gridRenderer) Objects() []fyne.CanvasObject {
	s := r.surface
	need := 1 + len(s.cells)*3
	if cap(r.objs) < need {
		r.objs = make([]fyne.CanvasObject, 0, need)
	}
	r.objs = r.objs[:0]
	r.objs = append(r.objs, s.rootBG)
	for _, c := range s.cells {
		r.objs = append(r.objs, c.icon, c.label, c.label2)
	}
	return r.objs
}

func (r *gridRenderer) Destroy() {
	r.objs = nil
	r.surface = nil
}

func (r *gridRenderer) Refresh() {
	s := r.surface
	s.rootBG.FillColor = winutil.ClearColor()
	s.rootBG.Refresh()
	for _, c := range s.cells {
		c.icon.Refresh()
		c.label.Refresh()
		c.label2.Refresh()
	}
}

func (s *gridSurface) MouseDown(e *desktop.MouseEvent) { s.drag.MouseDown(e) }
func (s *gridSurface) MouseUp(e *desktop.MouseEvent)   { s.drag.MouseUp(e) }
func (s *gridSurface) MouseMoved(e *desktop.MouseEvent) {
	s.drag.MouseMoved(e)
}
func (s *gridSurface) MouseIn(e *desktop.MouseEvent) { s.drag.MouseIn(e) }
func (s *gridSurface) MouseOut()                     { s.drag.MouseOut() }

var _ desktop.Mouseable = (*gridSurface)(nil)
var _ desktop.Hoverable = (*gridSurface)(nil)
