package widgetx

import (
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/config"
)

var fontCache sync.Map // abs path → fyne.Resource

// LoadFont reads a font from assets (or absolute path) and caches by resolved path.
func LoadFont(assetsDir, path string) (fyne.Resource, error) {
	if path == "" {
		return nil, nil
	}
	full := config.ResolveAsset(assetsDir, path)
	if full == "" {
		return nil, nil
	}
	if cached, ok := fontCache.Load(full); ok {
		return cached.(fyne.Resource), nil
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	res := fyne.NewStaticResource(filepath.Base(full), data)
	actual, _ := fontCache.LoadOrStore(full, res)
	return actual.(fyne.Resource), nil
}
