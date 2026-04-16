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

(function (global) {
    'use strict';

    const PORTLAND = { lat: 45.5152, lng: -122.6784 };
    const MI_20_METERS = 32186.88;

    const COLOR_FOUND   = '#2e7d32';
    const COLOR_NOT     = '#d4a24a';
    const COLOR_REMOVED = '#7a7a7a';
    const COLOR_STROKE  = '#1f3b2a';

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
            preferCanvas: true,   // canvas-backed circle markers scale better than SVG for ~400 pins
            worldCopyJump: false
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
        return map;
    }

    function getMap() { return map; }

    // ---------------------------------------------------------------------
    // User location
    // ---------------------------------------------------------------------

    function getUserLocation(timeoutMs) {
        return new Promise((resolve) => {
            if (!navigator.geolocation) return resolve(null);
            navigator.geolocation.getCurrentPosition(
                (pos) => {
                    userLatLng = { lat: pos.coords.latitude, lng: pos.coords.longitude };
                    resolve(userLatLng);
                },
                () => resolve(null),
                { timeout: timeoutMs || 8000, maximumAge: 300000, enableHighAccuracy: false }
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

    async function openInfoForTree(id, keepOpen) {
        const tree = await HeritageDB.getTree(id);
        if (!tree) return;
        const marker = markers.get(id);
        if (!marker) return;

        const html = buildInfoContent(tree);
        marker.unbindPopup();
        marker.bindPopup(html, {
            maxWidth: 340,
            minWidth: 240,
            autoPan: true,
            autoPanPadding: [24, 80],
            keepInView: true,
            closeButton: true,
            className: 'tree-info-popup'
        });

        if (keepOpen || marker.isPopupOpen()) {
            marker.openPopup();
        } else {
            marker.openPopup();
            map.panTo(marker.getLatLng(), { animate: true });
        }
        openTreeId = id;

        // Popup content is injected on open. Wire listeners after the popup
        // opens (Leaflet fires 'popupopen' on the marker).
        const onOpen = () => {
            marker.off('popupopen', onOpen);
            wireInfoListeners(tree);
        };
        marker.on('popupopen', onOpen);
        // If the popup is already open (re-render case) the event won't fire
        // again, so wire immediately.
        if (marker.isPopupOpen()) setTimeout(() => wireInfoListeners(tree), 0);
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

        const actions = tree.removed != null
            ? ''
            : tree.found
                ? `<span class="tree-info-found">Found ${escapeHtml(foundDateStr || '')}</span>
                   <button class="tree-info-btn ghost" data-action="undo">Undo</button>`
                : `<button class="tree-info-btn primary" data-action="found">Mark as found</button>`;

        const badges = [];
        badges.push(`<span class="badge">#${escapeHtml(tree.id)}</span>`);
        if (tree.year) badges.push(`<span class="badge">Year ${escapeHtml(String(tree.year))}</span>`);
        if (tree.removed != null) badges.push(`<span class="badge removed">Removed ${escapeHtml(String(tree.removed))}</span>`);

        return `
            <div class="tree-info" data-id="${escapeHtml(tree.id)}">
                <h3 class="tree-info-title">${titleHtml}</h3>
                <div class="tree-info-meta">${badges.join('')}</div>
                <p class="tree-info-loc">${escapeHtml(tree.location || '')}</p>
                <div class="tree-info-actions">${actions}</div>
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
        renderTrees,
        updateTreeMarker,
        clearMarkers,
        openInfoForTree,
        setOnTreeUpdate,
        autoFit,
        recenter,
        distanceMeters,
        PORTLAND
    };
})(window);
