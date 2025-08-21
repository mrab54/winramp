package library

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
	"github.com/winramp/winramp/internal/audio/decoder"
	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/logger"
)

// ScanResult represents the result of a scan operation
type ScanResult struct {
	TotalFiles      int
	ScannedFiles    int
	ImportedTracks  int
	FailedFiles     int
	SkippedFiles    int
	Duration        time.Duration
	Errors          []error
}

// Scanner scans directories for audio files
type Scanner struct {
	trackRepo     domain.TrackRepository
	libraryRepo   domain.LibraryRepository
	library       *domain.Library
	
	// Scan state
	isScanning    bool
	cancelFunc    context.CancelFunc
	progress      float64
	currentFile   string
	
	// Configuration
	recursive     bool
	followSymlinks bool
	skipDuplicates bool
	extractMetadata bool
	minDuration   time.Duration
	maxDuration   time.Duration
	filePatterns  []string
	excludePatterns []string
	
	// Concurrency
	workerCount   int
	fileChan      chan string
	resultChan    chan *domain.Track
	errorChan     chan error
	
	mu            sync.RWMutex
	wg            sync.WaitGroup
}

// NewScanner creates a new library scanner
func NewScanner(trackRepo domain.TrackRepository, libraryRepo domain.LibraryRepository) *Scanner {
	return &Scanner{
		trackRepo:       trackRepo,
		libraryRepo:     libraryRepo,
		recursive:       true,
		followSymlinks:  false,
		skipDuplicates:  true,
		extractMetadata: true,
		minDuration:     10 * time.Second,
		maxDuration:     10 * time.Hour,
		workerCount:     4,
		filePatterns:    []string{"*.mp3", "*.flac", "*.ogg", "*.wav", "*.aac", "*.wma", "*.m4a"},
		excludePatterns: []string{"*.tmp", "*.temp", "*.partial"},
	}
}

// ScanFolder scans a folder for audio files
func (s *Scanner) ScanFolder(ctx context.Context, path string) (*ScanResult, error) {
	s.mu.Lock()
	if s.isScanning {
		s.mu.Unlock()
		return nil, fmt.Errorf("scan already in progress")
	}
	s.isScanning = true
	s.progress = 0
	s.mu.Unlock()
	
	defer func() {
		s.mu.Lock()
		s.isScanning = false
		s.progress = 100
		s.mu.Unlock()
	}()
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	defer cancel()
	
	startTime := time.Now()
	result := &ScanResult{
		Errors: make([]error, 0),
	}
	
	// Get library
	library, err := s.getOrCreateLibrary()
	if err != nil {
		return nil, fmt.Errorf("failed to get library: %w", err)
	}
	s.library = library
	
	// Mark scan start
	s.library.StartScan()
	if s.libraryRepo != nil {
		s.libraryRepo.Update(s.library)
	}
	
	// Initialize channels
	s.fileChan = make(chan string, 100)
	s.resultChan = make(chan *domain.Track, 100)
	s.errorChan = make(chan error, 100)
	
	// Start workers
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.scanWorker(ctx)
	}
	
	// Start result processor
	s.wg.Add(1)
	go s.processResults(ctx, result)
	
	// Walk directory
	logger.Info("Starting scan", logger.String("path", path))
	
	err = s.walkDirectory(ctx, path)
	if err != nil && err != context.Canceled {
		result.Errors = append(result.Errors, err)
	}
	
	// Close file channel and wait for workers
	close(s.fileChan)
	s.wg.Wait()
	
	// Close result channels
	close(s.resultChan)
	close(s.errorChan)
	
	// Mark scan complete
	s.library.StopScan()
	if s.libraryRepo != nil {
		s.libraryRepo.Update(s.library)
	}
	
	result.Duration = time.Since(startTime)
	
	logger.Info("Scan completed",
		logger.Int("total_files", result.TotalFiles),
		logger.Int("imported", result.ImportedTracks),
		logger.Int("failed", result.FailedFiles),
		logger.Duration("duration", result.Duration),
	)
	
	return result, nil
}

func (s *Scanner) walkDirectory(ctx context.Context, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		
		if err != nil {
			logger.Warn("Error accessing path", logger.String("path", path), logger.Error(err))
			return nil // Continue walking
		}
		
		// Skip directories if not recursive
		if d.IsDir() && path != root && !s.recursive {
			return fs.SkipDir
		}
		
		// Skip symlinks if configured
		if !s.followSymlinks && d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		
		// Check if file matches patterns
		if !d.IsDir() && s.matchesPattern(path) && !s.isExcluded(path) {
			select {
			case <-ctx.Done():
				return context.Canceled
			case s.fileChan <- path:
				s.mu.Lock()
				s.currentFile = path
				s.mu.Unlock()
			}
		}
		
		return nil
	})
}

func (s *Scanner) scanWorker(ctx context.Context) {
	defer s.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case path, ok := <-s.fileChan:
			if !ok {
				return
			}
			
			track, err := s.scanFile(ctx, path)
			if err != nil {
				select {
				case s.errorChan <- fmt.Errorf("%s: %w", path, err):
				case <-ctx.Done():
					return
				}
				continue
			}
			
			if track != nil {
				select {
				case s.resultChan <- track:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (s *Scanner) scanFile(ctx context.Context, path string) (*domain.Track, error) {
	// Check if file already exists in database
	if s.skipDuplicates {
		existing, _ := s.trackRepo.FindByPath(path)
		if existing != nil {
			return nil, nil // Skip duplicate
		}
	}
	
	// Create track
	track, err := domain.NewTrack(path)
	if err != nil {
		return nil, err
	}
	
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	track.FileSize = info.Size()
	
	// Extract metadata if enabled
	if s.extractMetadata {
		if err := s.extractMetadata(track); err != nil {
			logger.Warn("Failed to extract metadata", 
				logger.String("path", path),
				logger.Error(err))
		}
	}
	
	// Validate duration
	if s.minDuration > 0 && track.Duration < s.minDuration {
		return nil, fmt.Errorf("track too short: %v", track.Duration)
	}
	if s.maxDuration > 0 && track.Duration > s.maxDuration {
		return nil, fmt.Errorf("track too long: %v", track.Duration)
	}
	
	return track, nil
}

func (s *Scanner) extractMetadata(track *domain.Track) error {
	file, err := os.Open(track.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Try to extract tags
	m, err := tag.ReadFrom(file)
	if err == nil {
		track.Title = m.Title()
		track.Artist = m.Artist()
		track.Album = m.Album()
		track.AlbumArtist = m.AlbumArtist()
		track.Genre = m.Genre()
		track.Year = m.Year()
		track.Comment = m.Comment()
		
		if trackNum, _ := m.Track(); trackNum > 0 {
			track.TrackNumber = trackNum
		}
		if discNum, _ := m.Disc(); discNum > 0 {
			track.DiscNumber = discNum
		}
		
		// Extract album art
		if pic := m.Picture(); pic != nil && len(pic.Data) > 0 {
			// Save album art to cache
			artPath := s.saveAlbumArt(track, pic.Data, pic.Ext)
			if artPath != "" {
				track.AlbumArtPath = artPath
			}
		}
	}
	
	// Try to get duration from decoder
	file.Seek(0, 0)
	if dec, err := decoder.CreateDecoderForFile(track.FilePath); err == nil {
		defer dec.Close()
		track.Duration = dec.Duration()
		
		format := dec.Format()
		track.SampleRate = format.SampleRate
		track.Channels = format.Channels
		track.BitDepth = format.BitDepth
		
		// Calculate bitrate if not set
		if track.Bitrate == 0 && track.Duration > 0 {
			track.Bitrate = int((track.FileSize * 8) / int64(track.Duration.Seconds()))
		}
	}
	
	return nil
}

func (s *Scanner) saveAlbumArt(track *domain.Track, data []byte, ext string) string {
	// Create album art cache directory with secure permissions
	cacheDir := filepath.Join(os.TempDir(), "winramp", "albumart")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		logger.Warn("Failed to create album art directory", logger.Error(err))
		return ""
	}
	
	// Sanitize inputs BEFORE using them
	safeArtist := sanitizeFilename(track.Artist)
	safeAlbum := sanitizeFilename(track.Album)
	safeExt := sanitizeFilename(ext)
	
	// Additional validation for extension
	if safeExt == "" || len(safeExt) > 5 {
		safeExt = "jpg"
	}
	
	// Generate filename with sanitized inputs
	filename := fmt.Sprintf("%s_%s.%s", safeArtist, safeAlbum, safeExt)
	
	// Construct path and validate it's within cache directory
	path := filepath.Join(cacheDir, filename)
	cleanedPath := filepath.Clean(path)
	
	// Verify the final path is within our cache directory
	relPath, err := filepath.Rel(cacheDir, cleanedPath)
	if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		logger.Warn("Invalid path detected: potential traversal attempt",
			logger.String("path", cleanedPath),
			logger.String("cacheDir", cacheDir))
		return ""
	}
	
	// Save file with secure permissions
	if err := os.WriteFile(cleanedPath, data, 0600); err != nil {
		logger.Warn("Failed to save album art", logger.Error(err))
		return ""
	}
	
	return cleanedPath
}

func (s *Scanner) processResults(ctx context.Context, result *ScanResult) {
	defer s.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
			
		case track, ok := <-s.resultChan:
			if !ok {
				return
			}
			
			result.ScannedFiles++
			
			// Save to database
			if err := s.trackRepo.Create(track); err != nil {
				result.FailedFiles++
				result.Errors = append(result.Errors, err)
				logger.Warn("Failed to save track", 
					logger.String("path", track.FilePath),
					logger.Error(err))
			} else {
				result.ImportedTracks++
				
				// Add to library
				if s.library != nil {
					s.library.AddTrack(track)
				}
			}
			
			// Update progress
			s.updateProgress(result)
			
		case err := <-s.errorChan:
			if err != nil {
				result.FailedFiles++
				result.Errors = append(result.Errors, err)
			}
		}
	}
}

func (s *Scanner) updateProgress(result *ScanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if result.TotalFiles > 0 {
		s.progress = float64(result.ScannedFiles) / float64(result.TotalFiles) * 100
	}
	
	if s.library != nil {
		s.library.UpdateScanProgress(s.progress)
	}
}

func (s *Scanner) matchesPattern(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	for _, pattern := range s.filePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), name); matched {
			return true
		}
	}
	return false
}

func (s *Scanner) isExcluded(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	for _, pattern := range s.excludePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), name); matched {
			return true
		}
	}
	return false
}

func (s *Scanner) getOrCreateLibrary() (*domain.Library, error) {
	if s.libraryRepo == nil {
		return domain.NewLibrary("Default")
	}
	
	library, err := s.libraryRepo.GetDefault()
	if err != nil {
		// Create default library
		library, err = domain.NewLibrary("Default")
		if err != nil {
			return nil, err
		}
		
		if err := s.libraryRepo.Create(library); err != nil {
			return nil, err
		}
	}
	
	return library, nil
}

// Cancel cancels the current scan
func (s *Scanner) Cancel() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

// IsScanning returns whether a scan is in progress
func (s *Scanner) IsScanning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isScanning
}

// GetProgress returns the current scan progress (0-100)
func (s *Scanner) GetProgress() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.progress
}

// GetCurrentFile returns the currently scanning file
func (s *Scanner) GetCurrentFile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentFile
}

func sanitizeFilename(s string) string {
	// Remove invalid filename characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := s
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}