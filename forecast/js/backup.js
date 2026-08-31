// Export / Import Settings for Forecast.
// Portable JSON backup of favorites (incl. per-location colors), preferences, and API keys.
// Weather caches and station catalogs are intentionally excluded.

(function (global) {
    'use strict';

    const FORMAT = 'forecast-settings';
    const FORMAT_VERSION = 1;

    /** Preference keys included in backups (no weather/station caches). */
    const SETTINGS_KEYS = [
        'forecastAccentPrimary',
        'forecastAccentSecondary',
        'forecastShowIrradiance',
        'forecastShowMagicHours',
        'forecastShowRadar',
        'forecastPerLocationColors',
        'forecastAutoUpdate',
        'forecastUpdateAll',
        'forecastEnableAqi',
        'forecastAirNowApiKey',
        'forecastAirNowApiKeyValid',
        'forecastEnableWildfire',
        'forecastWildfireRadiusMiles',
        'forecastFilterSmallWildfires',
        'forecastUseMetric',
        'forecastUse24h',
        'forecastUiDensity',
        'forecastFeelsLikeWbgt',
        'forecastLocationsDrawerOpen',
        'forecastCurrentMode',
        'forecastLastViewedLocation'
    ];

    function cloneJson(value) {
        return value == null ? value : JSON.parse(JSON.stringify(value));
    }

    function readSettingsFromStorage() {
        const settings = {};
        for (const key of SETTINGS_KEYS) {
            const raw = localStorage.getItem(key);
            if (raw !== null) settings[key] = raw;
        }
        return settings;
    }

    function normalizeCityState(location) {
        if (!location || typeof location !== 'object') return null;
        const city = String(location.city || '')
            .trim()
            .toLowerCase()
            .replace(/[^a-zA-Z0-9\s]/g, '');
        const state = String(location.state || '')
            .trim()
            .toUpperCase()
            .replace(/[^a-zA-Z0-9]/g, '');
        if (!city || !state) return null;
        return `${city}|${state}`;
    }

    function normalizeFavorite(raw) {
        if (!raw || typeof raw !== 'object') return null;
        const fav = {
            uid: raw.uid ? String(raw.uid) : undefined,
            key: raw.key ? String(raw.key) : undefined,
            name: raw.name != null ? String(raw.name) : '',
            searchQuery: raw.searchQuery != null ? String(raw.searchQuery) : (raw.name != null ? String(raw.name) : ''),
            location: raw.location && typeof raw.location === 'object' ? cloneJson(raw.location) : null
        };
        if (raw.customName != null && String(raw.customName).trim()) {
            fav.customName = String(raw.customName).trim();
        }
        if (raw.primaryColor) fav.primaryColor = String(raw.primaryColor);
        if (raw.secondaryColor) fav.secondaryColor = String(raw.secondaryColor);
        if (!fav.uid && !fav.key && !normalizeCityState(fav.location) && !fav.searchQuery) {
            return null;
        }
        return fav;
    }

    function favoriteMatchIndex(list, candidate) {
        if (!candidate) return -1;
        if (candidate.uid) {
            const byUid = list.findIndex((f) => f && f.uid === candidate.uid);
            if (byUid !== -1) return byUid;
        }
        const cs = normalizeCityState(candidate.location);
        if (cs) {
            const byCs = list.findIndex((f) => normalizeCityState(f && f.location) === cs);
            if (byCs !== -1) return byCs;
        }
        if (candidate.key) {
            const byKey = list.findIndex((f) => f && f.key === candidate.key);
            if (byKey !== -1) return byKey;
        }
        return -1;
    }

    function mergeFavoriteFields(local, backup) {
        const base = cloneJson(local) || {};
        const b = backup || {};
        const merged = {
            ...base,
            uid: base.uid || b.uid,
            key: base.key || b.key,
            name: (b.name != null && String(b.name).trim()) ? String(b.name) : (base.name || ''),
            searchQuery: (b.searchQuery != null && String(b.searchQuery).trim())
                ? String(b.searchQuery)
                : (base.searchQuery || base.name || ''),
            location: b.location || base.location || null
        };
        const backupName = b.customName != null ? String(b.customName).trim() : '';
        const localName = base.customName != null ? String(base.customName).trim() : '';
        if (backupName) merged.customName = backupName;
        else if (localName) merged.customName = localName;
        else delete merged.customName;

        if (b.primaryColor) merged.primaryColor = String(b.primaryColor);
        else if (base.primaryColor) merged.primaryColor = base.primaryColor;
        else delete merged.primaryColor;

        if (b.secondaryColor) merged.secondaryColor = String(b.secondaryColor);
        else if (base.secondaryColor) merged.secondaryColor = base.secondaryColor;
        else delete merged.secondaryColor;

        return merged;
    }

    function mergeFavoriteLists(localList, backupList) {
        const result = (localList || []).map((f) => cloneJson(f));
        for (const raw of backupList || []) {
            const backup = normalizeFavorite(raw);
            if (!backup) continue;
            const idx = favoriteMatchIndex(result, backup);
            if (idx === -1) result.push(backup);
            else result[idx] = mergeFavoriteFields(result[idx], backup);
        }
        return result;
    }

    function readFavoritesFromStorage() {
        try {
            const raw = localStorage.getItem('forecastFavorites');
            if (!raw) return [];
            const parsed = JSON.parse(raw);
            if (!Array.isArray(parsed)) return [];
            return parsed.map(normalizeFavorite).filter(Boolean);
        } catch (e) {
            console.warn('ForecastBackup: failed to read favorites', e);
            return [];
        }
    }

    function hasLocalUserData() {
        if (readFavoritesFromStorage().length > 0) return true;
        for (const key of SETTINGS_KEYS) {
            if (localStorage.getItem(key) !== null) return true;
        }
        return false;
    }

    function getAppVersionHint() {
        try {
            if (global.appState && appState.currentAppVersion) {
                return String(appState.currentAppVersion);
            }
        } catch (e) { /* ignore */ }
        const el = document.getElementById('configModalVersion');
        if (el && el.textContent) {
            const m = el.textContent.match(/v([\d.]+)/i);
            if (m) return m[1];
        }
        return undefined;
    }

    function buildBackup() {
        const favorites = readFavoritesFromStorage();
        return {
            format: FORMAT,
            formatVersion: FORMAT_VERSION,
            exportedAt: new Date().toISOString(),
            appVersion: getAppVersionHint(),
            favorites,
            settings: readSettingsFromStorage()
        };
    }

    function validateBackup(data) {
        if (!data || typeof data !== 'object') {
            throw new Error('Backup file is empty or not JSON.');
        }
        if (data.format !== FORMAT) {
            throw new Error('Not a Forecast settings backup file.');
        }
        const ver = Number(data.formatVersion);
        if (!Number.isFinite(ver) || ver < 1 || ver > FORMAT_VERSION) {
            throw new Error(`Unsupported backup format version (${data.formatVersion}).`);
        }
        if (!Array.isArray(data.favorites)) {
            throw new Error('Backup is missing the favorites list.');
        }
        if (!data.settings || typeof data.settings !== 'object' || Array.isArray(data.settings)) {
            throw new Error('Backup is missing settings.');
        }
        const favorites = [];
        for (const raw of data.favorites) {
            const fav = normalizeFavorite(raw);
            if (fav) favorites.push(fav);
        }
        const settings = {};
        for (const key of SETTINGS_KEYS) {
            if (Object.prototype.hasOwnProperty.call(data.settings, key)
                && data.settings[key] != null) {
                settings[key] = String(data.settings[key]);
            }
        }
        return {
            format: FORMAT,
            formatVersion: ver,
            exportedAt: data.exportedAt || null,
            appVersion: data.appVersion || null,
            favorites,
            settings
        };
    }

    function backupFilename(exportedAt) {
        const d = exportedAt ? new Date(exportedAt) : new Date();
        const iso = Number.isNaN(d.getTime()) ? new Date() : d;
        const yyyy = iso.getFullYear();
        const mm = String(iso.getMonth() + 1).padStart(2, '0');
        const dd = String(iso.getDate()).padStart(2, '0');
        return `forecast-settings-${yyyy}-${mm}-${dd}.json`;
    }

    function downloadJson(filename, text) {
        const blob = new Blob([text], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.rel = 'noopener';
        document.body.appendChild(a);
        a.click();
        a.remove();
        setTimeout(() => URL.revokeObjectURL(url), 2000);
    }

    async function shareOrDownload(backup) {
        const text = JSON.stringify(backup, null, 2);
        const filename = backupFilename(backup.exportedAt);
        const blob = new Blob([text], { type: 'application/json' });
        const file = new File([blob], filename, { type: 'application/json' });
        const count = (backup.favorites || []).length;

        if (navigator.canShare) {
            try {
                if (navigator.canShare({ files: [file] })) {
                    await navigator.share({
                        files: [file],
                        title: 'Forecast settings backup',
                        text: 'Favorites, colors, preferences, and keys for Forecast.'
                    });
                    return { method: 'share', filename, count };
                }
            } catch (err) {
                if (err && err.name === 'AbortError') {
                    return { method: 'cancelled', filename, count };
                }
                // Fall through to download.
            }
        }

        downloadJson(filename, text);
        return { method: 'download', filename, count };
    }

    async function exportState() {
        const backup = buildBackup();
        return shareOrDownload(backup);
    }

    function previewImport(backup) {
        const validated = validateBackup(backup);
        const localFavorites = readFavoritesFromStorage();
        const mergeFavorites = mergeFavoriteLists(localFavorites, validated.favorites);
        const currentLocations = localFavorites.length;
        const mergeLocations = mergeFavorites.length;
        const overwriteLocations = validated.favorites.length;
        const hasApiKey = !!(
            localStorage.getItem('forecastAirNowApiKey')
            || validated.settings.forecastAirNowApiKey
        );

        return {
            backup: validated,
            currentLocations,
            mergeLocations,
            overwriteLocations,
            overwriteLosesLocations: overwriteLocations < currentLocations,
            hasLocalData: hasLocalUserData(),
            hasApiKey,
            exportedAt: validated.exportedAt,
            backupLocations: validated.favorites.length
        };
    }

    function writeSettings(settings, mode) {
        const incoming = settings || {};
        if (mode === 'overwrite') {
            for (const key of SETTINGS_KEYS) {
                if (Object.prototype.hasOwnProperty.call(incoming, key)) {
                    localStorage.setItem(key, String(incoming[key]));
                } else {
                    localStorage.removeItem(key);
                }
            }
            return;
        }
        // merge: backup values win when present; leave other local keys alone
        for (const key of SETTINGS_KEYS) {
            if (Object.prototype.hasOwnProperty.call(incoming, key)) {
                localStorage.setItem(key, String(incoming[key]));
            }
        }
    }

    function applyBackup(backup, mode) {
        const validated = validateBackup(backup);
        const useMerge = mode === 'merge';
        const localFavorites = readFavoritesFromStorage();
        const nextFavorites = useMerge
            ? mergeFavoriteLists(localFavorites, validated.favorites)
            : validated.favorites.map((f) => cloneJson(f));

        localStorage.setItem('forecastFavorites', JSON.stringify(nextFavorites));
        writeSettings(validated.settings, useMerge ? 'merge' : 'overwrite');

        return {
            mode: useMerge ? 'merge' : 'overwrite',
            locations: nextFavorites.length,
            imported: validated.favorites.length,
            exportedAt: validated.exportedAt
        };
    }

    function readFileAsText(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(String(reader.result || ''));
            reader.onerror = () => reject(reader.error || new Error('Could not read file.'));
            reader.readAsText(file);
        });
    }

    async function importFromFile(file, mode) {
        if (!file) throw new Error('No file selected.');
        const text = await readFileAsText(file);
        let data;
        try {
            data = JSON.parse(text);
        } catch (e) {
            throw new Error('Backup file is not valid JSON.');
        }
        return applyBackup(data, mode || 'overwrite');
    }

    global.ForecastBackup = {
        FORMAT,
        FORMAT_VERSION,
        SETTINGS_KEYS,
        buildBackup,
        validateBackup,
        previewImport,
        exportState,
        importFromFile,
        applyBackup,
        hasLocalUserData
    };
})(window);
