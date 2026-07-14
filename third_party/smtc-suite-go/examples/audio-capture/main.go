//go:build windows && cgo

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio/loopback"
)

func main() {
	duration := flag.Duration("duration", 10*time.Second, "capture duration")
	out := flag.String("out", filepath.Join("testdata", "loopback.wav"), "output WAV path")
	flag.Parse()

	if *duration <= 0 {
		panic("duration must be positive")
	}

	capturer, err := loopback.New(&loopback.Config{EventBuffer: 128})
	if err != nil {
		panic(err)
	}
	defer capturer.Close()

	format := capturer.Format()
	if err := validateWAVFormat(format); err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		panic(err)
	}

	file, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer, err := newWAVWriter(file, format)
	if err != nil {
		panic(err)
	}

	fmt.Printf("capturing %s to %s\n", duration.String(), *out)
	fmt.Printf("format: %d Hz, %d ch, %d bits, blockAlign=%d, sampleFormat=%v\n",
		format.SampleRate,
		format.Channels,
		format.BitsPerSample,
		format.BlockAlign,
		format.SampleFormat,
	)

	if err := capturer.Start(); err != nil {
		panic(err)
	}
	defer capturer.Stop()

	deadline := time.After(*duration)
	var packets int
	var silentPackets int

captureLoop:
	for {
		select {
		case frame, ok := <-capturer.Frames():
			if !ok {
				break captureLoop
			}
			packets++
			if frame.Silent {
				silentPackets++
			}
			if err := writer.WriteFrame(frame); err != nil {
				panic(err)
			}
		case err, ok := <-capturer.Errors():
			if ok && err != nil {
				panic(err)
			}
		case <-deadline:
			break captureLoop
		}
	}

	if err := writer.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("done: packets=%d silent=%d frames=%d bytes=%d\n",
		packets,
		silentPackets,
		writer.framesWritten,
		writer.dataBytes,
	)
}

func validateWAVFormat(format audio.Format) error {
	if format.SampleRate <= 0 || format.Channels <= 0 || format.BlockAlign <= 0 {
		return fmt.Errorf("invalid audio format: %+v", format)
	}
	switch format.SampleFormat {
	case audio.SampleFormatFloat32:
		if format.BitsPerSample != 32 {
			return fmt.Errorf("float WAV requires 32-bit samples, got %d", format.BitsPerSample)
		}
	case audio.SampleFormatInt16, audio.SampleFormatInt24, audio.SampleFormatInt32:
		// Supported as PCM WAV.
	default:
		return fmt.Errorf("unsupported WAV sample format: %v", format.SampleFormat)
	}
	return nil
}

type wavWriter struct {
	w                io.WriteSeeker
	format           audio.Format
	dataBytes        uint32
	framesWritten    uint32
	factSampleOffset int64
	dataSizeOffset   int64
	dataStartOffset  int64
	closed           bool
}

func newWAVWriter(w io.WriteSeeker, format audio.Format) (*wavWriter, error) {
	ww := &wavWriter{w: w, format: format, factSampleOffset: -1}
	if err := ww.writeHeader(); err != nil {
		return nil, err
	}
	return ww, nil
}

func (w *wavWriter) WriteFrame(frame loopback.Frame) error {
	if w.closed {
		return fmt.Errorf("wav writer is closed")
	}
	if frame.Frames <= 0 {
		return nil
	}

	expected := frame.Frames * w.format.BlockAlign
	if frame.Silent {
		return w.writeSilence(expected, frame.Frames)
	}
	if len(frame.Data) < expected {
		return fmt.Errorf("short audio frame: got %d bytes, want %d", len(frame.Data), expected)
	}
	if _, err := w.w.Write(frame.Data[:expected]); err != nil {
		return err
	}
	w.dataBytes += uint32(expected)
	w.framesWritten += uint32(frame.Frames)
	return nil
}

func (w *wavWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	end, err := w.w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := w.w.Seek(4, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(end-8)); err != nil {
		return err
	}
	if w.factSampleOffset >= 0 {
		if _, err := w.w.Seek(w.factSampleOffset, io.SeekStart); err != nil {
			return err
		}
		if err := binary.Write(w.w, binary.LittleEndian, w.framesWritten); err != nil {
			return err
		}
	}
	if _, err := w.w.Seek(w.dataSizeOffset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, w.dataBytes); err != nil {
		return err
	}
	_, err = w.w.Seek(end, io.SeekStart)
	return err
}

func (w *wavWriter) writeHeader() error {
	if _, err := w.w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	if _, err := w.w.Write([]byte("WAVE")); err != nil {
		return err
	}

	if _, err := w.w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, w.wavFormatTag()); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint16(w.format.Channels)); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(w.format.SampleRate)); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint32(w.format.SampleRate*w.format.BlockAlign)); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint16(w.format.BlockAlign)); err != nil {
		return err
	}
	if err := binary.Write(w.w, binary.LittleEndian, uint16(w.format.BitsPerSample)); err != nil {
		return err
	}

	if w.format.SampleFormat == audio.SampleFormatFloat32 {
		if _, err := w.w.Write([]byte("fact")); err != nil {
			return err
		}
		if err := binary.Write(w.w, binary.LittleEndian, uint32(4)); err != nil {
			return err
		}
		pos, err := w.w.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		w.factSampleOffset = pos
		if err := binary.Write(w.w, binary.LittleEndian, uint32(0)); err != nil {
			return err
		}
	}

	if _, err := w.w.Write([]byte("data")); err != nil {
		return err
	}
	pos, err := w.w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	w.dataSizeOffset = pos
	if err := binary.Write(w.w, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	w.dataStartOffset = pos + 4
	return nil
}

func (w *wavWriter) wavFormatTag() uint16 {
	if w.format.SampleFormat == audio.SampleFormatFloat32 {
		return 3
	}
	return 1
}

func (w *wavWriter) writeSilence(bytes int, frames int) error {
	const zeroChunkSize = 32 * 1024
	var zero [zeroChunkSize]byte
	remaining := bytes
	for remaining > 0 {
		chunk := remaining
		if chunk > len(zero) {
			chunk = len(zero)
		}
		if _, err := w.w.Write(zero[:chunk]); err != nil {
			return err
		}
		remaining -= chunk
	}
	w.dataBytes += uint32(bytes)
	w.framesWritten += uint32(frames)
	return nil
}
