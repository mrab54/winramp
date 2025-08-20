package audio

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/winramp/winramp/internal/audio/decoder"
	"github.com/winramp/winramp/internal/audio/output"
	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/logger"
)

var (
	ErrNoTrackLoaded = errors.New("no track loaded")
	ErrAlreadyPlaying = errors.New("already playing")
	ErrNotPlaying = errors.New("not playing")
)

// PlayerState represents the current state of the player
type PlayerState int

const (
	StateStopped PlayerState = iota
	StatePlaying
	StatePaused
	StateBuffering
	StateError
)

func (s PlayerState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateBuffering:
		return "buffering"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// PlayerEvent represents player events
type PlayerEvent int

const (
	EventStateChanged PlayerEvent = iota
	EventTrackChanged
	EventPositionChanged
	EventVolumeChanged
	EventTrackFinished
	EventError
)

// EventListener is a callback for player events
type EventListener func(event PlayerEvent, data interface{})

// Player is the main audio player
type Player struct {
	// State
	state         PlayerState
	currentTrack  *domain.Track
	nextTrack     *domain.Track
	position      time.Duration
	duration      time.Duration
	volume        float64
	speed         float64
	
	// Audio components
	decoder       decoder.Decoder
	nextDecoder   decoder.Decoder // For gapless playback
	output        output.Output
	deviceManager output.DeviceManager
	
	// Buffering
	buffer        []float32
	bufferSize    int
	prebuffer     []float32 // For gapless playback
	
	// Control
	mu            sync.RWMutex
	playing       chan bool
	stop          chan bool
	seekRequest   chan time.Duration
	
	// Events
	listeners     []EventListener
	listenerMu    sync.RWMutex
	
	// Settings
	crossfade     time.Duration
	gapless       bool
	replayGain    bool
	fadeOnPause   bool
	fadeDuration  time.Duration
}

// NewPlayer creates a new audio player
func NewPlayer() *Player {
	p := &Player{
		state:         StateStopped,
		volume:        1.0,
		speed:         1.0,
		bufferSize:    8192,
		buffer:        make([]float32, 8192),
		playing:       make(chan bool, 1),
		stop:          make(chan bool, 1),
		seekRequest:   make(chan time.Duration, 1),
		listeners:     make([]EventListener, 0),
		crossfade:     5 * time.Second,
		gapless:       true,
		fadeOnPause:   true,
		fadeDuration:  200 * time.Millisecond,
		deviceManager: output.NewOtoDeviceManager(),
	}
	
	// Initialize output device
	if err := p.initializeOutput(); err != nil {
		logger.Error("Failed to initialize audio output", logger.Error(err))
	}
	
	// Start playback loop
	go p.playbackLoop()
	
	return p
}

func (p *Player) initializeOutput() error {
	device, err := p.deviceManager.GetDefaultDevice()
	if err != nil {
		return fmt.Errorf("failed to get default device: %w", err)
	}
	
	p.output, err = p.deviceManager.CreateOutput(device)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	
	// Open with default format
	format := output.Format{
		SampleRate: 44100,
		Channels:   2,
		BitDepth:   16,
		Latency:    50 * time.Millisecond,
	}
	
	if err := p.output.Open(format); err != nil {
		return fmt.Errorf("failed to open output: %w", err)
	}
	
	p.output.SetVolume(p.volume)
	return nil
}

// Load loads a track for playback
func (p *Player) Load(track *domain.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if track == nil {
		return errors.New("track is nil")
	}
	
	// Close existing decoder
	if p.decoder != nil {
		p.decoder.Close()
		p.decoder = nil
	}
	
	// Create new decoder
	dec, err := decoder.CreateDecoderForFile(track.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}
	
	p.decoder = dec
	p.currentTrack = track
	p.position = 0
	p.duration = dec.Duration()
	
	// Update track duration if not set
	if track.Duration == 0 {
		track.Duration = p.duration
	}
	
	p.setState(StateStopped)
	p.notifyListeners(EventTrackChanged, track)
	
	logger.Info("Track loaded",
		logger.String("title", track.GetDisplayTitle()),
		logger.String("artist", track.GetDisplayArtist()),
		logger.Duration("duration", p.duration),
	)
	
	return nil
}

// Play starts or resumes playback
func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.decoder == nil {
		return ErrNoTrackLoaded
	}
	
	switch p.state {
	case StatePlaying:
		return ErrAlreadyPlaying
	case StatePaused:
		if p.output != nil {
			p.output.Resume()
		}
		p.setState(StatePlaying)
		p.playing <- true
	case StateStopped:
		p.setState(StatePlaying)
		p.playing <- true
		if p.output != nil {
			p.output.Resume()
		}
	}
	
	return nil
}

// Pause pauses playback
func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state != StatePlaying {
		return ErrNotPlaying
	}
	
	if p.fadeOnPause {
		// Apply fade out
		go p.fadeOut(p.fadeDuration)
	}
	
	if p.output != nil {
		p.output.Pause()
	}
	
	p.setState(StatePaused)
	return nil
}

// Stop stops playback
func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state == StateStopped {
		return nil
	}
	
	select {
	case p.stop <- true:
	default:
	}
	
	if p.output != nil {
		p.output.Pause()
		p.output.Flush()
	}
	
	p.position = 0
	p.setState(StateStopped)
	
	return nil
}

// Seek seeks to a position in the track
func (p *Player) Seek(position time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.decoder == nil {
		return ErrNoTrackLoaded
	}
	
	if position < 0 || position > p.duration {
		return errors.New("position out of range")
	}
	
	select {
	case p.seekRequest <- position:
	default:
	}
	
	return nil
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (p *Player) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.volume = volume
	if p.output != nil {
		p.output.SetVolume(volume)
	}
	
	p.notifyListeners(EventVolumeChanged, volume)
	return nil
}

// SetSpeed sets the playback speed (0.5 to 2.0)
func (p *Player) SetSpeed(speed float64) error {
	if speed < 0.5 || speed > 2.0 {
		return errors.New("speed must be between 0.5 and 2.0")
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.speed = speed
	return nil
}

// GetState returns the current player state
func (p *Player) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// GetPosition returns the current playback position
func (p *Player) GetPosition() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.position
}

// GetDuration returns the track duration
func (p *Player) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.duration
}

// GetCurrentTrack returns the current track
func (p *Player) GetCurrentTrack() *domain.Track {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTrack
}

// SetNextTrack sets the next track for gapless playback
func (p *Player) SetNextTrack(track *domain.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if track == nil {
		p.nextTrack = nil
		if p.nextDecoder != nil {
			p.nextDecoder.Close()
			p.nextDecoder = nil
		}
		return nil
	}
	
	// Create decoder for next track
	dec, err := decoder.CreateDecoderForFile(track.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create decoder for next track: %w", err)
	}
	
	p.nextTrack = track
	p.nextDecoder = dec
	
	// Pre-buffer if gapless is enabled
	if p.gapless && len(p.prebuffer) > 0 {
		p.nextDecoder.Decode(p.prebuffer)
	}
	
	return nil
}

// AddListener adds an event listener
func (p *Player) AddListener(listener EventListener) {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()
	p.listeners = append(p.listeners, listener)
}

// RemoveListener removes an event listener
func (p *Player) RemoveListener(listener EventListener) {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()
	
	for i, l := range p.listeners {
		// Compare function pointers
		if fmt.Sprintf("%p", l) == fmt.Sprintf("%p", listener) {
			p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
			break
		}
	}
}

func (p *Player) setState(state PlayerState) {
	if p.state != state {
		p.state = state
		p.notifyListeners(EventStateChanged, state)
	}
}

func (p *Player) notifyListeners(event PlayerEvent, data interface{}) {
	p.listenerMu.RLock()
	listeners := make([]EventListener, len(p.listeners))
	copy(listeners, p.listeners)
	p.listenerMu.RUnlock()
	
	for _, listener := range listeners {
		go listener(event, data)
	}
}

func (p *Player) playbackLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.playing:
			p.processAudio()
			
		case <-p.stop:
			p.mu.Lock()
			if p.decoder != nil {
				p.decoder.Close()
				p.decoder = nil
			}
			p.mu.Unlock()
			
		case position := <-p.seekRequest:
			p.mu.Lock()
			if p.decoder != nil {
				if err := p.decoder.Seek(position); err != nil {
					logger.Error("Failed to seek", logger.Error(err))
				} else {
					p.position = position
					p.notifyListeners(EventPositionChanged, position)
				}
			}
			p.mu.Unlock()
			
		case <-ticker.C:
			// Update position periodically
			if p.state == StatePlaying {
				p.mu.RLock()
				pos := p.position
				p.mu.RUnlock()
				p.notifyListeners(EventPositionChanged, pos)
			}
		}
	}
}

func (p *Player) processAudio() {
	p.mu.RLock()
	dec := p.decoder
	out := p.output
	bufSize := p.bufferSize
	p.mu.RUnlock()
	
	if dec == nil || out == nil {
		return
	}
	
	for p.state == StatePlaying {
		// Check for seek requests
		select {
		case position := <-p.seekRequest:
			p.mu.Lock()
			if err := dec.Seek(position); err != nil {
				logger.Error("Failed to seek", logger.Error(err))
			} else {
				p.position = position
			}
			p.mu.Unlock()
			continue
		case <-p.stop:
			return
		default:
		}
		
		// Decode audio
		n, err := dec.Decode(p.buffer[:bufSize])
		if err != nil {
			if err == decoder.ErrEndOfStream {
				// Track finished
				p.handleTrackFinished()
				return
			}
			logger.Error("Decode error", logger.Error(err))
			p.mu.Lock()
			p.setState(StateError)
			p.mu.Unlock()
			return
		}
		
		if n == 0 {
			continue
		}
		
		// Apply speed adjustment if needed
		samples := p.buffer[:n*2] // Stereo
		if p.speed != 1.0 {
			samples = p.applySpeedChange(samples, p.speed)
		}
		
		// Write to output
		_, err = out.Write(samples)
		if err != nil {
			logger.Error("Output error", logger.Error(err))
			continue
		}
		
		// Update position
		p.mu.Lock()
		p.position = dec.Position()
		p.mu.Unlock()
	}
}

func (p *Player) handleTrackFinished() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check for next track (gapless playback)
	if p.nextDecoder != nil && p.nextTrack != nil {
		// Switch to next track
		if p.decoder != nil {
			p.decoder.Close()
		}
		
		p.decoder = p.nextDecoder
		p.currentTrack = p.nextTrack
		p.position = 0
		p.duration = p.decoder.Duration()
		
		p.nextDecoder = nil
		p.nextTrack = nil
		
		p.notifyListeners(EventTrackChanged, p.currentTrack)
		
		// Continue playing
		if p.state == StatePlaying {
			go p.processAudio()
		}
	} else {
		// No next track, stop
		p.setState(StateStopped)
		p.position = 0
		p.notifyListeners(EventTrackFinished, p.currentTrack)
	}
}

func (p *Player) fadeOut(duration time.Duration) {
	steps := int(duration / (10 * time.Millisecond))
	if steps <= 0 {
		steps = 1
	}
	
	startVolume := p.volume
	volumeStep := startVolume / float64(steps)
	
	for i := 0; i < steps; i++ {
		newVolume := startVolume - (volumeStep * float64(i+1))
		if newVolume < 0 {
			newVolume = 0
		}
		
		p.mu.Lock()
		if p.output != nil {
			p.output.SetVolume(newVolume)
		}
		p.mu.Unlock()
		
		time.Sleep(10 * time.Millisecond)
	}
	
	// Restore original volume
	p.mu.Lock()
	if p.output != nil {
		p.output.SetVolume(startVolume)
	}
	p.mu.Unlock()
}

func (p *Player) applySpeedChange(samples []float32, speed float64) []float32 {
	// Simple speed change by resampling
	// This is a basic implementation - production would use a proper resampler
	if speed == 1.0 {
		return samples
	}
	
	inputLen := len(samples)
	outputLen := int(float64(inputLen) / speed)
	output := make([]float32, outputLen)
	
	for i := 0; i < outputLen; i++ {
		srcIndex := int(float64(i) * speed)
		if srcIndex < inputLen {
			output[i] = samples[srcIndex]
		}
	}
	
	return output
}

// Close closes the player and releases resources
func (p *Player) Close() error {
	p.Stop()
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.decoder != nil {
		p.decoder.Close()
		p.decoder = nil
	}
	
	if p.nextDecoder != nil {
		p.nextDecoder.Close()
		p.nextDecoder = nil
	}
	
	if p.output != nil {
		p.output.Close()
		p.output = nil
	}
	
	return nil
}