// Main application logic and state management

// Application state
const appState = {
    currentMode: 'full',
    weatherData: null,
    location: null,
    autoUpdateEnabled: true,
    lastFetchTime: null,
    hourlyScrollIndex: 0,
    loading: false,
    error: null
};

// Constants
const DATA_STALE_THRESHOLD = 600000; // 10 minutes in milliseconds
const AUTO_UPDATE_INTERVAL = 600000; // 10 minutes

// DOM elements - will be initialized when DOM is ready
let elements = {};

// Initialize DOM elements
function initializeElements() {
    console.log('Initializing DOM elements');
    elements = {
        locationInput: document.getElementById('locationInput'),
        searchBtn: document.getElementById('searchBtn'),
        hereBtn: document.getElementById('hereBtn'),
        locationDisplay: document.getElementById('locationDisplay'),
        modeButtons: document.querySelectorAll('.mode-btn'),
        refreshBtn: document.getElementById('refreshBtn'),
        autoUpdateToggle: document.getElementById('autoUpdateToggle'),
        lastUpdate: document.getElementById('lastUpdate'),
        loadingIndicator: document.getElementById('loadingIndicator'),
        errorMessage: document.getElementById('errorMessage'),
        weatherContent: document.getElementById('weatherContent'),
        updateNotification: document.getElementById('updateNotification'),
        reloadBtn: document.getElementById('reloadBtn'),
        shareBtn: null // Will be created dynamically
    };
    
    console.log('Elements initialized:', {
        locationInput: !!elements.locationInput,
        searchBtn: !!elements.searchBtn,
        hereBtn: !!elements.hereBtn
    });
    
    // Verify critical elements exist
    if (!elements.locationInput || !elements.searchBtn || !elements.hereBtn) {
        console.error('Critical DOM elements not found');
        return false;
    }
    
    return true;
}

// Initialize app
async function init() {
    console.log('init() called');
    
    // Initialize DOM elements
    if (!initializeElements()) {
        console.error('Failed to initialize DOM elements');
        return;
    }
    
    // Set up event listeners FIRST - this is critical!
    console.log('Setting up event listeners...');
    setupEventListeners();
    console.log('Event listeners set up');
    
    // Register service worker (non-blocking)
    if ('serviceWorker' in navigator) {
        try {
            const registration = await navigator.serviceWorker.register('/forecast/service-worker.js');
            console.log('Service Worker registered:', registration);
            
            // Check for updates
            registration.addEventListener('updatefound', () => {
                const newWorker = registration.installing;
                newWorker.addEventListener('statechange', () => {
                    if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                        showUpdateNotification();
                    }
                });
            });
            
            // Listen for messages from service worker
            navigator.serviceWorker.addEventListener('message', (event) => {
                if (event.data && event.data.type === 'UPDATE_AVAILABLE') {
                    showUpdateNotification();
                }
            });
            
            // Check for manifest update periodically
            setInterval(checkForUpdate, 300000); // Check every 5 minutes
        } catch (error) {
            console.error('Service Worker registration failed:', error);
            // Don't block app initialization if service worker fails
        }
    }
    
    // Check for stored auto-update preference
    const storedAutoUpdate = localStorage.getItem('forecastAutoUpdate');
    if (storedAutoUpdate !== null) {
        appState.autoUpdateEnabled = storedAutoUpdate === 'true';
        if (elements.autoUpdateToggle) {
            elements.autoUpdateToggle.checked = appState.autoUpdateEnabled;
        }
    }
    
    // Check for URL query parameter
    const urlParams = new URLSearchParams(window.location.search);
    const locationParam = urlParams.get('location');
    
    // Load initial weather data (don't block if it fails)
    try {
        if (locationParam) {
            // Use location from URL
            elements.locationInput.value = locationParam;
            await loadWeatherData(locationParam);
        } else {
            // Check for stored location
            const storedLocation = localStorage.getItem('forecastLocation');
            if (storedLocation) {
                elements.locationInput.value = storedLocation;
            } else {
                // Default to 'here' if no location specified
                elements.locationInput.value = 'here';
                await loadWeatherData('here');
            }
        }
    } catch (error) {
        console.error('Error loading initial weather data:', error);
        // Don't block app - user can still search manually
    }
}

// Set up event listeners
function setupEventListeners() {
    // Verify elements exist
    if (!elements.searchBtn || !elements.hereBtn || !elements.locationInput) {
        console.error('Required elements not found. Retrying...');
        console.error('searchBtn:', elements.searchBtn);
        console.error('hereBtn:', elements.hereBtn);
        console.error('locationInput:', elements.locationInput);
        setTimeout(setupEventListeners, 100);
        return;
    }
    
    console.log('Setting up event listeners');
    
    // Location input
    elements.searchBtn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        console.log('Search button clicked');
        handleSearch();
    });
    elements.hereBtn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        console.log('Here button clicked');
        handleHere();
    });
    elements.locationInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            console.log('Enter key pressed');
            handleSearch();
        }
    });
    
    // Mode buttons
    if (elements.modeButtons && elements.modeButtons.length > 0) {
        elements.modeButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                const mode = btn.dataset.mode;
                switchMode(mode);
            });
        });
    }
    
    // Control buttons
    if (elements.refreshBtn) {
        elements.refreshBtn.addEventListener('click', (e) => {
            e.preventDefault();
            handleRefresh();
        });
    }
    if (elements.autoUpdateToggle) {
        elements.autoUpdateToggle.addEventListener('change', (e) => {
            appState.autoUpdateEnabled = e.target.checked;
            localStorage.setItem('forecastAutoUpdate', appState.autoUpdateEnabled.toString());
            if (appState.autoUpdateEnabled) {
                appState.lastFetchTime = new Date();
            }
        });
    }
    
    // Update notification
    if (elements.reloadBtn) {
        elements.reloadBtn.addEventListener('click', (e) => {
            e.preventDefault();
            window.location.reload();
        });
    }
    
    // Create share button
    createShareButton();
}

// Handle search
async function handleSearch() {
    console.log('handleSearch called');
    const location = elements.locationInput.value.trim();
    console.log('Location:', location);
    if (!location) {
        showError('Please enter a location');
        return;
    }
    
    localStorage.setItem('forecastLocation', location);
    updateURL(location);
    await loadWeatherData(location);
}

// Handle "here" button
async function handleHere() {
    console.log('handleHere called');
    elements.locationInput.value = 'here';
    localStorage.setItem('forecastLocation', 'here');
    updateURL('here');
    await loadWeatherData('here');
}

// Load weather data
async function loadWeatherData(location) {
    try {
        setLoading(true);
        hideError();
        
        // Ensure fetchWeatherData is available (from api.js)
        if (typeof fetchWeatherData === 'undefined') {
            throw new Error('fetchWeatherData function not found. Please ensure api.js is loaded.');
        }
        
        const weatherData = await fetchWeatherData(location);
        const processedWeather = processWeatherData(weatherData);
        
        appState.weatherData = processedWeather;
        appState.location = weatherData.location;
        appState.lastFetchTime = new Date();
        appState.hourlyScrollIndex = 0;
        
        // Update location display
        elements.locationDisplay.textContent = `${weatherData.location.city}, ${weatherData.location.state}`;
        
        // Update URL if location changed
        if (location && location.toLowerCase() !== 'here') {
            updateURL(location);
        }
        
        // Update last update time
        updateLastUpdateTime();
        
        // Render current mode
        renderCurrentMode();
        
        setLoading(false);
    } catch (error) {
        setLoading(false);
        showError(`Error loading weather data: ${error.message}`);
        console.error('Error loading weather data:', error);
    }
}

// Handle refresh
async function handleRefresh() {
    const location = elements.locationInput.value.trim() || 'here';
    await loadWeatherData(location);
}

// Switch display mode
function switchMode(mode) {
    appState.currentMode = mode;
    
    // Update active button
    elements.modeButtons.forEach(btn => {
        if (btn.dataset.mode === mode) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });
    
    // Render current mode
    renderCurrentMode();
}

// Render current mode
function renderCurrentMode() {
    if (!appState.weatherData || !appState.location) {
        return;
    }
    
    let html = '';
    
    switch (appState.currentMode) {
        case 'full':
            html = displayFullWeatherReport(appState.weatherData, appState.location);
            break;
        case 'terse':
            html = displayTerseMode(appState.weatherData, appState.location);
            break;
        case 'hourly':
            html = displayHourlyForecast(appState.weatherData, appState.location, appState.hourlyScrollIndex, 12);
            break;
        case 'daily':
            html = displaySevenDayForecast(appState.weatherData, appState.location, true);
            break;
        case 'rain':
            html = displayRainForecast(appState.weatherData, appState.location);
            break;
        case 'wind':
            html = displayWindForecast(appState.weatherData, appState.location);
            break;
        default:
            html = displayFullWeatherReport(appState.weatherData, appState.location);
    }
    
    elements.weatherContent.innerHTML = html;
}

// Set loading state
function setLoading(loading) {
    appState.loading = loading;
    if (loading) {
        elements.loadingIndicator.classList.remove('hidden');
        elements.weatherContent.innerHTML = '';
    } else {
        elements.loadingIndicator.classList.add('hidden');
    }
}

// Show error
function showError(message) {
    appState.error = message;
    elements.errorMessage.textContent = message;
    elements.errorMessage.classList.remove('hidden');
}

// Hide error
function hideError() {
    appState.error = null;
    elements.errorMessage.classList.add('hidden');
}

// Update last update time
function updateLastUpdateTime() {
    if (appState.lastFetchTime) {
        const timeAgo = getTimeAgo(appState.lastFetchTime);
        elements.lastUpdate.textContent = `Last updated: ${timeAgo}`;
    }
}

// Get time ago string
function getTimeAgo(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (seconds < 60) {
        return 'just now';
    } else if (minutes < 60) {
        return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
    } else if (hours < 24) {
        return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
    } else {
        return date.toLocaleString();
    }
}

// Check if data is stale
function isDataStale() {
    if (!appState.lastFetchTime) return true;
    const now = new Date();
    const diff = now - appState.lastFetchTime;
    return diff > DATA_STALE_THRESHOLD;
}

// Auto-refresh check
function checkAutoRefresh() {
    if (appState.autoUpdateEnabled && isDataStale() && appState.location) {
        const location = elements.locationInput.value.trim() || 'here';
        loadWeatherData(location);
    }
}

// Check for app update
async function checkForUpdate() {
    try {
        const response = await fetch('/forecast/manifest.json?t=' + Date.now());
        if (!response.ok) return;
        
        const manifest = await response.json();
        const currentVersion = localStorage.getItem('forecastVersion') || '1.0.0';
        
        if (manifest.version !== currentVersion) {
            showUpdateNotification();
            localStorage.setItem('forecastVersion', manifest.version);
            
            // Notify service worker to check for update
            if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
                navigator.serviceWorker.controller.postMessage({ type: 'CHECK_UPDATE' });
            }
        }
    } catch (error) {
        console.error('Error checking for update:', error);
    }
}

// Show update notification
function showUpdateNotification() {
    elements.updateNotification.classList.remove('hidden');
}

// Hide update notification
function hideUpdateNotification() {
    elements.updateNotification.classList.add('hidden');
}

// Update URL with location parameter
function updateURL(location) {
    const url = new URL(window.location);
    if (location && location.toLowerCase() !== 'here') {
        url.searchParams.set('location', location);
    } else {
        url.searchParams.delete('location');
    }
    window.history.pushState({}, '', url);
    updateShareButton();
}

// Create share button
function createShareButton() {
    const controlBar = document.querySelector('.control-bar');
    if (!controlBar) return;
    
    const shareBtn = document.createElement('button');
    shareBtn.id = 'shareBtn';
    shareBtn.className = 'btn btn-secondary';
    shareBtn.innerHTML = '🔗 Share';
    shareBtn.title = 'Copy shareable link';
    shareBtn.addEventListener('click', handleShare);
    
    // Insert after refresh button
    elements.refreshBtn.parentNode.insertBefore(shareBtn, elements.refreshBtn.nextSibling);
    elements.shareBtn = shareBtn;
    
    updateShareButton();
}

// Update share button state
function updateShareButton() {
    if (!elements.shareBtn) return;
    
    const urlParams = new URLSearchParams(window.location.search);
    const locationParam = urlParams.get('location');
    const currentLocation = elements.locationInput.value.trim();
    
    // Enable/disable based on whether there's a location to share
    if (currentLocation && currentLocation.toLowerCase() !== 'here') {
        elements.shareBtn.disabled = false;
    } else {
        elements.shareBtn.disabled = false; // Always allow sharing current view
    }
}

// Handle share button click
async function handleShare() {
    const location = elements.locationInput.value.trim();
    const url = new URL(window.location);
    
    if (location && location.toLowerCase() !== 'here') {
        url.searchParams.set('location', location);
    } else {
        url.searchParams.delete('location');
    }
    
    const shareUrl = url.toString();
    
    try {
        // Try to use Web Share API if available
        if (navigator.share) {
            await navigator.share({
                title: 'Forecast Weather',
                text: `Check the weather for ${location || 'your location'}`,
                url: shareUrl
            });
        } else {
            // Fallback to clipboard
            await navigator.clipboard.writeText(shareUrl);
            // Show feedback
            const originalText = elements.shareBtn.innerHTML;
            elements.shareBtn.innerHTML = '✓ Copied!';
            setTimeout(() => {
                elements.shareBtn.innerHTML = originalText;
            }, 2000);
        }
    } catch (error) {
        // Fallback to clipboard if share fails
        try {
            await navigator.clipboard.writeText(shareUrl);
            const originalText = elements.shareBtn.innerHTML;
            elements.shareBtn.innerHTML = '✓ Copied!';
            setTimeout(() => {
                elements.shareBtn.innerHTML = originalText;
            }, 2000);
        } catch (clipboardError) {
            // Last resort: show URL in alert
            alert(`Share this link:\n${shareUrl}`);
        }
    }
}

// Initialize app when DOM is ready and scripts are loaded
function startApp() {
    // Ensure all required functions are available
    if (typeof fetchWeatherData === 'undefined') {
        console.error('fetchWeatherData not found - api.js may not be loaded');
        setTimeout(startApp, 100); // Retry after 100ms
        return;
    }
    if (typeof processWeatherData === 'undefined') {
        console.error('processWeatherData not found - weather.js may not be loaded');
        setTimeout(startApp, 100); // Retry after 100ms
        return;
    }
    
    // All scripts loaded, initialize app
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
}

// Start the app
startApp();

// Set up auto-refresh interval
setInterval(checkAutoRefresh, 60000); // Check every minute

// Update last update time periodically
setInterval(updateLastUpdateTime, 30000); // Update every 30 seconds

