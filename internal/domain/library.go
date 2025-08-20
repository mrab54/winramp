package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrLibraryNotInitialized = errors.New("library not initialized")
	ErrDuplicateLibraryPath  = errors.New("path already exists in library")
	ErrInvalidLibraryPath    = errors.New("invalid library path")
)

type Library struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"not null;uniqueIndex"`
	Description   string         `json:"description"`
	RootPaths     []string       `json:"root_paths" gorm:"type:json"`
	WatchFolders  []WatchFolder  `json:"watch_folders" gorm:"foreignKey:LibraryID"`
	TrackCount    int            `json:"track_count"`
	TotalDuration time.Duration  `json:"total_duration"`
	TotalSize     int64          `json:"total_size"` // in bytes
	LastScanTime  *time.Time     `json:"last_scan_time"`
	IsScanning    bool           `json:"is_scanning" gorm:"-"`
	ScanProgress  float64        `json:"scan_progress" gorm:"-"` // 0-100
	Settings      LibrarySettings `json:"settings" gorm:"embedded"`
	Statistics    LibraryStats   `json:"statistics" gorm:"embedded"`
	UpdatedAt     time.Time      `json:"updated_at"`
	CreatedAt     time.Time      `json:"created_at"`
	
	mu            sync.RWMutex   `json:"-" gorm:"-"`
	tracks        map[string]*Track `json:"-" gorm:"-"` // In-memory cache
	playlists     map[string]*Playlist `json:"-" gorm:"-"`
}

type WatchFolder struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	LibraryID    string    `json:"library_id" gorm:"index"`
	Path         string    `json:"path" gorm:"not null"`
	IsRecursive  bool      `json:"is_recursive" gorm:"default:true"`
	IsEnabled    bool      `json:"is_enabled" gorm:"default:true"`
	IncludeHidden bool     `json:"include_hidden" gorm:"default:false"`
	FilePatterns []string  `json:"file_patterns" gorm:"type:json"` // e.g., ["*.mp3", "*.flac"]
	ExcludePatterns []string `json:"exclude_patterns" gorm:"type:json"`
	LastScanned  *time.Time `json:"last_scanned"`
	CreatedAt    time.Time `json:"created_at"`
}

type LibrarySettings struct {
	AutoScan          bool          `json:"auto_scan" gorm:"default:true"`
	ScanInterval      time.Duration `json:"scan_interval" gorm:"default:3600000000000"` // 1 hour
	WatchForChanges   bool          `json:"watch_for_changes" gorm:"default:true"`
	ExtractMetadata   bool          `json:"extract_metadata" gorm:"default:true"`
	ExtractAlbumArt   bool          `json:"extract_album_art" gorm:"default:true"`
	GenerateWaveforms bool          `json:"generate_waveforms" gorm:"default:false"`
	SkipDuplicates    bool          `json:"skip_duplicates" gorm:"default:true"`
	MinTrackDuration  time.Duration `json:"min_track_duration" gorm:"default:10000000000"` // 10 seconds
	MaxTrackDuration  time.Duration `json:"max_track_duration" gorm:"default:36000000000000"` // 10 hours
}

type LibraryStats struct {
	UniqueArtists int            `json:"unique_artists"`
	UniqueAlbums  int            `json:"unique_albums"`
	UniqueGenres  int            `json:"unique_genres"`
	AverageRating float64        `json:"average_rating"`
	TotalPlayTime time.Duration  `json:"total_play_time"`
	MostPlayedTrack string       `json:"most_played_track"`
	MostPlayedArtist string      `json:"most_played_artist"`
	LastAddedTrack string        `json:"last_added_track"`
	FormatCounts   map[string]int `json:"format_counts" gorm:"type:json"`
	YearRange      YearRange      `json:"year_range" gorm:"embedded"`
}

type YearRange struct {
	Earliest int `json:"earliest"`
	Latest   int `json:"latest"`
}

func NewLibrary(name string) (*Library, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidLibraryPath)
	}

	now := time.Now()
	return &Library{
		ID:           generateLibraryID(),
		Name:         name,
		RootPaths:    make([]string, 0),
		WatchFolders: make([]WatchFolder, 0),
		Settings:     DefaultLibrarySettings(),
		Statistics:   LibraryStats{FormatCounts: make(map[string]int)},
		CreatedAt:    now,
		UpdatedAt:    now,
		tracks:       make(map[string]*Track),
		playlists:    make(map[string]*Playlist),
	}, nil
}

func DefaultLibrarySettings() LibrarySettings {
	return LibrarySettings{
		AutoScan:          true,
		ScanInterval:      time.Hour,
		WatchForChanges:   true,
		ExtractMetadata:   true,
		ExtractAlbumArt:   true,
		GenerateWaveforms: false,
		SkipDuplicates:    true,
		MinTrackDuration:  10 * time.Second,
		MaxTrackDuration:  10 * time.Hour,
	}
}

func (l *Library) AddWatchFolder(path string, recursive bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if path already exists
	for _, folder := range l.WatchFolders {
		if folder.Path == path {
			return ErrDuplicateLibraryPath
		}
	}

	// Validate path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLibraryPath, err)
	}

	watchFolder := WatchFolder{
		ID:          generateWatchFolderID(),
		LibraryID:   l.ID,
		Path:        absPath,
		IsRecursive: recursive,
		IsEnabled:   true,
		CreatedAt:   time.Now(),
		FilePatterns: []string{"*.mp3", "*.flac", "*.ogg", "*.wav", "*.aac", "*.wma", "*.m4a", "*.opus"},
	}

	l.WatchFolders = append(l.WatchFolders, watchFolder)
	l.RootPaths = append(l.RootPaths, absPath)
	l.UpdatedAt = time.Now()

	return nil
}

func (l *Library) RemoveWatchFolder(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	index := -1
	for i, folder := range l.WatchFolders {
		if folder.Path == path {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("watch folder not found: %s", path)
	}

	l.WatchFolders = append(l.WatchFolders[:index], l.WatchFolders[index+1:]...)
	
	// Remove from root paths
	for i, p := range l.RootPaths {
		if p == path {
			l.RootPaths = append(l.RootPaths[:i], l.RootPaths[i+1:]...)
			break
		}
	}

	l.UpdatedAt = time.Now()
	return nil
}

func (l *Library) AddTrack(track *Track) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if track == nil {
		return ErrInvalidTrack
	}

	if err := track.Validate(); err != nil {
		return err
	}

	l.tracks[track.ID] = track
	l.updateStatistics()
	return nil
}

func (l *Library) RemoveTrack(trackID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.tracks[trackID]; !exists {
		return ErrTrackNotFound
	}

	delete(l.tracks, trackID)
	l.updateStatistics()
	return nil
}

func (l *Library) GetTrack(trackID string) (*Track, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	track, exists := l.tracks[trackID]
	if !exists {
		return nil, ErrTrackNotFound
	}
	return track, nil
}

func (l *Library) GetAllTracks() []*Track {
	l.mu.RLock()
	defer l.mu.RUnlock()

	tracks := make([]*Track, 0, len(l.tracks))
	for _, track := range l.tracks {
		tracks = append(tracks, track)
	}
	return tracks
}

func (l *Library) AddPlaylist(playlist *Playlist) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if playlist == nil {
		return ErrInvalidPlaylist
	}

	if err := playlist.Validate(); err != nil {
		return err
	}

	l.playlists[playlist.ID] = playlist
	l.UpdatedAt = time.Now()
	return nil
}

func (l *Library) RemovePlaylist(playlistID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.playlists[playlistID]; !exists {
		return fmt.Errorf("playlist not found: %s", playlistID)
	}

	delete(l.playlists, playlistID)
	l.UpdatedAt = time.Now()
	return nil
}

func (l *Library) GetPlaylist(playlistID string) (*Playlist, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	playlist, exists := l.playlists[playlistID]
	if !exists {
		return nil, fmt.Errorf("playlist not found: %s", playlistID)
	}
	return playlist, nil
}

func (l *Library) GetAllPlaylists() []*Playlist {
	l.mu.RLock()
	defer l.mu.RUnlock()

	playlists := make([]*Playlist, 0, len(l.playlists))
	for _, playlist := range l.playlists {
		playlists = append(playlists, playlist)
	}
	return playlists
}

func (l *Library) StartScan() {
	l.mu.Lock()
	l.IsScanning = true
	l.ScanProgress = 0
	now := time.Now()
	l.LastScanTime = &now
	l.mu.Unlock()
}

func (l *Library) StopScan() {
	l.mu.Lock()
	l.IsScanning = false
	l.ScanProgress = 100
	l.mu.Unlock()
}

func (l *Library) UpdateScanProgress(progress float64) {
	l.mu.Lock()
	l.ScanProgress = progress
	l.mu.Unlock()
}

func (l *Library) updateStatistics() {
	// This would calculate all library statistics
	// For now, just update basic counts
	l.TrackCount = len(l.tracks)
	
	var totalDuration time.Duration
	var totalSize int64
	artistMap := make(map[string]bool)
	albumMap := make(map[string]bool)
	genreMap := make(map[string]bool)
	
	for _, track := range l.tracks {
		totalDuration += track.Duration
		totalSize += track.FileSize
		
		if track.Artist != "" {
			artistMap[track.Artist] = true
		}
		if track.Album != "" {
			albumMap[track.Album] = true
		}
		if track.Genre != "" {
			genreMap[track.Genre] = true
		}
		
		// Update format counts
		if l.Statistics.FormatCounts == nil {
			l.Statistics.FormatCounts = make(map[string]int)
		}
		l.Statistics.FormatCounts[string(track.Format)]++
	}
	
	l.TotalDuration = totalDuration
	l.TotalSize = totalSize
	l.Statistics.UniqueArtists = len(artistMap)
	l.Statistics.UniqueAlbums = len(albumMap)
	l.Statistics.UniqueGenres = len(genreMap)
	l.UpdatedAt = time.Now()
}

func (l *Library) GetStatistics() LibraryStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Statistics
}

func (l *Library) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tracks = make(map[string]*Track)
	l.playlists = make(map[string]*Playlist)
	l.TrackCount = 0
	l.TotalDuration = 0
	l.TotalSize = 0
	l.Statistics = LibraryStats{FormatCounts: make(map[string]int)}
	l.UpdatedAt = time.Now()
}

func generateLibraryID() string {
	return fmt.Sprintf("library_%d_%d", time.Now().UnixNano(), randomInt())
}

func generateWatchFolderID() string {
	return fmt.Sprintf("watch_%d_%d", time.Now().UnixNano(), randomInt())
}

type LibraryRepository interface {
	Create(library *Library) error
	Update(library *Library) error
	Delete(id string) error
	FindByID(id string) (*Library, error)
	FindByName(name string) (*Library, error)
	FindAll() ([]*Library, error)
	GetDefault() (*Library, error)
	SetDefault(id string) error
	UpdateStatistics(library *Library) error
}