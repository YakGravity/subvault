(function() {
    var saved = localStorage.getItem('subtrackr-theme') || 'system';
    var theme;
    if (saved === 'system') {
        theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'default';
    } else {
        theme = saved === 'light' ? 'default' : 'dark';
    }
    document.documentElement.setAttribute('data-theme', theme);
})();
