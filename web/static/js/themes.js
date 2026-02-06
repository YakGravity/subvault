// SubTrackr Theme System - Light/Dark/System
function setThemeMode(mode) {
    localStorage.setItem('subtrackr-theme', mode);
    var theme;
    if (mode === 'system') {
        theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'default';
    } else {
        theme = mode === 'light' ? 'default' : 'dark';
    }
    document.documentElement.setAttribute('data-theme', theme);
    // Save to server
    fetch('/api/settings/theme', { method: 'POST', headers: {'Content-Type': 'application/x-www-form-urlencoded'}, body: 'theme=' + mode });
    // Update active button
    updateThemeButtons(mode);
}

function updateThemeButtons(mode) {
    document.querySelectorAll('.theme-switch-btn').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.mode === mode);
    });
}

// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
    if (localStorage.getItem('subtrackr-theme') === 'system') {
        var theme = this.matches ? 'dark' : 'default';
        document.documentElement.setAttribute('data-theme', theme);
    }
});

// Init active buttons on load
document.addEventListener('DOMContentLoaded', function() {
    var mode = localStorage.getItem('subtrackr-theme') || 'system';
    updateThemeButtons(mode);
});
