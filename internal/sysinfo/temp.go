package sysinfo

import "strconv"

// Temp is a hardware temperature reading in Celsius.
type Temp struct {
	Celsius float64
	OK      bool
}

// CPUTemp returns CPU temperature when available (MSI Afterburner MAHM).
func CPUTemp() Temp {
	t, err := afterburnerCPUTemp()
	if err != nil {
		return Temp{}
	}
	return t
}

// GPUTempIndex returns GPU temperature for a zero-based GPU index.
func GPUTempIndex(gpuIndex int) Temp {
	if gpuIndex < 0 {
		gpuIndex = 0
	}
	t, err := afterburnerGPUTemp(gpuIndex)
	if err != nil {
		return Temp{}
	}
	return t
}

// FormatTempLine renders a compact °C line for metrics secondary rows.
func FormatTempLine(t Temp) string {
	if !t.OK {
		return ""
	}
	n := int(t.Celsius + 0.5)
	if t.Celsius < 0 {
		n = int(t.Celsius - 0.5)
	}
	return strconv.Itoa(n) + "°"
}
