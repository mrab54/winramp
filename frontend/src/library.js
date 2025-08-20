export class Library {
    constructor() {
        this.tracks = [];
        this.view = 'all'; // all, artists, albums, genres
    }
    
    async init() {
        // Initial setup
    }
    
    render() {
        return `
            <div class="library-container">
                <div class="library-header">
                    <h2>Music Library</h2>
                    <div class="library-actions">
                        <input type="text" id="library-search" placeholder="Search...">
                        <button class="btn" id="btn-scan-folder">Scan Folder</button>
                        <button class="btn" id="btn-import-files">Import Files</button>
                    </div>
                </div>
                
                <div class="library-toolbar">
                    <button class="view-btn active" data-view="all">All</button>
                    <button class="view-btn" data-view="artists">Artists</button>
                    <button class="view-btn" data-view="albums">Albums</button>
                    <button class="view-btn" data-view="genres">Genres</button>
                </div>
                
                <div class="library-content">
                    <table class="library-table">
                        <thead>
                            <tr>
                                <th>Title</th>
                                <th>Artist</th>
                                <th>Album</th>
                                <th>Genre</th>
                                <th>Year</th>
                                <th>Duration</th>
                            </tr>
                        </thead>
                        <tbody id="library-tracks">
                            <!-- Tracks will be loaded here -->
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }
    
    updateTracks(tracks) {
        this.tracks = tracks || [];
        this.displayTracks();
    }
    
    displayTracks() {
        const tbody = document.getElementById('library-tracks');
        if (!tbody) return;
        
        tbody.innerHTML = this.tracks.map(track => `
            <tr data-track-id="${track.id}" class="library-track">
                <td>${track.title || 'Unknown'}</td>
                <td>${track.artist || 'Unknown'}</td>
                <td>${track.album || ''}</td>
                <td>${track.genre || ''}</td>
                <td>${track.year || ''}</td>
                <td>${this.formatDuration(track.duration)}</td>
            </tr>
        `).join('');
        
        // Add double-click to play
        tbody.querySelectorAll('.library-track').forEach(row => {
            row.addEventListener('dblclick', async () => {
                const trackId = row.dataset.trackId;
                const track = this.tracks.find(t => t.id === trackId);
                if (track) {
                    try {
                        await window.go.main.App.LoadTrack(track);
                        await window.go.main.App.Play();
                    } catch (error) {
                        console.error('Failed to play track:', error);
                    }
                }
            });
        });
    }
    
    showArtists() {
        this.view = 'artists';
        // Group by artists - implementation needed
    }
    
    showAlbums() {
        this.view = 'albums';
        // Group by albums - implementation needed
    }
    
    showGenres() {
        this.view = 'genres';
        // Group by genres - implementation needed
    }
    
    formatDuration(seconds) {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
}