// Local-First Service Worker for vBTC Ledger Analysis
const CACHE_VERSION = '112026@0950'; // MMDDYY@HHmm from system clock (Get-Date -Format 'MMddyy@HHmm') — bump when shipping app updates
const CACHE_PREFIX = 'vbtc-ledger-pwa-';
const CACHE_NAME = CACHE_PREFIX + CACHE_VERSION;
const APP_SHELL_URL = './LedgerAnalysis.html';
const MANIFEST_URL = 'ledger-manifest.json';
const PRECACHE_URLS = [
  './LedgerAnalysis.html',
  './ledger-manifest.json',
  './ledger-icons/icon-180.png',
  './ledger-icons/icon-192.png',
  './ledger-icons/icon-512.png',
  './ledger-icons/icon-maskable-512.png'
];

function isLedgerAppAsset(url) {
  const scopePath = new URL('./', self.location.href).pathname;
  if (!url.pathname.startsWith(scopePath)) return false;
  const rel = url.pathname.slice(scopePath.length);
  return (
    rel.toLowerCase() === 'ledgeranalysis.html' ||
    rel === MANIFEST_URL ||
    rel.startsWith('ledger-icons/')
  );
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return Promise.all(
        PRECACHE_URLS.map((url) =>
          cache.add(new Request(url, { cache: 'reload' }))
        )
      );
    }).then(() => {
      // Activate immediately on first install; wait cleanly for user confirmation on upgrades
      if (!self.registration || !self.registration.active) {
        return self.skipWaiting();
      }
    })
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((names) =>
        Promise.all(
          names
            .filter((name) => name.startsWith(CACHE_PREFIX) && name !== CACHE_NAME)
            .map((name) => caches.delete(name))
        )
      )
      .then(() => self.clients.claim())
  );
});

self.addEventListener('message', (event) => {
  const data = event.data || {};
  if (data.type === 'SKIP_WAITING' || data.action === 'skipWaiting') {
    self.skipWaiting();
  } else if (data.type === 'GET_VERSION') {
    const payload = { type: 'VERSION', version: CACHE_VERSION, cacheName: CACHE_NAME };
    if (event.ports && event.ports[0]) {
      event.ports[0].postMessage(payload);
    } else if (event.source) {
      event.source.postMessage(payload);
    }
  }
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;

  let url;
  try {
    url = new URL(request.url);
  } catch (_) {
    return;
  }

  if (url.protocol === 'data:' || url.protocol === 'blob:') return;
  if (url.origin !== self.location.origin) return;

  // Allow explicit update checks to bypass local cache directly to network
  if (request.cache === 'no-store' || url.searchParams.has('__vbtc_ledger_update_check')) {
    return;
  }

  // Strictly scope to Ledger Analysis assets; never hijack /vbtc/ or /vbtc/index.html
  if (!isLedgerAppAsset(url)) {
    return;
  }

  const scopePath = new URL('./', self.location.href).pathname;
  const rel = url.pathname.slice(scopePath.length);

  // Always refresh manifest when online so PWA metadata/icons/id stay current, falling back to cache offline
  if (rel === MANIFEST_URL) {
    event.respondWith(
      caches.open(CACHE_NAME).then((cache) => {
        return fetch(new Request(request, { cache: 'no-cache' }))
          .then((networkResp) => {
            if (networkResp && networkResp.ok) {
              cache.put(request, networkResp.clone());
            }
            return networkResp;
          })
          .catch(() => cache.match(request, { ignoreSearch: true }));
      })
    );
    return;
  }

  // Local-First (Cache-First) strategy: boot instantaneously from local CacheStorage
  event.respondWith(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.match(request, { ignoreSearch: true }).then((cached) => {
        if (cached) return cached;
        if (request.mode === 'navigate') {
          return cache.match(APP_SHELL_URL, { ignoreSearch: true }).then((shellCached) => {
            if (shellCached) return shellCached;
            return fetch(request).then((networkResp) => {
              if (networkResp && networkResp.ok) {
                cache.put(APP_SHELL_URL, networkResp.clone());
              }
              return networkResp;
            });
          });
        }
        return fetch(request).then((networkResp) => {
          if (networkResp && networkResp.ok) {
            cache.put(request, networkResp.clone());
          }
          return networkResp;
        });
      });
    })
  );
});
