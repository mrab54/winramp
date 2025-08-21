package db

import (
	"fmt"
	"strings"

	"github.com/winramp/winramp/internal/domain"
	"gorm.io/gorm"
)

type TrackRepository struct {
	db *gorm.DB
}

func NewTrackRepository(database *Database) domain.TrackRepository {
	return &TrackRepository{
		db: database.DB(),
	}
}

func (r *TrackRepository) Create(track *domain.Track) error {
	if err := track.Validate(); err != nil {
		return err
	}
	
	if err := r.db.Create(track).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("failed to create track: %w", err)
	}
	
	return nil
}

func (r *TrackRepository) Update(track *domain.Track) error {
	if err := track.Validate(); err != nil {
		return err
	}
	
	result := r.db.Model(track).Updates(track)
	if result.Error != nil {
		return fmt.Errorf("failed to update track: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrTrackNotFound
	}
	
	return nil
}

func (r *TrackRepository) Delete(id string) error {
	result := r.db.Delete(&domain.Track{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete track: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrTrackNotFound
	}
	
	return nil
}

func (r *TrackRepository) FindByID(id string) (*domain.Track, error) {
	var track domain.Track
	if err := r.db.First(&track, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("failed to find track: %w", err)
	}
	
	return &track, nil
}

func (r *TrackRepository) FindByPath(path string) (*domain.Track, error) {
	var track domain.Track
	if err := r.db.First(&track, "file_path = ?", path).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("failed to find track by path: %w", err)
	}
	
	return &track, nil
}

func (r *TrackRepository) FindAll() ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find all tracks: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) FindByArtist(artist string) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("artist = ? OR album_artist = ?", artist, artist).
		Order("album, disc_number, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by artist: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) FindByAlbum(album string) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("album = ?", album).
		Order("disc_number, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by album: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) FindByGenre(genre string) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("genre = ?", genre).
		Order("artist, album, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by genre: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) Search(query string) ([]*domain.Track, error) {
	var tracks []*domain.Track
	
	// Input validation
	query = strings.TrimSpace(query)
	if query == "" {
		return tracks, nil
	}
	
	// Limit query length to prevent DoS
	const maxQueryLength = 100
	if len(query) > maxQueryLength {
		query = query[:maxQueryLength]
	}
	
	// Remove any SQL meta-characters for extra safety
	// Even though GORM parameterizes, this adds defense in depth
	query = sanitizeSearchQuery(query)
	
	// Build search query with wildcards
	searchPattern := "%" + strings.ToLower(query) + "%"
	
	// Use parameterized query through GORM (already safe)
	if err := r.db.Where(
		"LOWER(title) LIKE ? OR LOWER(artist) LIKE ? OR LOWER(album) LIKE ? OR LOWER(genre) LIKE ?",
		searchPattern, searchPattern, searchPattern, searchPattern,
	).Limit(1000).Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	
	return tracks, nil
}

// sanitizeSearchQuery removes potentially dangerous characters from search queries
func sanitizeSearchQuery(query string) string {
	// Remove SQL comment markers and other dangerous patterns
	replacer := strings.NewReplacer(
		"--", "",
		"/*", "",
		"*/", "",
		";", "",
		"\\", "",
		"\x00", "", // null bytes
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	return replacer.Replace(query)
}

func (r *TrackRepository) GetRecentlyPlayed(limit int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("last_played IS NOT NULL").
		Order("last_played DESC").
		Limit(limit).
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to get recently played tracks: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) GetMostPlayed(limit int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("play_count > 0").
		Order("play_count DESC").
		Limit(limit).
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to get most played tracks: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) GetRecentlyAdded(limit int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Order("date_added DESC").
		Limit(limit).
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to get recently added tracks: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&domain.Track{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count tracks: %w", err)
	}
	
	return count, nil
}

// Additional repository methods

func (r *TrackRepository) FindByYear(year int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("year = ?", year).
		Order("artist, album, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by year: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) FindByRating(rating int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("rating = ?", rating).
		Order("artist, album, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by rating: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) FindByFormat(format domain.AudioFormat) ([]*domain.Track, error) {
	var tracks []*domain.Track
	if err := r.db.Where("format = ?", format).
		Order("artist, album, track_number").
		Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tracks by format: %w", err)
	}
	
	return tracks, nil
}

func (r *TrackRepository) UpdatePlayCount(id string) error {
	return r.db.Model(&domain.Track{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"play_count":  gorm.Expr("play_count + ?", 1),
			"last_played": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *TrackRepository) BatchCreate(tracks []*domain.Track) error {
	if len(tracks) == 0 {
		return nil
	}
	
	// Validate all tracks first
	for _, track := range tracks {
		if err := track.Validate(); err != nil {
			return fmt.Errorf("validation failed for track %s: %w", track.FilePath, err)
		}
	}
	
	// Create in batches of 100
	batchSize := 100
	for i := 0; i < len(tracks); i += batchSize {
		end := i + batchSize
		if end > len(tracks) {
			end = len(tracks)
		}
		
		if err := r.db.CreateInBatches(tracks[i:end], batchSize).Error; err != nil {
			return fmt.Errorf("failed to batch create tracks: %w", err)
		}
	}
	
	return nil
}

func (r *TrackRepository) DeleteByPath(path string) error {
	result := r.db.Delete(&domain.Track{}, "file_path = ?", path)
	if result.Error != nil {
		return fmt.Errorf("failed to delete track by path: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return domain.ErrTrackNotFound
	}
	
	return nil
}

func (r *TrackRepository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Total tracks
	var totalTracks int64
	r.db.Model(&domain.Track{}).Count(&totalTracks)
	stats["total_tracks"] = totalTracks
	
	// Unique artists
	var uniqueArtists int64
	r.db.Model(&domain.Track{}).Distinct("artist").Count(&uniqueArtists)
	stats["unique_artists"] = uniqueArtists
	
	// Unique albums
	var uniqueAlbums int64
	r.db.Model(&domain.Track{}).Distinct("album").Count(&uniqueAlbums)
	stats["unique_albums"] = uniqueAlbums
	
	// Unique genres
	var uniqueGenres int64
	r.db.Model(&domain.Track{}).Distinct("genre").Count(&uniqueGenres)
	stats["unique_genres"] = uniqueGenres
	
	// Total duration
	var totalDuration int64
	r.db.Model(&domain.Track{}).Select("SUM(duration)").Scan(&totalDuration)
	stats["total_duration"] = totalDuration
	
	// Total file size
	var totalSize int64
	r.db.Model(&domain.Track{}).Select("SUM(file_size)").Scan(&totalSize)
	stats["total_file_size"] = totalSize
	
	// Average rating
	var avgRating float64
	r.db.Model(&domain.Track{}).Where("rating > 0").Select("AVG(rating)").Scan(&avgRating)
	stats["average_rating"] = avgRating
	
	// Most played track
	var mostPlayed domain.Track
	r.db.Order("play_count DESC").First(&mostPlayed)
	if mostPlayed.ID != "" {
		stats["most_played_track"] = mostPlayed.GetDisplayTitle()
		stats["most_played_count"] = mostPlayed.PlayCount
	}
	
	return stats, nil
}