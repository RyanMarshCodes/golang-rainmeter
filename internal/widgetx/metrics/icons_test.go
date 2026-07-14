package metrics

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/RyanMarshCodes/golang-rainmeter/internal/icons"
)

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	// internal/widgetx/metrics → repo root assets
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	_ = icons.LoadAssets(filepath.Join(root, "assets"), "")
	m.Run()
}

func TestIconRune(t *testing.T) {
	if r := IconRune("computer", ""); r != 0xe4e5 {
		t.Fatalf("computer: %U", r)
	}
	if r := IconRune("desktop", ""); r != 0xf390 {
		t.Fatalf("desktop: %U", r)
	}
	if r := IconRune("fa-memory", ""); r != 0xf538 {
		t.Fatalf("memory: %U", r)
	}
	if r := IconRune("router", ""); r != 0xf1eb { // free wifi stand-in
		t.Fatalf("router: %U", r)
	}
	if r := IconRune("gpu", ""); r != 0xf2db { // free microchip stand-in
		t.Fatalf("gpu: %U", r)
	}
	if r := IconRune("", "e843"); r != 0xe843 {
		t.Fatalf("code: %U", r)
	}
	if r := IconRune("nope", ""); r != 0 {
		t.Fatalf("missing: %U", r)
	}
}
