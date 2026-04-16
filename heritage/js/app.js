// Main application wiring.
//
// Boot sequence:
//   1. If no Google Maps API key in localStorage -> show setup screen.
//      On save: persist key, continue boot.
//   2. Open IndexedDB.
//   3. If DB empty: fetch heritage/data/trees.json and initial-load it.
//   4. Load Google Maps JS API with the saved key.
//   5. Init map, render markers for all trees that have coords.
//   6. Kick off geocoding of any pending trees in the background.
//   7. Camera auto-fit based on user location vs. Portland.
//   8. Wire Nearby, Check-for-updates, Settings, Update banner.

(function (global) {
    'use strict';

    const KEY_API = 'pdxHeritageGoogleApiKey';
    const APP_VERSION = '1.0.0';

    const state = {
        apiKey: null,
        mapsReady: false,
        geocodingActive: false
    };

    document.addEventListener('DOMContentLoaded', init);

    async function init() {
        HeritageSW.register();
        wireStaticUi();

        state.apiKey = (localStorage.getItem(KEY_API) || '').trim();
        if (!state.apiKey) {
            showSetup();
            return;
        }

        try {
            await bootWithKey();
        } catch (err) {
            console.error('Boot failed:', err);
            HeritageUI.toast('Something went wrong starting up. Check your API key in Settings.', 5000);
            showSetup(err && err.message);
        }
    }

    // -------------------------------------------------------------------
    // Setup screen
    // -------------------------------------------------------------------

    function showSetup(errMsg) {
        const screen = document.getElementById('setupScreen');
        if (!screen) return;
        screen.classList.remove('hidden');
        const errEl = document.getElementById('setupError');
        if (errMsg && errEl) {
            errEl.textContent = errMsg;
            errEl.classList.remove('hidden');
        } else if (errEl) {
            errEl.classList.add('hidden');
        }
        const input = document.getElementById('apiKeyInput');
        if (input) {
            input.value = state.apiKey || '';
            setTimeout(() => input.focus(), 50);
        }
    }

    function hideSetup() {
        const screen = document.getElementById('setupScreen');
        if (screen) screen.classList.add('hidden');
    }

    function wireStaticUi() {
        const saveBtn = document.getElementById('saveApiKeyBtn');
        if (saveBtn) saveBtn.addEventListener('click', onSaveApiKey);
        const input = document.getElementById('apiKeyInput');
        if (input) input.addEventListener('keydown', (e) => { if (e.key === 'Enter') onSaveApiKey(); });

        const settingsBtn = document.getElementById('settingsBtn');
        if (settingsBtn) settingsBtn.addEventListener('click', openSettings);
        document.querySelectorAll('[data-close-modal]').forEach((el) => {
            el.addEventListener('click', () => HeritageUI.closeModal('settingsModal'));
        });

        const nearbyBtn = document.getElementById('nearbyBtn');
        if (nearbyBtn) nearbyBtn.addEventListener('click', HeritageUI.openNearby);
        const closeNearbyBtn = document.getElementById('closeNearbyBtn');
        if (closeNearbyBtn) closeNearbyBtn.addEventListener('click', HeritageUI.closeNearby);

        const recenterBtn = document.getElementById('recenterBtn');
        if (recenterBtn) recenterBtn.addEventListener('click', onRecenter);

        const checkUpdates = document.getElementById('checkUpdatesBtn');
        if (checkUpdates) checkUpdates.addEventListener('click', onCheckUpdates);

        const modalSaveKey = document.getElementById('settingsSaveApiKey');
        if (modalSaveKey) modalSaveKey.addEventListener('click', onModalSaveKey);

        const modalCheck = document.getElementById('modalCheckUpdates');
        if (modalCheck) modalCheck.addEventListener('click', onCheckUpdates);

        const modalRe = document.getElementById('modalRegeocode');
        if (modalRe) modalRe.addEventListener('click', onRetryFailedGeocodes);

        const modalAppUpdate = document.getElementById('modalCheckAppUpdate');
        if (modalAppUpdate) modalAppUpdate.addEventListener('click', () => {
            HeritageSW.requestUpdateCheck();
            HeritageUI.toast('Checked for app updates.');
        });
    }

    async function onSaveApiKey() {
        const input = document.getElementById('apiKeyInput');
        const key = (input && input.value || '').trim();
        if (!key || !/^AIza[\w-]{10,}$/.test(key) && key.length < 20) {
            const err = document.getElementById('setupError');
            if (err) {
                err.textContent = 'That does not look like a Google Maps API key. It should start with "AIza".';
                err.classList.remove('hidden');
            }
            return;
        }
        localStorage.setItem(KEY_API, key);
        state.apiKey = key;
        try {
            await bootWithKey();
        } catch (err) {
            console.error(err);
            showSetup(err && err.message);
        }
    }

    async function onModalSaveKey() {
        const input = document.getElementById('settingsApiKey');
        const key = (input && input.value || '').trim();
        if (!key) { HeritageUI.toast('Enter a key first.'); return; }
        localStorage.setItem(KEY_API, key);
        HeritageUI.toast('API key saved. Reloading\u2026');
        setTimeout(() => window.location.reload(), 700);
    }

    // -------------------------------------------------------------------
    // Boot pipeline
    // -------------------------------------------------------------------

    async function bootWithKey() {
        hideSetup();
        await HeritageDB.open();

        // Initial data load if DB is empty.
        const count = await HeritageDB.countTrees();
        if (count === 0) {
            HeritageUI.showProgress('Loading tree list\u2026');
            HeritageUI.updateProgress(0, 1, 'Loading tree list');
            const snap = await HeritageSync.fetchSnapshot();
            const { inserted } = await HeritageSync.initialLoad(snap);
            HeritageUI.updateProgress(1, 1, 'Loading tree list');
            HeritageUI.hideProgress();
            HeritageUI.toast(`Loaded ${inserted} trees.`);
        }

        // Load Google Maps JS API with the saved key, then init map.
        await loadGoogleMaps(state.apiKey);
        state.mapsReady = true;
        await HeritageMap.init('map');
        HeritageMap.setOnTreeUpdate(onTreeRecordChanged);

        // Render whatever we already have coordinates for.
        const initialTrees = await HeritageDB.getAllTrees();
        HeritageMap.renderTrees(initialTrees);

        // Resolve user location (best-effort) and place marker + auto-fit camera.
        const user = await HeritageMap.getUserLocation(7000);
        if (user) HeritageMap.placeUserMarker();
        HeritageMap.autoFit(user, initialTrees);

        // Kick off geocoding of any pending trees in the background.
        runBackgroundGeocode();
    }

    function loadGoogleMaps(apiKey) {
        return new Promise((resolve, reject) => {
            if (global.google && global.google.maps) return resolve();
            const cbName = '__heritageMapsLoaded';
            global[cbName] = () => {
                try { delete global[cbName]; } catch (e) { global[cbName] = undefined; }
                resolve();
            };
            const s = document.createElement('script');
            s.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(apiKey)}&libraries=geometry&callback=${cbName}&loading=async`;
            s.async = true;
            s.defer = true;
            s.onerror = () => reject(new Error('Failed to load Google Maps. Check the API key.'));
            document.head.appendChild(s);
        });
    }

    async function runBackgroundGeocode(opts) {
        if (state.geocodingActive) return;
        state.geocodingActive = true;
        const includeFailed = !!(opts && opts.includeFailed);
        try {
            const trees = await HeritageDB.getAllTrees();
            const pending = trees.filter((t) =>
                (t.geocodeStatus === 'pending' || (includeFailed && t.geocodeStatus === 'failed')) &&
                t.removed == null
            );
            if (pending.length === 0) return;

            HeritageUI.showProgress('Geocoding trees');
            HeritageUI.updateProgress(0, pending.length, 'Geocoding trees');

            const result = await HeritageGeocode.runAll({
                includeFailed,
                onProgress: ({ done, total }) => {
                    HeritageUI.updateProgress(done, total, 'Geocoding trees');
                },
                onTree: (tree) => {
                    HeritageMap.updateTreeMarker(tree);
                }
            });

            HeritageUI.hideProgress();
            if (result.failed > 0) {
                HeritageUI.toast(`Geocoded ${result.succeeded} / ${result.attempted} trees. ${result.failed} failed \u2014 you can retry from Settings.`, 5000);
            } else if (result.attempted > 0) {
                HeritageUI.toast(`Geocoded ${result.succeeded} tree${result.succeeded === 1 ? '' : 's'}.`);
            }
        } catch (err) {
            console.error('Geocoding failed:', err);
            HeritageUI.hideProgress();
            HeritageUI.toast('Geocoding failed. See console for details.', 4500);
        } finally {
            state.geocodingActive = false;
        }
    }

    function onTreeRecordChanged(_tree) {
        // Could update stats panel if open; keep it cheap.
        if (!document.getElementById('settingsModal').classList.contains('hidden')) {
            HeritageUI.refreshStats();
        }
    }

    // -------------------------------------------------------------------
    // Action bar / modal handlers
    // -------------------------------------------------------------------

    async function onRecenter() {
        const user = HeritageMap.getCachedUserLocation() || await HeritageMap.getUserLocation(5000);
        if (user) HeritageMap.placeUserMarker();
        const trees = await HeritageDB.getAllTrees();
        HeritageMap.recenter(user, trees);
    }

    async function onCheckUpdates() {
        try {
            HeritageUI.showProgress('Checking for updates\u2026');
            const snap = await HeritageSync.fetchSnapshot();
            const summary = await HeritageSync.mergeUpdate(snap);
            HeritageUI.hideProgress();

            // Refresh markers for the full set (cheap at ~400).
            const trees = await HeritageDB.getAllTrees();
            HeritageMap.renderTrees(trees);

            const parts = [];
            if (summary.added)           parts.push(`+${summary.added} new`);
            if (summary.newlyRemoved)    parts.push(`${summary.newlyRemoved} newly removed`);
            if (summary.locationChanged) parts.push(`${summary.locationChanged} re-geocoding`);
            if (parts.length === 0) parts.push('no changes to the list');
            HeritageUI.toast(`Check complete \u2014 ${parts.join(', ')}. Your notes and found dates are untouched.`, 5000);

            // If anything was added or re-geocoded, run the geocoder for the new queue.
            if (summary.added > 0 || summary.locationChanged > 0) {
                runBackgroundGeocode();
            }
        } catch (err) {
            console.error(err);
            HeritageUI.hideProgress();
            HeritageUI.toast('Could not check for updates. Are you offline?', 4500);
        }
    }

    function onRetryFailedGeocodes() {
        HeritageUI.toast('Retrying failed geocodes\u2026');
        runBackgroundGeocode({ includeFailed: true });
    }

    async function openSettings() {
        const input = document.getElementById('settingsApiKey');
        if (input) input.value = state.apiKey || '';
        const verEl = document.getElementById('statVersion');
        if (verEl) verEl.textContent = APP_VERSION;
        const swEl = document.getElementById('statSw');
        if (swEl) {
            const v = await HeritageSW.askForVersion();
            swEl.textContent = v ? `active (v${v})` : 'not registered yet';
        }
        await HeritageUI.refreshStats();
        HeritageUI.openModal('settingsModal');
    }

    global.HeritageApp = {
        _state: state,
        runBackgroundGeocode
    };
})(window);
