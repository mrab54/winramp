package decoder

import (
	"errors"
	"io"
	"time"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported audio format")
	ErrInvalidData       = errors.New("invalid audio data")
	ErrSeekNotSupported  = errors.New("seek not supported")
	ErrEndOfStream       = errors.New("end of stream")
)

// AudioFormat represents the format of decoded audio
type AudioFormat struct {
	SampleRate int     // Sample rate in Hz (e.g., 44100)
	Channels   int     // Number of channels (1 = mono, 2 = stereo)
	BitDepth   int     // Bits per sample (e.g., 16, 24)
	Float      bool    // Whether samples are floating point
	Encoding   string  // Encoding type (e.g., "pcm", "float32")
}

// Metadata contains track metadata extracted from the audio file
type Metadata struct {
	Title        string
	Artist       string
	Album        string
	AlbumArtist  string
	Genre        string
	Year         int
	TrackNumber  int
	DiscNumber   int
	Comment      string
	Duration     time.Duration
	Bitrate      int
	VariableBitrate bool
	AlbumArt     []byte
	AlbumArtMIME string
}

// Decoder is the interface for all audio decoders
type Decoder interface {
	// Decode reads and decodes audio data into the provided buffer
	// Returns the number of samples decoded per channel
	Decode(buffer []float32) (int, error)
	
	// DecodeInt16 reads and decodes audio data into int16 format
	DecodeInt16(buffer []int16) (int, error)
	
	// Format returns the audio format of the decoded stream
	Format() AudioFormat
	
	// Metadata returns the metadata of the audio file
	Metadata() *Metadata
	
	// Duration returns the total duration of the audio
	Duration() time.Duration
	
	// Position returns the current playback position
	Position() time.Duration
	
	// Seek seeks to the specified position in the audio stream
	Seek(position time.Duration) error
	
	// SeekSample seeks to a specific sample position
	SeekSample(sample int64) error
	
	// SampleCount returns the total number of samples
	SampleCount() int64
	
	// CurrentSample returns the current sample position
	CurrentSample() int64
	
	// Close closes the decoder and releases resources
	Close() error
}

// StreamDecoder extends Decoder with streaming capabilities
type StreamDecoder interface {
	Decoder
	
	// SetBufferSize sets the internal buffer size for streaming
	SetBufferSize(size int)
	
	// Buffered returns the amount of buffered data in bytes
	Buffered() int
	
	// IsStreaming returns true if this is a streaming source
	IsStreaming() bool
}

// Factory creates decoders for different audio formats
type Factory interface {
	// CreateDecoder creates a decoder for the given reader
	CreateDecoder(reader io.ReadSeeker) (Decoder, error)
	
	// CreateDecoderForFile creates a decoder for a file
	CreateDecoderForFile(path string) (Decoder, error)
	
	// CreateStreamDecoder creates a decoder for streaming
	CreateStreamDecoder(reader io.Reader) (StreamDecoder, error)
	
	// SupportsFormat checks if the factory supports the given format
	SupportsFormat(format string) bool
	
	// SupportedFormats returns a list of supported formats
	SupportedFormats() []string
}

// BaseDecoder provides common functionality for decoders
type BaseDecoder struct {
	format       AudioFormat
	metadata     *Metadata
	sampleCount  int64
	currentSample int64
}

func (d *BaseDecoder) Format() AudioFormat {
	return d.format
}

func (d *BaseDecoder) Metadata() *Metadata {
	return d.metadata
}

func (d *BaseDecoder) SampleCount() int64 {
	return d.sampleCount
}

func (d *BaseDecoder) CurrentSample() int64 {
	return d.currentSample
}

func (d *BaseDecoder) Duration() time.Duration {
	if d.format.SampleRate == 0 {
		return 0
	}
	return time.Duration(d.sampleCount) * time.Second / time.Duration(d.format.SampleRate)
}

func (d *BaseDecoder) Position() time.Duration {
	if d.format.SampleRate == 0 {
		return 0
	}
	return time.Duration(d.currentSample) * time.Second / time.Duration(d.format.SampleRate)
}

// ConvertToFloat32 converts int16 samples to float32 [-1.0, 1.0]
func ConvertToFloat32(samples []int16) []float32 {
	output := make([]float32, len(samples))
	for i, s := range samples {
		output[i] = float32(s) / 32768.0
	}
	return output
}

// ConvertToInt16 converts float32 samples to int16
func ConvertToInt16(samples []float32) []int16 {
	output := make([]int16, len(samples))
	for i, s := range samples {
		// Clamp to [-1.0, 1.0]
		if s < -1.0 {
			s = -1.0
		} else if s > 1.0 {
			s = 1.0
		}
		output[i] = int16(s * 32767.0)
	}
	return output
}

// Interleave interleaves separate channel buffers into a single buffer
func Interleave(channels [][]float32) []float32 {
	if len(channels) == 0 {
		return nil
	}
	
	samplesPerChannel := len(channels[0])
	output := make([]float32, samplesPerChannel*len(channels))
	
	for i := 0; i < samplesPerChannel; i++ {
		for ch, channel := range channels {
			output[i*len(channels)+ch] = channel[i]
		}
	}
	
	return output
}

// Deinterleave separates interleaved audio into separate channel buffers
func Deinterleave(interleaved []float32, channels int) [][]float32 {
	if channels == 0 || len(interleaved) == 0 {
		return nil
	}
	
	samplesPerChannel := len(interleaved) / channels
	output := make([][]float32, channels)
	
	for ch := 0; ch < channels; ch++ {
		output[ch] = make([]float32, samplesPerChannel)
		for i := 0; i < samplesPerChannel; i++ {
			output[ch][i] = interleaved[i*channels+ch]
		}
	}
	
	return output
}