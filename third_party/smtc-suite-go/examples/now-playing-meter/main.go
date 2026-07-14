//go:build windows && cgo

package main

import (
	"flag"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/audio/loopback"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

func main() {
	duration := flag.Duration("duration", 30*time.Second, "run duration")
	interval := flag.Duration("interval", 500*time.Millisecond, "display refresh interval")
	width := flag.Int("width", 24, "meter bar width")
	flag.Parse()

	if *duration <= 0 {
		panic("duration must be positive")
	}
	if *interval <= 0 {
		panic("interval must be positive")
	}

	mgr, err := monitor.New(nil)
	if err != nil {
		panic(err)
	}
	defer mgr.Close()

	capturer, err := loopback.New(&loopback.Config{EventBuffer: 128})
	if err != nil {
		panic(err)
	}
	defer capturer.Close()

	format := capturer.Format()
	if !audio.CanConvertToFloat32(format) {
		panic(fmt.Sprintf("unsupported sample format: %v", format.SampleFormat))
	}

	fmt.Printf("audio: %d Hz, %d ch, %s\n", format.SampleRate, format.Channels, format.SampleFormat)
	if err := capturer.Start(); err != nil {
		panic(err)
	}
	defer capturer.Stop()

	stats := &meterStats{}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	done := time.After(*duration)

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
			printNowPlayingMeter(mgr.CurrentSession(), stats, *width)
			stats.Reset()
		case <-done:
			fmt.Println()
			return
		}
	}
}

type meterStats struct {
	samples    int
	sumSquares float64
	peak       float64
	buf        []float32
}

func (s *meterStats) AddFrame(frame loopback.Frame) {
	if frame.Silent || len(frame.Data) == 0 {
		return
	}
	samples, err := audio.ConvertToFloat32(frame.Format, frame.Data, s.buf)
	if err != nil {
		return
	}
	s.buf = samples[:0]
	for _, sample := range samples {
		v := float64(sample)
		abs := math.Abs(v)
		if abs > s.peak {
			s.peak = abs
		}
		s.sumSquares += v * v
		s.samples++
	}
}

func (s *meterStats) Reset() {
	buf := s.buf
	*s = meterStats{}
	s.buf = buf[:0]
}

func (s *meterStats) RMS() float64 {
	if s.samples == 0 {
		return 0
	}
	return math.Sqrt(s.sumSquares / float64(s.samples))
}

func printNowPlayingMeter(session *smtc.SessionInfo, stats *meterStats, width int) {
	filled := int(math.Round(stats.peak * float64(width)))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	fmt.Printf("\r%-50s | peak %.3f rms %.3f | %s",
		nowPlaying(session),
		stats.peak,
		stats.RMS(),
		bar,
	)
}

func nowPlaying(session *smtc.SessionInfo) string {
	if session == nil {
		return "No SMTC session"
	}
	title := session.MediaInfo.Title
	artist := session.MediaInfo.Artist
	if title == "" && artist == "" {
		return session.SourceAppUserModelID
	}
	if title == "" {
		return artist
	}
	if artist == "" {
		return title
	}
	return artist + " - " + title
}
