package config

import (
	"image/color"
	"testing"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		in   string
		want color.NRGBA
	}{
		{"#fff", color.NRGBA{255, 255, 255, 255}},
		{"#FFFFFF", color.NRGBA{255, 255, 255, 255}},
		{"#FFFFFFFF", color.NRGBA{255, 255, 255, 255}},
		{"#FF000080", color.NRGBA{255, 0, 0, 128}},
		{"255,255,255", color.NRGBA{255, 255, 255, 255}},
		{"255, 0, 0, 128", color.NRGBA{255, 0, 0, 128}},
	}
	for _, tc := range tests {
		got, err := ParseColor(tc.in, color.NRGBA{})
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		nrgba, ok := got.(color.NRGBA)
		if !ok {
			t.Fatalf("%q: got %T", tc.in, got)
		}
		if nrgba != tc.want {
			t.Fatalf("%q: got %#v want %#v", tc.in, nrgba, tc.want)
		}
	}
	fb := color.NRGBA{1, 2, 3, 4}
	got, err := ParseColor("", fb)
	if err != nil || got != fb {
		t.Fatalf("empty fallback: got %#v err %v", got, err)
	}
}
