package visualizer

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

	"github.com/RyanMarshCodes/golang-rainmeter/internal/audio"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/icons"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/media"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/winutil"
)

func init() {
	widgetx.Register("visualizer", New)
}

var colorWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

const (
	defaultBars      = 16
	defaultVizH      float32 = 28 // bar max height within the row
	defaultIconFont  = "fonts/icons/fa-solid.otf"
	defaultTitleFont = widgetx.DisplayFont
	defaultLabelFont = widgetx.CaptionFont
	defaultIconSize  = float32(18)
	defaultTextSize  = float32(16) // song title
	defaultMusicIcon = 0xf001     // fa-music (needs icon font + map)
)

// Visualizer is a WASAPI spectrum with optional SMTC now-playing under it.
type Visualizer struct {
	id      string
	win     fyne.Window
	surface *surface

	mu        sync.Mutex
	cfg       config.WidgetConfig
	analyzer  *audio.Analyzer
	vizCancel context.CancelFunc
	npCancel  context.CancelFunc
}

// New creates the visualizer widget panel (no personal window).
func New(_ fyne.App, cfg config.WidgetConfig) (widgetx.Instance, error) {
	return &Visualizer{
		id:      cfg.ID,
		surface: newSurface(),
		cfg:     cfg,
	}, nil
}

func (v *Visualizer) ID() string   { return v.id }
func (v *Visualizer) Type() string { return "visualizer" }

func (v *Visualizer) Content() fyne.CanvasObject { return v.surface }

func (v *Visualizer) SetHost(win fyne.Window) {
	v.win = win
	v.surface.drag.Win = win
}

func (v *Visualizer) FlexWeight() float32 {
	v.mu.Lock()
	h := v.cfg.Height
	v.mu.Unlock()
	if h <= 0 {
		return 110
	}
	return h
}

func (v *Visualizer) MinSize() fyne.Size { return v.surface.MinSize() }

func (v *Visualizer) Start(ctx context.Context) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.startAudioLocked(ctx)
	v.startNowPlayingLocked(ctx)
}

func (v *Visualizer) Close() {
	v.mu.Lock()
	v.stopAudioLocked()
	v.stopNowPlayingLocked()
	v.mu.Unlock()
}

func (v *Visualizer) Apply(cfg config.WidgetConfig) error {
	v.mu.Lock()
	v.cfg = cfg
	v.mu.Unlock()

	v.surface.SetDraggable(cfg.EditMode)

	bars := cfg.VisualizerBars
	if bars <= 0 {
		bars = defaultBars
	}
	vizH := cfg.VisualizerHeight
	if vizH <= 0 {
		vizH = defaultVizH
	}
	v.surface.setBarCount(bars, vizH)
	if err := v.surface.applyStyle(cfg); err != nil {
		log.Printf("visualizer %q style: %v", cfg.ID, err)
	}
	v.surface.layoutAll(v.surface.Size())

	v.mu.Lock()
	if v.analyzer != nil {
		v.analyzer.SetBandCount(bars)
	}
	v.mu.Unlock()

	return nil
}

func (v *Visualizer) startAudioLocked(parent context.Context) {
	v.stopAudioLocked()
	n := v.cfg.VisualizerBars
	if n <= 0 {
		n = defaultBars
	}
	if v.analyzer == nil {
		v.analyzer = audio.NewAnalyzer(n)
	} else {
		v.analyzer.SetBandCount(n)
	}
	ctx, cancel := context.WithCancel(parent)
	v.vizCancel = cancel
	if err := v.analyzer.Start(ctx); err != nil {
		log.Printf("visualizer %q audio: %v", v.id, err)
	}
	analyzer := v.analyzer
	go func() {
		t := time.NewTicker(16 * time.Millisecond)
		defer t.Stop()
		var (
			buf      []float32
			pending  []float32
			mu       sync.Mutex
			scheduled bool
		)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				buf = analyzer.CopyBands(buf)
				mu.Lock()
				if cap(pending) < len(buf) {
					pending = make([]float32, len(buf))
				} else {
					pending = pending[:len(buf)]
				}
				copy(pending, buf)
				if scheduled {
					mu.Unlock()
					continue
				}
				scheduled = true
				mu.Unlock()
				fyne.Do(func() {
					mu.Lock()
					levels := append([]float32(nil), pending...)
					scheduled = false
					mu.Unlock()
					v.surface.setBands(levels)
				})
			}
		}
	}()
}

func (v *Visualizer) stopAudioLocked() {
	if v.vizCancel != nil {
		v.vizCancel()
		v.vizCancel = nil
	}
	if v.analyzer != nil {
		v.analyzer.Stop()
	}
}

func (v *Visualizer) startNowPlayingLocked(parent context.Context) {
	v.stopNowPlayingLocked()
	ctx, cancel := context.WithCancel(parent)
	v.npCancel = cancel
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		v.pushTrack()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				v.pushTrack()
			}
		}
	}()
}

func (v *Visualizer) stopNowPlayingLocked() {
	if v.npCancel != nil {
		v.npCancel()
		v.npCancel = nil
	}
}

func (v *Visualizer) mediaFilter() media.Filter {
	v.mu.Lock()
	defer v.mu.Unlock()
	return media.Filter{Allow: v.cfg.MediaApps, Deny: v.cfg.MediaIgnore}
}

func (v *Visualizer) pushTrack() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("visualizer: now-playing panic recovered: %v", r)
		}
	}()
	tr := media.NowPlayingFiltered(v.mediaFilter())
	fyne.Do(func() {
		v.surface.setTrack(tr)
	})
}

type surface struct {
	widget.BaseWidget
	drag widgetx.Drag

	rootBG *canvas.Rectangle
	bars   []*canvas.Rectangle
	levels []float32
	vizH   float32

	icon   *canvas.Text
	title  *canvas.Text
	artist *canvas.Text

	progressTrack *canvas.Rectangle
	progressFill  *canvas.Rectangle
	progress      float64 // 0–1
	timeLeft      *canvas.Text // elapsed / current
	timeTotal     *canvas.Text // remaining

	fg      color.Color
	iconSz  float32
	textSz  float32
	iconGap float32

	lastTitle    string
	lastArtist   string
	lastOK       bool
	lastProgress float64
	lastHasDur   bool
	lastTimeKey  string

	rawTitle  string // untruncated SMTC title
	rawArtist string
}

func newSurface() *surface {
	trackCol := color.NRGBA{R: 255, G: 255, B: 255, A: 40}
	s := &surface{
		drag:    widgetx.Drag{},
		rootBG:  canvas.NewRectangle(winutil.ClearColor()),
		vizH:    defaultVizH,
		fg:      colorWhite,
		iconSz:  defaultIconSize,
		textSz:  defaultTextSize,
		iconGap: 10,
		icon: &canvas.Text{
			Text:      string(rune(defaultMusicIcon)),
			Color:     colorWhite,
			TextSize:  defaultIconSize,
			Alignment: fyne.TextAlignCenter,
		},
		title: &canvas.Text{
			Color:     colorWhite,
			TextSize:  defaultTextSize,
			Alignment: fyne.TextAlignLeading,
		},
		artist: &canvas.Text{
			Color:     colorWhite,
			TextSize:  defaultTextSize * 0.85,
			Alignment: fyne.TextAlignLeading,
		},
		progressTrack: canvas.NewRectangle(trackCol),
		progressFill:  canvas.NewRectangle(colorWhite),
		timeLeft: &canvas.Text{
			Color:     colorWhite,
			TextSize:  10,
			Alignment: fyne.TextAlignLeading,
		},
		timeTotal: &canvas.Text{
			Color:     colorWhite,
			TextSize:  10,
			Alignment: fyne.TextAlignTrailing,
		},
	}
	s.progressTrack.Hide()
	s.progressFill.Hide()
	s.timeLeft.Hide()
	s.timeTotal.Hide()
	s.ExtendBaseWidget(s)
	return s
}

func (s *surface) SetDraggable(on bool) { s.drag.Enabled = on }

func (s *surface) setBarCount(n int, height float32) {
	if n < 4 {
		n = 4
	}
	s.vizH = height
	if len(s.bars) != n {
		s.bars = make([]*canvas.Rectangle, n)
		s.levels = make([]float32, n)
		for i := range s.bars {
			s.bars[i] = canvas.NewRectangle(colorWhite)
		}
	}
	s.Refresh()
}

func (s *surface) setBands(levels []float32) {
	if len(s.bars) == 0 {
		return
	}
	n := len(s.bars)
	if len(levels) < n {
		n = len(levels)
	}
	for i := 0; i < n; i++ {
		s.levels[i] = levels[i]
	}
	s.layoutBars(s.Size())
	for _, b := range s.bars {
		canvas.Refresh(b)
	}
}

func (s *surface) setTrack(tr media.Track) {
	prog := tr.Progress()
	hasDur := tr.OK && tr.Duration > 0
	remain := tr.Duration - tr.Position
	if remain < 0 {
		remain = 0
	}
	timeKey := ""
	if hasDur {
		timeKey = formatTrackClock(tr.Position) + "|-" + formatTrackClock(remain)
	}
	sameMeta := tr.OK == s.lastOK && tr.Title == s.lastTitle && tr.Artist == s.lastArtist && hasDur == s.lastHasDur
	const progEps = 0.002
	sameProg := abs64(prog-s.lastProgress) < progEps && timeKey == s.lastTimeKey
	if sameMeta && sameProg {
		return
	}
	s.lastOK = tr.OK
	s.lastTitle = tr.Title
	s.lastArtist = tr.Artist
	s.lastProgress = prog
	s.lastHasDur = hasDur
	s.lastTimeKey = timeKey

	if !tr.OK {
		s.title.Text = ""
		s.artist.Text = ""
		s.rawTitle = ""
		s.rawArtist = ""
		s.progress = 0
		s.timeLeft.Text = ""
		s.timeTotal.Text = ""
		s.icon.Hide()
		s.title.Hide()
		s.artist.Hide()
		s.progressTrack.Hide()
		s.progressFill.Hide()
		s.timeLeft.Hide()
		s.timeTotal.Hide()
	} else {
		s.icon.Show()
		s.title.Show()
		s.artist.Show()
		s.rawTitle = tr.Title
		s.rawArtist = tr.Artist
		if s.rawTitle == "" {
			s.rawTitle = tr.Artist
			s.rawArtist = ""
		}
		s.progress = prog
		if hasDur {
			s.timeLeft.Text = formatTrackClock(tr.Position)
			s.timeTotal.Text = "-" + formatTrackClock(remain)
			s.progressTrack.Show()
			s.progressFill.Show()
			s.timeLeft.Show()
			s.timeTotal.Show()
		} else {
			s.timeLeft.Text = ""
			s.timeTotal.Text = ""
			s.progressTrack.Hide()
			s.progressFill.Hide()
			s.timeLeft.Hide()
			s.timeTotal.Hide()
		}
	}
	s.layoutAll(s.Size())
	canvas.Refresh(s.icon)
	canvas.Refresh(s.title)
	canvas.Refresh(s.artist)
	canvas.Refresh(s.progressTrack)
	canvas.Refresh(s.progressFill)
	canvas.Refresh(s.timeLeft)
	canvas.Refresh(s.timeTotal)
}

func formatTrackClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Round(time.Second) / time.Second)
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
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
	vizCol, err := config.ParseColor(cfg.VisualizerColor, colorWhite)
	if err != nil {
		return fmt.Errorf("visualizer_color: %w", err)
	}
	s.fg = col
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
	titleFontPath := cfg.WeekdayFont
	if titleFontPath == "" {
		titleFontPath = defaultTitleFont
	}
	captionFontPath := cfg.LabelFont
	if captionFontPath == "" {
		captionFontPath = defaultLabelFont
	}
	iconRes, err := widgetx.LoadFont(cfg.AssetsDir, iconFontPath)
	if err != nil {
		return fmt.Errorf("icon_font: %w", err)
	}
	titleRes, err := widgetx.LoadFont(cfg.AssetsDir, titleFontPath)
	if err != nil {
		return fmt.Errorf("title_font: %w", err)
	}
	captionRes, err := widgetx.LoadFont(cfg.AssetsDir, captionFontPath)
	if err != nil {
		return fmt.Errorf("label_font: %w", err)
	}

	r := musicIconRune(cfg)
	s.icon.Text = string(r)
	s.icon.Color = col
	s.icon.TextSize = s.iconSz
	s.icon.FontSource = iconRes
	s.title.Color = col
	s.title.TextSize = s.textSz
	s.title.FontSource = titleRes
	s.title.TextStyle = fyne.TextStyle{}
	dim := col
	if c, ok := col.(color.NRGBA); ok {
		c.A = 220
		dim = c
	}
	s.artist.Color = dim
	s.artist.TextSize = widgetx.CaptionSize
	s.artist.FontSource = captionRes
	s.artist.TextStyle = fyne.TextStyle{}
	s.timeLeft.Color = dim
	s.timeLeft.TextSize = widgetx.CaptionSize
	s.timeLeft.FontSource = captionRes
	s.timeLeft.TextStyle = fyne.TextStyle{}
	s.timeTotal.Color = dim
	s.timeTotal.TextSize = widgetx.CaptionSize
	s.timeTotal.FontSource = captionRes
	s.timeTotal.TextStyle = fyne.TextStyle{}
	for _, b := range s.bars {
		b.FillColor = vizCol
	}
	if c, ok := col.(color.NRGBA); ok {
		s.progressTrack.FillColor = color.NRGBA{R: c.R, G: c.G, B: c.B, A: 40}
		s.progressFill.FillColor = c
	} else {
		s.progressTrack.FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 40}
		s.progressFill.FillColor = colorWhite
	}
	return nil
}

func musicIconRune(cfg config.WidgetConfig) rune {
	if r := icons.Rune(cfg.MusicIcon, cfg.MusicIconCode); r != 0 {
		return r
	}
	return defaultMusicIcon
}

func (s *surface) barRegion(size fyne.Size) (vizH, barW, gapX, originX, innerW float32) {
	pad := widgetx.Pad
	innerW = size.Width - pad*2
	if innerW < 40 {
		innerW = size.Width
		pad = 0
	}
	originX = pad

	bottomReserve := s.progressReserve()
	usableH := size.Height - bottomReserve - widgetx.PadY
	if usableH < 1 {
		usableH = 1
	}
	vizH = s.vizH
	if vizH <= 0 {
		vizH = defaultVizH
	}
	if vizH > usableH*0.55 {
		vizH = usableH * 0.55
	}
	n := float32(len(s.bars))
	gapX = float32(1)
	barW = float32(3)
	if n > 0 {
		barW = (innerW - gapX*(n-1)) / n
		if barW < 1 {
			barW = 1
		}
	}
	return vizH, barW, gapX, originX, innerW
}

func (s *surface) progressReserve() float32 {
	if s.progressTrack.Hidden {
		return 0
	}
	const (
		progressH   float32 = 2
		progressGap float32 = 3
	)
	timeH := s.timeLeft.TextSize
	if timeH < 9 {
		timeH = 9
	}
	rowH := progressH
	if timeH > rowH {
		rowH = timeH
	}
	return rowH + progressGap
}

// layoutBars updates only spectrum bar geometry (60 Hz hot path).
func (s *surface) layoutBars(size fyne.Size) {
	vizH, barW, gapX, originX, _ := s.barRegion(size)
	for i, b := range s.bars {
		level := float32(0)
		if i < len(s.levels) {
			level = s.levels[i]
		}
		h := vizH * level
		if h < 1 && level > 0 {
			h = 1
		}
		x := originX + float32(i)*(barW+gapX)
		b.Move(fyne.NewPos(x, widgetx.PadY+vizH-h))
		b.Resize(fyne.NewSize(barW, h))
	}
}

func (s *surface) layoutAll(size fyne.Size) {
	s.rootBG.Move(fyne.NewPos(0, 0))
	s.rootBG.Resize(size)

	bottomReserve := s.progressReserve()
	usableH := size.Height - bottomReserve - widgetx.PadY
	if usableH < 1 {
		usableH = 1
	}

	vizH, _, _, originX, innerW := s.barRegion(size)
	s.layoutBars(size)

	y := widgetx.PadY + vizH + widgetx.RowGap
	remain := usableH - (vizH + widgetx.RowGap)
	if remain < 1 {
		remain = 1
	}

	// Music row fills inner width; icon + text vertically centered.
	iconBox := s.iconSz
	iconW := s.iconSz + 4
	lineH := s.textSz + 1
	if lineH*2+2 > remain {
		lineH = (remain - 2) / 2
	}
	stackH := lineH*2 + 2
	groupH := stackH
	if iconBox > groupH {
		groupH = iconBox
	}
	groupTop := y + (remain-groupH)/2
	if groupTop < y {
		groupTop = y
	}

	textX := originX
	textW := innerW
	iconColW := float32(0)
	if !s.icon.Hidden {
		s.icon.Move(fyne.NewPos(originX, groupTop+(groupH-iconBox)/2))
		s.icon.Resize(fyne.NewSize(iconW, iconBox))
		iconColW = iconW

		textX = originX + iconW + s.iconGap
		textW = originX + innerW - textX
		if textW < 1 {
			textW = 1
		}
		ty := groupTop + (groupH-stackH)/2
		s.title.Text = fitEllipsis(s.rawTitle, s.title.TextSize, textW)
		s.artist.Text = fitEllipsis(s.rawArtist, s.artist.TextSize, textW)
		s.title.Move(fyne.NewPos(textX, ty))
		s.title.Resize(fyne.NewSize(textW, lineH))
		s.artist.Move(fyne.NewPos(textX, ty+lineH+2))
		s.artist.Resize(fyne.NewSize(textW, lineH))
	}

	s.layoutProgress(size, originX, iconColW, textX, textW)
}

// fitEllipsis shortens s so it measures within maxW, appending "…".
func fitEllipsis(s string, size, maxW float32) string {
	if s == "" || maxW <= 0 {
		return s
	}
	style := fyne.TextStyle{}
	if fyne.MeasureText(s, size, style).Width <= maxW {
		return s
	}
	const ell = "…"
	ellW := fyne.MeasureText(ell, size, style).Width
	if ellW >= maxW {
		return ell
	}
	runes := []rune(s)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		cand := string(runes[:mid]) + ell
		if fyne.MeasureText(cand, size, style).Width <= maxW {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo <= 0 {
		return ell
	}
	return string(runes[:lo]) + ell
}

func (s *surface) layoutProgress(size fyne.Size, iconX, iconColW, textX, textW float32) {
	const (
		progressH float32 = 2
		timeGap   float32 = 8
		timeMinW  float32 = 40
	)
	timeH := s.timeLeft.TextSize
	if timeH < 9 {
		timeH = 9
	}
	rowH := progressH
	if timeH > rowH {
		rowH = timeH
	}
	py := size.Height - rowH - widgetx.PadY/2
	if py < 0 {
		py = size.Height - rowH
	}
	ty := py + (rowH-timeH)/2
	by := py + (rowH-progressH)/2

	showTimes := !s.timeLeft.Hidden
	if !showTimes {
		s.progressTrack.Move(fyne.NewPos(textX, by))
		s.progressTrack.Resize(fyne.NewSize(textW, progressH))
		fillW := float32(s.progress) * textW
		if fillW < 0 {
			fillW = 0
		}
		s.progressFill.Move(fyne.NewPos(textX, by))
		s.progressFill.Resize(fyne.NewSize(fillW, progressH))
		return
	}

	// Col 1: elapsed under the music icon.
	leftW := iconColW
	if leftW < timeMinW {
		leftW = timeMinW
	}
	s.timeLeft.Alignment = fyne.TextAlignCenter
	s.timeLeft.Move(fyne.NewPos(iconX, ty))
	s.timeLeft.Resize(fyne.NewSize(leftW, timeH))

	// Col 2: progress bar + remaining, aligned with title/artist.
	rightW := timeMinW
	barW := textW - rightW - timeGap
	if barW < 24 {
		barW = 24
		rightW = textW - barW - timeGap
		if rightW < 28 {
			rightW = 28
		}
	}
	s.progressTrack.Move(fyne.NewPos(textX, by))
	s.progressTrack.Resize(fyne.NewSize(barW, progressH))
	fillW := float32(s.progress) * barW
	if fillW < 0 {
		fillW = 0
	}
	s.progressFill.Move(fyne.NewPos(textX, by))
	s.progressFill.Resize(fyne.NewSize(fillW, progressH))

	s.timeTotal.Alignment = fyne.TextAlignTrailing
	s.timeTotal.Move(fyne.NewPos(textX+textW-rightW, ty))
	s.timeTotal.Resize(fyne.NewSize(rightW, timeH))
}

func (s *surface) CreateRenderer() fyne.WidgetRenderer {
	return &renderer{surface: s}
}

func (s *surface) MinSize() fyne.Size { return fyne.NewSize(120, 64) }

type renderer struct {
	surface *surface
	objs    []fyne.CanvasObject
}

func (r *renderer) Layout(size fyne.Size) { r.surface.layoutAll(size) }
func (r *renderer) MinSize() fyne.Size    { return r.surface.MinSize() }
func (r *renderer) Destroy() {
	r.objs = nil
	r.surface = nil
}

func (r *renderer) Objects() []fyne.CanvasObject {
	s := r.surface
	need := 1 + len(s.bars) + 7
	if cap(r.objs) < need {
		r.objs = make([]fyne.CanvasObject, 0, need)
	}
	r.objs = r.objs[:0]
	r.objs = append(r.objs, s.rootBG)
	for _, b := range s.bars {
		r.objs = append(r.objs, b)
	}
	r.objs = append(r.objs, s.icon, s.title, s.artist, s.progressTrack, s.progressFill, s.timeLeft, s.timeTotal)
	return r.objs
}

func (r *renderer) Refresh() {
	s := r.surface
	s.rootBG.FillColor = winutil.ClearColor()
	s.rootBG.Refresh()
	for _, o := range r.Objects() {
		o.Refresh()
	}
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
