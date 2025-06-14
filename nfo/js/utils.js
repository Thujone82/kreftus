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
        let inList = false;

        const applyInlineFormatting = (lineContent) => {
            // Apply link formatting first: text to <a href="url" target="_blank">text</a>
            lineContent = lineContent.replace(/\[([^\]]+?)\]\(([^)]+?)\)/g, '<a href="$2" target="_blank">$1</a>');
            // Then apply bold formatting: **bold** to <b>bold</b>
            lineContent = lineContent.replace(/\*\*(.+?)\*\*/g, '<b>$1</b>');
            return lineContent;
        };

        for (let i = 0; i < lines.length; i++) {
            let line = lines[i];
            // Process the line once to get its effective content after trimming special whitespace
            let effectiveContent = line.replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();

            if (effectiveContent.length === 0) { // Line is effectively blank
                if (inList) {
                    // If inside a list, check if the next non-blank line is also a list item.
                    // If so, this blank line is between items of the same list and should be ignored.
                    let nextLineIsListItem = false;
                    for (let j = i + 1; j < lines.length; j++) {
                        let nextPreview = lines[j].replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();
                        if (nextPreview.length > 0) {
                            if (nextPreview.startsWith('* ')) {
                                nextLineIsListItem = true;
                            }
                            break; // Found the next non-blank line
                        }
                    }

                    if (nextLineIsListItem) {
                        continue; // Ignore this blank line, it's between <li> of the same list.
                    } else {
                        // Blank line is at the end of a list, or the list was empty. Close it.
                        processedLines.push('</ul>');
                        inList = false;
                        // Now, this blank line will be processed by the general logic below
                        // as if it were outside any list context.
                    }
                }

                // General blank line processing (applies if not inList initially, or if a list was just closed by the block above)
                let addBlankLine = true;
                if (processedLines.length > 0) {
                    const lastProcessedLineTrimmed = processedLines[processedLines.length - 1].trim();
                    if (lastProcessedLineTrimmed.length === 0 || // Previous was already a processed blank line
                        lastProcessedLineTrimmed.endsWith('</h3>') ||
                        lastProcessedLineTrimmed.endsWith('</ul>') ||
                        lastProcessedLineTrimmed.endsWith('</li>')) {
                        addBlankLine = false;
                    }
                }

                // Lookahead: if still considering adding a blank line, and the next non-blank input
                // starts a list or header, suppress this blank line.
                if (addBlankLine && (i + 1 < lines.length)) {
                    let nextMeaningfulContent = "";
                    for (let j = i + 1; j < lines.length; j++) {
                        nextMeaningfulContent = lines[j].replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();
                        if (nextMeaningfulContent.length > 0) break;
                    }
                    if (nextMeaningfulContent.startsWith('* ') || nextMeaningfulContent.startsWith('### ')) {
                        addBlankLine = false;
                    }
                }

                if (addBlankLine) {
                    processedLines.push(''); // Add the blank line representation
                }
            } else { // Line has actual content
                if (effectiveContent.startsWith('### ')) {
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    let headerContent = effectiveContent.substring(4).trim();
                    processedLines.push(`<h3>${applyInlineFormatting(headerContent)}</h3>`);
                } else if (effectiveContent.startsWith('* ')) {
                    if (!inList) {
                        processedLines.push('<ul>');
                        inList = true;
                    }
                    let listItemContent = effectiveContent.substring(2).trim();
                    processedLines.push(`  <li>${applyInlineFormatting(listItemContent)}</li>`);
                } else { 
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    processedLines.push(applyInlineFormatting(effectiveContent));
                }
            }
        }

        if (inList) {
            processedLines.push('</ul>');
        }
        let finalHtml = processedLines.join('\n');

        // Post-processing pass to remove extra newlines specifically around list structures
        finalHtml = finalHtml.replace(/<\/li>\n\n+<li>/g, '</li>\n<li>'); // Between list items
        finalHtml = finalHtml.replace(/<ul>\n\n+<li>/g, '<ul>\n<li>');   // After <ul>, before first <li>
        finalHtml = finalHtml.replace(/<\/li>\n\n+<\/ul>/g, '</li>\n</ul>'); // After last <li>, before </ul>
        
        return finalHtml.trim();
    }
};
