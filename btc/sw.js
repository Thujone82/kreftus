const CACHE_NAME = 'btc-track-cache-v1';
const urlsToCache = [
    '/',
    'index.html',
    'manifest.json',
    './icons/192x192.png',
    './icons/512x512.png'
    // Add other static assets like CSS or JS files if you separate them
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
    // For API calls, always go to the network.
    // For LiveCoinWatch, their API is POST only for these endpoints,
    // which are typically not cached by service workers by default for GET.
    // So, we'll just let network requests pass through.
    if (event.request.url.startsWith('https://api.livecoinwatch.com')) {
        event.respondWith(fetch(event.request));
        return;
    }

    event.respondWith(
        caches.match(event.request)
            .then(response => {
                // Cache hit - return response
                if (response) {
                    return response;
                }
                return fetch(event.request).then(
                    // Network request successful, cache it for next time (if it's a GET request)
                    // This part is more for other static assets if you add them.
                    function(response) {
                        if(!response || response.status !== 200 || response.type !== 'basic' || event.request.method !== 'GET') {
                            return response;
                        }
                        var responseToCache = response.clone();
                        caches.open(CACHE_NAME)
                            .then(function(cache) {
                                cache.put(event.request, responseToCache);
                            });
                        return response;
                    }
                );
            })
    );
});

self.addEventListener('activate', event => {
    const cacheWhitelist = [CACHE_NAME];
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames.map(cacheName => {
                    if (cacheWhitelist.indexOf(cacheName) === -1) {
                        return caches.delete(cacheName);
                    }
                })
            );
        })
    );
});
