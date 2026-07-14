package widgetx

import (
	"context"
	"fmt"
	"log"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
	"github.com/RyanMarshCodes/golang-rainmeter/internal/icons"
)

// Instance is a widget panel hosted inside the shell.
type Instance interface {
	ID() string
	Type() string
	Content() fyne.CanvasObject
	SetHost(win fyne.Window)
	Start(ctx context.Context)
	Apply(cfg config.WidgetConfig) error
	Close()
	FlexWeight() float32
	MinSize() fyne.Size
}

// Factory creates a widget instance from config (no personal window).
type Factory func(a fyne.App, cfg config.WidgetConfig) (Instance, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register adds a widget factory for a type name.
func Register(typeName string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[typeName] = f
}

func lookup(typeName string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[typeName]
	return f, ok
}

// Manager owns the shell and live widget panels.
type Manager struct {
	app fyne.App

	mu        sync.Mutex
	shell     *Shell
	instances map[string]Instance
	cancels   map[string]context.CancelFunc
	editMode  bool
	assetsDir string
}

func NewManager(a fyne.App) *Manager {
	return &Manager{
		app:       a,
		instances: map[string]Instance{},
		cancels:   map[string]context.CancelFunc{},
	}
}

func (m *Manager) SetAssetsDir(dir string) {
	m.mu.Lock()
	m.assetsDir = dir
	m.mu.Unlock()
}

func (m *Manager) EditMode() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.editMode
}

func (m *Manager) ShellWindow() fyne.Window {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shell == nil {
		return nil
	}
	return m.shell.Window()
}

// Instances returns a snapshot of live instances.
func (m *Manager) Instances() []Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Instance, 0, len(m.instances))
	for _, inst := range m.instances {
		out = append(out, inst)
	}
	return out
}

// Reconcile creates/updates/removes panels and rebuilds the shell stack.
// Must be called on the Fyne UI thread (via fyne.Do).
func (m *Manager) Reconcile(cfg *config.Config) {
	m.mu.Lock()
	m.editMode = cfg.EditMode
	if m.shell == nil {
		m.shell = NewShell(m.app)
	}
	shell := m.shell
	editMode := m.editMode
	assetsDir := m.assetsDir
	m.mu.Unlock()

	if err := icons.LoadAssets(assetsDir, cfg.IconMap); err != nil {
		log.Printf("icons: load map: %v", err)
	}

	desired := map[string]config.WidgetConfig{}
	for _, w := range cfg.Widgets {
		if !w.Enabled {
			continue
		}
		desired[w.ID] = w
	}

	m.mu.Lock()
	for id, inst := range m.instances {
		if _, ok := desired[id]; !ok {
			if cancel, ok := m.cancels[id]; ok {
				cancel()
				delete(m.cancels, id)
			}
			inst.Close()
			delete(m.instances, id)
		}
	}
	m.mu.Unlock()

	host := shell.Window()
	designW := cfg.Shell.DesignWidth
	if designW <= 0 {
		if cfg.Shell.Width > 0 {
			designW = cfg.Shell.Width
		} else {
			designW = DefaultDesignWidth
		}
	}
	designH := cfg.Shell.DesignHeight
	if designH <= 0 {
		if cfg.Shell.Height > 0 {
			designH = cfg.Shell.Height
		} else {
			designH = DefaultDesignHeight
		}
	}

	for id, wcfg := range desired {
		applyCfg := wcfg
		applyCfg.AssetsDir = assetsDir
		applyCfg.EditMode = editMode
		applyCfg.DesignWidth = designW
		applyCfg.DesignHeight = designH
		bandH := wcfg.Height
		if bandH <= 0 {
			bandH = designH
		}
		applyCfg.DesignBandHeight = bandH

		m.mu.Lock()
		inst, exists := m.instances[id]
		m.mu.Unlock()

		if !exists {
			f, ok := lookup(wcfg.Type)
			if !ok {
				log.Printf("unknown widget type %q for id %q — skipping", wcfg.Type, wcfg.ID)
				continue
			}
			created, err := f(m.app, applyCfg)
			if err != nil {
				log.Printf("create widget %q: %v", wcfg.ID, err)
				continue
			}
			created.SetHost(host)
			ctx, cancel := context.WithCancel(context.Background())
			m.mu.Lock()
			m.instances[id] = created
			m.cancels[id] = cancel
			m.mu.Unlock()
			created.Start(ctx)
			if err := created.Apply(applyCfg); err != nil {
				log.Printf("apply widget %q: %v", id, err)
			}
			continue
		}

		inst.SetHost(host)
		if err := inst.Apply(applyCfg); err != nil {
			log.Printf("apply widget %q: %v", id, err)
		}
	}

	// Build ordered stack.
	order := cfg.Shell.Order
	if len(order) == 0 {
		for _, w := range cfg.Widgets {
			if w.Enabled {
				order = append(order, w.ID)
			}
		}
	}

	var objs []fyne.CanvasObject
	var weights []float32
	var mins []fyne.Size
	m.mu.Lock()
	for _, id := range order {
		inst, ok := m.instances[id]
		if !ok {
			continue
		}
		objs = append(objs, inst.Content())
		weights = append(weights, inst.FlexWeight())
		min := inst.MinSize()
		mins = append(mins, fyne.NewSize(min.Width, inst.FlexWeight()))
	}
	m.mu.Unlock()

	shell.SetPanels(objs, weights, mins)
	shell.Apply(cfg.Shell, editMode)
	shell.Show() // Show centers brand-new windows; Show re-applies native geometry
}

// CloseAll stops every panel and the shell.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cancel := range m.cancels {
		cancel()
	}
	clear(m.cancels)
	for _, inst := range m.instances {
		inst.Close()
	}
	clear(m.instances)
	if m.shell != nil {
		m.shell.Close()
		m.shell = nil
	}
}

// EnsureRegistered is a tiny helper for tests/docs.
func EnsureRegistered(typeName string) error {
	if _, ok := lookup(typeName); !ok {
		return fmt.Errorf("widget type %q not registered", typeName)
	}
	return nil
}
