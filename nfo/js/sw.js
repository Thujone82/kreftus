const CACHE_NAME = 'info2go-v1-cache';
const urlsToCache = [
    '/',
    '/index.html',
    '/css/style.css',
    '/js/app.js',
    '/js/ui.js',
    '/js/store.js',
    '/js/api.js',
    '/js/utils.js',
    '/icons/i32.png',
    '/icons/i192.png',
    '/icons/i512.png',
    '/manifest.json'
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