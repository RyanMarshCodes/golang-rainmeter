package sysinfo

import "testing"

func TestFormatBytesAuto(t *testing.T) {
	cases := map[uint64]string{
		500:                       "500B",
		5 * 1024 * 1024:           "5MB",
		200 * 1024 * 1024 * 1024:  "200GB",
		1024 * 1024 * 1024 * 1024: "1TB",
	}
	for n, want := range cases {
		if got := FormatBytesAuto(n); got != want {
			t.Fatalf("FormatBytesAuto(%d)=%q want %q", n, got, want)
		}
	}
}

func TestFormatBytesShort(t *testing.T) {
	used := uint64(325) * 1024 * 1024 * 1024
	total := uint64(19) * 1024 * 1024 * 1024 * 1024 / 10 // 1.9TB
	if got := FormatBytesShort(used); got != "325G" {
		t.Fatalf("used short: %q", got)
	}
	if got := FormatBytesShort(total); got != "1.9T" {
		t.Fatalf("total short: %q", got)
	}
}

func TestFormatStorageLines(t *testing.T) {
	used := uint64(200) * 1024 * 1024 * 1024
	total := uint64(1024) * 1024 * 1024 * 1024
	p, s := FormatStorageLines("C:", used, total, true)
	if p != "C: 200 G / 1 T" || s != "20%" {
		t.Fatalf("storage lines: %q %q", p, s)
	}
	p, s = FormatStorageLines("D:", 0, 0, false)
	if p != "D: —" || s != "" {
		t.Fatalf("offline: %q %q", p, s)
	}
}

func TestFormatStorageStack(t *testing.T) {
	const gb = 1024 * 1024 * 1024
	used := uint64(3237) * gb / 10
	total := uint64(19) * gb * 1024 / 10
	drive, capacity, percent := FormatStorageStack("C:", used, total, true)
	if drive != "C:" {
		t.Fatalf("drive: %q", drive)
	}
	if capacity != "323.7 G / 1.9 T" {
		t.Fatalf("capacity: %q", capacity)
	}
	if percent == "" {
		t.Fatal("expected percent")
	}
	drive, capacity, percent = FormatStorageStack("E:", 0, 0, false)
	if drive != "E:" || capacity != "—" || percent != "" {
		t.Fatalf("offline stack: %q %q %q", drive, capacity, percent)
	}
}

func TestFormatNetworkLines(t *testing.T) {
	up, down := FormatNetworkLines(1.5*1024*1024, 400*1024, true)
	if up != "↑\u20071.5M" || down != "↓\u2007400K" {
		t.Fatalf("network: %q %q", up, down)
	}
	up, down = FormatNetworkLines(8*1024, 3*1024, true)
	if up != "↑\u2007\u2007\u20078K" || down != "↓\u2007\u2007\u20073K" {
		t.Fatalf("network low: %q %q", up, down)
	}
	up, down = FormatNetworkLines(0, 0, false)
	if up != "—" || down != "" {
		t.Fatalf("network offline: %q %q", up, down)
	}
}

func TestFormatUsageLine(t *testing.T) {
	if got := FormatUsageLine("CPU", 50, true); got != "50%" {
		t.Fatalf("usage: %q", got)
	}
	if got := FormatUsageLine("GPU", 0, false); got != "—" {
		t.Fatalf("usage missing: %q", got)
	}
}

func TestFormatTempLine(t *testing.T) {
	if got := FormatTempLine(Temp{Celsius: 72.4, OK: true}); got != "72°" {
		t.Fatalf("temp: %q", got)
	}
	if got := FormatTempLine(Temp{}); got != "" {
		t.Fatalf("temp missing: %q", got)
	}
}
