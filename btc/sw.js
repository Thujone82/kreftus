const CACHE_NAME = 'btc-track-cache-v1-0618@0718'; // Ensure this is updated with current MMDD@HHMM
const API_DATA_CACHE_NAME = 'btc-api-data-v1';

// IndexedDB constants for API Key retrieval
const API_KEY_DB_NAME = 'btcAppDB'; // Should match DB name used in index.html
const APP_DATA_STORE_NAME = 'appDataStore'; // Should match store name used in index.html
const CONFIG_IDB_KEY = 'appConfig'; // Key for the main config object in IndexedDB
const CURRENT_DATA_IDB_KEY = 'currentDataCache'; // Key for current data in IndexedDB


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

const DB_VERSION = 2; // Ensure this matches the version used in index.html that creates/upgrades the DB

async function openSwDb() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open(API_KEY_DB_NAME, DB_VERSION);
        request.onerror = event => {
            console.error('SW: IndexedDB error in openSwDb:', event.target.error);
            reject("Error opening IDB in SW");
        };
        request.onsuccess = event => {
            resolve(event.target.result);
        };
        // No onupgradeneeded here; client (index.html) handles schema creation and upgrades.
    });
}

async function getApiKeyFromIndexedDB() {
    let db;
    try {
        db = await openSwDb();
        return new Promise((resolve, rejectInner) => { // Renamed reject to avoid conflict
            if (!db.objectStoreNames.contains(APP_DATA_STORE_NAME)) {
                console.warn(`SW: Object store ${APP_DATA_STORE_NAME} not found in getApiKey.`);
                resolve(null); // Store doesn't exist yet
                return;
            }
            const transaction = db.transaction(APP_DATA_STORE_NAME, 'readonly');
            const store = transaction.objectStore(APP_DATA_STORE_NAME);
            const getRequest = store.get(CONFIG_IDB_KEY);

            getRequest.onsuccess = () => {
                const configObject = getRequest.result ? getRequest.result.value : null;
                resolve(configObject ? configObject.apiKey : null);
            };
            getRequest.onerror = (event) => {
                console.error('SW: Error fetching config from IDB store in getApiKey.', event.target.error);
                resolve(null);
            };
        });
    } catch (error) {
        console.error('SW: Error in getApiKeyFromIndexedDB (outer):', error);
        return null;
    } finally {
        if (db) {
            db.close();
        }
    }
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
    let db;
    try {
        const currentDataBody = JSON.stringify({ currency: CURRENCY, code: COIN_CODE, meta: true });
        const currentResponse = await fetch(`${API_BASE_URL}/coins/single`, {
            method: 'POST',
            headers: { "Content-Type": "application/json", "x-api-key": apiKey },
            body: currentDataBody
        });
        if (!currentResponse.ok) throw new Error(`SW Background Sync: Failed to fetch current data: ${currentResponse.status}`);
        
        const currentData = await currentResponse.json();

        // Update Cache Storage
        const responseToCache = new Response(JSON.stringify(currentData), {
            headers: {
                'Content-Type': 'application/json',
                'Date': new Date().toUTCString() // Add a Date header for client-side freshness check
            }
        });
        const cache = await caches.open(API_DATA_CACHE_NAME);
        await cache.put('/api/btc/current-data', responseToCache);
        console.log('Service Worker: Current data cached in Cache Storage.');

        // Update IndexedDB
        db = await openSwDb();
        const transaction = db.transaction(APP_DATA_STORE_NAME, 'readwrite');
        const store = transaction.objectStore(APP_DATA_STORE_NAME);

        const getConfigRequest = store.get(CONFIG_IDB_KEY);
        const configResult = await new Promise((resolve, reject) => { getRequest.onsuccess = () => resolve(getRequest.result); getRequest.onerror = event => reject(event.target.error); });
        let configObject = configResult ? configResult.value : {};
        configObject.lastFetchedCurrentDataTimestamp = Date.now();

        store.put({ id: CONFIG_IDB_KEY, value: configObject });
        store.put({ id: CURRENT_DATA_IDB_KEY, value: currentData });

        await new Promise((resolve, reject) => { transaction.oncomplete = resolve; transaction.onerror = event => reject(event.target.error); });
        console.log('Service Worker: Successfully updated currentDataCache and appConfig timestamp in IndexedDB.');

    } catch (error) {
        console.error('Service Worker: Error during background data update:', error);
    } finally {
        if (db) {
            db.close();
        }
    }
}

self.addEventListener('periodicsync', (event) => {
    if (event.tag === 'btc-data-update') { // This tag must match the one registered in index.html
        console.log('Service Worker: Periodic sync event received for btc-data-update.');
        event.waitUntil(performBackgroundDataUpdate());
    }
});
