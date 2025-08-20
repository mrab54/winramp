export class Settings {
    constructor() {
        this.settings = {};
    }
    
    render() {
        return `
            <div class="settings-container">
                <h2>Settings</h2>
                
                <div class="settings-section">
                    <h3>Audio</h3>
                    <div class="setting-item">
                        <label>Crossfade Duration (seconds)</label>
                        <input type="number" id="setting-crossfade" min="0" max="10" value="5">
                    </div>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-replaygain">
                            Enable Replay Gain
                        </label>
                    </div>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-gapless">
                            Enable Gapless Playback
                        </label>
                    </div>
                </div>
                
                <div class="settings-section">
                    <h3>Library</h3>
                    <div class="setting-item">
                        <label>Watch Folders</label>
                        <div id="watch-folders-list"></div>
                        <button class="btn" id="btn-add-watch-folder">Add Folder</button>
                    </div>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-autoscan">
                            Auto-scan for new files
                        </label>
                    </div>
                </div>
                
                <div class="settings-section">
                    <h3>Interface</h3>
                    <div class="setting-item">
                        <label>Theme</label>
                        <select id="setting-theme">
                            <option value="dark">Dark</option>
                            <option value="light">Light</option>
                            <option value="auto">Auto</option>
                        </select>
                    </div>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-alwaysontop">
                            Always on Top
                        </label>
                    </div>
                </div>
                
                <div class="settings-actions">
                    <button class="btn btn-primary" id="btn-save-settings">Save Settings</button>
                    <button class="btn" id="btn-reset-settings">Reset to Defaults</button>
                </div>
            </div>
        `;
    }
    
    async load() {
        try {
            this.settings = await window.go.main.App.GetSettings();
            this.updateUI();
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    }
    
    updateUI() {
        // Update UI elements with current settings
        if (this.settings.audio) {
            const audio = this.settings.audio;
            this.setInputValue('setting-crossfade', audio.crossfade);
            this.setCheckboxValue('setting-replaygain', audio.replayGain);
            this.setCheckboxValue('setting-gapless', audio.gapless);
        }
        
        if (this.settings.library) {
            const library = this.settings.library;
            this.setCheckboxValue('setting-autoscan', library.autoScan);
        }
        
        if (this.settings.ui) {
            const ui = this.settings.ui;
            this.setSelectValue('setting-theme', ui.theme);
            this.setCheckboxValue('setting-alwaysontop', ui.alwaysOnTop);
        }
    }
    
    setInputValue(id, value) {
        const element = document.getElementById(id);
        if (element) element.value = value;
    }
    
    setCheckboxValue(id, checked) {
        const element = document.getElementById(id);
        if (element) element.checked = checked;
    }
    
    setSelectValue(id, value) {
        const element = document.getElementById(id);
        if (element) element.value = value;
    }
    
    async save() {
        const settings = {
            audio: {
                crossfade: parseFloat(document.getElementById('setting-crossfade').value),
                replayGain: document.getElementById('setting-replaygain').checked,
                gapless: document.getElementById('setting-gapless').checked,
            },
            library: {
                autoScan: document.getElementById('setting-autoscan').checked,
            },
            ui: {
                theme: document.getElementById('setting-theme').value,
                alwaysOnTop: document.getElementById('setting-alwaysontop').checked,
            }
        };
        
        try {
            await window.go.main.App.UpdateSettings(settings);
            alert('Settings saved successfully!');
        } catch (error) {
            console.error('Failed to save settings:', error);
            alert('Failed to save settings: ' + error);
        }
    }
}