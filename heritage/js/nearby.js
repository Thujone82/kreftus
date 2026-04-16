// Nearby panel: list the 10 closest trees to the user's current location,
// with a Navigate link that opens Google Maps walking directions (works on
// iOS, Android, Windows, and desktop web alike).

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
        const la = Number(lat);
        const ln = Number(lng);
        if (!Number.isFinite(la) || !Number.isFinite(ln)) return '#';
        return `https://www.google.com/maps/dir/?api=1&destination=${la},${ln}&travelmode=walking`;
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
            listEl.appendChild(buildRow(row));
        }
    }

    function buildRow(row) {
        const li = document.createElement('li');
        li.className = 'nearby-row';

        // Main "View on map" button: name + metadata row. Tapping it closes
        // the Nearby panel and opens the tree's popup on the map.
        const view = document.createElement('button');
        view.type = 'button';
        view.className = 'nearby-item';
        view.title = 'Show this tree on the map';

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

        view.appendChild(nameSpan);
        view.appendChild(dist);
        view.appendChild(meta);

        view.addEventListener('click', () => onView(row.tree));

        // Navigate link: Google Maps walking directions. Rendered as a real
        // anchor so middle/cmd-click and "open in new tab" work on desktop,
        // and mobile OSes still deep-link into the native Google Maps app
        // when it's installed.
        const nav = document.createElement('a');
        nav.className = 'nearby-item-nav';
        nav.href = walkingNavUrl(row.tree.lat, row.tree.lng);
        nav.target = '_blank';
        nav.rel = 'noopener';
        nav.title = 'Open walking directions in Google Maps';
        nav.setAttribute('aria-label',
            `Navigate to ${row.tree.species || row.tree.name || row.tree.id}`);
        const navIcon = document.createElement('span');
        navIcon.className = 'nearby-item-nav-icon';
        navIcon.setAttribute('aria-hidden', 'true');
        // Unicode walking-person (U+1F6B6) + right arrow keeps the icon
        // readable on both light-ish and very dark backgrounds.
        navIcon.textContent = '\u{1F6B6}';
        const navLabel = document.createElement('span');
        navLabel.className = 'nearby-item-nav-label';
        navLabel.textContent = 'Navigate';
        nav.appendChild(navIcon);
        nav.appendChild(navLabel);

        li.appendChild(view);
        li.appendChild(nav);
        return li;
    }

    async function onView(tree) {
        // Close the panel first so the popup isn't hidden under it on small
        // screens, then defer to the map for the actual pan/zoom + open.
        try { HeritageUI.closeNearby(); } catch (e) { /* no-op */ }
        if (HeritageMap && typeof HeritageMap.focusTree === 'function') {
            await HeritageMap.focusTree(tree.id);
        } else if (HeritageMap && typeof HeritageMap.openInfoForTree === 'function') {
            HeritageMap.openInfoForTree(tree.id);
        }
    }

    global.HeritageNearby = {
        render,
        computeNearest,
        walkingNavUrl,
        formatDistance
    };
})(window);
