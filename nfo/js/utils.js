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
            // Robustly trim the line: replace common problematic Unicode spaces with a regular space, then trim.
            // \u00A0 is Non-Breaking Space, \u200B is Zero-Width Space, \uFEFF is BOM.
            let cleanTrimmedLine = line.replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();

            if (cleanTrimmedLine.startsWith('### ')) {
                if (inList) {
                    processedLines.push('</ul>');
                    inList = false;
                }
                // Extract content AFTER '### ' from the cleanTrimmedLine
                let headerContent = cleanTrimmedLine.substring(4).trim(); // .trim() again for safety after substring
                headerContent = applyInlineFormatting(headerContent);
                processedLines.push(`<h3>${headerContent}</h3>`);
            } else if (cleanTrimmedLine.startsWith('* ')) {
                if (!inList) {
                    processedLines.push('<ul>');
                    inList = true;
                }
                // Extract content AFTER '* ' from the cleanTrimmedLine
                let listItemContent = cleanTrimmedLine.substring(2).trim(); // .trim() again for safety
                listItemContent = applyInlineFormatting(listItemContent);
                processedLines.push(`  <li>${listItemContent}</li>`);
            } else { // Regular line or truly blank line
                if (inList) {
                    processedLines.push('</ul>');
                    inList = false;
                }

                if (cleanTrimmedLine.length > 0) { // If content exists after robust cleaning
                    // Use the original line for formatting if cleanTrimmedLine was derived from it and had content.
                    // This preserves original spacing within the line if robust cleaning only removed leading/trailing problematic chars.
                    // However, if cleanTrimmedLine IS the content (e.g. only standard spaces were trimmed), use it.
                    // For simplicity and to ensure problematic chars don't make it to output, we'll use cleanTrimmedLine.
                    // If original internal spacing is critical and different from cleanTrimmedLine, this might need adjustment.
                    // For now, prioritizing clean output.
                    processedLines.push(applyInlineFormatting(cleanTrimmedLine));
                } else { // Line is genuinely blank
                    // Add a blank line only if the previous line wasn't also blank
                    if (processedLines.length === 0 || processedLines[processedLines.length - 1].trim().length > 0) {
                        processedLines.push(''); // Add a single empty string for a blank line
                    }
                }
            }
        }

        if (inList) {
            processedLines.push('</ul>');
        }
        return processedLines.join('\n').trim();
    }
};
