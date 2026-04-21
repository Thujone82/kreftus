(function (global) {
    'use strict';

    let treesCache = [];
    let indexCache = [];
    let debounceTimer = null;
    let bootstrapped = false;

    function normalize(text) {
        return String(text == null ? '' : text).toLowerCase();
    }

    function tokenize(query) {
        return normalize(query).split(/\s+/).map((x) => x.trim()).filter(Boolean);
    }

    function parseQueryTokens(query) {
        return tokenize(query).map((token) => {
            if (token.startsWith('#')) {
                return {
                    kind: 'id',
                    value: token.slice(1).trim()
                };
            }
            return {
                kind: 'text',
                value: token
            };
        }).filter((t) => t.value.length > 0);
    }

    function toNum(v) {
        const n = Number(v);
        return Number.isFinite(n) ? n : null;
    }

    function buildSearchBlob(tree) {
        const lat = toNum(tree.lat);
        const lng = toNum(tree.lng);
        const coords = (lat != null && lng != null)
            ? [
                `${lat}`,
                `${lng}`,
                `${lat.toFixed(6)}`,
                `${lng.toFixed(6)}`,
                `${lat},${lng}`,
                `${lat.toFixed(6)},${lng.toFixed(6)}`
            ].join(' ')
            : '';
        return normalize([
            tree.id,
            tree.name,
            tree.species,
            tree.commonName,
            tree.location,
            tree.geocodeAddress,
            tree.geocodeFormatted,
            tree.notes,
            tree.year,
            tree.removed,
            coords
        ].join(' '));
    }

    async function ensureIndex() {
        treesCache = await HeritageDB.getAllTrees();
        indexCache = treesCache.map((tree) => {
            const idText = normalize(tree.id);
            const idNumeric = String(parseInt(tree.id, 10));
            return {
                tree,
                blob: buildSearchBlob(tree),
                idText,
                idNumeric: Number.isFinite(Number(idNumeric)) ? idNumeric : ''
            };
        });
    }

    function getOrigin() {
        if (global.HeritageNearby && typeof HeritageNearby.originFor === 'function') {
            return HeritageNearby.originFor().origin;
        }
        const user = HeritageMap.getCachedUserLocation();
        if (user) return user;
        return HeritageMap.PORTLAND;
    }

    function byDistanceThenId(a, b) {
        const da = a.distance;
        const db = b.distance;
        if (Number.isFinite(da) && Number.isFinite(db) && da !== db) return da - db;
        if (Number.isFinite(da) && !Number.isFinite(db)) return -1;
        if (!Number.isFinite(da) && Number.isFinite(db)) return 1;
        return String(a.tree.id).localeCompare(String(b.tree.id), undefined, { numeric: true });
    }

    function queryRows(query) {
        const tokens = parseQueryTokens(query);
        if (tokens.length === 0) return [];
        const origin = getOrigin();
        const rows = indexCache
            .filter((row) => tokens.every((t) => {
                if (t.kind === 'id') {
                    return row.idText.includes(t.value) || row.idNumeric.includes(t.value);
                }
                return row.blob.includes(t.value);
            }))
            .map((row) => {
                const isMappable = HeritageMap.isTreeMappable(row.tree);
                const distance = isMappable
                    ? HeritageMap.distanceMeters(origin, { lat: row.tree.lat, lng: row.tree.lng })
                    : Infinity;
                return { tree: row.tree, distance };
            });
        rows.sort(byDistanceThenId);
        return rows;
    }

    function openTreeFromSearch(tree) {
        HeritageUI.closeModal('searchModal');
        if (global.HeritageSearch && typeof HeritageSearch.close === 'function') {
            HeritageSearch.close();
        }
        if (HeritageMap && typeof HeritageMap.focusTree === 'function') {
            void HeritageMap.focusTree(tree.id);
        }
    }

    function renderEmpty(message) {
        const listEl = document.getElementById('searchResults');
        if (!listEl) return;
        listEl.innerHTML = '';
        const li = document.createElement('li');
        li.className = 'nearby-empty search-empty';
        li.textContent = message;
        listEl.appendChild(li);
    }

    function renderRows(rows) {
        const listEl = document.getElementById('searchResults');
        if (!listEl) return;
        listEl.innerHTML = '';
        for (const row of rows) {
            listEl.appendChild(HeritageNearby.buildTreeRow(row, { onView: openTreeFromSearch }));
        }
    }

    function onInput() {
        const input = document.getElementById('searchInput');
        if (!input) return;
        if (debounceTimer) clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
            const q = input.value || '';
            if (!q.trim()) {
                renderEmpty('Type to search trees.');
                return;
            }
            const rows = queryRows(q);
            if (rows.length === 0) {
                renderEmpty('No matching trees found.');
                return;
            }
            renderRows(rows);
        }, 80);
    }

    function bindInput() {
        if (bootstrapped) return;
        const input = document.getElementById('searchInput');
        if (!input) return;
        input.addEventListener('input', onInput);
        bootstrapped = true;
    }

    async function open() {
        HeritageUI.openModal('searchModal');
        bindInput();
        await ensureIndex();
        const input = document.getElementById('searchInput');
        if (input) {
            input.focus();
            onInput();
        } else {
            renderEmpty('Type to search trees.');
        }
    }

    function close() {
        const input = document.getElementById('searchInput');
        if (input) input.value = '';
        renderEmpty('Type to search trees.');
    }

    global.HeritageSearch = {
        open,
        close
    };
})(window);
