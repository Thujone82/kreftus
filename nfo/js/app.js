console.log("app.js loaded");

const APP_CONSTANTS = {
    CACHE_EXPIRY_MS: 60 * 60 * 1000, // 1 hour
    MODAL_IDS: {
        APP_CONFIG: 'appConfigModal',
        LOCATION_CONFIG: 'locationConfigModal',
        INFO_COLLECTION_CONFIG: 'infoCollectionConfigModal',
        INFO: 'infoModal'
    },
     CACHE_EXPIRY_WEATHER_MS: 20 * 60 * 1000, // 20 minutes for weather
     CACHE_EXPIRY_GEOCODE_MS: 7 * 24 * 60 * 60 * 1000, // 7 days for geocode cache
     SW_MESSAGES: {
        SKIP_WAITING: 'SKIP_WAITING'
    }
};

const app = {
    config: { // Initialize config as an object
        apiKey: null, // For Gemini
        owmApiKey: null, // For OpenWeatherMap
        rpmLimit: 10,  // Default RPM limit for Gemini
        primaryColor: '#029ec5',
        backgroundColor: '#1E1E1E'
    },
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    editingTopicId: null,
    currentLocationIdForInfoModal: null,
    fetchingStatus: {},
    initialEditingLocationsString: '', // For unsaved changes check
    initialEditingTopicsString: '',    // For unsaved changes check
    activeLoadingOperations: 0,        // Counter for global loading state
    isRefreshingAllStale: false,       // Flag to prevent overlapping global refreshes
    userInitiatedUpdate: false,        // Flag for SW update
    newWorkerForUpdate: null,          // Store the waiting worker

    init: async () => {
        console.log("App initializing...");
        // Synchronous setup first
        app.loadAndApplyAppSettings(); // This will populate app.config
        app.loadLocations();
        app.loadTopics();
        app.registerServiceWorker();
        app.setupEventListeners();
        // Ensure connectivity check completes before proceeding
        await app.testConnectivityAndSetFlag();

        const areTopicsDefined = app.topics && app.topics.length > 0;
        
        // EARLY EXIT: Handle OFFLINE state
        if (!window.isActuallyOnline) {
            console.log("App is OFFLINE. Initializing in offline mode.");
            ui.showOfflineIndicator(true); // Show the offline banner
            ui.toggleConfigButtons(true); // Allow config access

            // Render buttons immediately from cache, weather will be pulled from cache by the render function
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            
            app.updateGlobalRefreshButtonVisibility();
            console.log("App initialized in offline mode.");
            return; // Stop further initialization
        }

        // --- ONLINE INITIALIZATION ---
        // The rest of this function ONLY runs if we are online.
        console.log("App is ONLINE. Proceeding with online initialization.");
        ui.showOfflineIndicator(false); // Ensure offline banner is hidden

        // Check for Gemini API key presence
        if (!app.config.apiKey) {
            ui.toggleConfigButtons(false);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            if (ui.appConfigError) ui.appConfigError.textContent = "Gemini API Key is required for core functionality.";
            return; // Stop initialization
        }

        // Re-validate to ensure the key itself is valid, not just that we're online
        const isKeyValid = await app.validateAndDisplayGeminiKeyStatus(app.config.apiKey);
        if (!isKeyValid) {
            ui.toggleConfigButtons(false);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            if (ui.appConfigError) ui.appConfigError.textContent = "Gemini API Key is invalid. Please update it.";
            return; // Stop initialization
        }

        // Key is present and valid, enable config buttons that depend on it
        ui.toggleConfigButtons(true);

        // Now handle weather features if OWM key exists
        if (app.config.owmApiKey) {
            console.log('OpenWeatherMap API Key is set. Enabling weather features.');
            const performWeatherRefresh = async () => {
                app.incrementActiveLoaders();
                try {
                    await app.refreshOutdatedWeather();
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    await app.updateOpenInfoModalWeather();
                } catch (error) {
                    console.error("Error during weather refresh:", error);
                } finally {
                    app.decrementActiveLoaders();
                }
            };
            await performWeatherRefresh(); // Initial refresh

            // Set up periodic refresh
            setInterval(() => { if (window.isActuallyOnline && app.config.owmApiKey) performWeatherRefresh(); }, APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS);

            // Refresh weather when app becomes visible
            document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible' && window.isActuallyOnline && app.config.owmApiKey) performWeatherRefresh(); });
        } else {
            console.log('OpenWeatherMap API Key is NOT set. Weather features disabled.');
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        }

        // Final UI updates
        app.updateGlobalRefreshButtonVisibility();
        console.log("App initialized.");
    },
     // Initialize the "actually online" flag *after* settings are loaded
    testConnectivityAndSetFlag: async () => {
       window.isActuallyOnline = false; // Default to offline
       if (app.config.apiKey) {
            // The result object from validate... is { isValid: boolean, reason: string }
            const validationResult = await api.validateGeminiApiKey(app.config.apiKey);
            // We are "online" if the request didn't fail due to a network error.
            // An invalid key still means we successfully reached the service.
            window.isActuallyOnline = validationResult.reason !== 'network_error';
       }
       console.log(`App startup: window.isActuallyOnline initialized to: ${window.isActuallyOnline}`);
    },

   registerServiceWorker: () => {
     if ('serviceWorker' in navigator) {
       navigator.serviceWorker.register('/nfo/sw.js', { scope: '/nfo/' })
         .then(registration => {
                    console.log('Service Worker registered with scope:', registration.scope);
                    if (registration.waiting) {
                        console.log('SW Registration: Found a waiting SW immediately. Prompting user.');
                        app.promptUserToUpdate(registration.waiting);
                    }
                    registration.onupdatefound = () => {
                        console.log('SW Registration: New service worker found installing.');
                        const installingWorker = registration.installing;
                        if (installingWorker) {
                            installingWorker.onstatechange = () => {
                                console.log('SW Registration: Installing worker state changed:', installingWorker.state);
                                if (installingWorker.state === 'installed') {
                                    if (navigator.serviceWorker.controller) {
                                        console.log('SW Registration: New SW installed and waiting (controller exists). Prompting user.');
                                        app.promptUserToUpdate(installingWorker);
                                    } else {
                                        console.log('SW Registration: SW installed. No active controller. Will activate on next load or if claimed.');
                                    }
                                } else if (installingWorker.state === 'redundant') {
                                    console.error('SW Registration: The installing service worker became redundant.');
                                }
                            };
                        }
                    };
                })
                .catch(error => console.error('Service Worker registration failed:', error));
        }
        app.listenForControllerChange();
    },

    loadAndApplyAppSettings: () => {
        const storedSettings = store.getAppSettings();
        app.config = {
            apiKey: storedSettings.apiKey || null,
            owmApiKey: storedSettings.owmApiKey || null,
            rpmLimit: storedSettings.rpmLimit || 10,
            primaryColor: storedSettings.primaryColor || '#029ec5',
            backgroundColor: storedSettings.backgroundColor || '#1E1E1E'
        };

        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.loadAppConfigForm(app.config);
        console.log("App settings loaded and applied:", app.config);
    },

    loadLocations: () => {
        app.locations = store.getLocations();
        console.log("Locations loaded:", app.locations);
    },
    loadTopics: () => {
        app.topics = store.getTopics();
        console.log("Topics loaded:", app.topics);
        // The call to renderLocationButtons is now handled in init() after connectivity checks
    },

    setupEventListeners: () => {
        ui.btnAppConfig.onclick = () => {
            ui.loadAppConfigForm(app.config);
            if (ui.appConfigError) ui.appConfigError.textContent = '';
            // Validate keys when modal is opened if they exist
            if (app.config.apiKey) {
                app.validateAndDisplayGeminiKeyStatus(app.config.apiKey, true);
            } else {
                ui.setApiKeyStatus('gemini', 'checking', 'Key Test');
            }
            if (app.config.owmApiKey) {
                app.validateAndDisplayOwmKeyStatus(app.config.owmApiKey, true);
            } else {
                ui.setApiKeyStatus('owm', 'checking', 'Key Test');
            }
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
        };
        ui.btnLocationsConfig.onclick = () => {
            app.currentEditingLocations = JSON.parse(JSON.stringify(app.locations));
            app.initialEditingLocationsString = JSON.stringify(app.currentEditingLocations);
            if (ui.locationConfigError) ui.locationConfigError.textContent = '';
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG);
        };
        ui.btnInfoCollectionConfig.onclick = () => {
            app.currentEditingTopics = JSON.parse(JSON.stringify(app.topics));
            app.initialEditingTopicsString = JSON.stringify(app.currentEditingTopics);
            app.editingTopicId = null;
            ui.addTopicBtn.textContent = 'Add Topic';
            if (ui.topicConfigError) ui.topicConfigError.textContent = '';
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG);
        };

        ui.saveAppConfigBtn.onclick = app.handleSaveAppSettings;
        ui.addLocationBtn.onclick = app.handleAddLocationToEditList;
        ui.saveLocationConfigBtn.onclick = app.handleSaveLocationConfig;
        ui.enableDragAndDrop(ui.locationsListUI, (newOrderIds) => {
            app.currentEditingLocations = app.reorderArrayByIds(app.currentEditingLocations, newOrderIds);
        });

        ui.addTopicBtn.onclick = app.handleAddOrUpdateTopicInEditList;
        ui.saveTopicConfigBtn.onclick = app.handleSaveTopicConfig;
        ui.enableDragAndDrop(ui.topicsListUI, (newOrderIds) => {
            app.currentEditingTopics = app.reorderArrayByIds(app.currentEditingTopics, newOrderIds);
        });

        ui.refreshInfoButton.onclick = () => {
            const locationId = ui.refreshInfoButton.dataset.locationId || app.currentLocationIdForInfoModal;
            if (locationId) app.handleRefreshLocationInfo(locationId);
        };
        if (ui.globalRefreshButton) {
            ui.globalRefreshButton.onclick = () => {
                if (!window.isActuallyOnline) {
                    console.warn("Global refresh clicked but app is offline. Aborting.");
                    ui.showOfflineIndicator(true);
                    return;
                }
                console.log("Global refresh triggered.");
                app.refreshOutdatedQueries(false); // Pass false for normal stale check
            };
        }
        if (ui.btnAppUpdate) { // Setup listener for the new update button
            ui.btnAppUpdate.onclick = () => {
                console.log('App Update Button: Clicked.');
                app.triggerUpdate();
            };
        }
        // Show/hide API keys on focus/blur for easier editing
        if (ui.apiKeyInput) {
            ui.apiKeyInput.addEventListener('focus', (e) => { e.target.type = 'text'; });
            ui.apiKeyInput.addEventListener('blur', (e) => { e.target.type = 'password'; });
        }
        if (ui.owmApiKeyInput) {
            ui.owmApiKeyInput.addEventListener('focus', (e) => { e.target.type = 'text'; });
            ui.owmApiKeyInput.addEventListener('blur', (e) => { e.target.type = 'password'; });
        }
         // Add listeners for online/offline events to update the UI
        window.addEventListener('online', async () => {
            console.log("Browser reports online. Re-testing connectivity...");
            await app.testConnectivityAndSetFlag();
            ui.showOfflineIndicator(!window.isActuallyOnline);
            if (window.isActuallyOnline) {
                // Optionally trigger a refresh or re-check things
                console.log("Connectivity re-established. Re-checking outdated items.");
                app.refreshOutdatedQueries(false);
            }
        });
        window.addEventListener('offline', () => {
            console.log("Browser reports offline.");
            window.isActuallyOnline = false;
            const areTopicsDefined = app.topics && app.topics.length > 0;
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            app.updateGlobalRefreshButtonVisibility();
            ui.showOfflineIndicator(true);
        });

        console.log("Event listeners set up.");
    },

    reorderArrayByIds: (originalArray, idOrderArray) => {
        const itemMap = new Map(originalArray.map(item => [item.id, item]));
        return idOrderArray.map(id => itemMap.get(id)).filter(item => item !== undefined);
    },
    handleSaveAppSettings: () => {
        const newGeminiApiKey = ui.apiKeyInput.value.trim();
        const newOwmApiKey = ui.owmApiKeyInput.value.trim();
        const newRpmLimit = parseInt(ui.rpmLimitInput.value, 10) || 10;
        const newPrimaryColor = ui.primaryColorInput.value;
        const newBackgroundColor = ui.backgroundColorInput.value;

        if (!newGeminiApiKey) {
            if(ui.appConfigError) ui.appConfigError.textContent = "Gemini API Key is required.";
            if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.remove('hidden');
            return;
        }
        if(ui.appConfigError) ui.appConfigError.textContent = "";
        if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.add('hidden');

        const hadOwmKeyBefore = !!app.config.owmApiKey;

        app.config.apiKey = newGeminiApiKey;
        app.config.owmApiKey = newOwmApiKey;
        app.config.rpmLimit = newRpmLimit;
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config);
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.toggleConfigButtons(true);

        if (ui.getOwmApiKeyLinkContainer) {
            ui.getOwmApiKeyLinkContainer.classList.toggle('hidden', !!newOwmApiKey);
        }

        ui.closeModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
        console.log("App settings saved:", app.config);
        app.updateGlobalRefreshButtonVisibility();

        // If OWM key was just added, refresh weather and UI
        const hasOwmKeyNow = !!app.config.owmApiKey;
        if (hasOwmKeyNow && !hadOwmKeyBefore) {
            console.log('OpenWeatherMap API Key was added. Triggering weather and UI refresh.');
            app.incrementActiveLoaders();
            app.refreshOutdatedWeather()
                .then(() => {
                    const areTopicsDefined = app.topics && app.topics.length > 0;
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    app.updateOpenInfoModalWeather();
                })
                .catch(error => console.error("Error refreshing weather after OWM key save:", error))
                .finally(() => app.decrementActiveLoaders());
        }
    },

    handleAddLocationToEditList: () => {
        const description = ui.locationDescriptionInput.value.trim();
        const location = ui.locationValueInput.value.trim();
        if (!description || !location) {
            ui.locationConfigError.textContent = 'Both description and location value are required.';
            return;
        }
        ui.locationConfigError.textContent = '';
        const newLocation = { id: utils.generateId(), description, location };
        app.currentEditingLocations.push(newLocation);
        ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
        ui.clearInputFields([ui.locationDescriptionInput, ui.locationValueInput]);
    },

    handleRemoveLocationFromEditList: (idToRemove) => {
        app.currentEditingLocations = app.currentEditingLocations.filter(loc => loc.id !== idToRemove);
        ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
    },

    handleSaveLocationConfig: () => {
        app.locations = app.currentEditingLocations;
        store.saveLocations(app.locations);
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG);
        app.updateGlobalRefreshButtonVisibility();
    },

    handleAddOrUpdateTopicInEditList: () => {
        const description = ui.topicDescriptionInput.value.trim();
        const aiQuery = ui.topicAiQueryInput.value.trim();
        if (!description || !aiQuery) {
            ui.topicConfigError.textContent = 'Both topic description and AI query are required.';
            return;
        }
        ui.topicConfigError.textContent = '';
        if (app.editingTopicId) {
            const topic = app.currentEditingTopics.find(t => t.id === app.editingTopicId);
            if (topic) {
                topic.description = description;
                topic.aiQuery = aiQuery;
            }
        } else {
            const newTopic = { id: utils.generateId(), description, aiQuery };
            app.currentEditingTopics.push(newTopic);
        }
        app.editingTopicId = null;
        ui.addTopicBtn.textContent = 'Add Topic';
        ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
    },

    handleRemoveTopicFromEditList: (idToRemove) => {
        app.currentEditingTopics = app.currentEditingTopics.filter(topic => topic.id !== idToRemove);
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
    },

    prepareEditTopic: (idToEdit) => {
        const topic = app.currentEditingTopics.find(t => t.id === idToEdit);
        if (topic) {
            app.editingTopicId = idToEdit;
            ui.topicDescriptionInput.value = topic.description;
            ui.topicAiQueryInput.value = topic.aiQuery;
            ui.addTopicBtn.textContent = 'Update Topic';
            ui.topicDescriptionInput.focus();
        }
    },

    handleSaveTopicConfig: () => {
        app.topics = app.currentEditingTopics;
        store.saveTopics(app.topics);
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG);
        app.updateGlobalRefreshButtonVisibility();
    },

    hasUnsavedChanges: (modalId) => {
        if (modalId === APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG) {
            return JSON.stringify(app.currentEditingLocations) !== app.initialEditingLocationsString;
        }
        if (modalId === APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG) {
            return JSON.stringify(app.currentEditingTopics) !== app.initialEditingTopicsString;
        }
        return false;
    },

    handleLocationButtonClick: async (locationId) => {
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;

        app.currentLocationIdForInfoModal = locationId;
        const isCurrentlyFetching = !!app.fetchingStatus[locationId];

        if (!app.topics || app.topics.length === 0) {
            ui.displayInfoModal(location, [], {}, false, !window.isActuallyOnline);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO);
            if(ui.infoModalUpdated) ui.infoModalUpdated.textContent = 'No Info Structure defined in Config.';
            if(ui.refreshInfoButton) ui.refreshInfoButton.classList.add('hidden');
            return;
        }
        
        const cachedData = {};
        let needsRefresh = false;
        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(location.id, topic.id);
            cachedData[topic.id] = cacheEntry;
            if (!isCurrentlyFetching && (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS)) {
                needsRefresh = true;
            }
        });
        
        await ui.displayInfoModal(location, app.topics, cachedData, isCurrentlyFetching, !window.isActuallyOnline);
        ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO);

        if (needsRefresh && window.isActuallyOnline) {
            console.log(`Data for ${location.description} is stale. Triggering refresh.`);
            app.handleRefreshLocationInfo(locationId);
        }
    },

    handleRefreshLocationInfo: async (locationId, forceAll = false) => {
        if (!window.isActuallyOnline) {
            console.warn("Attempted to refresh info while offline. Aborting.");
            ui.showOfflineIndicator(true);
            return;
        }

        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;
        const areTopicsDefined = app.topics && app.topics.length > 0;

        if (app.fetchingStatus[locationId]) {
            console.log(`Already fetching data for ${location.description}.`);
            return;
        }

        app.fetchingStatus[locationId] = true;
        app.incrementActiveLoaders();
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);

        let topicsToFetch = app.topics;
        if (!forceAll) {
            topicsToFetch = app.topics.filter(topic => {
                const cacheEntry = store.getAiCache(locationId, topic.id);
                const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
                const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                return isStale || hasError;
            });
        }
        
        if (topicsToFetch.length === 0) {
            console.log(`All topics for ${location.description} are fresh. No refresh needed.`);
            delete app.fetchingStatus[locationId];
            app.decrementActiveLoaders();
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            return;
        }

        console.log(`Fetching ${topicsToFetch.length} topics for ${location.description}`);
        const totalTopics = topicsToFetch.length;
        let fetchedCount = 0;
        
        const updateLoadingMessage = () => {
             if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
                if (ui.infoModalTitle) ui.infoModalTitle.textContent = `Fetching ${location.description} (${fetchedCount}/${totalTopics})...`;
                if (ui.infoModalUpdated) ui.infoModalUpdated.textContent = 'Fetching latest AI data...';
             }
        };
        updateLoadingMessage();

        const fetchPromises = topicsToFetch.map(async (topic) => {
            try {
                const result = await api.fetchAiData(app.config.apiKey, location.description, topic.aiQuery);
                store.saveAiCache(locationId, topic.id, result);
            } catch (error) {
                console.error(`Error fetching topic ${topic.description} for ${location.description}:`, error);
                store.saveAiCache(locationId, topic.id, `Error: ${error.message}`);
            } finally {
                fetchedCount++;
                updateLoadingMessage();
            }
        });

        await Promise.all(fetchPromises);

        delete app.fetchingStatus[locationId];
        app.decrementActiveLoaders();

        // After all fetches complete, update UI
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
            const newCachedData = {};
            app.topics.forEach(topic => { newCachedData[topic.id] = store.getAiCache(location.id, topic.id); });
            ui.displayInfoModal(location, app.topics, newCachedData, false, !window.isActuallyOnline);
        }
        app.updateGlobalRefreshButtonVisibility();
    },

    refreshOutdatedQueries: async (forceAll = false) => {
        if (app.isRefreshingAllStale) {
            console.log("Global refresh already in progress. Skipping.");
            return;
        }
        if (!window.isActuallyOnline) {
            console.warn("Attempted to refresh all while offline. Aborting.");
            ui.showOfflineIndicator(true);
            return;
        }

        app.isRefreshingAllStale = true;
        let completedCount = 0;
        ui.pauseHeaderIconLoading();

        for (const location of app.locations) {
            await app.handleRefreshLocationInfo(location.id, forceAll);
        }

        app.isRefreshingAllStale = false;
        ui.resumeHeaderIconLoading();
    },




    updateGlobalRefreshButtonVisibility: () => {
        if (!ui.globalRefreshButton || !app.topics || app.topics.length === 0 || !app.locations || app.locations.length === 0) {
            if (ui.globalRefreshButton) ui.globalRefreshButton.classList.add('hidden');
            return;
        }

        let isAnyStale = false;
        for (const location of app.locations) {
            for (const topic of app.topics) {
                const cacheEntry = store.getAiCache(location.id, topic.id);
                if (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS) {
                    isAnyStale = true;
                    break;
                }
            }
            if (isAnyStale) break;
        }

        ui.globalRefreshButton.classList.toggle('hidden', !isAnyStale || !window.isActuallyOnline);
    },

    // Loading indicator management
    incrementActiveLoaders: () => {
        app.activeLoadingOperations++;
        if (app.activeLoadingOperations > 0) {
            ui.startHeaderIconLoading();
        }
    },
    decrementActiveLoaders: () => {
        app.activeLoadingOperations--;
        if (app.activeLoadingOperations <= 0) {
            app.activeLoadingOperations = 0;
            ui.stopHeaderIconLoading();
        }
    },

    // API Key Validation
    validateAndDisplayGeminiKeyStatus: async (key, onOpen = false) => {
        if (!key) {
            ui.setApiKeyStatus('gemini', 'invalid', 'No Key');
            return false;
        }
        if (!onOpen) {
            ui.setApiKeyStatus('gemini', 'checking', 'Checking...');
        }
        const result = await api.validateGeminiApiKey(key);
        // The key is considered "valid" for the purpose of proceeding if the API call didn't fail because of a network issue
        if (result.reason === 'network_error') {
            ui.setApiKeyStatus('gemini', 'invalid', 'Network Error');
            return false; // Cannot validate, so treat as invalid for now
        }
        
        if (result.isValid) {
            ui.setApiKeyStatus('gemini', 'valid', 'Valid');
            return true;
        } else {
            const message = result.reason === 'rate_limit' ? 'Rate Limit' : 'Invalid';
            ui.setApiKeyStatus('gemini', 'invalid', message);
            return false;
        }
    },

    validateAndDisplayOwmKeyStatus: async (key, onOpen = false) => {
        if (!key) {
            ui.setApiKeyStatus('owm', 'invalid', 'No Key');
            return false;
        }
        if (!onOpen) {
            ui.setApiKeyStatus('owm', 'checking', 'Checking...');
        }
        const result = await api.validateOwmApiKey(key);
        if (result.isValid) {
            ui.setApiKeyStatus('owm', 'valid', 'Valid');
            return true;
        } else {
             const message = result.reason === 'rate_limit' ? 'Rate Limit' : 'Invalid';
            ui.setApiKeyStatus('owm', 'invalid', message);
            return false;
        }
    },

    // SW Update Logic
    promptUserToUpdate: (worker) => {
        app.newWorkerForUpdate = worker;
        if (ui.btnAppUpdate) {
            ui.btnAppUpdate.classList.remove('hidden');
        }
    },

    triggerUpdate: () => {
        if (app.newWorkerForUpdate) {
            app.userInitiatedUpdate = true;
            app.newWorkerForUpdate.postMessage({ type: APP_CONSTANTS.SW_MESSAGES.SKIP_WAITING });
        }
    },

    listenForControllerChange: () => {
         navigator.serviceWorker.addEventListener('controllerchange', () => {
            if (app.userInitiatedUpdate) {
                window.location.reload();
            }
        });
    },

    // Weather Management Functions
    getWeatherDisplayForLocation: async (location) => {
        if (!app.isOwmConfigured()) {
            // Offline mode: only rely on cache.
            const cachedWeather = store.getWeatherCache(location.id);
            if (cachedWeather) {
                 return app.formatWeatherDisplay(cachedWeather.data, true); // true indicates cached data
            }
            return null; // No key or no cache, show nothing
        }

        // Online mode
        if (!window.isActuallyOnline) {
             // We think we have a key, but we are actually offline. Fallback to cache.
            const cachedWeather = store.getWeatherCache(location.id);
            if (cachedWeather) return app.formatWeatherDisplay(cachedWeather.data, true);
            return null;
        }
        
        const coords = await utils.extractCoordinates(location.location);
        if (!coords) return null;

        const cachedWeather = store.getWeatherCache(location.id);
        if (cachedWeather && (Date.now() - cachedWeather.timestamp < APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS)) {
            return app.formatWeatherDisplay(cachedWeather.data);
        }

        try {
            const weatherData = await api.fetchWeatherData(coords.lat, coords.lon, app.config.owmApiKey);
            if (weatherData) {
                store.saveWeatherCache(location.id, weatherData);
                return app.formatWeatherDisplay(weatherData);
            }
        } catch (error) {
            console.error("Error in getWeatherDisplayForLocation fetch:", error);
            // Fallback to cache if network fails
            if (cachedWeather) return app.formatWeatherDisplay(cachedWeather.data, true);
        }
        return null;
    },

    formatWeatherDisplay: (weatherData, isFromCache = false) => {
        if (!weatherData) return '';
        const temp = Math.round(weatherData.temp);
        const description = weatherData.weather[0] ? weatherData.weather[0].main : 'N/A';
        const iconCode = weatherData.weather[0] ? weatherData.weather[0].icon : null;
        const iconUrl = iconCode ? `https://openweathermap.org/img/wn/${iconCode}.png` : '';
        
        let displayHtml = `<img src="${iconUrl}" alt="${description}" class="weather-icon-small"> ${temp}°F`;
        if(isFromCache) {
            displayHtml += ` <span class="weather-cached-indicator">(cached)</span>`;
        }
        return displayHtml;
    },

    refreshOutdatedWeather: async () => {
        if (!app.isOwmConfigured() || !window.isActuallyOnline) return;

        console.log("Checking for outdated weather data for all locations...");
        const refreshPromises = app.locations.map(async (location) => {
            const cachedWeather = store.getWeatherCache(location.id);
            if (!cachedWeather || (Date.now() - cachedWeather.timestamp > APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS)) {
                console.log(`Weather for ${location.description} is stale or missing, refreshing.`);
                const coords = await utils.extractCoordinates(location.location);
                if (coords) {
                    try {
                        const weatherData = await api.fetchWeatherData(coords.lat, coords.lon, app.config.owmApiKey);
                        if (weatherData) {
                            store.saveWeatherCache(location.id, weatherData);
                        }
                    } catch (error) {
                         console.error(`Failed to refresh weather for ${location.description}:`, error);
                    }
                }
            }
        });
        await Promise.all(refreshPromises);
        console.log("Weather refresh check complete.");
    },

    updateOpenInfoModalWeather: async () => {
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal) {
            const location = app.locations.find(l => l.id === app.currentLocationIdForInfoModal);
            if (location) {
                await ui.refreshInfoModalWeatherOnly(location);
            }
        }
    },
    
    isOwmConfigured: () => {
        return !!(app.config && app.config.owmApiKey);
    }
};

document.addEventListener('DOMContentLoaded', app.init);
