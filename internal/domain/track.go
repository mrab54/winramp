package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidTrack     = errors.New("invalid track")
	ErrInvalidDuration  = errors.New("invalid duration")
	ErrInvalidFilePath  = errors.New("invalid file path")
	ErrUnsupportedFormat = errors.New("unsupported audio format")
)

type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatFLAC AudioFormat = "flac"
	FormatOGG  AudioFormat = "ogg"
	FormatWAV  AudioFormat = "wav"
	FormatAAC  AudioFormat = "aac"
	FormatWMA  AudioFormat = "wma"
	FormatM4A  AudioFormat = "m4a"
	FormatOPUS AudioFormat = "opus"
)

type Track struct {
	ID           string        `json:"id" gorm:"primaryKey"`
	FilePath     string        `json:"file_path" gorm:"uniqueIndex;not null"`
	Title        string        `json:"title"`
	Artist       string        `json:"artist" gorm:"index"`
	Album        string        `json:"album" gorm:"index"`
	AlbumArtist  string        `json:"album_artist"`
	Genre        string        `json:"genre" gorm:"index"`
	Year         int           `json:"year" gorm:"index"`
	TrackNumber  int           `json:"track_number"`
	DiscNumber   int           `json:"disc_number"`
	Duration     time.Duration `json:"duration"`
	Bitrate      int           `json:"bitrate"`
	SampleRate   int           `json:"sample_rate"`
	Channels     int           `json:"channels"`
	Format       AudioFormat   `json:"format"`
	FileSize     int64         `json:"file_size"`
	DateAdded    time.Time     `json:"date_added" gorm:"index"`
	LastPlayed   *time.Time    `json:"last_played"`
	PlayCount    int           `json:"play_count" gorm:"default:0"`
	Rating       int           `json:"rating" gorm:"default:0"` // 0-5 stars
	BPM          int           `json:"bpm"`
	Comment      string        `json:"comment"`
	Composer     string        `json:"composer"`
	Publisher    string        `json:"publisher"`
	Lyrics       string        `json:"lyrics" gorm:"type:text"`
	AlbumArtPath string        `json:"album_art_path"`
	ReplayGain   *ReplayGain   `json:"replay_gain" gorm:"embedded"`
	Fingerprint  string        `json:"fingerprint"` // Acoustic fingerprint for duplicate detection
	Checksum     string        `json:"checksum"`    // File checksum for integrity
	IsValid      bool          `json:"is_valid" gorm:"default:true"`
	Error        string        `json:"error,omitempty"`
	UpdatedAt    time.Time     `json:"updated_at"`
	CreatedAt    time.Time     `json:"created_at"`
}

type ReplayGain struct {
	TrackGain float64 `json:"track_gain"`
	TrackPeak float64 `json:"track_peak"`
	AlbumGain float64 `json:"album_gain"`
	AlbumPeak float64 `json:"album_peak"`
}

func NewTrack(filePath string) (*Track, error) {
	if filePath == "" {
		return nil, ErrInvalidFilePath
	}

	format := detectFormat(filePath)
	if format == "" {
		return nil, ErrUnsupportedFormat
	}

	now := time.Now()
	return &Track{
		ID:        generateTrackID(),
		FilePath:  filepath.Clean(filePath),
		Format:    format,
		DateAdded: now,
		CreatedAt: now,
		UpdatedAt: now,
		IsValid:   true,
		Channels:  2, // Default to stereo
	}, nil
}

func (t *Track) Validate() error {
	if t.FilePath == "" {
		return fmt.Errorf("%w: file path is required", ErrInvalidTrack)
	}

	if t.Duration < 0 {
		return fmt.Errorf("%w: duration cannot be negative", ErrInvalidDuration)
	}

	if t.Rating < 0 || t.Rating > 5 {
		return fmt.Errorf("%w: rating must be between 0 and 5", ErrInvalidTrack)
	}

	if t.Format == "" {
		t.Format = detectFormat(t.FilePath)
		if t.Format == "" {
			return ErrUnsupportedFormat
		}
	}

	return nil
}

func (t *Track) IncrementPlayCount() {
	t.PlayCount++
	now := time.Now()
	t.LastPlayed = &now
	t.UpdatedAt = now
}

func (t *Track) SetRating(rating int) error {
	if rating < 0 || rating > 5 {
		return fmt.Errorf("%w: rating must be between 0 and 5", ErrInvalidTrack)
	}
	t.Rating = rating
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Track) GetDisplayTitle() string {
	if t.Title != "" {
		return t.Title
	}
	return filepath.Base(t.FilePath)
}

func (t *Track) GetDisplayArtist() string {
	if t.Artist != "" {
		return t.Artist
	}
	if t.AlbumArtist != "" {
		return t.AlbumArtist
	}
	return "Unknown Artist"
}

func (t *Track) GetSortKey() string {
	artist := strings.ToLower(t.GetDisplayArtist())
	album := strings.ToLower(t.Album)
	track := fmt.Sprintf("%03d", t.TrackNumber)
	return fmt.Sprintf("%s-%s-%s-%s", artist, album, track, t.ID)
}

func (t *Track) IsNetworkPath() bool {
	return strings.HasPrefix(t.FilePath, "\\\\") || 
		   strings.HasPrefix(t.FilePath, "//") ||
		   strings.HasPrefix(t.FilePath, "smb://") ||
		   strings.HasPrefix(t.FilePath, "http://") ||
		   strings.HasPrefix(t.FilePath, "https://")
}

func (t *Track) Clone() *Track {
	clone := *t
	if t.LastPlayed != nil {
		lastPlayed := *t.LastPlayed
		clone.LastPlayed = &lastPlayed
	}
	if t.ReplayGain != nil {
		replayGain := *t.ReplayGain
		clone.ReplayGain = &replayGain
	}
	return &clone
}

func detectFormat(filePath string) AudioFormat {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	switch ext {
	case "mp3":
		return FormatMP3
	case "flac":
		return FormatFLAC
	case "ogg", "oga":
		return FormatOGG
	case "wav":
		return FormatWAV
	case "aac":
		return FormatAAC
	case "wma":
		return FormatWMA
	case "m4a":
		return FormatM4A
	case "opus":
		return FormatOPUS
	default:
		return ""
	}
}

func generateTrackID() string {
	return fmt.Sprintf("track_%d_%d", time.Now().UnixNano(), randomInt())
}

func randomInt() int {
	return int(time.Now().UnixNano() % 1000000)
}

func IsAudioFile(filePath string) bool {
	return detectFormat(filePath) != ""
}

func GetSupportedFormats() []AudioFormat {
	return []AudioFormat{
		FormatMP3,
		FormatFLAC,
		FormatOGG,
		FormatWAV,
		FormatAAC,
		FormatWMA,
		FormatM4A,
		FormatOPUS,
	}
}

type TrackRepository interface {
	Create(track *Track) error
	Update(track *Track) error
	Delete(id string) error
	FindByID(id string) (*Track, error)
	FindByPath(path string) (*Track, error)
	FindAll() ([]*Track, error)
	FindByArtist(artist string) ([]*Track, error)
	FindByAlbum(album string) ([]*Track, error)
	FindByGenre(genre string) ([]*Track, error)
	Search(query string) ([]*Track, error)
	GetRecentlyPlayed(limit int) ([]*Track, error)
	GetMostPlayed(limit int) ([]*Track, error)
	GetRecentlyAdded(limit int) ([]*Track, error)
	Count() (int64, error)
}