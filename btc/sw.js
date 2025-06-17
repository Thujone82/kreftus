const CACHE_NAME = 'btc-track-cache-v1-0617@1116'; // Ensure this is updated with current MMDD@HHMM
const API_DATA_CACHE_NAME = 'btc-api-data-v1';

// IndexedDB constants for API Key retrieval
const API_KEY_DB_NAME = 'btcAppDB'; // Should match DB name used in index.html
const APP_DATA_STORE_NAME = 'appDataStore'; // Should match store name used in index.html
const CONFIG_IDB_KEY = 'appConfig';      // Key for the main config object in IndexedDB


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

async function getApiKeyFromIndexedDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open(API_KEY_DB_NAME, 1); // Version 1, or match client

        request.onerror = event => {
            console.error('Service Worker: IndexedDB error:', event.target.errorCode);
            reject("Error opening IDB for API key");
        };

        request.onsuccess = event => {
            const db = event.target.result;
            if (!db.objectStoreNames.contains(APP_DATA_STORE_NAME)) {
                console.warn(`Service Worker: Object store ${APP_DATA_STORE_NAME} not found.`);
                db.close();
                resolve(null); // Store doesn't exist yet
                return;
            }
            try {
                const transaction = db.transaction(APP_DATA_STORE_NAME, 'readonly');
                const store = transaction.objectStore(APP_DATA_STORE_NAME);
                const getRequest = store.get(CONFIG_IDB_KEY); // Fetch the whole config object

                getRequest.onsuccess = () => {
                    const configObject = getRequest.result ? getRequest.result.value : null;
                    resolve(configObject ? configObject.apiKey : null); // Extract apiKey
                };
                getRequest.onerror = (event) => {
                    console.error('Service Worker: Error fetching config from IDB store.', event.target.error);
                    resolve(null);
                };
            } catch (e) {
                console.error('Service Worker: Exception during IDB transaction for config.', e);
                resolve(null);
            } finally {
                // db.close(); // Closing might be premature if other operations are queued.
                           // Typically, transactions auto-close.
            }
        };
        // onupgradeneeded is typically handled by the client-side that creates the DB.
        // If the SW is the first to try and open with a new version or non-existent store,
        // it might need its own onupgradeneeded, but it's safer if client manages schema.
    });
}

async function performBackgroundDataUpdate() {
    console.log('Service Worker: Performing background data update...');
    const apiKey = await getApiKeyFromIndexedDB();

    if (!apiKey) {
        console.log('Service Worker: API key not found in IndexedDB. Cannot perform background update.');
        return;
    }

    const API_BASE_URL = "https://api.livecoinwatch.com";
    const COIN_CODE = "BTC";
    const CURRENCY = "USD";

    try {
        const currentDataBody = JSON.stringify({ currency: CURRENCY, code: COIN_CODE, meta: true });
        const currentResponse = await fetch(`${API_BASE_URL}/coins/single`, {
            method: 'POST',
            headers: { "Content-Type": "application/json", "x-api-key": apiKey },
            body: currentDataBody
        });

        if (!currentResponse.ok) throw new Error(`Background Sync: Failed to fetch current data: ${currentResponse.status}`);
        
        // We need to clone the response to be able to read it here and also cache it.
        const currentResponseToCache = currentResponse.clone();
        const cache = await caches.open(API_DATA_CACHE_NAME);
        await cache.put('/api/btc/current-data', currentResponseToCache); // Using a representative key

        console.log('Service Worker: Background data update successful. Current data cached.');
        // Optionally, fetch and cache historical data for a default timeframe as well.
    } catch (error) {
        console.error('Service Worker: Error during background data update:', error);
    }
}

self.addEventListener('periodicsync', (event) => {
    if (event.tag === 'btc-data-update') { // This tag must match the one registered in index.html
        console.log('Service Worker: Periodic sync event received for btc-data-update.');
        event.waitUntil(performBackgroundDataUpdate());
    }
});
