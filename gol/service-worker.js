const CACHE_NAME = 'gameoflife-cache-v3';
const MANIFEST_URL = './manifest.json';

const URLS_TO_CACHE = [
  './',
  './index.html',
  './manifest.json',
  './service-worker.js',
  './icons/icon-192.png',
  './icons/icon-512.png'
];

// Store the last known manifest content for comparison
let lastManifestContent = null;

// Function to check for manifest changes
async function checkForUpdates() {
  try {
    const response = await fetch(MANIFEST_URL + '?t=' + Date.now());
    const manifestContent = await response.text();
    
    if (lastManifestContent && lastManifestContent !== manifestContent) {
      console.log('Manifest changed - update available');
      // Notify all clients about the update
      const clients = await self.clients.matchAll();
      clients.forEach(client => {
        client.postMessage({
          type: 'UPDATE_AVAILABLE',
          message: 'A new version is available. Refresh to update.'
        });
      });
    }
    
    lastManifestContent = manifestContent;
  } catch (error) {
    console.log('Failed to check for updates:', error);
  }
}

self.addEventListener('install', event => {
  console.log('Service Worker installing...');
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(URLS_TO_CACHE))
      .then(() => {
        // Check for updates immediately after install
        return checkForUpdates();
      })
  );
  // Skip waiting to activate immediately
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  console.log('Service Worker activating...');
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.map(cacheName => {
          if (cacheName !== CACHE_NAME) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      // Take control of all clients immediately
      return self.clients.claim();
    })
  );
});

self.addEventListener('fetch', event => {
  // For manifest, always check network first for updates
  if (event.request.url.includes('manifest.json')) {
    event.respondWith(
      fetch(event.request)
        .then(response => {
          // Check for updates when manifest is fetched
          checkForUpdates();
          return response;
        })
        .catch(() => {
          // Fallback to cache if network fails
          return caches.match(event.request);
        })
    );
    return;
  }

  // For other resources, use cache-first strategy
  event.respondWith(
    caches.match(event.request)
      .then(response => {
        if (response) {
          return response;
        }
        return fetch(event.request).then(fetchResponse => {
          // Don't cache everything, just cache successful responses
          if (fetchResponse.status === 200) {
            const responseClone = fetchResponse.clone();
            caches.open(CACHE_NAME).then(cache => {
              cache.put(event.request, responseClone);
            });
          }
          return fetchResponse;
        });
      })
  );
});

// Periodic update checking (every 5 minutes)
setInterval(checkForUpdates, 5 * 60 * 1000);

