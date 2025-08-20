package decoder

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dhowden/tag"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

// FLACDecoder implements the Decoder interface for FLAC files
type FLACDecoder struct {
	BaseDecoder
	stream      *flac.Stream
	reader      io.ReadSeeker
	currentFrame int
	eof         bool
}

// NewFLACDecoder creates a new FLAC decoder
func NewFLACDecoder(reader io.ReadSeeker) (*FLACDecoder, error) {
	// Parse FLAC stream
	stream, err := flac.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC stream: %w", err)
	}

	// Get format information from stream info
	info := stream.Info
	format := AudioFormat{
		SampleRate: int(info.SampleRate),
		Channels:   int(info.NChannels),
		BitDepth:   int(info.BitsPerSample),
		Float:      false,
		Encoding:   "pcm",
	}

	// Extract metadata
	metadata := &Metadata{
		Duration: time.Duration(info.NSamples) * time.Second / time.Duration(info.SampleRate),
		Bitrate:  int(info.SampleRate * uint32(info.NChannels) * uint32(info.BitsPerSample)),
	}

	// Parse Vorbis comments for metadata
	for _, block := range stream.Blocks {
		switch b := block.Body.(type) {
		case *meta.VorbisComment:
			for _, tag := range b.Tags {
				switch tag[0] {
				case "TITLE":
					metadata.Title = tag[1]
				case "ARTIST":
					metadata.Artist = tag[1]
				case "ALBUM":
					metadata.Album = tag[1]
				case "ALBUMARTIST":
					metadata.AlbumArtist = tag[1]
				case "GENRE":
					metadata.Genre = tag[1]
				case "DATE", "YEAR":
					// Parse year from date string
					if len(tag[1]) >= 4 {
						fmt.Sscanf(tag[1][:4], "%d", &metadata.Year)
					}
				case "TRACKNUMBER":
					fmt.Sscanf(tag[1], "%d", &metadata.TrackNumber)
				case "DISCNUMBER":
					fmt.Sscanf(tag[1], "%d", &metadata.DiscNumber)
				case "COMMENT":
					metadata.Comment = tag[1]
				}
			}
		case *meta.Picture:
			// Get album art
			metadata.AlbumArt = b.Data
			metadata.AlbumArtMIME = b.MIME
		}
	}

	// Fallback to tag library if needed
	if metadata.Title == "" {
		reader.Seek(0, io.SeekStart)
		if m, err := tag.ReadFrom(reader); err == nil {
			if metadata.Title == "" {
				metadata.Title = m.Title()
			}
			if metadata.Artist == "" {
				metadata.Artist = m.Artist()
			}
			if metadata.Album == "" {
				metadata.Album = m.Album()
			}
		}
		reader.Seek(0, io.SeekStart)
		stream, _ = flac.Parse(reader)
	}

	return &FLACDecoder{
		BaseDecoder: BaseDecoder{
			format:      format,
			metadata:    metadata,
			sampleCount: int64(info.NSamples),
		},
		stream: stream,
		reader: reader,
	}, nil
}

// Decode reads and decodes audio data into float32 format
func (d *FLACDecoder) Decode(buffer []float32) (int, error) {
	if d.eof {
		return 0, ErrEndOfStream
	}

	samplesNeeded := len(buffer) / d.format.Channels
	samplesRead := 0

	for samplesRead < samplesNeeded {
		// Check if we have more frames
		if d.currentFrame >= len(d.stream.Frames) {
			// Try to parse next frame
			frame, err := d.stream.ParseNext()
			if err != nil {
				if err == io.EOF {
					d.eof = true
					if samplesRead > 0 {
						return samplesRead, nil
					}
					return 0, ErrEndOfStream
				}
				return samplesRead, fmt.Errorf("failed to parse FLAC frame: %w", err)
			}
			d.stream.Frames = append(d.stream.Frames, frame)
		}

		frame := d.stream.Frames[d.currentFrame]
		
		// Convert samples based on bit depth
		frameIndex := 0
		for samplesRead < samplesNeeded && frameIndex < len(frame.Subframes[0].Samples) {
			for ch := 0; ch < d.format.Channels; ch++ {
				if ch < len(frame.Subframes) {
					sample := frame.Subframes[ch].Samples[frameIndex]
					// Normalize to [-1.0, 1.0]
					buffer[samplesRead*d.format.Channels+ch] = d.normalizeToFloat32(sample)
				}
			}
			frameIndex++
			samplesRead++
		}

		if frameIndex >= len(frame.Subframes[0].Samples) {
			d.currentFrame++
		}
	}

	d.currentSample += int64(samplesRead)
	return samplesRead, nil
}

// DecodeInt16 reads and decodes audio data into int16 format
func (d *FLACDecoder) DecodeInt16(buffer []int16) (int, error) {
	if d.eof {
		return 0, ErrEndOfStream
	}

	samplesNeeded := len(buffer) / d.format.Channels
	samplesRead := 0

	for samplesRead < samplesNeeded {
		// Check if we have more frames
		if d.currentFrame >= len(d.stream.Frames) {
			// Try to parse next frame
			frame, err := d.stream.ParseNext()
			if err != nil {
				if err == io.EOF {
					d.eof = true
					if samplesRead > 0 {
						return samplesRead, nil
					}
					return 0, ErrEndOfStream
				}
				return samplesRead, fmt.Errorf("failed to parse FLAC frame: %w", err)
			}
			d.stream.Frames = append(d.stream.Frames, frame)
		}

		frame := d.stream.Frames[d.currentFrame]
		
		frameIndex := 0
		for samplesRead < samplesNeeded && frameIndex < len(frame.Subframes[0].Samples) {
			for ch := 0; ch < d.format.Channels; ch++ {
				if ch < len(frame.Subframes) {
					sample := frame.Subframes[ch].Samples[frameIndex]
					// Convert to int16
					buffer[samplesRead*d.format.Channels+ch] = d.normalizeToInt16(sample)
				}
			}
			frameIndex++
			samplesRead++
		}

		if frameIndex >= len(frame.Subframes[0].Samples) {
			d.currentFrame++
		}
	}

	d.currentSample += int64(samplesRead)
	return samplesRead, nil
}

func (d *FLACDecoder) normalizeToFloat32(sample int32) float32 {
	// Normalize based on bit depth
	maxValue := float32(1 << (d.format.BitDepth - 1))
	return float32(sample) / maxValue
}

func (d *FLACDecoder) normalizeToInt16(sample int32) int16 {
	// Convert to 16-bit range
	if d.format.BitDepth == 16 {
		return int16(sample)
	} else if d.format.BitDepth > 16 {
		// Downscale
		shift := uint(d.format.BitDepth - 16)
		return int16(sample >> shift)
	} else {
		// Upscale
		shift := uint(16 - d.format.BitDepth)
		return int16(sample << shift)
	}
}

// Seek seeks to the specified position
func (d *FLACDecoder) Seek(position time.Duration) error {
	targetSample := int64(position.Seconds() * float64(d.format.SampleRate))
	return d.SeekSample(targetSample)
}

// SeekSample seeks to a specific sample position
func (d *FLACDecoder) SeekSample(sample int64) error {
	if sample < 0 || sample > d.sampleCount {
		return fmt.Errorf("sample position out of range")
	}

	// Reset stream and seek
	d.reader.Seek(0, io.SeekStart)
	stream, err := flac.Parse(d.reader)
	if err != nil {
		return fmt.Errorf("failed to reparse FLAC stream: %w", err)
	}

	d.stream = stream
	d.currentFrame = 0
	d.currentSample = 0
	d.eof = false

	// Skip samples to reach target position
	// This is not optimal but works for now
	skipBuffer := make([]float32, 1024*d.format.Channels)
	for d.currentSample < sample {
		toSkip := sample - d.currentSample
		if toSkip > 1024 {
			toSkip = 1024
		}
		_, err := d.Decode(skipBuffer[:toSkip*int64(d.format.Channels)])
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the decoder
func (d *FLACDecoder) Close() error {
	if closer, ok := d.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// FLACFactory creates FLAC decoders
type FLACFactory struct{}

// CreateDecoder creates a decoder for the given reader
func (f *FLACFactory) CreateDecoder(reader io.ReadSeeker) (Decoder, error) {
	return NewFLACDecoder(reader)
}

// CreateDecoderForFile creates a decoder for a file
func (f *FLACFactory) CreateDecoderForFile(path string) (Decoder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	decoder, err := NewFLACDecoder(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	
	return decoder, nil
}

// CreateStreamDecoder creates a decoder for streaming
func (f *FLACFactory) CreateStreamDecoder(reader io.Reader) (StreamDecoder, error) {
	return nil, fmt.Errorf("streaming not yet implemented for FLAC")
}

// SupportsFormat checks if the factory supports the given format
func (f *FLACFactory) SupportsFormat(format string) bool {
	return format == "flac" || format == ".flac" || format == "audio/flac"
}

// SupportedFormats returns a list of supported formats
func (f *FLACFactory) SupportedFormats() []string {
	return []string{"flac"}
}