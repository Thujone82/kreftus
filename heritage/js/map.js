// Leaflet-based map integration for PDX Heritage Trees.
//
// Basemap: CARTO Voyager raster tiles (OpenStreetMap data, CARTO-styled).
// CARTO explicitly permits free use for personal projects and hobby apps, and
// no API key is required. Nominatim is still used for any rare in-browser
// geocoding fallback.
//
// Marker colors:
//   Found    -> forest green
//   Not found-> amber
//   Removed  -> gray
//
// Camera logic on boot:
//   - If user is within 20 miles of Portland: fit bounds of (user, nearest tree)
//     with small padding so both are visible.
//   - Otherwise: fit bounds of all trees centered on Portland.
//
// User location:
//   - Press-and-hold (~600 ms) on empty map tiles asks to refresh location.
//   - After a successful initial fix, watchPosition keeps the blue dot updated
//     while the tab is visible (throttled by time + distance).

(function (global) {
    'use strict';

    const PORTLAND = { lat: 45.5152, lng: -122.6784 };
    const MI_20_METERS = 32186.88;

    const COLOR_FOUND   = '#2e7d32';
    const COLOR_NOT     = '#d4a24a';
    const COLOR_REMOVED = '#7a7a7a';
    const COLOR_STROKE  = '#1f3b2a';

    function walkingDirectionsUrl(lat, lng) {
        return `https://www.google.com/maps/dir/?api=1&travelmode=walking&destination=${lat},${lng}`;
    }

    // CARTO Voyager: muted, natural palette that matches the PNW/woodsy theme
    // and keeps colored markers legible. Allowed for hobby/personal apps per
    // CARTO's basemap usage terms.
    const TILE_URL = 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png';
    const TILE_SUBDOMAINS = 'abcd';
    const TILE_MAX_ZOOM = 20;
    const TILE_ATTRIBUTION =
        '&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noopener">OpenStreetMap</a> contributors ' +
        '&copy; <a href="https://carto.com/attributions" target="_blank" rel="noopener">CARTO</a>';

    let map = null;
    let tileLayer = null;
    let markers = new Map();    // id -> L.CircleMarker
    let userLatLng = null;      // {lat, lng} or null
    let userMarker = null;
    let onTreeUpdateCallback = null;
    let openTreeId = null;

    let geoWatchId = null;
    let hadLiveWatchPermission = false;
    let lastWatchEmitMs = 0;
    let lastWatchEmittedLatLng = null;

    const LIVE_WATCH_MIN_MS = 3500;
    const LIVE_WATCH_MIN_MOVE_M = 7;

    const LONG_PRESS_MS = 600;
    const LONG_PRESS_MOVE_PX = 14;
    let longPressTimer = null;
    let longPressPointerId = null;
    let longPressStartX = 0;
    let longPressStartY = 0;

    // ---------------------------------------------------------------------
    // Init
    // ---------------------------------------------------------------------

    async function init(containerId) {
        if (!global.L) {
            throw new Error('Leaflet is not loaded.');
        }
        const container = document.getElementById(containerId);
        if (!container) throw new Error(`Map container #${containerId} not found.`);

        map = L.map(container, {
            center: [PORTLAND.lat, PORTLAND.lng],
            zoom: 12,
            zoomControl: true,
            attributionControl: true,
            worldCopyJump: false
            // Note: SVG renderer (default) - canvas-rendered circleMarkers had
            // flaky click/popup interactions with a bound tooltip. With ~400
            // pins SVG performance is fine on any modern browser.
        });
        // Place the zoom control in the bottom-left so the top-right gear button
        // and the bottom action bar keep their real estate.
        map.zoomControl.setPosition('bottomleft');

        tileLayer = L.tileLayer(TILE_URL, {
            subdomains: TILE_SUBDOMAINS,
            maxZoom: TILE_MAX_ZOOM,
            attribution: TILE_ATTRIBUTION,
            crossOrigin: true
        }).addTo(map);

        map.on('popupclose', () => { openTreeId = null; });
        wireMapLongPressLocationRefresh();
        wireVisibilityForLiveLocation();
        return map;
    }

    function getMap() { return map; }

    // ---------------------------------------------------------------------
    // User location
    // ---------------------------------------------------------------------

    // Two-stage location lookup:
    //   1. High-accuracy first (GPS on mobile, WiFi + sensors on laptops,
    //      forces a fresh fix via maximumAge: 0). Gets us a device-grade
    //      position when it's available.
    //   2. If the high-accuracy attempt times out or errors (but NOT if the
    //      user denied permission - no point re-asking), fall back to a
    //      fast coarse fix with a generous cache window. This is the IP /
    //      WiFi-triangulated estimate we were using before.
    // The resolved object includes `accuracy` (meters) and `highAccuracy`
    // (boolean) so callers can decide whether to show an accuracy ring.
    function getUserLocation(totalTimeoutMs) {
        return new Promise((resolve) => {
            if (!navigator.geolocation) return resolve(null);

            const total = Math.max(6000, totalTimeoutMs || 14000);
            const preciseMs = Math.min(9000, Math.floor(total * 0.65));
            const fallbackMs = Math.max(3000, total - preciseMs);

            const accept = (pos, highAccuracy) => {
                userLatLng = {
                    lat: pos.coords.latitude,
                    lng: pos.coords.longitude,
                    accuracy: pos.coords.accuracy,
                    highAccuracy: !!highAccuracy
                };
                resolve(userLatLng);
            };

            navigator.geolocation.getCurrentPosition(
                (pos) => accept(pos, true),
                (err) => {
                    if (err && err.code === 1 /* PERMISSION_DENIED */) {
                        return resolve(null);
                    }
                    navigator.geolocation.getCurrentPosition(
                        (pos) => accept(pos, false),
                        () => resolve(null),
                        { timeout: fallbackMs, maximumAge: 600000, enableHighAccuracy: false }
                    );
                },
                { timeout: preciseMs, maximumAge: 0, enableHighAccuracy: true }
            );
        });
    }

    function getCachedUserLocation() { return userLatLng; }

    function placeUserMarker() {
        if (!map || !userLatLng) return;
        if (userMarker) { userMarker.remove(); userMarker = null; }
        userMarker = L.circleMarker([userLatLng.lat, userLatLng.lng], {
            radius: 7,
            fillColor: '#4a7c8a',
            fillOpacity: 0.95,
            color: '#ffffff',
            weight: 2,
            pane: 'markerPane'
        }).addTo(map);
        userMarker.bindTooltip('Your location', { direction: 'top', offset: [0, -6] });
        if (typeof userMarker.bringToFront === 'function') {
            try { userMarker.bringToFront(); } catch (e) { /* ignore */ }
        }
    }

    function shouldAllowMapLongPressTarget(target) {
        if (!target || !target.closest) return false;
        if (!target.closest('#map')) return false;
        if (target.closest('.leaflet-control-container')) return false;
        if (target.closest('.leaflet-popup')) return false;
        // Tree markers, user dot, etc. — long-press empty map / tiles only.
        if (target.closest('.leaflet-interactive')) return false;
        return true;
    }

    function clearLongPressTimer() {
        if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
        }
        longPressPointerId = null;
    }

    function wireMapLongPressLocationRefresh() {
        if (!map) return;
        const el = map.getContainer();

        const onPointerDown = (e) => {
            if (e.pointerType === 'mouse' && e.button !== 0) return;
            if (!shouldAllowMapLongPressTarget(e.target)) return;
            longPressPointerId = e.pointerId;
            longPressStartX = e.clientX;
            longPressStartY = e.clientY;
            clearLongPressTimer();
            longPressTimer = setTimeout(() => {
                longPressTimer = null;
                longPressPointerId = null;
                void promptAndRefreshUserLocation();
            }, LONG_PRESS_MS);
        };

        const onPointerMove = (e) => {
            if (longPressPointerId == null || e.pointerId !== longPressPointerId || !longPressTimer) return;
            const dx = e.clientX - longPressStartX;
            const dy = e.clientY - longPressStartY;
            if (dx * dx + dy * dy > LONG_PRESS_MOVE_PX * LONG_PRESS_MOVE_PX) {
                clearLongPressTimer();
            }
        };

        const onPointerEnd = (e) => {
            if (longPressPointerId != null && e.pointerId === longPressPointerId) {
                clearLongPressTimer();
            }
        };

        el.addEventListener('pointerdown', onPointerDown);
        el.addEventListener('pointermove', onPointerMove);
        el.addEventListener('pointerup', onPointerEnd);
        el.addEventListener('pointercancel', onPointerEnd);
    }

    async function promptAndRefreshUserLocation() {
        const msg = 'Update your location on the map?\n\n' +
            'The blue dot will move to your current position.';
        if (!window.confirm(msg)) return;

        const u = await getUserLocation(22000);
        if (u) {
            placeUserMarker();
            startLiveLocationUpdates();
            if (global.HeritageUI && typeof global.HeritageUI.toast === 'function') {
                global.HeritageUI.toast('Location updated.', 2600);
            }
        } else if (global.HeritageUI && typeof global.HeritageUI.toast === 'function') {
            global.HeritageUI.toast(
                'Could not read your location. Check browser permissions and try again.',
                4500
            );
        }
    }

    function maybeApplyWatchPosition(pos) {
        const now = Date.now();
        const cand = {
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
            accuracy: pos.coords.accuracy,
            highAccuracy: !!(pos.coords.accuracy != null && pos.coords.accuracy <= 80)
        };
        if (lastWatchEmittedLatLng) {
            const moved = distanceMeters(lastWatchEmittedLatLng, cand);
            const elapsed = now - lastWatchEmitMs;
            if (moved < LIVE_WATCH_MIN_MOVE_M && elapsed < LIVE_WATCH_MIN_MS) return;
        }
        lastWatchEmitMs = now;
        lastWatchEmittedLatLng = { lat: cand.lat, lng: cand.lng };
        userLatLng = {
            lat: cand.lat,
            lng: cand.lng,
            accuracy: cand.accuracy,
            highAccuracy: cand.highAccuracy
        };
        placeUserMarker();
    }

    function pauseLiveLocationUpdates() {
        if (typeof geoWatchId === 'number') {
            try { navigator.geolocation.clearWatch(geoWatchId); } catch (e) { /* ignore */ }
            geoWatchId = null;
        }
    }

    function startLiveLocationUpdates() {
        if (!navigator.geolocation || typeof geoWatchId === 'number') return;
        if (!userLatLng) return;

        geoWatchId = navigator.geolocation.watchPosition(
            (pos) => { maybeApplyWatchPosition(pos); },
            (err) => {
                if (err && err.code === 1) {
                    hadLiveWatchPermission = false;
                    pauseLiveLocationUpdates();
                }
            },
            { enableHighAccuracy: true, maximumAge: 4000, timeout: 25000 }
        );
        hadLiveWatchPermission = true;
    }

    let visibilityForLiveLocationBound = false;
    function wireVisibilityForLiveLocation() {
        if (visibilityForLiveLocationBound) return;
        visibilityForLiveLocationBound = true;

        document.addEventListener('visibilitychange', () => {
            if (!hadLiveWatchPermission) return;
            if (document.hidden) {
                pauseLiveLocationUpdates();
            } else if (userLatLng) {
                startLiveLocationUpdates();
            }
        });
        window.addEventListener('beforeunload', pauseLiveLocationUpdates);
    }

    // ---------------------------------------------------------------------
    // Markers
    // ---------------------------------------------------------------------

    function markerStyleFor(tree) {
        let color = COLOR_NOT;
        if (tree.removed != null) color = COLOR_REMOVED;
        else if (tree.found) color = COLOR_FOUND;
        return {
            radius: 7,
            fillColor: color,
            fillOpacity: 0.92,
            color: COLOR_STROKE,
            weight: 1.25
        };
    }

    function titleFor(tree) {
        return `#${tree.id} \u2014 ${tree.species || tree.name}`;
    }

    function renderTrees(trees) {
        clearMarkers();
        for (const t of trees) {
            if (typeof t.lat !== 'number' || typeof t.lng !== 'number') continue;
            addMarker(t);
        }
    }

    function addMarker(tree) {
        const m = L.circleMarker([tree.lat, tree.lng], markerStyleFor(tree)).addTo(map);
        m.bindTooltip(titleFor(tree), { direction: 'top', offset: [0, -6] });
        m.on('click', () => openInfoForTree(tree.id));
        markers.set(tree.id, m);
        return m;
    }

    function updateTreeMarker(tree) {
        let m = markers.get(tree.id);
        if (!m && typeof tree.lat === 'number' && typeof tree.lng === 'number') {
            addMarker(tree);
            return;
        }
        if (m) {
            if (typeof tree.lat === 'number' && typeof tree.lng === 'number') {
                m.setLatLng([tree.lat, tree.lng]);
            }
            m.setStyle(markerStyleFor(tree));
            const tt = m.getTooltip();
            if (tt) tt.setContent(titleFor(tree));
        }
        // If this tree's popup is open, refresh its contents in place.
        if (openTreeId === tree.id) openInfoForTree(tree.id, /*keepOpen=*/true);
    }

    function clearMarkers() {
        for (const m of markers.values()) m.remove();
        markers.clear();
    }

    // ---------------------------------------------------------------------
    // Info popup
    // ---------------------------------------------------------------------

    const POPUP_OPTIONS = {
        maxWidth: 340,
        minWidth: 240,
        autoPan: true,
        autoPanPadding: [24, 80],
        keepInView: true,
        closeButton: true,
        className: 'tree-info-popup'
    };

    async function openInfoForTree(id, keepOpen) {
        const tree = await HeritageDB.getTree(id);
        if (!tree) return;
        const marker = markers.get(id);
        if (!marker) return;

        const html = buildInfoContent(tree);
        if (marker.getPopup()) {
            marker.setPopupContent(html);
        } else {
            marker.bindPopup(html, POPUP_OPTIONS);
        }

        const wasOpen = marker.isPopupOpen();
        if (!wasOpen) marker.openPopup();
        if (!keepOpen && !wasOpen) {
            map.panTo(marker.getLatLng(), { animate: true });
        }
        openTreeId = id;

        // openPopup() inserts the popup DOM synchronously, but the elements
        // inside .tree-info are not always queryable until the next tick
        // depending on the Leaflet build. setTimeout(0) is a safe fence and
        // works whether the popup was freshly opened or just had its content
        // swapped.
        setTimeout(() => wireInfoListeners(tree), 0);
    }

    function buildInfoContent(tree) {
        const { species, common } = HeritageWiki.splitSpeciesAndCommon(tree.name);
        const wikiUrl = HeritageWiki.wikipediaUrlForSpecies(species);
        const titleHtml = wikiUrl
            ? `<a href="${wikiUrl}" target="_blank" rel="noopener">
                   <span class="tree-info-species">${escapeHtml(species)}</span>${common ? ' &mdash; <span class="tree-info-common">' + escapeHtml(common) + '</span>' : ''}
               </a>`
            : `<span class="tree-info-species">${escapeHtml(tree.name || '')}</span>`;

        const foundDateStr = tree.foundDate ? formatLocalDate(tree.foundDate) : null;

        const hasCoords = typeof tree.lat === 'number' && typeof tree.lng === 'number';
        const navHref = hasCoords ? walkingDirectionsUrl(tree.lat, tree.lng) : '';

        let primary = '';
        if (tree.removed == null) {
            primary = tree.found
                ? `<span class="tree-info-found">Found ${escapeHtml(foundDateStr || '')}</span>
                   <button type="button" class="tree-info-btn ghost" data-action="undo">Undo</button>`
                : `<button type="button" class="tree-info-btn primary" data-action="found">Mark as found</button>`;
        }

        const navLink = hasCoords
            ? `<a class="tree-info-nav" href="${escapeHtml(navHref)}" target="_blank" rel="noopener"
                   title="Walking directions in Google Maps"
                   aria-label="Walking directions to heritage tree #${escapeHtml(tree.id)}">
                   <span class="tree-info-nav-icon" aria-hidden="true">\u{1F6B6}</span>
                   <span class="tree-info-nav-label">Navigate</span>
               </a>`
            : '';

        const actionsInner = [
            primary ? `<div class="tree-info-actions-main">${primary}</div>` : '',
            navLink
        ].filter(Boolean).join('');
        const actionsBlock = actionsInner
            ? `<div class="tree-info-actions">${actionsInner}</div>`
            : '';

        const badges = [];
        badges.push(`<span class="badge">#${escapeHtml(tree.id)}</span>`);
        if (tree.year) badges.push(`<span class="badge">Year ${escapeHtml(String(tree.year))}</span>`);
        if (tree.removed != null) badges.push(`<span class="badge removed">Removed ${escapeHtml(String(tree.removed))}</span>`);

        return `
            <div class="tree-info" data-id="${escapeHtml(tree.id)}">
                <h3 class="tree-info-title">${titleHtml}</h3>
                <div class="tree-info-meta">${badges.join('')}</div>
                <p class="tree-info-loc">${escapeHtml(tree.location || '')}</p>
                ${actionsBlock}
                <label class="tree-info-notes-label" for="notes-${escapeHtml(tree.id)}">
                    Notes <span class="tree-info-saved" data-saved>Saved</span>
                </label>
                <textarea id="notes-${escapeHtml(tree.id)}" class="tree-info-notes"
                    placeholder="Add a note\u2026" data-notes>${escapeHtml(tree.notes || '')}</textarea>
            </div>
        `;
    }

    function wireInfoListeners(tree) {
        const container = document.querySelector(`.tree-info[data-id="${cssEscape(tree.id)}"]`);
        if (!container) return;

        const markBtn = container.querySelector('[data-action="found"]');
        if (markBtn) markBtn.addEventListener('click', () => handleMarkFound(tree.id, true));

        const undoBtn = container.querySelector('[data-action="undo"]');
        if (undoBtn) undoBtn.addEventListener('click', () => handleMarkFound(tree.id, false));

        const notesEl = container.querySelector('[data-notes]');
        const savedEl = container.querySelector('[data-saved]');
        if (notesEl) {
            let savedTimer = null;
            const persist = async () => {
                const newVal = notesEl.value;
                const updated = await HeritageDB.updateTree(tree.id, (t) => {
                    t.notes = newVal;
                    return t;
                });
                if (updated) emitTreeUpdate(updated);
                if (savedEl) {
                    savedEl.classList.add('show');
                    if (savedTimer) clearTimeout(savedTimer);
                    savedTimer = setTimeout(() => savedEl.classList.remove('show'), 1200);
                }
            };
            notesEl.addEventListener('blur', persist);
        }
    }

    async function handleMarkFound(id, makeFound) {
        const updated = await HeritageDB.updateTree(id, (t) => {
            if (t.removed != null) return t;
            t.found = !!makeFound;
            t.foundDate = makeFound ? new Date().toISOString() : null;
            return t;
        });
        if (!updated) return;
        updateTreeMarker(updated);
        emitTreeUpdate(updated);
        openInfoForTree(id, /*keepOpen=*/true);
    }

    function setOnTreeUpdate(cb) { onTreeUpdateCallback = cb; }
    function emitTreeUpdate(tree) { if (onTreeUpdateCallback) onTreeUpdateCallback(tree); }

    // ---------------------------------------------------------------------
    // Camera
    // ---------------------------------------------------------------------

    function distanceMeters(a, b) {
        if (!a || !b) return Infinity;
        const R = 6371000;
        const toRad = (d) => d * Math.PI / 180;
        const dLat = toRad(b.lat - a.lat);
        const dLng = toRad(b.lng - a.lng);
        const la1 = toRad(a.lat);
        const la2 = toRad(b.lat);
        const s = Math.sin(dLat/2)**2 + Math.cos(la1)*Math.cos(la2)*Math.sin(dLng/2)**2;
        return 2 * R * Math.asin(Math.sqrt(s));
    }

    function nearestTreeTo(center, trees) {
        let best = null;
        let bestD = Infinity;
        for (const t of trees) {
            if (typeof t.lat !== 'number' || typeof t.lng !== 'number') continue;
            if (t.removed != null) continue;
            const d = distanceMeters(center, { lat: t.lat, lng: t.lng });
            if (d < bestD) { bestD = d; best = t; }
        }
        if (!best) {
            for (const t of trees) {
                if (typeof t.lat !== 'number' || typeof t.lng !== 'number') continue;
                const d = distanceMeters(center, { lat: t.lat, lng: t.lng });
                if (d < bestD) { bestD = d; best = t; }
            }
        }
        return best;
    }

    function autoFit(user, trees) {
        if (!map) return;
        const geoTrees = trees.filter((t) => typeof t.lat === 'number' && typeof t.lng === 'number');
        if (geoTrees.length === 0) {
            map.setView([PORTLAND.lat, PORTLAND.lng], 12);
            return;
        }

        if (user && distanceMeters(user, PORTLAND) <= MI_20_METERS) {
            const nearest = nearestTreeTo(user, geoTrees);
            if (nearest) {
                const bounds = L.latLngBounds([
                    [user.lat, user.lng],
                    [nearest.lat, nearest.lng]
                ]);
                // Leaflet's padding is TL/BR pixel offsets. Match the old
                // Google call's {top:90,bottom:130,left:60,right:60} layout so
                // the header and action bar don't clip the pins.
                map.fitBounds(bounds, {
                    paddingTopLeft: [60, 90],
                    paddingBottomRight: [60, 130],
                    animate: false
                });
                return;
            }
        }

        const bounds = L.latLngBounds(geoTrees.map((t) => [t.lat, t.lng]));
        map.fitBounds(bounds, { padding: [40, 40], animate: false });
    }

    function recenter(user, trees) {
        autoFit(user, trees);
    }

    // Pan/zoom to a tree and open its info popup. Used by the Nearby panel.
    // Returns true if the tree was found and focused.
    async function focusTree(id, opts) {
        const tree = await HeritageDB.getTree(id);
        if (!tree || typeof tree.lat !== 'number' || typeof tree.lng !== 'number') return false;
        const targetZoom = Math.max(map ? map.getZoom() : 16, (opts && opts.minZoom) || 16);
        map.setView([tree.lat, tree.lng], targetZoom, { animate: true });
        // Give the pan a tick so the popup opens at the final position.
        setTimeout(() => { openInfoForTree(id); }, 120);
        return true;
    }

    // ---------------------------------------------------------------------
    // Utils
    // ---------------------------------------------------------------------

    function escapeHtml(s) {
        return String(s == null ? '' : s)
            .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    }
    function cssEscape(s) {
        if (window.CSS && window.CSS.escape) return window.CSS.escape(s);
        return String(s).replace(/([^\w-])/g, '\\$1');
    }
    function formatLocalDate(iso) {
        try {
            const d = new Date(iso);
            return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
        } catch (e) { return iso; }
    }

    global.HeritageMap = {
        init,
        getMap,
        getUserLocation,
        getCachedUserLocation,
        placeUserMarker,
        startLiveLocationUpdates,
        renderTrees,
        updateTreeMarker,
        clearMarkers,
        openInfoForTree,
        setOnTreeUpdate,
        autoFit,
        recenter,
        focusTree,
        distanceMeters,
        PORTLAND
    };
})(window);
