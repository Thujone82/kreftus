const CACHE_NAME = 'spirograph-generator-v2-062825@2031'; 
const urlsToCache = [
  './', // For accessing the root
  './index.html',
  './css/style.css',
  './js/app.js',
  './manifest.json',
  './icons/s192.png', // Ensure this path is correct and file exists
  './icons/s512.png'  // Ensure this path is correct and file exists
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

  // For navigation requests (HTML pages), try network first, then cache.
  if (event.request.mode === 'navigate') {
    event.respondWith(
      fetch(event.request)
        .then(response => {
          // If successful, cache it and return
          if (response.ok) {
            const responseToCache = response.clone();
            caches.open(CACHE_NAME).then(cache => {
              cache.put(event.request, responseToCache);
            });
          }
          return response;
        })
        .catch(() => {
          // If network fails, try to serve from cache
          return caches.match(event.request)
            .then(cachedResponse => {
              return cachedResponse || caches.match('./index.html'); // Fallback to cached index.html
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