export class Playlist {
    constructor() {
        this.currentPlaylist = null;
        this.tracks = [];
    }
    
    async init() {
        // Initial setup
    }
    
    render() {
        return `
            <div class="playlist-container">
                <div class="playlist-header">
                    <h2 id="playlist-name">No Playlist Selected</h2>
                    <div class="playlist-actions">
                        <button class="btn" id="btn-add-tracks">Add Tracks</button>
                        <button class="btn" id="btn-clear-playlist">Clear</button>
                        <button class="btn" id="btn-save-playlist">Save</button>
                    </div>
                </div>
                
                <div class="playlist-content">
                    <table class="playlist-table">
                        <thead>
                            <tr>
                                <th width="30">#</th>
                                <th>Title</th>
                                <th>Artist</th>
                                <th>Album</th>
                                <th width="60">Duration</th>
                                <th width="80">Actions</th>
                            </tr>
                        </thead>
                        <tbody id="playlist-tracks">
                            <!-- Tracks will be loaded here -->
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }
    
    load(playlist) {
        this.currentPlaylist = playlist;
        this.tracks = playlist.tracks || [];
        
        // Update UI
        document.getElementById('playlist-name').textContent = playlist.name;
        this.updateTrackList();
    }
    
    updateTrackList() {
        const tbody = document.getElementById('playlist-tracks');
        if (!tbody) return;
        
        tbody.innerHTML = this.tracks.map((track, index) => `
            <tr data-track-id="${track.id}">
                <td>${index + 1}</td>
                <td>${track.title || 'Unknown'}</td>
                <td>${track.artist || 'Unknown'}</td>
                <td>${track.album || ''}</td>
                <td>${this.formatDuration(track.duration)}</td>
                <td>
                    <button class="btn-small play-track" data-index="${index}">▶</button>
                    <button class="btn-small remove-track" data-index="${index}">✕</button>
                </td>
            </tr>
        `).join('');
        
        // Add event handlers
        tbody.querySelectorAll('.play-track').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const index = parseInt(e.target.dataset.index);
                this.playTrack(index);
            });
        });
        
        tbody.querySelectorAll('.remove-track').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const index = parseInt(e.target.dataset.index);
                this.removeTrack(index);
            });
        });
    }
    
    async playTrack(index) {
        if (index >= 0 && index < this.tracks.length) {
            const track = this.tracks[index];
            try {
                await window.go.main.App.LoadTrack(track);
                await window.go.main.App.Play();
            } catch (error) {
                console.error('Failed to play track:', error);
            }
        }
    }
    
    async removeTrack(index) {
        if (index >= 0 && index < this.tracks.length) {
            const track = this.tracks[index];
            try {
                await window.go.main.App.RemoveFromPlaylist(this.currentPlaylist.id, [track.id]);
                this.tracks.splice(index, 1);
                this.updateTrackList();
            } catch (error) {
                console.error('Failed to remove track:', error);
            }
        }
    }
    
    formatDuration(seconds) {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
}