const CACHE_NAME = 'info2go-061524@0801-cache'; // KKREFT 
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

self.addEventListener('fetch', event => {
    event.respondWith(
        caches.match(event.request)
            .then(response => response || fetch(event.request))
    );
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