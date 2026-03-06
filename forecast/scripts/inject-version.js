/**
 * Deploy-time script: injects VERSION from service-worker.js into index.html
 * so one version variable drives both the PWA cache and asset cache busting.
 *
 * Run from repo root: node forecast/scripts/inject-version.js
 * Or from forecast/: node scripts/inject-version.js
 */
const fs = require('fs');
const path = require('path');

const forecastDir = path.join(__dirname, '..');
const swPath = path.join(forecastDir, 'service-worker.js');
const indexPath = path.join(forecastDir, 'index.html');

const swContent = fs.readFileSync(swPath, 'utf8');
const match = swContent.match(/const\s+VERSION\s*=\s*['"]([^'"]+)['"]/);
const version = match ? match[1] : '0.0.0';

let html = fs.readFileSync(indexPath, 'utf8');
if (!html.includes('{{VERSION}}')) {
    console.log('inject-version: no {{VERSION}} placeholder in index.html, skipping');
    process.exit(0);
}
html = html.replace(/\{\{VERSION\}\}/g, version);
fs.writeFileSync(indexPath, html);
console.log('Injected VERSION:', version);
