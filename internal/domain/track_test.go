package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrack(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid MP3 file",
			filePath: "/music/song.mp3",
			wantErr:  false,
		},
		{
			name:     "Valid FLAC file",
			filePath: "/music/song.flac",
			wantErr:  false,
		},
		{
			name:     "Empty file path",
			filePath: "",
			wantErr:  true,
		},
		{
			name:     "Unsupported format",
			filePath: "/music/document.pdf",
			wantErr:  true,
		},
		{
			name:     "Network path",
			filePath: "\\\\server\\share\\music\\song.mp3",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track, err := NewTrack(tt.filePath)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, track)
			} else {
				require.NoError(t, err)
				require.NotNil(t, track)
				assert.NotEmpty(t, track.ID)
				assert.Equal(t, tt.filePath, track.FilePath)
				assert.NotZero(t, track.DateAdded)
				assert.NotZero(t, track.CreatedAt)
				assert.True(t, track.IsValid)
			}
		})
	}
}

func TestTrack_Validate(t *testing.T) {
	tests := []struct {
		name    string
		track   *Track
		wantErr bool
	}{
		{
			name: "Valid track",
			track: &Track{
				FilePath: "/music/song.mp3",
				Duration: 3 * time.Minute,
				Rating:   3,
				Format:   FormatMP3,
			},
			wantErr: false,
		},
		{
			name: "Empty file path",
			track: &Track{
				Duration: 3 * time.Minute,
				Rating:   3,
			},
			wantErr: true,
		},
		{
			name: "Negative duration",
			track: &Track{
				FilePath: "/music/song.mp3",
				Duration: -1 * time.Second,
				Rating:   3,
			},
			wantErr: true,
		},
		{
			name: "Invalid rating (too low)",
			track: &Track{
				FilePath: "/music/song.mp3",
				Duration: 3 * time.Minute,
				Rating:   -1,
			},
			wantErr: true,
		},
		{
			name: "Invalid rating (too high)",
			track: &Track{
				FilePath: "/music/song.mp3",
				Duration: 3 * time.Minute,
				Rating:   6,
			},
			wantErr: true,
		},
		{
			name: "Auto-detect format",
			track: &Track{
				FilePath: "/music/song.flac",
				Duration: 3 * time.Minute,
				Rating:   3,
				Format:   "", // Should auto-detect
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.track.Validate()
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.track.Format == "" {
					assert.NotEmpty(t, tt.track.Format)
				}
			}
		})
	}
}

func TestTrack_IncrementPlayCount(t *testing.T) {
	track := &Track{
		FilePath:  "/music/song.mp3",
		PlayCount: 5,
		Format:    FormatMP3,
	}

	beforeUpdate := track.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	
	track.IncrementPlayCount()
	
	assert.Equal(t, 6, track.PlayCount)
	assert.NotNil(t, track.LastPlayed)
	assert.True(t, track.UpdatedAt.After(beforeUpdate))
}

func TestTrack_SetRating(t *testing.T) {
	track := &Track{
		FilePath: "/music/song.mp3",
		Rating:   0,
		Format:   FormatMP3,
	}

	// Valid rating
	err := track.SetRating(4)
	assert.NoError(t, err)
	assert.Equal(t, 4, track.Rating)

	// Invalid rating (too low)
	err = track.SetRating(-1)
	assert.Error(t, err)
	assert.Equal(t, 4, track.Rating) // Should remain unchanged

	// Invalid rating (too high)
	err = track.SetRating(6)
	assert.Error(t, err)
	assert.Equal(t, 4, track.Rating) // Should remain unchanged
}

func TestTrack_GetDisplayTitle(t *testing.T) {
	tests := []struct {
		name     string
		track    *Track
		expected string
	}{
		{
			name: "Has title",
			track: &Track{
				FilePath: "/music/song.mp3",
				Title:    "My Song",
			},
			expected: "My Song",
		},
		{
			name: "No title",
			track: &Track{
				FilePath: "/music/amazing_track.mp3",
				Title:    "",
			},
			expected: "amazing_track.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.track.GetDisplayTitle())
		})
	}
}

func TestTrack_GetDisplayArtist(t *testing.T) {
	tests := []struct {
		name     string
		track    *Track
		expected string
	}{
		{
			name: "Has artist",
			track: &Track{
				Artist: "The Beatles",
			},
			expected: "The Beatles",
		},
		{
			name: "No artist but has album artist",
			track: &Track{
				Artist:      "",
				AlbumArtist: "Various Artists",
			},
			expected: "Various Artists",
		},
		{
			name: "No artist info",
			track: &Track{
				Artist:      "",
				AlbumArtist: "",
			},
			expected: "Unknown Artist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.track.GetDisplayArtist())
		})
	}
}

func TestTrack_IsNetworkPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "Local path",
			filePath: "/music/song.mp3",
			expected: false,
		},
		{
			name:     "Windows local path",
			filePath: "C:\\Music\\song.mp3",
			expected: false,
		},
		{
			name:     "UNC path",
			filePath: "\\\\server\\share\\music\\song.mp3",
			expected: true,
		},
		{
			name:     "Unix network path",
			filePath: "//server/share/music/song.mp3",
			expected: true,
		},
		{
			name:     "SMB URL",
			filePath: "smb://server/share/music/song.mp3",
			expected: true,
		},
		{
			name:     "HTTP URL",
			filePath: "http://example.com/song.mp3",
			expected: true,
		},
		{
			name:     "HTTPS URL",
			filePath: "https://example.com/song.mp3",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := &Track{FilePath: tt.filePath}
			assert.Equal(t, tt.expected, track.IsNetworkPath())
		})
	}
}

func TestTrack_Clone(t *testing.T) {
	now := time.Now()
	original := &Track{
		ID:          "track_123",
		FilePath:    "/music/song.mp3",
		Title:       "Original Song",
		Artist:      "Original Artist",
		PlayCount:   10,
		LastPlayed:  &now,
		ReplayGain: &ReplayGain{
			TrackGain: 1.5,
			TrackPeak: 0.95,
		},
	}

	clone := original.Clone()

	// Verify clone is equal but not the same object
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Title, clone.Title)
	assert.Equal(t, original.PlayCount, clone.PlayCount)
	assert.NotSame(t, original, clone)

	// Verify deep copy of pointers
	if original.LastPlayed != nil {
		assert.NotSame(t, original.LastPlayed, clone.LastPlayed)
		assert.Equal(t, *original.LastPlayed, *clone.LastPlayed)
	}

	if original.ReplayGain != nil {
		assert.NotSame(t, original.ReplayGain, clone.ReplayGain)
		assert.Equal(t, *original.ReplayGain, *clone.ReplayGain)
	}

	// Modify clone and ensure original is unchanged
	clone.Title = "Modified Song"
	clone.PlayCount = 20
	assert.NotEqual(t, original.Title, clone.Title)
	assert.NotEqual(t, original.PlayCount, clone.PlayCount)
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filePath string
		expected AudioFormat
	}{
		{"/music/song.mp3", FormatMP3},
		{"/music/song.MP3", FormatMP3},
		{"/music/song.flac", FormatFLAC},
		{"/music/song.ogg", FormatOGG},
		{"/music/song.oga", FormatOGG},
		{"/music/song.wav", FormatWAV},
		{"/music/song.aac", FormatAAC},
		{"/music/song.wma", FormatWMA},
		{"/music/song.m4a", FormatM4A},
		{"/music/song.opus", FormatOPUS},
		{"/music/document.pdf", ""},
		{"/music/noextension", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			assert.Equal(t, tt.expected, detectFormat(tt.filePath))
		})
	}
}

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		filePath string
		expected bool
	}{
		{"/music/song.mp3", true},
		{"/music/song.flac", true},
		{"/music/document.pdf", false},
		{"/music/image.jpg", false},
		{"/music/video.mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsAudioFile(tt.filePath))
		})
	}
}

func TestGetSupportedFormats(t *testing.T) {
	formats := GetSupportedFormats()
	
	assert.NotEmpty(t, formats)
	assert.Contains(t, formats, FormatMP3)
	assert.Contains(t, formats, FormatFLAC)
	assert.Contains(t, formats, FormatOGG)
	assert.Contains(t, formats, FormatWAV)
	assert.Contains(t, formats, FormatAAC)
	assert.Contains(t, formats, FormatWMA)
	assert.Contains(t, formats, FormatM4A)
	assert.Contains(t, formats, FormatOPUS)
}