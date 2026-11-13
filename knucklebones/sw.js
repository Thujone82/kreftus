const CACHE_NAME = 'knucklebones-v1-111226@2000'; // MMDDYY@HHmm from system clock — bump when shipping app updates

const PRECACHE_URLS = [
  './',
  './index.html',
  './style.css',
  './app.js',
  './manifest.json',
  './die.png',
  './banner-web.png',
  './icons/32x32.png',
  './icons/180x180.png',
  './icons/192x192.png',
  './icons/512x512.png',
  './icons/maskable-512x512.png',
  './icons/favicon.ico'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return Promise.all(
        PRECACHE_URLS.map((url) =>
          cache.add(new Request(url, { cache: 'reload' })).catch((err) => {
            console.warn('[SW] Precache skipped:', url, err);
          })
        )
      );
    }).then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((cacheNames) =>
        Promise.all(
          cacheNames
            .filter((name) => name.startsWith('knucklebones-') && name !== CACHE_NAME)
            .map((name) => caches.delete(name))
        )
      )
      .then(() => self.clients.claim())
      .then(() =>
        self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
          for (const client of clients) {
            client.postMessage({ type: 'SW_UPDATED', version: CACHE_NAME });
          }
        })
      )
  );
});

self.addEventListener('message', (event) => {
  const data = event.data;
  if (!data || typeof data !== 'object') return;
  if (data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  } else if (data.type === 'GET_VERSION' && event.source) {
    event.source.postMessage({ type: 'VERSION', version: CACHE_NAME });
  }
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;

  let url;
  try {
    url = new URL(request.url);
  } catch {
    return;
  }

  if (url.protocol === 'data:' || url.protocol === 'blob:') return;
  if (url.origin !== self.location.origin) return;

  // Navigation requests (app shell): network-first, bypassing HTTP cache so updates appear right away.
  const isNav =
    request.mode === 'navigate' &&
    (url.pathname === '/knucklebones' ||
      url.pathname === '/knucklebones/' ||
      /\/knucklebones\/index\.html$/i.test(url.pathname));

  if (isNav) {
    event.respondWith(
      fetch(request, { cache: 'no-cache' })
        .then((response) => {
          if (response && response.status === 200) {
            const clone = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(request, clone));
          }
          return response;
        })
        .catch(() =>
          caches.match(request).then(
            (hit) => hit || caches.match('./index.html') || caches.match('./')
          )
        )
    );
    return;
  }

  // Code, stylesheets, manifest, and HTML fragments: stale-while-revalidate with HTTP-cache bypass.
  if (/\.(?:js|css|json|html)$/i.test(url.pathname)) {
    event.respondWith(
      caches.open(CACHE_NAME).then(async (cache) => {
        const cached = await cache.match(request);
        const netPromise = fetch(request, { cache: 'no-cache' })
          .then((response) => {
            if (response && response.status === 200) {
              cache.put(request, response.clone());
            }
            return response;
          })
          .catch(() => null);

        if (cached) {
          void netPromise;
          return cached;
        }
        return (await netPromise) || (await caches.match(request));
      })
    );
    return;
  }

  // Images and icons: cache-first with network fallback.
  event.respondWith(
    caches.match(request).then((cached) => {
      if (cached) return cached;
      return fetch(request).then((response) => {
        if (response && response.status === 200 && response.type === 'basic') {
          const clone = response.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(request, clone));
        }
        return response;
      });
    })
  );
});
