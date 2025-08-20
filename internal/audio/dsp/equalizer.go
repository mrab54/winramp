package dsp

import (
	"math"
	"sync"
)

// EqualizerBand represents a single band in the equalizer
type EqualizerBand struct {
	Frequency float64 // Center frequency in Hz
	Gain      float64 // Gain in dB (-12 to +12)
	Q         float64 // Q factor (bandwidth)
}

// Equalizer implements a 10-band parametric equalizer
type Equalizer struct {
	bands      [10]EqualizerBand
	filters    [10]*BiquadFilter
	enabled    bool
	sampleRate int
	mu         sync.RWMutex
}

// NewEqualizer creates a new 10-band equalizer
func NewEqualizer(sampleRate int) *Equalizer {
	eq := &Equalizer{
		enabled:    false,
		sampleRate: sampleRate,
	}
	
	// Initialize standard 10-band frequencies
	frequencies := []float64{
		31.25,  // Sub-bass
		62.5,   // Bass
		125,    // Low-mid
		250,    // Mid
		500,    // Mid
		1000,   // Mid-high
		2000,   // High-mid
		4000,   // Presence
		8000,   // Brilliance
		16000,  // Air
	}
	
	// Initialize bands with flat response (0 dB gain)
	for i := 0; i < 10; i++ {
		eq.bands[i] = EqualizerBand{
			Frequency: frequencies[i],
			Gain:      0.0,
			Q:         0.7, // Standard Q factor
		}
		eq.filters[i] = NewBiquadFilter(sampleRate)
		eq.updateFilter(i)
	}
	
	return eq
}

// SetBandGain sets the gain for a specific band
func (eq *Equalizer) SetBandGain(band int, gain float64) error {
	if band < 0 || band >= 10 {
		return ErrInvalidParameter
	}
	
	// Clamp gain to -12 to +12 dB
	if gain < -12 {
		gain = -12
	} else if gain > 12 {
		gain = 12
	}
	
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	eq.bands[band].Gain = gain
	eq.updateFilter(band)
	
	return nil
}

// GetBandGain gets the gain for a specific band
func (eq *Equalizer) GetBandGain(band int) float64 {
	if band < 0 || band >= 10 {
		return 0
	}
	
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	return eq.bands[band].Gain
}

// SetAllBands sets gains for all bands
func (eq *Equalizer) SetAllBands(gains [10]float64) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	for i := 0; i < 10; i++ {
		gain := gains[i]
		if gain < -12 {
			gain = -12
		} else if gain > 12 {
			gain = 12
		}
		eq.bands[i].Gain = gain
		eq.updateFilter(i)
	}
}

// GetAllBands returns gains for all bands
func (eq *Equalizer) GetAllBands() [10]float64 {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	var gains [10]float64
	for i := 0; i < 10; i++ {
		gains[i] = eq.bands[i].Gain
	}
	return gains
}

// SetEnabled enables or disables the equalizer
func (eq *Equalizer) SetEnabled(enabled bool) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	eq.enabled = enabled
}

// IsEnabled returns whether the equalizer is enabled
func (eq *Equalizer) IsEnabled() bool {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	return eq.enabled
}

// Process applies equalization to audio samples
func (eq *Equalizer) Process(samples []float32) {
	eq.mu.RLock()
	enabled := eq.enabled
	eq.mu.RUnlock()
	
	if !enabled {
		return
	}
	
	// Apply each band filter in series
	for i := 0; i < 10; i++ {
		eq.filters[i].Process(samples)
	}
}

// ProcessStereo applies equalization to stereo audio samples
func (eq *Equalizer) ProcessStereo(left, right []float32) {
	eq.mu.RLock()
	enabled := eq.enabled
	eq.mu.RUnlock()
	
	if !enabled {
		return
	}
	
	// Apply each band filter to both channels
	for i := 0; i < 10; i++ {
		eq.filters[i].ProcessStereo(left, right)
	}
}

// Reset resets all bands to flat response (0 dB)
func (eq *Equalizer) Reset() {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	
	for i := 0; i < 10; i++ {
		eq.bands[i].Gain = 0
		eq.updateFilter(i)
	}
}

// LoadPreset loads a predefined equalizer preset
func (eq *Equalizer) LoadPreset(preset string) {
	var gains [10]float64
	
	switch preset {
	case "flat":
		gains = [10]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	case "rock":
		gains = [10]float64{5, 4, 3, 1, -1, -1, 1, 3, 4, 5}
	case "pop":
		gains = [10]float64{-2, -1, 0, 2, 4, 4, 2, 0, -1, -2}
	case "jazz":
		gains = [10]float64{0, 0, 0, 2, 4, 4, 2, 0, 0, 0}
	case "classical":
		gains = [10]float64{0, 0, 0, 0, 0, 0, -2, -2, -2, -3}
	case "dance":
		gains = [10]float64{6, 5, 2, 0, 0, -2, -2, -2, 0, 0}
	case "bass_boost":
		gains = [10]float64{8, 6, 4, 2, 0, 0, 0, 0, 0, 0}
	case "treble_boost":
		gains = [10]float64{0, 0, 0, 0, 0, 0, 2, 4, 6, 8}
	case "vocal":
		gains = [10]float64{-2, -3, -3, 1, 4, 4, 3, 1, 0, -1}
	case "powerful":
		gains = [10]float64{6, 5, 0, -2, 1, 3, 5, 6, 4, 0}
	default:
		return
	}
	
	eq.SetAllBands(gains)
}

// GetPresets returns available preset names
func (eq *Equalizer) GetPresets() []string {
	return []string{
		"flat",
		"rock",
		"pop",
		"jazz",
		"classical",
		"dance",
		"bass_boost",
		"treble_boost",
		"vocal",
		"powerful",
	}
}

// updateFilter updates the biquad filter coefficients for a band
func (eq *Equalizer) updateFilter(band int) {
	if band < 0 || band >= 10 {
		return
	}
	
	b := eq.bands[band]
	
	// Convert gain from dB to linear
	gain := math.Pow(10, b.Gain/20)
	
	// Calculate filter coefficients for peaking EQ
	omega := 2 * math.Pi * b.Frequency / float64(eq.sampleRate)
	cos_omega := math.Cos(omega)
	sin_omega := math.Sin(omega)
	alpha := sin_omega / (2 * b.Q)
	
	a := gain
	
	b0 := 1 + alpha*a
	b1 := -2 * cos_omega
	b2 := 1 - alpha*a
	a0 := 1 + alpha/a
	a1 := -2 * cos_omega
	a2 := 1 - alpha/a
	
	// Normalize coefficients
	eq.filters[band].SetCoefficients(
		b0/a0,
		b1/a0,
		b2/a0,
		a1/a0,
		a2/a0,
	)
}

// BiquadFilter implements a second-order IIR filter
type BiquadFilter struct {
	// Coefficients
	b0, b1, b2 float64
	a1, a2     float64
	
	// State variables (for stereo)
	x1L, x2L float64
	y1L, y2L float64
	x1R, x2R float64
	y1R, y2R float64
	
	sampleRate int
	mu         sync.RWMutex
}

// NewBiquadFilter creates a new biquad filter
func NewBiquadFilter(sampleRate int) *BiquadFilter {
	return &BiquadFilter{
		sampleRate: sampleRate,
		b0:         1.0,
		b1:         0.0,
		b2:         0.0,
		a1:         0.0,
		a2:         0.0,
	}
}

// SetCoefficients sets the filter coefficients
func (f *BiquadFilter) SetCoefficients(b0, b1, b2, a1, a2 float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.b0 = b0
	f.b1 = b1
	f.b2 = b2
	f.a1 = a1
	f.a2 = a2
}

// Process applies the filter to mono samples
func (f *BiquadFilter) Process(samples []float32) {
	f.mu.RLock()
	b0, b1, b2 := f.b0, f.b1, f.b2
	a1, a2 := f.a1, f.a2
	x1, x2 := f.x1L, f.x2L
	y1, y2 := f.y1L, f.y2L
	f.mu.RUnlock()
	
	for i := range samples {
		x0 := float64(samples[i])
		y0 := b0*x0 + b1*x1 + b2*x2 - a1*y1 - a2*y2
		
		samples[i] = float32(y0)
		
		x2 = x1
		x1 = x0
		y2 = y1
		y1 = y0
	}
	
	f.mu.Lock()
	f.x1L, f.x2L = x1, x2
	f.y1L, f.y2L = y1, y2
	f.mu.Unlock()
}

// ProcessStereo applies the filter to stereo samples
func (f *BiquadFilter) ProcessStereo(left, right []float32) {
	f.mu.RLock()
	b0, b1, b2 := f.b0, f.b1, f.b2
	a1, a2 := f.a1, f.a2
	x1L, x2L := f.x1L, f.x2L
	y1L, y2L := f.y1L, f.y2L
	x1R, x2R := f.x1R, f.x2R
	y1R, y2R := f.y1R, f.y2R
	f.mu.RUnlock()
	
	for i := range left {
		// Process left channel
		x0L := float64(left[i])
		y0L := b0*x0L + b1*x1L + b2*x2L - a1*y1L - a2*y2L
		left[i] = float32(y0L)
		
		x2L = x1L
		x1L = x0L
		y2L = y1L
		y1L = y0L
		
		// Process right channel
		x0R := float64(right[i])
		y0R := b0*x0R + b1*x1R + b2*x2R - a1*y1R - a2*y2R
		right[i] = float32(y0R)
		
		x2R = x1R
		x1R = x0R
		y2R = y1R
		y1R = y0R
	}
	
	f.mu.Lock()
	f.x1L, f.x2L = x1L, x2L
	f.y1L, f.y2L = y1L, y2L
	f.x1R, f.x2R = x1R, x2R
	f.y1R, f.y2R = y1R, y2R
	f.mu.Unlock()
}

// Reset resets the filter state
func (f *BiquadFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.x1L, f.x2L = 0, 0
	f.y1L, f.y2L = 0, 0
	f.x1R, f.x2R = 0, 0
	f.y1R, f.y2R = 0, 0
}