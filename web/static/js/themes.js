// SubTrackr Theme System

// Compute the data-theme value from palette + mode
function computeTheme(palette, isDark) {
    if (palette === 'default') return isDark ? 'dark' : 'default';
    return palette + '-' + (isDark ? 'dark' : 'light');
}

function isDarkMode() {
    var mode = localStorage.getItem('subtrackr-theme') || 'system';
    if (mode === 'system') return window.matchMedia('(prefers-color-scheme: dark)').matches;
    return mode === 'dark';
}

// Set light/dark/system mode
function setThemeMode(mode) {
    localStorage.setItem('subtrackr-theme', mode);
    var isDark;
    if (mode === 'system') {
        isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    } else {
        isDark = mode === 'dark';
    }
    var palette = localStorage.getItem('subtrackr-palette') || 'default';
    document.documentElement.setAttribute('data-theme', computeTheme(palette, isDark));
    document.documentElement.setAttribute('data-mode', isDark ? 'dark' : 'light');
    // Save to server
    fetch('/api/settings/theme', { method: 'POST', headers: {'Content-Type': 'application/x-www-form-urlencoded'}, body: 'theme=' + mode });
    updateThemeButtons(mode);
}

function updateThemeButtons(mode) {
    document.querySelectorAll('.theme-switch-btn').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.mode === mode);
    });
}

// Set theme palette
function setPalette(palette) {
    localStorage.setItem('subtrackr-palette', palette);
    var isDark = isDarkMode();
    document.documentElement.setAttribute('data-theme', computeTheme(palette, isDark));
    updatePaletteOptions(palette);
}

function updatePaletteOptions(palette) {
    document.querySelectorAll('.theme-option').forEach(function(opt) {
        opt.classList.toggle('active', opt.dataset.palette === palette);
    });
}

// Accent color
function setAccentColor(color) {
    localStorage.setItem('subtrackr-accent', color);
    document.documentElement.setAttribute('data-accent', color);
    updateAccentButtons(color);
}

function updateAccentButtons(color) {
    document.querySelectorAll('.accent-option').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.accent === color);
    });
}

// Compact mode
function setCompactMode(enabled) {
    if (enabled) {
        localStorage.setItem('subtrackr-compact', 'true');
        document.documentElement.setAttribute('data-compact', '');
    } else {
        localStorage.removeItem('subtrackr-compact');
        document.documentElement.removeAttribute('data-compact');
    }
}

// Sidebar collapsed
function setSidebarCollapsed(collapsed) {
    if (collapsed) {
        localStorage.setItem('subtrackr-sidebar', 'collapsed');
        document.documentElement.setAttribute('data-sidebar', 'collapsed');
    } else {
        localStorage.removeItem('subtrackr-sidebar');
        document.documentElement.removeAttribute('data-sidebar');
    }
}

// Font size
function setFontSize(size) {
    var map = { small: '13px', normal: '14.5px', large: '16px' };
    localStorage.setItem('subtrackr-fontsize', size);
    document.documentElement.style.fontSize = map[size] || '14.5px';
    document.querySelectorAll('#fontsize-group button').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.val === size);
    });
}

// Default view
function setDefaultView(view) {
    localStorage.setItem('subtrackr-view', view);
    document.querySelectorAll('#view-group button').forEach(function(btn) {
        btn.classList.toggle('active', btn.dataset.val === view);
    });
}

// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
    if (localStorage.getItem('subtrackr-theme') === 'system') {
        var isDark = this.matches;
        var palette = localStorage.getItem('subtrackr-palette') || 'default';
        document.documentElement.setAttribute('data-theme', computeTheme(palette, isDark));
        document.documentElement.setAttribute('data-mode', isDark ? 'dark' : 'light');
    }
});

// Init active buttons on load
document.addEventListener('DOMContentLoaded', function() {
    var mode = localStorage.getItem('subtrackr-theme') || 'system';
    updateThemeButtons(mode);
    var palette = localStorage.getItem('subtrackr-palette') || 'default';
    updatePaletteOptions(palette);
    var accent = localStorage.getItem('subtrackr-accent') || 'orange';
    updateAccentButtons(accent);
});
