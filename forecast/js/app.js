// Main application logic and state management

// Application state
const appState = {
    currentMode: 'full',
    weatherData: null,
    location: null,
    currentLocationKey: null, // Store the current location key to ensure accurate favorite matching
    isCurrentLocationActive: false, // Track if current location (here) is selected
    observationsData: null,
    observationsAvailable: false,
    autoUpdateEnabled: true,
    lastFetchTime: null,
    hourlyScrollIndex: 0,
    loading: false,
    error: null,
    // In-memory cache for parsed weather data (keyed by locationKey + timestamp)
    // This avoids repeated JSON.parse operations when switching between favorites
    cachedWeatherDataByKey: new Map()
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
        shareBtn: null, // Will be created dynamically
        favoriteBtn: document.getElementById('favoriteBtn'),
        currentLocationBtn: document.getElementById('currentLocationBtn'),
        locationsBtn: document.getElementById('locationsBtn'),
        locationsDrawer: document.getElementById('locationsDrawer'),
        locationButtons: document.getElementById('locationButtons')
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

// Reset all favorites and cached data (triggered by ?reset=true URL parameter)
function performFullReset() {
    try {
        console.log('Performing full reset: clearing all favorites and cached data...');
        
        // Clear all favorites
        localStorage.removeItem('forecastFavorites');
        console.log('Cleared all favorites');
        
        // Clear all weather data cache (location-specific and default)
        const keys = Object.keys(localStorage);
        keys.forEach(key => {
            if (key.startsWith('forecastCachedData_') || 
                key.startsWith('forecastCachedLocation_') || 
                key.startsWith('forecastCachedTimestamp_') ||
                key.startsWith('forecastTidePredictions_') ||
                key.startsWith('forecastTidePredictionsTimestamp_') ||
                key.startsWith('forecastWaterLevelSupport_') ||
                key.startsWith('forecastWaterLevelSupportTimestamp_')) {
                localStorage.removeItem(key);
            }
        });
        
        // Clear default weather cache
        localStorage.removeItem('forecastCachedData');
        localStorage.removeItem('forecastCachedLocation');
        localStorage.removeItem('forecastCachedTimestamp');
        
        // Clear NOAA stations cache
        localStorage.removeItem('forecastNoaaStations');
        localStorage.removeItem('forecastNoaaStationsTimestamp');
        
        // Clear last viewed location and stored location
        localStorage.removeItem('forecastLastViewedLocation');
        localStorage.removeItem('forecastLocation');
        
        // Clear current mode (will reset to default)
        localStorage.removeItem('forecastCurrentMode');
        
        // Clear in-memory cache
        if (appState && appState.cachedWeatherDataByKey) {
            appState.cachedWeatherDataByKey.clear();
        }
        
        // Clear app state
        if (appState) {
            appState.weatherData = null;
            appState.location = null;
            appState.currentLocationKey = null;
            appState.observationsData = null;
            appState.lastFetchTime = null;
        }
        
        console.log('Full reset completed: all favorites and cached data cleared');
        
        // Show alert to user
        alert('Reset complete! All favorites and cached data have been cleared. The page will now reload.');
        
        return true;
    } catch (error) {
        console.error('Error during full reset:', error);
        alert('Error during reset: ' + error.message);
        return false;
    }
}

// Initialize app
async function init() {
    console.log('init() called');
    
    // Check for reset parameter in URL (check early, before DOM initialization)
    const resetUrlParams = new URLSearchParams(window.location.search);
    const resetParam = resetUrlParams.get('reset');
    if (resetParam === 'true') {
        console.log('Reset parameter detected, performing full reset...');
        performFullReset();
        
        // Remove reset parameter from URL and reload
        resetUrlParams.delete('reset');
        const newUrl = window.location.pathname + (resetUrlParams.toString() ? '?' + resetUrlParams.toString() : '');
        window.history.replaceState({}, '', newUrl);
        
        // Reload the page to start fresh
        window.location.reload();
        return; // Exit early, page will reload
    }
    
    // Initialize DOM elements
    if (!initializeElements()) {
        console.error('Failed to initialize DOM elements');
        return;
    }
    
    // Migrate old favorites to new format (do this early, before favorites are used)
    // Run after elements are initialized so error messages can be displayed
    const migrationSuccess = migrateFavorites();
    if (!migrationSuccess) {
        console.warn('Favorites migration failed - favorites have been cleared');
    }
    
    // Set up event listeners FIRST - this is critical!
    console.log('Setting up event listeners...');
    setupEventListeners();
    console.log('Event listeners set up');
    
    // Set up forecast text visibility observer
    setupForecastTextVisibility();
    
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
                } else if (event.data && event.data.type === 'CLEAR_CACHE') {
                    // Service worker updated - clear localStorage cache to get fresh data with new features
                    console.log('Service worker updated, clearing cache:', event.data.reason);
                    clearAllCachedData();
                    // If we have a current location, refresh the data to get new features (like NOAA station)
                    if (appState.location || elements.locationInput.value) {
                        const locationToRefresh = elements.locationInput.value || 'here';
                        setTimeout(() => {
                            loadWeatherData(locationToRefresh, false, false).catch(error => {
                                console.error('Error refreshing data after cache clear:', error);
                            });
                        }, 500);
                    }
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
    
    // Set mode from URL if provided, otherwise restore from localStorage
    if (modeParam && ['full', 'history', 'hourly', 'daily', 'rain', 'wind'].includes(modeParam)) {
        appState.currentMode = modeParam;
    } else {
        // Restore mode from localStorage for PWA state persistence
        try {
            const savedMode = localStorage.getItem('forecastCurrentMode');
            if (savedMode && ['full', 'history', 'hourly', 'daily', 'rain', 'wind'].includes(savedMode)) {
                appState.currentMode = savedMode;
            }
        } catch (error) {
            console.warn('Failed to restore current mode:', error);
        }
    }
    
    // Update active button for the mode
    if (elements.modeButtons) {
        elements.modeButtons.forEach(btn => {
            if (btn.dataset.mode === appState.currentMode) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });
    }
    
    // Try to load cached data first (unless URL specifies a different location)
    let cachedDataLoaded = false;
    let locationToLoad = null;
    
    if (locationParam) {
        // URL parameter takes precedence
        locationToLoad = locationParam;
    } else {
        // Check for last viewed location (even if not a favorite)
        const lastViewed = getLastViewedLocation();
        if (lastViewed) {
            // Check if lastViewed is a favorite - if so, prefer it over current location
            const favorites = getFavorites();
            let isFavorite = false;
            if (lastViewed.uid) {
                isFavorite = favorites.some(fav => fav.uid === lastViewed.uid);
            } else if (lastViewed.location) {
                const generatedUID = generateLocationUID(lastViewed.location);
                isFavorite = favorites.some(fav => fav.uid === generatedUID);
            }
            
            // If it's a favorite, prefer it and don't do current location detection
            if (isFavorite) {
                // Use the cache key from lastViewed (could be UID-based or location key)
                const cacheKey = lastViewed.key;
                console.log('Restoring last viewed favorite location:', lastViewed.name, 'cacheKey:', cacheKey, 'uid:', lastViewed.uid);
                
                // Try to load from cache for last viewed location
                const cache = loadWeatherDataFromCache(cacheKey);
                if (cache) {
                    // Pass searchQuery so loadCachedWeatherData can refresh if stale
                    cachedDataLoaded = loadCachedWeatherData(cacheKey, lastViewed.searchQuery);
                    if (cachedDataLoaded) {
                        // Update location input field with the saved location name
                        if (lastViewed.name && elements.locationInput) {
                            elements.locationInput.value = lastViewed.name;
                        }
                        
                        // Find matching favorite UID for button highlighting
                        let activeUID = lastViewed.uid;
                        if (!activeUID && cacheKey.startsWith('uid_')) {
                            // Extract UID from UID-based cache key
                            activeUID = cacheKey.replace('uid_', '');
                        } else if (!activeUID && lastViewed.location) {
                            // Generate UID from location
                            activeUID = generateLocationUID(lastViewed.location);
                        }
                        
                        // Update favorite button state and location buttons
                        updateFavoriteButtonState(activeUID);
                        // Render location buttons if favorites exist
                        if (favorites.length > 0) {
                            renderLocationButtons(activeUID);
                        }
                        // Update current location button state (not active since we're using a favorite)
                        updateCurrentLocationButtonState(false);
                        return; // Successfully loaded last viewed favorite location
                    }
                }
                // Cache load failed, will load fresh below
                locationToLoad = lastViewed.searchQuery;
            } else {
                // Not a favorite - check if it's 'here' (current location)
                if (lastViewed.searchQuery && lastViewed.searchQuery.toLowerCase() === 'here') {
                    // It's current location - set flag and load it
                    appState.isCurrentLocationActive = true;
                    locationToLoad = 'here';
                } else {
                    // Regular location (not favorite, not 'here') - load it
                    locationToLoad = lastViewed.searchQuery;
                }
            }
        } else {
            // Check for stored location (backward compatibility)
            const storedLocation = localStorage.getItem('forecastLocation');
            if (storedLocation && storedLocation.trim() !== '') {
                locationToLoad = storedLocation;
            }
        }
    }
    
    // If we have a location to load, try cache first
    if (locationToLoad) {
        // Try to find cache for this location
        let cache = null;
        if (locationToLoad.toLowerCase() !== 'here') {
            // Try to find favorite for this location
            const favorites = getFavorites();
            const favorite = favorites.find(fav => fav.searchQuery.toLowerCase() === locationToLoad.toLowerCase());
            if (favorite) {
                // Use UID-based cache key if available
                const favoriteCacheKey = favorite.uid ? `uid_${favorite.uid}` : favorite.key;
                cache = loadWeatherDataFromCache(favoriteCacheKey);
                if (cache) {
                    // Pass searchQuery so loadCachedWeatherData can refresh if stale
                    cachedDataLoaded = loadCachedWeatherData(favoriteCacheKey, favorite.searchQuery);
                }
            } else {
                // Not a favorite - try to load using the location key from lastViewed if available
                const lastViewed = getLastViewedLocation();
                if (lastViewed && lastViewed.key && !lastViewed.key.startsWith('uid_')) {
                    // Non-favorite location - use the location key (not UID-based)
                    cache = loadWeatherDataFromCache(lastViewed.key);
                    if (cache) {
                        cachedDataLoaded = loadCachedWeatherData(lastViewed.key, lastViewed.searchQuery);
                    }
                }
            }
        }
        
        // Fallback to current location cache
        if (!cachedDataLoaded) {
            cache = loadWeatherDataFromCache();
            if (cache) {
                // Check if cached location matches
                if (cache.location && locationToLoad.toLowerCase() === cache.location.toLowerCase()) {
                    cachedDataLoaded = loadCachedWeatherData();
                }
            }
        }
        
        if (cachedDataLoaded && cache) {
            // Cached data was loaded (stale check and refresh handled by loadCachedWeatherData)
            // Update favorite button state
            updateFavoriteButtonState();
            // Render location buttons if favorites exist
            const favorites = getFavorites();
            if (favorites.length > 0) {
                renderLocationButtons();
            }
            return; // Successfully loaded from cache
        }
    } else {
        // No specific location, check if we should prefer a favorite over current location
        const lastViewed = getLastViewedLocation();
        const favorites = getFavorites();
        
        // Check if lastViewed is a favorite
        let shouldUseFavorite = false;
        if (lastViewed) {
            if (lastViewed.uid) {
                shouldUseFavorite = favorites.some(fav => fav.uid === lastViewed.uid);
            } else if (lastViewed.location) {
                const generatedUID = generateLocationUID(lastViewed.location);
                shouldUseFavorite = favorites.some(fav => fav.uid === generatedUID);
            }
        }
        
        if (shouldUseFavorite && lastViewed) {
            // Prefer favorite over current location
            const cacheKey = lastViewed.key;
            const cache = loadWeatherDataFromCache(cacheKey);
            if (cache) {
                cachedDataLoaded = loadCachedWeatherData(cacheKey, lastViewed.searchQuery);
                if (cachedDataLoaded) {
                    if (lastViewed.name && elements.locationInput) {
                        elements.locationInput.value = lastViewed.name;
                    }
                    let activeUID = lastViewed.uid;
                    if (!activeUID && cacheKey.startsWith('uid_')) {
                        activeUID = cacheKey.replace('uid_', '');
                    } else if (!activeUID && lastViewed.location) {
                        activeUID = generateLocationUID(lastViewed.location);
                    }
                    updateFavoriteButtonState(activeUID);
                    if (favorites.length > 0) {
                        renderLocationButtons(activeUID);
                    }
                    updateCurrentLocationButtonState(false);
                    return;
                }
            }
            locationToLoad = lastViewed.searchQuery;
        } else {
            // Try current cache (might be 'here' or a non-favorite location)
            const cache = loadWeatherDataFromCache();
            if (cache) {
                // Determine search query from cache location
                const searchQuery = cache.location || elements.locationInput.value.trim() || 'here';
                cachedDataLoaded = loadCachedWeatherData(null, searchQuery);
                if (cachedDataLoaded) {
                    // Check if this is current location
                    if (searchQuery.toLowerCase() === 'here') {
                        appState.isCurrentLocationActive = true;
                        updateCurrentLocationButtonState(true);
                    } else {
                        updateCurrentLocationButtonState(false);
                    }
                    // Update favorite button state
                    updateFavoriteButtonState();
                    // Render location buttons if favorites exist
                    if (favorites.length > 0) {
                        renderLocationButtons();
                    }
                    // Update current location button state
                    // Check if this is 'here' or a favorite
                    if (locationToLoad && locationToLoad.toLowerCase() === 'here') {
                        appState.isCurrentLocationActive = true;
                        updateCurrentLocationButtonState(true);
                    } else {
                        appState.isCurrentLocationActive = false;
                        updateCurrentLocationButtonState(false);
                    }
                    return; // Successfully loaded from cache
                }
            }
        }
    }
    
    // No cached data available, proceed with normal initial load
    if (locationToLoad) {
        try {
            // Set location input to the saved name if available, otherwise use searchQuery
            const lastViewed = getLastViewedLocation();
            if (lastViewed && lastViewed.name && elements.locationInput) {
                elements.locationInput.value = lastViewed.name;
            } else if (elements.locationInput) {
                elements.locationInput.value = locationToLoad;
            }
            // Check if this is current location
            if (locationToLoad.toLowerCase() === 'here') {
                appState.isCurrentLocationActive = true;
            }
            await loadWeatherData(locationToLoad, false); // false = show errors
        } catch (error) {
            console.error('Error loading weather data:', error);
        }
    } else {
        // Only try to detect location automatically if no favorite was last viewed
        // This prevents overriding favorite selection
        const lastViewed = getLastViewedLocation();
        const favorites = getFavorites();
        let shouldDetectLocation = true;
        
        if (lastViewed) {
            // Check if lastViewed is a favorite
            if (lastViewed.uid) {
                shouldDetectLocation = !favorites.some(fav => fav.uid === lastViewed.uid);
            } else if (lastViewed.location) {
                const generatedUID = generateLocationUID(lastViewed.location);
                shouldDetectLocation = !favorites.some(fav => fav.uid === generatedUID);
            }
        }
        
        if (shouldDetectLocation) {
            // Try to detect location automatically (silently on failure)
            try {
                appState.isCurrentLocationActive = true;
                await loadWeatherData('here', true); // true = silent on location detection failure
            } catch (error) {
                console.error('Error loading initial weather data:', error);
                // Don't block app - user can still search manually
            }
        }
    }
    
    // Update favorite button state after load
    updateFavoriteButtonState();
    
    // Update current location button state based on what was loaded
    // Check if the loaded location is 'here' or a favorite
    const lastViewed = getLastViewedLocation();
    if (lastViewed && lastViewed.searchQuery && lastViewed.searchQuery.toLowerCase() === 'here') {
        appState.isCurrentLocationActive = true;
        updateCurrentLocationButtonState(true);
    } else {
        // Check if it's a favorite
        const favorites = getFavorites();
        let isFavorite = false;
        if (lastViewed && lastViewed.uid) {
            isFavorite = favorites.some(fav => fav.uid === lastViewed.uid);
        } else if (lastViewed && lastViewed.location) {
            const generatedUID = generateLocationUID(lastViewed.location);
            isFavorite = favorites.some(fav => fav.uid === generatedUID);
        }
        // If it's not 'here' and not a favorite, current location is not active
        appState.isCurrentLocationActive = false;
        updateCurrentLocationButtonState(false);
    }
    
    // Render location buttons if favorites exist
    const favorites = getFavorites();
    if (favorites.length > 0) {
        renderLocationButtons();
    }
}

// Set up event listeners
// Set up ResizeObserver to hide/show "Forecast" text based on input width
function setupForecastTextVisibility() {
    const locationInput = elements.locationInput;
    const forecastText = document.querySelector('.forecast-text');
    const headerLeft = document.querySelector('.header-left');
    
    if (!locationInput || !forecastText || !headerLeft) {
        console.warn('Required elements not found for forecast text visibility setup');
        return;
    }
    
    // Width thresholds for search input (in pixels) - using hysteresis to prevent oscillation
    // Hide text when input is narrow, show when it's wider (different thresholds prevent rapid toggling)
    const HIDE_THRESHOLD = 120;  // Hide text when input is below this width
    const SHOW_THRESHOLD = 270;  // Show text when input is above this width (much higher to prevent oscillation with large fonts)
    
    // Track current state to prevent unnecessary DOM updates
    let isTextHidden = false;
    
    // Function to check and update visibility
    const updateForecastTextVisibility = () => {
        const inputWidth = locationInput.offsetWidth;
        
        // Use hysteresis: different thresholds for hiding vs showing
        if (!isTextHidden && inputWidth < HIDE_THRESHOLD) {
            // Hide the text
            forecastText.classList.add('hidden');
            headerLeft.classList.add('forecast-text-hidden');
            isTextHidden = true;
        } else if (isTextHidden && inputWidth > SHOW_THRESHOLD) {
            // Show the text
            forecastText.classList.remove('hidden');
            headerLeft.classList.remove('forecast-text-hidden');
            isTextHidden = false;
        }
    };
    
    // Check on initial load
    updateForecastTextVisibility();
    
    // Set up ResizeObserver to monitor the search input width
    if (typeof ResizeObserver !== 'undefined') {
        const resizeObserver = new ResizeObserver(() => {
            updateForecastTextVisibility();
        });
        
        resizeObserver.observe(locationInput);
        
        // Also observe the header-right container in case it affects the input width
        const headerRight = document.querySelector('.header-right');
        if (headerRight) {
            resizeObserver.observe(headerRight);
        }
        
        // Also observe the header in case overall layout changes
        const header = document.querySelector('header');
        if (header) {
            resizeObserver.observe(header);
        }
    } else {
        // Fallback for browsers that don't support ResizeObserver
        // Use window resize event as fallback
        window.addEventListener('resize', updateForecastTextVisibility);
    }
}

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
            // DO NOT overwrite lastFetchTime here - it should only be set when data is actually fetched
            // The lastFetchTime represents when NWS data was fetched, not when auto-update is enabled
        });
    }
    
    // Favorite button
    if (elements.favoriteBtn) {
        elements.favoriteBtn.addEventListener('click', (e) => {
            e.preventDefault();
            handleFavoriteToggle();
        });
    }
    
    // Current location button
    if (elements.currentLocationBtn) {
        elements.currentLocationBtn.addEventListener('click', (e) => {
            e.preventDefault();
            handleCurrentLocationClick();
        });
    }
    
    // Locations button
    if (elements.locationsBtn) {
        elements.locationsBtn.addEventListener('click', (e) => {
            e.preventDefault();
            toggleLocationsDrawer();
        });
    }
    
    // Location buttons in drawer (delegated event handler)
    if (elements.locationButtons) {
        let clickTimer = null;
        let isDoubleClick = false;
        
        elements.locationButtons.addEventListener('click', (e) => {
            const locationBtn = e.target.closest('.location-btn');
            if (!locationBtn) return;
            
            // Ignore clicks on input fields (edit mode)
            if (e.target.classList.contains('location-btn-edit')) {
                return;
            }
            
            // Handle double-click for edit mode
            if (clickTimer) {
                clearTimeout(clickTimer);
                clickTimer = null;
                isDoubleClick = true;
                
                // Enter edit mode
                e.preventDefault();
                e.stopPropagation();
                handleLocationButtonEdit(locationBtn);
                return;
            }
            
            // Single click - wait to see if it becomes a double-click
            isDoubleClick = false;
            clickTimer = setTimeout(() => {
                if (!isDoubleClick) {
                    // Single click confirmed - navigate to location
                    // Prefer UID, fallback to key for backward compatibility
                    const uid = locationBtn.dataset.locationUid || locationBtn.dataset.locationKey;
                    handleLocationButtonClick(uid);
                }
                clickTimer = null;
            }, 300); // 300ms delay to detect double-click
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

// Handle favorite toggle
function handleFavoriteToggle() {
    if (!elements.favoriteBtn) {
        console.warn('Favorite button not found');
        return;
    }
    
    // Use stored locationKey if available, otherwise try to generate from appState.location
    let locationKey = appState.currentLocationKey;
    
    if (!locationKey && appState.location) {
        if (!appState.location.city || !appState.location.state) {
            console.warn('Cannot toggle favorite: location object missing city or state', appState.location);
            return;
        }
        locationKey = generateLocationKey(appState.location);
    }
    
    if (!locationKey) {
        console.warn('Cannot toggle favorite: location key not available', appState);
        return;
    }
    
    const currentlyFavorite = isFavorite(locationKey);
    console.log('Toggling favorite:', { locationKey, currentlyFavorite, location: appState.location });
    
    if (currentlyFavorite) {
        // Remove favorite
        removeFavorite(locationKey);
        appState.currentLocationKey = null;
        updateFavoriteButtonState();
        
        // Re-render location buttons if drawer is open
        if (elements.locationsDrawer && !elements.locationsDrawer.classList.contains('hidden')) {
            renderLocationButtons();
        }
    } else {
        // Add favorite - need location object
        if (!appState.location || !appState.location.city || !appState.location.state) {
            console.warn('Cannot add favorite: location object missing city or state', appState.location);
            return;
        }
        
        const locationText = formatLocationDisplayName(appState.location.city, appState.location.state);
        const searchQuery = elements.locationInput.value.trim() || locationText;
        const saved = saveFavorite(locationText, appState.location, searchQuery);
        if (saved) {
            // Update the stored locationKey to match the newly saved favorite
            appState.currentLocationKey = locationKey;
            elements.favoriteBtn.classList.add('active');
            
            // Re-render location buttons if drawer is open
            if (elements.locationsDrawer && !elements.locationsDrawer.classList.contains('hidden')) {
                renderLocationButtons();
            }
        } else {
            console.error('Failed to save favorite');
        }
    }
}

// Handle location button edit mode
function handleLocationButtonEdit(locationBtn) {
    if (!locationBtn) return;
    
    // Prefer UID, fallback to key for backward compatibility
    const uid = locationBtn.dataset.locationUid || locationBtn.dataset.locationKey;
    if (!uid) return;
    
    const favorite = getFavoriteByUID(uid) || getFavoriteByKey(uid);
    if (!favorite) return;
    
    // Get current display name (customName or name)
    const currentName = favorite.customName || favorite.name;
    
    // Create input field
    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'location-btn-edit';
    input.value = currentName;
    input.maxLength = 50; // Reasonable limit for button text
    
    // Store original button content and replace with input
    const originalContent = locationBtn.innerHTML;
    locationBtn.innerHTML = '';
    locationBtn.appendChild(input);
    input.focus();
    input.select();
    
    // Save handler
    const saveEdit = () => {
        const newName = input.value.trim();
        const oldName = favorite.customName || favorite.name;
        
        // Only update if name changed
        if (newName !== oldName) {
            const favoriteUID = favorite.uid || favorite.key;
            if (updateFavoriteCustomName(favoriteUID, newName)) {
                renderLocationButtons();
            }
        } else {
            // Name unchanged, just restore button
            locationBtn.innerHTML = originalContent;
        }
    };
    
    // Cancel handler
    const cancelEdit = () => {
        locationBtn.innerHTML = originalContent;
    };
    
    // Handle Enter key (save)
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            saveEdit();
        } else if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            cancelEdit();
        }
    });
    
    // Handle blur (save on click outside)
    input.addEventListener('blur', (e) => {
        // Use setTimeout to allow click events to process first
        setTimeout(() => {
            saveEdit();
        }, 200);
    });
}

// Handle location button click from drawer
async function handleLocationButtonClick(uid) {
    if (!uid) return;
    
    // Support both UID and key for backward compatibility
    let favorite = getFavoriteByUID(uid) || getFavoriteByKey(uid);
    if (!favorite) {
        console.warn('handleLocationButtonClick: Favorite not found for UID:', uid);
        return;
    }
    
    // CRITICAL: Ensure favorite has a UID - generate one if missing
    if (!favorite.uid && favorite.location) {
        const generatedUID = generateLocationUID(favorite.location);
        if (generatedUID) {
            // Update favorite with UID
            favorite.uid = generatedUID;
            const favorites = getFavorites();
            const favoriteIndex = favorites.findIndex(fav => 
                (fav.uid === uid || fav.key === uid) || 
                (fav.key === favorite.key && !fav.uid)
            );
            if (favoriteIndex !== -1) {
                favorites[favoriteIndex].uid = generatedUID;
                localStorage.setItem('forecastFavorites', JSON.stringify(favorites));
                console.log('Generated and saved UID for favorite:', generatedUID, favorite.name || favorite.searchQuery);
            }
            // Update local favorite reference
            favorite = getFavoriteByUID(generatedUID) || favorite;
        }
    }
    
    // Use UID-based cache key for favorites (more stable than location keys)
    // Fallback to location key for backward compatibility
    const cacheKey = favorite.uid ? `uid_${favorite.uid}` : favorite.key;
    console.log('handleLocationButtonClick: Loading favorite', favorite.name || favorite.searchQuery, 'UID:', favorite.uid, 'cacheKey:', cacheKey, 'searchQuery:', favorite.searchQuery);
    
    // Load cached data immediately (this will update UI state including favorite button and location buttons)
    // Pass searchQuery so loadCachedWeatherData can refresh if stale
    const cacheLoaded = loadCachedWeatherData(cacheKey, favorite.searchQuery);
    
    if (cacheLoaded) {
        console.log('handleLocationButtonClick: Successfully loaded cached data for favorite:', favorite.name || favorite.searchQuery);
        // Update URL
        updateURL(favorite.searchQuery, appState.currentMode);
        
        // CRITICAL: Ensure star button is enabled when a favorite is selected
        // Use UID (preferred) or key as fallback - this MUST match what isFavorite expects
        const favoriteUID = favorite.uid || favorite.key;
        // Explicitly enable star button since we know this is a favorite
        updateFavoriteButtonState(favoriteUID);
        // Double-check: if star button is not active, force it active
        if (elements.favoriteBtn && !elements.favoriteBtn.classList.contains('active')) {
            console.warn('Star button not active after updateFavoriteButtonState, forcing active for favorite:', favoriteUID);
            elements.favoriteBtn.classList.add('active');
            appState.currentLocationKey = favoriteUID;
        }
        renderLocationButtons(favoriteUID);
        
        // Update current location button state (not active since we're using a favorite)
        appState.isCurrentLocationActive = false;
        updateCurrentLocationButtonState(false);
        
        // Stale cache check and background refresh is now handled by loadCachedWeatherData
    } else {
        console.log('handleLocationButtonClick: No cache found for favorite:', favorite.name || favorite.searchQuery, 'key:', cacheKey, 'will fetch fresh data');
        // No cache, load fresh data
        await loadWeatherData(favorite.searchQuery, false, false);
        // CRITICAL: Ensure star button is enabled after loading fresh data
        // Use UID (preferred) or key as fallback
        const favoriteUID = favorite.uid || favorite.key;
        updateFavoriteButtonState(favoriteUID);
        // Double-check: if star button is not active, force it active
        if (elements.favoriteBtn && !elements.favoriteBtn.classList.contains('active')) {
            console.warn('Star button not active after loadWeatherData, forcing active for favorite:', favoriteUID);
            elements.favoriteBtn.classList.add('active');
            appState.currentLocationKey = favoriteUID;
        }
        renderLocationButtons(favoriteUID);
        
        // Update current location button state (not active since we're using a favorite)
        appState.isCurrentLocationActive = false;
        updateCurrentLocationButtonState(false);
    }
}

// Handle current location button click
async function handleCurrentLocationClick() {
    if (!elements.currentLocationBtn) {
        console.warn('Current location button not found');
        return;
    }
    
    try {
        // Clear current location key to ensure we're not matching a favorite
        appState.currentLocationKey = null;
        
        // Set current location as active BEFORE loading to ensure cache check works correctly
        appState.isCurrentLocationActive = true;
        updateCurrentLocationButtonState(true);
        
        // Clear favorite button state since we're switching to current location
        updateFavoriteButtonState(null);
        
        // Clear any existing weather data to force a fresh load
        // This ensures we don't use stale data from a favorite location
        appState.weatherData = null;
        appState.location = null;
        
        // Load current location - this will now properly detect that we want 'here'
        await loadWeatherData('here', false);
    } catch (error) {
        console.error('Error loading current location:', error);
        // Reset button state on error
        appState.isCurrentLocationActive = false;
        updateCurrentLocationButtonState(false);
    }
}

// Update current location button state (color when active, grayscale when not)
function updateCurrentLocationButtonState(isActive) {
    if (!elements.currentLocationBtn) {
        return;
    }
    
    if (isActive) {
        elements.currentLocationBtn.classList.add('active');
        elements.currentLocationBtn.style.filter = 'none'; // Color
    } else {
        elements.currentLocationBtn.classList.remove('active');
        elements.currentLocationBtn.style.filter = 'grayscale(100%)'; // Grayscale
    }
}

// Update favorite button state based on current location
function updateFavoriteButtonState(identifier = null) {
    if (!elements.favoriteBtn) {
        return;
    }
    
    // Use provided identifier (UID or key) if available, otherwise try to generate from appState.location
    let identifierToCheck = identifier;
    if (!identifierToCheck && appState.location) {
        // Prefer UID (more stable)
        identifierToCheck = generateLocationUID(appState.location);
        
        // If UID doesn't match, try key as fallback
        if (identifierToCheck && !isFavorite(identifierToCheck)) {
            const keyToCheck = generateLocationKey(appState.location);
            if (keyToCheck && isFavorite(keyToCheck)) {
                identifierToCheck = keyToCheck;
            } else {
                // Try to find a matching favorite by comparing location objects
                const favorites = getFavorites();
                const matchingFavorite = favorites.find(fav => {
                    if (fav.location && appState.location) {
                        const favCity = (fav.location.city || '').trim().toLowerCase();
                        const favState = (fav.location.state || '').trim().toUpperCase();
                        const currentCity = (appState.location.city || '').trim().toLowerCase();
                        const currentState = (appState.location.state || '').trim().toUpperCase();
                        
                        if (favCity === currentCity && favState === currentState) {
                            return true;
                        }
                    }
                    return false;
                });
                
                if (matchingFavorite) {
                    // Use the favorite's UID (preferred) or key as fallback
                    identifierToCheck = matchingFavorite.uid || matchingFavorite.key;
                }
            }
        }
    }
    
    // Store the identifier in appState for use by handleFavoriteToggle
    if (identifierToCheck) {
        appState.currentLocationKey = identifierToCheck; // Keep same property name for compatibility
    }
    
    if (!identifierToCheck) {
        appState.currentLocationKey = null;
        elements.favoriteBtn.classList.remove('active');
        return;
    }
    
    // CRITICAL: Check if identifier is a favorite - try both UID and key
    // This ensures favorites are correctly identified even if identifier format varies
    const isFav = isFavorite(identifierToCheck);
    
    if (isFav) {
        elements.favoriteBtn.classList.add('active');
        // Ensure appState.currentLocationKey is set to the correct identifier
        // Prefer UID if available, otherwise use the identifier passed
        if (identifierToCheck && !identifierToCheck.startsWith('uid_')) {
            // Try to find the favorite and use its UID if available
            const favorite = getFavoriteByUID(identifierToCheck) || getFavoriteByKey(identifierToCheck);
            if (favorite && favorite.uid) {
                appState.currentLocationKey = favorite.uid;
            }
        }
    } else {
        // Not a favorite - check if any favorite button is active
        // If no favorite button is active, star button should be disabled
        const hasActiveFavoriteButton = elements.locationButtons && 
            elements.locationButtons.querySelector('.location-btn.active');
        
        if (!hasActiveFavoriteButton) {
            // No favorite button is active, so star button should be disabled
            elements.favoriteBtn.classList.remove('active');
            appState.currentLocationKey = null;
        }
    }
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
// Only fetches if not already cached and fresh (10 minutes)
async function checkObservationsAvailability(pointsData, timeZone) {
    // Preserve existing observations data in case refresh fails (especially important in history mode)
    const existingObservationsData = appState.observationsData;
    const existingObservationsAvailable = appState.observationsAvailable;
    
    try {
        // Check if we already have fresh observations in appState
        if (appState.observationsData && appState.observationsAvailable && appState.lastFetchTime) {
            if (!isCacheStale(appState.lastFetchTime)) {
                console.log('Using existing fresh observations data (age:', Math.round((Date.now() - appState.lastFetchTime) / 1000), 'seconds)');
                return true;
            } else {
                console.log('Existing observations are stale, will refresh');
            }
        }
        
        if (!pointsData || !pointsData.properties || !pointsData.properties.observationStations) {
            // Only clear observations if we don't have existing cached data to preserve
            if (!existingObservationsData) {
                appState.observationsAvailable = false;
                appState.observationsData = null;
            } else {
                console.log('No observation stations available, preserving existing cached observations data');
            }
            return false;
        }
        
        // Try to fetch observation stations
        const stationId = await fetchNWSObservationStations(pointsData);
        if (!stationId) {
            // Only clear observations if we don't have existing cached data to preserve
            if (!existingObservationsData) {
                appState.observationsAvailable = false;
                appState.observationsData = null;
            } else {
                console.log('No observation station found, preserving existing cached observations data');
            }
            return false;
        }
        
        // Try to fetch observations
        const observationsData = await fetchNWSObservations(stationId, timeZone);
        if (!observationsData || !observationsData.features || observationsData.features.length === 0) {
            // Only clear observations if we don't have existing cached data to preserve
            if (!existingObservationsData) {
                appState.observationsAvailable = false;
                appState.observationsData = null;
            } else {
                console.log('No observations data returned, preserving existing cached observations data');
            }
            return false;
        }
        
        // Process observations data
        const processedObservations = processObservationsData(observationsData, timeZone);
        if (!processedObservations || processedObservations.length === 0) {
            // Only clear observations if we don't have existing cached data to preserve
            if (!existingObservationsData) {
                appState.observationsAvailable = false;
                appState.observationsData = null;
            } else {
                console.log('Processed observations empty, preserving existing cached observations data');
            }
            return false;
        }
        
        appState.observationsData = processedObservations;
        appState.observationsAvailable = true;
        return true;
    } catch (error) {
        console.error('Error checking observations availability:', error);
        // Preserve existing observations data on error (especially important in history mode)
        if (!existingObservationsData) {
            appState.observationsAvailable = false;
            appState.observationsData = null;
        } else {
            console.log('Error fetching observations, preserving existing cached observations data');
        }
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

// Generate a stable UID for a location based on coordinates or location data
function generateLocationUID(location) {
    if (!location) return null;
    
    // Prefer coordinates for stable UID (most reliable)
    if (location.lat !== undefined && location.lon !== undefined) {
        // Round to 4 decimal places (~11 meters precision) to handle slight variations
        const lat = Math.round(location.lat * 10000) / 10000;
        const lon = Math.round(location.lon * 10000) / 10000;
        return `loc_${lat}_${lon}`;
    }
    
    // Fallback to city+state hash if coordinates not available
    const city = (location.city || '').trim().toLowerCase().replace(/[^a-zA-Z0-9\s]/g, '');
    const state = (location.state || '').trim().toUpperCase().replace(/[^a-zA-Z0-9]/g, '');
    if (!city || !state) return null;
    
    // Create a simple hash for stability
    const hash = `${city}_${state}`;
    return `loc_${hash}`;
}

// Location key generation - normalizes city and state to create unique key (used for cache lookups)
function generateLocationKey(location) {
    if (!location) return null;
    const city = (location.city || '').trim().replace(/[^a-zA-Z0-9\s]/g, '');
    const state = (location.state || '').trim().replace(/[^a-zA-Z0-9]/g, '').toUpperCase();
    if (!city || !state) return null;
    return `${city},${state}`;
}

// Migrate old favorites to use UID system
function migrateFavorites() {
    try {
        const favorites = getFavorites();
        if (favorites.length === 0) {
            return true; // No favorites to migrate, success
        }
        
        // Check if all favorites already have UIDs (already migrated)
        const allHaveUIDs = favorites.every(fav => fav.uid);
        if (allHaveUIDs) {
            console.log('All favorites already have UIDs, skipping migration');
            return true; // Already migrated, no work needed
        }
        
        const migratedFavorites = [];
        const failedFavorites = [];
        let newlyMigratedCount = 0;
        
        // Verify and fix each favorite
        for (let i = 0; i < favorites.length; i++) {
            const favorite = favorites[i];
            let canFix = false;
            let fixedFavorite = null;
            
            try {
                // Check if favorite has required structure
                if (!favorite || typeof favorite !== 'object') {
                    console.warn('Invalid favorite structure at index', i, favorite);
                    failedFavorites.push({ index: i, reason: 'Invalid structure' });
                    continue;
                }
                
                // Skip favorites that already have UIDs (already migrated)
                if (favorite.uid) {
                    migratedFavorites.push(favorite);
                    continue; // Already migrated, no work needed
                }
                
                // Check if favorite has a location object with city and state
                if (!favorite.location || !favorite.location.city || !favorite.location.state) {
                    console.warn('Favorite missing location data at index', i, favorite);
                    failedFavorites.push({ index: i, reason: 'Missing location data' });
                    continue;
                }
                
                // Try to get the correct location object from cache (if available)
                // Old favorites might have state as "US" instead of the actual state code
                let correctLocation = favorite.location;
                let correctKey = null;
                
                // First, try to load cache with the old key to get the correct location
                if (favorite.key) {
                    try {
                        const oldCache = loadWeatherDataFromCache(favorite.key);
                        if (oldCache && oldCache.data && oldCache.data.weatherData && oldCache.data.weatherData.location) {
                            // Use the location from cached weather data (this has the correct state)
                            correctLocation = oldCache.data.weatherData.location;
                            console.log('Found correct location in cache for favorite:', {
                                oldLocation: favorite.location,
                                correctLocation: correctLocation
                            });
                        }
                    } catch (cacheError) {
                        console.warn('Could not load cache to fix location for favorite:', favorite.key, cacheError);
                    }
                }
                
                // Generate UID (primary identifier) and location key (for cache)
                const uid = generateLocationUID(correctLocation);
                correctKey = generateLocationKey(correctLocation);
                
                if (!uid) {
                    console.warn('Cannot generate UID for favorite at index', i, favorite);
                    failedFavorites.push({ index: i, reason: 'Cannot generate UID' });
                    continue;
                }
                
                if (!correctKey) {
                    console.warn('Cannot generate key for favorite at index', i, favorite);
                    failedFavorites.push({ index: i, reason: 'Cannot generate key' });
                    continue;
                }
                
                // Create fixed favorite with UID, correct key, and location
                fixedFavorite = {
                    uid: uid, // Primary identifier - stable and not dependent on location name format
                    key: correctKey, // Used for cache lookups
                    name: favorite.name || '',
                    location: correctLocation, // Use the corrected location object
                    searchQuery: favorite.searchQuery || favorite.name || ''
                };
                
                // Preserve customName if it exists
                if (favorite.customName !== undefined) {
                    fixedFavorite.customName = favorite.customName;
                }
                
                // If old favorite had a UID, preserve it (for consistency)
                if (favorite.uid) {
                    // Only update if the UID would be different (location changed significantly)
                    if (favorite.uid !== uid) {
                        console.log('UID changed for favorite:', { oldUID: favorite.uid, newUID: uid });
                    }
                    // Use new UID to ensure it matches current location
                }
                
                // Verify the fixed favorite is valid
                if (!fixedFavorite.key || !fixedFavorite.location || !fixedFavorite.location.city || !fixedFavorite.location.state) {
                    console.warn('Fixed favorite is still invalid at index', i, fixedFavorite);
                    failedFavorites.push({ index: i, reason: 'Fixed favorite invalid' });
                    continue;
                }
                
                canFix = true;
                
                // Migrate cache if it exists with old key (or update key if location was corrected)
                if (favorite.key && favorite.key !== correctKey) {
                    try {
                        const oldCache = loadWeatherDataFromCache(favorite.key);
                        if (oldCache) {
                            // Copy cache to new key location
                            localStorage.setItem(`forecastCachedData_${correctKey}`, JSON.stringify(oldCache.data));
                            localStorage.setItem(`forecastCachedLocation_${correctKey}`, oldCache.location);
                            localStorage.setItem(`forecastCachedTimestamp_${correctKey}`, oldCache.timestamp.toISOString());
                            console.log('Migrated cache from', favorite.key, 'to', correctKey);
                        }
                    } catch (cacheError) {
                        console.warn('Failed to migrate cache for favorite:', favorite.key, cacheError);
                        // Don't fail the migration if cache migration fails
                    }
                } else if (favorite.key === correctKey && correctLocation !== favorite.location) {
                    // Key is the same but location object was corrected - update cache if it exists
                    try {
                        const existingCache = loadWeatherDataFromCache(favorite.key);
                        if (existingCache && existingCache.data && existingCache.data.weatherData) {
                            // Update the location in the cached weather data
                            existingCache.data.weatherData.location = correctLocation;
                            localStorage.setItem(`forecastCachedData_${correctKey}`, JSON.stringify(existingCache.data));
                            console.log('Updated location object in cache for favorite:', correctKey);
                        }
                    } catch (cacheError) {
                        console.warn('Failed to update cache location for favorite:', favorite.key, cacheError);
                        // Don't fail the migration if cache update fails
                    }
                }
                
            } catch (error) {
                console.error('Error processing favorite at index', i, error);
                failedFavorites.push({ index: i, reason: 'Processing error: ' + error.message });
                continue;
            }
            
            if (canFix && fixedFavorite) {
                migratedFavorites.push(fixedFavorite);
                newlyMigratedCount++;
                const displayName = fixedFavorite.customName || fixedFavorite.name;
                console.log('Successfully migrated favorite:', {
                    oldKey: favorite.key,
                    newKey: fixedFavorite.key,
                    uid: fixedFavorite.uid,
                    name: fixedFavorite.name,
                    displayName: displayName,
                    hasCustomName: !!fixedFavorite.customName
                });
            }
        }
        
        // If any favorites failed to migrate, flush all and show error
        if (failedFavorites.length > 0) {
            console.error('Failed to migrate', failedFavorites.length, 'favorites:', failedFavorites);
            
            // Clear all favorites
            try {
                localStorage.removeItem('forecastFavorites');
                console.log('Cleared all favorites due to migration failure');
            } catch (error) {
                console.error('Failed to clear favorites:', error);
            }
            
            // Show error message to user
            if (elements.errorMessage) {
                showError('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
            } else {
                // If errorMessage element isn't ready yet, show alert
                setTimeout(() => {
                    if (elements.errorMessage) {
                        showError('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
                    } else {
                        alert('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
                    }
                }, 1000);
            }
            
            return false;
        }
        
        // All favorites were successfully processed
        // Only save if we actually made changes (migrated some favorites)
        if (newlyMigratedCount > 0 && migratedFavorites.length > 0) {
            try {
                localStorage.setItem('forecastFavorites', JSON.stringify(migratedFavorites));
                console.log('Favorites migration completed successfully. Migrated', newlyMigratedCount, 'favorites');
                return true;
            } catch (error) {
                console.error('Failed to save migrated favorites:', error);
                
                // Clear favorites and show error
                try {
                    localStorage.removeItem('forecastFavorites');
                } catch (clearError) {
                    console.error('Failed to clear favorites:', clearError);
                }
                
                if (elements.errorMessage) {
                    showError('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
                }
                
                return false;
            }
        }
        
        // No changes needed (all favorites already migrated or no favorites)
        return true;
        
    } catch (error) {
        console.error('Critical error during favorites migration:', error);
        
        // Clear favorites and show error
        try {
            localStorage.removeItem('forecastFavorites');
        } catch (clearError) {
            console.error('Failed to clear favorites:', clearError);
        }
        
        if (elements.errorMessage) {
            showError('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
        } else {
            setTimeout(() => {
                if (elements.errorMessage) {
                    showError('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
                } else {
                    alert('Failed to migrate Favorites. All favorites have been cleared. Please re-add your favorite locations.');
                }
            }, 1000);
        }
        
        return false;
    }
}

// Favorites management functions
function getFavorites() {
    try {
        const favorites = localStorage.getItem('forecastFavorites');
        return favorites ? JSON.parse(favorites) : [];
    } catch (error) {
        console.warn('Failed to load favorites:', error);
        return [];
    }
}

function saveFavorite(location, locationObject, searchQuery, customName) {
    try {
        if (!locationObject || !locationObject.city || !locationObject.state) {
            console.warn('Cannot save favorite: invalid location object', locationObject);
            return false;
        }
        
        const uid = generateLocationUID(locationObject);
        const locationKey = generateLocationKey(locationObject);
        
        if (!uid) {
            console.warn('Cannot save favorite: invalid UID', locationObject);
            return false;
        }
        
        if (!locationKey) {
            console.warn('Cannot save favorite: invalid location key', locationObject);
            return false;
        }
        
        const favorites = getFavorites();
        
        // Check if already exists by UID (primary identifier)
        const existingIndex = favorites.findIndex(fav => fav.uid === uid);
        if (existingIndex !== -1) {
            // Update existing favorite if customName is provided
            if (customName !== undefined) {
                favorites[existingIndex].customName = customName || null;
                localStorage.setItem('forecastFavorites', JSON.stringify(favorites));
                console.log('Favorite custom name updated:', uid, customName);
            }
            return true; // Already favorited
        }
        
        // Add new favorite with UID
        const newFavorite = {
            uid: uid, // Primary identifier
            key: locationKey, // Used for cache lookups
            name: location,
            location: locationObject,
            searchQuery: searchQuery || location
        };
        
        // Add customName if provided
        if (customName !== undefined) {
            newFavorite.customName = customName || null;
        }
        
        favorites.push(newFavorite);
        
        localStorage.setItem('forecastFavorites', JSON.stringify(favorites));
        console.log('Favorite saved:', newFavorite);
        return true;
    } catch (error) {
        console.error('Failed to save favorite:', error);
        return false;
    }
}

function removeFavorite(identifier) {
    try {
        const favorites = getFavorites();
        // Support both UID and key for backward compatibility
        const filtered = favorites.filter(fav => fav.uid !== identifier && fav.key !== identifier);
        localStorage.setItem('forecastFavorites', JSON.stringify(filtered));
        
        // Find the favorite to get its key for cache clearing
        const removedFavorite = favorites.find(fav => fav.uid === identifier || fav.key === identifier);
        if (removedFavorite && removedFavorite.key) {
            clearLocationCache(removedFavorite.key);
        }
        
        return true;
    } catch (error) {
        console.warn('Failed to remove favorite:', error);
        return false;
    }
}

function isFavorite(identifier) {
    if (!identifier) return false;
    const favorites = getFavorites();
    // Check by UID first (preferred), then by key (backward compatibility)
    return favorites.some(fav => fav.uid === identifier || fav.key === identifier);
}

function getFavoriteByUID(uid) {
    if (!uid) return null;
    const favorites = getFavorites();
    return favorites.find(fav => fav.uid === uid) || null;
}

function getFavoriteByKey(locationKey) {
    if (!locationKey) return null;
    const favorites = getFavorites();
    // Prefer UID-based lookup, but support key for backward compatibility
    const byKey = favorites.find(fav => fav.key === locationKey);
    if (byKey) return byKey;
    
    // If not found by key, try to find by matching location
    if (appState.location) {
        const uid = generateLocationUID(appState.location);
        if (uid) {
            return favorites.find(fav => fav.uid === uid) || null;
        }
    }
    
    return null;
}

function updateFavoriteCustomName(identifier, customName) {
    try {
        if (!identifier) {
            console.warn('Cannot update custom name: identifier (UID or key) is required');
            return false;
        }
        
        const favorites = getFavorites();
        // Support both UID and key for backward compatibility
        const favoriteIndex = favorites.findIndex(fav => fav.uid === identifier || fav.key === identifier);
        
        if (favoriteIndex === -1) {
            console.warn('Cannot update custom name: favorite not found', identifier);
            return false;
        }
        
        // Update customName (null or empty string removes it)
        if (customName && customName.trim()) {
            favorites[favoriteIndex].customName = customName.trim();
        } else {
            delete favorites[favoriteIndex].customName;
        }
        
        localStorage.setItem('forecastFavorites', JSON.stringify(favorites));
        console.log('Favorite custom name updated:', identifier, favorites[favoriteIndex].customName || 'removed');
        return true;
    } catch (error) {
        console.error('Failed to update favorite custom name:', error);
        return false;
    }
}

// Last viewed location functions
function saveLastViewedLocation(location, locationObject, searchQuery) {
    try {
        // Use appState.currentLocationKey if available (this is the actual cache key being used)
        // This ensures we save the same key that was used for caching (UID-based if favorite)
        let cacheKey = appState.currentLocationKey;
        let locationUID = null;
        
        // Extract UID from cache key if it's UID-based
        if (cacheKey && cacheKey.startsWith('uid_')) {
            locationUID = cacheKey.replace('uid_', '');
        } else if (locationObject) {
            // Try to find matching favorite to get UID
            locationUID = generateLocationUID(locationObject);
            if (locationUID) {
                const favorite = getFavoriteByUID(locationUID);
                if (favorite) {
                    // Use UID-based cache key if favorite found
                    cacheKey = favorite.uid ? `uid_${favorite.uid}` : favorite.key;
                }
            }
        }
        
        // Fallback to location key if no cache key set
        if (!cacheKey && locationObject) {
            cacheKey = generateLocationKey(locationObject);
        }
        
        if (!cacheKey) return;
        
        // If we don't have UID yet, try to get it from the cache key or location
        if (!locationUID && cacheKey.startsWith('uid_')) {
            locationUID = cacheKey.replace('uid_', '');
        } else if (!locationUID && locationObject) {
            locationUID = generateLocationUID(locationObject);
        }
        
        const lastViewed = {
            key: cacheKey, // Use cache key (UID-based if favorite, otherwise location key)
            uid: locationUID, // Store UID for matching
            name: location,
            location: locationObject,
            searchQuery: searchQuery || location
        };
        
        localStorage.setItem('forecastLastViewedLocation', JSON.stringify(lastViewed));
        console.log('Saved last viewed location:', { name: location, cacheKey, uid: locationUID });
    } catch (error) {
        console.warn('Failed to save last viewed location:', error);
    }
}

function getLastViewedLocation() {
    try {
        const lastViewed = localStorage.getItem('forecastLastViewedLocation');
        return lastViewed ? JSON.parse(lastViewed) : null;
    } catch (error) {
        console.warn('Failed to load last viewed location:', error);
        return null;
    }
}

// Cache storage functions
// timestamp: Optional. If provided, updates the cache timestamp (only when fetching from NWS API).
//           If not provided, preserves the existing cache timestamp (when updating cache with other data).
function saveWeatherDataToCache(weatherData, location, timestamp = null) {
    try {
        const cacheData = {
            weatherData: weatherData,
            observationsData: appState.observationsData,
            observationsAvailable: appState.observationsAvailable
        };
        
        // Generate cache key for location-specific cache
        // CRITICAL: Prefer UID-based cache keys for favorites (more stable than location keys)
        // This avoids issues with location name formatting (e.g., US vs AK)
        let cacheKey = null;
        let locationString = '';
        
        // First, check if appState.currentLocationKey is already a UID-based key
        // (This is set after fetching when a matching favorite is found)
        if (appState.currentLocationKey && appState.currentLocationKey.startsWith('uid_')) {
            // Use the UID-based key that was already set (from the favorite's stored UID)
            cacheKey = appState.currentLocationKey;
            console.log('saveWeatherDataToCache: Using existing UID-based cache key from appState.currentLocationKey:', cacheKey);
        } else {
            // Try to get the favorite's stored UID directly (don't generate a new one)
            let favorite = null;
            if (appState.location) {
                // Try to find favorite by location UID (but use the favorite's stored UID, not the generated one)
                const locationUID = generateLocationUID(appState.location);
                if (locationUID) {
                    favorite = getFavoriteByUID(locationUID);
                }
            }
            
            // If we found a favorite, use its stored UID (not the generated one)
            if (favorite && favorite.uid) {
                cacheKey = `uid_${favorite.uid}`;
                console.log('saveWeatherDataToCache: Using favorite\'s stored UID for cache key:', cacheKey, 'for favorite:', favorite.name || favorite.searchQuery);
            }
        }
        
        // Set locationString from location object (needed for both UID-based and location-based keys)
        // Prefer appState.location if available (more reliable than the location parameter)
        const locationToUse = appState.location || location;
        if (locationToUse) {
            if (typeof locationToUse === 'object' && locationToUse.city && locationToUse.state) {
                // Store formatted location string for display (removes ", US")
                locationString = formatLocationDisplayName(locationToUse.city, locationToUse.state);
            } else if (typeof locationToUse === 'string') {
                // Location string (backward compatibility)
                locationString = locationToUse;
            }
        }
        
        // Ensure locationString is set (fallback to empty string if not available)
        if (!locationString && appState.location) {
            locationString = formatLocationDisplayName(appState.location.city, appState.location.state) || '';
        }
        
        // Fallback to location key if no UID-based key found
        if (!cacheKey) {
            // Use appState.currentLocationKey if available (from favorite's key)
            cacheKey = appState.currentLocationKey || null;
            
            if (location) {
                if (typeof location === 'object' && location.city && location.state) {
                    // Only generate key if we don't already have one from appState.currentLocationKey
                    if (!cacheKey) {
                        cacheKey = generateLocationKey(location);
                    }
                } else if (typeof location === 'string') {
                    // Try to parse location string to get key (only if we don't have one from appState)
                    if (!cacheKey) {
                        const parts = location.split(',').map(s => s.trim());
                        if (parts.length >= 2) {
                            cacheKey = generateLocationKey({ city: parts[0], state: parts[1] });
                        }
                    }
                }
            }
            
            // Log which key we're using for cache
            if (cacheKey) {
                console.log('saveWeatherDataToCache: Using location key:', cacheKey, 'from appState.currentLocationKey:', !!appState.currentLocationKey);
            }
        }
        
        // Use cacheKey as locationKey for the rest of the function (for backward compatibility)
        const locationKey = cacheKey;
        
        // Determine timestamp to use
        // If timestamp is provided (NWS API fetch), use it
        // Otherwise, preserve existing cache timestamp (don't update it)
        let timestampToUse = null;
        if (timestamp) {
            // New timestamp provided (from NWS API fetch)
            timestampToUse = timestamp instanceof Date ? timestamp.toISOString() : timestamp;
        } else {
            // No timestamp provided - preserve existing cache timestamp
            // Load existing timestamp from cache
            if (locationKey) {
                const existingTimestamp = localStorage.getItem(`forecastCachedTimestamp_${locationKey}`);
                if (existingTimestamp) {
                    timestampToUse = existingTimestamp;
                }
            }
            // Fallback to current location cache timestamp
            if (!timestampToUse) {
                const existingTimestamp = localStorage.getItem('forecastCachedTimestamp');
                if (existingTimestamp) {
                    timestampToUse = existingTimestamp;
                }
            }
            // If no existing timestamp found, try to use appState.lastFetchTime (should be set from cache)
            // This is a safety fallback to preserve the timestamp even if localStorage lookup fails
            if (!timestampToUse) {
                if (appState.lastFetchTime) {
                    timestampToUse = appState.lastFetchTime instanceof Date 
                        ? appState.lastFetchTime.toISOString() 
                        : appState.lastFetchTime;
                    console.warn('No cache timestamp found in localStorage, using appState.lastFetchTime:', timestampToUse);
                } else {
                    // Last resort: use current time but log error
                    timestampToUse = new Date().toISOString();
                    console.error('CRITICAL: No cache timestamp found and appState.lastFetchTime not set. Using current time (cache age will be incorrect):', timestampToUse);
                }
            }
        }
        
        if (locationKey) {
            // Save to location-specific cache
            localStorage.setItem(`forecastCachedData_${locationKey}`, JSON.stringify(cacheData));
            // Ensure locationString is not empty (use a placeholder if needed)
            const locationStringToSave = locationString || (appState.location ? formatLocationDisplayName(appState.location.city, appState.location.state) : 'Unknown Location');
            localStorage.setItem(`forecastCachedLocation_${locationKey}`, locationStringToSave);
            localStorage.setItem(`forecastCachedTimestamp_${locationKey}`, timestampToUse);
            
            // Update in-memory cache
            const memoryCacheKey = `${locationKey}_${timestampToUse}`;
            appState.cachedWeatherDataByKey.set(memoryCacheKey, cacheData);
        }
        
        // Also maintain current location cache for backward compatibility
        localStorage.setItem('forecastCachedData', JSON.stringify(cacheData));
        localStorage.setItem('forecastCachedLocation', locationString);
        localStorage.setItem('forecastCachedTimestamp', timestampToUse);
        
        // Update in-memory cache for default location
        const defaultMemoryCacheKey = `default_${timestampToUse}`;
        appState.cachedWeatherDataByKey.set(defaultMemoryCacheKey, cacheData);
        
        // Limit memory cache size to prevent memory leaks (keep last 10 entries)
        if (appState.cachedWeatherDataByKey.size > 10) {
            const firstKey = appState.cachedWeatherDataByKey.keys().next().value;
            appState.cachedWeatherDataByKey.delete(firstKey);
        }
    } catch (error) {
        console.warn('Failed to save weather data to cache:', error);
    }
}

function loadWeatherDataFromCache(locationKey = null) {
    try {
        let cachedData, cachedLocation, cachedTimestamp;
        
        if (locationKey) {
            // Load from location-specific cache
            const dataKey = `forecastCachedData_${locationKey}`;
            const locationKeyStr = `forecastCachedLocation_${locationKey}`;
            const timestampKey = `forecastCachedTimestamp_${locationKey}`;
            cachedData = localStorage.getItem(dataKey);
            cachedLocation = localStorage.getItem(locationKeyStr);
            cachedTimestamp = localStorage.getItem(timestampKey);
            console.log('loadWeatherDataFromCache - checking keys:', { locationKey, dataKey, locationKeyStr, timestampKey, hasData: !!cachedData, hasLocation: !!cachedLocation, hasTimestamp: !!cachedTimestamp });
        } else {
            // Load from current location cache (backward compatibility)
            cachedData = localStorage.getItem('forecastCachedData');
            cachedLocation = localStorage.getItem('forecastCachedLocation');
            cachedTimestamp = localStorage.getItem('forecastCachedTimestamp');
            console.log('loadWeatherDataFromCache - checking default cache:', { hasData: !!cachedData, hasLocation: !!cachedLocation, hasTimestamp: !!cachedTimestamp });
        }
        
        // Check if cache exists - allow empty location string (it's optional for display purposes)
        // The important checks are cachedData and cachedTimestamp
        if (cachedData && cachedTimestamp) {
            // Check in-memory cache first to avoid JSON.parse
            const cacheKey = locationKey || 'default';
            const memoryCacheKey = `${cacheKey}_${cachedTimestamp}`;
            const cachedParsed = appState.cachedWeatherDataByKey.get(memoryCacheKey);
            
            if (cachedParsed) {
                // Use cached parsed data (much faster than JSON.parse)
                return {
                    data: cachedParsed,
                    location: cachedLocation,
                    timestamp: new Date(cachedTimestamp)
                };
            }
            
            // Parse and cache in memory
            const parsedData = JSON.parse(cachedData);
            appState.cachedWeatherDataByKey.set(memoryCacheKey, parsedData);
            
            // Limit memory cache size to prevent memory leaks (keep last 10 entries)
            if (appState.cachedWeatherDataByKey.size > 10) {
                const firstKey = appState.cachedWeatherDataByKey.keys().next().value;
                appState.cachedWeatherDataByKey.delete(firstKey);
            }
            
            return {
                data: parsedData,
                location: cachedLocation,
                timestamp: new Date(cachedTimestamp)
            };
        }
    } catch (error) {
        console.warn('Failed to load weather data from cache:', error);
    }
    return null;
}

// Clear all cached weather data (used when service worker updates)
function clearAllCachedData() {
    try {
        // Clear all location-specific caches
        const keys = Object.keys(localStorage);
        keys.forEach(key => {
            if (key.startsWith('forecastCachedData_') || 
                key.startsWith('forecastCachedLocation_') || 
                key.startsWith('forecastCachedTimestamp_')) {
                localStorage.removeItem(key);
            }
        });
        
        // Clear current location cache
        localStorage.removeItem('forecastCachedData');
        localStorage.removeItem('forecastCachedLocation');
        localStorage.removeItem('forecastCachedTimestamp');
        
        console.log('All cached weather data cleared');
    } catch (error) {
        console.warn('Failed to clear all cached data:', error);
    }
}

function clearLocationCache(locationKey) {
    try {
        if (!locationKey) return;
        localStorage.removeItem(`forecastCachedData_${locationKey}`);
        localStorage.removeItem(`forecastCachedLocation_${locationKey}`);
        localStorage.removeItem(`forecastCachedTimestamp_${locationKey}`);
        
        // Clear in-memory cache entries for this location
        for (const [key, value] of appState.cachedWeatherDataByKey.entries()) {
            if (key.startsWith(`${locationKey}_`)) {
                appState.cachedWeatherDataByKey.delete(key);
            }
        }
    } catch (error) {
        console.warn('Failed to clear location cache:', error);
    }
}

function isCacheStale(cacheTimestamp) {
    if (!cacheTimestamp) return true;
    const now = new Date();
    const diff = now - cacheTimestamp;
    return diff > DATA_STALE_THRESHOLD;
}

// Drawer management functions
function toggleLocationsDrawer() {
    if (!elements.locationsDrawer || !elements.locationsBtn) return;
    
    const isHidden = elements.locationsDrawer.classList.contains('hidden');
    
    if (isHidden) {
        // Open drawer
        elements.locationsDrawer.classList.remove('hidden');
        elements.locationsBtn.textContent = '🌎Locations ▴';
        elements.locationsBtn.classList.add('active');
        renderLocationButtons();
    } else {
        // Close drawer
        elements.locationsDrawer.classList.add('hidden');
        elements.locationsBtn.textContent = '🌎Locations ▾';
        elements.locationsBtn.classList.remove('active');
    }
}

function renderLocationButtons(activeUID = null) {
    if (!elements.locationButtons) return;
    
    const favorites = getFavorites();
    
    if (favorites.length === 0) {
        elements.locationButtons.innerHTML = '';
        return;
    }
    
    // CRITICAL: Ensure all favorites have UIDs - generate missing ones
    let favoritesUpdated = false;
    favorites.forEach(favorite => {
        if (!favorite.uid && favorite.location) {
            const generatedUID = generateLocationUID(favorite.location);
            if (generatedUID) {
                favorite.uid = generatedUID;
                favoritesUpdated = true;
                console.log('Generated UID for favorite in renderLocationButtons:', generatedUID, favorite.name || favorite.searchQuery);
            }
        }
    });
    
    // Save updated favorites if any were modified
    if (favoritesUpdated) {
        localStorage.setItem('forecastFavorites', JSON.stringify(favorites));
    }
    
    // Get current location UID to determine which button should be active
    // Use provided activeUID if available, otherwise try to generate from appState.location
    let currentUID = activeUID;
    if (!currentUID && appState.location) {
        currentUID = generateLocationUID(appState.location);
    }
    
    let html = '';
    favorites.forEach(favorite => {
        // Use customName if available, otherwise fall back to name
        const displayName = favorite.customName || favorite.name;
        const truncatedName = truncateCityName(displayName, 20);
        // Add 'active' class if this favorite matches the current location (check by UID)
        // Also check by key for backward compatibility
        const isActive = currentUID && (favorite.uid === currentUID || favorite.key === currentUID);
        const activeClass = isActive ? ' active' : '';
        // Use UID as the primary identifier in data attribute (should always exist now)
        const uid = favorite.uid || favorite.key; // Fallback to key for old favorites without UID
        html += `<button class="location-btn${activeClass}" data-location-uid="${uid}" data-location-key="${favorite.key || ''}" data-custom-name="${favorite.customName || ''}">${truncatedName}</button>`;
    });
    
    elements.locationButtons.innerHTML = html;
    
    // CRITICAL: If no favorite button is active, ensure star button is disabled
    // This ensures the converse: no favorite selected = star button disabled (can add favorite)
    // Check DOM after insertion to see if any button has the 'active' class
    const hasActiveFavoriteButton = elements.locationButtons.querySelector('.location-btn.active');
    if (!hasActiveFavoriteButton && elements.favoriteBtn) {
        // No favorite button is active, so star button should be disabled
        // Only disable if the current location is not a favorite (double-check)
        const currentIsFavorite = appState.currentLocationKey && isFavorite(appState.currentLocationKey);
        if (!currentIsFavorite) {
            elements.favoriteBtn.classList.remove('active');
            // Clear currentLocationKey since no favorite is selected
            appState.currentLocationKey = null;
        }
    }
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
    
    // Restore tide data dates (lastTide.time and nextTide.time)
    if (weatherData.noaaStation && weatherData.noaaStation.tideData) {
        const tideData = weatherData.noaaStation.tideData;
        if (tideData.lastTide && tideData.lastTide.time) {
            tideData.lastTide.time = new Date(tideData.lastTide.time);
        }
        if (tideData.nextTide && tideData.nextTide.time) {
            tideData.nextTide.time = new Date(tideData.nextTide.time);
        }
    }
    
    return weatherData;
}

// Check if observations data covers a full week (7 days)
function observationsCoverFullWeek(observationsData) {
    if (!observationsData || !Array.isArray(observationsData) || observationsData.length === 0) {
        return false;
    }
    
    // Get unique dates from observations
    const uniqueDates = new Set(observationsData.map(obs => obs.date));
    
    // Check if we have at least 7 unique dates
    if (uniqueDates.size < 7) {
        return false;
    }
    
    // Check if the dates cover the last 7 days
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    const datesArray = Array.from(uniqueDates).map(dateStr => {
        const [year, month, day] = dateStr.split('-').map(Number);
        return new Date(year, month - 1, day);
    }).sort((a, b) => a - b);
    
    // Check if we have data for the last 7 days
    const sevenDaysAgo = new Date(today);
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6); // 6 days ago + today = 7 days
    
    // Check if the oldest date in observations is within the last 7 days
    const oldestDate = datesArray[0];
    if (oldestDate > today) {
        // Future dates shouldn't happen, but handle gracefully
        return false;
    }
    
    // Check if we have at least 7 days of data within the last 7 days
    const recentDates = datesArray.filter(date => date >= sevenDaysAgo);
    return recentDates.length >= 7;
}

// Load cached weather data and display it
function loadCachedWeatherData(locationKey = null, searchQuery = null) {
    try {
        const cache = loadWeatherDataFromCache(locationKey);
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
        
        // CRITICAL: Use the cache timestamp (when data was actually fetched), not current time
        // This must be set BEFORE any other operations to ensure it's preserved
        // The cache timestamp represents when the NWS API was called, which is what we display
        const cacheTimestamp = cache.timestamp instanceof Date ? cache.timestamp : new Date(cache.timestamp);
        // DEFENSIVE: Log if lastFetchTime is being overwritten
        if (appState.lastFetchTime && appState.lastFetchTime !== cacheTimestamp) {
            console.warn('Overwriting lastFetchTime:', appState.lastFetchTime.toISOString(), 'with cache timestamp:', cacheTimestamp.toISOString());
        }
        appState.lastFetchTime = cacheTimestamp;
        console.log('Loaded cache timestamp (NWS fetch time):', cacheTimestamp.toISOString(), 'Age:', Math.round((Date.now() - cacheTimestamp.getTime()) / 1000), 'seconds');
        
        // Ensure current.time matches the cache timestamp (the actual fetch time)
        // This ensures the "Updated:" field shows the correct time, not the current time
        if (restoredWeatherData && restoredWeatherData.current) {
            restoredWeatherData.current.time = cacheTimestamp;
        }
        
        // Restore app state from cache
        appState.weatherData = restoredWeatherData;
        appState.observationsData = cache.data.observationsData || null;
        appState.observationsAvailable = cache.data.observationsAvailable || false;
        
        // Find matching favorite UID (declare at function scope so it's accessible in requestAnimationFrame)
        let matchingFavoriteUID = null;
        
        // Check if cache is stale - if so, trigger background refresh immediately
        // Declare at function scope so it can be used later for observations check
        const cacheIsStale = isCacheStale(cache.timestamp);
        if (cacheIsStale) {
            // Determine the location to refresh
            let locationToRefresh = searchQuery;
            if (!locationToRefresh && restoredWeatherData && restoredWeatherData.location) {
                // Build search query from location object
                if (restoredWeatherData.location.city && restoredWeatherData.location.state) {
                    locationToRefresh = `${restoredWeatherData.location.city}, ${restoredWeatherData.location.state}`;
                }
            }
            if (!locationToRefresh) {
                locationToRefresh = cache.location || 'here';
            }
            
            // Trigger background refresh (non-blocking, won't show loading indicator)
            console.log('Cache is stale, refreshing in background...');
            setTimeout(() => {
                loadWeatherData(locationToRefresh, false, true).catch(error => {
                    console.error('Background refresh failed for cached location:', error);
                });
            }, 100); // Small delay to ensure UI renders first
        }
        
        // If cached data doesn't have NOAA station but we have coordinates, fetch it in background
        // This handles old cached data from before NOAA feature was added
        if (restoredWeatherData && restoredWeatherData.location && !restoredWeatherData.noaaStation) {
            console.log('Cached data missing NOAA station, fetching in background...');
            // Fetch NOAA station in background (non-blocking)
            // fetchNoaaTideStation is available globally from api.js
            // CRITICAL: Preserve the cache timestamp when updating cache with NOAA station data
            const preservedTimestamp = appState.lastFetchTime;
            setTimeout(async () => {
                try {
                    if (typeof fetchNoaaTideStation === 'function') {
                        const noaaStation = await fetchNoaaTideStation(
                            restoredWeatherData.location.lat, 
                            restoredWeatherData.location.lon
                        );
                        if (noaaStation) {
                            // Update weather data with NOAA station
                            restoredWeatherData.noaaStation = noaaStation;
                            appState.weatherData = restoredWeatherData;
                            // Re-render to show NOAA station
                            renderCurrentMode();
                            // Update cache with new data, but PRESERVE the original cache timestamp
                            // Pass the preserved timestamp to ensure it's not updated
                            saveWeatherDataToCache(restoredWeatherData, restoredWeatherData.location, preservedTimestamp);
                        }
                    }
                } catch (error) {
                    console.error('Error fetching NOAA station for cached location:', error);
                }
            }, 500);
        }
        
        // Restore location if available in cached data
        if (restoredWeatherData && restoredWeatherData.location) {
            appState.location = restoredWeatherData.location;
            
            // Update currentLocationKey and find matching favorite UID
            
            if (locationKey) {
                // Fast path: locationKey provided (most common case when loading from favorite)
                appState.currentLocationKey = locationKey;
                
                // If locationKey is UID-based, extract the UID to find the matching favorite
                if (locationKey.startsWith('uid_')) {
                    const uidFromKey = locationKey.replace('uid_', '');
                    const favorite = getFavoriteByUID(uidFromKey);
                    if (favorite && favorite.uid) {
                        matchingFavoriteUID = favorite.uid;
                    }
                } else {
                    // Location key - find matching favorite by key
                    const favorite = getFavoriteByKey(locationKey);
                    if (favorite && favorite.uid) {
                        matchingFavoriteUID = favorite.uid;
                    }
                }
            } else {
                // Slow path: need to find matching favorite (defer to avoid blocking UI)
                // Generate key first as fallback, then try to find matching favorite in background
                appState.currentLocationKey = generateLocationKey(restoredWeatherData.location);
                
                // Try to find matching favorite immediately (for UI update)
                const locationUID = generateLocationUID(restoredWeatherData.location);
                if (locationUID) {
                    const favorite = getFavoriteByUID(locationUID);
                    if (favorite && favorite.uid) {
                        matchingFavoriteUID = favorite.uid;
                        appState.currentLocationKey = `uid_${favorite.uid}`;
                    }
                }
                
                // If not found by UID, defer expensive favorite matching to background
                if (!matchingFavoriteUID) {
                    const deferMatching = (callback) => {
                        if (typeof requestIdleCallback !== 'undefined') {
                            requestIdleCallback(callback, { timeout: 500 });
                        } else {
                            setTimeout(callback, 0);
                        }
                    };
                    
                    deferMatching(() => {
                        // Try to find a matching favorite by comparing location objects
                        // This handles old favorites that might have different key formats
                        const favorites = getFavorites();
                        const matchingFavorite = favorites.find(fav => {
                            if (fav.location && restoredWeatherData.location) {
                                const favCity = (fav.location.city || '').trim().toLowerCase();
                                const favState = (fav.location.state || '').trim().toUpperCase();
                                const currentCity = (restoredWeatherData.location.city || '').trim().toLowerCase();
                                const currentState = (restoredWeatherData.location.state || '').trim().toUpperCase();
                                
                                if (favCity === currentCity && favState === currentState) {
                                    return true;
                                }
                            }
                            return false;
                        });
                        
                        if (matchingFavorite) {
                            appState.currentLocationKey = matchingFavorite.key;
                            // Update UI if needed (but don't re-render everything)
                            updateFavoriteButtonState();
                            renderLocationButtons(matchingFavorite.uid || matchingFavorite.key);
                        }
                    });
                }
            }
            
            const locationText = formatLocationDisplayName(restoredWeatherData.location.city, restoredWeatherData.location.state);
            elements.locationInput.value = locationText;
        } else {
            // Fallback to cached location string
            // Format the location string for display (remove ", US" if present)
            if (cache.location && cache.location !== 'here') {
                const parts = cache.location.split(',').map(s => s.trim());
                if (parts.length >= 2) {
                    const city = parts[0];
                    const state = parts[1];
                    elements.locationInput.value = formatLocationDisplayName(city, state);
                    appState.location = {
                        city: city,
                        state: state,
                        // Use default values for other required fields
                        lat: 0,
                        lon: 0,
                        timeZone: 'America/New_York'
                    };
                } else {
                    elements.locationInput.value = cache.location;
                }
            } else {
                elements.locationInput.value = cache.location || '';
            }
        }
        
        // Check if observations are incomplete and need refresh (deferred to background for performance)
        // Only refresh if cache is not stale (stale cache is already being refreshed above)
        // Note: cacheIsStale was checked above, reuse that value
        // Defer this expensive check to avoid blocking UI during favorite switching
        if (!cacheIsStale && appState.observationsAvailable && appState.observationsData && restoredWeatherData && restoredWeatherData.location) {
            // Use requestIdleCallback if available, otherwise setTimeout
            const deferCheck = (callback) => {
                if (typeof requestIdleCallback !== 'undefined') {
                    requestIdleCallback(callback, { timeout: 1000 });
                } else {
                    setTimeout(callback, 0);
                }
            };
            
            deferCheck(() => {
                const observationsComplete = observationsCoverFullWeek(appState.observationsData);
                if (!observationsComplete) {
                    console.log('Cached observations are incomplete (less than 7 days), refreshing in background...');
                    // Refresh observations in the background
                    // We need pointsData to refresh observations, so we'll need to fetch it
                    // For now, trigger a full refresh which will update observations
                    const locationToRefresh = restoredWeatherData.location.city && restoredWeatherData.location.state
                        ? `${restoredWeatherData.location.city}, ${restoredWeatherData.location.state}`
                        : (cache.location || 'here');
                    // Use a small delay to allow the UI to render first
                    setTimeout(() => {
                        loadWeatherData(locationToRefresh, false, true).catch(error => {
                            console.error('Background observations refresh failed:', error);
                        });
                    }, 100);
                }
            });
        }
        
        // Batch UI state updates using requestAnimationFrame for better performance
        // This allows browser to batch all DOM updates together
        // IMPORTANT: Capture cacheTimestamp here to ensure it's not modified by any async operations
        const preservedCacheTimestamp = appState.lastFetchTime;
        console.log('Preserving cache timestamp for UI updates:', preservedCacheTimestamp.toISOString());
        requestAnimationFrame(() => {
            // CRITICAL: Ensure lastFetchTime is still set to cache timestamp (defensive check)
            // This prevents any accidental overwrites during async operations
            if (preservedCacheTimestamp) {
                if (appState.lastFetchTime !== preservedCacheTimestamp) {
                    console.warn('lastFetchTime was modified during async operations, restoring cache timestamp');
                    console.warn('  Was:', appState.lastFetchTime instanceof Date ? appState.lastFetchTime.toISOString() : appState.lastFetchTime);
                    console.warn('  Restoring to:', preservedCacheTimestamp.toISOString());
                    appState.lastFetchTime = preservedCacheTimestamp;
                } else {
                    console.log('lastFetchTime correctly preserved:', appState.lastFetchTime.toISOString());
                }
            }
            
            // Update History button state
            updateHistoryButtonState();
            
            // Update last update time (this uses appState.lastFetchTime, so it should be correct now)
            updateLastUpdateTime();
            
            // Update favorite button state and location buttons
            // Extract UID from cache key if it's UID-based, or find matching favorite
            let activeIdentifier = matchingFavoriteUID;
            if (!activeIdentifier) {
                if (locationKey && locationKey.startsWith('uid_')) {
                    // Extract UID from UID-based cache key
                    const uidFromKey = locationKey.replace('uid_', '');
                    const favorite = getFavoriteByUID(uidFromKey);
                    if (favorite && favorite.uid) {
                        activeIdentifier = favorite.uid;
                    }
                } else if (locationKey) {
                    // Location key - find matching favorite by key
                    const favorite = getFavoriteByKey(locationKey);
                    if (favorite && favorite.uid) {
                        activeIdentifier = favorite.uid;
                    } else {
                        activeIdentifier = locationKey;
                    }
                } else if (appState.location) {
                    // Try to find favorite by UID first
                    const locationUID = generateLocationUID(appState.location);
                    if (locationUID) {
                        const favorite = getFavoriteByUID(locationUID);
                        if (favorite && favorite.uid) {
                            activeIdentifier = favorite.uid;
                        } else {
                            activeIdentifier = locationUID;
                        }
                    } else {
                        // Fallback to key if no UID match
                        activeIdentifier = generateLocationKey(appState.location);
                    }
                }
            }
            
            updateFavoriteButtonState(activeIdentifier);
            
            // Update location buttons to highlight the active one (use UID if available)
            renderLocationButtons(activeIdentifier);
            
            // Update current location button state
            // Check if this is 'here' or a favorite
            if (searchQuery && searchQuery.toLowerCase() === 'here') {
                appState.isCurrentLocationActive = true;
                updateCurrentLocationButtonState(true);
            } else if (activeIdentifier) {
                // It's a favorite, so current location is not active
                appState.isCurrentLocationActive = false;
                updateCurrentLocationButtonState(false);
            } else {
                // Not 'here' and not a favorite
                appState.isCurrentLocationActive = false;
                updateCurrentLocationButtonState(false);
            }
            
            // Render current mode to display cached data (already uses requestAnimationFrame internally)
            // This will call displayCurrentConditions which uses appState.lastFetchTime
            renderCurrentMode();
            
            // Final defensive check after rendering
            if (preservedCacheTimestamp && appState.lastFetchTime !== preservedCacheTimestamp) {
                console.error('CRITICAL: lastFetchTime was modified during rendering, restoring cache timestamp');
                appState.lastFetchTime = preservedCacheTimestamp;
                // Re-render to show correct timestamp
                updateLastUpdateTime();
            }
        });
        
        return true;
    } catch (error) {
        console.error('Error loading cached weather data:', error);
        // Clear potentially corrupted cache
        try {
            if (locationKey) {
                clearLocationCache(locationKey);
            } else {
                localStorage.removeItem('forecastCachedData');
                localStorage.removeItem('forecastCachedLocation');
                localStorage.removeItem('forecastCachedTimestamp');
            }
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
        
        // Check if we already have fresh data for this location in appState
        // This prevents unnecessary API calls when switching tabs or re-rendering
        if (appState.weatherData && appState.location && appState.lastFetchTime) {
            const locationText = formatLocationDisplayName(appState.location.city, appState.location.state);
            let locationMatch = false;
            
            if (location.toLowerCase() === 'here') {
                // For 'here', only match if current location is actually 'here' (not a favorite)
                // Check if current location is a favorite
                const favorites = getFavorites();
                const currentUID = generateLocationUID(appState.location);
                const isCurrentLocationFavorite = favorites.some(fav => fav.uid === currentUID);
                // Only match 'here' if current location is NOT a favorite and isCurrentLocationActive is true
                locationMatch = !isCurrentLocationFavorite && appState.isCurrentLocationActive && locationText.toLowerCase() !== '';
            } else {
                // For specific locations, require exact match
                locationMatch = location.toLowerCase() === locationText.toLowerCase();
            }
            
            if (locationMatch && !isCacheStale(appState.lastFetchTime)) {
                // We already have fresh data for this location - just render it
                console.log('Using existing fresh data for location:', location);
                setLoading(false, background);
                renderCurrentMode();
                updateFavoriteButtonState();
                renderLocationButtons();
                // Update current location button state
                if (location.toLowerCase() === 'here') {
                    updateCurrentLocationButtonState(true);
                } else {
                    updateCurrentLocationButtonState(false);
                }
                return;
            }
        }
        
        // Check cache first (unless this is a forced refresh)
        // Try to find a matching favorite or cached location by checking:
        // 1. If location matches a favorite, use that favorite's key
        // 2. If location matches the default cached location string
        // 3. Check all location-specific caches to see if any match
        let cacheKeyToUse = null;
        let cacheToUse = null;
        
        // First, try to find a matching favorite
        const favorites = getFavorites();
        const matchingFavorite = favorites.find(fav => {
            if (!fav.searchQuery) return false;
            // Check if location matches the favorite's searchQuery or display name
            const favSearchQuery = fav.searchQuery.toLowerCase().trim();
            const favDisplayName = fav.name ? fav.name.toLowerCase().trim() : '';
            const locationLower = location.toLowerCase().trim();
            return locationLower === favSearchQuery || 
                   locationLower === favDisplayName ||
                   (fav.location && formatLocationDisplayName(fav.location.city, fav.location.state).toLowerCase().trim() === locationLower);
        });
        
        if (matchingFavorite && matchingFavorite.key) {
            // Found a matching favorite - use UID-based cache key (more stable)
            // Fallback to location key for backward compatibility
            cacheKeyToUse = matchingFavorite.uid ? `uid_${matchingFavorite.uid}` : matchingFavorite.key;
            console.log('Found matching favorite for location:', location, 'UID:', matchingFavorite.uid, 'cacheKey:', cacheKeyToUse);
            cacheToUse = loadWeatherDataFromCache(cacheKeyToUse);
            console.log('Cache lookup result for key', cacheKeyToUse, ':', cacheToUse ? 'FOUND' : 'NOT FOUND');
            if (!cacheToUse) {
                // Cache might not exist for this key - check if cache exists in localStorage
                const cacheDataKey = `forecastCachedData_${cacheKeyToUse}`;
                const cacheLocationKey = `forecastCachedLocation_${cacheKeyToUse}`;
                const cacheTimestampKey = `forecastCachedTimestamp_${cacheKeyToUse}`;
                const hasData = localStorage.getItem(cacheDataKey) !== null;
                const hasLocation = localStorage.getItem(cacheLocationKey) !== null;
                const hasTimestamp = localStorage.getItem(cacheTimestampKey) !== null;
                console.log('Cache keys check:', { hasData, hasLocation, hasTimestamp, cacheDataKey, cacheLocationKey, cacheTimestampKey });
            }
        } else {
            // No matching favorite - check default cache
            // For 'here', skip cache and always fetch fresh (location is dynamic)
            if (location.toLowerCase() === 'here') {
                console.log('Location is "here" - skipping cache and fetching fresh data for current location');
                cacheToUse = null; // Don't use cache for 'here'
                cacheKeyToUse = null;
            } else {
                const cachedLocation = localStorage.getItem('forecastCachedLocation');
                if (cachedLocation && cachedLocation.toLowerCase() === location.toLowerCase()) {
                    cacheToUse = loadWeatherDataFromCache();
                    console.log('Location matches default cached location:', cachedLocation);
                } else {
                    // Check all location-specific caches to see if any match
                    // This handles cases where location string doesn't match exactly
                    const keys = Object.keys(localStorage);
                    for (const key of keys) {
                        if (key.startsWith('forecastCachedLocation_')) {
                            const locationKey = key.replace('forecastCachedLocation_', '');
                            const cachedLoc = localStorage.getItem(key);
                            if (cachedLoc && cachedLoc.toLowerCase() === location.toLowerCase()) {
                                cacheKeyToUse = locationKey;
                                cacheToUse = loadWeatherDataFromCache(locationKey);
                                console.log('Found matching location-specific cache for location:', location, 'key:', locationKey);
                                break;
                            }
                        }
                    }
                }
            }
        }
        
        // If we found a cache, check if it's fresh
        if (cacheToUse && cacheToUse.data) {
            const cacheTimestamp = cacheToUse.timestamp instanceof Date ? cacheToUse.timestamp : new Date(cacheToUse.timestamp);
            const cacheAge = Date.now() - cacheTimestamp.getTime();
            const isStale = isCacheStale(cacheToUse.timestamp);
            console.log('Cache found - key:', cacheKeyToUse || 'default', 'timestamp:', cacheTimestamp.toISOString(), 'age:', Math.round(cacheAge / 1000), 'seconds, isStale:', isStale);
            
            if (!isStale) {
                // Cache is fresh - use it instead of fetching
                console.log('Using fresh cached data for location:', location, 'cache timestamp:', cacheTimestamp.toISOString());
                // CRITICAL: Preserve the cache timestamp BEFORE loading cached data
                // This ensures appState.lastFetchTime reflects the actual NWS fetch time
                const preservedCacheTimestamp = cacheTimestamp;
                const cacheLoaded = loadCachedWeatherData(cacheKeyToUse, location);
                if (cacheLoaded) {
                    // DEFENSIVE: Ensure lastFetchTime is set to the cache timestamp
                    // This prevents any accidental overwrites during async operations
                    if (appState.lastFetchTime !== preservedCacheTimestamp) {
                        console.warn('lastFetchTime was modified during loadCachedWeatherData, restoring cache timestamp');
                        appState.lastFetchTime = preservedCacheTimestamp;
                    }
                    console.log('Cache loaded successfully, lastFetchTime set to:', appState.lastFetchTime.toISOString(), 'Age:', Math.round((Date.now() - appState.lastFetchTime.getTime()) / 1000), 'seconds');
                    
                    // Update current location button state based on location parameter
                    if (location.toLowerCase() === 'here') {
                        appState.isCurrentLocationActive = true;
                        updateCurrentLocationButtonState(true);
                    } else {
                        // Check if this is a favorite - if so, current location is not active
                        const favorites = getFavorites();
                        const isFavorite = favorites.some(fav => {
                            if (fav.location && appState.location) {
                                const favCity = (fav.location.city || '').trim().toLowerCase();
                                const favState = (fav.location.state || '').trim().toUpperCase();
                                const currentCity = (appState.location.city || '').trim().toLowerCase();
                                const currentState = (appState.location.state || '').trim().toUpperCase();
                                return favCity === currentCity && favState === currentState;
                            }
                            return false;
                        });
                        appState.isCurrentLocationActive = false;
                        updateCurrentLocationButtonState(false);
                    }
                    
                    setLoading(false, background);
                    
                    // Ensure the display is updated (loadCachedWeatherData renders in requestAnimationFrame,
                    // but we also call it here to ensure immediate update)
                    renderCurrentMode();
                    updateFavoriteButtonState();
                    renderLocationButtons();
                    // Still trigger background refresh if cache is getting close to stale
                    const refreshThreshold = DATA_STALE_THRESHOLD * 0.8; // Refresh at 80% of stale threshold
                    if (cacheAge > refreshThreshold) {
                        console.log('Cache is getting close to stale, refreshing in background...');
                        // CRITICAL: Preserve timestamp before background refresh
                        // The background refresh should NOT overwrite the displayed timestamp until it completes
                        const timestampBeforeRefresh = appState.lastFetchTime;
                        setTimeout(() => {
                            loadWeatherData(location, false, true).then(() => {
                                // Background refresh completed - timestamp will be updated with new fetch time
                                console.log('Background refresh completed, timestamp updated to:', appState.lastFetchTime.toISOString());
                            }).catch(error => {
                                // On error, restore the original timestamp
                                console.error('Background refresh failed:', error);
                                if (appState.lastFetchTime !== timestampBeforeRefresh) {
                                    appState.lastFetchTime = timestampBeforeRefresh;
                                    updateLastUpdateTime();
                                }
                            });
                        }, 100);
                    }
                    return;
                } else {
                    console.warn('loadCachedWeatherData returned false, will fetch fresh data');
                }
            } else {
                console.log('Cache is stale, will fetch fresh data');
            }
        } else {
            console.log('No cache found for location:', location, 'will fetch fresh data');
        }
        
        // Ensure fetchWeatherData is available (from api.js)
        if (typeof fetchWeatherData === 'undefined') {
            throw new Error('fetchWeatherData function not found. Please ensure api.js is loaded.');
        }
        
        const weatherData = await fetchWeatherData(location);
        const processedWeather = processWeatherData(weatherData);
        
        appState.weatherData = processedWeather;
        appState.location = weatherData.location;
        
        // Update current location button state based on whether this is 'here' or not
        if (location.toLowerCase() === 'here') {
            appState.isCurrentLocationActive = true;
            updateCurrentLocationButtonState(true);
        } else {
            // Check if this is a favorite - if so, current location is not active
            const favorites = getFavorites();
            const isFavorite = favorites.some(fav => {
                if (fav.location && weatherData.location) {
                    const favCity = (fav.location.city || '').trim().toLowerCase();
                    const favState = (fav.location.state || '').trim().toUpperCase();
                    const currentCity = (weatherData.location.city || '').trim().toLowerCase();
                    const currentState = (weatherData.location.state || '').trim().toUpperCase();
                    return favCity === currentCity && favState === currentState;
                }
                return false;
            });
            if (!isFavorite) {
                // Not a favorite and not 'here' - check if it matches lastViewed
                const lastViewed = getLastViewedLocation();
                if (lastViewed && lastViewed.searchQuery && lastViewed.searchQuery.toLowerCase() === 'here') {
                    // Last viewed was 'here', so this is a different location
                    appState.isCurrentLocationActive = false;
                    updateCurrentLocationButtonState(false);
                } else {
                    // Regular location (not favorite, not 'here')
                    appState.isCurrentLocationActive = false;
                    updateCurrentLocationButtonState(false);
                }
            } else {
                // It's a favorite, so current location is not active
                appState.isCurrentLocationActive = false;
                updateCurrentLocationButtonState(false);
            }
        }
        
        // Update currentLocationKey from the loaded location
        // CRITICAL: We need to find the matching favorite using multiple methods:
        // 1. Check if generated key matches a favorite's key
        // 2. Check if location matches a favorite's location object (city + state)
        // 3. Check if the search query (location parameter) matches a favorite's searchQuery
        // This ensures we use the favorite's key even if the state differs (e.g., US vs AK)
        let generatedKey = generateLocationKey(weatherData.location);
        let matchingFavoriteAfterFetch = null;
        
        // First, check if generated key matches a favorite's key
        if (generatedKey && isFavorite(generatedKey)) {
            matchingFavoriteAfterFetch = getFavoriteByKey(generatedKey);
        }
        
        // If not found, try to find by location object (city + state match)
        if (!matchingFavoriteAfterFetch) {
            const favorites = getFavorites();
            matchingFavoriteAfterFetch = favorites.find(fav => {
                if (fav.location && weatherData.location) {
                    const favCity = (fav.location.city || '').trim().toLowerCase();
                    const favState = (fav.location.state || '').trim().toUpperCase();
                    const currentCity = (weatherData.location.city || '').trim().toLowerCase();
                    const currentState = (weatherData.location.state || '').trim().toUpperCase();
                    
                    if (favCity === currentCity && favState === currentState) {
                        return true;
                    }
                }
                return false;
            });
        }
        
        // If still not found, try to find by searchQuery (the location parameter we passed)
        // This handles cases where the favorite was created with a different state (US vs AK)
        if (!matchingFavoriteAfterFetch && location) {
            const favorites = getFavorites();
            const locationLower = location.toLowerCase().trim();
            matchingFavoriteAfterFetch = favorites.find(fav => {
                if (!fav.searchQuery) return false;
                const favSearchQuery = fav.searchQuery.toLowerCase().trim();
                const favDisplayName = fav.name ? fav.name.toLowerCase().trim() : '';
                return locationLower === favSearchQuery || 
                       locationLower === favDisplayName ||
                       (fav.location && formatLocationDisplayName(fav.location.city, fav.location.state).toLowerCase().trim() === locationLower);
            });
        }
        
        // Use the favorite's UID for cache key if found (more stable than location key)
        // Otherwise use generated key
        if (matchingFavoriteAfterFetch && matchingFavoriteAfterFetch.uid) {
            // Use UID-based cache key for favorites (more stable, avoids US vs AK issues)
            appState.currentLocationKey = `uid_${matchingFavoriteAfterFetch.uid}`;
            console.log('Found matching favorite after fetch, using UID-based cache key:', appState.currentLocationKey, 'instead of generated key:', generatedKey);
        } else if (matchingFavoriteAfterFetch && matchingFavoriteAfterFetch.key) {
            // Fallback to location key if no UID
            appState.currentLocationKey = matchingFavoriteAfterFetch.key;
            console.log('Found matching favorite after fetch, using location key:', matchingFavoriteAfterFetch.key, 'instead of generated key:', generatedKey);
        } else {
            appState.currentLocationKey = generatedKey;
        }
        
        // Use the actual NWS API fetch time (from weatherData.fetchTime) as the cache timestamp
        // This ensures the "Updated:" field reflects when the NWS data was actually fetched
        // weatherData.fetchTime is set at the START of NWS API calls in fetchWeatherData()
        const nwsFetchTime = weatherData.fetchTime || new Date();
        // DEFENSIVE: Log if lastFetchTime is being overwritten (only if it was set from cache)
        if (appState.lastFetchTime && appState.lastFetchTime !== nwsFetchTime) {
            const previousTime = appState.lastFetchTime instanceof Date ? appState.lastFetchTime.toISOString() : appState.lastFetchTime;
            console.log('Updating lastFetchTime from cache:', previousTime, 'to NWS fetch time:', nwsFetchTime.toISOString());
        }
        appState.lastFetchTime = nwsFetchTime;
        console.log('Set lastFetchTime from NWS API fetch:', nwsFetchTime.toISOString());
        appState.hourlyScrollIndex = 0;
        
        // Update location in input field
        const locationText = formatLocationDisplayName(weatherData.location.city, weatherData.location.state);
        elements.locationInput.value = locationText;
        
        // Update URL if location changed (preserve mode)
        if (location && location.toLowerCase() !== 'here') {
            updateURL(location, appState.currentMode);
        }
        
        // Update last update time
        updateLastUpdateTime();
        
        // Save to cache after successful fetch (pass location object and NWS fetch time)
        // The timestamp parameter ensures the cache timestamp reflects when NWS data was fetched
        saveWeatherDataToCache(processedWeather, weatherData.location, nwsFetchTime);
        
        // Save as last viewed location
        const searchQuery = location && location.toLowerCase() !== 'here' ? location : locationText;
        saveLastViewedLocation(locationText, weatherData.location, searchQuery);
        
        // Update favorite button state and location buttons
        // Use the favorite's UID if we found a matching favorite (stored in appState.currentLocationKey)
        let activeIdentifier = null;
        if (appState.currentLocationKey && appState.currentLocationKey.startsWith('uid_')) {
            // Extract UID from UID-based cache key
            const uidFromKey = appState.currentLocationKey.replace('uid_', '');
            const favorite = getFavoriteByUID(uidFromKey);
            if (favorite && favorite.uid) {
                activeIdentifier = favorite.uid;
            }
        } else if (appState.currentLocationKey) {
            // Location key - find matching favorite by key
            const favorite = getFavoriteByKey(appState.currentLocationKey);
            if (favorite && favorite.uid) {
                activeIdentifier = favorite.uid;
            } else {
                activeIdentifier = appState.currentLocationKey;
            }
        } else if (appState.location) {
            // Try to find favorite by UID first
            const locationUID = generateLocationUID(appState.location);
            if (locationUID) {
                const favorite = getFavoriteByUID(locationUID);
                if (favorite && favorite.uid) {
                    activeIdentifier = favorite.uid;
                } else {
                    activeIdentifier = locationUID;
                }
            } else {
                // Fallback to key if no UID match
                activeIdentifier = generateLocationKey(appState.location);
            }
        }
        
        updateFavoriteButtonState(activeIdentifier);
        
        // Update location buttons to highlight the active one (use UID if available)
        renderLocationButtons(activeIdentifier);
        
        // Update current location button state (already set earlier in function based on location parameter)
        // This ensures the button state is correct after all processing
        if (location.toLowerCase() === 'here') {
            updateCurrentLocationButtonState(true);
        } else if (activeIdentifier) {
            // It's a favorite, so current location is not active
            updateCurrentLocationButtonState(false);
        } else {
            // Not 'here' and not a favorite - current location is not active
            updateCurrentLocationButtonState(false);
        }
        
        // Render current mode immediately (don't wait for observations)
        renderCurrentMode();
        
        setLoading(false, background);
        
        // Check observations availability and fetch observations in background (non-blocking)
        // This allows the UI to display immediately while observations load
        checkObservationsAvailability(weatherData.points, weatherData.location.timeZone).then(() => {
            // Observations completed - update History button state
            updateHistoryButtonState();
            // If we're currently viewing history mode, re-render to show the observations
            if (appState.currentMode === 'history') {
                renderCurrentMode();
            }
        }).catch(error => {
            console.error('Error fetching observations in background:', error);
            // Still update button state even on error
            updateHistoryButtonState();
        });
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
// Marks all 10-minute cache items (weather data, observations) as stale and forces reload
// Preserves 1-week cache items (stations.json, water level support, tide predictions)
async function handleRefresh() {
    console.log('Refresh button clicked - loading cached data first, then refreshing');
    
    // Get current location
    const location = elements.locationInput.value.trim() || 'here';
    
    // First, try to load cached data (even if stale) to display immediately
    // This ensures observations data is shown while fresh data is being fetched
    // Use the same cache lookup logic as loadWeatherData
    let cacheKeyToUse = null;
    let cacheToUse = null;
    
    const favorites = getFavorites();
    const matchingFavorite = favorites.find(fav => {
        if (!fav.searchQuery) return false;
        const favSearchQuery = fav.searchQuery.toLowerCase().trim();
        const favDisplayName = fav.name ? fav.name.toLowerCase().trim() : '';
        const locationLower = location.toLowerCase().trim();
        return locationLower === favSearchQuery || 
               locationLower === favDisplayName ||
               (fav.location && formatLocationDisplayName && formatLocationDisplayName(fav.location.city, fav.location.state).toLowerCase().trim() === locationLower);
    });
    
    if (matchingFavorite && matchingFavorite.key) {
        cacheKeyToUse = matchingFavorite.uid ? `uid_${matchingFavorite.uid}` : matchingFavorite.key;
        cacheToUse = loadWeatherDataFromCache(cacheKeyToUse);
    } else {
        // Check default cache
        const cachedLocation = localStorage.getItem('forecastCachedLocation');
        if (cachedLocation && cachedLocation.toLowerCase() === location.toLowerCase()) {
            cacheToUse = loadWeatherDataFromCache();
        }
    }
    
    // Try to load cached data first (preserves observations data)
    if (cacheToUse && cacheToUse.data && cacheToUse.data.weatherData) {
        console.log('Loading cached data first to preserve observations during refresh');
        const cachedLoaded = loadCachedWeatherData(cacheKeyToUse, location);
        if (cachedLoaded) {
            // Render immediately with cached data (including observations)
            renderCurrentMode();
            setLoading(false, false);
        }
    }
    
    // Mark all weather data cache timestamps as stale (10 minutes ago)
    // This forces a reload of weather data, observations, etc.
    const staleTimestamp = new Date(Date.now() - DATA_STALE_THRESHOLD - 1000); // 1 second past stale threshold
    
    try {
        // Get all localStorage keys
        const keys = Object.keys(localStorage);
        
        // Mark weather data cache timestamps as stale
        keys.forEach(key => {
            // Mark location-specific weather cache timestamps
            if (key.startsWith('forecastCachedTimestamp_')) {
                localStorage.setItem(key, staleTimestamp.toISOString());
                console.log('Marked cache as stale:', key);
            }
        });
        
        // Mark default weather cache timestamp
        if (localStorage.getItem('forecastCachedTimestamp')) {
            localStorage.setItem('forecastCachedTimestamp', staleTimestamp.toISOString());
            console.log('Marked default cache as stale');
        }
        
        // Clear in-memory cache to force reload from localStorage (which now has stale timestamps)
        appState.cachedWeatherDataByKey.clear();
        
        // Mark appState.lastFetchTime as stale so loadWeatherData will fetch fresh data
        if (appState.lastFetchTime) {
            appState.lastFetchTime = staleTimestamp;
        }
        
        console.log('Fetching fresh data for location:', location);
        
        // Now fetch fresh data (this will update the display when it completes)
        await loadWeatherData(location, false, false); // Don't use background mode for manual refresh - show loading
    } catch (error) {
        console.error('Error during refresh:', error);
        // If refresh fails, cached data should still be displayed (loaded above)
        // Just update the UI to show we're not loading anymore
        setLoading(false, false);
    }
}

// Switch display mode
function switchMode(mode) {
    appState.currentMode = mode;
    
    // Save mode to localStorage for PWA state persistence
    try {
        localStorage.setItem('forecastCurrentMode', mode);
    } catch (error) {
        console.warn('Failed to save current mode:', error);
    }
    
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

// Render current mode (with requestAnimationFrame batching for performance)
let renderCurrentModeScheduled = false;
let renderCurrentModePending = false;

function renderCurrentMode() {
    if (!appState.weatherData || !appState.location) {
        return;
    }
    
    // If already scheduled, mark as pending and return (prevents multiple renders)
    if (renderCurrentModeScheduled) {
        renderCurrentModePending = true;
        return;
    }
    
    // Schedule render in next animation frame for batching
    renderCurrentModeScheduled = true;
    requestAnimationFrame(() => {
        renderCurrentModeScheduled = false;
        const wasPending = renderCurrentModePending;
        renderCurrentModePending = false;
        
        // If another render was requested while we were waiting, skip this one
        if (wasPending) {
            renderCurrentMode();
            return;
        }
        
        // Perform the actual render
        _renderCurrentModeImpl();
    });
}

// Internal implementation of render (called from requestAnimationFrame)
function _renderCurrentModeImpl() {
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
    // Deferred to requestIdleCallback for better performance - not critical for initial render
    if (appState.currentMode === 'hourly') {
        if (typeof requestIdleCallback !== 'undefined') {
            requestIdleCallback(() => {
                setupHourlyNavigation();
            }, { timeout: 500 });
        } else {
            // Fallback for browsers without requestIdleCallback
            setTimeout(() => {
                setupHourlyNavigation();
            }, 0);
        }
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
        
        // Also update the "[X minutes ago]" portion in the display
        const updatedTimestampElement = document.querySelector('.updated-timestamp');
        if (updatedTimestampElement) {
            // Check if data is stale (>10 minutes)
            const now = new Date();
            const diff = now - appState.lastFetchTime;
            const isStale = diff > DATA_STALE_THRESHOLD;
            
            updatedTimestampElement.textContent = `[${timeAgo}]`;
            if (isStale) {
                updatedTimestampElement.classList.add('stale-data');
            } else {
                updatedTimestampElement.classList.remove('stale-data');
            }
        }
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

