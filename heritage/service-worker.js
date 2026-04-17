// PDX Heritage Trees service worker.
//
// Caching strategy:
//   - Navigation requests for /heritage/ and /heritage/index.html: network-first,
//     fall back to cache. Avoids the stale-shell problem where users on the PWA
//     start_url never pick up a new UI even after the SW updates.
//   - JS / CSS / JSON / SVG / HTML assets: stale-while-revalidate.
//   - Tree snapshot (data/trees.json): network-first (we *want* updates to the
//     list whenever we can get them; cache is a fallback for offline use).
//   - Leaflet CDN (unpkg.com/leaflet@...): stale-while-revalidate, so the
//     map library stays available offline after the first visit.
//   - CARTO basemap tiles (basemaps.cartocdn.com): cache-first with a fetch
//     fallback. Tiles are immutable for a given z/x/y so cache-first is ideal,
//     and it lets the map render offline wherever the user has already panned.
//   - Everything else (images, icons, manifest): cache-first.
//
// IndexedDB is NEVER touched by the service worker, so installing an app update
// cannot wipe a user's Found marks or notes.

const VERSION = '1.1.4';
const STATIC_CACHE = `heritage-static-v${VERSION}`;
const DATA_CACHE   = `heritage-data-v${VERSION}`;
const TILE_CACHE   = `heritage-tiles-v1`;    // tile URLs are versionless, so keep across app bumps
const VENDOR_CACHE = `heritage-vendor-v1`;   // leaflet CDN - not worth re-downloading on every app bump

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
    '/heritage/js/found.js',
    '/heritage/js/ui.js',
    '/heritage/js/sw-register.js',
    '/heritage/icons/icon-192.png',
    '/heritage/icons/icon-512.png',
    '/heritage/icons/PDXTrees.png'
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
    const KEEP = new Set([STATIC_CACHE, DATA_CACHE, TILE_CACHE, VENDOR_CACHE]);
    event.waitUntil(
        caches.keys().then((names) =>
            Promise.all(
                names
                    .filter((n) => n.startsWith('heritage-') && !KEEP.has(n))
                    .map((n) => caches.delete(n))
            )
        ).then(() => self.clients.claim())
    );
});

self.addEventListener('fetch', (event) => {
    const { request } = event;
    if (request.method !== 'GET') return;
    const url = new URL(request.url);

    // Cross-origin: CARTO tile host, Leaflet CDN, anything else (Wikipedia,
    // Nominatim, Google walking-directions deep links) passes through.
    if (url.origin !== self.location.origin) {
        if (/^https:\/\/[a-d]\.basemaps\.cartocdn\.com\//i.test(url.href)) {
            event.respondWith(cacheFirst(request, TILE_CACHE));
            return;
        }
        if (/^https:\/\/unpkg\.com\/leaflet@/i.test(url.href)) {
            event.respondWith(staleWhileRevalidate(request, VENDOR_CACHE));
            return;
        }
        return;
    }

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

function cacheFirst(request, cacheName) {
    return caches.match(request).then((hit) => {
        if (hit) return hit;
        return fetch(request).then((response) => {
            if (response && (response.status === 200 || response.type === 'opaque')) {
                const clone = response.clone();
                caches.open(cacheName).then((cache) => cache.put(request, clone));
            }
            return response;
        }).catch(() => hit || offlineFallback(request));
    });
}

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
