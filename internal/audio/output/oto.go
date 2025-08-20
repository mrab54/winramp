package output

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/oto/v3"
)

// OtoOutput implements Output interface using oto library
type OtoOutput struct {
	BaseOutput
	context *oto.Context
	player  oto.Player
	mu      sync.Mutex
	closed  bool
}

// NewOtoOutput creates a new Oto-based audio output
func NewOtoOutput(device *Device) *OtoOutput {
	return &OtoOutput{
		BaseOutput: BaseOutput{
			device: device,
			volume: 1.0,
		},
	}
}

// Open opens the audio output with the specified format
func (o *OtoOutput) Open(format Format) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.context != nil {
		return fmt.Errorf("output already open")
	}

	// Create oto context options
	options := &oto.NewContextOptions{
		SampleRate:   format.SampleRate,
		ChannelCount: format.Channels,
		Format:       oto.FormatFloat32LE,
	}

	// Calculate buffer size based on latency
	if format.Latency > 0 {
		samplesPerSecond := format.SampleRate * format.Channels
		bufferSamples := int(format.Latency.Seconds() * float64(samplesPerSecond))
		options.BufferSize = time.Duration(bufferSamples) * time.Second / time.Duration(samplesPerSecond)
	}

	// Create context
	context, ready, err := oto.NewContext(options)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}

	// Wait for context to be ready
	<-ready

	o.context = context
	o.format = format
	o.bufferSize = int(options.BufferSize.Seconds() * float64(format.SampleRate))
	
	// Create player
	o.player = o.context.NewPlayer(o)
	o.player.Play()
	o.isPlaying = true

	return nil
}

// Read implements io.Reader for oto.Player
func (o *OtoOutput) Read(p []byte) (n int, err error) {
	// This method is called by oto when it needs audio data
	// For now, return silence (zeros)
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// Write writes audio samples to the output
func (o *OtoOutput) Write(samples []float32) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed || o.player == nil {
		return 0, fmt.Errorf("output not open")
	}

	// Apply volume
	if o.volume != 1.0 {
		ApplyVolume(samples, o.volume)
	}

	// Convert float32 to bytes for oto
	bytes := make([]byte, len(samples)*4)
	for i, sample := range samples {
		// Convert float32 to bytes (little-endian)
		bits := float32ToUint32(sample)
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}

	written, err := o.player.Write(bytes)
	if err != nil {
		return 0, fmt.Errorf("failed to write audio: %w", err)
	}

	samplesWritten := written / 4
	
	// Update position
	o.position += time.Duration(samplesWritten/o.format.Channels) * time.Second / time.Duration(o.format.SampleRate)

	return samplesWritten, nil
}

// WriteInt16 writes int16 samples to the output
func (o *OtoOutput) WriteInt16(samples []int16) (int, error) {
	// Convert int16 to float32
	float32Samples := ConvertInt16ToFloat32(samples)
	return o.Write(float32Samples)
}

// Close closes the audio output
func (o *OtoOutput) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return nil
	}

	o.closed = true

	if o.player != nil {
		o.player.Close()
		o.player = nil
	}

	if o.context != nil {
		// Note: oto v3 doesn't have a Close method for context
		o.context = nil
	}

	return nil
}

// Pause pauses playback
func (o *OtoOutput) Pause() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.player == nil {
		return fmt.Errorf("output not open")
	}

	o.player.Pause()
	o.isPlaying = false
	return nil
}

// Resume resumes playback
func (o *OtoOutput) Resume() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.player == nil {
		return fmt.Errorf("output not open")
	}

	o.player.Play()
	o.isPlaying = true
	return nil
}

// Flush flushes the audio buffer
func (o *OtoOutput) Flush() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.player == nil {
		return fmt.Errorf("output not open")
	}

	// Reset position
	o.player.Reset()
	o.position = 0
	return nil
}

// OtoDeviceManager implements DeviceManager using oto
type OtoDeviceManager struct {
	defaultDevice *Device
	mu            sync.RWMutex
}

// NewOtoDeviceManager creates a new Oto device manager
func NewOtoDeviceManager() *OtoDeviceManager {
	return &OtoDeviceManager{
		defaultDevice: &Device{
			ID:          "default",
			Name:        "Default Audio Device",
			Type:        "Oto",
			IsDefault:   true,
			MaxChannels: 2,
			SampleRates: []int{22050, 44100, 48000, 88200, 96000, 192000},
		},
	}
}

// EnumerateDevices returns all available audio devices
func (m *OtoDeviceManager) EnumerateDevices() ([]*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Oto doesn't provide device enumeration, return default device
	return []*Device{m.defaultDevice}, nil
}

// GetDefaultDevice returns the default audio device
func (m *OtoDeviceManager) GetDefaultDevice() (*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultDevice, nil
}

// GetDevice returns a specific device by ID
func (m *OtoDeviceManager) GetDevice(id string) (*Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id == "default" || id == m.defaultDevice.ID {
		return m.defaultDevice, nil
	}
	return nil, ErrDeviceNotFound
}

// CreateOutput creates an output for a device
func (m *OtoDeviceManager) CreateOutput(device *Device) (Output, error) {
	if device == nil {
		device = m.defaultDevice
	}
	return NewOtoOutput(device), nil
}

// SetDefaultDevice sets the default audio device
func (m *OtoDeviceManager) SetDefaultDevice(id string) error {
	// Oto doesn't support changing devices
	if id != "default" && id != m.defaultDevice.ID {
		return ErrDeviceNotFound
	}
	return nil
}

// WatchDevices watches for device changes
func (m *OtoDeviceManager) WatchDevices(callback func(added, removed []*Device)) {
	// Oto doesn't support device watching
}

// Helper function to convert float32 to uint32
func float32ToUint32(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}

// For systems that don't have unsafe
func float32ToUint32Safe(f float32) uint32 {
	bytes := make([]byte, 4)
	// This is a simplified conversion - in production use encoding/binary
	if f >= 0 {
		return uint32(f * 2147483647)
	}
	return uint32(int32(f * 2147483648))
}