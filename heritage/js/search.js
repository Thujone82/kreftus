(function (global) {
    'use strict';

    let treesCache = [];
    let indexCache = [];
    let debounceTimer = null;
    let bootstrapped = false;
    const activeFieldFilters = new Set();

    const FIELD_LABELS = {
        id: 'ID',
        name: 'Name',
        commonName: 'Common',
        location: 'Address',
        geocodeAddress: 'Geocode',
        coordinates: 'Coordinate',
        notes: 'Notes',
        year: 'Year',
        removed: 'Removed'
    };

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

    function buildCoordinateSearchValue(tree) {
        const lat = toNum(tree.lat);
        const lng = toNum(tree.lng);
        return (lat != null && lng != null)
            ? [
                `${lat}`,
                `${lng}`,
                `${lat.toFixed(6)}`,
                `${lng.toFixed(6)}`,
                `${lat},${lng}`,
                `${lat.toFixed(6)},${lng.toFixed(6)}`
            ].join(' ')
            : '';
    }

    function buildFieldSearchMap(tree) {
        return {
            id: normalize(tree.id),
            name: normalize([tree.name, tree.species].filter(Boolean).join(' ')),
            commonName: normalize(tree.commonName),
            location: normalize([tree.location, tree.geocodeFormatted].filter(Boolean).join(' ')),
            geocodeAddress: normalize(tree.geocodeAddress),
            coordinates: normalize(buildCoordinateSearchValue(tree)),
            notes: normalize(tree.notes),
            year: normalize(tree.year),
            removed: normalize(tree.removed)
        };
    }

    async function ensureIndex() {
        treesCache = await HeritageDB.getAllTrees();
        indexCache = treesCache.map((tree) => {
            const idText = normalize(tree.id);
            const idNumeric = String(parseInt(tree.id, 10));
            const fieldMap = buildFieldSearchMap(tree);
            const blob = normalize(Object.values(fieldMap).join(' '));
            return {
                tree,
                blob,
                fieldMap,
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

    function computeFieldMatches(row, tokens) {
        const matched = new Set();
        for (const t of tokens) {
            if (t.kind === 'id') {
                if (row.idText.includes(t.value) || row.idNumeric.includes(t.value)) {
                    matched.add('id');
                    continue;
                }
                return null;
            }
            if (!row.blob.includes(t.value)) {
                return null;
            }
            let matchedAnyField = false;
            for (const fieldKey of Object.keys(row.fieldMap)) {
                if (row.fieldMap[fieldKey].includes(t.value)) {
                    matched.add(fieldKey);
                    matchedAnyField = true;
                }
            }
            if (!matchedAnyField) return null;
        }
        if (activeFieldFilters.size > 0) {
            for (const filterKey of activeFieldFilters) {
                if (!matched.has(filterKey)) return null;
            }
        }
        return matched;
    }

    function buildMatchTags(matchedFieldKeys) {
        return Array.from(matchedFieldKeys)
            .filter((k) => FIELD_LABELS[k])
            .sort((a, b) => FIELD_LABELS[a].localeCompare(FIELD_LABELS[b]))
            .map((k) => ({ key: k, label: FIELD_LABELS[k] }));
    }

    function queryRows(query) {
        const tokens = parseQueryTokens(query);
        if (tokens.length === 0) return [];
        const origin = getOrigin();
        const rows = indexCache
            .map((row) => {
                const matchedFieldKeys = computeFieldMatches(row, tokens);
                if (!matchedFieldKeys) return null;
                return { row, matchedFieldKeys };
            })
            .filter(Boolean)
            .map((row) => {
                const isMappable = HeritageMap.isTreeMappable(row.row.tree);
                const distance = isMappable
                    ? HeritageMap.distanceMeters(origin, { lat: row.row.tree.lat, lng: row.row.tree.lng })
                    : Infinity;
                return {
                    tree: row.row.tree,
                    distance,
                    matchedFieldKeys: row.matchedFieldKeys,
                    matchTags: buildMatchTags(row.matchedFieldKeys)
                };
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
            listEl.appendChild(HeritageNearby.buildTreeRow(row, {
                onView: openTreeFromSearch,
                metaTags: row.matchTags,
                activeTagSet: activeFieldFilters,
                onTagClick: onToggleFieldFilter
            }));
        }
    }

    function onToggleFieldFilter(fieldKey) {
        if (activeFieldFilters.has(fieldKey)) activeFieldFilters.delete(fieldKey);
        else activeFieldFilters.add(fieldKey);
        runQueryAndRender(true);
    }

    function runQueryAndRender(immediate) {
        const input = document.getElementById('searchInput');
        if (!input) return;
        const run = () => {
            const q = input.value || '';
            if (!q.trim()) {
                renderEmpty('Type to search, click tags to toggle filters...');
                return;
            }
            const rows = queryRows(q);
            if (rows.length === 0) {
                renderEmpty('No matching trees found.');
                return;
            }
            renderRows(rows);
        };
        if (immediate) {
            run();
            return;
        }
        if (debounceTimer) clearTimeout(debounceTimer);
        debounceTimer = setTimeout(run, 80);
    }

    function onInput() {
        runQueryAndRender(false);
    }

    function bindInput() {
        if (bootstrapped) return;
        const input = document.getElementById('searchInput');
        if (!input) return;
        input.addEventListener('input', onInput);
        bootstrapped = true;
    }

    function focusSearchInput(input) {
        if (!input) return;
        const tryFocus = () => {
            try {
                input.focus({ preventScroll: true });
            } catch (e) {
                try { input.focus(); } catch (err) { /* ignore */ }
            }
            return document.activeElement === input;
        };
        if (tryFocus()) {
            try { input.select(); } catch (e) { /* ignore */ }
            return;
        }

        // Retry across a few render ticks/transitions for slower clients.
        const retryDelays = [0, 32, 96, 220, 420];
        for (const delay of retryDelays) {
            setTimeout(() => {
                if (document.activeElement === input) return;
                if (tryFocus()) {
                    try { input.select(); } catch (e) { /* ignore */ }
                }
            }, delay);
        }
    }

    async function open() {
        HeritageUI.openModal('searchModal');
        bindInput();
        const input = document.getElementById('searchInput');
        if (input) {
            // iOS/PWA focus is most reliable when it happens in the same tap
            // gesture, before any awaited async work.
            focusSearchInput(input);
        }
        await ensureIndex();
        if (input) {
            runQueryAndRender(true);
        } else {
            renderEmpty('Type to search, click tags to toggle filters...');
        }
    }

    function close() {
        const input = document.getElementById('searchInput');
        if (input) input.value = '';
        activeFieldFilters.clear();
        renderEmpty('Type to search, click tags to toggle filters...');
    }

    global.HeritageSearch = {
        open,
        close
    };
})(window);
