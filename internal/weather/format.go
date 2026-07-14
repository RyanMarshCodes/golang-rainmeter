package weather

import "strconv"

// Describe maps a WMO weather interpretation code to a short label and
// logical icon name (resolved via assets/fonts/icons/icon-map.json).
func Describe(code int) (label, icon string) {
	switch {
	case code == 0:
		return "Clear", "sun"
	case code == 1:
		return "Mostly Clear", "sun"
	case code == 2:
		return "Partly Cloudy", "cloud-sun"
	case code == 3:
		return "Overcast", "cloud"
	case code == 45, code == 48:
		return "Fog", "fog"
	case code >= 51 && code <= 57:
		return "Drizzle", "cloud-rain"
	case code >= 61 && code <= 67:
		return "Rain", "cloud-rain"
	case code >= 71 && code <= 77:
		return "Snow", "snowflake"
	case code >= 80 && code <= 82:
		return "Showers", "cloud-showers-heavy"
	case code >= 85 && code <= 86:
		return "Snow Showers", "snowflake"
	case code >= 95 && code <= 99:
		return "Thunder", "cloud-bolt"
	default:
		return "Cloudy", "cloud"
	}
}

// CompassDir converts degrees to an 8-point compass abbreviation.
func CompassDir(deg int) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	if deg < 0 {
		deg = 0
	}
	i := ((deg % 360) + 22) / 45
	if i >= len(dirs) {
		i = 0
	}
	return dirs[i]
}

// FormatTemp rounds to whole degrees with a degree sign.
func FormatTemp(v float64) string {
	n := int(v + 0.5)
	if v < 0 {
		n = int(v - 0.5)
	}
	return strconv.Itoa(n) + "°"
}

// FormatRange formats high / low.
func FormatRange(high, low float64) string {
	return FormatTemp(high) + " / " + FormatTemp(low)
}

// FormatWind formats compass + speed + unit.
func FormatWind(deg int, speed float64, metric bool) string {
	unit := "mph"
	if metric {
		unit = "km/h"
	}
	n := int(speed + 0.5)
	return CompassDir(deg) + " " + strconv.Itoa(n) + " " + unit
}
