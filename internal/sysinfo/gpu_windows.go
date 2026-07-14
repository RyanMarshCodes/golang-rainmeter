//go:build windows

package sysinfo

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pdh               = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQuery  = pdh.NewProc("PdhOpenQueryW")
	procPdhAddCounter = pdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollect    = pdh.NewProc("PdhCollectQueryData")
	procPdhGetFmt     = pdh.NewProc("PdhGetFormattedCounterArrayW")
	procPdhCloseQuery = pdh.NewProc("PdhCloseQuery")
	gpuLogOnce        sync.Once
	abLogOnce         sync.Once
)

type pdhFmtCounterValueItem struct {
	szName   *uint16
	fmtValue struct {
		CStatus uint32
		_       uint32
		Value   float64
	}
}

func gpuPercent(gpuIndex int) Usage {
	if gpuIndex < 0 {
		gpuIndex = 0
	}
	if u, err := afterburnerGPUUsage(gpuIndex); err == nil && u.OK {
		return u
	} else if err != nil {
		abLogOnce.Do(func() {
			log.Printf("sysinfo: afterburner GPU: %v (falling back to Windows PDH)", err)
		})
	}

	u, err := readGPUEngineUtil()
	if err != nil {
		gpuLogOnce.Do(func() {
			log.Printf("sysinfo: gpu metrics unavailable: %v", err)
		})
		return Usage{}
	}
	return u
}

func readGPUEngineUtil() (Usage, error) {
	var hQuery windows.Handle
	r, _, err := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&hQuery)))
	if r != 0 {
		return Usage{}, fmt.Errorf("PdhOpenQuery: %v", err)
	}
	defer procPdhCloseQuery.Call(uintptr(hQuery))

	path, err := windows.UTF16PtrFromString(`\GPU Engine(*)\Utilization Percentage`)
	if err != nil {
		return Usage{}, err
	}
	var hCounter windows.Handle
	r, _, err = procPdhAddCounter.Call(uintptr(hQuery), uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&hCounter)))
	if r != 0 {
		return Usage{}, fmt.Errorf("PdhAddEnglishCounter: %v", err)
	}

	r, _, err = procPdhCollect.Call(uintptr(hQuery))
	if r != 0 {
		return Usage{}, fmt.Errorf("PdhCollectQueryData(1): %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	r, _, err = procPdhCollect.Call(uintptr(hQuery))
	if r != 0 {
		return Usage{}, fmt.Errorf("PdhCollectQueryData(2): %v", err)
	}

	var bufSize, itemCount uint32
	const pdhFmtDouble = 0x00000200
	_, _, _ = procPdhGetFmt.Call(
		uintptr(hCounter),
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)
	if bufSize == 0 {
		return Usage{}, fmt.Errorf("PdhGetFormattedCounterArray: empty buffer")
	}
	buf := make([]byte, bufSize)
	r, _, err = procPdhGetFmt.Call(
		uintptr(hCounter),
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if r != 0 {
		return Usage{}, fmt.Errorf("PdhGetFormattedCounterArray: %#x %v", r, err)
	}

	const itemSize = int(unsafe.Sizeof(pdhFmtCounterValueItem{}))
	max := 0.0
	n := 0
	for i := 0; i < int(itemCount); i++ {
		item := (*pdhFmtCounterValueItem)(unsafe.Pointer(&buf[i*itemSize]))
		if item.fmtValue.CStatus != 0 {
			continue
		}
		v := item.fmtValue.Value
		if v < 0 || v > 100 {
			continue
		}
		if item.szName != nil {
			_ = strings.ToLower(windows.UTF16PtrToString(item.szName))
		}
		if v > max {
			max = v
		}
		n++
	}
	if n == 0 {
		return Usage{Percent: 0, OK: true}, nil
	}
	return Usage{Percent: max, OK: true}, nil
}
