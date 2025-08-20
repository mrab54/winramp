package dsp

import (
	"errors"
	"math"
	"sync"
)

var (
	ErrInvalidParameter = errors.New("invalid parameter")
	ErrEffectNotFound   = errors.New("effect not found")
)

// Effect is the interface for all DSP effects
type Effect interface {
	// Process applies the effect to audio samples
	Process(samples []float32)
	
	// ProcessStereo applies the effect to stereo samples
	ProcessStereo(left, right []float32)
	
	// SetEnabled enables or disables the effect
	SetEnabled(enabled bool)
	
	// IsEnabled returns whether the effect is enabled
	IsEnabled() bool
	
	// Reset resets the effect state
	Reset()
	
	// GetName returns the effect name
	GetName() string
}

// EffectChain manages a chain of audio effects
type EffectChain struct {
	effects []Effect
	enabled bool
	mu      sync.RWMutex
}

// NewEffectChain creates a new effect chain
func NewEffectChain() *EffectChain {
	return &EffectChain{
		effects: make([]Effect, 0),
		enabled: true,
	}
}

// AddEffect adds an effect to the chain
func (c *EffectChain) AddEffect(effect Effect) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.effects = append(c.effects, effect)
}

// RemoveEffect removes an effect from the chain
func (c *EffectChain) RemoveEffect(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for i, effect := range c.effects {
		if effect.GetName() == name {
			c.effects = append(c.effects[:i], c.effects[i+1:]...)
			return nil
		}
	}
	return ErrEffectNotFound
}

// Process applies all effects in the chain
func (c *EffectChain) Process(samples []float32) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.enabled {
		return
	}
	
	for _, effect := range c.effects {
		if effect.IsEnabled() {
			effect.Process(samples)
		}
	}
}

// ProcessStereo applies all effects to stereo samples
func (c *EffectChain) ProcessStereo(left, right []float32) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.enabled {
		return
	}
	
	for _, effect := range c.effects {
		if effect.IsEnabled() {
			effect.ProcessStereo(left, right)
		}
	}
}

// SetEnabled enables or disables the entire chain
func (c *EffectChain) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// IsEnabled returns whether the chain is enabled
func (c *EffectChain) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// Reset resets all effects in the chain
func (c *EffectChain) Reset() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	for _, effect := range c.effects {
		effect.Reset()
	}
}

// Crossfader implements crossfading between two audio sources
type Crossfader struct {
	position float64 // 0.0 = source A, 1.0 = source B
	curve    string  // "linear", "equal_power", "logarithmic"
	enabled  bool
	mu       sync.RWMutex
}

// NewCrossfader creates a new crossfader
func NewCrossfader() *Crossfader {
	return &Crossfader{
		position: 0.0,
		curve:    "equal_power",
		enabled:  false,
	}
}

// SetPosition sets the crossfade position (0.0 to 1.0)
func (c *Crossfader) SetPosition(position float64) {
	if position < 0.0 {
		position = 0.0
	} else if position > 1.0 {
		position = 1.0
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	c.position = position
}

// GetPosition returns the current crossfade position
func (c *Crossfader) GetPosition() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.position
}

// SetCurve sets the crossfade curve type
func (c *Crossfader) SetCurve(curve string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.curve = curve
}

// Mix mixes two audio sources based on crossfade position
func (c *Crossfader) Mix(sourceA, sourceB, output []float32) {
	c.mu.RLock()
	position := c.position
	curve := c.curve
	enabled := c.enabled
	c.mu.RUnlock()
	
	if !enabled {
		// Just copy source A if disabled
		copy(output, sourceA)
		return
	}
	
	var gainA, gainB float64
	
	switch curve {
	case "linear":
		gainA = 1.0 - position
		gainB = position
		
	case "equal_power":
		// Equal power crossfade for constant perceived volume
		angle := position * math.Pi / 2
		gainA = math.Cos(angle)
		gainB = math.Sin(angle)
		
	case "logarithmic":
		// Logarithmic curve
		if position < 0.5 {
			gainA = 1.0
			gainB = math.Pow(position*2, 2) / 2
		} else {
			gainA = math.Pow((1-position)*2, 2) / 2
			gainB = 1.0
		}
		
	default:
		gainA = 1.0 - position
		gainB = position
	}
	
	// Mix the sources
	for i := range output {
		if i < len(sourceA) && i < len(sourceB) {
			output[i] = float32(float64(sourceA[i])*gainA + float64(sourceB[i])*gainB)
		} else if i < len(sourceA) {
			output[i] = float32(float64(sourceA[i]) * gainA)
		} else if i < len(sourceB) {
			output[i] = float32(float64(sourceB[i]) * gainB)
		} else {
			output[i] = 0
		}
	}
}

// SetEnabled enables or disables the crossfader
func (c *Crossfader) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// IsEnabled returns whether the crossfader is enabled
func (c *Crossfader) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// ReplayGain implements replay gain normalization
type ReplayGain struct {
	trackGain float64
	trackPeak float64
	albumGain float64
	albumPeak float64
	mode      string // "track", "album", "off"
	preamp    float64
	enabled   bool
	mu        sync.RWMutex
}

// NewReplayGain creates a new replay gain processor
func NewReplayGain() *ReplayGain {
	return &ReplayGain{
		mode:    "track",
		preamp:  0.0,
		enabled: false,
	}
}

// SetTrackGain sets the track replay gain values
func (r *ReplayGain) SetTrackGain(gain, peak float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trackGain = gain
	r.trackPeak = peak
}

// SetAlbumGain sets the album replay gain values
func (r *ReplayGain) SetAlbumGain(gain, peak float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.albumGain = gain
	r.albumPeak = peak
}

// SetMode sets the replay gain mode
func (r *ReplayGain) SetMode(mode string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mode = mode
}

// SetPreamp sets the preamp gain in dB
func (r *ReplayGain) SetPreamp(preamp float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.preamp = preamp
}

// Process applies replay gain to samples
func (r *ReplayGain) Process(samples []float32) {
	r.mu.RLock()
	enabled := r.enabled
	mode := r.mode
	preamp := r.preamp
	r.mu.RUnlock()
	
	if !enabled || mode == "off" {
		return
	}
	
	var gain, peak float64
	
	r.mu.RLock()
	if mode == "album" {
		gain = r.albumGain
		peak = r.albumPeak
	} else {
		gain = r.trackGain
		peak = r.trackPeak
	}
	r.mu.RUnlock()
	
	// Calculate total gain
	totalGain := math.Pow(10, (gain+preamp)/20.0)
	
	// Prevent clipping
	if peak > 0 && totalGain*peak > 1.0 {
		totalGain = 1.0 / peak
	}
	
	// Apply gain
	for i := range samples {
		samples[i] = float32(float64(samples[i]) * totalGain)
		
		// Hard limit to prevent clipping
		if samples[i] > 1.0 {
			samples[i] = 1.0
		} else if samples[i] < -1.0 {
			samples[i] = -1.0
		}
	}
}

// ProcessStereo applies replay gain to stereo samples
func (r *ReplayGain) ProcessStereo(left, right []float32) {
	r.Process(left)
	r.Process(right)
}

// SetEnabled enables or disables replay gain
func (r *ReplayGain) SetEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabled
}

// IsEnabled returns whether replay gain is enabled
func (r *ReplayGain) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

// Reset resets replay gain values
func (r *ReplayGain) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trackGain = 0
	r.trackPeak = 0
	r.albumGain = 0
	r.albumPeak = 0
}

// GetName returns the effect name
func (r *ReplayGain) GetName() string {
	return "ReplayGain"
}

// Limiter implements a simple audio limiter
type Limiter struct {
	threshold   float64
	ratio       float64
	attack      float64
	release     float64
	envelope    float64
	enabled     bool
	sampleRate  int
	mu          sync.RWMutex
}

// NewLimiter creates a new limiter
func NewLimiter(sampleRate int) *Limiter {
	return &Limiter{
		threshold:  0.95,
		ratio:      10.0,
		attack:     0.001, // 1ms
		release:    0.050, // 50ms
		envelope:   0.0,
		enabled:    true,
		sampleRate: sampleRate,
	}
}

// Process applies limiting to samples
func (l *Limiter) Process(samples []float32) {
	l.mu.RLock()
	threshold := float32(l.threshold)
	ratio := float32(l.ratio)
	enabled := l.enabled
	l.mu.RUnlock()
	
	if !enabled {
		return
	}
	
	attackCoeff := float32(math.Exp(-1.0 / (l.attack * float64(l.sampleRate))))
	releaseCoeff := float32(math.Exp(-1.0 / (l.release * float64(l.sampleRate))))
	
	for i := range samples {
		input := samples[i]
		absInput := input
		if absInput < 0 {
			absInput = -absInput
		}
		
		// Update envelope
		targetEnv := float32(0.0)
		if absInput > threshold {
			targetEnv = absInput - threshold
		}
		
		var envCoeff float32
		if targetEnv > l.envelope {
			envCoeff = attackCoeff
		} else {
			envCoeff = releaseCoeff
		}
		
		l.envelope = targetEnv + (l.envelope-targetEnv)*float64(envCoeff)
		
		// Apply limiting
		if l.envelope > 0 {
			gain := 1.0 - (l.envelope * float64(1.0-1.0/ratio))
			samples[i] = float32(float64(input) * gain)
		}
	}
}

// ProcessStereo applies limiting to stereo samples
func (l *Limiter) ProcessStereo(left, right []float32) {
	// Process both channels together to maintain stereo image
	combined := make([]float32, len(left)+len(right))
	for i := range left {
		combined[i*2] = left[i]
		combined[i*2+1] = right[i]
	}
	
	l.Process(combined)
	
	for i := range left {
		left[i] = combined[i*2]
		right[i] = combined[i*2+1]
	}
}

// SetEnabled enables or disables the limiter
func (l *Limiter) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// IsEnabled returns whether the limiter is enabled
func (l *Limiter) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

// Reset resets the limiter state
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.envelope = 0.0
}

// GetName returns the effect name
func (l *Limiter) GetName() string {
	return "Limiter"
}