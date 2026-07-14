package audio

import (
	"encoding/binary"
	"fmt"
	"math"
)

// CanConvertToFloat32 reports whether ConvertToFloat32 supports the format.
func CanConvertToFloat32(format Format) bool {
	switch format.SampleFormat {
	case SampleFormatFloat32:
		return format.BitsPerSample == 32
	case SampleFormatInt16:
		return format.BitsPerSample == 16
	case SampleFormatInt24:
		return format.BitsPerSample == 24
	case SampleFormatInt32:
		return format.BitsPerSample == 32
	default:
		return false
	}
}

// ConvertToFloat32 converts interleaved PCM bytes into normalized float32
// samples in the range [-1, 1]. The returned slice may reuse dst.
func ConvertToFloat32(format Format, data []byte, dst []float32) ([]float32, error) {
	if !CanConvertToFloat32(format) {
		return nil, fmt.Errorf("audio: unsupported sample format: %v/%d-bit", format.SampleFormat, format.BitsPerSample)
	}
	bytesPerSample := format.BitsPerSample / 8
	if bytesPerSample <= 0 {
		return nil, fmt.Errorf("audio: invalid bits per sample: %d", format.BitsPerSample)
	}
	if len(data)%bytesPerSample != 0 {
		return nil, fmt.Errorf("audio: data length %d is not aligned to %d-byte samples", len(data), bytesPerSample)
	}

	samples := len(data) / bytesPerSample
	if cap(dst) < samples {
		dst = make([]float32, samples)
	} else {
		dst = dst[:samples]
	}

	switch format.SampleFormat {
	case SampleFormatFloat32:
		for i := 0; i < samples; i++ {
			v := math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				v = 0
			}
			dst[i] = clampFloat32(v)
		}
	case SampleFormatInt16:
		const scale = 1.0 / 32768.0
		for i := 0; i < samples; i++ {
			dst[i] = float32(float64(int16(binary.LittleEndian.Uint16(data[i*2:]))) * scale)
		}
	case SampleFormatInt24:
		const scale = 1.0 / 8388608.0
		for i := 0; i < samples; i++ {
			off := i * 3
			raw := int32(data[off]) | int32(data[off+1])<<8 | int32(data[off+2])<<16
			if raw&0x800000 != 0 {
				raw |= ^0xFFFFFF
			}
			dst[i] = float32(float64(raw) * scale)
		}
	case SampleFormatInt32:
		const scale = 1.0 / 2147483648.0
		for i := 0; i < samples; i++ {
			dst[i] = float32(float64(int32(binary.LittleEndian.Uint32(data[i*4:]))) * scale)
		}
	}
	return dst, nil
}

func clampFloat32(v float32) float32 {
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}
