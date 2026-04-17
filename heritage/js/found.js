// Found-trees panel: lists every tree the user has marked as found, most
// recent first, as a three-column table (Tree #, common name, find date).
// Tapping a row closes the modal and focuses that tree on the map, which
// opens the same info popup used everywhere else.

(function (global) {
    'use strict';

    function formatLocalDate(iso) {
        try {
            return new Date(iso).toLocaleString(undefined, {
                dateStyle: 'medium',
                timeStyle: 'short'
            });
        } catch (e) {
            return iso || '';
        }
    }

    // Extract only the "common name" half of a tree's stored name, which is
    // formatted "<species> - <common>" by the scraper. Falls back to the raw
    // name if there is no separator, so trees lacking a hyphen still render.
    function commonNameOf(tree) {
        if (tree && tree.commonName) return tree.commonName;
        const name = (tree && tree.name) || '';
        if (!name) return '';
        if (global.HeritageWiki && typeof HeritageWiki.splitSpeciesAndCommon === 'function') {
            const { common, species } = HeritageWiki.splitSpeciesAndCommon(name);
            return common || species || name;
        }
        const idx = name.search(/\s+-\s+/);
        return (idx === -1) ? name : name.slice(idx).replace(/\s+-\s+/, '').trim();
    }

    /**
     * Load every found tree from IndexedDB, sort by foundDate descending (newest
     * first), and fall back to name ordering if two rows share a timestamp or
     * are missing one.
     */
    async function getFoundTrees() {
        const trees = await HeritageDB.getAllTrees();
        const found = trees.filter((t) => t && t.found);
        found.sort((a, b) => {
            const at = a.foundDate ? Date.parse(a.foundDate) : 0;
            const bt = b.foundDate ? Date.parse(b.foundDate) : 0;
            if (bt !== at) return bt - at;
            const an = (a.name || a.id || '').toLowerCase();
            const bn = (b.name || b.id || '').toLowerCase();
            return an.localeCompare(bn);
        });
        return found;
    }

    async function countFound() {
        const trees = await HeritageDB.getAllTrees();
        let n = 0;
        for (const t of trees) { if (t && t.found) n++; }
        return n;
    }

    async function render() {
        const tbody = document.getElementById('foundTableBody');
        const subtitle = document.getElementById('foundSubtitle');
        if (!tbody) return;

        const rows = await getFoundTrees();
        tbody.innerHTML = '';

        if (subtitle) {
            subtitle.textContent = rows.length > 0
                ? `${rows.length} tree${rows.length === 1 ? '' : 's'} found \u2014 most recent first. Tap a row to view it on the map.`
                : 'No trees marked as found yet.';
        }

        if (rows.length === 0) {
            const tr = document.createElement('tr');
            const td = document.createElement('td');
            td.colSpan = 3;
            td.className = 'found-empty';
            td.textContent = 'You have not marked any trees as found yet.';
            tr.appendChild(td);
            tbody.appendChild(tr);
            return;
        }

        for (const tree of rows) {
            tbody.appendChild(buildRow(tree));
        }
    }

    function buildRow(tree) {
        const tr = document.createElement('tr');
        tr.className = 'found-row';
        tr.tabIndex = 0;
        tr.setAttribute('role', 'button');
        tr.setAttribute('aria-label',
            `View tree #${tree.id} ${commonNameOf(tree)} on the map`);

        const idCell = document.createElement('td');
        idCell.className = 'found-col-id';
        idCell.textContent = `#${tree.id}`;

        const nameCell = document.createElement('td');
        nameCell.className = 'found-col-name';
        nameCell.textContent = commonNameOf(tree) || tree.name || '';

        const dateCell = document.createElement('td');
        dateCell.className = 'found-col-date';
        dateCell.textContent = tree.foundDate ? formatLocalDate(tree.foundDate) : '';

        tr.appendChild(idCell);
        tr.appendChild(nameCell);
        tr.appendChild(dateCell);

        const activate = () => onSelect(tree);
        tr.addEventListener('click', activate);
        tr.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                activate();
            }
        });
        return tr;
    }

    async function onSelect(tree) {
        try { HeritageUI.closeModal('foundModal'); } catch (e) { /* no-op */ }
        if (HeritageMap && typeof HeritageMap.focusTree === 'function') {
            await HeritageMap.focusTree(tree.id);
        } else if (HeritageMap && typeof HeritageMap.openInfoForTree === 'function') {
            HeritageMap.openInfoForTree(tree.id);
        }
    }

    global.HeritageFound = {
        render,
        getFoundTrees,
        countFound
    };
})(window);
