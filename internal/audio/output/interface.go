package output

import (
	"errors"
	"time"
)

var (
	ErrDeviceNotFound     = errors.New("audio device not found")
	ErrDeviceInUse        = errors.New("audio device is in use")
	ErrInvalidFormat      = errors.New("invalid audio format")
	ErrBufferUnderrun     = errors.New("audio buffer underrun")
	ErrDeviceDisconnected = errors.New("audio device disconnected")
)

// Format represents audio output format
type Format struct {
	SampleRate int
	Channels   int
	BitDepth   int
	Latency    time.Duration
}

// Device represents an audio output device
type Device struct {
	ID          string
	Name        string
	Type        string // "WASAPI", "DirectSound", etc.
	IsDefault   bool
	MaxChannels int
	SampleRates []int
	Exclusive   bool // Supports exclusive mode
}

// Output is the interface for audio output backends
type Output interface {
	// Open opens the audio output with the specified format
	Open(format Format) error
	
	// Write writes audio samples to the output
	// Returns the number of samples written
	Write(samples []float32) (int, error)
	
	// WriteInt16 writes int16 samples to the output
	WriteInt16(samples []int16) (int, error)
	
	// Close closes the audio output
	Close() error
	
	// Pause pauses playback
	Pause() error
	
	// Resume resumes playback
	Resume() error
	
	// Flush flushes the audio buffer
	Flush() error
	
	// GetLatency returns the current output latency
	GetLatency() time.Duration
	
	// GetBufferSize returns the buffer size in samples
	GetBufferSize() int
	
	// SetVolume sets the output volume (0.0 to 1.0)
	SetVolume(volume float64) error
	
	// GetVolume returns the current volume
	GetVolume() float64
	
	// IsPlaying returns true if audio is playing
	IsPlaying() bool
	
	// GetDevice returns the current device info
	GetDevice() *Device
	
	// GetPosition returns the current playback position
	GetPosition() time.Duration
}

// DeviceManager manages audio devices
type DeviceManager interface {
	// EnumerateDevices returns all available audio devices
	EnumerateDevices() ([]*Device, error)
	
	// GetDefaultDevice returns the default audio device
	GetDefaultDevice() (*Device, error)
	
	// GetDevice returns a specific device by ID
	GetDevice(id string) (*Device, error)
	
	// CreateOutput creates an output for a device
	CreateOutput(device *Device) (Output, error)
	
	// SetDefaultDevice sets the default audio device
	SetDefaultDevice(id string) error
	
	// WatchDevices watches for device changes
	WatchDevices(callback func(added, removed []*Device))
}

// BaseOutput provides common functionality for outputs
type BaseOutput struct {
	device     *Device
	format     Format
	volume     float64
	isPlaying  bool
	position   time.Duration
	bufferSize int
}

func (o *BaseOutput) GetDevice() *Device {
	return o.device
}

func (o *BaseOutput) GetVolume() float64 {
	return o.volume
}

func (o *BaseOutput) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}
	o.volume = volume
	return nil
}

func (o *BaseOutput) IsPlaying() bool {
	return o.isPlaying
}

func (o *BaseOutput) GetPosition() time.Duration {
	return o.position
}

func (o *BaseOutput) GetBufferSize() int {
	return o.bufferSize
}

func (o *BaseOutput) GetLatency() time.Duration {
	return o.format.Latency
}

// ApplyVolume applies volume to samples
func ApplyVolume(samples []float32, volume float64) {
	for i := range samples {
		samples[i] = float32(float64(samples[i]) * volume)
	}
}

// ApplyVolumeInt16 applies volume to int16 samples
func ApplyVolumeInt16(samples []int16, volume float64) {
	for i := range samples {
		samples[i] = int16(float64(samples[i]) * volume)
	}
}

// ConvertFloat32ToInt16 converts float32 samples to int16
func ConvertFloat32ToInt16(input []float32) []int16 {
	output := make([]int16, len(input))
	for i, sample := range input {
		// Clamp to [-1.0, 1.0]
		if sample < -1.0 {
			sample = -1.0
		} else if sample > 1.0 {
			sample = 1.0
		}
		output[i] = int16(sample * 32767.0)
	}
	return output
}

// ConvertInt16ToFloat32 converts int16 samples to float32
func ConvertInt16ToFloat32(input []int16) []float32 {
	output := make([]float32, len(input))
	for i, sample := range input {
		output[i] = float32(sample) / 32768.0
	}
	return output
}