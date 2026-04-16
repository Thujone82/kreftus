// IndexedDB wrapper for PDX Heritage Trees.
//
// Stores:
//   trees  - keyPath "id" (zero-padded 3-digit string like "001"). One record per
//            tree; contains canonical fields plus user-owned fields (found,
//            foundDate, notes, lat, lng).
//   meta   - keyPath "key". Arbitrary singleton values (lastSyncAt, sourceUrl,
//            sourceVersion, etc.). Never contains user-owned data.
//
// Versioning: onupgradeneeded is ADDITIVE ONLY. We never delete stores or wipe
// user data during upgrades. That is a hard rule so app updates cannot cost a
// user their Found marks or notes.

(function (global) {
    'use strict';

    const DB_NAME = 'pdxHeritage';
    const DB_VERSION = 1;
    const STORE_TREES = 'trees';
    const STORE_META  = 'meta';

    let dbPromise = null;

    function open() {
        if (dbPromise) return dbPromise;
        dbPromise = new Promise((resolve, reject) => {
            const req = indexedDB.open(DB_NAME, DB_VERSION);
            req.onupgradeneeded = (ev) => {
                const db = req.result;
                if (!db.objectStoreNames.contains(STORE_TREES)) {
                    const store = db.createObjectStore(STORE_TREES, { keyPath: 'id' });
                    store.createIndex('by_found',   'found',   { unique: false });
                    store.createIndex('by_removed', 'removed', { unique: false });
                }
                if (!db.objectStoreNames.contains(STORE_META)) {
                    db.createObjectStore(STORE_META, { keyPath: 'key' });
                }
                // Future versions: add migrations here, but never delete or clear
                // existing stores. User-owned fields (found, foundDate, notes)
                // must be preserved.
            };
            req.onsuccess = () => resolve(req.result);
            req.onerror   = () => reject(req.error);
            req.onblocked = () => reject(new Error('IndexedDB open was blocked by another tab.'));
        });
        return dbPromise;
    }

    function tx(storeNames, mode) {
        return open().then((db) => db.transaction(storeNames, mode));
    }

    function wrap(req) {
        return new Promise((resolve, reject) => {
            req.onsuccess = () => resolve(req.result);
            req.onerror   = () => reject(req.error);
        });
    }

    // -- Trees ---------------------------------------------------------------

    async function getAllTrees() {
        const t = await tx(STORE_TREES, 'readonly');
        return wrap(t.objectStore(STORE_TREES).getAll());
    }

    async function getTree(id) {
        const t = await tx(STORE_TREES, 'readonly');
        return wrap(t.objectStore(STORE_TREES).get(id));
    }

    async function putTree(tree) {
        const t = await tx(STORE_TREES, 'readwrite');
        await wrap(t.objectStore(STORE_TREES).put(tree));
        return tree;
    }

    async function putManyTrees(trees) {
        const t = await tx(STORE_TREES, 'readwrite');
        const store = t.objectStore(STORE_TREES);
        await Promise.all(trees.map((tree) => wrap(store.put(tree))));
        return trees.length;
    }

    async function countTrees() {
        const t = await tx(STORE_TREES, 'readonly');
        return wrap(t.objectStore(STORE_TREES).count());
    }

    async function updateTree(id, mutator) {
        const t = await tx(STORE_TREES, 'readwrite');
        const store = t.objectStore(STORE_TREES);
        const existing = await wrap(store.get(id));
        if (!existing) return null;
        const updated = mutator({ ...existing }) || existing;
        updated.id = id;
        updated.lastUpdatedAt = new Date().toISOString();
        await wrap(store.put(updated));
        return updated;
    }

    // -- Meta ----------------------------------------------------------------

    async function getMeta(key) {
        const t = await tx(STORE_META, 'readonly');
        const rec = await wrap(t.objectStore(STORE_META).get(key));
        return rec ? rec.value : undefined;
    }

    async function setMeta(key, value) {
        const t = await tx(STORE_META, 'readwrite');
        await wrap(t.objectStore(STORE_META).put({ key, value }));
        return value;
    }

    global.HeritageDB = {
        open,
        getAllTrees,
        getTree,
        putTree,
        putManyTrees,
        countTrees,
        updateTree,
        getMeta,
        setMeta,
        _constants: { DB_NAME, STORE_TREES, STORE_META }
    };
})(window);
