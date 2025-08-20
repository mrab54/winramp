# WinRamp Development Plan

## Overview
This plan outlines the development of WinRamp in 5 major phases, with each phase building upon the previous one. The approach prioritizes getting a minimal viable player working early, then iteratively adding features.

**Timeline**: Estimated 16-20 weeks for full implementation
**Methodology**: Iterative development with working prototypes at each phase end

---

## Phase 1: Foundation & Core Architecture (Weeks 1-3)

### 1.1 Project Setup
- [ ] Initialize Go module with proper versioning
- [ ] Set up Git repository with .gitignore for Go/Wails
- [ ] Create project directory structure as defined in CLAUDE.md
- [ ] Configure GitHub Actions for CI/CD
- [ ] Set up pre-commit hooks (gofmt, golint, go vet)
- [ ] Initialize Makefile with build targets
- [ ] Set up development environment documentation

### 1.2 Core Domain Models
- [ ] Create `domain/track.go` with Track entity
  - [ ] Define Track struct with all metadata fields
  - [ ] Implement Track validation methods
  - [ ] Add Track serialization/deserialization
- [ ] Create `domain/playlist.go` with Playlist entity
  - [ ] Define Playlist struct with tracks and metadata
  - [ ] Implement playlist validation
  - [ ] Add playlist versioning for undo/redo
- [ ] Create `domain/library.go` with Library entity
  - [ ] Define Library struct for media collection
  - [ ] Implement library statistics methods
- [ ] Create `domain/errors.go` with custom error types
- [ ] Write unit tests for all domain models (target: 100% coverage)

### 1.3 Configuration System
- [ ] Set up Viper for configuration management
- [ ] Create `config/config.go` with configuration schema
- [ ] Define default configuration values
- [ ] Implement configuration file loading hierarchy
- [ ] Create configuration migration system for updates
- [ ] Add configuration validation
- [ ] Implement hot-reload for configuration changes
- [ ] Write configuration tests

### 1.4 Logging Infrastructure
- [ ] Set up zerolog for structured logging
- [ ] Create logging middleware for all layers
- [ ] Implement log rotation and archiving
- [ ] Add performance metrics logging
- [ ] Create debug mode with verbose logging
- [ ] Set up log aggregation for crash reports

### 1.5 Database Layer
- [ ] Set up SQLite with GORM
- [ ] Create database schema migrations
- [ ] Implement `infrastructure/db/connection.go`
- [ ] Create repository interfaces in domain layer
- [ ] Implement `infrastructure/db/track_repository.go`
- [ ] Implement `infrastructure/db/playlist_repository.go`
- [ ] Implement `infrastructure/db/library_repository.go`
- [ ] Add database connection pooling
- [ ] Implement database backup/restore functionality
- [ ] Write integration tests for repositories

---

## Phase 2: Audio Engine Implementation (Weeks 4-6)

### 2.1 Audio Decoder Framework
- [ ] Create `audio/decoder/interface.go` with Decoder interface
- [ ] Implement `audio/decoder/mp3.go` using go-mp3
  - [ ] Handle variable bitrate MP3s
  - [ ] Extract MP3 metadata during decode
  - [ ] Implement seek functionality
- [ ] Implement `audio/decoder/flac.go` using mewkiz/flac
  - [ ] Handle high-resolution FLAC files
  - [ ] Implement FLAC metadata extraction
- [ ] Implement `audio/decoder/wav.go` for WAV support
- [ ] Implement `audio/decoder/ogg.go` for OGG Vorbis
- [ ] Create `audio/decoder/factory.go` for decoder selection
- [ ] Add format detection from file headers
- [ ] Implement decoder pooling for performance
- [ ] Write decoder benchmarks and tests

### 2.2 Audio Output System
- [ ] Create `audio/output/interface.go` with Output interface
- [ ] Implement `audio/output/wasapi.go` for Windows audio
  - [ ] Support exclusive mode for bit-perfect playback
  - [ ] Implement shared mode with mixing
  - [ ] Handle device changes dynamically
- [ ] Implement `audio/output/directsound.go` as fallback
- [ ] Create output device enumeration
- [ ] Implement sample rate conversion
- [ ] Add volume control at output level
- [ ] Handle audio device hot-plugging
- [ ] Write output system tests

### 2.3 Core Player Engine
- [ ] Create `audio/player.go` with main Player struct
- [ ] Implement play/pause/stop functionality
- [ ] Add seeking with frame-accurate positioning
- [ ] Implement volume control with logarithmic scaling
- [ ] Create playback speed control (0.5x - 2.0x)
- [ ] Add current position tracking
- [ ] Implement player state machine
- [ ] Create event system for player state changes
- [ ] Add error recovery for corrupted streams
- [ ] Write comprehensive player tests

### 2.4 Audio Pipeline
- [ ] Create `audio/pipeline/pipeline.go` for audio processing chain
- [ ] Implement ring buffer for gapless playback
- [ ] Add crossfade support between tracks
- [ ] Create resampler for different sample rates
- [ ] Implement bit depth conversion
- [ ] Add channel mixing (mono/stereo/surround)
- [ ] Create pipeline benchmarks
- [ ] Optimize pipeline for low latency

### 2.5 DSP Effects
- [ ] Create `audio/dsp/equalizer.go` with 10-band EQ
  - [ ] Implement IIR filters for each band
  - [ ] Add presets (Rock, Pop, Jazz, etc.)
  - [ ] Create custom preset saving
- [ ] Implement `audio/dsp/normalizer.go` for replay gain
- [ ] Add `audio/dsp/compressor.go` for dynamic range
- [ ] Create `audio/dsp/effects_chain.go` for effect ordering
- [ ] Implement DSP bypass for bit-perfect playback
- [ ] Write DSP unit tests and benchmarks

---

## Phase 3: User Interface Development (Weeks 7-10)

### 3.1 Wails Integration
- [ ] Set up Wails v2 project structure
- [ ] Create `cmd/winramp/main.go` entry point
- [ ] Configure Wails build settings
- [ ] Set up frontend build pipeline
- [ ] Implement IPC between Go and frontend
- [ ] Create type definitions for frontend/backend communication
- [ ] Set up hot-reload for development
- [ ] Configure production build optimizations

### 3.2 Frontend Architecture
- [ ] Set up React/Vue/Svelte (choose based on performance)
- [ ] Configure TypeScript for type safety
- [ ] Set up Tailwind CSS or similar for styling
- [ ] Create state management (Redux/Zustand/Pinia)
- [ ] Implement frontend routing
- [ ] Set up frontend testing framework
- [ ] Create component library structure
- [ ] Add Storybook for component development

### 3.3 Main Player UI
- [ ] Create `MainWindow` component with player controls
  - [ ] Implement play/pause/stop buttons
  - [ ] Add previous/next track controls
  - [ ] Create seek bar with preview
  - [ ] Add volume slider with mute
  - [ ] Implement time display (elapsed/remaining)
- [ ] Create `NowPlaying` component
  - [ ] Display track metadata
  - [ ] Show album art with fallback
  - [ ] Add rating stars
  - [ ] Implement "love" button
- [ ] Add shuffle and repeat controls
- [ ] Implement playback speed control UI
- [ ] Create mini/compact mode view
- [ ] Add keyboard shortcuts
- [ ] Implement context menus

### 3.4 Playlist UI
- [ ] Create `PlaylistWindow` component
  - [ ] Implement playlist display with virtual scrolling
  - [ ] Add drag-and-drop reordering
  - [ ] Create multi-select with Shift/Ctrl
  - [ ] Implement inline editing
  - [ ] Add search/filter bar
- [ ] Implement playlist tabs for multiple playlists
- [ ] Create playlist context menu
  - [ ] Add to queue options
  - [ ] Remove/duplicate items
  - [ ] Track information dialog
- [ ] Add playlist statistics display
- [ ] Implement undo/redo for playlist operations
- [ ] Create playlist import/export UI

### 3.5 Library Browser
- [ ] Create `LibraryWindow` component
  - [ ] Implement tree view for artists/albums
  - [ ] Add grid view for albums with art
  - [ ] Create list view for all tracks
  - [ ] Implement column sorting
- [ ] Add library search with filters
- [ ] Create quick filter buttons (genre, year, etc.)
- [ ] Implement album art wall view
- [ ] Add recently played section
- [ ] Create most played section
- [ ] Implement smart playlist creation UI

### 3.6 Equalizer UI
- [ ] Create `EqualizerWindow` component
  - [ ] Implement 10-band slider controls
  - [ ] Add frequency labels
  - [ ] Create preset dropdown
  - [ ] Add save/delete preset buttons
  - [ ] Implement real-time preview
- [ ] Add spectrum analyzer display
- [ ] Create balance/pan control
- [ ] Implement effects toggle switches

### 3.7 Skin Engine
- [ ] Create `skins/engine.go` for skin loading
- [ ] Implement classic Winamp 2.x skin parser
- [ ] Create skin rendering system
- [ ] Add skin switching UI
- [ ] Implement skin download manager
- [ ] Create default modern skin
- [ ] Add skin customization options

### 3.8 Drag & Drop System
- [ ] Implement file drop from Windows Explorer
- [ ] Add folder drop with recursive scanning
- [ ] Create drop zones in UI (playlist, queue, library)
- [ ] Implement drag between windows
- [ ] Add visual feedback during drag
- [ ] Support multiple file selection
- [ ] Handle network path drops

---

## Phase 4: Advanced Features (Weeks 11-14)

### 4.1 Media Library Scanner
- [ ] Create `library/scanner.go` with directory watcher
- [ ] Implement parallel file scanning
- [ ] Add incremental scan support
- [ ] Create file change detection
- [ ] Implement metadata extraction
  - [ ] ID3v1/v2 for MP3
  - [ ] Vorbis comments for OGG/FLAC
  - [ ] Extract embedded album art
  - [ ] Handle multiple art images
- [ ] Add duplicate detection
- [ ] Create scan progress reporting
- [ ] Implement scan scheduling
- [ ] Write scanner performance tests

### 4.2 Metadata Management
- [ ] Create `library/metadata.go` for tag editing
- [ ] Implement batch metadata editing
- [ ] Add MusicBrainz integration
  - [ ] Acoustic fingerprinting
  - [ ] Automatic metadata correction
  - [ ] Album art fetching
- [ ] Create metadata conflict resolution
- [ ] Implement undo for metadata changes
- [ ] Add metadata export functionality

### 4.3 Search System
- [ ] Implement `library/search.go` with full-text search
- [ ] Set up SQLite FTS5 for indexing
- [ ] Create search query parser
- [ ] Add search suggestions/autocomplete
- [ ] Implement search history
- [ ] Create saved searches
- [ ] Add search result ranking
- [ ] Implement fuzzy matching

### 4.4 Playlist Management
- [ ] Create `playlist/service.go` for playlist operations
- [ ] Implement playlist CRUD operations
- [ ] Add playlist sharing functionality
- [ ] Create playlist versioning for collaboration
- [ ] Implement smart playlists
  - [ ] Rule-based generation
  - [ ] Auto-updating based on criteria
  - [ ] Limit and sorting options
- [ ] Add playlist folders/categories
- [ ] Create playlist templates

### 4.5 Queue System
- [ ] Implement `playlist/queue.go` for play queue
- [ ] Create queue persistence
- [ ] Add queue manipulation methods
  - [ ] Add next/add last
  - [ ] Move items up/down
  - [ ] Clear queue
  - [ ] Save queue as playlist
- [ ] Implement queue shuffle
- [ ] Add queue history
- [ ] Create queue predictions

### 4.6 Network Features
- [ ] Implement `network/shares.go` for SMB/CIFS
  - [ ] Network discovery
  - [ ] Credential management
  - [ ] Browse network shares
  - [ ] Handle network interruptions
- [ ] Create `network/streaming.go`
  - [ ] Internet radio support
  - [ ] Podcast RSS parsing
  - [ ] Stream buffering
  - [ ] Bandwidth adaptation
- [ ] Add DLNA/UPnP client
- [ ] Implement local network sync

### 4.7 Import/Export
- [ ] Implement M3U/M3U8 parser and writer
- [ ] Add PLS format support
- [ ] Create XSPF support
- [ ] Implement WPL for Windows Media Player
- [ ] Add iTunes library import
- [ ] Create Spotify playlist import (via API)
- [ ] Implement batch export functionality

### 4.8 Keyboard & Hotkeys
- [ ] Create global hotkey system
- [ ] Implement media key support
- [ ] Add customizable shortcuts
- [ ] Create shortcut conflict detection
- [ ] Implement shortcut profiles
- [ ] Add shortcut import/export

---

## Phase 5: Polish & Optimization (Weeks 15-16)

### 5.1 Performance Optimization
- [ ] Profile application with pprof
- [ ] Optimize database queries
  - [ ] Add appropriate indexes
  - [ ] Implement query caching
  - [ ] Optimize JOIN operations
- [ ] Reduce memory allocations
  - [ ] Implement object pooling
  - [ ] Use sync.Pool for buffers
  - [ ] Optimize string operations
- [ ] Optimize UI rendering
  - [ ] Implement virtual lists
  - [ ] Add lazy loading
  - [ ] Optimize React re-renders
- [ ] Implement audio decode caching
- [ ] Add speculative track pre-loading
- [ ] Optimize startup time
  - [ ] Lazy load modules
  - [ ] Defer non-critical initialization
  - [ ] Implement splash screen

### 5.2 Testing Suite
- [ ] Achieve 80% code coverage
- [ ] Create integration test suite
- [ ] Add E2E tests with Playwright
- [ ] Implement performance benchmarks
- [ ] Create stress tests
  - [ ] Large library handling (100k+ tracks)
  - [ ] Long playlist performance
  - [ ] Memory leak detection
- [ ] Add fuzz testing for parsers
- [ ] Create compatibility test matrix

### 5.3 Documentation
- [ ] Write user manual
- [ ] Create keyboard shortcut reference
- [ ] Document plugin API (future)
- [ ] Write troubleshooting guide
- [ ] Create video tutorials
- [ ] Generate API documentation
- [ ] Write developer guide

### 5.4 Installer & Distribution
- [ ] Create MSI installer with WiX
- [ ] Add portable version
- [ ] Implement auto-updater
- [ ] Create Windows Store package
- [ ] Add telemetry (opt-in)
- [ ] Implement crash reporting
- [ ] Create uninstaller

### 5.5 Accessibility
- [ ] Add screen reader support
- [ ] Implement high contrast mode
- [ ] Create keyboard-only navigation
- [ ] Add UI scaling options
- [ ] Implement color blind modes
- [ ] Add audio cues for actions

### 5.6 Final Polish
- [ ] Create onboarding flow
- [ ] Add tips and hints system
- [ ] Implement feedback collection
- [ ] Create bug report tool
- [ ] Add about dialog
- [ ] Implement EULA/license display
- [ ] Final security audit
- [ ] Performance profiling and optimization

---

## Milestones & Deliverables

### Milestone 1: Core Player (End of Phase 2)
- Working audio playback for MP3/FLAC
- Basic play/pause/seek functionality
- Command-line interface for testing

### Milestone 2: Basic UI (End of Phase 3)
- Functional player window
- Playlist management
- File browser integration

### Milestone 3: Full Featured (End of Phase 4)
- Complete media library
- All audio formats supported
- Network functionality
- Smart playlists

### Milestone 4: Release Candidate (End of Phase 5)
- Fully optimized performance
- Complete documentation
- Installer ready
- All tests passing

---

## Risk Mitigation

### Technical Risks
1. **Audio latency issues**
   - Mitigation: Early prototype testing, multiple output backends
2. **Memory leaks in long sessions**
   - Mitigation: Continuous profiling, automated testing
3. **Skin compatibility issues**
   - Mitigation: Extensive skin testing, compatibility mode

### Schedule Risks
1. **UI framework performance**
   - Mitigation: Early benchmarking, framework alternatives ready
2. **Network share complexity**
   - Mitigation: Phase network features, basic support first
3. **Metadata service dependencies**
   - Mitigation: Offline fallbacks, multiple service providers

---

## Success Metrics

- **Performance**: Startup < 500ms, CPU < 5% during playback
- **Stability**: Zero crashes in 24-hour playback test
- **Memory**: < 100MB for 50k track library
- **User Experience**: 60 FPS UI, instant response to actions
- **Compatibility**: Support 95% of Winamp 2.x skins
- **Quality**: 80% test coverage, all critical paths tested

---

## Post-Launch Roadmap

### Version 1.1
- Visualization plugin system
- Cloud sync support
- Mobile remote control

### Version 1.2
- Lyrics display with sync
- Last.fm integration
- Discord Rich Presence

### Version 2.0
- Video playback support
- Streaming service integration
- Social features

---

*This plan is a living document and should be updated as development progresses. Each task should be tracked in the project management system with appropriate time estimates and dependencies.*