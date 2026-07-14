package icons

import "testing"

func TestRuneCodeOverridesName(t *testing.T) {
	c := &Catalog{byName: map[string]rune{"music": 0xf001}}
	if r := c.Rune("music", "e843"); r != 0xe843 {
		t.Fatalf("got %U", r)
	}
}

func TestRuneNameLookup(t *testing.T) {
	c := &Catalog{byName: map[string]rune{"memory": 0xf538, "fa-memory": 0xf538}}
	if r := c.Rune("fa-memory", ""); r != 0xf538 {
		t.Fatalf("got %U", r)
	}
	if r := c.Rune("nope", ""); r != 0 {
		t.Fatalf("got %U", r)
	}
}
