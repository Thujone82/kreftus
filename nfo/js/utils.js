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

        // 1. Replace markdown-style links: [text](url) with <a href="url" target="_blank">text</a>
        //    Using a non-greedy match for text and URL. Added target="_blank" to open in new tab.
        let html = text.replace(/\[([^\]]+?)\]\(([^)]+?)\)/g, '<a href="$2" target="_blank">$1</a>');

        // 2. Replace **bold** with <b>bold</b>
        //    Using a non-greedy match for the content within **
        html = html.replace(/\*\*(.+?)\*\*/g, '<b>$1</b>');

        // 3. Process bullet points: * item
        const lines = html.split('\n');
        let resultHtml = '';
        let inList = false;

        for (let i = 0; i < lines.length; i++) {
            let line = lines[i];
            // Check for lines starting with "* " (asterisk and a space)
            if (line.trim().startsWith('* ')) {
                if (!inList) {
                    resultHtml += '<ul>\n';
                    inList = true;
                }
                // Remove the "* " and wrap with <li>, then process the rest of the line
                resultHtml += '  <li>' + line.trim().substring(2).trim() + '</li>\n';
            } else {
                if (inList) {
                    resultHtml += '</ul>\n';
                    inList = false;
                }
                resultHtml += line + '\n';
            }
        }
        // If the text ends with a list, close it
        if (inList) {
            resultHtml += '</ul>\n';
        }
        return resultHtml.trim(); // Trim trailing newline if any
    }
};