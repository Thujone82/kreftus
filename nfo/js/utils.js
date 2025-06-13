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
            let trimmedLine = line.trim();

            if (trimmedLine.startsWith('### ')) {
                if (inList) {
                    processedLines.push('</ul>');
                    inList = false;
                }
                let headerContent = trimmedLine.substring(4).trim(); // Get content after "### "
                headerContent = applyInlineFormatting(headerContent);
                processedLines.push(`<h3>${headerContent}</h3>`);
            } else if (trimmedLine.startsWith('* ')) {
                if (!inList) {
                    processedLines.push('<ul>');
                    inList = true;
                }
                let listItemContent = trimmedLine.substring(2).trim(); // Get content after "* "
                listItemContent = applyInlineFormatting(listItemContent);
                processedLines.push(`  <li>${listItemContent}</li>`);
            } else {
                if (inList) {
                    processedLines.push('</ul>');
                    inList = false;
                }

                if (line.trim().length > 0) {
                    processedLines.push(applyInlineFormatting(line));
                } else {
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
