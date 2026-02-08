// SubVault Sort Preference Persistence
// Saves and restores user's sort preference using localStorage

const SORT_STORAGE_KEY = 'subvault-sort';
const VALID_SORT_FIELDS = ['name', 'cost', 'renewal_date', 'status', 'category', 'schedule', 'created_at'];
const VALID_SORT_ORDERS = ['asc', 'desc'];

// Validate sort parameters
function isValidSortPreference(sortBy, order) {
    return VALID_SORT_FIELDS.includes(sortBy) && VALID_SORT_ORDERS.includes(order);
}

// Save sort preference to localStorage
function saveSortPreference(sortBy, order) {
    if (!isValidSortPreference(sortBy, order)) return;
    const preference = { sortBy, order };
    localStorage.setItem(SORT_STORAGE_KEY, JSON.stringify(preference));
}

// Get saved sort preference
function getSortPreference() {
    const stored = localStorage.getItem(SORT_STORAGE_KEY);
    if (stored) {
        try {
            return JSON.parse(stored);
        } catch (e) {
            console.error('Failed to parse sort preference:', e);
            return null;
        }
    }
    return null;
}

// Extract sort params from URL
function extractSortParams(url) {
    try {
        const urlObj = new URL(url, window.location.origin);
        const sortBy = urlObj.searchParams.get('sort');
        const order = urlObj.searchParams.get('order');
        if (sortBy && order) {
            return { sortBy, order };
        }
    } catch (e) {
        console.error('Failed to extract sort params:', e);
    }
    return null;
}

// Apply saved sort preference on page load (local DOM sorting, no HTMX request)
function applySavedSortPreference() {
    const preference = getSortPreference();
    if (!preference) return;

    const subscriptionList = document.getElementById('sub-grid');
    if (!subscriptionList) return;

    // Validate preference before using
    if (!isValidSortPreference(preference.sortBy, preference.order)) return;

    // Use local sorting via toggleSort if available (defined in page script)
    if (typeof toggleSort === 'function') {
        const btn = document.querySelector('.sort-btn[data-sort="' + preference.sortBy + '"]');
        if (btn) {
            btn.dataset.order = preference.order;
            toggleSort(btn);
        }
    }
}

// Listen for HTMX requests to capture sort changes
document.addEventListener('htmx:configRequest', function(event) {
    const path = event.detail.path;

    // Check if this is a sort request to subscriptions API
    if (path && path.includes('/api/subscriptions')) {
        const params = extractSortParams(path);
        if (params) {
            saveSortPreference(params.sortBy, params.order);
        }
    }
});

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    // Apply saved sort preference once HTMX is ready
    if (typeof htmx !== 'undefined') {
        applySavedSortPreference();
    }
});
