package decoder

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// DecoderFactory manages all available audio decoders
type DecoderFactory struct {
	factories map[string]Factory
}

// NewDecoderFactory creates a new decoder factory with all available decoders
func NewDecoderFactory() *DecoderFactory {
	f := &DecoderFactory{
		factories: make(map[string]Factory),
	}
	
	// Register all available decoders
	f.RegisterFactory("mp3", &MP3Factory{})
	f.RegisterFactory("flac", &FLACFactory{})
	// Future: Add more decoders
	// f.RegisterFactory("ogg", &OGGFactory{})
	// f.RegisterFactory("wav", &WAVFactory{})
	// f.RegisterFactory("aac", &AACFactory{})
	
	return f
}

// RegisterFactory registers a decoder factory for a format
func (f *DecoderFactory) RegisterFactory(format string, factory Factory) {
	f.factories[strings.ToLower(format)] = factory
}

// CreateDecoder creates a decoder based on file extension
func (f *DecoderFactory) CreateDecoder(path string, reader io.ReadSeeker) (Decoder, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	
	factory, exists := f.factories[ext]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
	
	return factory.CreateDecoder(reader)
}

// CreateDecoderForFile creates a decoder for a file
func (f *DecoderFactory) CreateDecoderForFile(path string) (Decoder, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	
	factory, exists := f.factories[ext]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
	
	return factory.CreateDecoderForFile(path)
}

// CreateStreamDecoder creates a streaming decoder based on content type
func (f *DecoderFactory) CreateStreamDecoder(contentType string, reader io.Reader) (StreamDecoder, error) {
	// Map content types to formats
	format := f.contentTypeToFormat(contentType)
	if format == "" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, contentType)
	}
	
	factory, exists := f.factories[format]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	
	return factory.CreateStreamDecoder(reader)
}

// SupportsFormat checks if a format is supported
func (f *DecoderFactory) SupportsFormat(format string) bool {
	format = strings.ToLower(strings.TrimPrefix(format, "."))
	_, exists := f.factories[format]
	return exists
}

// SupportedFormats returns all supported formats
func (f *DecoderFactory) SupportedFormats() []string {
	formats := make([]string, 0, len(f.factories))
	for format := range f.factories {
		formats = append(formats, format)
	}
	return formats
}

func (f *DecoderFactory) contentTypeToFormat(contentType string) string {
	switch strings.ToLower(contentType) {
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/flac":
		return "flac"
	case "audio/ogg", "application/ogg":
		return "ogg"
	case "audio/wav", "audio/wave":
		return "wav"
	case "audio/aac":
		return "aac"
	default:
		return ""
	}
}

// Global decoder factory instance
var globalFactory = NewDecoderFactory()

// GetDecoderFactory returns the global decoder factory
func GetDecoderFactory() *DecoderFactory {
	return globalFactory
}

// CreateDecoderForFile is a convenience function using the global factory
func CreateDecoderForFile(path string) (Decoder, error) {
	return globalFactory.CreateDecoderForFile(path)
}

// SupportsFile checks if a file format is supported
func SupportsFile(path string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	return globalFactory.SupportsFormat(ext)
}