import './style.css';
import { Player } from './player.js';
import { Playlist } from './playlist.js';
import { Library } from './library.js';
import { Settings } from './settings.js';

// Main application class
class WinRampApp {
    constructor() {
        this.player = new Player();
        this.playlist = new Playlist();
        this.library = new Library();
        this.settings = new Settings();
        this.currentView = 'player';
        
        this.init();
    }
    
    async init() {
        // Build UI
        this.buildUI();
        
        // Initialize components
        await this.player.init();
        await this.playlist.init();
        await this.library.init();
        
        // Set up event handlers
        this.setupEventHandlers();
        
        // Load initial data
        await this.loadInitialData();
    }
    
    buildUI() {
        const app = document.getElementById('app');
        app.innerHTML = `
            <div class="winramp-container">
                <div class="title-bar">
                    <div class="title-bar-text">WinRamp</div>
                    <div class="title-bar-controls">
                        <button class="minimize">_</button>
                        <button class="maximize">□</button>
                        <button class="close">×</button>
                    </div>
                </div>
                
                <div class="menu-bar">
                    <button class="menu-item" data-menu="file">File</button>
                    <button class="menu-item" data-menu="view">View</button>
                    <button class="menu-item" data-menu="playback">Playback</button>
                    <button class="menu-item" data-menu="playlist">Playlist</button>
                    <button class="menu-item" data-menu="help">Help</button>
                </div>
                
                <div class="main-content">
                    <div class="sidebar">
                        <div class="sidebar-section">
                            <h3>Library</h3>
                            <ul class="tree-view">
                                <li><span class="icon">🎵</span> All Music</li>
                                <li><span class="icon">👤</span> Artists</li>
                                <li><span class="icon">💿</span> Albums</li>
                                <li><span class="icon">🎼</span> Genres</li>
                            </ul>
                        </div>
                        
                        <div class="sidebar-section">
                            <h3>Playlists</h3>
                            <ul class="tree-view" id="playlist-list">
                                <!-- Playlists will be loaded here -->
                            </ul>
                        </div>
                    </div>
                    
                    <div class="content-area">
                        <div id="player-view" class="view active">
                            ${this.player.render()}
                        </div>
                        
                        <div id="playlist-view" class="view">
                            ${this.playlist.render()}
                        </div>
                        
                        <div id="library-view" class="view">
                            ${this.library.render()}
                        </div>
                        
                        <div id="settings-view" class="view">
                            ${this.settings.render()}
                        </div>
                    </div>
                </div>
                
                <div class="status-bar">
                    <span id="status-text">Ready</span>
                    <span id="track-info"></span>
                </div>
            </div>
        `;
    }
    
    setupEventHandlers() {
        // Window controls
        document.querySelector('.minimize').addEventListener('click', () => {
            window.runtime.WindowMinimise();
        });
        
        document.querySelector('.maximize').addEventListener('click', () => {
            window.runtime.WindowToggleMaximise();
        });
        
        document.querySelector('.close').addEventListener('click', () => {
            window.runtime.Quit();
        });
        
        // Menu items
        document.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', (e) => {
                this.handleMenuClick(e.target.dataset.menu);
            });
        });
        
        // Sidebar navigation
        document.querySelectorAll('.tree-view li').forEach(item => {
            item.addEventListener('click', (e) => {
                this.handleSidebarClick(e.target.textContent);
            });
        });
        
        // Listen for backend events
        window.runtime.EventsOn('player:stateChanged', (state) => {
            this.player.updateState(state);
        });
        
        window.runtime.EventsOn('player:trackChanged', (track) => {
            this.player.updateTrack(track);
            this.updateTrackInfo(track);
        });
        
        window.runtime.EventsOn('player:positionChanged', (position) => {
            this.player.updatePosition(position);
        });
        
        window.runtime.EventsOn('player:error', (error) => {
            this.showError(error);
        });
    }
    
    async loadInitialData() {
        try {
            // Load playlists
            const playlists = await window.go.main.App.GetPlaylists();
            this.updatePlaylistList(playlists);
            
            // Load library
            const tracks = await window.go.main.App.GetLibraryTracks();
            this.library.updateTracks(tracks);
            
            // Get player state
            const playerState = await window.go.main.App.GetPlayerState();
            this.player.updateFullState(playerState);
            
        } catch (error) {
            console.error('Failed to load initial data:', error);
        }
    }
    
    handleMenuClick(menu) {
        switch (menu) {
            case 'file':
                this.showFileMenu();
                break;
            case 'view':
                this.showViewMenu();
                break;
            case 'playback':
                this.showPlaybackMenu();
                break;
            case 'playlist':
                this.showPlaylistMenu();
                break;
            case 'help':
                this.showHelpMenu();
                break;
        }
    }
    
    handleSidebarClick(item) {
        if (item.includes('All Music')) {
            this.showView('library');
        } else if (item.includes('Artists')) {
            this.library.showArtists();
        } else if (item.includes('Albums')) {
            this.library.showAlbums();
        } else if (item.includes('Genres')) {
            this.library.showGenres();
        }
    }
    
    showView(viewName) {
        document.querySelectorAll('.view').forEach(view => {
            view.classList.remove('active');
        });
        
        const view = document.getElementById(`${viewName}-view`);
        if (view) {
            view.classList.add('active');
            this.currentView = viewName;
        }
    }
    
    updatePlaylistList(playlists) {
        const listElement = document.getElementById('playlist-list');
        listElement.innerHTML = playlists.map(pl => `
            <li data-playlist-id="${pl.id}">
                <span class="icon">📄</span> ${pl.name}
                <span class="count">(${pl.trackCount})</span>
            </li>
        `).join('');
        
        // Add click handlers
        listElement.querySelectorAll('li').forEach(item => {
            item.addEventListener('click', () => {
                const playlistId = item.dataset.playlistId;
                this.loadPlaylist(playlistId);
            });
        });
    }
    
    async loadPlaylist(playlistId) {
        try {
            const playlist = await window.go.main.App.GetPlaylist(playlistId);
            this.playlist.load(playlist);
            this.showView('playlist');
        } catch (error) {
            console.error('Failed to load playlist:', error);
        }
    }
    
    updateTrackInfo(track) {
        const info = document.getElementById('track-info');
        if (track) {
            info.textContent = `${track.artist} - ${track.title}`;
        } else {
            info.textContent = '';
        }
    }
    
    showError(error) {
        const status = document.getElementById('status-text');
        status.textContent = `Error: ${error}`;
        status.style.color = 'red';
        
        setTimeout(() => {
            status.textContent = 'Ready';
            status.style.color = '';
        }, 5000);
    }
    
    async showFileMenu() {
        // Implementation for file menu
    }
    
    async showViewMenu() {
        // Implementation for view menu
    }
    
    async showPlaybackMenu() {
        // Implementation for playback menu
    }
    
    async showPlaylistMenu() {
        // Implementation for playlist menu
    }
    
    async showHelpMenu() {
        // Implementation for help menu
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new WinRampApp();
});