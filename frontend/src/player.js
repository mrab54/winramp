export class Player {
    constructor() {
        this.state = 'stopped';
        this.currentTrack = null;
        this.position = 0;
        this.duration = 0;
        this.volume = 1.0;
        this.isSeekingUser = false;
    }
    
    async init() {
        // Initial setup
    }
    
    render() {
        return `
            <div class="player-container">
                <div class="player-display">
                    <div class="track-info">
                        <div class="track-title" id="track-title">No Track Loaded</div>
                        <div class="track-artist" id="track-artist">-</div>
                        <div class="track-album" id="track-album">-</div>
                    </div>
                    
                    <div class="album-art">
                        <img id="album-art" src="" alt="Album Art" style="display: none;">
                        <div class="no-art">🎵</div>
                    </div>
                    
                    <div class="visualizer">
                        <canvas id="visualizer"></canvas>
                    </div>
                </div>
                
                <div class="player-controls">
                    <div class="time-display">
                        <span id="time-current">0:00</span>
                        <div class="seek-bar">
                            <input type="range" id="seek-slider" min="0" max="100" value="0">
                            <div class="seek-progress" id="seek-progress"></div>
                        </div>
                        <span id="time-total">0:00</span>
                    </div>
                    
                    <div class="control-buttons">
                        <button class="control-btn" id="btn-previous" title="Previous">⏮</button>
                        <button class="control-btn" id="btn-play" title="Play">▶</button>
                        <button class="control-btn" id="btn-pause" title="Pause" style="display: none;">⏸</button>
                        <button class="control-btn" id="btn-stop" title="Stop">⏹</button>
                        <button class="control-btn" id="btn-next" title="Next">⏭</button>
                    </div>
                    
                    <div class="volume-control">
                        <button class="volume-icon" id="volume-icon">🔊</button>
                        <input type="range" id="volume-slider" min="0" max="100" value="100">
                        <span id="volume-value">100%</span>
                    </div>
                    
                    <div class="playback-options">
                        <button class="option-btn" id="btn-shuffle" title="Shuffle">🔀</button>
                        <button class="option-btn" id="btn-repeat" title="Repeat">🔁</button>
                        <button class="option-btn" id="btn-equalizer" title="Equalizer">⚡</button>
                    </div>
                </div>
            </div>
        `;
    }
    
    updateState(state) {
        this.state = state;
        this.updateControls();
    }
    
    updateTrack(track) {
        this.currentTrack = track;
        
        if (track) {
            document.getElementById('track-title').textContent = track.title || 'Unknown Title';
            document.getElementById('track-artist').textContent = track.artist || 'Unknown Artist';
            document.getElementById('track-album').textContent = track.album || 'Unknown Album';
            
            // Update duration
            this.duration = track.duration || 0;
            document.getElementById('time-total').textContent = this.formatTime(this.duration);
            
            // Update seek slider max
            const seekSlider = document.getElementById('seek-slider');
            seekSlider.max = Math.floor(this.duration);
        } else {
            document.getElementById('track-title').textContent = 'No Track Loaded';
            document.getElementById('track-artist').textContent = '-';
            document.getElementById('track-album').textContent = '-';
            document.getElementById('time-total').textContent = '0:00';
        }
    }
    
    updatePosition(position) {
        if (this.isSeekingUser) {
            return; // Don't update while user is seeking
        }
        
        this.position = position;
        document.getElementById('time-current').textContent = this.formatTime(position);
        
        const seekSlider = document.getElementById('seek-slider');
        seekSlider.value = Math.floor(position);
        
        // Update progress bar
        const progress = this.duration > 0 ? (position / this.duration) * 100 : 0;
        document.getElementById('seek-progress').style.width = `${progress}%`;
    }
    
    updateFullState(state) {
        if (state.state) {
            this.updateState(state.state);
        }
        if (state.track) {
            this.updateTrack(state.track);
        }
        if (state.position !== undefined) {
            this.updatePosition(state.position);
        }
        if (state.duration !== undefined) {
            this.duration = state.duration;
        }
    }
    
    updateControls() {
        const playBtn = document.getElementById('btn-play');
        const pauseBtn = document.getElementById('btn-pause');
        
        if (this.state === 'playing') {
            playBtn.style.display = 'none';
            pauseBtn.style.display = 'inline-block';
        } else {
            playBtn.style.display = 'inline-block';
            pauseBtn.style.display = 'none';
        }
    }
    
    formatTime(seconds) {
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
    
    setupEventHandlers() {
        // Play button
        document.getElementById('btn-play').addEventListener('click', async () => {
            try {
                await window.go.main.App.Play();
            } catch (error) {
                console.error('Failed to play:', error);
            }
        });
        
        // Pause button
        document.getElementById('btn-pause').addEventListener('click', async () => {
            try {
                await window.go.main.App.Pause();
            } catch (error) {
                console.error('Failed to pause:', error);
            }
        });
        
        // Stop button
        document.getElementById('btn-stop').addEventListener('click', async () => {
            try {
                await window.go.main.App.Stop();
            } catch (error) {
                console.error('Failed to stop:', error);
            }
        });
        
        // Previous button
        document.getElementById('btn-previous').addEventListener('click', async () => {
            try {
                await window.go.main.App.Previous();
            } catch (error) {
                console.error('Failed to play previous:', error);
            }
        });
        
        // Next button
        document.getElementById('btn-next').addEventListener('click', async () => {
            try {
                await window.go.main.App.Next();
            } catch (error) {
                console.error('Failed to play next:', error);
            }
        });
        
        // Seek slider
        const seekSlider = document.getElementById('seek-slider');
        seekSlider.addEventListener('mousedown', () => {
            this.isSeekingUser = true;
        });
        
        seekSlider.addEventListener('mouseup', async () => {
            this.isSeekingUser = false;
            const position = parseFloat(seekSlider.value);
            try {
                await window.go.main.App.Seek(position);
            } catch (error) {
                console.error('Failed to seek:', error);
            }
        });
        
        seekSlider.addEventListener('input', () => {
            const position = parseFloat(seekSlider.value);
            document.getElementById('time-current').textContent = this.formatTime(position);
        });
        
        // Volume slider
        const volumeSlider = document.getElementById('volume-slider');
        volumeSlider.addEventListener('input', async () => {
            const volume = volumeSlider.value / 100;
            this.volume = volume;
            document.getElementById('volume-value').textContent = `${volumeSlider.value}%`;
            
            try {
                await window.go.main.App.SetVolume(volume);
            } catch (error) {
                console.error('Failed to set volume:', error);
            }
            
            // Update volume icon
            const icon = document.getElementById('volume-icon');
            if (volume === 0) {
                icon.textContent = '🔇';
            } else if (volume < 0.5) {
                icon.textContent = '🔉';
            } else {
                icon.textContent = '🔊';
            }
        });
        
        // Volume icon (mute toggle)
        document.getElementById('volume-icon').addEventListener('click', async () => {
            const volumeSlider = document.getElementById('volume-slider');
            if (this.volume > 0) {
                this.previousVolume = this.volume;
                volumeSlider.value = 0;
            } else {
                volumeSlider.value = (this.previousVolume || 0.5) * 100;
            }
            volumeSlider.dispatchEvent(new Event('input'));
        });
    }
}