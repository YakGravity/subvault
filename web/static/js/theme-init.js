(function() {
    // Mode: light/dark/system
    var mode = localStorage.getItem('subtrackr-theme') || 'system';
    var isDark;
    if (mode === 'system') {
        isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    } else {
        isDark = mode === 'dark';
    }

    // Palette: default/nord/catppuccin/dracula/rosepine/gruvbox
    var palette = localStorage.getItem('subtrackr-palette') || 'default';
    var theme;
    if (palette === 'default') {
        theme = isDark ? 'dark' : 'default';
    } else {
        theme = palette + '-' + (isDark ? 'dark' : 'light');
    }

    document.documentElement.setAttribute('data-theme', theme);
    document.documentElement.setAttribute('data-mode', isDark ? 'dark' : 'light');

    // Accent color
    var accent = localStorage.getItem('subtrackr-accent') || 'orange';
    document.documentElement.setAttribute('data-accent', accent);

    // Sidebar collapsed
    if (localStorage.getItem('subtrackr-sidebar') === 'collapsed') {
        document.documentElement.setAttribute('data-sidebar', 'collapsed');
    }

})();
