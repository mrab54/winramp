import './style.css';
import { Player } from './player.js';
import { Playlist } from './playlist.js';
import { Library } from './library.js';
import { Settings } from './settings.js';
import { Security, createElement, setTextContent } from './security.js';

// Secure version of main application class
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
        // Build UI safely
        this.buildUISecure();
        
        // Initialize components
        await this.player.init();
        await this.playlist.init();
        await this.library.init();
        
        // Set up event handlers
        this.setupEventHandlers();
        
        // Load initial data
        await this.loadInitialData();
    }
    
    buildUISecure() {
        const app = document.getElementById('app');
        
        // Clear existing content safely
        while (app.firstChild) {
            app.removeChild(app.firstChild);
        }
        
        // Build UI using safe DOM methods
        const container = createElement('div', { className: 'winramp-container' });
        
        // Title bar
        const titleBar = this.createTitleBar();
        container.appendChild(titleBar);
        
        // Menu bar
        const menuBar = this.createMenuBar();
        container.appendChild(menuBar);
        
        // Main content
        const mainContent = this.createMainContent();
        container.appendChild(mainContent);
        
        // Status bar
        const statusBar = this.createStatusBar();
        container.appendChild(statusBar);
        
        app.appendChild(container);
    }
    
    createTitleBar() {
        const titleBar = createElement('div', { className: 'title-bar' });
        
        const titleText = createElement('div', { className: 'title-bar-text' }, 'WinRamp');
        titleBar.appendChild(titleText);
        
        const controls = createElement('div', { className: 'title-bar-controls' });
        
        const minimizeBtn = createElement('button', { className: 'minimize' }, '_');
        const maximizeBtn = createElement('button', { className: 'maximize' }, '□');
        const closeBtn = createElement('button', { className: 'close' }, '×');
        
        controls.appendChild(minimizeBtn);
        controls.appendChild(maximizeBtn);
        controls.appendChild(closeBtn);
        
        titleBar.appendChild(controls);
        
        return titleBar;
    }
    
    createMenuBar() {
        const menuBar = createElement('div', { className: 'menu-bar' });
        
        const menuItems = ['File', 'View', 'Playback', 'Playlist', 'Help'];
        
        menuItems.forEach(item => {
            const menuBtn = createElement('button', {
                className: 'menu-item',
                dataset: { menu: item.toLowerCase() }
            }, item);
            menuBar.appendChild(menuBtn);
        });
        
        return menuBar;
    }
    
    createMainContent() {
        const mainContent = createElement('div', { className: 'main-content' });
        
        // Sidebar
        const sidebar = this.createSidebar();
        mainContent.appendChild(sidebar);
        
        // Content area
        const contentArea = this.createContentArea();
        mainContent.appendChild(contentArea);
        
        return mainContent;
    }
    
    createSidebar() {
        const sidebar = createElement('div', { className: 'sidebar' });
        
        // Library section
        const librarySection = createElement('div', { className: 'sidebar-section' });
        const libraryTitle = createElement('h3', {}, 'Library');
        librarySection.appendChild(libraryTitle);
        
        const treeView = createElement('ul', { className: 'tree-view' });
        
        const libraryItems = [
            { icon: '🎵', text: 'All Music' },
            { icon: '👤', text: 'Artists' },
            { icon: '💿', text: 'Albums' },
            { icon: '🎼', text: 'Genres' }
        ];
        
        libraryItems.forEach(item => {
            const li = createElement('li');
            const icon = createElement('span', { className: 'icon' }, item.icon);
            const text = document.createTextNode(' ' + item.text);
            li.appendChild(icon);
            li.appendChild(text);
            treeView.appendChild(li);
        });
        
        librarySection.appendChild(treeView);
        sidebar.appendChild(librarySection);
        
        // Playlists section
        const playlistSection = createElement('div', { className: 'sidebar-section' });
        const playlistTitle = createElement('h3', {}, 'Playlists');
        playlistSection.appendChild(playlistTitle);
        
        const playlistList = createElement('ul', {
            className: 'tree-view',
            id: 'playlist-list'
        });
        
        playlistSection.appendChild(playlistList);
        sidebar.appendChild(playlistSection);
        
        return sidebar;
    }
    
    createContentArea() {
        const contentArea = createElement('div', { className: 'content-area' });
        
        // Create view containers (content will be rendered by components)
        const views = [
            { id: 'player-view', className: 'view active' },
            { id: 'playlist-view', className: 'view' },
            { id: 'library-view', className: 'view' },
            { id: 'settings-view', className: 'view' }
        ];
        
        views.forEach(view => {
            const viewDiv = createElement('div', {
                id: view.id,
                className: view.className
            });
            
            // Components will render their content here
            if (view.id === 'player-view') {
                viewDiv.innerHTML = this.player.render(); // Component controls its own rendering
            } else if (view.id === 'playlist-view') {
                viewDiv.innerHTML = this.playlist.render();
            } else if (view.id === 'library-view') {
                viewDiv.innerHTML = this.library.render();
            } else if (view.id === 'settings-view') {
                viewDiv.innerHTML = this.settings.render();
            }
            
            contentArea.appendChild(viewDiv);
        });
        
        return contentArea;
    }
    
    createStatusBar() {
        const statusBar = createElement('div', { className: 'status-bar' });
        
        const statusText = createElement('span', { id: 'status-text' }, 'Ready');
        const trackInfo = createElement('span', { id: 'track-info' });
        
        statusBar.appendChild(statusText);
        statusBar.appendChild(trackInfo);
        
        return statusBar;
    }
    
    setupEventHandlers() {
        // Window controls
        document.querySelector('.minimize')?.addEventListener('click', () => {
            window.runtime?.WindowMinimise();
        });
        
        document.querySelector('.maximize')?.addEventListener('click', () => {
            window.runtime?.WindowToggleMaximise();
        });
        
        document.querySelector('.close')?.addEventListener('click', () => {
            window.runtime?.Quit();
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
                const text = e.target.textContent.trim();
                this.handleSidebarClick(text);
            });
        });
        
        // Listen for backend events
        window.runtime?.EventsOn('player:stateChanged', (state) => {
            this.player.updateState(state);
        });
        
        window.runtime?.EventsOn('player:trackChanged', (track) => {
            this.player.updateTrack(track);
            this.updateTrackInfo(track);
        });
        
        window.runtime?.EventsOn('player:positionChanged', (position) => {
            this.player.updatePosition(position);
        });
        
        window.runtime?.EventsOn('player:error', (error) => {
            this.showError(error);
        });
    }
    
    async loadInitialData() {
        try {
            // Load playlists
            const playlists = await window.go?.main?.App?.GetPlaylists();
            if (playlists) {
                this.updatePlaylistList(playlists);
            }
            
            // Load library
            const tracks = await window.go?.main?.App?.GetLibraryTracks();
            if (tracks) {
                this.library.updateTracks(tracks);
            }
            
            // Get player state
            const playerState = await window.go?.main?.App?.GetPlayerState();
            if (playerState) {
                this.player.updateFullState(playerState);
            }
            
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
        if (!listElement) return;
        
        // Clear existing content safely
        while (listElement.firstChild) {
            listElement.removeChild(listElement.firstChild);
        }
        
        // Add playlists using safe DOM methods
        playlists.forEach(pl => {
            const li = createElement('li', {
                dataset: { playlistId: pl.id }
            });
            
            const icon = createElement('span', { className: 'icon' }, '📄');
            const name = document.createTextNode(' ' + Security.escapeHtml(pl.name));
            const count = createElement('span', { className: 'count' }, `(${pl.trackCount})`);
            
            li.appendChild(icon);
            li.appendChild(name);
            li.appendChild(count);
            
            li.addEventListener('click', () => {
                this.loadPlaylist(pl.id);
            });
            
            listElement.appendChild(li);
        });
    }
    
    async loadPlaylist(playlistId) {
        try {
            const playlist = await window.go?.main?.App?.GetPlaylist(playlistId);
            if (playlist) {
                this.playlist.load(playlist);
                this.showView('playlist');
            }
        } catch (error) {
            console.error('Failed to load playlist:', error);
        }
    }
    
    updateTrackInfo(track) {
        const info = document.getElementById('track-info');
        if (info) {
            if (track) {
                // Use safe text content setting
                setTextContent(info, `${track.artist} - ${track.title}`);
            } else {
                setTextContent(info, '');
            }
        }
    }
    
    showError(error) {
        const status = document.getElementById('status-text');
        if (status) {
            setTextContent(status, `Error: ${error}`);
            status.style.color = 'red';
            
            setTimeout(() => {
                setTextContent(status, 'Ready');
                status.style.color = '';
            }, 5000);
        }
    }
    
    // Menu methods remain empty as in original
    async showFileMenu() {}
    async showViewMenu() {}
    async showPlaybackMenu() {}
    async showPlaylistMenu() {}
    async showHelpMenu() {}
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new WinRampApp();
});