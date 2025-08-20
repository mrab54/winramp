package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/winramp/winramp/internal/domain"
	"github.com/winramp/winramp/internal/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	instance *Database
	once     sync.Once
)

type Database struct {
	db *gorm.DB
	mu sync.RWMutex
}

type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        string
}

func DefaultConfig() Config {
	dataDir := getDataDir()
	return Config{
		Path:            filepath.Join(dataDir, "winramp.db"),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		LogLevel:        "warn",
	}
}

func Get() *Database {
	once.Do(func() {
		instance = &Database{}
		if err := instance.Initialize(DefaultConfig()); err != nil {
			logger.Fatal("Failed to initialize database", logger.Error(err))
		}
	})
	return instance
}

func Initialize(cfg Config) error {
	return Get().Initialize(cfg)
}

func (d *Database) Initialize(cfg Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger
	var logLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	default:
		logLevel = gormlogger.Warn
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Enable foreign keys for SQLite
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to avoid database locked errors
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	d.db = db

	// Run migrations
	if err := d.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database initialized successfully", logger.String("path", cfg.Path))
	return nil
}

func (d *Database) Migrate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Auto-migrate domain models
	models := []interface{}{
		&domain.Track{},
		&domain.Playlist{},
		&domain.Library{},
		&domain.WatchFolder{},
		&domain.PlaylistVersion{},
		&PlaylistTrack{}, // Junction table for playlist-track many-to-many
	}

	for _, model := range models {
		if err := d.db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Create indexes
	if err := d.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}

func (d *Database) createIndexes() error {
	indexes := []struct {
		Table   string
		Name    string
		Columns []string
	}{
		// Track indexes
		{"tracks", "idx_tracks_artist_album", []string{"artist", "album"}},
		{"tracks", "idx_tracks_album_track", []string{"album", "track_number"}},
		{"tracks", "idx_tracks_genre_year", []string{"genre", "year"}},
		{"tracks", "idx_tracks_date_added", []string{"date_added"}},
		{"tracks", "idx_tracks_last_played", []string{"last_played"}},
		{"tracks", "idx_tracks_play_count", []string{"play_count"}},
		{"tracks", "idx_tracks_rating", []string{"rating"}},
		
		// Playlist indexes
		{"playlists", "idx_playlists_type", []string{"type"}},
		{"playlists", "idx_playlists_created_at", []string{"created_at"}},
		{"playlists", "idx_playlists_last_played", []string{"last_played"}},
		{"playlists", "idx_playlists_is_favorite", []string{"is_favorite"}},
		
		// Library indexes
		{"libraries", "idx_libraries_name", []string{"name"}},
		
		// Watch folder indexes
		{"watch_folders", "idx_watch_folders_library_id", []string{"library_id"}},
		
		// Playlist tracks junction table
		{"playlist_tracks", "idx_playlist_tracks_playlist_id", []string{"playlist_id"}},
		{"playlist_tracks", "idx_playlist_tracks_track_id", []string{"track_id"}},
	}

	for _, idx := range indexes {
		indexName := idx.Name
		if indexName == "" {
			indexName = fmt.Sprintf("idx_%s_%s", idx.Table, idx.Columns[0])
		}
		
		// Check if index exists before creating
		var count int64
		d.db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", indexName).Scan(&count)
		if count > 0 {
			continue // Index already exists
		}
		
		columns := ""
		for i, col := range idx.Columns {
			if i > 0 {
				columns += ", "
			}
			columns += col
		}
		
		sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", indexName, idx.Table, columns)
		if err := d.db.Exec(sql).Error; err != nil {
			logger.Warn("Failed to create index", 
				logger.String("index", indexName),
				logger.Error(err))
		}
	}

	return nil
}

func (d *Database) DB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		sqlDB, err := d.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func (d *Database) Backup(path string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Ensure backup directory exists
	backupDir := filepath.Dir(path)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Perform backup using SQLite backup API
	var result struct{}
	err := d.db.Raw("VACUUM INTO ?", path).Scan(&result).Error
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	logger.Info("Database backed up successfully", logger.String("path", path))
	return nil
}

func (d *Database) Restore(path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if backup file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", path)
	}

	// Close current database
	if d.db != nil {
		sqlDB, err := d.db.DB()
		if err != nil {
			return err
		}
		sqlDB.Close()
	}

	// Get current database path
	var dbPath string
	if d.db != nil {
		sqlDB, _ := d.db.DB()
		// This is a simplified approach - in production you'd store the path
		dbPath = DefaultConfig().Path
		sqlDB.Close()
	}

	// Copy backup file to database path
	backupData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(dbPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	// Reinitialize database
	if err := d.Initialize(DefaultConfig()); err != nil {
		return fmt.Errorf("failed to reinitialize database: %w", err)
	}

	logger.Info("Database restored successfully", logger.String("path", path))
	return nil
}

func (d *Database) Vacuum() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := d.db.Exec("VACUUM").Error; err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	logger.Info("Database vacuumed successfully")
	return nil
}

func (d *Database) GetStats() (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	stats := make(map[string]interface{})

	// Get table counts
	tables := []string{"tracks", "playlists", "libraries", "watch_folders"}
	for _, table := range tables {
		var count int64
		if err := d.db.Table(table).Count(&count).Error; err != nil {
			logger.Warn("Failed to get table count", 
				logger.String("table", table),
				logger.Error(err))
			continue
		}
		stats[table+"_count"] = count
	}

	// Get database size
	sqlDB, err := d.db.DB()
	if err == nil {
		stats["connections"] = sqlDB.Stats()
	}

	// Get database file size
	var dbSize int64
	d.db.Raw("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&dbSize)
	stats["size_bytes"] = dbSize

	return stats, nil
}

func getDataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(appData, "WinRamp")
}

// PlaylistTrack represents the junction table for playlist-track many-to-many relationship
type PlaylistTrack struct {
	PlaylistID string `gorm:"primaryKey"`
	TrackID    string `gorm:"primaryKey"`
	Position   int    `gorm:"not null"`
	AddedAt    time.Time
}