//go:build windows

package audio

import (
	"context"
	"encoding/binary"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

const (
	sampleRate = 44100
	fftSize    = 2048
)

// Analyzer captures system audio via WASAPI loopback and exposes smoothed spectrum bands.
type Analyzer struct {
	mu     sync.Mutex
	bands  []float32
	ring   []float32
	ringN  int
	running bool
	cancel context.CancelFunc
}

// NewAnalyzer creates an analyzer for the given band count.
func NewAnalyzer(bandCount int) *Analyzer {
	if bandCount < 4 {
		bandCount = 4
	}
	return &Analyzer{
		bands: make([]float32, bandCount),
		ring:  make([]float32, fftSize),
	}
}

// Bands returns a copy of the current smoothed levels (0–1).
func (a *Analyzer) Bands() []float32 {
	return a.CopyBands(nil)
}

// CopyBands copies levels into dst (reallocating if needed) and returns the slice.
func (a *Analyzer) CopyBands(dst []float32) []float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cap(dst) < len(a.bands) {
		dst = make([]float32, len(a.bands))
	} else {
		dst = dst[:len(a.bands)]
	}
	copy(dst, a.bands)
	return dst
}

// SetBandCount resizes the output band slice.
func (a *Analyzer) SetBandCount(n int) {
	if n < 4 {
		n = 4
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.bands) == n {
		return
	}
	a.bands = make([]float32, n)
}

// Start begins WASAPI loopback capture. Safe to call multiple times.
func (a *Analyzer) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.running = true
	a.mu.Unlock()

	go a.run(ctx)
	return nil
}

// Stop ends capture.
func (a *Analyzer) Stop() {
	a.mu.Lock()
	cancel := a.cancel
	a.running = false
	a.cancel = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *Analyzer) run(ctx context.Context) {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("audio: %s", message)
	})
	if err != nil {
		log.Printf("audio: init context: %v", err)
		return
	}
	defer func() {
		_ = malgoCtx.Uninit()
		malgoCtx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Loopback)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = sampleRate
	deviceConfig.Alsa.NoMMap = 1

	// Keep the WASAPI callback tiny — FFT here stalls capture after a frame or two.
	onRecv := func(_, pSample []byte, framecount uint32) {
		if framecount == 0 || len(pSample) < 2 {
			return
		}
		a.mu.Lock()
		for i := 0; i+1 < len(pSample) && uint32(i/2) < framecount; i += 2 {
			v := int16(binary.LittleEndian.Uint16(pSample[i : i+2]))
			a.ring[a.ringN%fftSize] = float32(v) / 32768.0
			a.ringN++
		}
		a.mu.Unlock()
	}

	device, err := malgo.InitDevice(malgoCtx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecv,
	})
	if err != nil {
		log.Printf("audio: loopback init failed (is a playback device available?): %v", err)
		return
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Printf("audio: loopback start: %v", err)
		return
	}

	ticker := time.NewTicker(16 * time.Millisecond) // ~60 Hz analysis
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshBands()
		}
	}
}

func (a *Analyzer) refreshBands() {
	a.mu.Lock()
	n := len(a.bands)
	samples := make([]float32, fftSize)
	if a.ringN < fftSize {
		// not enough data yet
		a.mu.Unlock()
		return
	}
	start := a.ringN % fftSize
	copy(samples, a.ring[start:])
	copy(samples[fftSize-start:], a.ring[:start])
	prev := append([]float32(nil), a.bands...)
	a.mu.Unlock()

	raw := SpectrumBands(samples, n)
	// Snappy response: rise quickly, fall fast enough to feel alive.
	const attack, release = float32(0.85), float32(0.55)
	for i := range raw {
		if raw[i] > prev[i] {
			prev[i] += (raw[i] - prev[i]) * attack
		} else {
			prev[i] += (raw[i] - prev[i]) * release
		}
		if prev[i] < 0.015 {
			prev[i] = 0
		}
		prev[i] = float32(math.Min(1, float64(prev[i])))
	}

	a.mu.Lock()
	copy(a.bands, prev)
	a.mu.Unlock()
}
