// Nearby panel: list the 10 closest trees to the user's current location,
// with a click handler that opens Google Maps walking directions.

(function (global) {
    'use strict';

    function formatDistance(meters) {
        if (!Number.isFinite(meters)) return '';
        const miles = meters / 1609.34;
        if (miles < 0.1) return `${Math.round(meters)} m`;
        if (miles < 10)  return `${miles.toFixed(2)} mi`;
        return `${Math.round(miles)} mi`;
    }

    function walkingNavUrl(lat, lng) {
        return `https://www.google.com/maps/dir/?api=1&travelmode=walking&destination=${lat},${lng}`;
    }

    function originFor() {
        const user = HeritageMap.getCachedUserLocation();
        if (user) return { origin: user, fromUser: true };
        // Fallback to current map center so the panel always has something useful.
        const m = HeritageMap.getMap();
        if (m) {
            const c = m.getCenter();
            if (c) return { origin: { lat: c.lat(), lng: c.lng() }, fromUser: false };
        }
        return { origin: HeritageMap.PORTLAND, fromUser: false };
    }

    /**
     * Build a list of the N nearest trees (default 10), excluding trees without
     * coordinates. Removed trees are included but visibly marked.
     */
    async function computeNearest(limit) {
        const n = Math.max(1, limit || 10);
        const { origin, fromUser } = originFor();
        const trees = await HeritageDB.getAllTrees();
        const withGeo = trees.filter((t) => typeof t.lat === 'number' && typeof t.lng === 'number');
        const scored = withGeo.map((t) => ({
            tree: t,
            distance: HeritageMap.distanceMeters(origin, { lat: t.lat, lng: t.lng })
        }));
        scored.sort((a, b) => a.distance - b.distance);
        return { rows: scored.slice(0, n), origin, fromUser };
    }

    /**
     * Render the nearby panel body. Expects DOM elements:
     *   #nearbyList, #nearbyOrigin
     */
    async function render() {
        const listEl = document.getElementById('nearbyList');
        const originEl = document.getElementById('nearbyOrigin');
        if (!listEl) return;

        const { rows, fromUser } = await computeNearest(10);
        originEl.textContent = fromUser
            ? 'From your current location'
            : 'From the map center (location unavailable)';

        if (rows.length === 0) {
            listEl.innerHTML = '';
            const empty = document.createElement('li');
            empty.className = 'nearby-empty';
            empty.textContent = 'No trees with known coordinates yet. Geocoding may still be running.';
            listEl.appendChild(empty);
            return;
        }

        listEl.innerHTML = '';
        for (const row of rows) {
            const li = document.createElement('li');
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'nearby-item';

            const nameSpan = document.createElement('span');
            nameSpan.className = 'nearby-item-name';
            nameSpan.textContent = row.tree.species || row.tree.name || `#${row.tree.id}`;

            const dist = document.createElement('span');
            dist.className = 'nearby-item-distance';
            dist.textContent = formatDistance(row.distance);

            const meta = document.createElement('span');
            meta.className = 'nearby-item-meta';
            const idBadge = document.createElement('span');
            idBadge.textContent = `#${row.tree.id}`;
            meta.appendChild(idBadge);
            if (row.tree.commonName) {
                const common = document.createElement('span');
                common.textContent = row.tree.commonName;
                meta.appendChild(common);
            }
            const badge = document.createElement('span');
            if (row.tree.removed != null) {
                badge.className = 'nearby-item-badge removed';
                badge.textContent = `Removed ${row.tree.removed}`;
            } else if (row.tree.found) {
                badge.className = 'nearby-item-badge found';
                badge.textContent = 'Found';
            } else {
                badge.className = 'nearby-item-badge notfound';
                badge.textContent = 'Not found';
            }
            meta.appendChild(badge);

            btn.appendChild(nameSpan);
            btn.appendChild(dist);
            btn.appendChild(meta);

            btn.addEventListener('click', () => {
                const url = walkingNavUrl(row.tree.lat, row.tree.lng);
                window.open(url, '_blank', 'noopener');
            });

            li.appendChild(btn);
            listEl.appendChild(li);
        }
    }

    global.HeritageNearby = {
        render,
        computeNearest,
        walkingNavUrl,
        formatDistance
    };
})(window);
