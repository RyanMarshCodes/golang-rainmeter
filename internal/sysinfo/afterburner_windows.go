//go:build windows

package sysinfo

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	mahmMapName       = "MAHMSharedMemory"
	mahmSignatureMAHM = 0x4D41484D // 'MAHM' as MSVC multi-char literal
	mahmSignatureDead = 0xDEAD
	mahmSrcGPUTemp    = 0x00000000
	mahmSrcGPUUsage   = 0x00000030
	mahmSrcCPUTemp    = 0x00000080
	mahmGlobalGPU     = 0xFFFFFFFF
)

var (
	modKernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procOpenFileMappingW = modKernel32.NewProc("OpenFileMappingW")
)

func openFileMapping(access uint32, inherit bool, name *uint16) (windows.Handle, error) {
	var inheritFlag uintptr
	if inherit {
		inheritFlag = 1
	}
	r0, _, e1 := procOpenFileMappingW.Call(uintptr(access), inheritFlag, uintptr(unsafe.Pointer(name)))
	if r0 == 0 {
		if e1 != windows.Errno(0) {
			return 0, e1
		}
		return 0, windows.ERROR_FILE_NOT_FOUND
	}
	return windows.Handle(r0), nil
}

// afterburnerGPUUsage reads GPU usage % from MSI Afterburner's MAHM shared memory.
// gpuIndex is zero-based (0 = first GPU).
func afterburnerGPUUsage(gpuIndex int) (Usage, error) {
	v, err := afterburnerValue(gpuIndex, mahmSrcGPUUsage)
	if err != nil {
		return Usage{}, err
	}
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return Usage{Percent: float64(v), OK: true}, nil
}

func afterburnerGPUTemp(gpuIndex int) (Temp, error) {
	v, err := afterburnerValue(gpuIndex, mahmSrcGPUTemp)
	if err != nil {
		return Temp{}, err
	}
	if v <= -100 || v >= 200 || math.IsNaN(float64(v)) {
		return Temp{}, fmt.Errorf("afterburner GPU temperature out of range")
	}
	return Temp{Celsius: float64(v), OK: true}, nil
}

func afterburnerCPUTemp() (Temp, error) {
	v, err := afterburnerValue(-1, mahmSrcCPUTemp)
	if err != nil {
		return Temp{}, err
	}
	if v <= -100 || v >= 200 || math.IsNaN(float64(v)) {
		return Temp{}, fmt.Errorf("afterburner CPU temperature out of range")
	}
	return Temp{Celsius: float64(v), OK: true}, nil
}

func afterburnerValue(gpuIndex int, srcID uint32) (float32, error) {
	name, err := windows.UTF16PtrFromString(mahmMapName)
	if err != nil {
		return 0, err
	}
	h, err := openFileMapping(windows.FILE_MAP_READ, false, name)
	if err != nil {
		return 0, fmt.Errorf("afterburner shared memory not found (is MSI Afterburner running?): %w", err)
	}
	defer windows.CloseHandle(h)

	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("MapViewOfFile: %w", err)
	}
	defer windows.UnmapViewOfFile(addr)

	hdr := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 64)

	sig := binary.LittleEndian.Uint32(hdr[0:4])
	if sig == mahmSignatureDead {
		return 0, fmt.Errorf("afterburner shared memory is shutting down")
	}
	if sig != mahmSignatureMAHM {
		return 0, fmt.Errorf("afterburner shared memory not ready (signature %#x)", sig)
	}

	headerSize := binary.LittleEndian.Uint32(hdr[8:12])
	numEntries := binary.LittleEndian.Uint32(hdr[12:16])
	entrySize := binary.LittleEndian.Uint32(hdr[16:20])
	if headerSize < 32 || entrySize < 1324 || numEntries == 0 || numEntries > 4096 {
		return 0, fmt.Errorf("afterburner header looks invalid (header=%d entry=%d n=%d)", headerSize, entrySize, numEntries)
	}

	base := unsafe.Pointer(addr)
	const (
		offData  = 5 * 260 // after 5 char[MAX_PATH] fields
		offGpu   = offData + 4 + 4 + 4 + 4
		offSrcID = offGpu + 4
	)

	var (
		found bool
		value float32
	)
	for i := uint32(0); i < numEntries; i++ {
		entry := unsafe.Add(base, uintptr(headerSize)+uintptr(i)*uintptr(entrySize))
		eb := unsafe.Slice((*byte)(entry), entrySize)

		entrySrc := binary.LittleEndian.Uint32(eb[offSrcID : offSrcID+4])
		if entrySrc != srcID {
			continue
		}
		gpu := binary.LittleEndian.Uint32(eb[offGpu : offGpu+4])
		if gpuIndex >= 0 {
			if gpu != mahmGlobalGPU && int(gpu) != gpuIndex {
				continue
			}
		}

		dataBits := binary.LittleEndian.Uint32(eb[offData : offData+4])
		data := math.Float32frombits(dataBits)
		if math.IsInf(float64(data), 0) || math.IsNaN(float64(data)) || data >= math.MaxFloat32/2 {
			continue
		}
		found = true
		value = data
		if gpuIndex >= 0 && gpu == uint32(gpuIndex) {
			break
		}
	}
	if !found {
		return 0, fmt.Errorf("afterburner src %#x for GPU %d not published", srcID, gpuIndex)
	}
	return value, nil
}
