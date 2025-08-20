package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/winramp/winramp/internal/audio"
	"github.com/winramp/winramp/internal/audio/decoder"
	"github.com/winramp/winramp/internal/config"
	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/infrastructure/db"
	"github.com/winramp/winramp/internal/library"
	"github.com/winramp/winramp/internal/playlist"
)

func TestIntegration_FullPlaybackFlow(t *testing.T) {
	// Skip if no test files available
	testFile := "testdata/sample.mp3"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not found:", testFile)
	}
	
	// Initialize components
	cfg := config.Get()
	database := setupTestDatabase(t)
	defer database.Close()
	
	trackRepo := db.NewTrackRepository(database)
	player := audio.NewPlayer()
	defer player.Close()
	
	// Test loading a track
	track, err := domain.NewTrack(testFile)
	require.NoError(t, err)
	require.NotNil(t, track)
	
	// Save track to database
	err = trackRepo.Create(track)
	require.NoError(t, err)
	
	// Load track in player
	err = player.Load(track)
	assert.NoError(t, err)
	
	// Test playback controls
	err = player.Play()
	assert.NoError(t, err)
	assert.Equal(t, audio.StatePlaying, player.GetState())
	
	// Let it play for a moment
	time.Sleep(100 * time.Millisecond)
	
	err = player.Pause()
	assert.NoError(t, err)
	assert.Equal(t, audio.StatePaused, player.GetState())
	
	err = player.Stop()
	assert.NoError(t, err)
	assert.Equal(t, audio.StateStopped, player.GetState())
	
	// Test seeking
	err = player.Seek(5 * time.Second)
	assert.NoError(t, err)
	
	// Test volume
	err = player.SetVolume(0.5)
	assert.NoError(t, err)
}

func TestIntegration_PlaylistManagement(t *testing.T) {
	database := setupTestDatabase(t)
	defer database.Close()
	
	playlistRepo := &mockPlaylistRepo{
		playlists: make(map[string]*domain.Playlist),
	}
	
	mgr := playlist.NewManager(playlistRepo)
	
	// Create playlist
	pl, err := mgr.Create("Test Playlist")
	require.NoError(t, err)
	require.NotNil(t, pl)
	
	// Add tracks
	track1, _ := domain.NewTrack("track1.mp3")
	track2, _ := domain.NewTrack("track2.mp3")
	
	err = mgr.AddTrack(pl.ID, track1)
	assert.NoError(t, err)
	
	err = mgr.AddTrack(pl.ID, track2)
	assert.NoError(t, err)
	
	// Set as current playlist
	err = mgr.SetCurrentPlaylist(pl.ID)
	assert.NoError(t, err)
	
	// Get next track
	next := mgr.GetNextTrack()
	assert.NotNil(t, next)
	assert.Equal(t, track1.ID, next.ID)
	
	// Queue operations
	mgr.AddToQueue(track2)
	queue := mgr.GetQueue()
	assert.False(t, queue.IsEmpty())
}

func TestIntegration_LibraryScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping library scanning test in short mode")
	}
	
	database := setupTestDatabase(t)
	defer database.Close()
	
	trackRepo := db.NewTrackRepository(database)
	scanner := library.NewScanner(trackRepo, nil)
	
	// Create test directory with sample files
	testDir := t.TempDir()
	createTestAudioFile(t, filepath.Join(testDir, "test1.mp3"))
	createTestAudioFile(t, filepath.Join(testDir, "test2.mp3"))
	
	// Scan directory
	ctx := context.Background()
	result, err := scanner.ScanFolder(ctx, testDir)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Check results
	assert.Equal(t, 2, result.ScannedFiles)
	assert.Equal(t, 2, result.ImportedTracks)
	assert.Equal(t, 0, result.FailedFiles)
	
	// Verify tracks in database
	tracks, err := trackRepo.FindAll()
	require.NoError(t, err)
	assert.Len(t, tracks, 2)
}

func TestIntegration_DecoderFormats(t *testing.T) {
	factory := decoder.GetDecoderFactory()
	
	// Test MP3 support
	assert.True(t, factory.SupportsFormat("mp3"))
	
	// Test FLAC support
	assert.True(t, factory.SupportsFormat("flac"))
	
	// Test unsupported format
	assert.False(t, factory.SupportsFormat("xyz"))
	
	formats := factory.SupportedFormats()
	assert.Contains(t, formats, "mp3")
	assert.Contains(t, formats, "flac")
}

func TestIntegration_ConfigurationManagement(t *testing.T) {
	cfg := config.Get()
	
	// Test audio settings
	assert.NotNil(t, cfg.Audio)
	assert.Greater(t, cfg.Audio.BufferSize, 0)
	assert.Greater(t, cfg.Audio.SampleRate, 0)
	
	// Test library settings
	assert.NotNil(t, cfg.Library)
	assert.NotEmpty(t, cfg.Library.FilePatterns)
	
	// Test UI settings
	assert.NotNil(t, cfg.UI)
	assert.NotEmpty(t, cfg.UI.WindowMode)
	
	// Test setting values
	cfg.Set("audio.volume", 0.75)
	assert.Equal(t, 0.75, cfg.Audio.Volume)
}

// Helper functions

func setupTestDatabase(t *testing.T) *db.Database {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cfg := db.Config{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}
	
	database := &db.Database{}
	err := database.Initialize(cfg)
	require.NoError(t, err)
	
	return database
}

func createTestAudioFile(t *testing.T, path string) {
	// Create a minimal valid MP3 file (just the header)
	// This is a simplified MP3 header for testing
	mp3Header := []byte{
		0xFF, 0xFB, 0x90, 0x00, // MP3 header
		0x00, 0x00, 0x00, 0x00,
	}
	
	err := os.WriteFile(path, mp3Header, 0644)
	require.NoError(t, err)
}

// Mock implementations

type mockPlaylistRepo struct {
	playlists map[string]*domain.Playlist
}

func (r *mockPlaylistRepo) Create(playlist *domain.Playlist) error {
	r.playlists[playlist.ID] = playlist
	return nil
}

func (r *mockPlaylistRepo) Update(playlist *domain.Playlist) error {
	r.playlists[playlist.ID] = playlist
	return nil
}

func (r *mockPlaylistRepo) Delete(id string) error {
	delete(r.playlists, id)
	return nil
}

func (r *mockPlaylistRepo) FindByID(id string) (*domain.Playlist, error) {
	if pl, ok := r.playlists[id]; ok {
		return pl, nil
	}
	return nil, domain.ErrNotFound
}

func (r *mockPlaylistRepo) FindByName(name string) (*domain.Playlist, error) {
	for _, pl := range r.playlists {
		if pl.Name == name {
			return pl, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *mockPlaylistRepo) FindAll() ([]*domain.Playlist, error) {
	result := make([]*domain.Playlist, 0, len(r.playlists))
	for _, pl := range r.playlists {
		result = append(result, pl)
	}
	return result, nil
}

func (r *mockPlaylistRepo) FindByType(playlistType domain.PlaylistType) ([]*domain.Playlist, error) {
	result := make([]*domain.Playlist, 0)
	for _, pl := range r.playlists {
		if pl.Type == playlistType {
			result = append(result, pl)
		}
	}
	return result, nil
}

func (r *mockPlaylistRepo) FindFavorites() ([]*domain.Playlist, error) {
	result := make([]*domain.Playlist, 0)
	for _, pl := range r.playlists {
		if pl.IsFavorite {
			result = append(result, pl)
		}
	}
	return result, nil
}

func (r *mockPlaylistRepo) GetRecentlyPlayed(limit int) ([]*domain.Playlist, error) {
	return []*domain.Playlist{}, nil
}

func (r *mockPlaylistRepo) SaveVersion(playlist *domain.Playlist) error {
	return nil
}

func (r *mockPlaylistRepo) GetVersion(playlistID string, version int) (*domain.PlaylistVersion, error) {
	return nil, domain.ErrNotFound
}

func (r *mockPlaylistRepo) Count() (int64, error) {
	return int64(len(r.playlists)), nil
}