package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/winramp/winramp/internal/config"
	"github.com/winramp/winramp/internal/domain"
)

// StationManager manages radio stations from configuration
type StationManager struct {
	stations []domain.RadioStation
	mu       sync.RWMutex
	config   *config.Config
}

// NewStationManager creates a new station manager
func NewStationManager(cfg *config.Config) *StationManager {
	sm := &StationManager{
		config:   cfg,
		stations: make([]domain.RadioStation, 0),
	}
	sm.loadStations()
	return sm
}

// loadStations loads stations from configuration file
func (sm *StationManager) loadStations() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Try to load from user config first
	userConfigPath := filepath.Join(sm.config.App.DataDir, "stations.json")
	if _, err := os.Stat(userConfigPath); err == nil {
		return sm.loadStationsFromFile(userConfigPath)
	}

	// Fall back to default stations
	sm.stations = sm.getDefaultStations()
	
	// Save default stations to user config
	return sm.saveStations()
}

// loadStationsFromFile loads stations from a JSON file
func (sm *StationManager) loadStationsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read stations file: %w", err)
	}

	var stations []domain.RadioStation
	if err := json.Unmarshal(data, &stations); err != nil {
		return fmt.Errorf("failed to parse stations file: %w", err)
	}

	sm.stations = stations
	return nil
}

// saveStations saves current stations to user config
func (sm *StationManager) saveStations() error {
	userConfigPath := filepath.Join(sm.config.App.DataDir, "stations.json")
	
	// Ensure directory exists
	dir := filepath.Dir(userConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(sm.stations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stations: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(userConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write stations file: %w", err)
	}

	return nil
}

// GetStations returns all configured stations
func (sm *StationManager) GetStations() []domain.RadioStation {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	stations := make([]domain.RadioStation, len(sm.stations))
	copy(stations, sm.stations)
	return stations
}

// AddStation adds a new station
func (sm *StationManager) AddStation(station domain.RadioStation) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate station
	if station.Name == "" || station.URL == "" {
		return fmt.Errorf("station name and URL are required")
	}

	// Check for duplicates
	for _, s := range sm.stations {
		if s.URL == station.URL {
			return fmt.Errorf("station with URL %s already exists", station.URL)
		}
	}

	sm.stations = append(sm.stations, station)
	return sm.saveStations()
}

// RemoveStation removes a station by URL
func (sm *StationManager) RemoveStation(url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, s := range sm.stations {
		if s.URL == url {
			sm.stations = append(sm.stations[:i], sm.stations[i+1:]...)
			return sm.saveStations()
		}
	}

	return fmt.Errorf("station not found")
}

// UpdateStation updates an existing station
func (sm *StationManager) UpdateStation(url string, updated domain.RadioStation) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, s := range sm.stations {
		if s.URL == url {
			sm.stations[i] = updated
			return sm.saveStations()
		}
	}

	return fmt.Errorf("station not found")
}

// getDefaultStations returns a set of default radio stations
func (sm *StationManager) getDefaultStations() []domain.RadioStation {
	// These are example stations - users should configure their own
	return []domain.RadioStation{
		{
			Name:        "Example Station 1",
			URL:         "stream://example.com/station1",
			Genre:       "Various",
			Country:     "US",
			Bitrate:     128,
			Description: "Configure your own stations in stations.json",
		},
		{
			Name:        "Example Station 2",
			URL:         "stream://example.com/station2",
			Genre:       "Various",
			Country:     "UK",
			Bitrate:     192,
			Description: "Configure your own stations in stations.json",
		},
	}
}