package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	
	"github.com/winramp/winramp/internal/audio"
	"github.com/winramp/winramp/internal/config"
	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/infrastructure/db"
	"github.com/winramp/winramp/internal/logger"
	"github.com/winramp/winramp/internal/playlist"
)

// App struct
type App struct {
	ctx           context.Context
	config        *config.Config
	player        *audio.Player
	playlistMgr   *playlist.Manager
	libraryMgr    *LibraryManager
	trackRepo     domain.TrackRepository
	playlistRepo  domain.PlaylistRepository
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		config: config.Get(),
		player: audio.NewPlayer(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Initialize repositories
	database := db.Get()
	a.trackRepo = db.NewTrackRepository(database)
	
	// Initialize managers
	a.playlistMgr = playlist.NewManager(a.playlistRepo)
	a.libraryMgr = NewLibraryManager(a.trackRepo)
	
	// Set up player event listeners
	a.player.AddListener(func(event audio.PlayerEvent, data interface{}) {
		a.handlePlayerEvent(event, data)
	})
	
	logger.Info("WinRamp UI started")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.player != nil {
		a.player.Close()
	}
	logger.Info("WinRamp UI shutdown")
}

// Player Control Methods

// Play starts playback
func (a *App) Play() error {
	return a.player.Play()
}

// Pause pauses playback
func (a *App) Pause() error {
	return a.player.Pause()
}

// Stop stops playback
func (a *App) Stop() error {
	return a.player.Stop()
}

// Next plays the next track
func (a *App) Next() error {
	track := a.playlistMgr.GetNextTrack()
	if track == nil {
		return fmt.Errorf("no next track")
	}
	return a.LoadTrack(track)
}

// Previous plays the previous track
func (a *App) Previous() error {
	track := a.playlistMgr.GetPreviousTrack()
	if track == nil {
		return fmt.Errorf("no previous track")
	}
	return a.LoadTrack(track)
}

// Seek seeks to a position in seconds
func (a *App) Seek(seconds float64) error {
	duration := time.Duration(seconds * float64(time.Second))
	return a.player.Seek(duration)
}

// SetVolume sets the volume (0.0 to 1.0)
func (a *App) SetVolume(volume float64) error {
	return a.player.SetVolume(volume)
}

// GetPlayerState returns the current player state
func (a *App) GetPlayerState() map[string]interface{} {
	state := make(map[string]interface{})
	state["state"] = a.player.GetState().String()
	state["position"] = a.player.GetPosition().Seconds()
	state["duration"] = a.player.GetDuration().Seconds()
	
	if track := a.player.GetCurrentTrack(); track != nil {
		state["track"] = a.trackToMap(track)
	}
	
	return state
}

// LoadTrack loads a track for playback
func (a *App) LoadTrack(track *domain.Track) error {
	if err := a.player.Load(track); err != nil {
		return err
	}
	
	// Set next track for gapless playback
	if next := a.playlistMgr.PeekNextTrack(); next != nil {
		a.player.SetNextTrack(next)
	}
	
	return nil
}

// LoadFile loads a file for playback
func (a *App) LoadFile(path string) error {
	track, err := a.libraryMgr.ImportTrack(path)
	if err != nil {
		return err
	}
	return a.LoadTrack(track)
}

// Playlist Methods

// GetPlaylists returns all playlists
func (a *App) GetPlaylists() []map[string]interface{} {
	playlists := a.playlistMgr.GetAll()
	result := make([]map[string]interface{}, len(playlists))
	
	for i, pl := range playlists {
		result[i] = a.playlistToMap(pl)
	}
	
	return result
}

// GetPlaylist returns a playlist by ID
func (a *App) GetPlaylist(id string) (map[string]interface{}, error) {
	playlist, err := a.playlistMgr.Get(id)
	if err != nil {
		return nil, err
	}
	return a.playlistToMap(playlist), nil
}

// CreatePlaylist creates a new playlist
func (a *App) CreatePlaylist(name string) (map[string]interface{}, error) {
	playlist, err := a.playlistMgr.Create(name)
	if err != nil {
		return nil, err
	}
	return a.playlistToMap(playlist), nil
}

// DeletePlaylist deletes a playlist
func (a *App) DeletePlaylist(id string) error {
	return a.playlistMgr.Delete(id)
}

// AddToPlaylist adds tracks to a playlist
func (a *App) AddToPlaylist(playlistID string, trackIDs []string) error {
	for _, trackID := range trackIDs {
		track, err := a.trackRepo.FindByID(trackID)
		if err != nil {
			logger.Warn("Track not found", logger.String("id", trackID))
			continue
		}
		if err := a.playlistMgr.AddTrack(playlistID, track); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFromPlaylist removes tracks from a playlist
func (a *App) RemoveFromPlaylist(playlistID string, trackIDs []string) error {
	for _, trackID := range trackIDs {
		if err := a.playlistMgr.RemoveTrack(playlistID, trackID); err != nil {
			logger.Warn("Failed to remove track", logger.String("id", trackID), logger.Error(err))
		}
	}
	return nil
}

// Library Methods

// GetLibraryTracks returns all tracks in the library
func (a *App) GetLibraryTracks() []map[string]interface{} {
	tracks, err := a.trackRepo.FindAll()
	if err != nil {
		logger.Error("Failed to get library tracks", logger.Error(err))
		return []map[string]interface{}{}
	}
	
	result := make([]map[string]interface{}, len(tracks))
	for i, track := range tracks {
		result[i] = a.trackToMap(track)
	}
	
	return result
}

// SearchTracks searches for tracks
func (a *App) SearchTracks(query string) []map[string]interface{} {
	tracks, err := a.trackRepo.Search(query)
	if err != nil {
		logger.Error("Failed to search tracks", logger.Error(err))
		return []map[string]interface{}{}
	}
	
	result := make([]map[string]interface{}, len(tracks))
	for i, track := range tracks {
		result[i] = a.trackToMap(track)
	}
	
	return result
}

// ImportFiles imports audio files to the library
func (a *App) ImportFiles(paths []string) (int, error) {
	imported := 0
	for _, path := range paths {
		if _, err := a.libraryMgr.ImportTrack(path); err != nil {
			logger.Warn("Failed to import file", logger.String("path", path), logger.Error(err))
			continue
		}
		imported++
	}
	return imported, nil
}

// ScanFolder scans a folder for audio files
func (a *App) ScanFolder(path string) error {
	return a.libraryMgr.ScanFolder(path, true)
}

// Settings Methods

// GetSettings returns current settings
func (a *App) GetSettings() map[string]interface{} {
	return map[string]interface{}{
		"audio": map[string]interface{}{
			"volume":        a.config.Audio.Volume,
			"crossfade":     a.config.Audio.CrossfadeDuration.Seconds(),
			"replayGain":    a.config.Audio.ReplayGain,
			"gapless":       a.config.Audio.GaplessPlayback,
			"fadeOnPause":   a.config.Audio.FadeOnPause,
		},
		"library": map[string]interface{}{
			"watchFolders": a.config.Library.WatchFolders,
			"autoScan":     a.config.Library.AutoScan,
		},
		"ui": map[string]interface{}{
			"theme":         a.config.App.Theme,
			"windowMode":    a.config.UI.WindowMode,
			"alwaysOnTop":   a.config.UI.AlwaysOnTop,
		},
	}
}

// UpdateSettings updates settings
func (a *App) UpdateSettings(settings map[string]interface{}) error {
	// Update configuration
	if audio, ok := settings["audio"].(map[string]interface{}); ok {
		if volume, ok := audio["volume"].(float64); ok {
			a.config.Audio.Volume = volume
			a.player.SetVolume(volume)
		}
		if crossfade, ok := audio["crossfade"].(float64); ok {
			a.config.Audio.CrossfadeDuration = time.Duration(crossfade * float64(time.Second))
		}
		if replayGain, ok := audio["replayGain"].(bool); ok {
			a.config.Audio.ReplayGain = replayGain
		}
	}
	
	// Save configuration
	return a.config.Save()
}

// Helper methods

func (a *App) handlePlayerEvent(event audio.PlayerEvent, data interface{}) {
	eventData := map[string]interface{}{
		"event": event,
		"data":  data,
	}
	
	switch event {
	case audio.EventStateChanged:
		runtime.EventsEmit(a.ctx, "player:stateChanged", data)
	case audio.EventTrackChanged:
		if track, ok := data.(*domain.Track); ok {
			runtime.EventsEmit(a.ctx, "player:trackChanged", a.trackToMap(track))
		}
	case audio.EventPositionChanged:
		if pos, ok := data.(time.Duration); ok {
			runtime.EventsEmit(a.ctx, "player:positionChanged", pos.Seconds())
		}
	case audio.EventVolumeChanged:
		runtime.EventsEmit(a.ctx, "player:volumeChanged", data)
	case audio.EventTrackFinished:
		runtime.EventsEmit(a.ctx, "player:trackFinished", eventData)
	case audio.EventError:
		runtime.EventsEmit(a.ctx, "player:error", data)
	}
}

func (a *App) trackToMap(track *domain.Track) map[string]interface{} {
	return map[string]interface{}{
		"id":       track.ID,
		"title":    track.GetDisplayTitle(),
		"artist":   track.GetDisplayArtist(),
		"album":    track.Album,
		"duration": track.Duration.Seconds(),
		"path":     track.FilePath,
		"year":     track.Year,
		"genre":    track.Genre,
		"rating":   track.Rating,
	}
}

func (a *App) playlistToMap(playlist *domain.Playlist) map[string]interface{} {
	tracks := make([]map[string]interface{}, len(playlist.Tracks))
	for i, track := range playlist.Tracks {
		tracks[i] = a.trackToMap(track)
	}
	
	return map[string]interface{}{
		"id":          playlist.ID,
		"name":        playlist.Name,
		"description": playlist.Description,
		"trackCount":  playlist.TrackCount,
		"duration":    playlist.Duration.Seconds(),
		"tracks":      tracks,
	}
}

// LibraryManager manages the music library
type LibraryManager struct {
	trackRepo domain.TrackRepository
}

func NewLibraryManager(repo domain.TrackRepository) *LibraryManager {
	return &LibraryManager{
		trackRepo: repo,
	}
}

func (l *LibraryManager) ImportTrack(path string) (*domain.Track, error) {
	// Check if track already exists
	existing, _ := l.trackRepo.FindByPath(path)
	if existing != nil {
		return existing, nil
	}
	
	// Create new track
	track, err := domain.NewTrack(path)
	if err != nil {
		return nil, err
	}
	
	// Extract metadata
	// TODO: Use decoder to extract metadata
	
	// Save to database
	if err := l.trackRepo.Create(track); err != nil {
		return nil, err
	}
	
	return track, nil
}

func (l *LibraryManager) ScanFolder(path string, recursive bool) error {
	// TODO: Implement folder scanning
	return nil
}