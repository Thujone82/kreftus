const CACHE_NAME = 'info2go-v1-cache';
const urlsToCache = [
    '../', // Represents the parent directory (app root, e.g., c:\kreftus\nfo\)
    '../index.html',
    '../css/style.css',
    'app.js',       // Already in the same js/ directory as sw.js
    'ui.js',        // Already in the same js/ directory as sw.js
    'stor.js',      // Already in the same js/ directory as sw.js
    'api.js',       // Already in the same js/ directory as sw.js
    'utils.js',     // Already in the same js/ directory as sw.js
    '../icons/i32.png',
    '../icons/i192.png',
    '../icons/i512.png',
    '../manifest.json'
];

self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => {
                console.log('Opened cache');
                return cache.addAll(urlsToCache);
            })
    );
});

self.addEventListener('fetch', event => {
    event.respondWith(
        caches.match(event.request)
            .then(response => response || fetch(event.request))
    );
});