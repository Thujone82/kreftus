const CACHE_NAME = 'tc-track-cache-v1-0616@1225'; // Ensure this is updated with current MMDD@HHMM
const urlsToCache = [
    './',
    './index.html',
    './manifest.json',
    './icons/192.png', // Main app icon
    './icons/512.png' // Larger app icon
    // Add other static assets like CSS or JS files if you separate them
];

self.addEventListener('install', event => {
    event.waitUntil(
        // console.log('Service Worker: Install event triggered.'), // Optional: more logging
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
                // DO NOT call self.skipWaiting() here.
                // We want to wait for the user to click the update button.
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
        // If URL parsing fails (e.g. invalid/opaque URL), just let the network handle it
        event.respondWith(fetch(event.request));
        return;
    }

    // Let the browser handle data: and blob: URLs directly. Attempting to
    // respond to these from the service worker with fetch() can result in
    // "FetchEvent.respondWith received an error" messages, so we simply do not
    // intercept them.
    if (requestUrl.protocol === 'data:' || requestUrl.protocol === 'blob:') {
        // Let the browser handle data and blob URLs natively. Intercepting these
        // can lead to "FetchEvent.respondWith received an error" messages in
        // some browsers when the fetch fails. Simply return without calling
        // respondWith so the request bypasses the service worker entirely.
        console.log('Service Worker: bypassing data/blob URL', event.request.url);
        return;
    }

    // For API calls to LiveCoinWatch and Google's Generative Language API,
    // always go to the network. These are typically POST requests or dynamic
    // content that shouldn't be served from a simple cache.
    if (requestUrl.hostname === 'api.livecoinwatch.com' ||
        requestUrl.hostname === 'generativelanguage.googleapis.com') {
        // Pass the request directly to the network
        // console.log('Service Worker: Fetching from network (API call):', event.request.url);
        event.respondWith(fetch(event.request));
        return;
    }

    event.respondWith(
        caches.match(event.request)
            .then(response => {
                // Cache hit - return response
                if (response) {
                    // console.log('Service Worker: Serving from cache:', event.request.url);
                    return response;
                }
                return fetch(event.request).then(
                    // Network request successful, cache it for next time (if it's a GET request)
                    // This part is more for other static assets if you add them.
                    function(response) {
                        if(!response || response.status !== 200 || response.type !== 'basic' || event.request.method !== 'GET') {
                            // console.log('Service Worker: Not caching (not GET, or bad response):', event.request.url);
                            return response;
                        }
                        var responseToCache = response.clone();
                        caches.open(CACHE_NAME)
                            .then(function(cache) {
                                cache.put(event.request, responseToCache);
                                // console.log('Service Worker: Caching new resource:', event.request.url);
                            });
                        return response;
                    }
                );
            })
    );
});

self.addEventListener('activate', event => {
    const cacheWhitelist = [CACHE_NAME];
    // console.log('Service Worker: Activate event triggered. Current cache:', CACHE_NAME); // Optional: more logging
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

self.addEventListener('message', (event) => {
    if (event.data && event.data.action === 'skipWaiting') {
        console.log('Service Worker: Received skipWaiting message. Activating new version.');
        self.skipWaiting();
    }
});
