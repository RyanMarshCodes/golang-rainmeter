//go:build !windows

package sysinfo

import "errors"

var errNoAfterburner = errors.New("afterburner unavailable")

func afterburnerCPUTemp() (Temp, error) { return Temp{}, errNoAfterburner }
func afterburnerGPUTemp(int) (Temp, error) { return Temp{}, errNoAfterburner }
