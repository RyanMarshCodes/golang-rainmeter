package widgetx

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/winutil"
)

// Shell is the single splash window that stacks widget panels.
type Shell struct {
	win      fyne.Window
	root     *shellRoot
	stack    *fyne.Container
	layout   *vflexLayout
	cfg      config.ShellConfig
	editMode bool
}

type shellRoot struct {
	widget.BaseWidget
	bg     *canvas.Rectangle
	stack  *fyne.Container
	border *canvas.Rectangle
	shell  *Shell
}

func newShellRoot(s *Shell, stack *fyne.Container) *shellRoot {
	r := &shellRoot{
		bg:     canvas.NewRectangle(winutil.ClearColor()),
		stack:  stack,
		border: NewEditBorder(),
		shell:  s,
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *shellRoot) CreateRenderer() fyne.WidgetRenderer {
	return &shellRootRenderer{root: r}
}

func (r *shellRoot) MinSize() fyne.Size { return fyne.NewSize(120, 120) }

type shellRootRenderer struct {
	root *shellRoot
}

func (r *shellRootRenderer) Layout(size fyne.Size) {
	r.root.bg.Resize(size)
	r.root.stack.Resize(size)
	r.root.stack.Move(fyne.NewPos(0, 0))
	LayoutEditBorder(r.root.border, size, r.root.shell.editMode)
}
func (r *shellRootRenderer) MinSize() fyne.Size { return r.root.MinSize() }
func (r *shellRootRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root.bg, r.root.stack, r.root.border}
}
func (r *shellRootRenderer) Refresh() {
	r.root.bg.FillColor = winutil.ClearColor()
	r.root.bg.Refresh()
	r.root.stack.Refresh()
	LayoutEditBorder(r.root.border, r.root.Size(), r.root.shell.editMode)
	r.root.border.Refresh()
}
func (r *shellRootRenderer) Destroy() {}

// vflexLayout stacks objects top-to-bottom with a gap and distributes extra
// height by weight (proportional to FlexWeight).
type vflexLayout struct {
	gap     float32
	weights []float32
	mins    []fyne.Size
}

func (l *vflexLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	n := len(objects)
	if n == 0 {
		return
	}
	for len(l.weights) < n {
		l.weights = append(l.weights, 1)
	}
	for len(l.mins) < n {
		l.mins = append(l.mins, fyne.NewSize(0, 40))
	}

	gaps := float32(0)
	if n > 1 {
		gaps = l.gap * float32(n-1)
	}
	minSum := float32(0)
	weightSum := float32(0)
	for i := 0; i < n; i++ {
		minSum += l.mins[i].Height
		w := l.weights[i]
		if w <= 0 {
			w = 1
		}
		weightSum += w
	}
	extra := size.Height - gaps - minSum
	if extra < 0 {
		extra = 0
	}

	y := float32(0)
	for i, o := range objects {
		w := l.weights[i]
		if w <= 0 {
			w = 1
		}
		h := l.mins[i].Height
		if weightSum > 0 {
			h += extra * (w / weightSum)
		}
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, h))
		y += h + l.gap
	}
}

func (l *vflexLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	n := len(objects)
	if n == 0 {
		return fyne.NewSize(200, 200)
	}
	for len(l.mins) < n {
		l.mins = append(l.mins, fyne.NewSize(0, 40))
	}
	w := float32(120)
	h := float32(0)
	for i := 0; i < n; i++ {
		if l.mins[i].Width > w {
			w = l.mins[i].Width
		}
		h += l.mins[i].Height
	}
	if n > 1 {
		h += l.gap * float32(n-1)
	}
	return fyne.NewSize(w, h)
}

// NewShell creates the stacked splash host.
func NewShell(a fyne.App) *Shell {
	layout := &vflexLayout{gap: RowGap}
	stack := container.New(layout)
	s := &Shell{layout: layout, stack: stack}
	var win fyne.Window
	if drv, ok := a.Driver().(desktop.Driver); ok {
		win = drv.CreateSplashWindow()
	} else {
		win = a.NewWindow("rmgo")
	}
	win.SetTitle("rmgo")
	win.SetPadded(false)
	win.SetFixedSize(false)
	s.win = win
	root := newShellRoot(s, stack)
	s.root = root
	win.SetContent(root)
	win.SetCloseIntercept(func() { win.Hide() })
	return s
}

func (s *Shell) Window() fyne.Window { return s.win }

func (s *Shell) Apply(cfg config.ShellConfig, editMode bool) {
	s.cfg = cfg
	s.editMode = editMode

	gap := cfg.Gap
	if gap <= 0 {
		gap = RowGap
	}
	s.layout.gap = gap

	w, h := cfg.Width, cfg.Height
	if w <= 0 {
		w = 360
	}
	if h <= 0 {
		h = 680
	}
	s.win.Resize(fyne.NewSize(w, h))
	s.applyNative(int(w), int(h))
	if s.root != nil {
		s.root.Refresh()
	}
}

func (s *Shell) applyNative(w, h int) {
	clickThrough := s.cfg.ClickThrough
	if s.editMode {
		clickThrough = false
	}
	// Size is applied as client bounds after chrome so title-bar height
	// doesn't get baked into saved shell width/height.
	if winutil.ApplyDesktopProps(s.win, s.cfg.X, s.cfg.Y, 0, 0, s.cfg.AlwaysOnTop, s.cfg.Transparent, clickThrough, s.cfg.Opacity) {
		winutil.SetNativeChrome(s.win, s.editMode)
		winutil.SetClientBounds(s.win, s.cfg.X, s.cfg.Y, w, h)
		return
	}
	// HWND often missing until after first Show — retry on the UI thread.
	go func() {
		for i := 0; i < 40; i++ {
			time.Sleep(25 * time.Millisecond)
			ok := false
			fyne.DoAndWait(func() {
				if s.win == nil {
					return
				}
				ok = winutil.ApplyDesktopProps(s.win, s.cfg.X, s.cfg.Y, 0, 0, s.cfg.AlwaysOnTop, s.cfg.Transparent, clickThrough, s.cfg.Opacity)
				if ok {
					winutil.SetNativeChrome(s.win, s.editMode)
					winutil.SetClientBounds(s.win, s.cfg.X, s.cfg.Y, w, h)
				}
			})
			if ok {
				return
			}
		}
	}()
}

func (s *Shell) Show() {
	s.win.Show()
	// Fyne centers brand-new windows on Show; re-apply after the HWND exists.
	w, h := s.cfg.Width, s.cfg.Height
	if w <= 0 {
		w = 360
	}
	if h <= 0 {
		h = 680
	}
	s.applyNative(int(w), int(h))
}

// SetPanels replaces stack children; weights/mins parallel to panels.
func (s *Shell) SetPanels(objs []fyne.CanvasObject, weights []float32, mins []fyne.Size) {
	s.layout.weights = append([]float32(nil), weights...)
	s.layout.mins = append([]fyne.Size(nil), mins...)
	s.stack.Objects = objs
	s.stack.Refresh()
	if r := s.win.Content(); r != nil {
		r.Refresh()
	}
}

func (s *Shell) Hide() { s.win.Hide() }

func (s *Shell) Close() {
	if s.win != nil {
		s.win.SetCloseIntercept(nil)
		s.win.Close()
		s.win = nil
	}
}
