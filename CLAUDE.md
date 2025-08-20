# WinRamp - Modern Winamp Clone for Windows 11

## Project Overview
WinRamp is a modern, high-performance music player for Windows 11, inspired by the classic Winamp. Built with Go, it combines nostalgic UI elements with modern architecture patterns and cutting-edge audio processing capabilities.

## Core Architecture

### Design Patterns & Principles
- **Clean Architecture**: Separation of concerns with domain, application, infrastructure, and presentation layers
- **Event-Driven Architecture**: Using Go channels for reactive UI updates and audio events
- **Repository Pattern**: Abstract data access for playlists and media library
- **Observer Pattern**: For UI state management and real-time updates
- **Command Pattern**: For undoable/redoable operations (playlist editing, queue management)
- **Strategy Pattern**: For different audio decoders and output devices
- **Dependency Injection**: Using interfaces for testability and flexibility

### Technology Stack
- **Language**: Go 1.22+
- **UI Framework**: Wails v2 (Go + Web Technologies for native Windows 11 experience)
- **Audio Processing**: 
  - `github.com/hajimehoshi/oto/v3` - Low-level audio output
  - `github.com/hajimehoshi/go-mp3` - MP3 decoding
  - `github.com/mewkiz/flac` - FLAC support
  - `github.com/dhowden/tag` - Metadata extraction
  - `github.com/faiface/beep` - High-level audio processing pipeline
- **Database**: SQLite with GORM for media library and playlists
- **Networking**: Native Go net/http for streaming and network drives
- **Configuration**: Viper for settings management

## Core Functionality

### Audio Playback Engine
- **Multi-format Support**: MP3, FLAC, OGG, WAV, AAC, WMA, M4A
- **Gapless Playback**: Pre-buffering next track for seamless transitions
- **Audio Pipeline**:
  ```
  File/Stream → Decoder → DSP Chain → Resampler → Output Device
  ```
- **DSP Effects**: 10-band equalizer, crossfade, replay gain normalization
- **Output Devices**: WASAPI (exclusive/shared), DirectSound fallback
- **Streaming Support**: HTTP/HTTPS streams, network shares (SMB/CIFS)

### Playlist Management
- **Smart Playlists**: Auto-generated based on rules (recently played, most played, genre-based)
- **Playlist Formats**: M3U, M3U8, PLS, XSPF, WPL
- **Queue System**: 
  - Current play queue separate from playlists
  - Add to queue vs Play next functionality
  - Queue persistence across sessions
- **Shuffle Algorithms**: 
  - True random
  - Weighted random (based on play count/rating)
  - Smart shuffle (avoids same artist repetition)

### User Interface
- **Skinnable Interface**: Support for classic Winamp 2.x skins
- **Window Modes**:
  - Classic mode (main window, playlist, equalizer)
  - Modern mode (unified interface)
  - Mini mode (compact player)
- **Drag & Drop Support**:
  - Files/folders to playlist or queue
  - Reorder within playlists
  - Between different playlist windows
  - From Windows Explorer and network locations

### Media Library
- **Auto-scanning**: Watch folders for new media
- **Metadata Management**: 
  - ID3v1, ID3v2, Vorbis comments
  - Album art extraction and display
  - Automatic metadata correction via MusicBrainz
- **Search**: Full-text search across artist, album, title, genre
- **Performance**: 
  - Lazy loading for large libraries
  - Indexed search with FTS5
  - Thumbnail caching for album art

### Network Features
- **Network Drive Support**: 
  - SMB/CIFS share browsing
  - Credential management (Windows Credential Manager integration)
  - Offline cache for network files
- **Streaming**: 
  - Internet radio stations
  - Podcast support with RSS
  - DLNA/UPnP client capabilities

## Project Structure
```
winramp/
├── cmd/
│   └── winramp/          # Main application entry point
├── internal/
│   ├── domain/           # Core business entities
│   │   ├── track.go
│   │   ├── playlist.go
│   │   └── library.go
│   ├── audio/            # Audio engine
│   │   ├── player.go
│   │   ├── decoder/      # Format-specific decoders
│   │   ├── dsp/          # Digital signal processing
│   │   └── output/       # Audio output backends
│   ├── playlist/         # Playlist management
│   │   ├── service.go
│   │   ├── formats/      # Import/export formats
│   │   └── queue.go
│   ├── library/          # Media library
│   │   ├── scanner.go
│   │   ├── metadata.go
│   │   └── search.go
│   ├── ui/               # User interface
│   │   ├── wails/        # Wails bindings
│   │   ├── skins/        # Skin engine
│   │   └── components/   # UI components
│   ├── network/          # Network functionality
│   │   ├── shares.go
│   │   └── streaming.go
│   └── config/           # Configuration management
├── frontend/             # Web-based UI (Wails)
│   ├── src/
│   └── dist/
├── assets/               # Static assets
│   ├── skins/
│   └── icons/
├── build/                # Build configurations
└── tests/                # Test suites

```

## Development Guidelines

### Code Quality
- **Testing**: Minimum 80% coverage, focus on audio engine and playlist logic
- **Benchmarking**: Regular performance benchmarks for audio processing
- **Error Handling**: Graceful degradation, never crash on bad media files
- **Logging**: Structured logging with zerolog, configurable verbosity
- **Documentation**: Godoc comments for all public APIs

### Performance Targets
- **Startup Time**: < 500ms to playable state
- **Library Scan**: 10,000 tracks/minute
- **Memory Usage**: < 100MB for 50,000 track library
- **CPU Usage**: < 5% during playback (excluding DSP)
- **UI Responsiveness**: 60 FPS scrolling, instant feedback

### Key Implementation Notes

#### Audio Playback
- Use worker pool pattern for parallel decoding
- Implement ring buffer for gapless playback
- Pre-calculate waveform data for UI visualization
- Hardware acceleration via Windows Media Foundation when available

#### Playlist Operations
- Copy-on-write for efficient playlist duplication
- Batch operations for bulk adding/removing
- Lazy evaluation for smart playlists
- Incremental save for large playlists

#### File Handling
- Async I/O for all file operations
- Progressive loading for large playlists
- Memory-mapped files for frequently accessed data
- Watch for external file changes

#### UI Responsiveness
- Virtual scrolling for large lists
- Debounced search with progressive results
- Optimistic UI updates with rollback on error
- Web Workers for heavy computations

### Testing Strategy
- **Unit Tests**: Core audio logic, playlist algorithms
- **Integration Tests**: File format support, network operations
- **E2E Tests**: UI workflows, drag-drop operations
- **Performance Tests**: Load testing with large libraries
- **Compatibility Tests**: Various audio formats and bitrates

## Build & Deployment

### Prerequisites
```bash
# Install Go 1.22+
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
# Install build tools
choco install mingw make
```

### Build Commands
```bash
# Development build with hot reload
make dev

# Production build
make build

# Run tests
make test

# Generate installer
make installer
```

### Configuration
The application uses a hierarchical configuration system:
1. Default settings (embedded)
2. System-wide settings (`%ProgramData%\WinRamp\config.yaml`)
3. User settings (`%AppData%\WinRamp\config.yaml`)
4. Command-line flags

## Future Enhancements
- [ ] Visualization plugins API
- [ ] Cloud sync for playlists
- [ ] Mobile remote control app
- [ ] Last.fm scrobbling
- [ ] Discord Rich Presence
- [ ] Audio fingerprinting for duplicate detection
- [ ] Smart speaker integration
- [ ] Lyrics display with sync
- [ ] Concert/tour date notifications
- [ ] Social features (playlist sharing)

## Performance Optimizations
- Implement audio decode caching for frequently played tracks
- Use memory pooling for audio buffers
- Leverage Windows 11 DirectStorage for faster file access
- GPU acceleration for spectrum analysis
- Compile-time optimization with PGO (Profile-Guided Optimization)

## Security Considerations
- Sanitize all metadata to prevent XSS in UI
- Validate all playlist files to prevent path traversal
- Secure credential storage for network shares
- Content Security Policy for web UI
- Regular dependency audits

## Troubleshooting
Common issues and solutions will be documented here as the project develops.

## License
[Project License Information]

---
*This document should be updated as the project evolves. Each major feature addition should include updates to the relevant sections.*