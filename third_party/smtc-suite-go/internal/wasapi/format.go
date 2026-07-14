//go:build windows && cgo

package wasapi

// #include "c/helpers.h"
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
)

// Format describes a WASAPI stream format.
type Format struct {
	audio.Format
	AvgBytesPerSec     int
	WaveFormatTag      uint16
	ValidBitsPerSample int
	ChannelMask        uint32
	SubFormat          GUID
}

func parseWaveFormat(ptr unsafe.Pointer) (Format, error) {
	if ptr == nil {
		return Format{}, fmt.Errorf("nil WAVEFORMATEX")
	}

	tag := uint16(C.smtcWaveFormatTag(ptr))
	channels := int(C.smtcWaveFormatChannels(ptr))
	sampleRate := int(C.smtcWaveFormatSamplesPerSec(ptr))
	blockAlign := int(C.smtcWaveFormatBlockAlign(ptr))
	bitsPerSample := int(C.smtcWaveFormatBitsPerSample(ptr))
	cbSize := int(C.smtcWaveFormatCbSize(ptr))

	format := Format{
		Format: audio.Format{
			SampleRate:    sampleRate,
			Channels:      channels,
			BitsPerSample: bitsPerSample,
			SampleFormat:  sampleFormatFromTag(tag, bitsPerSample),
			BlockAlign:    blockAlign,
		},
		WaveFormatTag:  tag,
		AvgBytesPerSec: int(sampleRate * blockAlign),
	}

	if tag == waveFormatExtensible {
		if cbSize < 22 {
			return Format{}, fmt.Errorf("WAVEFORMATEXTENSIBLE has short cbSize: %d", cbSize)
		}
		format.ValidBitsPerSample = int(C.smtcWaveFormatValidBitsPerSample(ptr))
		if format.ValidBitsPerSample <= 0 || format.ValidBitsPerSample > bitsPerSample {
			format.ValidBitsPerSample = bitsPerSample
		}
		format.ChannelMask = uint32(C.smtcWaveFormatChannelMask(ptr))
		format.SubFormat = *(*GUID)(unsafe.Pointer(C.smtcWaveFormatSubFormat(ptr)))
		format.SampleFormat = sampleFormatFromSubFormat(&format.SubFormat, bitsPerSample, format.ValidBitsPerSample)
		if format.SampleFormat == audio.SampleFormatUnknown && bitsPerSample == 32 {
			format.SampleFormat = audio.SampleFormatFloat32
		}
	}

	return format, nil
}

func sampleFormatFromTag(tag uint16, bits int) audio.SampleFormat {
	switch tag {
	case waveFormatPCM:
		return intSampleFormat(bits)
	case waveFormatIEEEFloat:
		if bits == 32 {
			return audio.SampleFormatFloat32
		}
	}
	return audio.SampleFormatUnknown
}

func sampleFormatFromSubFormat(sub *GUID, bits int, validBits int) audio.SampleFormat {
	if equalGUID(sub, KSDATAFORMAT_SUBTYPE_PCM) {
		if validBits > 0 {
			return intSampleFormat(validBits)
		}
		return intSampleFormat(bits)
	}
	if equalGUID(sub, KSDATAFORMAT_SUBTYPE_IEEE_FLOAT) && bits == 32 {
		return audio.SampleFormatFloat32
	}
	return audio.SampleFormatUnknown
}

func intSampleFormat(bits int) audio.SampleFormat {
	switch bits {
	case 16:
		return audio.SampleFormatInt16
	case 24:
		return audio.SampleFormatInt24
	case 32:
		return audio.SampleFormatInt32
	default:
		return audio.SampleFormatUnknown
	}
}
