package sysinfo

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func cpuPercent(intervalSec float64) Usage {
	if intervalSec <= 0 {
		intervalSec = 0.2
	}
	vals, err := cpu.Percent(time.Duration(intervalSec*float64(time.Second)), false)
	if err != nil || len(vals) == 0 {
		return Usage{}
	}
	return Usage{Percent: vals[0], OK: true}
}
