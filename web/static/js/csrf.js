// CSRF token management for HTMX and fetch requests
(function() {
    'use strict';

    function getToken() {
        var meta = document.querySelector('meta[name="csrf-token"]');
        return meta ? meta.getAttribute('content') : '';
    }

    function setToken(token) {
        var meta = document.querySelector('meta[name="csrf-token"]');
        if (meta && token) {
            meta.setAttribute('content', token);
        }
    }

    var MUTATING_METHODS = ['POST', 'PUT', 'DELETE', 'PATCH'];

    // HTMX: inject CSRF token header on mutating requests
    document.addEventListener('htmx:configRequest', function(e) {
        var method = (e.detail.verb || '').toUpperCase();
        if (MUTATING_METHODS.indexOf(method) !== -1) {
            e.detail.headers['X-CSRF-Token'] = getToken();
        }
    });

    // HTMX: update token from response header after each request
    document.addEventListener('htmx:afterRequest', function(e) {
        var newToken = e.detail.xhr && e.detail.xhr.getResponseHeader('X-CSRF-Token');
        if (newToken) {
            setToken(newToken);
        }
    });

    // Override fetch to automatically include CSRF token
    var originalFetch = window.fetch;
    window.fetch = function(input, init) {
        init = init || {};
        var method = (init.method || 'GET').toUpperCase();
        if (MUTATING_METHODS.indexOf(method) !== -1) {
            init.headers = init.headers || {};
            if (init.headers instanceof Headers) {
                if (!init.headers.has('X-CSRF-Token')) {
                    init.headers.set('X-CSRF-Token', getToken());
                }
            } else {
                if (!init.headers['X-CSRF-Token']) {
                    init.headers['X-CSRF-Token'] = getToken();
                }
            }
        }
        return originalFetch.call(this, input, init);
    };
})();
