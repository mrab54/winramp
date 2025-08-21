package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/winramp/winramp/internal/config"
	"github.com/winramp/winramp/internal/logger"
)

var (
	ErrInvalidURL       = errors.New("invalid URL")
	ErrStreamNotFound   = errors.New("stream not found")
	ErrUnsupportedFormat = errors.New("unsupported stream format")
)

// StreamType represents the type of stream
type StreamType string

const (
	StreamTypeHTTP    StreamType = "http"
	StreamTypeRadio   StreamType = "radio"
	StreamTypePodcast StreamType = "podcast"
)

// Stream represents an audio stream
type Stream struct {
	URL         string
	Name        string
	Type        StreamType
	Format      string
	Bitrate     int
	ContentType string
	MetaInt     int // For SHOUTcast/Icecast metadata interval
	reader      io.ReadCloser
	client      *http.Client
	mu          sync.RWMutex
}

// StreamManager manages network streams
type StreamManager struct {
	streams map[string]*Stream
	client  *http.Client
	cache   *StreamCache
	mu      sync.RWMutex
}

// NewStreamManager creates a new stream manager
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*Stream),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		cache: NewStreamCache(),
	}
}

// OpenStream opens a network stream
func (m *StreamManager) OpenStream(ctx context.Context, streamURL string) (*Stream, error) {
	// Validate URL
	u, err := url.Parse(streamURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme %s not supported", ErrInvalidURL, u.Scheme)
	}
	
	// Check cache
	if cached := m.cache.Get(streamURL); cached != nil {
		return cached, nil
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		return nil, err
	}
	
	// Add headers for streaming
	req.Header.Set("User-Agent", "WinRamp/1.0")
	req.Header.Set("Icy-MetaData", "1") // Request metadata for SHOUTcast streams
	req.Header.Set("Accept", "audio/*")
	
	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stream: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: status %d", ErrStreamNotFound, resp.StatusCode)
	}
	
	// Create stream
	stream := &Stream{
		URL:         streamURL,
		Type:        m.detectStreamType(resp),
		ContentType: resp.Header.Get("Content-Type"),
		reader:      resp.Body,
		client:      m.client,
	}
	
	// Parse stream metadata
	m.parseStreamMetadata(stream, resp)
	
	// Detect format
	stream.Format = m.detectFormat(stream.ContentType)
	if stream.Format == "" {
		resp.Body.Close()
		return nil, ErrUnsupportedFormat
	}
	
	// Cache stream
	m.cache.Set(streamURL, stream)
	
	// Store in manager
	m.mu.Lock()
	m.streams[streamURL] = stream
	m.mu.Unlock()
	
	logger.Info("Stream opened",
		logger.String("url", streamURL),
		logger.String("type", string(stream.Type)),
		logger.String("format", stream.Format),
		logger.Int("bitrate", stream.Bitrate),
	)
	
	return stream, nil
}

// CloseStream closes a stream
func (m *StreamManager) CloseStream(streamURL string) error {
	m.mu.Lock()
	stream, exists := m.streams[streamURL]
	if exists {
		delete(m.streams, streamURL)
	}
	m.mu.Unlock()
	
	if stream != nil && stream.reader != nil {
		return stream.reader.Close()
	}
	
	return nil
}

// Read reads data from the stream
func (s *Stream) Read(p []byte) (n int, err error) {
	s.mu.RLock()
	reader := s.reader
	s.mu.RUnlock()
	
	if reader == nil {
		return 0, io.EOF
	}
	
	return reader.Read(p)
}

// Close closes the stream
func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.reader != nil {
		err := s.reader.Close()
		s.reader = nil
		return err
	}
	
	return nil
}

func (m *StreamManager) detectStreamType(resp *http.Response) StreamType {
	// Check for SHOUTcast/Icecast headers
	if resp.Header.Get("icy-name") != "" || resp.Header.Get("icy-br") != "" {
		return StreamTypeRadio
	}
	
	// Check content type
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "audio/") {
		return StreamTypeHTTP
	}
	
	return StreamTypeHTTP
}

func (m *StreamManager) parseStreamMetadata(stream *Stream, resp *http.Response) {
	// Parse Icecast/SHOUTcast headers
	if name := resp.Header.Get("icy-name"); name != "" {
		stream.Name = name
	}
	
	if br := resp.Header.Get("icy-br"); br != "" {
		fmt.Sscanf(br, "%d", &stream.Bitrate)
		stream.Bitrate *= 1000 // Convert to bps
	}
	
	if metaint := resp.Header.Get("icy-metaint"); metaint != "" {
		fmt.Sscanf(metaint, "%d", &stream.MetaInt)
	}
	
	// Parse standard headers
	if stream.Name == "" {
		if name := resp.Header.Get("X-Title"); name != "" {
			stream.Name = name
		}
	}
}

func (m *StreamManager) detectFormat(contentType string) string {
	contentType = strings.ToLower(contentType)
	
	switch {
	case strings.Contains(contentType, "audio/mpeg"), strings.Contains(contentType, "audio/mp3"):
		return "mp3"
	case strings.Contains(contentType, "audio/aac"):
		return "aac"
	case strings.Contains(contentType, "audio/ogg"), strings.Contains(contentType, "application/ogg"):
		return "ogg"
	case strings.Contains(contentType, "audio/flac"):
		return "flac"
	case strings.Contains(contentType, "audio/wav"):
		return "wav"
	default:
		// Try to detect from content type suffix
		parts := strings.Split(contentType, "/")
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
}

// RadioStation represents an internet radio station
type RadioStation struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Genre       string `json:"genre"`
	Country     string `json:"country"`
	Language    string `json:"language"`
	Bitrate     int    `json:"bitrate"`
	Format      string `json:"format"`
	Homepage    string `json:"homepage"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
}

// RadioDirectory provides access to internet radio stations
type RadioDirectory struct {
	stations   []RadioStation
	mu         sync.RWMutex
	configPath string
}

// NewRadioDirectory creates a new radio directory
func NewRadioDirectory(cfg *config.Config) *RadioDirectory {
	configPath := filepath.Join(cfg.App.DataDir, "radio_stations.json")
	rd := &RadioDirectory{
		stations:   make([]RadioStation, 0),
		configPath: configPath,
	}
	rd.loadStations()
	return rd
}

// loadStations loads stations from configuration file
func (d *RadioDirectory) loadStations() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Try to load from configuration file
	if data, err := os.ReadFile(d.configPath); err == nil {
		var stations []RadioStation
		if err := json.Unmarshal(data, &stations); err == nil {
			d.stations = stations
			return nil
		}
	}

	// Use example stations if no config exists
	d.stations = d.getExampleStations()
	
	// Save example stations to config
	return d.saveStations()
}

// getExampleStations returns example stations for initial setup
func (d *RadioDirectory) getExampleStations() []RadioStation {
	return []RadioStation{
		{
			Name:        "Example Station 1",
			URL:         "stream://configure-your-stations.example",
			Genre:       "Various",
			Country:     "US",
			Format:      "mp3",
			Bitrate:     128000,
			Description: "Configure your own stations in radio_stations.json",
		},
		{
			Name:        "Example Station 2",
			URL:         "stream://add-real-urls.example",
			Genre:       "Various",
			Country:     "UK",
			Format:      "mp3",
			Bitrate:     192000,
			Description: "Edit radio_stations.json to add real stations",
		},
	}
}

// saveStations saves stations to configuration file
func (d *RadioDirectory) saveStations() error {
	// Ensure directory exists
	dir := filepath.Dir(d.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(d.stations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stations: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(d.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write stations file: %w", err)
	}

	return nil
}

// GetStations returns all radio stations
func (d *RadioDirectory) GetStations() []RadioStation {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	stations := make([]RadioStation, len(d.stations))
	copy(stations, d.stations)
	return stations
}

// SearchStations searches for stations by name or genre
func (d *RadioDirectory) SearchStations(query string) []RadioStation {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	query = strings.ToLower(query)
	results := make([]RadioStation, 0)
	
	for _, station := range d.stations {
		if strings.Contains(strings.ToLower(station.Name), query) ||
			strings.Contains(strings.ToLower(station.Genre), query) ||
			strings.Contains(strings.ToLower(station.Country), query) {
			results = append(results, station)
		}
	}
	
	return results
}

// AddStation adds a custom radio station
func (d *RadioDirectory) AddStation(station RadioStation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Check for duplicates
	for _, s := range d.stations {
		if s.URL == station.URL {
			return fmt.Errorf("station with URL %s already exists", station.URL)
		}
	}
	
	d.stations = append(d.stations, station)
	return d.saveStations()
}

// RemoveStation removes a station by URL
func (d *RadioDirectory) RemoveStation(url string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for i, s := range d.stations {
		if s.URL == url {
			d.stations = append(d.stations[:i], d.stations[i+1:]...)
			return d.saveStations()
		}
	}
	
	return fmt.Errorf("station not found")
}

// UpdateStation updates an existing station
func (d *RadioDirectory) UpdateStation(url string, updated RadioStation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for i, s := range d.stations {
		if s.URL == url {
			d.stations[i] = updated
			return d.saveStations()
		}
	}
	
	return fmt.Errorf("station not found")
}

// StreamCache caches stream metadata
type StreamCache struct {
	cache map[string]*Stream
	mu    sync.RWMutex
}

// NewStreamCache creates a new stream cache
func NewStreamCache() *StreamCache {
	return &StreamCache{
		cache: make(map[string]*Stream),
	}
}

// Get returns a cached stream
func (c *StreamCache) Get(url string) *Stream {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[url]
}

// Set caches a stream
func (c *StreamCache) Set(url string, stream *Stream) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[url] = stream
}

// Clear clears the cache
func (c *StreamCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*Stream)
}