// PDX Heritage Trees service worker.
//
// Caching strategy:
//   - Navigation requests for /heritage/ and /heritage/index.html: network-first,
//     fall back to cache. Avoids the stale-shell problem where users on the PWA
//     start_url never pick up a new UI even after the SW updates.
//   - JS / CSS / JSON / SVG / HTML assets: stale-while-revalidate.
//   - Tree snapshot (data/trees.json): network-first (we *want* updates to the
//     list whenever we can get them; cache is a fallback for offline use).
//   - Everything else (images, icons, manifest): cache-first.
//
// IndexedDB is NEVER touched by the service worker, so installing an app update
// cannot wipe a user's Found marks or notes.

const VERSION = '1.0.0';
const STATIC_CACHE = `heritage-static-v${VERSION}`;
const DATA_CACHE   = `heritage-data-v${VERSION}`;

const STATIC_ASSETS = [
    '/heritage/',
    '/heritage/index.html',
    '/heritage/manifest.json',
    '/heritage/css/styles.css',
    '/heritage/js/app.js',
    '/heritage/js/db.js',
    '/heritage/js/wiki.js',
    '/heritage/js/sync.js',
    '/heritage/js/geocode.js',
    '/heritage/js/map.js',
    '/heritage/js/nearby.js',
    '/heritage/js/ui.js',
    '/heritage/js/sw-register.js',
    '/heritage/icons/icon.svg',
    '/heritage/icons/icon-192.svg',
    '/heritage/icons/icon-512.svg'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(STATIC_CACHE).then((cache) =>
            // Don't fail the whole install if one asset is missing.
            Promise.all(STATIC_ASSETS.map((u) =>
                cache.add(new Request(u, { cache: 'reload' })).catch(() => null)
            ))
        )
    );
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((names) =>
            Promise.all(
                names
                    .filter((n) => n.startsWith('heritage-') && n !== STATIC_CACHE && n !== DATA_CACHE)
                    .map((n) => caches.delete(n))
            )
        ).then(() => self.clients.claim())
    );
});

self.addEventListener('fetch', (event) => {
    const { request } = event;
    if (request.method !== 'GET') return;
    const url = new URL(request.url);

    // Same-origin only; let cross-origin requests (Google Maps, Wikipedia) pass through
    // without going through the SW cache.
    if (url.origin !== self.location.origin) return;

    const isNav =
        request.mode === 'navigate' &&
        (url.pathname === '/heritage/' ||
         url.pathname === '/heritage' ||
         /\/heritage\/index\.html$/i.test(url.pathname));

    if (isNav) {
        event.respondWith(networkFirst(request, STATIC_CACHE));
        return;
    }

    // trees.json - network-first so updates flow through when online.
    if (/\/heritage\/data\/trees\.json$/i.test(url.pathname)) {
        event.respondWith(networkFirst(request, DATA_CACHE));
        return;
    }

    // JS / CSS / JSON / SVG / HTML: stale-while-revalidate.
    if (/\.(?:js|css|json|svg|html)$/i.test(url.pathname)) {
        event.respondWith(staleWhileRevalidate(request, STATIC_CACHE));
        return;
    }

    // Default - cache-first (icons, images).
    event.respondWith(
        caches.match(request).then((hit) => hit || fetch(request))
    );
});

function networkFirst(request, cacheName) {
    return fetch(request, { cache: 'no-cache' })
        .then((response) => {
            if (response && response.status === 200) {
                const clone = response.clone();
                caches.open(cacheName).then((cache) => cache.put(request, clone));
            }
            return response;
        })
        .catch(() => caches.match(request).then((hit) => hit || offlineFallback(request)));
}

function staleWhileRevalidate(request, cacheName) {
    return caches.match(request).then((cached) => {
        const fetchPromise = fetch(request, { cache: 'no-cache' })
            .then((response) => {
                if (response && response.status === 200) {
                    const clone = response.clone();
                    caches.open(cacheName).then((cache) => cache.put(request, clone));
                }
                return response;
            })
            .catch(() => null);
        if (cached) {
            void fetchPromise;
            return cached;
        }
        return fetchPromise.then((resp) => resp || offlineFallback(request));
    });
}

function offlineFallback(request) {
    if (request.destination === 'document') {
        return caches.match('/heritage/index.html');
    }
    return new Response('', { status: 504, statusText: 'Offline' });
}

self.addEventListener('message', (event) => {
    const data = event.data || {};
    if (data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    } else if (data.type === 'GET_VERSION' && event.source) {
        event.source.postMessage({ type: 'VERSION', version: VERSION });
    } else if (data.type === 'CHECK_UPDATE') {
        checkForUpdate();
    }
});

async function checkForUpdate() {
    try {
        const resp = await fetch('/heritage/manifest.json?t=' + Date.now(), { cache: 'no-store' });
        if (!resp || !resp.ok) return;
        const manifest = await resp.json();
        if (manifest.version && manifest.version !== VERSION) {
            const clients = await self.clients.matchAll();
            clients.forEach((c) => c.postMessage({ type: 'UPDATE_AVAILABLE', version: manifest.version }));
        }
    } catch (err) {
        // swallow - next check will retry
    }
}
