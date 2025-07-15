const CACHE_NAME = 'spirograph-generator-v3-071525@0755'; // Update this version when you change the cache content 
const urlsToCache = [
  './', // For accessing the root
  './index.html',
  './css/style.css',
  './js/app.js',
  './js/lib/Animated_GIF.js',
  './js/lib/Animated_GIF.worker.js',
  './js/lib/NeuQuant.js',
  './manifest.json',
  './icons/s32.png',
  './icons/s192.png',
  './icons/s512.png'
];

// Install event: Cache core assets
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        console.log('Opened cache:', CACHE_NAME);
        // Use {cache: 'reload'} to ensure fresh copies are fetched during install,
        // especially important if you are iterating on these files.
        const requests = urlsToCache.map(url => new Request(url, {cache: 'reload'}));
        return cache.addAll(requests);
      })
      .catch(err => {
        console.error('Failed to open cache or add URLs during install:', err);
      })
  );
  self.skipWaiting(); // Force the waiting service worker to become the active service worker
});

// Activate event: Clean up old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.map(cacheName => {
          if (cacheName !== CACHE_NAME && cacheName.startsWith('spirograph-generator')) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
        console.log('Active cache is:', CACHE_NAME);
        return self.clients.claim(); // Take control of uncontrolled clients immediately
    })
  );
});

// Fetch event: Serve cached content when offline, or fetch from network
self.addEventListener('fetch', event => {
  const requestUrl = new URL(event.request.url);

  // If the request is for a data: URL, do not attempt to handle it with the service worker.
  if (requestUrl.protocol === 'data:') {
    return; // Let the browser handle it directly without calling event.respondWith()
  }

  // For navigation requests (HTML pages), use a cache-first strategy.
  // This ensures the app loads instantly from the cache, even in poor network conditions.
  if (event.request.mode === 'navigate') {
    event.respondWith(
      caches.match(event.request)
        .then(cachedResponse => {
          // Return cached response if found.
          if (cachedResponse) {
            return cachedResponse;
          }
          // If not in cache, fetch from network. This is for the first visit.
          return fetch(event.request)
            .then(networkResponse => {
              if (networkResponse.ok) {
                const responseToCache = networkResponse.clone();
                caches.open(CACHE_NAME).then(cache => {
                  cache.put(event.request, responseToCache);
                });
              }
              return networkResponse;
            })
            .catch(() => {
              // If network fails and it's not in cache, provide the main fallback.
              return caches.match('./index.html');
            });
        })
    );
    return;
  }

  // For other requests (assets like CSS, JS, images), use cache-first strategy
  event.respondWith(
    caches.match(event.request)
      .then(response => {
        if (response) {
          return response; // Serve from cache
        }
        // Not in cache, fetch from network
        return fetch(event.request).then(
          networkResponse => {
            // Check if we received a valid response to cache
            if (networkResponse && networkResponse.status === 200 && networkResponse.type === 'basic') {
              const responseToCache = networkResponse.clone();
              caches.open(CACHE_NAME)
                .then(cache => {
                  cache.put(event.request, responseToCache);
                });
            }
            return networkResponse;
          }
        ).catch(fetchError => {
          console.error('Fetch error for non-navigation request:', fetchError, event.request.url);
          // Optionally, return a placeholder for images or specific error responses
          // For example, if it's an image request:
          // if (event.request.destination === 'image') {
          //   return caches.match('./path/to/placeholder-image.png');
          // }
        });
      })
  );
});