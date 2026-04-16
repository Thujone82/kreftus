// Wikipedia link helpers.
//
// Tree names are formatted "$Species - $CommonName", e.g.
//   "Quercus garryana - Oregon white oak"
// We link the species portion to a Wikipedia search URL.

(function (global) {
    'use strict';

    const SEP = /\s+-\s+/; // " - " with flexible whitespace

    function splitSpeciesAndCommon(name) {
        if (!name) return { species: '', common: '' };
        const idx = name.search(SEP);
        if (idx === -1) return { species: name.trim(), common: '' };
        const species = name.slice(0, idx).trim();
        const common  = name.slice(idx).replace(SEP, '').trim();
        return { species, common };
    }

    function wikipediaUrlForSpecies(species) {
        const s = (species || '').trim();
        if (!s) return null;
        const q = encodeURIComponent(s).replace(/%20/g, '+');
        return `https://en.wikipedia.org/w/index.php?search=${q}`;
    }

    function wikipediaUrlForName(name) {
        const { species } = splitSpeciesAndCommon(name);
        return wikipediaUrlForSpecies(species || name);
    }

    global.HeritageWiki = {
        splitSpeciesAndCommon,
        wikipediaUrlForSpecies,
        wikipediaUrlForName
    };
})(window);
