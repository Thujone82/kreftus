const CACHE_NAME = 'info2go-v3-121925@1455-cache'; // Updated cache name for v.3
const SW_CONSTANTS = { // Defined here as sw.js doesn't import app.js
    SW_MESSAGES: {
        SKIP_WAITING: 'SKIP_WAITING'
    }
};
const urlsToCache = [
    './', // Represents the current directory (app root, e.g., c:\kreftus\nfo\)
    './index.html',
    './css/style.css',
    './js/app.js',
    './js/ui.js',
    './js/stor.js',
    './js/api.js',
    './js/utils.js',
    './icons/i32.png',
    './icons/i192.png',
    './icons/i512.png',
    './manifest.json'
];

self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(async (cache) => {
                console.log('Opened cache');
                // Create Request objects with 'reload' cache mode to bypass HTTP cache
                const requests = urlsToCache.map(url => new Request(url, { cache: 'reload' }));
                console.log('Attempting to add all URLs to cache with reload strategy:', urlsToCache);
                await cache.addAll(requests);
                console.log('All files added to cache successfully.');
            })
            .catch(error => console.error('Failed to cache files during install:', error))
    );
});

// Helper function to check if we have real internet access (not just WiFi)
// Uses the same API endpoints that the app uses for validation
async function hasRealInternetAccess() {
    // Try lightweight API endpoints to verify real internet connectivity
    // These are the same endpoints used by the app for online validation
    const testEndpoints = [
        {
            url: 'https://openrouter.ai/api/v1/models',
            method: 'GET' // OpenRouter models endpoint (no auth required)
        },
        {
            url: 'https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash',
            method: 'GET' // Google Gemini model endpoint (will fail without key, but confirms internet)
        }
    ];
    
    for (const endpoint of testEndpoints) {
        try {
            // Create an AbortController for timeout
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 3000); // 3 second timeout
            
            const response = await fetch(endpoint.url, {
                method: endpoint.method,
                cache: 'no-store',
                signal: controller.signal
            });
            
            clearTimeout(timeoutId);
            
            // If we get any response (even 401/403/404), we have internet
            // Status 0 typically means network error (no internet)
            if (response.status !== 0) {
                return true;
            }
        } catch (error) {
            // Network error, timeout, or abort - no real internet
            // Continue to next endpoint
            continue;
        }
    }
    return false;
}

self.addEventListener('fetch', event => {
    // Use network-first strategy for HTML files to ensure updates are picked up
    if (event.request.url.includes('index.html') || event.request.url.endsWith('/')) {
        event.respondWith(
            fetch(event.request, { cache: 'no-store' })
                .then(response => {
                    // Update cache with fresh response
                    const responseClone = response.clone();
                    caches.open(CACHE_NAME).then(cache => {
                        cache.put(event.request, responseClone);
                    });
                    return response;
                })
                .catch(async () => {
                    // Network fetch failed - check if we have real internet access
                    const hasInternet = await hasRealInternetAccess();
                    if (hasInternet) {
                        // We have internet but the HTML fetch failed - might be temporary
                        // Try cache as fallback, but log the issue
                        console.warn('HTML fetch failed but internet is available, serving from cache');
                        return caches.match(event.request);
                    } else {
                        // No real internet (WiFi but no internet scenario) - serve from cache
                        console.log('No internet access detected, serving HTML from cache');
                        return caches.match(event.request);
                    }
                })
        );
    } else {
        // Use cache-first for other resources
        event.respondWith(
            caches.match(event.request)
                .then(response => response || fetch(event.request))
        );
    }
});

self.addEventListener('message', event => {
    if (event.data && event.data.type === SW_CONSTANTS.SW_MESSAGES.SKIP_WAITING) {
        console.log('Service Worker: SKIP_WAITING message received, calling skipWaiting().');
        self.skipWaiting();
    }
});

self.addEventListener('activate', event => {
    console.log('Service Worker: Activating new version.');
    const cacheWhitelist = [CACHE_NAME]; // Add your current cache name here
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames.map(cacheName => {
                    if (cacheWhitelist.indexOf(cacheName) === -1) {
                        console.log('Service Worker: Deleting old cache', cacheName);
                        return caches.delete(cacheName);
                    }
                })
            );
        })
    );
});
