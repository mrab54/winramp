package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
)

var (
	instance *Config
	once     sync.Once
)

type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Audio      AudioConfig      `mapstructure:"audio"`
	Library    LibraryConfig    `mapstructure:"library"`
	UI         UIConfig         `mapstructure:"ui"`
	Network    NetworkConfig    `mapstructure:"network"`
	Shortcuts  ShortcutsConfig  `mapstructure:"shortcuts"`
	Advanced   AdvancedConfig   `mapstructure:"advanced"`
	v          *viper.Viper
	mu         sync.RWMutex
}

type AppConfig struct {
	Name            string `mapstructure:"name"`
	Version         string `mapstructure:"version"`
	DataDir         string `mapstructure:"data_dir"`
	LogDir          string `mapstructure:"log_dir"`
	CacheDir        string `mapstructure:"cache_dir"`
	AutoStart       bool   `mapstructure:"auto_start"`
	MinimizeToTray  bool   `mapstructure:"minimize_to_tray"`
	CheckForUpdates bool   `mapstructure:"check_for_updates"`
	Language        string `mapstructure:"language"`
	Theme           string `mapstructure:"theme"`
}

type AudioConfig struct {
	OutputDevice      string        `mapstructure:"output_device"`
	OutputMode        string        `mapstructure:"output_mode"` // WASAPI, DirectSound
	ExclusiveMode     bool          `mapstructure:"exclusive_mode"`
	BufferSize        int           `mapstructure:"buffer_size"`
	SampleRate        int           `mapstructure:"sample_rate"`
	BitDepth          int           `mapstructure:"bit_depth"`
	Volume            float64       `mapstructure:"volume"`
	CrossfadeDuration time.Duration `mapstructure:"crossfade_duration"`
	ReplayGain        bool          `mapstructure:"replay_gain"`
	ReplayGainMode    string        `mapstructure:"replay_gain_mode"` // track, album
	PreAmp            float64       `mapstructure:"preamp"`
	Equalizer         EqualizerConfig `mapstructure:"equalizer"`
	GaplessPlayback   bool          `mapstructure:"gapless_playback"`
	FadeOnPause       bool          `mapstructure:"fade_on_pause"`
	FadeDuration      time.Duration `mapstructure:"fade_duration"`
}

type EqualizerConfig struct {
	Enabled bool      `mapstructure:"enabled"`
	Preset  string    `mapstructure:"preset"`
	Bands   [10]float64 `mapstructure:"bands"` // -12 to +12 dB
}

type LibraryConfig struct {
	WatchFolders      []string      `mapstructure:"watch_folders"`
	AutoScan          bool          `mapstructure:"auto_scan"`
	ScanInterval      time.Duration `mapstructure:"scan_interval"`
	ExtractMetadata   bool          `mapstructure:"extract_metadata"`
	ExtractAlbumArt   bool          `mapstructure:"extract_album_art"`
	AlbumArtMaxSize   int           `mapstructure:"album_art_max_size"`
	SkipDuplicates    bool          `mapstructure:"skip_duplicates"`
	MinTrackDuration  time.Duration `mapstructure:"min_track_duration"`
	MaxTrackDuration  time.Duration `mapstructure:"max_track_duration"`
	FilePatterns      []string      `mapstructure:"file_patterns"`
	ExcludePatterns   []string      `mapstructure:"exclude_patterns"`
	DatabasePath      string        `mapstructure:"database_path"`
	BackupDatabase    bool          `mapstructure:"backup_database"`
	BackupInterval    time.Duration `mapstructure:"backup_interval"`
}

type UIConfig struct {
	WindowMode       string   `mapstructure:"window_mode"` // classic, modern, mini
	Skin             string   `mapstructure:"skin"`
	ShowPlaylist     bool     `mapstructure:"show_playlist"`
	ShowEqualizer    bool     `mapstructure:"show_equalizer"`
	ShowLibrary      bool     `mapstructure:"show_library"`
	AlwaysOnTop      bool     `mapstructure:"always_on_top"`
	SnapToEdges      bool     `mapstructure:"snap_to_edges"`
	Transparency     float64  `mapstructure:"transparency"`
	FontSize         int      `mapstructure:"font_size"`
	ShowNotifications bool    `mapstructure:"show_notifications"`
	AnimationSpeed   float64  `mapstructure:"animation_speed"`
	DoubleClickAction string  `mapstructure:"double_click_action"` // play, enqueue, info
	ColumnLayout     []string `mapstructure:"column_layout"`
	WindowPositions  map[string]WindowPosition `mapstructure:"window_positions"`
}

type WindowPosition struct {
	X      int `mapstructure:"x"`
	Y      int `mapstructure:"y"`
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
}

type NetworkConfig struct {
	EnableSharing     bool          `mapstructure:"enable_sharing"`
	EnableStreaming   bool          `mapstructure:"enable_streaming"`
	StreamingPort     int           `mapstructure:"streaming_port"`
	BufferSize        int           `mapstructure:"buffer_size"`
	Timeout           time.Duration `mapstructure:"timeout"`
	MaxConnections    int           `mapstructure:"max_connections"`
	ProxyEnabled      bool          `mapstructure:"proxy_enabled"`
	ProxyAddress      string        `mapstructure:"proxy_address"`
	CacheEnabled      bool          `mapstructure:"cache_enabled"`
	CacheSize         int64         `mapstructure:"cache_size"` // in MB
	CachePath         string        `mapstructure:"cache_path"`
}

type ShortcutsConfig struct {
	Global   map[string]string `mapstructure:"global"`
	Player   map[string]string `mapstructure:"player"`
	Playlist map[string]string `mapstructure:"playlist"`
	Library  map[string]string `mapstructure:"library"`
}

type AdvancedConfig struct {
	LogLevel          string        `mapstructure:"log_level"`
	EnableTelemetry   bool          `mapstructure:"enable_telemetry"`
	MemoryLimit       int64         `mapstructure:"memory_limit"` // in MB
	CPULimit          int           `mapstructure:"cpu_limit"`    // percentage
	ThreadPoolSize    int           `mapstructure:"thread_pool_size"`
	DatabasePoolSize  int           `mapstructure:"database_pool_size"`
	EnableProfiling   bool          `mapstructure:"enable_profiling"`
	ProfilePort       int           `mapstructure:"profile_port"`
	DebugMode         bool          `mapstructure:"debug_mode"`
	ExperimentalFeatures []string   `mapstructure:"experimental_features"`
}

func Get() *Config {
	once.Do(func() {
		instance = &Config{
			v: viper.New(),
		}
		instance.load()
	})
	return instance
}

func (c *Config) load() error {
	c.v.SetConfigName("config")
	c.v.SetConfigType("yaml")
	
	// Set config paths
	c.v.AddConfigPath(c.getUserConfigDir())
	c.v.AddConfigPath(c.getSystemConfigDir())
	c.v.AddConfigPath(".")
	
	// Set defaults
	c.setDefaults()
	
	// Read config
	if err := c.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create default
			if err := c.createDefaultConfig(); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}
	
	// Unmarshal config
	if err := c.v.Unmarshal(c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Watch for changes
	c.v.WatchConfig()
	c.v.OnConfigChange(func(e fsnotify.ConfigFileChangeEvent) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if err := c.v.Unmarshal(c); err != nil {
			fmt.Printf("Failed to reload config: %v\n", err)
		}
	})
	
	return nil
}

func (c *Config) setDefaults() {
	// App defaults
	c.v.SetDefault("app.name", "WinRamp")
	c.v.SetDefault("app.version", "1.0.0")
	c.v.SetDefault("app.data_dir", c.getDataDir())
	c.v.SetDefault("app.log_dir", filepath.Join(c.getDataDir(), "logs"))
	c.v.SetDefault("app.cache_dir", filepath.Join(c.getDataDir(), "cache"))
	c.v.SetDefault("app.auto_start", false)
	c.v.SetDefault("app.minimize_to_tray", true)
	c.v.SetDefault("app.check_for_updates", true)
	c.v.SetDefault("app.language", "en")
	c.v.SetDefault("app.theme", "dark")
	
	// Audio defaults
	c.v.SetDefault("audio.output_device", "default")
	c.v.SetDefault("audio.output_mode", "WASAPI")
	c.v.SetDefault("audio.exclusive_mode", false)
	c.v.SetDefault("audio.buffer_size", 2048)
	c.v.SetDefault("audio.sample_rate", 44100)
	c.v.SetDefault("audio.bit_depth", 16)
	c.v.SetDefault("audio.volume", 0.8)
	c.v.SetDefault("audio.crossfade_duration", 5*time.Second)
	c.v.SetDefault("audio.replay_gain", true)
	c.v.SetDefault("audio.replay_gain_mode", "track")
	c.v.SetDefault("audio.preamp", 0.0)
	c.v.SetDefault("audio.equalizer.enabled", false)
	c.v.SetDefault("audio.equalizer.preset", "flat")
	c.v.SetDefault("audio.equalizer.bands", [10]float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	c.v.SetDefault("audio.gapless_playback", true)
	c.v.SetDefault("audio.fade_on_pause", true)
	c.v.SetDefault("audio.fade_duration", 200*time.Millisecond)
	
	// Library defaults
	c.v.SetDefault("library.watch_folders", []string{})
	c.v.SetDefault("library.auto_scan", true)
	c.v.SetDefault("library.scan_interval", 1*time.Hour)
	c.v.SetDefault("library.extract_metadata", true)
	c.v.SetDefault("library.extract_album_art", true)
	c.v.SetDefault("library.album_art_max_size", 1024)
	c.v.SetDefault("library.skip_duplicates", true)
	c.v.SetDefault("library.min_track_duration", 10*time.Second)
	c.v.SetDefault("library.max_track_duration", 10*time.Hour)
	c.v.SetDefault("library.file_patterns", []string{"*.mp3", "*.flac", "*.ogg", "*.wav", "*.aac", "*.wma", "*.m4a"})
	c.v.SetDefault("library.exclude_patterns", []string{"*.tmp", "*.temp", "*.partial"})
	c.v.SetDefault("library.database_path", filepath.Join(c.getDataDir(), "library.db"))
	c.v.SetDefault("library.backup_database", true)
	c.v.SetDefault("library.backup_interval", 24*time.Hour)
	
	// UI defaults
	c.v.SetDefault("ui.window_mode", "modern")
	c.v.SetDefault("ui.skin", "default")
	c.v.SetDefault("ui.show_playlist", true)
	c.v.SetDefault("ui.show_equalizer", false)
	c.v.SetDefault("ui.show_library", true)
	c.v.SetDefault("ui.always_on_top", false)
	c.v.SetDefault("ui.snap_to_edges", true)
	c.v.SetDefault("ui.transparency", 1.0)
	c.v.SetDefault("ui.font_size", 12)
	c.v.SetDefault("ui.show_notifications", true)
	c.v.SetDefault("ui.animation_speed", 1.0)
	c.v.SetDefault("ui.double_click_action", "play")
	c.v.SetDefault("ui.column_layout", []string{"title", "artist", "album", "duration"})
	
	// Network defaults
	c.v.SetDefault("network.enable_sharing", false)
	c.v.SetDefault("network.enable_streaming", true)
	c.v.SetDefault("network.streaming_port", 8080)
	c.v.SetDefault("network.buffer_size", 65536)
	c.v.SetDefault("network.timeout", 30*time.Second)
	c.v.SetDefault("network.max_connections", 10)
	c.v.SetDefault("network.proxy_enabled", false)
	c.v.SetDefault("network.cache_enabled", true)
	c.v.SetDefault("network.cache_size", 500) // MB
	c.v.SetDefault("network.cache_path", filepath.Join(c.getDataDir(), "cache", "network"))
	
	// Shortcuts defaults
	c.v.SetDefault("shortcuts.global", map[string]string{
		"play_pause": "Space",
		"stop": "S",
		"next": "B",
		"previous": "Z",
		"volume_up": "Up",
		"volume_down": "Down",
	})
	
	// Advanced defaults
	c.v.SetDefault("advanced.log_level", "info")
	c.v.SetDefault("advanced.enable_telemetry", false)
	c.v.SetDefault("advanced.memory_limit", 512) // MB
	c.v.SetDefault("advanced.cpu_limit", 50)     // %
	c.v.SetDefault("advanced.thread_pool_size", runtime.NumCPU())
	c.v.SetDefault("advanced.database_pool_size", 10)
	c.v.SetDefault("advanced.enable_profiling", false)
	c.v.SetDefault("advanced.profile_port", 6060)
	c.v.SetDefault("advanced.debug_mode", false)
	c.v.SetDefault("advanced.experimental_features", []string{})
}

func (c *Config) getUserConfigDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "WinRamp")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "winramp")
}

func (c *Config) getSystemConfigDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "WinRamp")
	}
	return "/etc/winramp"
}

func (c *Config) getDataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "WinRamp")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "winramp")
}

func (c *Config) createDefaultConfig() error {
	configDir := c.getUserConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	
	configPath := filepath.Join(configDir, "config.yaml")
	return c.v.SafeWriteConfigAs(configPath)
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.WriteConfig()
}

func (c *Config) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v.ReadInConfig()
}

func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetString(key)
}

func (c *Config) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetInt(key)
}

func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetBool(key)
}

func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetDuration(key)
}

func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.v.Set(key, value)
}