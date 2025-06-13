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

            // Check if the line consists ONLY of whitespace characters.
            // \S is any non-whitespace character. So, !/\S/.test(line) is true if the line is all whitespace.
            if (!/\S/.test(line)) { // Line is visually blank
                if (inList) {
                    processedLines.push('</ul>');
                    inList = false;
                }
                // Add a blank line only if the previous processed line wasn't also an intentionally added blank line.
                // The .trim().length > 0 check ensures that if the last element was content (even if it ended up as empty after formatting),
                // we can add a blank line. If the last element was '', it means we just added a blank line.
                if (processedLines.length === 0 || processedLines[processedLines.length - 1].trim().length > 0) {
                    processedLines.push(''); 
                }
            } else { // Line has non-whitespace content
                // Clean this content line for parsing markdown structure and for final output.
                // Replace specific problematic Unicode spaces with a regular space, then trim.
                let contentToProcess = line.replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();
                
                // After this initial cleaning, if contentToProcess becomes empty,
                // it means the original line had non-\S characters that were all problematic ones we removed.
                // Treat such a line as blank. This is a fallback.
                if (contentToProcess.length === 0) {
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    if (processedLines.length === 0 || processedLines[processedLines.length - 1].trim().length > 0) {
                        processedLines.push('');
                    }
                    continue; // Skip to next line
                }

                // Now contentToProcess definitely has renderable content.
                if (contentToProcess.startsWith('### ')) {
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    let headerContent = contentToProcess.substring(4).trim();
                    processedLines.push(`<h3>${applyInlineFormatting(headerContent)}</h3>`);
                } else if (contentToProcess.startsWith('* ')) {
                    if (!inList) {
                        processedLines.push('<ul>');
                        inList = true;
                    }
                    let listItemContent = contentToProcess.substring(2).trim();
                    processedLines.push(`  <li>${applyInlineFormatting(listItemContent)}</li>`);
                } else { // Regular content line
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    processedLines.push(applyInlineFormatting(contentToProcess));
                }
            }
        }

        if (inList) {
            processedLines.push('</ul>');
        }
        return processedLines.join('\n').trim();
    }
};
