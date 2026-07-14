package audio

import (
	"math"
	"testing"
)

func TestConvertToFloat32Int16(t *testing.T) {
	format := Format{SampleFormat: SampleFormatInt16, BitsPerSample: 16, Channels: 1, SampleRate: 48000, BlockAlign: 2}
	data := []byte{0x00, 0x80, 0x00, 0x00, 0xff, 0x7f}
	out, err := ConvertToFloat32(format, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{-1, 0, float32(32767.0 / 32768.0)}
	assertFloat32s(t, out, want)
}

func TestConvertToFloat32Int24(t *testing.T) {
	format := Format{SampleFormat: SampleFormatInt24, BitsPerSample: 24, Channels: 1, SampleRate: 48000, BlockAlign: 3}
	data := []byte{0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x7f}
	out, err := ConvertToFloat32(format, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{-1, 0, float32(8388607.0 / 8388608.0)}
	assertFloat32s(t, out, want)
}

func TestConvertToFloat32RejectsUnalignedData(t *testing.T) {
	format := Format{SampleFormat: SampleFormatInt16, BitsPerSample: 16, Channels: 1, SampleRate: 48000, BlockAlign: 2}
	if _, err := ConvertToFloat32(format, []byte{1}, nil); err == nil {
		t.Fatal("expected unaligned data error")
	}
}

func assertFloat32s(t *testing.T, got []float32, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d", len(got), len(want))
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > 0.000001 {
			t.Fatalf("got[%d]=%f want %f", i, got[i], want[i])
		}
	}
}
