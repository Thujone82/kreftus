// Increment the Timestamp to trigger an update for all users.
const CACHE_NAME = 'habit-tracker-v2-072825@1113';
const ASSETS_TO_CACHE = [
  './',
  './index.html',
  './styles.css',
  './app.js',
  './db.js',
  './manifest.json',
  './icons/32.png',
  './icons/256.png',
  './icons/512.png'
];

// Install service worker and cache assets
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        return cache.addAll(ASSETS_TO_CACHE);
      }),
  );
  // Force the waiting service worker to become the active service worker.
  self.skipWaiting();
});

// Activate and clean up old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.filter(cacheName => {
          return cacheName !== CACHE_NAME;
        }).map(cacheName => {
          return caches.delete(cacheName);
        })
      );
    }),
  );
  // Take control of all clients as soon as the service worker is activated.
  self.clients.claim();
});

// Serve cached content when offline
self.addEventListener('fetch', event => {
  // For navigation requests (e.g., loading the page), try the network first.
  // This ensures users get the latest HTML shell if they are online.
  if (event.request.mode === 'navigate') {
    event.respondWith(
      fetch(event.request).catch(() => {
        // If the network fails, serve the cached index.html.
        return caches.match('./index.html');
      })
    );
    return;
  }

  // For all other requests (CSS, JS, images), use a "cache-first" strategy.
  event.respondWith(
    caches.match(event.request).then(response => {
      return response || fetch(event.request);
    })
  );
});