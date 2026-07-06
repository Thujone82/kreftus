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
    let dbConn = null;
    let openRecovered = false;

    function isRecoverableOpenError(err) {
        if (!err) return false;
        if (err.name === 'UnknownError' || err.name === 'InvalidStateError') return true;
        return /internal error/i.test(String(err.message || ''));
    }

    function deleteDatabase() {
        dbConn = null;
        return new Promise((resolve, reject) => {
            const req = indexedDB.deleteDatabase(DB_NAME);
            req.onblocked = () => {
                console.warn('IndexedDB delete blocked; close other Heritage tabs and retry.');
            };
            req.onsuccess = () => resolve();
            req.onerror = () => reject(req.error);
        });
    }

    function openOnce() {
        return new Promise((resolve, reject) => {
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
            req.onsuccess = () => {
                dbConn = req.result;
                dbConn.onversionchange = () => {
                    dbConn.close();
                    dbConn = null;
                    dbPromise = null;
                };
                resolve(dbConn);
            };
            req.onerror   = () => reject(req.error);
            req.onblocked = () => reject(new Error('IndexedDB open was blocked by another tab.'));
        });
    }

    async function open() {
        if (dbPromise) return dbPromise;
        dbPromise = (async () => {
            try {
                return await openOnce();
            } catch (err) {
                if (!isRecoverableOpenError(err)) throw err;
                console.warn('IndexedDB open failed with recoverable error; rebuilding database.', err);
                dbPromise = null;
                await deleteDatabase();
                openRecovered = true;
                return openOnce();
            }
        })();
        return dbPromise;
    }

    function wasRecovered() {
        return openRecovered;
    }

    function runTransaction(storeNames, mode, fn) {
        return open().then((db) => new Promise((resolve, reject) => {
            const t = db.transaction(storeNames, mode);
            t.oncomplete = () => resolve();
            t.onerror = () => reject(t.error || new Error('IndexedDB transaction failed'));
            t.onabort = () => reject(t.error || new Error('IndexedDB transaction aborted'));
            try {
                fn(t);
            } catch (err) {
                reject(err);
            }
        }));
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
        // Schedule every put synchronously inside one transaction. Using async
        // awaits between puts lets the transaction auto-commit on Windows/Edge
        // and surfaces as UnknownError: Internal error.
        const CHUNK = 150;
        for (let i = 0; i < trees.length; i += CHUNK) {
            const chunk = trees.slice(i, i + CHUNK);
            await runTransaction(STORE_TREES, 'readwrite', (t) => {
                const store = t.objectStore(STORE_TREES);
                for (const tree of chunk) {
                    store.put(tree);
                }
            });
        }
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
        wasRecovered,
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
