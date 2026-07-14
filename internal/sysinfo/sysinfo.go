package sysinfo

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Usage is a 0–100 percentage reading.
type Usage struct {
	Percent float64
	OK      bool
}

// Storage is disk capacity usage.
type Storage struct {
	UsedBytes  uint64
	TotalBytes uint64
	OK         bool
}

// CPUPercent returns total CPU utilization over intervalSec seconds.
func CPUPercent(intervalSec float64) Usage {
	return cpuPercent(intervalSec)
}

// MemoryPercent returns used / total virtual memory as a percentage.
func MemoryPercent() Usage {
	return memoryPercent()
}

// DiskUsage returns used/total bytes for a mount path (e.g. `C:\`).
func DiskUsage(path string) Storage {
	return diskUsage(path)
}

// GPUPercent returns GPU utilization for GPU 0.
// Prefers MSI Afterburner shared memory when available; falls back to Windows PDH.
func GPUPercent() Usage {
	return GPUPercentIndex(0)
}

// GPUPercentIndex returns GPU utilization for a zero-based GPU index.
func GPUPercentIndex(gpuIndex int) Usage {
	return gpuPercent(gpuIndex)
}

// FormatBytesAuto picks GB or TB (binary, 1024-based) for compact UI strings.
func FormatBytesAuto(n uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case n >= tb:
		v := float64(n) / float64(tb)
		if math.Abs(v-math.Round(v)) < 0.05 {
			return fmt.Sprintf("%.0fTB", math.Round(v))
		}
		return fmt.Sprintf("%.1fTB", v)
	case n >= gb:
		v := float64(n) / float64(gb)
		if math.Abs(v-math.Round(v)) < 0.05 {
			return fmt.Sprintf("%.0fGB", math.Round(v))
		}
		return fmt.Sprintf("%.1fGB", v)
	case n >= mb:
		return fmt.Sprintf("%.0fMB", float64(n)/float64(mb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// FormatBytesShort is like FormatBytesAuto but uses single-letter units (G/T)
// for denser metrics cells.
func FormatBytesShort(n uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case n >= tb:
		v := float64(n) / float64(tb)
		if math.Abs(v-math.Round(v)) < 0.05 {
			return fmt.Sprintf("%.0fT", math.Round(v))
		}
		return fmt.Sprintf("%.1fT", v)
	case n >= gb:
		v := float64(n) / float64(gb)
		if math.Abs(v-math.Round(v)) < 0.05 {
			return fmt.Sprintf("%.0fG", math.Round(v))
		}
		return fmt.Sprintf("%.1fG", v)
	case n >= mb:
		return fmt.Sprintf("%.0fM", float64(n)/float64(mb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// FormatUsageLine renders a compact percent for icon grids (e.g. "50%").
func FormatUsageLine(label string, percent float64, ok bool) string {
	_ = label
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", percent)
}

// driveLetter returns a single letter from labels like "C:" / "C".
func driveLetter(label string) string {
	lab := strings.TrimSpace(label)
	lab = strings.TrimSuffix(lab, ":")
	lab = strings.TrimSuffix(lab, `\`)
	lab = strings.TrimSuffix(lab, `/`)
	if lab == "" {
		return ""
	}
	// Prefer first rune if it looks like a drive letter.
	r := []rune(lab)
	if len(r) >= 1 {
		c := r[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return strings.ToUpper(string(c))
		}
	}
	return ""
}

// FormatStorageLines returns used/total and % used for two-line metrics cells.
// Offline disks: primary "—" (or "D —" when a drive letter is known), secondary "".
func FormatStorageLines(label string, used, total uint64, ok bool) (primary, secondary string) {
	letter := driveLetter(label)
	if !ok {
		if letter == "" {
			return "—", ""
		}
		return letter + " —", ""
	}
	body := FormatBytesShort(used) + "/" + FormatBytesShort(total)
	if letter != "" {
		primary = letter + " " + body
	} else {
		primary = body
	}
	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}
	secondary = fmt.Sprintf("%.0f%%", pct)
	return primary, secondary
}

// FormatStorageLine keeps a single-line used/total (legacy / tests).
func FormatStorageLine(label string, used, total uint64, ok bool) string {
	primary, secondary := FormatStorageLines(label, used, total, ok)
	if secondary == "" {
		return primary
	}
	return primary + " " + secondary
}

// rateFieldWidth is the fixed rune width for each rate token (figure-space padded)
// so ↑/↓ lines stack with aligned magnitudes in the two-line network cell.
const rateFieldWidth = 5

// FormatRate formats a bytes/sec throughput value into a fixed-width token.
func FormatRate(bps float64) string {
	if bps < 0 {
		bps = 0
	}
	const (
		kb = 1024.0
		mb = kb * 1024
		gb = mb * 1024
	)
	var s string
	switch {
	case bps >= gb:
		s = fmt.Sprintf("%.1fG", bps/gb)
	case bps >= mb:
		s = fmt.Sprintf("%.1fM", bps/mb)
	case bps >= kb:
		s = fmt.Sprintf("%.0fK", math.Round(bps/kb))
	default:
		s = fmt.Sprintf("%.0fB", math.Round(bps))
	}
	return padFigureWidth(s, rateFieldWidth)
}

func padFigureWidth(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	// U+2007 figure space ≈ digit width in most UI fonts.
	return strings.Repeat("\u2007", width-n) + s
}

// FormatNetworkLines returns upload and download captions for two-line cells.
// Magnitudes are fixed-width so arrows share a column when stacked.
func FormatNetworkLines(upBps, downBps float64, ok bool) (upLine, downLine string) {
	if !ok {
		return "—", ""
	}
	return "↑" + FormatRate(upBps), "↓" + FormatRate(downBps)
}

// FormatNetworkLine keeps a single-line up/down summary (legacy / tests).
func FormatNetworkLine(label string, upBps, downBps float64, ok bool) string {
	up, down := FormatNetworkLines(upBps, downBps, ok)
	body := up
	if down != "" {
		body = up + " " + down
	}
	if label == "" {
		return body
	}
	return strings.TrimSpace(label) + " " + body
}
