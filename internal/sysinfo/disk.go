package sysinfo

import "github.com/shirou/gopsutil/v4/disk"

func diskUsage(path string) Storage {
	if path == "" {
		return Storage{}
	}
	u, err := disk.Usage(path)
	if err != nil {
		return Storage{}
	}
	return Storage{UsedBytes: u.Used, TotalBytes: u.Total, OK: true}
}
