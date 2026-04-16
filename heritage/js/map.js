// Google Maps integration: markers, InfoWindow with Found/Notes controls,
// camera auto-fitting.
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

    let map = null;
    let infoWindow = null;
    let markers = new Map();    // id -> google.maps.Marker
    let userLatLng = null;      // {lat, lng} or null
    let userMarker = null;
    let onTreeUpdateCallback = null;
    let openTreeId = null;

    // ---------------------------------------------------------------------
    // Init
    // ---------------------------------------------------------------------

    async function init(containerId) {
        if (!global.google || !global.google.maps) {
            throw new Error('Google Maps JS API is not loaded.');
        }
        const container = document.getElementById(containerId);
        map = new google.maps.Map(container, {
            center: PORTLAND,
            zoom: 12,
            mapTypeControl: false,
            streetViewControl: false,
            fullscreenControl: false,
            clickableIcons: false,
            gestureHandling: 'greedy',
            styles: mapStyles()
        });
        infoWindow = new google.maps.InfoWindow({ maxWidth: 340 });
        google.maps.event.addListener(infoWindow, 'closeclick', () => { openTreeId = null; });
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
        if (userMarker) { userMarker.setMap(null); userMarker = null; }
        userMarker = new google.maps.Marker({
            position: userLatLng,
            map,
            icon: {
                path: google.maps.SymbolPath.CIRCLE,
                scale: 6,
                fillColor: '#4a7c8a',
                fillOpacity: 0.95,
                strokeColor: '#ffffff',
                strokeWeight: 2
            },
            title: 'Your location',
            zIndex: 9999
        });
    }

    // ---------------------------------------------------------------------
    // Markers
    // ---------------------------------------------------------------------

    function markerIconFor(tree) {
        let color = COLOR_NOT;
        if (tree.removed != null) color = COLOR_REMOVED;
        else if (tree.found) color = COLOR_FOUND;
        return {
            path: google.maps.SymbolPath.CIRCLE,
            scale: 6.5,
            fillColor: color,
            fillOpacity: 0.92,
            strokeColor: '#1f3b2a',
            strokeWeight: 1.25
        };
    }

    function renderTrees(trees) {
        clearMarkers();
        for (const t of trees) {
            if (typeof t.lat !== 'number' || typeof t.lng !== 'number') continue;
            const m = new google.maps.Marker({
                position: { lat: t.lat, lng: t.lng },
                map,
                icon: markerIconFor(t),
                title: `#${t.id} \u2014 ${t.species || t.name}`
            });
            m.addListener('click', () => openInfoForTree(t.id));
            markers.set(t.id, m);
        }
    }

    function updateTreeMarker(tree) {
        let m = markers.get(tree.id);
        if (!m && typeof tree.lat === 'number' && typeof tree.lng === 'number') {
            m = new google.maps.Marker({
                position: { lat: tree.lat, lng: tree.lng },
                map,
                icon: markerIconFor(tree),
                title: `#${tree.id} \u2014 ${tree.species || tree.name}`
            });
            m.addListener('click', () => openInfoForTree(tree.id));
            markers.set(tree.id, m);
            return;
        }
        if (m) {
            if (typeof tree.lat === 'number' && typeof tree.lng === 'number') {
                m.setPosition({ lat: tree.lat, lng: tree.lng });
            }
            m.setIcon(markerIconFor(tree));
            m.setTitle(`#${tree.id} \u2014 ${tree.species || tree.name}`);
        }
        // If this tree's info window is open, refresh its contents.
        if (openTreeId === tree.id) openInfoForTree(tree.id, /*keepOpen=*/true);
    }

    function clearMarkers() {
        for (const m of markers.values()) m.setMap(null);
        markers.clear();
    }

    // ---------------------------------------------------------------------
    // Info window
    // ---------------------------------------------------------------------

    async function openInfoForTree(id, keepOpen) {
        const tree = await HeritageDB.getTree(id);
        if (!tree) return;
        const marker = markers.get(id);
        if (!marker) return;
        const content = buildInfoContent(tree);
        infoWindow.setContent(content);
        infoWindow.open({ map, anchor: marker });
        openTreeId = id;

        // Attach listeners after DOM is inserted. The 'domready' fires each time
        // content is set, so we use a one-shot listener.
        const onReady = () => {
            google.maps.event.removeListener(domReadyHandle);
            wireInfoListeners(tree);
        };
        const domReadyHandle = google.maps.event.addListener(infoWindow, 'domready', onReady);

        if (!keepOpen) {
            map.panTo(marker.getPosition());
        }
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
            ? '' // no mark-as-found for removed trees
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
            if (t.removed != null) return t; // no-op for removed
            t.found = !!makeFound;
            t.foundDate = makeFound ? new Date().toISOString() : null;
            return t;
        });
        if (!updated) return;
        updateTreeMarker(updated);
        emitTreeUpdate(updated);
        // Re-open the info window so action row re-renders.
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
            if (t.removed != null) continue; // prefer not-removed trees for "zoom to include one"
            const d = distanceMeters(center, { lat: t.lat, lng: t.lng });
            if (d < bestD) { bestD = d; best = t; }
        }
        if (!best) {
            // Fallback: include removed trees if that's all we have.
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
        const bounds = new google.maps.LatLngBounds();
        const geoTrees = trees.filter((t) => typeof t.lat === 'number' && typeof t.lng === 'number');
        if (geoTrees.length === 0) {
            map.setCenter(PORTLAND);
            map.setZoom(12);
            return;
        }

        if (user && distanceMeters(user, PORTLAND) <= MI_20_METERS) {
            const nearest = nearestTreeTo(user, geoTrees);
            if (nearest) {
                bounds.extend(user);
                bounds.extend({ lat: nearest.lat, lng: nearest.lng });
                map.fitBounds(bounds, { top: 90, left: 60, right: 60, bottom: 130 });
                return;
            }
        }

        // Fit all trees centered on Portland.
        for (const t of geoTrees) bounds.extend({ lat: t.lat, lng: t.lng });
        map.fitBounds(bounds, 40);
    }

    function recenter(user, trees) {
        autoFit(user, trees);
    }

    // ---------------------------------------------------------------------
    // Style
    // ---------------------------------------------------------------------

    function mapStyles() {
        // Soft, muted green/earth palette so markers pop.
        return [
            { elementType: 'geometry',       stylers: [{ color: '#e8ebe3' }] },
            { elementType: 'labels.text.fill',   stylers: [{ color: '#475d49' }] },
            { elementType: 'labels.text.stroke', stylers: [{ color: '#f2f5ec' }] },
            { featureType: 'administrative.locality', elementType: 'labels.text.fill', stylers: [{ color: '#3c5a40' }] },
            { featureType: 'poi.park', elementType: 'geometry', stylers: [{ color: '#c9dcc1' }] },
            { featureType: 'poi.park', elementType: 'labels.text.fill', stylers: [{ color: '#2e7d32' }] },
            { featureType: 'road', elementType: 'geometry', stylers: [{ color: '#f6f4ea' }] },
            { featureType: 'road', elementType: 'labels.text.fill', stylers: [{ color: '#6a6a5a' }] },
            { featureType: 'road.arterial', elementType: 'geometry', stylers: [{ color: '#ede7d4' }] },
            { featureType: 'road.highway', elementType: 'geometry', stylers: [{ color: '#e1d6b3' }] },
            { featureType: 'transit', elementType: 'geometry', stylers: [{ color: '#d8d2c1' }] },
            { featureType: 'water', elementType: 'geometry', stylers: [{ color: '#a4c4cf' }] },
            { featureType: 'water', elementType: 'labels.text.fill', stylers: [{ color: '#2e5863' }] }
        ];
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
