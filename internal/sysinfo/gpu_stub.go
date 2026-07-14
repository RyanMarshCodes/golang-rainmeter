//go:build !windows

package sysinfo

func gpuPercent(gpuIndex int) Usage {
	_ = gpuIndex
	return Usage{}
}
