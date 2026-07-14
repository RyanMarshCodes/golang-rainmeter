//go:build windows && cgo

package main

import (
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio/loopback"
)

func main() {
	duration := flag.Duration("duration", 0, "optional run duration; zero runs until interrupted")
	interval := flag.Duration("interval", 100*time.Millisecond, "display refresh interval")
	width := flag.Int("width", 32, "meter bar width")
	flag.Parse()

	if *duration < 0 {
		panic("duration must be non-negative")
	}
	if *interval <= 0 {
		panic("interval must be positive")
	}
	if *width <= 0 {
		panic("width must be positive")
	}

	capturer, err := loopback.New(&loopback.Config{EventBuffer: 128})
	if err != nil {
		panic(err)
	}
	defer capturer.Close()

	format := capturer.Format()
	if !meterFormatSupported(format) {
		panic(fmt.Sprintf("unsupported meter sample format: %v", format.SampleFormat))
	}

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

	stats := &meterStats{}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	var done <-chan time.Time
	if *duration > 0 {
		done = time.After(*duration)
	}

	for {
		select {
		case frame, ok := <-capturer.Frames():
			if !ok {
				fmt.Println()
				return
			}
			stats.AddFrame(frame)
		case err, ok := <-capturer.Errors():
			if ok && err != nil {
				panic(err)
			}
		case <-ticker.C:
			printMeter(format, stats, *width)
			stats.Reset()
		case <-done:
			fmt.Println()
			return
		}
	}
}

type meterStats struct {
	samples       int
	sumSquares    float64
	peak          float64
	frames        int
	silentPackets int
	convertBuf    []float32
}

func (s *meterStats) AddFrame(frame loopback.Frame) {
	s.frames += frame.Frames
	if frame.Silent || len(frame.Data) == 0 {
		s.silentPackets++
		return
	}
	peak, sumSquares, samples, err := analyzeFrame(frame, s.convertBuf)
	if err != nil {
		return
	}
	s.convertBuf = s.convertBuf[:0]
	if peak > s.peak {
		s.peak = peak
	}
	s.sumSquares += sumSquares
	s.samples += samples
}

func (s *meterStats) Reset() {
	buf := s.convertBuf
	*s = meterStats{}
	s.convertBuf = buf[:0]
}

func (s *meterStats) RMS() float64 {
	if s.samples == 0 {
		return 0
	}
	return math.Sqrt(s.sumSquares / float64(s.samples))
}

func analyzeFrame(frame loopback.Frame, buf []float32) (peak float64, sumSquares float64, samples int, err error) {
	converted, err := audio.ConvertToFloat32(frame.Format, frame.Data, buf)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, sample := range converted {
		v := float64(sample)
		abs := math.Abs(v)
		if abs > peak {
			peak = abs
		}
		sumSquares += v * v
		samples++
	}
	return clamp01(peak), sumSquares, samples, nil
}

func printMeter(format audio.Format, stats *meterStats, width int) {
	peak := stats.peak
	rms := stats.RMS()
	filled := int(math.Round(peak * float64(width)))
	if filled > width {
		filled = width
	}
	bar := make([]byte, width)
	for i := range bar {
		if i < filled {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}
	state := "active"
	if stats.samples == 0 && stats.silentPackets > 0 {
		state = "silent"
	} else if peak == 0 {
		state = "quiet"
	}
	fmt.Printf("\r%d Hz %dch %-7s | peak %.3f rms %.3f | %s | %s",
		format.SampleRate,
		format.Channels,
		format.SampleFormat,
		peak,
		rms,
		string(bar),
		state,
	)
}

func meterFormatSupported(format audio.Format) bool {
	return audio.CanConvertToFloat32(format)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
