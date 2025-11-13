// Main application logic and state management

// Application state
const appState = {
    currentMode: 'full',
    weatherData: null,
    location: null,
    observationsData: null,
    observationsAvailable: false,
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
        locationDisplay: document.getElementById('locationDisplay'),
        modeButtons: document.querySelectorAll('.mode-btn'),
        refreshBtn: document.getElementById('refreshBtn'),
        autoUpdateToggle: document.getElementById('autoUpdateToggle'),
        autoUpdateToggleLabel: document.querySelector('.toggle-label'),
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
        searchBtn: !!elements.searchBtn
    });
    
    // Verify critical elements exist
    if (!elements.locationInput || !elements.searchBtn) {
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
    
    // Set up auto-refresh interval
    if (autoRefreshInterval) clearInterval(autoRefreshInterval);
    autoRefreshInterval = setInterval(checkAutoRefresh, 60000); // Check every minute
    
    // Set up update time interval
    if (updateTimeInterval) clearInterval(updateTimeInterval);
    updateTimeInterval = setInterval(updateLastUpdateTime, 30000); // Update every 30 seconds
    
    // Initialize History button state (disabled by default until data is loaded)
    updateHistoryButtonState();
    
    // Check for URL query parameters
    const urlParams = new URLSearchParams(window.location.search);
    const locationParam = urlParams.get('location');
    const modeParam = urlParams.get('mode');
    
    // Set mode from URL if provided (but don't render yet - wait for data)
    if (modeParam && ['full', 'history', 'hourly', 'daily', 'rain', 'wind'].includes(modeParam)) {
        appState.currentMode = modeParam;
        // Update active button
        elements.modeButtons.forEach(btn => {
            if (btn.dataset.mode === modeParam) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });
    }
    
    // Try to load cached data first (unless URL specifies a different location)
    const cache = loadWeatherDataFromCache();
    let cachedDataLoaded = false;
    
    if (cache) {
        // Check if URL location param matches cached location
        if (locationParam && locationParam.toLowerCase() !== cache.location.toLowerCase()) {
            // URL specifies different location - don't use cache, fetch new location
            cachedDataLoaded = false;
        } else {
            // Load cached data
            cachedDataLoaded = loadCachedWeatherData();
        }
    }
    
    if (cachedDataLoaded) {
        // Cached data was loaded and displayed
        const cacheIsStale = isCacheStale(cache.timestamp);
        
        if (cacheIsStale) {
            // Cache is stale, trigger background refresh
            const locationToRefresh = locationParam || cache.location || elements.locationInput.value.trim() || 'here';
            // Trigger background refresh (silent failure, keeps cached data visible)
            loadWeatherData(locationToRefresh, false, true).catch(error => {
                console.error('Background refresh failed, keeping cached data:', error);
            });
        }
        // If cache is fresh, no refresh needed - cached data is already displayed
    } else {
        // No cached data or different location requested, proceed with normal initial load
        try {
            if (locationParam) {
                // Use location from URL
                elements.locationInput.value = locationParam;
                await loadWeatherData(locationParam, false); // false = show errors
            } else {
                // Check for stored location
                const storedLocation = localStorage.getItem('forecastLocation');
                if (storedLocation && storedLocation.trim() !== '') {
                    elements.locationInput.value = storedLocation;
                    await loadWeatherData(storedLocation, false); // false = show errors
                } else {
                    // Try to detect location automatically (silently on failure)
                    await loadWeatherData('here', true); // true = silent on location detection failure
                }
            }
        } catch (error) {
            console.error('Error loading initial weather data:', error);
            // Don't block app - user can still search manually
        }
    }
}

// Set up event listeners
function setupEventListeners() {
    // Verify elements exist
    if (!elements.searchBtn || !elements.locationInput) {
        console.error('Required elements not found. Retrying...');
        console.error('searchBtn:', elements.searchBtn);
        console.error('locationInput:', elements.locationInput);
        setTimeout(setupEventListeners, 100);
        return;
    }
    
    console.log('Setting up event listeners');
    
    // Location input
    elements.searchBtn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        console.log('Load button clicked');
        handleSearch();
    });
    elements.locationInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            console.log('Enter key pressed');
            handleSearch();
            // Focus the search button after handling search
            if (elements.searchBtn) {
                elements.searchBtn.focus();
            }
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
            elements.reloadBtn.addEventListener('click', async (e) => {
                e.preventDefault();
                // Clear all caches and reload
                if ('caches' in window) {
                    const cacheNames = await caches.keys();
                    await Promise.all(
                        cacheNames.map(cacheName => caches.delete(cacheName))
                    );
                }
                // Unregister service worker to force update
                if ('serviceWorker' in navigator) {
                    const registration = await navigator.serviceWorker.getRegistration();
                    if (registration) {
                        await registration.unregister();
                    }
                }
                // Reload the page
                window.location.reload(true);
            });
        }
    
    // Create share button
    createShareButton();
    
    // Set up hourly navigation handlers once (using event delegation)
    setupHourlyNavigation();
}

// Handle search
async function handleSearch() {
    console.log('handleSearch called');
    let location = elements.locationInput.value.trim();
    console.log('Location:', location);
    
    // Handle "here" text shortcut
    if (location.toLowerCase() === 'here') {
        location = 'here';
    }
    
    if (!location) {
        showError('Please enter a location');
        return;
    }
    
    // Check if this is a different location than what's cached
    const cache = loadWeatherDataFromCache();
    if (cache && cache.location.toLowerCase() !== location.toLowerCase()) {
        // Different location - cache will be updated by loadWeatherData after successful fetch
        // No need to clear cache explicitly, it will be overwritten
    }
    
    localStorage.setItem('forecastLocation', location);
    updateURL(location, appState.currentMode);
    await loadWeatherData(location, false, false); // Don't use background mode for manual search - show loading
}

// Check observations availability and fetch observations
async function checkObservationsAvailability(pointsData, timeZone) {
    try {
        if (!pointsData || !pointsData.properties || !pointsData.properties.observationStations) {
            appState.observationsAvailable = false;
            appState.observationsData = null;
            return false;
        }
        
        // Try to fetch observation stations
        const stationId = await fetchNWSObservationStations(pointsData);
        if (!stationId) {
            appState.observationsAvailable = false;
            appState.observationsData = null;
            return false;
        }
        
        // Try to fetch observations
        const observationsData = await fetchNWSObservations(stationId, timeZone);
        if (!observationsData || !observationsData.features || observationsData.features.length === 0) {
            appState.observationsAvailable = false;
            appState.observationsData = null;
            return false;
        }
        
        // Process observations data
        const processedObservations = processObservationsData(observationsData, timeZone);
        if (!processedObservations || processedObservations.length === 0) {
            appState.observationsAvailable = false;
            appState.observationsData = null;
            return false;
        }
        
        appState.observationsData = processedObservations;
        appState.observationsAvailable = true;
        return true;
    } catch (error) {
        console.error('Error checking observations availability:', error);
        appState.observationsAvailable = false;
        appState.observationsData = null;
        return false;
    }
}

// Update History button state
function updateHistoryButtonState() {
    const historyBtn = document.getElementById('historyModeBtn');
    if (!historyBtn) {
        return;
    }
    
    if (appState.observationsAvailable) {
        historyBtn.disabled = false;
        historyBtn.classList.remove('disabled');
    } else {
        historyBtn.disabled = true;
        historyBtn.classList.add('disabled');
    }
}

// Cache storage functions
function saveWeatherDataToCache(weatherData, location) {
    try {
        const cacheData = {
            weatherData: weatherData,
            observationsData: appState.observationsData,
            observationsAvailable: appState.observationsAvailable
        };
        localStorage.setItem('forecastCachedData', JSON.stringify(cacheData));
        localStorage.setItem('forecastCachedLocation', location);
        localStorage.setItem('forecastCachedTimestamp', new Date().toISOString());
    } catch (error) {
        console.warn('Failed to save weather data to cache:', error);
    }
}

function loadWeatherDataFromCache() {
    try {
        const cachedData = localStorage.getItem('forecastCachedData');
        const cachedLocation = localStorage.getItem('forecastCachedLocation');
        const cachedTimestamp = localStorage.getItem('forecastCachedTimestamp');
        
        if (cachedData && cachedLocation && cachedTimestamp) {
            return {
                data: JSON.parse(cachedData),
                location: cachedLocation,
                timestamp: new Date(cachedTimestamp)
            };
        }
    } catch (error) {
        console.warn('Failed to load weather data from cache:', error);
    }
    return null;
}

function isCacheStale(cacheTimestamp) {
    if (!cacheTimestamp) return true;
    const now = new Date();
    const diff = now - cacheTimestamp;
    return diff > DATA_STALE_THRESHOLD;
}

// Restore Date objects from cached data (JSON serialization converts dates to strings)
function restoreDatesFromCache(weatherData) {
    if (!weatherData) return weatherData;
    
    // Restore current.time (Date object)
    if (weatherData.current && weatherData.current.time) {
        weatherData.current.time = new Date(weatherData.current.time);
    }
    
    // Restore location.sunrise and location.sunset (Date objects)
    if (weatherData.location) {
        if (weatherData.location.sunrise) {
            weatherData.location.sunrise = new Date(weatherData.location.sunrise);
        }
        if (weatherData.location.sunset) {
            weatherData.location.sunset = new Date(weatherData.location.sunset);
        }
    }
    
    return weatherData;
}

// Load cached weather data and display it
function loadCachedWeatherData() {
    try {
        const cache = loadWeatherDataFromCache();
        if (!cache || !cache.data) {
            return false;
        }
        
        // Validate cached data structure
        if (!cache.data.weatherData) {
            console.warn('Cached data missing weatherData, ignoring cache');
            return false;
        }
        
        // Restore Date objects from cached data (JSON serialization converts dates to strings)
        const restoredWeatherData = restoreDatesFromCache(cache.data.weatherData);
        
        // Restore app state from cache
        appState.weatherData = restoredWeatherData;
        appState.observationsData = cache.data.observationsData || null;
        appState.observationsAvailable = cache.data.observationsAvailable || false;
        appState.lastFetchTime = cache.timestamp;
        
        // Restore location if available in cached data
        if (restoredWeatherData && restoredWeatherData.location) {
            appState.location = restoredWeatherData.location;
            const locationText = `${restoredWeatherData.location.city}, ${restoredWeatherData.location.state}`;
            elements.locationInput.value = locationText;
        } else {
            // Fallback to cached location string
            elements.locationInput.value = cache.location;
            // Try to create a minimal location object from the cached location string
            // This ensures renderCurrentMode() can still work
            if (cache.location && cache.location !== 'here') {
                // Parse location string (e.g., "Portland, OR" or "Portland, Oregon")
                const parts = cache.location.split(',').map(s => s.trim());
                if (parts.length >= 2) {
                    appState.location = {
                        city: parts[0],
                        state: parts[1],
                        // Use default values for other required fields
                        lat: 0,
                        lon: 0,
                        timeZone: 'America/New_York'
                    };
                }
            }
        }
        
        // Update History button state
        updateHistoryButtonState();
        
        // Update last update time
        updateLastUpdateTime();
        
        // Render current mode to display cached data
        renderCurrentMode();
        
        return true;
    } catch (error) {
        console.error('Error loading cached weather data:', error);
        // Clear potentially corrupted cache
        try {
            localStorage.removeItem('forecastCachedData');
            localStorage.removeItem('forecastCachedLocation');
            localStorage.removeItem('forecastCachedTimestamp');
        } catch (e) {
            console.warn('Failed to clear corrupted cache:', e);
        }
        return false;
    }
}

// Load weather data
async function loadWeatherData(location, silentOnLocationFailure = false, background = false) {
    try {
        setLoading(true, background);
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
        
        // Check observations availability and fetch observations if available
        await checkObservationsAvailability(weatherData.points, weatherData.location.timeZone);
        
        // Update History button state
        updateHistoryButtonState();
        
        // Update location in input field
        const locationText = `${weatherData.location.city}, ${weatherData.location.state}`;
        elements.locationInput.value = locationText;
        
        // Update URL if location changed (preserve mode)
        if (location && location.toLowerCase() !== 'here') {
            updateURL(location, appState.currentMode);
        }
        
        // Update last update time
        updateLastUpdateTime();
        
        // Save to cache after successful fetch
        const locationForCache = location || (weatherData.location ? `${weatherData.location.city}, ${weatherData.location.state}` : 'here');
        saveWeatherDataToCache(processedWeather, locationForCache);
        
        // Render current mode
        renderCurrentMode();
        
        setLoading(false, background);
    } catch (error) {
        setLoading(false, background);
        let errorMessage = error.message;
        
        // Handle location detection failures silently if requested (when trying to auto-detect 'here')
        if (silentOnLocationFailure && location.toLowerCase() === 'here') {
            // Log error but don't show to user
            console.log('Location detection failed silently (geolocation and IP-based both failed):', error);
            // Clear input and focus it
            elements.locationInput.value = '';
            elements.locationInput.focus();
            // Update History button state
            updateHistoryButtonState();
            return;
        }
        
        // If background update, don't show errors to user (log only)
        if (background) {
            console.error('Error loading weather data (background update):', error);
            return;
        }
        
        // Provide more helpful error messages for location detection failures
        if (errorMessage.includes('Unable to detect location') || errorMessage.includes('IP geolocation')) {
            errorMessage = 'Unable to detect your location automatically. Please enter a zip code or city name (e.g., "97217" or "Portland, OR") in the search box above.';
        } else if (errorMessage.includes('Network error') || errorMessage.includes('fetch')) {
            errorMessage = 'Network error: Unable to connect to weather services. Please check your internet connection and try again.';
        } else if (errorMessage.includes('Geocoding')) {
            errorMessage = `Location not found: "${location}". Please try a different location (e.g., zip code or "City, State").`;
        }
        
        showError(`Error loading weather data: ${errorMessage}`);
        console.error('Error loading weather data:', error);
        
        // Update History button state even on error
        updateHistoryButtonState();
    }
}

// Handle refresh
async function handleRefresh() {
    const location = elements.locationInput.value.trim() || 'here';
    await loadWeatherData(location, false, true); // Use background mode to keep content visible
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
    
    // Update URL with mode
    const location = elements.locationInput.value.trim() || 'here';
    updateURL(location, mode);
    
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
        case 'history':
            if (appState.observationsData && appState.observationsData.length > 0) {
                html = displayObservations(appState.observationsData, appState.location);
            } else {
                html = '<div class="error-message">No historical observations available.</div>';
            }
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
    
    // Set up hourly navigation handlers after rendering (only if in hourly mode)
    if (appState.currentMode === 'hourly') {
        setupHourlyNavigation();
    }
}

// Set up hourly navigation handlers (using event delegation)
let hourlyNavHandler = null;
let hourlyNavHandlerAttached = false;

function setupHourlyNavigation() {
    // Only attach handler once
    if (hourlyNavHandlerAttached) {
        return;
    }
    
    // Create handler for hourly navigation buttons
    hourlyNavHandler = (e) => {
        // Check if clicked element or its parent is a navigation button
        const button = e.target.closest('.hourly-nav-btn');
        if (!button) return;
        
        e.preventDefault();
        e.stopPropagation();
        
        console.log('Hourly navigation button clicked:', button.dataset.action);
        
        const action = button.dataset.action;
        if (!appState.weatherData || !appState.weatherData.hourly) {
            console.error('No weather data available');
            return;
        }
        
        const { hourly } = appState.weatherData;
        const periods = hourly.periods;
        const totalHours = Math.min(periods.length, 48);
        const maxHours = 12;
        
        console.log('Current scroll index:', appState.hourlyScrollIndex, 'Total hours:', totalHours);
        
        if (action === 'scroll-up') {
            // Scroll up by 12 hours
            const newIndex = Math.max(0, appState.hourlyScrollIndex - maxHours);
            console.log('Scrolling up to index:', newIndex);
            appState.hourlyScrollIndex = newIndex;
            renderCurrentMode();
        } else if (action === 'scroll-down') {
            // Scroll down by 12 hours
            const newIndex = Math.min(
                totalHours - maxHours,
                appState.hourlyScrollIndex + maxHours
            );
            console.log('Scrolling down to index:', newIndex);
            appState.hourlyScrollIndex = newIndex;
            renderCurrentMode();
        }
    };
    
    // Add event listener using event delegation
    if (elements.weatherContent) {
        elements.weatherContent.addEventListener('click', hourlyNavHandler);
        hourlyNavHandlerAttached = true;
        console.log('Hourly navigation handler attached');
    }
}

// Show control bar update indicator (spinner + "Updating" text)
function showControlBarUpdateIndicator() {
    if (!elements.autoUpdateToggleLabel) return;
    
    // Hide checkbox and label text
    if (elements.autoUpdateToggle) {
        elements.autoUpdateToggle.style.display = 'none';
    }
    const labelSpan = elements.autoUpdateToggleLabel.querySelector('span');
    if (labelSpan) {
        labelSpan.style.display = 'none';
    }
    
    // Create or show update indicator
    let updateIndicator = elements.autoUpdateToggleLabel.querySelector('.update-indicator');
    if (!updateIndicator) {
        updateIndicator = document.createElement('div');
        updateIndicator.className = 'update-indicator';
        const spinner = document.createElement('div');
        spinner.className = 'spinner';
        const text = document.createElement('span');
        text.textContent = 'Updating';
        updateIndicator.appendChild(spinner);
        updateIndicator.appendChild(text);
        elements.autoUpdateToggleLabel.appendChild(updateIndicator);
    }
    updateIndicator.style.display = 'flex';
}

// Hide control bar update indicator (restore checkbox and label text)
function hideControlBarUpdateIndicator() {
    if (!elements.autoUpdateToggleLabel) return;
    
    // Show checkbox and label text
    if (elements.autoUpdateToggle) {
        elements.autoUpdateToggle.style.display = '';
    }
    const labelSpan = elements.autoUpdateToggleLabel.querySelector('span');
    if (labelSpan) {
        labelSpan.style.display = '';
    }
    
    // Hide update indicator
    const updateIndicator = elements.autoUpdateToggleLabel.querySelector('.update-indicator');
    if (updateIndicator) {
        updateIndicator.style.display = 'none';
    }
}

// Set loading state
function setLoading(loading, background = false) {
    appState.loading = loading;
    if (loading) {
        // Check if weather data already exists
        const hasExistingData = appState.weatherData !== null;
        
        if (hasExistingData) {
            // Data exists: show only compact spinner in control bar, keep content visible
            showControlBarUpdateIndicator();
            // Don't clear weatherContent.innerHTML
            // Don't show the full loading indicator
        } else {
            // No data exists (initial load): show full loading indicator
            elements.loadingIndicator.classList.remove('hidden');
            elements.weatherContent.innerHTML = '';
            // Don't show control bar indicator during initial load
        }
    } else {
        // Hide control bar update indicator
        hideControlBarUpdateIndicator();
        
        // Hide full loading indicator if it was shown
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
        loadWeatherData(location, false, true); // Use background mode to keep content visible
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

// Update URL with location and mode parameters
function updateURL(location, mode = null) {
    const url = new URL(window.location);
    if (location && location.toLowerCase() !== 'here') {
        url.searchParams.set('location', location);
    } else {
        url.searchParams.delete('location');
    }
    
    // Update mode parameter
    const currentMode = mode || appState.currentMode;
    if (currentMode && currentMode !== 'full') {
        url.searchParams.set('mode', currentMode);
    } else {
        url.searchParams.delete('mode');
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
    
    // Include mode in share URL
    if (appState.currentMode && appState.currentMode !== 'full') {
        url.searchParams.set('mode', appState.currentMode);
    } else {
        url.searchParams.delete('mode');
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
    console.log('startApp() called');
    console.log('fetchWeatherData:', typeof fetchWeatherData);
    console.log('processWeatherData:', typeof processWeatherData);
    console.log('document.readyState:', document.readyState);
    
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
    
    console.log('All required functions found, initializing app...');
    
    // All scripts loaded, initialize app
    if (document.readyState === 'loading') {
        console.log('DOM still loading, waiting for DOMContentLoaded');
        document.addEventListener('DOMContentLoaded', () => {
            console.log('DOMContentLoaded fired');
            init();
        });
    } else {
        console.log('DOM already ready, calling init()');
        init();
    }
}

// Wait for DOM to be ready before starting
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        console.log('DOMContentLoaded - starting app');
        startApp();
    });
} else {
    console.log('DOM already ready - starting app');
    startApp();
}

// Set up auto-refresh interval (only after app is initialized)
let autoRefreshInterval = null;
let updateTimeInterval = null;

// These will be set up in init() after app is ready

