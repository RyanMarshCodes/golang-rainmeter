package app

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/systray"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx"
	_ "github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx/clock"      // register clock
	_ "github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx/metrics"    // register metrics
	_ "github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx/visualizer" // register visualizer
	_ "github.com/RyanMarshCodes/golang-rainmeter/internal/widgetx/weather"    // register weather
	"github.com/RyanMarshCodes/golang-rainmeter/internal/winutil"
)

// App is the tray-hosted shell around the widget manager.
type App struct {
	fyne    fyne.App
	store   *config.Store
	manager *widgetx.Manager

	mu     sync.Mutex
	cfg    *config.Config
	stopWatch func()

	geomDebounce *time.Timer
	lastGeom     map[string]geom

	trayTapMu   sync.Mutex
	lastTrayTap time.Time
}

type geom struct {
	X, Y, W, H int
}

// Run boots the Fyne app, loads config, and blocks until quit.
func Run(configPath string) error {
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	a := &App{
		fyne:     fyneapp.NewWithID("com.github.RyanMarshCodes.golang-rainmeter"),
		store:    config.NewStore(configPath),
		lastGeom: map[string]geom{},
	}
	a.fyne.Settings().SetTheme(newSkinTheme())
	a.manager = widgetx.NewManager(a.fyne)
	a.manager.SetAssetsDir(a.store.AssetsDir())

	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	a.cfg = cfg

	a.setupTray()
	a.reloadUI()

	stop, err := config.Watch(a.store, func() {
		fyne.Do(func() {
			a.reloadFromDisk()
		})
	})
	if err != nil {
		log.Printf("config watch disabled: %v", err)
	} else {
		a.stopWatch = stop
	}

	go a.geometryLoop()

	a.fyne.Run()
	if a.stopWatch != nil {
		a.stopWatch()
	}
	a.manager.CloseAll()
	return nil
}

func defaultConfigPath() string {
	candidates := []string{
		filepath.Join("config", "config.yml"),
		"config.yml",
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append([]string{
			filepath.Join(dir, "config", "config.yml"),
			filepath.Join(dir, "config.yml"),
		}, candidates...)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join("config", "config.yml")
}

func (a *App) setupTray() {
	desk, ok := a.fyne.(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayMenu(a.buildMenu())
	a.installTrayDefaultAction()
}

// installTrayDefaultAction binds left double-tap on the tray icon to edit mode.
// Fyne/systray do not expose double-click; two left taps within 400ms count as one.
// Right-click still opens the tray menu.
func (a *App) installTrayDefaultAction() {
	systray.SetOnTapped(a.onTrayIconTap)
}

func (a *App) onTrayIconTap() {
	a.trayTapMu.Lock()
	now := time.Now()
	if !a.lastTrayTap.IsZero() && now.Sub(a.lastTrayTap) < 400*time.Millisecond {
		a.lastTrayTap = time.Time{}
		a.trayTapMu.Unlock()
		fyne.Do(func() { a.toggleEditMode() })
		return
	}
	a.lastTrayTap = now
	a.trayTapMu.Unlock()
}

func (a *App) buildMenu() *fyne.Menu {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()

	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Reload config", func() {
			fyne.Do(func() { a.reloadFromDisk() })
		}),
	}

	editLabel := "Enter edit mode"
	if cfg != nil && cfg.EditMode {
		editLabel = "Exit edit mode"
	}
	items = append(items, fyne.NewMenuItem(editLabel, func() {
		fyne.Do(func() { a.toggleEditMode() })
	}))

	items = append(items, fyne.NewMenuItemSeparator())

	shellVisible := a.isShellVisible()
	showItem := fyne.NewMenuItem("Show overlay", func() {
		fyne.Do(func() { a.toggleShellVisible() })
	})
	showItem.Checked = shellVisible
	items = append(items, showItem)

	if cfg != nil && len(cfg.Widgets) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
		for _, w := range a.trayWidgetList(cfg) {
			id := w.ID
			title := w.Title
			if title == "" {
				title = id
			}
			item := fyne.NewMenuItem(title, func() {
				fyne.Do(func() { a.toggleWidgetEnabled(id) })
			})
			item.Checked = w.Enabled
			items = append(items, item)
		}
	}

	items = append(items,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			fyne.Do(func() {
				a.fyne.Quit()
			})
		}),
	)
	return fyne.NewMenu("rmgo", items...)
}

// trayWidgetList returns widgets in shell.order first, then any extras.
func (a *App) trayWidgetList(cfg *config.Config) []config.WidgetConfig {
	byID := map[string]config.WidgetConfig{}
	for _, w := range cfg.Widgets {
		byID[w.ID] = w
	}
	var out []config.WidgetConfig
	seen := map[string]struct{}{}
	for _, id := range cfg.Shell.Order {
		w, ok := byID[id]
		if !ok {
			continue
		}
		out = append(out, w)
		seen[id] = struct{}{}
	}
	for _, w := range cfg.Widgets {
		if _, ok := seen[w.ID]; ok {
			continue
		}
		out = append(out, w)
	}
	return out
}

func (a *App) refreshTray() {
	if desk, ok := a.fyne.(desktop.App); ok {
		desk.SetSystemTrayMenu(a.buildMenu())
	}
	a.installTrayDefaultAction()
}

func (a *App) isShellVisible() bool {
	win := a.manager.ShellWindow()
	return win != nil && winutil.IsVisible(win)
}

func (a *App) toggleShellVisible() {
	win := a.manager.ShellWindow()
	if win == nil {
		return
	}
	if winutil.IsVisible(win) {
		win.Hide()
	} else {
		win.Show()
	}
	a.refreshTray()
}

func (a *App) toggleWidgetEnabled(id string) {
	a.mu.Lock()
	if a.cfg == nil {
		a.mu.Unlock()
		return
	}
	next := cloneConfig(a.cfg)
	wc := next.WidgetByID(id)
	if wc == nil {
		a.mu.Unlock()
		return
	}
	wc.Enabled = !wc.Enabled
	a.mu.Unlock()

	if err := a.store.Save(next); err != nil {
		log.Printf("save widget enabled: %v", err)
		return
	}
	a.mu.Lock()
	a.cfg = next
	a.mu.Unlock()
	a.reloadUI()
}

func (a *App) reloadFromDisk() {
	cfg, err := a.store.Load()
	if err != nil {
		log.Printf("reload config: %v", err)
		return
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	a.reloadUI()
}

func (a *App) reloadUI() {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if cfg == nil {
		return
	}
	a.manager.Reconcile(cfg)
	a.refreshTray()
}

func (a *App) toggleEditMode() {
	a.mu.Lock()
	if a.cfg == nil {
		a.mu.Unlock()
		return
	}
	a.cfg.EditMode = !a.cfg.EditMode
	cfg := cloneConfig(a.cfg)
	a.mu.Unlock()

	if err := a.store.Save(cfg); err != nil {
		log.Printf("save edit_mode: %v", err)
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	a.reloadUI()
}

func (a *App) geometryLoop() {
	t := time.NewTicker(350 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		a.mu.Lock()
		editing := a.cfg != nil && a.cfg.EditMode
		a.mu.Unlock()
		if !editing {
			continue
		}
		fyne.Do(func() {
			a.captureGeometry()
		})
	}
}

func (a *App) captureGeometry() {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if cfg == nil || !cfg.EditMode {
		return
	}

	win := a.manager.ShellWindow()
	if win == nil {
		return
	}
	x, y, w, h, ok := winutil.ClientBounds(win)
	if !ok {
		return
	}
	g := geom{X: x, Y: y, W: w, H: h}
	prev, seen := a.lastGeom["shell"]
	if seen && prev == g {
		return
	}
	a.lastGeom["shell"] = g

	if cfg.Shell.X == x && cfg.Shell.Y == y && int(cfg.Shell.Width) == w && int(cfg.Shell.Height) == h {
		return
	}

	next := cloneConfig(cfg)
	next.Shell.X = x
	next.Shell.Y = y
	next.Shell.Width = float32(w)
	next.Shell.Height = float32(h)

	if a.geomDebounce != nil {
		a.geomDebounce.Stop()
	}
	snapshot := next
	a.geomDebounce = time.AfterFunc(400*time.Millisecond, func() {
		if err := a.store.Save(snapshot); err != nil {
			log.Printf("save geometry: %v", err)
			return
		}
		a.mu.Lock()
		a.cfg = snapshot
		a.mu.Unlock()
	})
}

func cloneConfig(cfg *config.Config) *config.Config {
	out := *cfg
	out.Widgets = append([]config.WidgetConfig(nil), cfg.Widgets...)
	out.Shell.Order = append([]string(nil), cfg.Shell.Order...)
	return &out
}
