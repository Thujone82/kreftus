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
            // Links: [text](url)
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
                let listWasClosedByThisBlankLine = false;
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
                        listWasClosedByThisBlankLine = true;
                    }
                }

                // Add an empty string to processedLines if this blank line followed a list closure or an h3,
                // and the previously processed line wasn't already an empty string.
                // This adds a newline to the HTML source for readability.
                if (processedLines.length > 0) {
                    const lastPushedItem = processedLines[processedLines.length - 1];
                    if (lastPushedItem !== '' && (listWasClosedByThisBlankLine || (typeof lastPushedItem === 'string' && lastPushedItem.endsWith('</h3>')))) {
                        processedLines.push('');
                    }
                }
                continue; // Always continue to the next line after processing a blank line.
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
                    let listItemContent = effectiveContent.substring(2).trim(); // Get content after "* "
                    if (listItemContent.length > 0) {
                        processedLines.push(`<li>${applyInlineFormatting(listItemContent)}</li>`);
                    } else {
                        // This line was effectively an empty list item (e.g., "* " or "*    ")
                        // We should not render an empty <li>.
                        // If we are in a list and the next line is NOT a list item,
                        // this "empty" item might signify the end of the list.
                        if (inList) {
                            let nextLineIsListItemAfterEmpty = false;
                            for (let j = i + 1; j < lines.length; j++) {
                                let nextPreview = lines[j].replace(/[\u00A0\u200B\uFEFF]+/g, ' ').trim();
                                if (nextPreview.length > 0) {
                                    if (nextPreview.startsWith('* ')) nextLineIsListItemAfterEmpty = true;
                                    break;
                                }
                            }
                            if (!nextLineIsListItemAfterEmpty) { // If no more list items follow this empty one
                                processedLines.push('</ul>');
                                inList = false;
                            }
                        }
                        continue; // Skip adding an empty <li>
                    }
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
