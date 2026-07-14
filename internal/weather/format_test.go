package weather

import "testing"

func TestDescribeClear(t *testing.T) {
	label, icon := Describe(0)
	if label != "Clear" || icon != "sun" {
		t.Fatalf("got label=%q icon=%q", label, icon)
	}
}

func TestCompassDir(t *testing.T) {
	if CompassDir(0) != "N" {
		t.Fatalf("0 => %s", CompassDir(0))
	}
	if CompassDir(90) != "E" {
		t.Fatalf("90 => %s", CompassDir(90))
	}
	if CompassDir(205) != "SW" {
		t.Fatalf("205 => %s", CompassDir(205))
	}
}

func TestFormatTemp(t *testing.T) {
	if FormatTemp(74.6) != "75°" {
		t.Fatalf("got %s", FormatTemp(74.6))
	}
	if FormatTemp(-1.2) != "-1°" {
		t.Fatalf("got %s", FormatTemp(-1.2))
	}
}

func TestLooksUSZIP(t *testing.T) {
	if !looksUSZIP("10001") {
		t.Fatal("10001 should look like US ZIP")
	}
	if looksUSZIP("M5V") {
		t.Fatal("M5V should not look like US ZIP")
	}
}
