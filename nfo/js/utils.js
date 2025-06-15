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
                    let nextLineIsListItem = false;
                    for (let j = i + 1; j < lines.length; j++) {
                        let nextPreview = lines[j].replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();
                        if (nextPreview.length > 0) {
                            if (nextPreview.startsWith('* ')) {
                                nextLineIsListItem = true;
                            }
                            break; 
                        }
                    }

                    if (nextLineIsListItem) {
                        continue; // Ignore this blank line, it's between <li> of the same list.
                    } else {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                }
                // With <p> tags handling paragraph structure, explicit blank lines ('')
                // are generally not needed in processedLines for separation.
                // Margins on <p>, <h3>, <ul> will create visual separation.
                continue; 
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
                    processedLines.push(`<li>${applyInlineFormatting(listItemContent)}</li>`);
                } else { 
                    if (inList) {
                        processedLines.push('</ul>');
                        inList = false;
                    }
                    processedLines.push(`<p>${applyInlineFormatting(effectiveContent)}</p>`);
                }
            }
        }

        if (inList) {
            processedLines.push('</ul>');
        }
        let finalHtml = processedLines.join('\n'); // Join with \n for source readability

        // Post-processing pass to remove extra newlines specifically around list structures
        finalHtml = finalHtml.replace(/<\/li>(?:\s*\n\s*)+<li>/g, '</li><li>'); // Remove newlines between list items
        finalHtml = finalHtml.replace(/<ul>(?:\s*\n\s*)+<li>/g, '<ul><li>');   // Remove newlines after <ul> and before first <li>
        finalHtml = finalHtml.replace(/<\/li>(?:\s*\n\s*)+<\/ul>/g, '</li></ul>'); // Remove newlines after last <li> and before </ul>
        
        return finalHtml.trim();
    }
};
