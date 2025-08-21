package decoder

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dhowden/tag"
	"github.com/hajimehoshi/go-mp3"
)

// MP3Decoder implements the Decoder interface for MP3 files
type MP3Decoder struct {
	BaseDecoder
	reader     io.ReadSeeker
	decoder    *mp3.Decoder
	buffer     []byte
	eof        bool
}

// NewMP3Decoder creates a new MP3 decoder
func NewMP3Decoder(reader io.ReadSeeker) (*MP3Decoder, error) {
	// Create MP3 decoder
	decoder, err := mp3.NewDecoder(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create MP3 decoder: %w", err)
	}

	// Get format information
	format := AudioFormat{
		SampleRate: decoder.SampleRate(),
		Channels:   2, // MP3 decoder always outputs stereo
		BitDepth:   16,
		Float:      false,
		Encoding:   "pcm",
	}

	// Extract metadata
	metadata := &Metadata{}
	if seeker, ok := reader.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
		if m, err := tag.ReadFrom(reader); err == nil {
			metadata.Title = m.Title()
			metadata.Artist = m.Artist()
			metadata.Album = m.Album()
			metadata.AlbumArtist = m.AlbumArtist()
			metadata.Genre = m.Genre()
			metadata.Year = m.Year()
			
			if track, _ := m.Track(); track > 0 {
				metadata.TrackNumber = track
			}
			if disc, _ := m.Disc(); disc > 0 {
				metadata.DiscNumber = disc
			}
			
			metadata.Comment = m.Comment()
			
			// Get album art if available
			if pic := m.Picture(); pic != nil {
				metadata.AlbumArt = pic.Data
				metadata.AlbumArtMIME = pic.MIMEType
			}
		}
		// Reset reader position
		seeker.Seek(0, io.SeekStart)
		decoder, _ = mp3.NewDecoder(reader)
	}

	// Calculate duration and sample count
	sampleCount := decoder.Length() / 4 // 2 channels * 2 bytes per sample
	duration := time.Duration(sampleCount) * time.Second / time.Duration(format.SampleRate)
	metadata.Duration = duration

	// Use a reasonable initial buffer size
	initialBufferSize := 4096
	if initialBufferSize > 1024*1024 {
		initialBufferSize = 1024 * 1024
	}
	
	return &MP3Decoder{
		BaseDecoder: BaseDecoder{
			format:      format,
			metadata:    metadata,
			sampleCount: sampleCount,
		},
		reader:  reader,
		decoder: decoder,
		buffer:  make([]byte, initialBufferSize),
	}, nil
}

// Decode reads and decodes audio data into float32 format
func (d *MP3Decoder) Decode(buffer []float32) (int, error) {
	if d.eof {
		return 0, ErrEndOfStream
	}
	
	// Validate input buffer
	if len(buffer) == 0 {
		return 0, nil
	}
	
	// Limit buffer size to prevent excessive memory allocation
	const maxBufferSize = 1024 * 1024 // 1MB max
	if len(buffer) > maxBufferSize/2 {
		return 0, fmt.Errorf("buffer size exceeds maximum allowed: %d > %d", len(buffer), maxBufferSize/2)
	}

	// Calculate bytes needed
	bytesNeeded := len(buffer) * 2 // 2 bytes per sample (int16)
	if bytesNeeded > len(d.buffer) {
		// Allocate with size limit
		if bytesNeeded > maxBufferSize {
			bytesNeeded = maxBufferSize
		}
		d.buffer = make([]byte, bytesNeeded)
	}

	// Read from decoder
	n, err := d.decoder.Read(d.buffer[:bytesNeeded])
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to decode MP3: %w", err)
	}

	if n == 0 {
		d.eof = true
		return 0, ErrEndOfStream
	}

	// Convert bytes to int16 then to float32
	samplesRead := n / 2
	// Ensure we don't write beyond buffer bounds
	if samplesRead > len(buffer) {
		samplesRead = len(buffer)
	}
	
	for i := 0; i < samplesRead; i++ {
		// Bounds check for safety
		if i*2+1 >= n {
			break
		}
		sample := int16(d.buffer[i*2]) | int16(d.buffer[i*2+1])<<8
		buffer[i] = float32(sample) / 32768.0
	}

	d.currentSample += int64(samplesRead / d.format.Channels)
	return samplesRead / d.format.Channels, nil
}

// DecodeInt16 reads and decodes audio data into int16 format
func (d *MP3Decoder) DecodeInt16(buffer []int16) (int, error) {
	if d.eof {
		return 0, ErrEndOfStream
	}
	
	// Validate input buffer
	if len(buffer) == 0 {
		return 0, nil
	}
	
	// Limit buffer size to prevent excessive memory allocation
	const maxBufferSize = 1024 * 1024 // 1MB max
	if len(buffer) > maxBufferSize/2 {
		return 0, fmt.Errorf("buffer size exceeds maximum allowed: %d > %d", len(buffer), maxBufferSize/2)
	}

	// Calculate bytes needed
	bytesNeeded := len(buffer) * 2
	if bytesNeeded > len(d.buffer) {
		// Allocate with size limit
		if bytesNeeded > maxBufferSize {
			bytesNeeded = maxBufferSize
		}
		d.buffer = make([]byte, bytesNeeded)
	}

	// Read from decoder
	n, err := d.decoder.Read(d.buffer[:bytesNeeded])
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to decode MP3: %w", err)
	}

	if n == 0 {
		d.eof = true
		return 0, ErrEndOfStream
	}

	// Convert bytes to int16
	samplesRead := n / 2
	// Ensure we don't write beyond buffer bounds
	if samplesRead > len(buffer) {
		samplesRead = len(buffer)
	}
	
	for i := 0; i < samplesRead; i++ {
		// Bounds check for safety
		if i*2+1 >= n {
			break
		}
		buffer[i] = int16(d.buffer[i*2]) | int16(d.buffer[i*2+1])<<8
	}

	d.currentSample += int64(samplesRead / d.format.Channels)
	return samplesRead / d.format.Channels, nil
}

// Seek seeks to the specified position
func (d *MP3Decoder) Seek(position time.Duration) error {
	targetSample := int64(position.Seconds() * float64(d.format.SampleRate))
	return d.SeekSample(targetSample)
}

// SeekSample seeks to a specific sample position
func (d *MP3Decoder) SeekSample(sample int64) error {
	if sample < 0 {
		return fmt.Errorf("sample position cannot be negative: %d", sample)
	}
	if sample > d.sampleCount {
		return fmt.Errorf("sample position out of range: %d > %d", sample, d.sampleCount)
	}

	// Calculate byte position (approximate for MP3)
	bytePosition := sample * 4 // 2 channels * 2 bytes per sample
	
	if seeker, ok := d.reader.(io.Seeker); ok {
		_, err := seeker.Seek(bytePosition, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
		
		// Recreate decoder at new position
		d.decoder, err = mp3.NewDecoder(d.reader)
		if err != nil {
			return fmt.Errorf("failed to recreate decoder: %w", err)
		}
		
		d.currentSample = sample
		d.eof = false
		return nil
	}

	return ErrSeekNotSupported
}

// Close closes the decoder
func (d *MP3Decoder) Close() error {
	if closer, ok := d.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// MP3Factory creates MP3 decoders
type MP3Factory struct{}

// CreateDecoder creates a decoder for the given reader
func (f *MP3Factory) CreateDecoder(reader io.ReadSeeker) (Decoder, error) {
	return NewMP3Decoder(reader)
}

// CreateDecoderForFile creates a decoder for a file
func (f *MP3Factory) CreateDecoderForFile(path string) (Decoder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	decoder, err := NewMP3Decoder(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	
	return decoder, nil
}

// CreateStreamDecoder creates a decoder for streaming
func (f *MP3Factory) CreateStreamDecoder(reader io.Reader) (StreamDecoder, error) {
	// For streaming, we need a reader that supports seeking for metadata
	// In practice, we might buffer the stream
	return nil, fmt.Errorf("streaming not yet implemented for MP3")
}

// SupportsFormat checks if the factory supports the given format
func (f *MP3Factory) SupportsFormat(format string) bool {
	return format == "mp3" || format == ".mp3" || format == "audio/mpeg"
}

// SupportedFormats returns a list of supported formats
func (f *MP3Factory) SupportedFormats() []string {
	return []string{"mp3"}
}