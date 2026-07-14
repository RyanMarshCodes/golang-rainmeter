package sysinfo

import "github.com/shirou/gopsutil/v4/mem"

func memoryPercent() Usage {
	v, err := mem.VirtualMemory()
	if err != nil {
		return Usage{}
	}
	return Usage{Percent: v.UsedPercent, OK: true}
}
