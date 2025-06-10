const CACHE_NAME = 'btc-track-cache-v1';
const urlsToCache = [
    '/',
    'index.html',
    'manifest.json',
    './icons/192.png',
    './icons/512.png'
    // Add other static assets like CSS or JS files if you separate them
];

self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                console.log('Service Worker: Caching app shell');
                // Ensure fresh resources are fetched during install, bypassing HTTP cache.
                const cachePromises = urlsToCache.map(urlToCache => {
                    return cache.add(new Request(urlToCache, { cache: 'reload' }));
                });
                return Promise.all(cachePromises);
            })
            .then(() => {
                console.log('Service Worker: App shell cached successfully');
                return self.skipWaiting(); // Activate worker immediately
            })
            .catch((error) => {
                console.error('Service Worker: Caching failed', error);
            })
    );
});

self.addEventListener('fetch', event => {
    let requestUrl;
    try {
        requestUrl = new URL(event.request.url);
    } catch (e) {
        event.respondWith(fetch(event.request));
        return;
    }

    if (requestUrl.protocol === 'data:' || requestUrl.protocol === 'blob:') {
        event.respondWith(fetch(event.request));
        return;
    }

    // For API calls, always go to the network.
    // For LiveCoinWatch, their API is POST only for these endpoints,
    // which are typically not cached by service workers by default for GET.
    if (requestUrl.hostname === 'api.livecoinwatch.com') {
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
                    return null; // Explicitly return null for paths that don't delete
                })
            );
        }).then(() => {
            return self.clients.claim(); // Take control of all open clients
        })
    );
});
