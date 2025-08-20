package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidPlaylist = errors.New("invalid playlist")
	ErrTrackNotFound   = errors.New("track not found in playlist")
	ErrInvalidPosition = errors.New("invalid position in playlist")
	ErrEmptyPlaylist   = errors.New("playlist is empty")
)

type PlaylistType string

const (
	PlaylistTypeStatic PlaylistType = "static"
	PlaylistTypeSmart  PlaylistType = "smart"
	PlaylistTypeQueue  PlaylistType = "queue"
	PlaylistTypeRadio  PlaylistType = "radio"
)

type Playlist struct {
	ID          string       `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"not null;index"`
	Description string       `json:"description"`
	Type        PlaylistType `json:"type" gorm:"default:'static'"`
	Tracks      []*Track     `json:"tracks" gorm:"many2many:playlist_tracks;"`
	TrackIDs    []string     `json:"track_ids" gorm:"-"` // For efficient storage
	TrackOrder  string       `json:"track_order" gorm:"type:text"` // Comma-separated track IDs for order
	Rules       *SmartRules  `json:"rules,omitempty" gorm:"embedded"` // For smart playlists
	IsPublic    bool         `json:"is_public" gorm:"default:false"`
	IsFavorite  bool         `json:"is_favorite" gorm:"default:false"`
	ImagePath   string       `json:"image_path"`
	Duration    time.Duration `json:"duration" gorm:"-"`
	TrackCount  int          `json:"track_count" gorm:"-"`
	Version     int          `json:"version" gorm:"default:1"` // For undo/redo
	ParentID    string       `json:"parent_id"`                // For playlist folders
	SortOrder   int          `json:"sort_order"`                // Display order
	CreatedBy   string       `json:"created_by"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CreatedAt   time.Time    `json:"created_at"`
	LastPlayed  *time.Time   `json:"last_played"`
	PlayCount   int          `json:"play_count" gorm:"default:0"`
}

type SmartRules struct {
	Conditions []RuleCondition `json:"conditions" gorm:"type:json"`
	Limit      int             `json:"limit"`
	OrderBy    string          `json:"order_by"`
	OrderDesc  bool            `json:"order_desc"`
}

type RuleCondition struct {
	Field    string      `json:"field"`    // artist, album, genre, year, rating, etc.
	Operator string      `json:"operator"` // equals, contains, greater, less, between
	Value    interface{} `json:"value"`
	AndOr    string      `json:"and_or"` // AND or OR for combining conditions
}

type PlaylistVersion struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	PlaylistID string    `json:"playlist_id" gorm:"index"`
	Version    int       `json:"version"`
	TrackOrder string    `json:"track_order" gorm:"type:text"`
	ChangedBy  string    `json:"changed_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewPlaylist(name string, playlistType PlaylistType) (*Playlist, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidPlaylist)
	}

	now := time.Now()
	return &Playlist{
		ID:         generatePlaylistID(),
		Name:       name,
		Type:       playlistType,
		Tracks:     make([]*Track, 0),
		TrackIDs:   make([]string, 0),
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (p *Playlist) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidPlaylist)
	}

	if p.Type == "" {
		p.Type = PlaylistTypeStatic
	}

	if p.Type == PlaylistTypeSmart && p.Rules == nil {
		return fmt.Errorf("%w: smart playlist requires rules", ErrInvalidPlaylist)
	}

	return nil
}

func (p *Playlist) AddTrack(track *Track) error {
	if track == nil {
		return fmt.Errorf("%w: track is nil", ErrInvalidTrack)
	}

	for _, t := range p.Tracks {
		if t.ID == track.ID {
			return nil // Track already exists
		}
	}

	p.Tracks = append(p.Tracks, track)
	p.TrackIDs = append(p.TrackIDs, track.ID)
	p.updateMetadata()
	p.incrementVersion()
	return nil
}

func (p *Playlist) AddTrackAt(track *Track, position int) error {
	if track == nil {
		return fmt.Errorf("%w: track is nil", ErrInvalidTrack)
	}

	if position < 0 || position > len(p.Tracks) {
		return fmt.Errorf("%w: position %d out of range", ErrInvalidPosition, position)
	}

	// Check if track already exists
	for _, t := range p.Tracks {
		if t.ID == track.ID {
			return nil
		}
	}

	if position == len(p.Tracks) {
		return p.AddTrack(track)
	}

	p.Tracks = append(p.Tracks[:position+1], p.Tracks[position:]...)
	p.Tracks[position] = track
	
	p.TrackIDs = append(p.TrackIDs[:position+1], p.TrackIDs[position:]...)
	p.TrackIDs[position] = track.ID
	
	p.updateMetadata()
	p.incrementVersion()
	return nil
}

func (p *Playlist) RemoveTrack(trackID string) error {
	index := -1
	for i, track := range p.Tracks {
		if track.ID == trackID {
			index = i
			break
		}
	}

	if index == -1 {
		return ErrTrackNotFound
	}

	p.Tracks = append(p.Tracks[:index], p.Tracks[index+1:]...)
	p.TrackIDs = append(p.TrackIDs[:index], p.TrackIDs[index+1:]...)
	p.updateMetadata()
	p.incrementVersion()
	return nil
}

func (p *Playlist) RemoveTrackAt(position int) error {
	if position < 0 || position >= len(p.Tracks) {
		return fmt.Errorf("%w: position %d out of range", ErrInvalidPosition, position)
	}

	p.Tracks = append(p.Tracks[:position], p.Tracks[position+1:]...)
	p.TrackIDs = append(p.TrackIDs[:position], p.TrackIDs[position+1:]...)
	p.updateMetadata()
	p.incrementVersion()
	return nil
}

func (p *Playlist) MoveTrack(fromPos, toPos int) error {
	if fromPos < 0 || fromPos >= len(p.Tracks) {
		return fmt.Errorf("%w: from position %d out of range", ErrInvalidPosition, fromPos)
	}
	if toPos < 0 || toPos >= len(p.Tracks) {
		return fmt.Errorf("%w: to position %d out of range", ErrInvalidPosition, toPos)
	}
	if fromPos == toPos {
		return nil
	}

	track := p.Tracks[fromPos]
	trackID := p.TrackIDs[fromPos]

	// Remove from old position
	p.Tracks = append(p.Tracks[:fromPos], p.Tracks[fromPos+1:]...)
	p.TrackIDs = append(p.TrackIDs[:fromPos], p.TrackIDs[fromPos+1:]...)

	// Insert at new position
	if toPos > fromPos {
		toPos--
	}
	p.Tracks = append(p.Tracks[:toPos+1], p.Tracks[toPos:]...)
	p.Tracks[toPos] = track
	p.TrackIDs = append(p.TrackIDs[:toPos+1], p.TrackIDs[toPos:]...)
	p.TrackIDs[toPos] = trackID

	p.incrementVersion()
	return nil
}

func (p *Playlist) Clear() {
	p.Tracks = make([]*Track, 0)
	p.TrackIDs = make([]string, 0)
	p.updateMetadata()
	p.incrementVersion()
}

func (p *Playlist) GetTrackAt(position int) (*Track, error) {
	if position < 0 || position >= len(p.Tracks) {
		return nil, fmt.Errorf("%w: position %d out of range", ErrInvalidPosition, position)
	}
	return p.Tracks[position], nil
}

func (p *Playlist) GetDuration() time.Duration {
	var total time.Duration
	for _, track := range p.Tracks {
		total += track.Duration
	}
	return total
}

func (p *Playlist) Shuffle() {
	if len(p.Tracks) <= 1 {
		return
	}

	// Fisher-Yates shuffle
	for i := len(p.Tracks) - 1; i > 0; i-- {
		j := randomInt() % (i + 1)
		p.Tracks[i], p.Tracks[j] = p.Tracks[j], p.Tracks[i]
		p.TrackIDs[i], p.TrackIDs[j] = p.TrackIDs[j], p.TrackIDs[i]
	}
	p.incrementVersion()
}

func (p *Playlist) Sort(field string, descending bool) {
	// Implementation would sort tracks based on field
	// This is a placeholder - actual implementation would use sort.Slice
	p.incrementVersion()
}

func (p *Playlist) Clone() *Playlist {
	clone := *p
	clone.ID = generatePlaylistID()
	clone.Name = p.Name + " (Copy)"
	clone.Version = 1
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = time.Now()
	
	// Deep copy tracks
	clone.Tracks = make([]*Track, len(p.Tracks))
	copy(clone.Tracks, p.Tracks)
	
	clone.TrackIDs = make([]string, len(p.TrackIDs))
	copy(clone.TrackIDs, p.TrackIDs)
	
	if p.Rules != nil {
		rules := *p.Rules
		clone.Rules = &rules
	}
	
	return &clone
}

func (p *Playlist) IncrementPlayCount() {
	p.PlayCount++
	now := time.Now()
	p.LastPlayed = &now
	p.UpdatedAt = now
}

func (p *Playlist) updateMetadata() {
	p.TrackCount = len(p.Tracks)
	p.Duration = p.GetDuration()
	p.UpdatedAt = time.Now()
}

func (p *Playlist) incrementVersion() {
	p.Version++
	p.UpdatedAt = time.Now()
}

func generatePlaylistID() string {
	return fmt.Sprintf("playlist_%d_%d", time.Now().UnixNano(), randomInt())
}

type PlaylistRepository interface {
	Create(playlist *Playlist) error
	Update(playlist *Playlist) error
	Delete(id string) error
	FindByID(id string) (*Playlist, error)
	FindByName(name string) (*Playlist, error)
	FindAll() ([]*Playlist, error)
	FindByType(playlistType PlaylistType) ([]*Playlist, error)
	FindFavorites() ([]*Playlist, error)
	GetRecentlyPlayed(limit int) ([]*Playlist, error)
	SaveVersion(playlist *Playlist) error
	GetVersion(playlistID string, version int) (*PlaylistVersion, error)
	Count() (int64, error)
}