package audio

import (
	"math"
	"math/cmplx"
)

func nextPow2(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func hann(n int) []float64 {
	w := make([]float64, n)
	if n <= 1 {
		return w
	}
	for i := 0; i < n; i++ {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

// fftInPlace is a classic radix-2 Cooley–Tukey FFT (in-place).
func fftInPlace(a []complex128) {
	n := len(a)
	for i, j := 0, 0; i < n; i++ {
		if j > i {
			a[i], a[j] = a[j], a[i]
		}
		m := n >> 1
		for m >= 1 && j >= m {
			j -= m
			m >>= 1
		}
		j += m
	}
	for len := 2; len <= n; len <<= 1 {
		ang := -2 * math.Pi / float64(len)
		wlen := cmplx.Rect(1, ang)
		for i := 0; i < n; i += len {
			w := complex(1, 0)
			half := len >> 1
			for j := 0; j < half; j++ {
				u := a[i+j]
				v := a[i+j+half] * w
				a[i+j] = u + v
				a[i+j+half] = u - v
				w *= wlen
			}
		}
	}
}

// SpectrumBands converts time-domain mono samples into numBands normalized magnitudes (0–1),
// using log-spaced frequency buckets typical of bar visualizers.
func SpectrumBands(samples []float32, numBands int) []float32 {
	if numBands < 1 || len(samples) < 32 {
		return make([]float32, numBands)
	}
	n := nextPow2(len(samples))
	if n > 4096 {
		n = 4096
	}
	buf := make([]complex128, n)
	w := hann(n)
	lim := n
	if lim > len(samples) {
		lim = len(samples)
	}
	for i := 0; i < lim; i++ {
		buf[i] = complex(float64(samples[i])*w[i], 0)
	}
	fftInPlace(buf)

	half := n / 2
	mags := make([]float64, half)
	var peak float64
	for i := 0; i < half; i++ {
		m := cmplx.Abs(buf[i])
		mags[i] = m
		if m > peak {
			peak = m
		}
	}
	if peak < 1e-9 {
		return make([]float32, numBands)
	}

	out := make([]float32, numBands)
	// Skip DC; map remaining bins logarithmically.
	lo, hi := 1, half-1
	for b := 0; b < numBands; b++ {
		t0 := float64(b) / float64(numBands)
		t1 := float64(b+1) / float64(numBands)
		i0 := logBin(t0, lo, hi)
		i1 := logBin(t1, lo, hi)
		if i1 <= i0 {
			i1 = i0 + 1
		}
		if i1 > half {
			i1 = half
		}
		var sum float64
		for i := i0; i < i1; i++ {
			sum += mags[i]
		}
		avg := sum / float64(i1-i0) / peak
		// mild gamma so quiet music still moves bars
		out[b] = float32(math.Pow(avg, 0.55))
		if out[b] > 1 {
			out[b] = 1
		}
	}
	return out
}

func logBin(t float64, lo, hi int) int {
	if t <= 0 {
		return lo
	}
	if t >= 1 {
		return hi
	}
	span := float64(hi - lo)
	if span < 1 {
		return lo
	}
	return lo + int(math.Round(math.Pow(span, t)))
}
