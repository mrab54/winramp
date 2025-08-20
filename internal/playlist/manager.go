package playlist

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/logger"
)

var (
	ErrPlaylistNotFound = errors.New("playlist not found")
	ErrEmptyQueue       = errors.New("queue is empty")
)

// Manager manages playlists and playback queue
type Manager struct {
	playlists      map[string]*domain.Playlist
	currentPlaylist *domain.Playlist
	queue          *Queue
	history        []string // Track IDs
	repo           domain.PlaylistRepository
	mu             sync.RWMutex
}

// NewManager creates a new playlist manager
func NewManager(repo domain.PlaylistRepository) *Manager {
	m := &Manager{
		playlists: make(map[string]*domain.Playlist),
		queue:     NewQueue(),
		history:   make([]string, 0, 100),
		repo:      repo,
	}
	
	// Load playlists from repository if available
	if repo != nil {
		m.loadPlaylists()
	}
	
	return m
}

func (m *Manager) loadPlaylists() {
	playlists, err := m.repo.FindAll()
	if err != nil {
		logger.Error("Failed to load playlists", logger.Error(err))
		return
	}
	
	for _, pl := range playlists {
		m.playlists[pl.ID] = pl
	}
	
	logger.Info("Loaded playlists", logger.Int("count", len(playlists)))
}

// Create creates a new playlist
func (m *Manager) Create(name string) (*domain.Playlist, error) {
	playlist, err := domain.NewPlaylist(name, domain.PlaylistTypeStatic)
	if err != nil {
		return nil, err
	}
	
	m.mu.Lock()
	m.playlists[playlist.ID] = playlist
	m.mu.Unlock()
	
	// Save to repository
	if m.repo != nil {
		if err := m.repo.Create(playlist); err != nil {
			logger.Error("Failed to save playlist", logger.Error(err))
		}
	}
	
	return playlist, nil
}

// Get returns a playlist by ID
func (m *Manager) Get(id string) (*domain.Playlist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	playlist, exists := m.playlists[id]
	if !exists {
		return nil, ErrPlaylistNotFound
	}
	
	return playlist, nil
}

// GetAll returns all playlists
func (m *Manager) GetAll() []*domain.Playlist {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	playlists := make([]*domain.Playlist, 0, len(m.playlists))
	for _, pl := range m.playlists {
		playlists = append(playlists, pl)
	}
	
	return playlists
}

// Update updates a playlist
func (m *Manager) Update(playlist *domain.Playlist) error {
	if playlist == nil {
		return errors.New("playlist is nil")
	}
	
	m.mu.Lock()
	m.playlists[playlist.ID] = playlist
	m.mu.Unlock()
	
	// Save to repository
	if m.repo != nil {
		if err := m.repo.Update(playlist); err != nil {
			return fmt.Errorf("failed to update playlist: %w", err)
		}
	}
	
	return nil
}

// Delete deletes a playlist
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.playlists[id]; !exists {
		return ErrPlaylistNotFound
	}
	
	delete(m.playlists, id)
	
	// Delete from repository
	if m.repo != nil {
		if err := m.repo.Delete(id); err != nil {
			logger.Error("Failed to delete playlist from repository", logger.Error(err))
		}
	}
	
	return nil
}

// AddTrack adds a track to a playlist
func (m *Manager) AddTrack(playlistID string, track *domain.Track) error {
	playlist, err := m.Get(playlistID)
	if err != nil {
		return err
	}
	
	if err := playlist.AddTrack(track); err != nil {
		return err
	}
	
	return m.Update(playlist)
}

// RemoveTrack removes a track from a playlist
func (m *Manager) RemoveTrack(playlistID, trackID string) error {
	playlist, err := m.Get(playlistID)
	if err != nil {
		return err
	}
	
	if err := playlist.RemoveTrack(trackID); err != nil {
		return err
	}
	
	return m.Update(playlist)
}

// SetCurrentPlaylist sets the current playlist
func (m *Manager) SetCurrentPlaylist(id string) error {
	playlist, err := m.Get(id)
	if err != nil {
		return err
	}
	
	m.mu.Lock()
	m.currentPlaylist = playlist
	m.mu.Unlock()
	
	// Clear queue and add playlist tracks
	m.queue.Clear()
	for _, track := range playlist.Tracks {
		m.queue.Add(track)
	}
	
	playlist.IncrementPlayCount()
	m.Update(playlist)
	
	return nil
}

// GetCurrentPlaylist returns the current playlist
func (m *Manager) GetCurrentPlaylist() *domain.Playlist {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentPlaylist
}

// GetNextTrack returns the next track to play
func (m *Manager) GetNextTrack() *domain.Track {
	track := m.queue.Next()
	if track != nil {
		m.addToHistory(track.ID)
	}
	return track
}

// GetPreviousTrack returns the previous track from history
func (m *Manager) GetPreviousTrack() *domain.Track {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.history) < 2 {
		return nil
	}
	
	// Remove current track from history
	m.history = m.history[:len(m.history)-1]
	
	// Get previous track ID
	trackID := m.history[len(m.history)-1]
	
	// Find track in current playlist
	if m.currentPlaylist != nil {
		for _, track := range m.currentPlaylist.Tracks {
			if track.ID == trackID {
				return track
			}
		}
	}
	
	return nil
}

// PeekNextTrack returns the next track without removing it from queue
func (m *Manager) PeekNextTrack() *domain.Track {
	return m.queue.Peek()
}

// GetQueue returns the current queue
func (m *Manager) GetQueue() *Queue {
	return m.queue
}

// AddToQueue adds a track to the queue
func (m *Manager) AddToQueue(track *domain.Track) {
	m.queue.Add(track)
}

// AddToQueueNext adds a track to play next
func (m *Manager) AddToQueueNext(track *domain.Track) {
	m.queue.AddNext(track)
}

// ClearQueue clears the queue
func (m *Manager) ClearQueue() {
	m.queue.Clear()
}

// GetHistory returns the playback history
func (m *Manager) GetHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	history := make([]string, len(m.history))
	copy(history, m.history)
	return history
}

func (m *Manager) addToHistory(trackID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.history = append(m.history, trackID)
	
	// Limit history size
	if len(m.history) > 100 {
		m.history = m.history[1:]
	}
}

// Queue manages the playback queue
type Queue struct {
	tracks   []*domain.Track
	position int
	shuffle  bool
	repeat   RepeatMode
	mu       sync.RWMutex
}

type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatOne
	RepeatAll
)

// NewQueue creates a new queue
func NewQueue() *Queue {
	return &Queue{
		tracks:   make([]*domain.Track, 0),
		position: 0,
		shuffle:  false,
		repeat:   RepeatOff,
	}
}

// Add adds a track to the queue
func (q *Queue) Add(track *domain.Track) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.tracks = append(q.tracks, track)
}

// AddNext adds a track to play next
func (q *Queue) AddNext(track *domain.Track) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.position >= len(q.tracks) {
		q.tracks = append(q.tracks, track)
	} else {
		// Insert after current position
		q.tracks = append(q.tracks[:q.position+1], append([]*domain.Track{track}, q.tracks[q.position+1:]...)...)
	}
}

// Remove removes a track from the queue
func (q *Queue) Remove(index int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if index < 0 || index >= len(q.tracks) {
		return errors.New("index out of range")
	}
	
	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	
	// Adjust position if necessary
	if q.position > index {
		q.position--
	} else if q.position >= len(q.tracks) && len(q.tracks) > 0 {
		q.position = len(q.tracks) - 1
	}
	
	return nil
}

// Next returns the next track in the queue
func (q *Queue) Next() *domain.Track {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if len(q.tracks) == 0 {
		return nil
	}
	
	// Handle repeat one
	if q.repeat == RepeatOne && q.position < len(q.tracks) {
		return q.tracks[q.position]
	}
	
	// Move to next position
	q.position++
	
	// Handle end of queue
	if q.position >= len(q.tracks) {
		if q.repeat == RepeatAll {
			q.position = 0
		} else {
			q.position = len(q.tracks)
			return nil
		}
	}
	
	return q.tracks[q.position]
}

// Peek returns the next track without advancing position
func (q *Queue) Peek() *domain.Track {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	if len(q.tracks) == 0 {
		return nil
	}
	
	nextPos := q.position + 1
	if nextPos >= len(q.tracks) {
		if q.repeat == RepeatAll {
			nextPos = 0
		} else {
			return nil
		}
	}
	
	return q.tracks[nextPos]
}

// Previous returns the previous track in the queue
func (q *Queue) Previous() *domain.Track {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if len(q.tracks) == 0 {
		return nil
	}
	
	q.position--
	if q.position < 0 {
		if q.repeat == RepeatAll {
			q.position = len(q.tracks) - 1
		} else {
			q.position = 0
		}
	}
	
	return q.tracks[q.position]
}

// Clear clears the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.tracks = make([]*domain.Track, 0)
	q.position = 0
}

// GetTracks returns all tracks in the queue
func (q *Queue) GetTracks() []*domain.Track {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	tracks := make([]*domain.Track, len(q.tracks))
	copy(tracks, q.tracks)
	return tracks
}

// SetShuffle enables or disables shuffle
func (q *Queue) SetShuffle(shuffle bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.shuffle = shuffle
	
	if shuffle && len(q.tracks) > 1 {
		// Shuffle tracks after current position
		if q.position < len(q.tracks)-1 {
			remaining := q.tracks[q.position+1:]
			shuffleTracks(remaining)
			q.tracks = append(q.tracks[:q.position+1], remaining...)
		}
	}
}

// SetRepeat sets the repeat mode
func (q *Queue) SetRepeat(mode RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.repeat = mode
}

// GetPosition returns the current queue position
func (q *Queue) GetPosition() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.position
}

// GetLength returns the queue length
func (q *Queue) GetLength() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.tracks)
}

// IsEmpty returns true if the queue is empty
func (q *Queue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.tracks) == 0
}

func shuffleTracks(tracks []*domain.Track) {
	for i := len(tracks) - 1; i > 0; i-- {
		j := int(time.Now().UnixNano()) % (i + 1)
		tracks[i], tracks[j] = tracks[j], tracks[i]
	}
}