//go:build !windows

package audio

import "context"

// Analyzer is a no-op spectrum analyzer on non-Windows platforms.
type Analyzer struct {
	bands []float32
}

func NewAnalyzer(bandCount int) *Analyzer {
	if bandCount < 4 {
		bandCount = 4
	}
	return &Analyzer{bands: make([]float32, bandCount)}
}

func (a *Analyzer) Bands() []float32 {
	return a.CopyBands(nil)
}

func (a *Analyzer) CopyBands(dst []float32) []float32 {
	if cap(dst) < len(a.bands) {
		dst = make([]float32, len(a.bands))
	} else {
		dst = dst[:len(a.bands)]
	}
	copy(dst, a.bands)
	return dst
}

func (a *Analyzer) SetBandCount(n int) {
	if n < 4 {
		n = 4
	}
	a.bands = make([]float32, n)
}

func (a *Analyzer) Start(context.Context) error { return nil }
func (a *Analyzer) Stop()                       {}
