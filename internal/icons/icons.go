package icons

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// DefaultMapPath is relative to the assets directory.
const DefaultMapPath = "fonts/icons/icon-map.json"

// Catalog maps logical icon names → font codepoints.
// Any icon font works; names are yours to define in icon-map.json.
type Catalog struct {
	mu     sync.RWMutex
	byName map[string]rune
}

var defaultCatalog = &Catalog{byName: map[string]rune{}}

// Default returns the process-wide catalog (loaded from assets at startup).
func Default() *Catalog { return defaultCatalog }

// LoadFile replaces the catalog from a JSON map file.
//
//	{
//	  "icons": {
//	    "music": "f001",
//	    "cloud": "f0c2"
//	  }
//	}
func (c *Catalog) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw struct {
		Icons map[string]string `json:"icons"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("icon map: %w", err)
	}
	next := make(map[string]rune, len(raw.Icons))
	for name, hex := range raw.Icons {
		name = normalizeName(name)
		if name == "" {
			continue
		}
		r, ok := parseHexRune(hex)
		if !ok {
			continue
		}
		next[name] = r
		// Accept "fa-music" as an alias of "music".
		if !strings.HasPrefix(name, "fa-") {
			next["fa-"+name] = r
		}
	}
	c.mu.Lock()
	c.byName = next
	c.mu.Unlock()
	return nil
}

// LoadAssets loads pathRel under assetsDir. Empty pathRel uses DefaultMapPath.
func LoadAssets(assetsDir, pathRel string) error {
	if pathRel == "" {
		pathRel = DefaultMapPath
	}
	path := pathRel
	if assetsDir != "" && !filepath.IsAbs(pathRel) {
		path = filepath.Join(assetsDir, pathRel)
	}
	return Default().LoadFile(path)
}

// Rune resolves an icon: explicit hex icon_code wins, then name lookup.
func Rune(name, codeHex string) rune {
	return Default().Rune(name, codeHex)
}

// Rune resolves an icon from this catalog.
func (c *Catalog) Rune(name, codeHex string) rune {
	if r, ok := parseHexRune(codeHex); ok {
		return r
	}
	name = normalizeName(name)
	if name == "" {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.byName[name]
}

func normalizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.TrimPrefix(name, "fa-")
}

func parseHexRune(codeHex string) (rune, bool) {
	codeHex = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(codeHex), "0x"))
	if codeHex == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(codeHex, 16, 32)
	if err != nil || v <= 0 {
		return 0, false
	}
	return rune(v), true
}
