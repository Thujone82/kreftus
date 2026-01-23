let CACHE_NAME = 'forecast-v1.0.0-012226@1956';
let STATIC_CACHE = 'forecast-static-v1.0.0';
let DATA_CACHE = 'forecast-data-v1.0.0';

// Initialize cache names from manifest
async function initializeCacheNames() {
    try {
        const response = await fetch('/forecast/manifest.json?t=' + Date.now());
        const manifest = await response.json();
        const version = manifest.version || '1.0.0';
        CACHE_NAME = `forecast-v${version}`;
        STATIC_CACHE = `forecast-static-v${version}`;
        DATA_CACHE = `forecast-data-v${version}`;
    } catch (error) {
        console.error('Error initializing cache names:', error);
    }
}

const STATIC_ASSETS = [
    '/forecast/',
    '/forecast/index.html',
    '/forecast/css/style.css',
    '/forecast/js/app.js',
    '/forecast/js/api.js',
    '/forecast/js/weather.js',
    '/forecast/js/display.js',
    '/forecast/js/utils.js',
    '/forecast/manifest.json',
    '/forecast/icons/light-icon-192.png',
    '/forecast/icons/light-icon-512.png'
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
    event.waitUntil(
        initializeCacheNames().then(() => {
            return caches.open(STATIC_CACHE).then((cache) => {
                return cache.addAll(STATIC_ASSETS);
            });
        })
    );
    self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
    event.waitUntil(
        initializeCacheNames().then(() => {
            return caches.keys().then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => {
                        // Delete all old caches that don't match current version
                        if (cacheName.startsWith('forecast-') && 
                            cacheName !== STATIC_CACHE && 
                            cacheName !== DATA_CACHE) {
                            console.log('Deleting old cache:', cacheName);
                            return caches.delete(cacheName);
                        }
                    })
                );
            });
        }).then(() => {
            // Clear localStorage cache when service worker updates
            // This ensures users get fresh data with new features (like NOAA station data)
            return self.clients.matchAll().then((clients) => {
                clients.forEach((client) => {
                    client.postMessage({
                        type: 'CLEAR_CACHE',
                        reason: 'Service worker updated'
                    });
                });
            });
        }).then(() => {
            return self.clients.claim();
        })
    );
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // Handle API requests with network-first strategy
    if (url.pathname.includes('/api.weather.gov/') || 
        url.pathname.includes('/nominatim.openstreetmap.org/') ||
        url.pathname.includes('/ip-api.com/')) {
        event.respondWith(
            fetch(request)
                .then((response) => {
                    // Clone the response
                    const responseClone = response.clone();
                    // Cache successful responses
                    if (response.status === 200) {
                        caches.open(DATA_CACHE).then((cache) => {
                            cache.put(request, responseClone);
                        });
                    }
                    return response;
                })
                .catch(() => {
                    // Network failed, try cache
                    return caches.match(request).then((response) => {
                        if (response) {
                            return response;
                        }
                        // Return offline response if no cache
                        return new Response(JSON.stringify({ error: 'Offline' }), {
                            headers: { 'Content-Type': 'application/json' }
                        });
                    });
                })
        );
    } else if (url.pathname.endsWith('.js') || url.pathname.endsWith('.html') || url.pathname.endsWith('.css') || url.pathname.endsWith('.json')) {
        // Handle JS, HTML, CSS, and JSON files with network-first strategy to ensure updates
        event.respondWith(
            fetch(request, { cache: 'no-cache' })
                .then((response) => {
                    // Clone the response
                    const responseClone = response.clone();
                    // Cache successful responses
                    if (response.status === 200) {
                        caches.open(STATIC_CACHE).then((cache) => {
                            cache.put(request, responseClone);
                        });
                    }
                    return response;
                })
                .catch(() => {
                    // Network failed, try cache
                    return caches.match(request);
                })
        );
    } else {
        // Handle other static assets with cache-first strategy
        event.respondWith(
            caches.match(request).then((response) => {
                return response || fetch(request);
            })
        );
    }
});

// Listen for messages from the app to check for updates
self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'CHECK_UPDATE') {
        checkForUpdate();
    }
});

// Check for manifest update
async function checkForUpdate() {
    try {
        const response = await fetch('/forecast/manifest.json?t=' + Date.now());
        const manifest = await response.json();
        
        // Get current version from cache name or default
        const currentVersion = CACHE_NAME.match(/v([\d.]+)/)?.[1] || '1.0.0';
        
        if (manifest.version !== currentVersion) {
            // Notify all clients about the update
            const clients = await self.clients.matchAll();
            clients.forEach((client) => {
                client.postMessage({
                    type: 'UPDATE_AVAILABLE',
                    version: manifest.version
                });
            });
        }
    } catch (error) {
        console.error('Error checking for update:', error);
    }
}

