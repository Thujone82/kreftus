// Main application wiring.
//
// Boot sequence (no API key required - basemap via Leaflet + CARTO Voyager):
//   1. Open IndexedDB.
//   2. If DB empty: fetch heritage/data/trees.json and initial-load it.
//      (Tree coordinates ship pre-geocoded from heritage/heritage.ps1, so this
//      is immediate - no live geocoding needed for the common case.)
//   3. Init Leaflet map and render markers for all trees that have coords.
//   4. Silent coord backfill: if any tree is missing coords (e.g. upgrading
//      from a pre-0.2 build that geocoded live), re-merge the bundled JSON
//      to pick up the pre-resolved coordinates.
//   5. Camera auto-fit based on user location vs. Portland.
//   6. Fallback live geocoder runs ONLY for trees the scraper couldn't
//      resolve, and stays silent when nothing is pending.
//   7. Wire Nearby, Check-for-updates, Settings, Update banner.
//
// One-time migration: older builds stored a Google Maps API key in
// localStorage under 'pdxHeritageGoogleApiKey'. We clean that up on first
// boot so nothing lingers.

(function (global) {
    'use strict';

    const LEGACY_KEY_API = 'pdxHeritageGoogleApiKey';
    const APP_VERSION = '1.0.8';

    const state = {
        mapReady: false,
        geocodingActive: false
    };

    /** Stored for deferred `beforeinstallprompt` (Chromium/Edge/Android). */
    let deferredInstallPrompt = null;

    function isPwaStandalone() {
        return window.matchMedia('(display-mode: standalone)').matches ||
            window.navigator.standalone === true ||
            document.referrer.includes('android-app://');
    }

    document.addEventListener('DOMContentLoaded', init);

    async function init() {
        HeritageSW.register();
        wireStaticUi();

        // Drop any leftover Google Maps API key from an older install.
        try { localStorage.removeItem(LEGACY_KEY_API); } catch (e) { /* ignore */ }

        try {
            await boot();
        } catch (err) {
            console.error('Boot failed:', err);
            HeritageUI.toast('Something went wrong starting up. See console for details.', 5000);
        }
    }

    function wireStaticUi() {
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

        const modalAppUpdate = document.getElementById('modalCheckAppUpdate');
        if (modalAppUpdate) modalAppUpdate.addEventListener('click', onCheckAppUpdate);

        const modalMapZoom = document.getElementById('modalMapZoomToggle');
        if (modalMapZoom) {
            modalMapZoom.addEventListener('click', () => { void HeritageMap.toggleModalMapZoom(); });
        }

        wirePwaInstall();
    }

    function wirePwaInstall() {
        const installBtn = document.getElementById('installPwaBtn');
        const iosHint = document.getElementById('pwaIosInstallHint');

        const hideInstall = () => {
            if (installBtn) installBtn.classList.add('hidden');
        };

        if (isPwaStandalone()) {
            hideInstall();
            deferredInstallPrompt = null;
        }

        window.addEventListener('beforeinstallprompt', (e) => {
            e.preventDefault();
            if (isPwaStandalone()) {
                deferredInstallPrompt = null;
                hideInstall();
                return;
            }
            deferredInstallPrompt = e;
            if (installBtn) installBtn.classList.remove('hidden');
        });

        window.addEventListener('appinstalled', () => {
            deferredInstallPrompt = null;
            hideInstall();
        });

        if (installBtn) {
            installBtn.addEventListener('click', async () => {
                if (!deferredInstallPrompt) return;
                try {
                    deferredInstallPrompt.prompt();
                    await deferredInstallPrompt.userChoice;
                } catch (err) {
                    console.warn('Install prompt failed:', err);
                }
                deferredInstallPrompt = null;
                installBtn.classList.add('hidden');
            });
        }

        const isIos = /iPad|iPhone|iPod/.test(navigator.userAgent) ||
            (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
        if (iosHint && isIos && !isPwaStandalone()) {
            iosHint.textContent =
                'To install on iPhone or iPad: tap Share, then Add to Home Screen.';
            iosHint.classList.remove('hidden');
        }
    }

    // -------------------------------------------------------------------
    // Boot pipeline
    // -------------------------------------------------------------------

    async function boot() {
        await HeritageDB.open();

        // Initial data load if DB is empty.
        const count = await HeritageDB.countTrees();
        let snap = null;
        if (count === 0) {
            HeritageUI.showProgress('Loading tree list\u2026');
            HeritageUI.updateProgress(0, 1, 'Loading tree list');
            snap = await HeritageSync.fetchSnapshot();
            const { inserted } = await HeritageSync.initialLoad(snap);
            HeritageUI.updateProgress(1, 1, 'Loading tree list');
            HeritageUI.hideProgress();
            HeritageUI.toast(`Loaded ${inserted} trees.`);
        }

        // Init Leaflet map.
        await HeritageMap.init('map');
        state.mapReady = true;
        HeritageMap.setOnTreeUpdate(onTreeRecordChanged);

        // Render whatever we already have coordinates for.
        let initialTrees = await HeritageDB.getAllTrees();
        HeritageMap.renderTrees(initialTrees);

        // Silent coord backfill: the bundled snapshot ships pre-geocoded by
        // heritage.ps1 (via OpenStreetMap Nominatim). If any non-removed tree
        // in the DB is still missing coords (e.g. the user is upgrading from a
        // version that geocoded live), pull them straight from the JSON rather
        // than hitting a geocoding service in the browser.
        const needsBackfill = initialTrees.some((t) =>
            (t.lat == null || t.lng == null) && t.removed == null &&
            t.geocodeStatus !== 'skipped-no-address'
        );
        if (needsBackfill) {
            try {
                if (!snap) snap = await HeritageSync.fetchSnapshot();
                const summary = await HeritageSync.mergeUpdate(snap);
                if (summary.updated > 0 || summary.added > 0) {
                    initialTrees = await HeritageDB.getAllTrees();
                    HeritageMap.renderTrees(initialTrees);
                }
            } catch (e) {
                // Offline or snapshot missing; fall back to the live geocoder below.
            }
        }

        // Resolve user location (best-effort) and place marker + auto-fit camera.
        const user = await HeritageMap.getUserLocation(7000);
        if (user) {
            HeritageMap.placeUserMarker();
            HeritageMap.startLiveLocationUpdates();
        }
        HeritageMap.autoFit(user, initialTrees);

        // Fallback live geocoder: only fires if the snapshot didn't cover a tree.
        // Stays silent when the backfill (above) handled everything.
        runBackgroundGeocode();
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

    async function onTreeRecordChanged(_tree) {
        const modal = document.getElementById('settingsModal');
        if (modal && !modal.classList.contains('hidden')) {
            await HeritageUI.refreshStats();
            await HeritageMap.syncModalZoomToggleButton();
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

    // "Check for app update" does two things in one tap:
    //   1. Refresh the tree database from data/trees.json (bypassing the SW
    //      data cache) and diff-merge it into IndexedDB without touching the
    //      user's found marks or notes.
    //   2. Ping the service worker to check for a new app shell; if there is
    //      one, sw-register surfaces the "New app version available" banner.
    async function onCheckAppUpdate() {
        const btn = document.getElementById('modalCheckAppUpdate');
        if (btn) btn.disabled = true;

        let dataSummary = null;
        try {
            HeritageUI.showProgress('Refreshing tree database\u2026');
            // fetchSnapshot already appends a ?t= bust and uses cache: 'no-cache';
            // the SW serves trees.json network-first, so this always hits the origin.
            const snap = await HeritageSync.fetchSnapshot();
            dataSummary = await HeritageSync.mergeUpdate(snap);
            const trees = await HeritageDB.getAllTrees();
            HeritageMap.renderTrees(trees);
            await HeritageUI.refreshStats();
            await HeritageMap.syncModalZoomToggleButton();
            if (dataSummary.added > 0 || dataSummary.locationChanged > 0) {
                // Extremely rare with server-side geocoding, but keep the
                // silent fallback alive just in case a snapshot ships a
                // tree without coordinates.
                runBackgroundGeocode();
            }
        } catch (err) {
            console.error('Tree database refresh failed:', err);
        } finally {
            HeritageUI.hideProgress();
        }

        try {
            HeritageSW.requestUpdateCheck();
        } catch (err) {
            console.error('App update check failed:', err);
        }

        const parts = [];
        if (dataSummary) {
            if (dataSummary.added)        parts.push(`+${dataSummary.added} new tree${dataSummary.added === 1 ? '' : 's'}`);
            if (dataSummary.newlyRemoved) parts.push(`${dataSummary.newlyRemoved} newly removed`);
            if (dataSummary.updated && !dataSummary.added && !dataSummary.newlyRemoved) {
                parts.push(`${dataSummary.updated} updated`);
            }
            if (parts.length === 0) parts.push('tree database is up to date');
        } else {
            parts.push('could not refresh tree database (offline?)');
        }
        parts.push('checked for app update');
        HeritageUI.toast(parts.join(' \u2014 ') + '.', 5000);

        if (btn) btn.disabled = false;
    }

    async function openSettings() {
        const verEl = document.getElementById('statVersion');
        if (verEl) verEl.textContent = APP_VERSION;
        const swEl = document.getElementById('statSw');
        if (swEl) {
            const v = await HeritageSW.askForVersion();
            swEl.textContent = v ? `active (v${v})` : 'not registered yet';
        }
        await HeritageUI.refreshStats();
        await HeritageMap.syncModalZoomToggleButton();
        HeritageUI.openModal('settingsModal');
    }

    global.HeritageApp = {
        _state: state,
        runBackgroundGeocode
    };
})(window);
