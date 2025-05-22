// sw.js - Service Worker for Snake VS PWA

const CACHE_NAME = 'snake-vs-cache-v1';
const urlsToCache = [
  '/', // Assuming your main game HTML is at the root
  '/index.html', // Or whatever your main HTML file is named if not at root
  '/manifest.json',
  // Corrected icon paths to match manifest.json (PNG files)
  '/icons/192x192.png',
  '/icons/512x512.png'
  // Add other critical assets here if needed,
  // e.g., '/js/game.js', '/css/style.css'
  // Ensure all paths are relative to the service worker's location (usually the root).
];

// Install event: open cache and add core files
self.addEventListener('install', (event) => {
  console.log('Service Worker: Installing...');
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('Service Worker: Caching app shell');
        // Using { cache: 'reload' } for addAll to ensure fresh resources are fetched during install
        // This is important if you update assets and the browser has an old version cached by HTTP cache.
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

// Activate event: clean up old caches
self.addEventListener('activate', (event) => {
  console.log('Service Worker: Activating...');
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            console.log('Service Worker: Clearing old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      console.log('Service Worker: Activated successfully');
      return self.clients.claim(); // Take control of all open clients
    })
  );
});

// Fetch event: serve cached content when offline, or fetch from network (Cache-First Strategy)
self.addEventListener('fetch', (event) => {
  // We only want to cache GET requests.
  if (event.request.method !== 'GET') {
    // For non-GET requests, just fetch from the network.
    // Or handle them specifically if your app needs to (e.g., POST requests for a game might not make sense to cache).
    return;
  }

  event.respondWith(
    caches.match(event.request)
      .then((cachedResponse) => {
        if (cachedResponse) {
          // console.log('Service Worker: Serving from cache:', event.request.url);
          return cachedResponse; // Serve from cache if found
        }

        // console.log('Service Worker: Fetching from network:', event.request.url);
        // Not found in cache, fetch from network.
        return fetch(event.request)
          .then((networkResponse) => {
            // Optional: Cache new requests dynamically if they are successful and you want to.
            // This is useful if your game loads additional assets not in the initial urlsToCache.
            // Be careful with this for a game; you might not want to cache everything.
            // if (networkResponse && networkResponse.status === 200 && networkResponse.type === 'basic') {
            //   const responseToCache = networkResponse.clone();
            //   caches.open(CACHE_NAME)
            //     .then(cache => {
            //       console.log('Service Worker: Caching new resource:', event.request.url);
            //       cache.put(event.request, responseToCache);
            //     });
            // }
            return networkResponse;
          })
          .catch((error) => {
            console.error('Service Worker: Fetch failed for:', event.request.url, error);
            // Fallback for failed fetch (e.g., user is offline and resource isn't cached)
            // You could return a custom offline page or a specific fallback asset.
            // For a game, if a critical asset fails and isn't cached, the game might not work.
            // Ensure all critical assets are in urlsToCache.
            // Example: return caches.match('/offline.html');
            // If you don't have a specific offline page, the browser will show its default offline error.
          });
      })
  );
});
