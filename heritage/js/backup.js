// Import / export of user-owned app state (found marks, find dates, notes).
// Canonical registry fields stay in data/trees.json; this file is the portable
// backup you can move between devices or restore after clearing site data.

(function (global) {
    'use strict';

    const FORMAT = 'pdx-heritage-state';
    const FORMAT_VERSION = 1;

    function padTreeId(id) {
        const n = parseInt(String(id == null ? '' : id).replace(/[^\d]/g, ''), 10);
        if (!Number.isFinite(n) || n <= 0) return String(id == null ? '' : id);
        return String(n).padStart(3, '0');
    }

    function hasUserData(tree) {
        if (!tree) return false;
        if (tree.found) return true;
        const notes = String(tree.notes == null ? '' : tree.notes).trim();
        return notes.length > 0;
    }

    function normalizeUserEntry(raw) {
        if (!raw || typeof raw !== 'object') return null;
        const id = padTreeId(raw.id);
        if (!id) return null;
        const found = !!raw.found;
        const notes = String(raw.notes == null ? '' : raw.notes);
        let foundDate = raw.foundDate == null || raw.foundDate === '' ? null : String(raw.foundDate);
        if (found && !foundDate) foundDate = new Date().toISOString();
        if (!found) foundDate = null;
        return { id, found, foundDate, notes };
    }

    function earlierIso(a, b) {
        if (!a) return b || null;
        if (!b) return a;
        return a <= b ? a : b;
    }

    function mergeNotes(localNotes, backupNotes) {
        const a = String(localNotes == null ? '' : localNotes).trim();
        const b = String(backupNotes == null ? '' : backupNotes).trim();
        if (!a) return backupNotes || '';
        if (!b) return localNotes || '';
        if (a === b) return localNotes || '';
        if (a.includes(b)) return localNotes || '';
        if (b.includes(a)) return backupNotes || '';
        return `${a}\n\n${b}`;
    }

    function mergeUserFields(local, backup) {
        const localFound = !!local.found;
        const backupFound = !!(backup && backup.found);
        const found = localFound || backupFound;
        let foundDate = null;
        if (found) {
            foundDate = earlierIso(
                localFound ? (local.foundDate || null) : null,
                backupFound ? (backup.foundDate || null) : null
            ) || new Date().toISOString();
        }
        const notes = backup
            ? mergeNotes(local.notes, backup.notes)
            : (local.notes || '');
        return { found, foundDate, notes };
    }

    function overwriteUserFields(backup) {
        if (!backup) {
            return { found: false, foundDate: null, notes: '' };
        }
        return {
            found: !!backup.found,
            foundDate: backup.found ? (backup.foundDate || null) : null,
            notes: backup.notes || ''
        };
    }

    async function buildBackup() {
        const trees = await HeritageDB.getAllTrees();
        const userTrees = trees
            .filter(hasUserData)
            .map((t) => ({
                id: padTreeId(t.id),
                found: !!t.found,
                foundDate: t.foundDate || null,
                notes: t.notes || ''
            }))
            .sort((a, b) => a.id.localeCompare(b.id, undefined, { numeric: true }));

        const appVersion = (global.HeritageApp && global.HeritageApp.version)
            || (document.getElementById('statVersion') || {}).textContent
            || '';

        return {
            format: FORMAT,
            formatVersion: FORMAT_VERSION,
            exportedAt: new Date().toISOString(),
            appVersion: String(appVersion || '').trim() || undefined,
            trees: userTrees
        };
    }

    function validateBackup(data) {
        if (!data || typeof data !== 'object') {
            throw new Error('Backup file is empty or not JSON.');
        }
        if (data.format !== FORMAT) {
            throw new Error('Not a PDX Heritage Trees backup file.');
        }
        const ver = Number(data.formatVersion);
        if (!Number.isFinite(ver) || ver < 1 || ver > FORMAT_VERSION) {
            throw new Error(`Unsupported backup format version (${data.formatVersion}).`);
        }
        if (!Array.isArray(data.trees)) {
            throw new Error('Backup is missing the trees list.');
        }
        const entries = [];
        for (const raw of data.trees) {
            const entry = normalizeUserEntry(raw);
            if (entry) entries.push(entry);
        }
        return {
            format: FORMAT,
            formatVersion: ver,
            exportedAt: data.exportedAt || null,
            appVersion: data.appVersion || null,
            trees: entries
        };
    }

    function backupFilename(exportedAt) {
        const d = exportedAt ? new Date(exportedAt) : new Date();
        const iso = Number.isNaN(d.getTime()) ? new Date() : d;
        const yyyy = iso.getFullYear();
        const mm = String(iso.getMonth() + 1).padStart(2, '0');
        const dd = String(iso.getDate()).padStart(2, '0');
        return `pdx-heritage-backup-${yyyy}-${mm}-${dd}.json`;
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

        if (navigator.canShare) {
            try {
                if (navigator.canShare({ files: [file] })) {
                    await navigator.share({
                        files: [file],
                        title: 'PDX Heritage Trees backup',
                        text: 'Found marks and notes backup for PDX Heritage Trees.'
                    });
                    return { method: 'share', filename, count: backup.trees.length };
                }
            } catch (err) {
                if (err && err.name === 'AbortError') {
                    return { method: 'cancelled', filename, count: backup.trees.length };
                }
                // Fall through to download.
            }
        }

        downloadJson(filename, text);
        return { method: 'download', filename, count: backup.trees.length };
    }

    async function exportState() {
        const backup = await buildBackup();
        return shareOrDownload(backup);
    }

    /**
     * Compare local DB trees against a validated backup and predict outcomes.
     * @param {object} backup validated backup
     * @param {object[]} [localTrees] optional preloaded trees
     */
    async function previewImport(backup, localTrees) {
        const validated = validateBackup(backup);
        const trees = localTrees || await HeritageDB.getAllTrees();
        const byId = new Map(validated.trees.map((t) => [t.id, t]));

        let currentFound = 0;
        let currentWithNotes = 0;
        let mergeFound = 0;
        let overwriteFound = 0;

        for (const tree of trees) {
            const id = padTreeId(tree.id);
            const u = byId.get(id);
            if (tree.found) currentFound++;
            if (String(tree.notes || '').trim()) currentWithNotes++;

            const merged = mergeUserFields(tree, u || null);
            const overwritten = overwriteUserFields(u || null);
            if (merged.found) mergeFound++;
            if (overwritten.found) overwriteFound++;
        }

        return {
            backup: validated,
            currentFound,
            currentWithNotes,
            hasLocalData: currentFound > 0 || currentWithNotes > 0,
            mergeFound,
            overwriteFound,
            overwriteLosesFound: overwriteFound < currentFound,
            backupFound: validated.trees.filter((t) => t.found).length,
            backupWithNotes: validated.trees.filter((t) => String(t.notes || '').trim()).length,
            exportedAt: validated.exportedAt
        };
    }

    async function applyBackup(backup, mode) {
        const validated = validateBackup(backup);
        const byId = new Map(validated.trees.map((t) => [t.id, t]));
        const trees = await HeritageDB.getAllTrees();
        const now = new Date().toISOString();
        const toWrite = [];
        const useMerge = mode === 'merge';

        for (const tree of trees) {
            const id = padTreeId(tree.id);
            const u = byId.get(id);
            const next = useMerge
                ? mergeUserFields(tree, u || null)
                : overwriteUserFields(u || null);
            const same =
                !!tree.found === next.found &&
                (tree.foundDate || null) === (next.foundDate || null) &&
                String(tree.notes || '') === String(next.notes || '');
            if (same) continue;
            toWrite.push({
                ...tree,
                found: next.found,
                foundDate: next.foundDate,
                notes: next.notes,
                lastUpdatedAt: now
            });
        }

        if (toWrite.length > 0) {
            await HeritageDB.putManyTrees(toWrite);
        }

        const after = await HeritageDB.getAllTrees();
        const found = after.reduce((n, t) => n + (t.found ? 1 : 0), 0);

        return {
            mode: useMerge ? 'merge' : 'overwrite',
            updated: toWrite.length,
            imported: validated.trees.length,
            found,
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

    global.HeritageBackup = {
        FORMAT,
        FORMAT_VERSION,
        buildBackup,
        validateBackup,
        previewImport,
        exportState,
        importFromFile,
        applyBackup
    };
})(window);
