console.log("utils.js loaded");

const utils = {
    generateId: () => {
        return Math.random().toString(36).substr(2, 9);
    },

    formatTimeAgo: (timestamp) => {
        if (!timestamp) return 'N/A';
        const now = Date.now();
        const seconds = Math.round((now - timestamp) / 1000);

        if (seconds < 60) return `${seconds}s ago`;

        const minutes = Math.round(seconds / 60);
        if (minutes < 120) return `${minutes}m ago`;

        const hours = Math.round(minutes / 60);
        return `${hours}h ago`;
    },

    formatAiResponseToHtml: (text) => {
        if (!text) return '';

        const lines = text.split('\n');
        let processedLines = [];
        let inUnorderedList = false;
        let inOrderedList = false;
        let inBlockquote = false;

        const applyInlineFormatting = (lineContent) => {
            // Links: text
            lineContent = lineContent.replace(/\[([^\]]+?)\]\(([^)]+?)\)/g, '<a href="$2" target="_blank">$1</a>');
            // Inline Code: `code`
            lineContent = lineContent.replace(/`([^`]+?)`/g, '<code>$1</code>');
            // Bold and Italic: ***text***
            lineContent = lineContent.replace(/\*\*\*([^\*]+?)\*\*\*/g, '<b><em>$1</em></b>');
            // Bold: **text**
            lineContent = lineContent.replace(/\*\*(.+?)\*\*/g, '<b>$1</b>');
            // Italic: *text*
            lineContent = lineContent.replace(/\*([^\*]+?)\*/g, '<em>$1</em>');
            // Italic: _text_
            lineContent = lineContent.replace(/_([^_]+?)_/g, '<em>$1</em>');
            return lineContent;
        };
        
        const closeAllOpenBlocks = () => {
            if (inUnorderedList) {
                processedLines.push('</ul>');
                inUnorderedList = false;
            }
            if (inOrderedList) {
                processedLines.push('</ol>');
                inOrderedList = false;
            }
            if (inBlockquote) {
                processedLines.push('</blockquote>');
                inBlockquote = false;
            }
        };

        for (let i = 0; i < lines.length; i++) {
            let line = lines[i];
            // Process the line once to get its effective content after trimming special whitespace
            let effectiveContent = line.replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();

            if (effectiveContent.length === 0) { // Line is effectively blank
                // If in a multi-line block, a blank line might end it if the next line isn't part of it.
                // This logic attempts to preserve list/blockquote continuity if a blank line is followed by a continuation.
                if (inUnorderedList) {
                    let nextLineIsListItem = false;
                    if (i + 1 < lines.length && lines[i+1].trim().startsWith('* ')) nextLineIsListItem = true;
                    if (!nextLineIsListItem) closeAllOpenBlocks(); else continue;
                } else if (inOrderedList) {
                    let nextLineIsOLItem = false;
                    if (i + 1 < lines.length && /^\d+\.\s/.test(lines[i+1].trim())) nextLineIsOLItem = true;
                    if (!nextLineIsOLItem) closeAllOpenBlocks(); else continue;
                } else if (inBlockquote) {
                    let nextLineIsBQItem = false;
                    if (i + 1 < lines.length && lines[i+1].trim().startsWith('>')) nextLineIsBQItem = true;
                    if (!nextLineIsBQItem) closeAllOpenBlocks(); else continue;
                }
                continue; // Always continue to the next line after processing a blank line.
            } else { // Line has actual content
                // Headings (H1-H6)
                let headingMatch = effectiveContent.match(/^(#{1,6})\s+(.*)/);
                if (headingMatch) {
                    closeAllOpenBlocks();
                    let level = headingMatch[1].length;
                    let headerContent = headingMatch[2].trim();
                    processedLines.push(`<h${level}>${applyInlineFormatting(headerContent)}</h${level}>`);
                    continue;
                }

                // Horizontal Rule (---, ***, ___)
                if (/^(\-{3,}|\*{3,}|_{3,})$/.test(effectiveContent)) {
                    closeAllOpenBlocks();
                    processedLines.push('<hr>');
                    continue;
                }


                // Blockquotes
                if (effectiveContent.startsWith('>')) {
                    if (!inBlockquote) {
                        closeAllOpenBlocks(); // Close other blocks
                        processedLines.push('<blockquote>');
                        inBlockquote = true;
                    }
                    let quoteContent = effectiveContent.substring(1).trim();
                    if (quoteContent) { // Avoid empty <p> for lines like "> "
                        processedLines.push(`<p>${applyInlineFormatting(quoteContent)}</p>`);
                    }
                    continue;
                }
                if (inBlockquote && !effectiveContent.startsWith('>')) { // Exiting blockquote
                    closeAllOpenBlocks(); // This closes blockquote
                }

                // Unordered Lists
                if (effectiveContent.startsWith('* ')) {
                    if (!inUnorderedList) {
                        closeAllOpenBlocks(); // Close other blocks
                        processedLines.push('<ul>');
                        inUnorderedList = true;
                    }
                    let listItemContent = effectiveContent.substring(2).trim(); // Get content after "* "
                    if (listItemContent.length > 0) {
                        processedLines.push(`<li>${applyInlineFormatting(listItemContent)}</li>`);
                    } // Empty list items (e.g. "* ") are skipped
                    continue;
                }
                if (inUnorderedList && !effectiveContent.startsWith('* ')) { // Exiting Unordered List
                    closeAllOpenBlocks();
                }

                // Ordered Lists
                let orderedListMatch = effectiveContent.match(/^(\d+)\.\s+(.*)/);
                if (orderedListMatch) {
                    if (!inOrderedList) {
                        closeAllOpenBlocks(); // Close other blocks
                        processedLines.push('<ol>');
                        inOrderedList = true;
                    }
                    let listItemContent = orderedListMatch[2].trim();
                    if (listItemContent.length > 0) {
                        processedLines.push(`<li>${applyInlineFormatting(listItemContent)}</li>`);
                    } // Empty list items (e.g. "1. ") are skipped
                    continue;
                }
                if (inOrderedList && !/^\d+\.\s/.test(effectiveContent)) { // Exiting Ordered List
                    closeAllOpenBlocks();
                }

                // Default to paragraph if none of the above matched
                // Ensure any open blocks are closed before starting a paragraph
                if (!inUnorderedList && !inOrderedList && !inBlockquote) {
                     processedLines.push(`<p>${applyInlineFormatting(effectiveContent)}</p>`);
                } else {
                    // This case might occur if a line doesn't match any block but a block is open.
                    // It implies the block should have been closed by a blank line or a different block starter.
                    // For safety, close blocks and then process as a paragraph.
                    closeAllOpenBlocks();
                    processedLines.push(`<p>${applyInlineFormatting(effectiveContent)}</p>`);
                }
            }
        }

        closeAllOpenBlocks(); // Ensure any remaining open blocks are closed at the end

        let finalHtml = processedLines.join('\n'); // Join with \n for source readability

        // Post-processing pass to remove extra newlines specifically around list structures
        finalHtml = finalHtml.replace(/<\/li>(?:\s*\n\s*)+<li>/g, '</li><li>'); // Remove newlines between list items
        finalHtml = finalHtml.replace(/<ul>(?:\s*\n\s*)+<li>/g, '<ul><li>');   // Remove newlines after <ul> and before first <li>
        finalHtml = finalHtml.replace(/<\/li>(?:\s*\n\s*)+<\/ul>/g, '</li></ul>'); // Remove newlines after last <li> and before </ul>
        finalHtml = finalHtml.replace(/<ol>(?:\s*\n\s*)+<li>/g, '<ol><li>');   // Remove newlines after <ol> and before first <li>
        finalHtml = finalHtml.replace(/<\/li>(?:\s*\n\s*)+<\/ol>/g, '</li></ol>'); // Remove newlines after last <li> and before </ol>
        
        return finalHtml.trim();
    },

    /**
     * Attempts to geocode a location string to latitude and longitude.
     * @param {string} locationString The location string (e.g., "Portland, OR").
     * @returns {Promise<Object|null>} A promise that resolves to { lat, lon } or null.
     */
    geocodeLocationString: async (locationString) => {
        if (!locationString || typeof locationString !== 'string') {
            console.warn("geocodeLocationString: Invalid locationString provided:", locationString);
            return null;
        }

        // Check cache first
        const cachedGeocode = store.getGeocodeCache(locationString);
        if (cachedGeocode && (Date.now() - cachedGeocode.timestamp < APP_CONSTANTS.CACHE_EXPIRY_GEOCODE_MS)) {
            console.log(`Using cached geocode for "${locationString}"`);
            return cachedGeocode.data;
        }

        const nominatimUrl = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(locationString)}&format=json&limit=1`;
        
        try {
            console.log(`Geocoding "${locationString}" via Nominatim`);
            // Nominatim requires a custom User-Agent.
            // Replace 'nfo2go/2.0 (YourAppNameOrContactInfo)' with your actual app name/contact.
            const response = await fetch(nominatimUrl, {
                method: 'GET',
                headers: {
                    'User-Agent': 'nfo2go/2.0 (github.com/kurtisgr/nfo2go)' 
                }
            });

            if (!response.ok) {
                console.error(`Geocoding error for "${locationString}": ${response.status} ${response.statusText}`);
                return null;
            }

            const data = await response.json();

            if (data && data.length > 0) {
                const lat = parseFloat(data[0].lat);
                const lon = parseFloat(data[0].lon);
                if (!isNaN(lat) && !isNaN(lon)) {
                    store.saveGeocodeCache(locationString, { lat, lon });
                    console.log(`Geocoded "${locationString}" to:`, { lat, lon });
                    return { lat, lon };
                }
            }
            console.warn(`No valid geocoding results for "${locationString}"`);
            return null;
        } catch (error) {
            console.error(`Network or other error during geocoding for "${locationString}":`, error);
            return null;
        }
    },

    extractCoordinates: async (locationString) => {
        if (!locationString || typeof locationString !== 'string') {
            console.warn("extractCoordinates: Invalid locationString provided:", locationString);
            return null;
        }

        const latLonRegex = /^(-?\d{1,3}(?:\.\d+)?)\s*,\s*(-?\d{1,3}(?:\.\d+)?)$/;
        const match = locationString.match(latLonRegex);

        if (match) {
            const lat = parseFloat(match[1]);
            const lon = parseFloat(match[2]);
            if (lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180) {
                return { lat, lon };
            } else {
                console.warn("extractCoordinates: Parsed coordinates are out of valid range:", { lat, lon });
                // Fall through to geocoding if direct parse is invalid range
            }
        }
        // If not a direct lat,lon string or if parsed coordinates were invalid, try geocoding
        return await utils.geocodeLocationString(locationString);
    },

    /**
     * Debounces a function, delaying its execution until after a specified wait time
     * has elapsed since the last time it was invoked.
     * @param {Function} func The function to debounce.
     * @param {number} delay The number of milliseconds to delay.
     * @returns {Function} The new debounced function.
     */
    debounce: (func, delay) => {
        let timeoutId;
        return (...args) => {
            clearTimeout(timeoutId);
            timeoutId = setTimeout(() => {
                func.apply(this, args);
            }, delay);
        };
    },

    /**
     * Reliably tests for an active internet connection by making a network request.
     * Sets a global window.isActuallyOnline flag based on the result.
     * @returns {Promise<boolean>} A promise that resolves to true if online, false if offline.
     */
    testAndSetOnlineStatus: async () => {
        // Start with the browser's less-reliable opinion for a quick offline check.
        if (!navigator.onLine) {
            console.warn("Connectivity Test: navigator.onLine is false. Setting status to OFFLINE.");
            window.isActuallyOnline = false;
            return false;
        }

        // If the browser thinks it's online, verify with a real network request.
        // We use a HEAD request with 'no-cors' to a reliable, small resource.
        // This is fast as it doesn't download content, it just checks for a response.
        // The response will be opaque, but success/failure is all we need to determine connectivity.
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 3000); // 3-second timeout

        try {
            // Using a well-known, highly available resource that supports CORS.
            const response = await fetch('https://httpstat.us/204', {
                method: 'HEAD',
                signal: controller.signal,
                cache: 'no-store' // Important: Ensures a fresh network request
            });

            if (!response.ok) throw new Error(`HTTP status ${response.status}`);

            clearTimeout(timeoutId);
            console.log("Connectivity Test: Network request succeeded. Setting status to ONLINE.");
            window.isActuallyOnline = true;
            return true;
        } catch (error) {
            clearTimeout(timeoutId);
            console.warn("Connectivity Test: Network request failed (likely no internet). Setting status to OFFLINE. Error:", error.name);
            window.isActuallyOnline = false;
            return false;
        }
    }
};
