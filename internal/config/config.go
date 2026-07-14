package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root on-disk configuration.
type Config struct {
	EditMode bool           `yaml:"edit_mode,omitempty"`
	IconMap  string         `yaml:"icon_map,omitempty"` // relative to assets/; default fonts/icons/icon-map.json
	Shell    ShellConfig    `yaml:"shell,omitempty"`
	Widgets  []WidgetConfig `yaml:"widgets"`
}

// ShellConfig is the single desktop overlay that stacks widgets.
type ShellConfig struct {
	X            int      `yaml:"x,omitempty"`
	Y            int      `yaml:"y,omitempty"`
	Width        float32  `yaml:"width,omitempty"`
	Height       float32  `yaml:"height,omitempty"`
	Gap          float32  `yaml:"gap,omitempty"`
	Order        []string `yaml:"order,omitempty"` // widget ids, top → bottom
	AlwaysOnTop  bool     `yaml:"always_on_top,omitempty"`
	Transparent  bool     `yaml:"transparent,omitempty"`
	ClickThrough bool     `yaml:"click_through,omitempty"`
	Opacity      float32  `yaml:"opacity,omitempty"`
}

// WidgetConfig describes one widget instance window.
type WidgetConfig struct {
	ID      string `yaml:"id"`
	Type    string `yaml:"type"`
	Enabled bool   `yaml:"enabled"`
	Title   string `yaml:"title,omitempty"`

	// Clock
	Format          string  `yaml:"format,omitempty"`
	WeekdayFont     string  `yaml:"weekday_font,omitempty"`
	DetailFont      string  `yaml:"detail_font,omitempty"`
	TimeColor       string  `yaml:"time_color,omitempty"`
	DateColor       string  `yaml:"date_color,omitempty"`
	RuleColor       string  `yaml:"rule_color,omitempty"`
	VisualizerBars  int     `yaml:"visualizer_bars,omitempty"`
	VisualizerColor string  `yaml:"visualizer_color,omitempty"`
	VisualizerHeight float32 `yaml:"visualizer_height,omitempty"`

	// Metrics
	Color      string          `yaml:"color,omitempty"`
	IconFont   string          `yaml:"icon_font,omitempty"`
	LabelFont  string          `yaml:"label_font,omitempty"`
	IconSize   float32         `yaml:"icon_size,omitempty"`
	TextSize   float32         `yaml:"text_size,omitempty"`
	Columns    int             `yaml:"columns,omitempty"`
	GapX       float32         `yaml:"gap_x,omitempty"`
	GapY       float32         `yaml:"gap_y,omitempty"`
	IntervalMS int             `yaml:"interval_ms,omitempty"`
	Measures   []MeasureConfig `yaml:"measures,omitempty"`

	// Visualizer widget + optional music icon for now-playing row
	MusicIcon     string   `yaml:"music_icon,omitempty"`
	MusicIconCode string   `yaml:"music_icon_code,omitempty"`
	MediaApps     []string `yaml:"media_apps,omitempty"`    // allowlist (substring on SMTC app id + title/artist/album); empty = any
	MediaIgnore   []string `yaml:"media_ignore,omitempty"`  // denylist (same match fields); prefer allowlist alone


	// Weather (Open-Meteo geocode: city name, US ZIP, etc.)
	Place       string `yaml:"place,omitempty"`
	Zip         string `yaml:"zip,omitempty"` // deprecated alias for place
	Units       string `yaml:"units,omitempty"`        // f (default) or c
	AccentColor string `yaml:"accent_color,omitempty"` // top rule
	PanelColor  string `yaml:"panel_color,omitempty"`  // optional card fill; empty → fully transparent

	// Window / desktop
	X            int     `yaml:"x,omitempty"`
	Y            int     `yaml:"y,omitempty"`
	Width        float32 `yaml:"width,omitempty"`
	Height       float32 `yaml:"height,omitempty"`
	AlwaysOnTop  bool    `yaml:"always_on_top,omitempty"`
	Transparent  bool    `yaml:"transparent,omitempty"`
	ClickThrough bool    `yaml:"click_through,omitempty"`
	Opacity      float32 `yaml:"opacity,omitempty"` // 0–1 whole-window alpha; omit / 0 = fully opaque

	// AssetsDir is set at runtime (not serialized) for resolving relative resource paths.
	AssetsDir string `yaml:"-"`
	// EditMode is set at runtime by the manager (not serialized).
	EditMode bool `yaml:"-"`
}

// MeasureConfig describes one system metric cell in a metrics widget.
type MeasureConfig struct {
	Kind     string `yaml:"kind"`
	Icon     string `yaml:"icon,omitempty"`
	IconCode string `yaml:"icon_code,omitempty"`
	Label    string `yaml:"label,omitempty"`
	Device   string `yaml:"device,omitempty"`
	GPU      int    `yaml:"gpu,omitempty"`
}

// Store loads and saves config with an ignore window for self-writes.
type Store struct {
	path string

	mu           sync.Mutex
	ignoreUntil  time.Time
	ignoreDigest string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string { return s.path }

// Dir returns the directory containing the config file.
func (s *Store) Dir() string {
	return filepath.Dir(s.path)
}

// AssetsDir returns the project assets/ directory used for fonts and other resources.
func (s *Store) AssetsDir() string {
	return FindAssetsDir(s.path)
}

// ResolveAsset joins a relative path against assets/. Absolute paths are returned as-is.
func (s *Store) ResolveAsset(p string) string {
	return ResolveAsset(s.AssetsDir(), p)
}

// FindAssetsDir locates the assets folder relative to the config file (or cwd).
func FindAssetsDir(configPath string) string {
	configDir := filepath.Dir(configPath)
	candidates := []string{
		filepath.Join(configDir, "..", "assets"), // config/config.yml → ../assets
		filepath.Join(configDir, "assets"),
		"assets",
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "assets"),
			filepath.Join(exeDir, "..", "assets"),
		)
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			return abs
		}
	}
	// Stable default even if the folder is missing yet.
	abs, err := filepath.Abs(filepath.Join(configDir, "..", "assets"))
	if err != nil {
		return filepath.Join(configDir, "..", "assets")
	}
	return abs
}

// ResolveAsset joins rel against assetsDir. Empty input stays empty.
func ResolveAsset(assetsDir, rel string) string {
	if rel == "" {
		return ""
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(assetsDir, rel)
}

func (s *Store) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		example := filepath.Join(filepath.Dir(s.path), "config.example.yml")
		ex, exErr := os.ReadFile(example)
		if exErr != nil {
			return nil, fmt.Errorf("missing %s (and no config.example.yml to copy): %w", s.path, err)
		}
		if writeErr := os.WriteFile(s.path, ex, 0o644); writeErr != nil {
			return nil, fmt.Errorf("create config from example: %w", writeErr)
		}
		data = ex
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) Save(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	_ = enc.Close()
	data := buf.Bytes()

	s.mu.Lock()
	s.ignoreUntil = time.Now().Add(750 * time.Millisecond)
	s.ignoreDigest = digest(data)
	s.mu.Unlock()

	return os.WriteFile(s.path, data, 0o644)
}

// ShouldIgnoreReload reports whether a filesystem event should be skipped
// because it was caused by our own Save.
func (s *Store) ShouldIgnoreReload() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.ignoreUntil) {
		return true
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return false
	}
	return digest(data) == s.ignoreDigest && s.ignoreDigest != ""
}

func (c *Config) Validate() error {
	seen := map[string]struct{}{}
	for i, w := range c.Widgets {
		if w.ID == "" {
			return fmt.Errorf("widgets[%d]: id is required", i)
		}
		if w.Type == "" {
			return fmt.Errorf("widget %q: type is required", w.ID)
		}
		if _, ok := seen[w.ID]; ok {
			return fmt.Errorf("duplicate widget id %q", w.ID)
		}
		seen[w.ID] = struct{}{}
		if w.Width < 0 || w.Height < 0 {
			return fmt.Errorf("widget %q: width/height must be >= 0", w.ID)
		}
		if err := validateMeasures(w); err != nil {
			return err
		}
		if w.Type == "weather" && w.WeatherPlace() == "" {
			return fmt.Errorf("widget %q: weather type requires place (city name or US ZIP)", w.ID)
		}
	}
	if c.Shell.Width < 0 || c.Shell.Height < 0 {
		return fmt.Errorf("shell: width/height must be >= 0")
	}
	for _, id := range c.Shell.Order {
		if _, ok := seen[id]; !ok {
			return fmt.Errorf("shell.order: unknown widget id %q", id)
		}
	}
	return nil
}

// WeatherPlace returns the weather location query (place, or legacy zip).
func (w WidgetConfig) WeatherPlace() string {
	if p := strings.TrimSpace(w.Place); p != "" {
		return p
	}
	return strings.TrimSpace(w.Zip)
}

func validateMeasures(w WidgetConfig) error {
	if w.Type != "metrics" {
		return nil
	}
	if len(w.Measures) == 0 {
		return fmt.Errorf("widget %q: metrics type requires at least one measure", w.ID)
	}
	for i, m := range w.Measures {
		switch m.Kind {
		case "cpu", "gpu", "memory", "network":
			// ok
		case "storage":
			if m.Device == "" {
				return fmt.Errorf("widget %q measures[%d]: storage requires device", w.ID, i)
			}
		case "":
			return fmt.Errorf("widget %q measures[%d]: kind is required", w.ID, i)
		default:
			return fmt.Errorf("widget %q measures[%d]: unknown kind %q (cpu|gpu|memory|storage|network)", w.ID, i, m.Kind)
		}
	}
	return nil
}

// WidgetByID returns a pointer to the widget config with the given id.
func (c *Config) WidgetByID(id string) *WidgetConfig {
	for i := range c.Widgets {
		if c.Widgets[i].ID == id {
			return &c.Widgets[i]
		}
	}
	return nil
}

func digest(data []byte) string {
	// cheap content fingerprint for ignore-own-write
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return fmt.Sprintf("%x", h)
}
