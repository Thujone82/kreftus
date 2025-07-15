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

    init: () => {
        console.log("App initializing...");
        app.loadAndApplyAppSettings(); // This will populate app.config
        app.loadLocations();
        app.loadTopics();
        app.registerServiceWorker();
        app.setupEventListeners();

        // Check for Gemini API Key
        if (app.config && app.config.apiKey) {
            ui.toggleConfigButtons(true); // Enable location/topic config
        } else {
            ui.toggleConfigButtons(false);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            if (ui.appConfigError) ui.appConfigError.textContent = "Gemini API Key is required for core functionality.";
        }


        // Check for OpenWeatherMap API Key for weather features (logging for now)
        if (app.config && app.config.owmApiKey) {
            console.log('OpenWeatherMap API Key is set. Weather features can be enabled.');
            // Initial weather refresh on load, then update buttons
            app.incrementActiveLoaders(); // Weather refresh starting
            app.refreshOutdatedWeather()
                .then(() => {
                    const areTopicsDefined = app.topics && app.topics.length > 0;
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    app.updateOpenInfoModalWeather(); // Refresh weather in open info modal
                })
                .catch(error => {
                    console.error("Error during initial weather refresh:", error);
                })
                .finally(() => {
                    app.decrementActiveLoaders(); // Decrement after initial weather refresh completes
                });

            // Set up periodic refresh for weather
            setInterval(async () => {
                if (app.config.owmApiKey) { // Re-check in case it's removed during runtime
                    console.log("Periodic weather refresh triggered by timer.");
                    app.incrementActiveLoaders();
                    await app.refreshOutdatedWeather();
                    const areTopicsDefined = app.topics && app.topics.length > 0;
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    await app.updateOpenInfoModalWeather(); // Refresh weather in open info modal
                    app.decrementActiveLoaders();
                }
            }, APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS); // Refresh at the same interval as stale time

            // Refresh weather when app becomes visible
            document.addEventListener('visibilitychange', async () => {
                if (document.visibilityState === 'visible' && app.config.owmApiKey) {
                    console.log("App became visible, refreshing weather.");
                    app.incrementActiveLoaders();
                    await app.refreshOutdatedWeather();
                    const areTopicsDefined = app.topics && app.topics.length > 0;
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    await app.updateOpenInfoModalWeather(); // Refresh weather in open info modal
                    app.decrementActiveLoaders();
                }
            });
            // app.decrementActiveLoaders(); // Moved to the .finally() block of the initial refresh
        } else {
            console.log('OpenWeatherMap API Key is NOT set. Weather features will be disabled.');
        }

        app.updateGlobalRefreshButtonVisibility(); // Initial check for button visibility
        console.log("App initialized.");
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
                                    // A new SW has installed.
                                    // If there's an active SW controlling the page, this new one is waiting.
                                    if (navigator.serviceWorker.controller) {
                                        console.log('SW Registration: New SW installed and waiting (controller exists). Prompting user.');
                                        app.promptUserToUpdate(installingWorker);
                                    } else {
                                        // No current controller, so this installed SW will activate on next load/claim.
                                        // This is typical for the first SW installation.
                                        console.log('SW Registration: SW installed. No active controller. Will activate on next load or if claimed.');
                                    }
                                } else if (installingWorker.state === 'redundant') { // Added for completeness
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
        const storedSettings = store.getAppSettings(); // getAppSettings provides defaults
        app.config = { // Ensure all expected keys are present
            apiKey: storedSettings.apiKey || null,
            owmApiKey: storedSettings.owmApiKey || null,
            rpmLimit: storedSettings.rpmLimit || 10,
            primaryColor: storedSettings.primaryColor || '#029ec5',
            backgroundColor: storedSettings.backgroundColor || '#1E1E1E'
        };

        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.loadAppConfigForm(app.config); // Pass the fully populated app.config
        console.log("App settings loaded and applied:", app.config);
    },

    loadLocations: () => {
        app.locations = store.getLocations();
        console.log("Locations loaded:", app.locations);
    },

    loadTopics: () => {
        app.topics = store.getTopics();
        console.log("Topics loaded:", app.topics);
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
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
        console.log("Event listeners set up.");
    },

    reorderArrayByIds: (originalArray, idOrderArray) => {
        const itemMap = new Map(originalArray.map(item => [item.id, item]));
        return idOrderArray.map(id => itemMap.get(id)).filter(item => item !== undefined);
    },

    handleSaveAppSettings: () => {
        const newGeminiApiKey = ui.apiKeyInput.value.trim(); // Gemini Key
        const newOwmApiKey = ui.owmApiKeyInput.value.trim(); // OpenWeatherMap Key
        const newRpmLimit = parseInt(ui.rpmLimitInput.value, 10) || 10; // RPM Limit
        const newPrimaryColor = ui.primaryColorInput.value;
        const newBackgroundColor = ui.backgroundColorInput.value;

        if (!newGeminiApiKey) {
            if(ui.appConfigError) ui.appConfigError.textContent = "Gemini API Key is required.";
            if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.remove('hidden');
            return;
        }
        if(ui.appConfigError) ui.appConfigError.textContent = "";
        if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.add('hidden');

        app.config.apiKey = newGeminiApiKey;
        app.config.owmApiKey = newOwmApiKey; // Save OWM key to app.config
        app.config.rpmLimit = newRpmLimit;   // Save RPM limit
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config); // saveAppSettings in stor.js needs to handle all these
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.toggleConfigButtons(true); // For Gemini key dependent buttons

        // Update visibility for OWM API key link
        if (ui.getOwmApiKeyLinkContainer) {
            ui.getOwmApiKeyLinkContainer.classList.toggle('hidden', !!newOwmApiKey);
        }

        ui.closeModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
        console.log("App settings saved:", app.config);
        app.updateGlobalRefreshButtonVisibility();

        // Log OWM key status after save
        if (app.config.owmApiKey) {
            console.log('OpenWeatherMap API Key is set. Weather features can be enabled.');
            // Trigger a weather refresh and UI update if the key was just added
            app.incrementActiveLoaders();
            app.refreshOutdatedWeather()
                .then(() => {
                    const areTopicsDefined = app.topics && app.topics.length > 0;
                    ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                    app.updateOpenInfoModalWeather(); // Refresh weather in open info modal
                })
                .catch(error => {
                    console.error("Error refreshing weather after OWM key save:", error);
                })
                .finally(() => {
                    app.decrementActiveLoaders(); // Decrement after weather refresh completes
                });
            // Validate and display status for OWM key
            app.validateAndDisplayOwmKeyStatus(newOwmApiKey);
        } else {
            console.log('OpenWeatherMap API Key is NOT set. Weather features will be disabled.');
            if(ui.owmApiKeyStatusUI) ui.owmApiKeyStatusUI.textContent = ''; // Clear status if key removed
        }
    },

    handleAddLocationToEditList: () => {
        const description = ui.locationDescriptionInput.value.trim();
        const locationVal = ui.locationValueInput.value.trim();
        if (description && locationVal) {
            const newLocation = { id: utils.generateId(), description, location: locationVal };
            app.currentEditingLocations.push(newLocation);
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            if (ui.locationConfigError) ui.locationConfigError.textContent = '';
            ui.clearInputFields([ui.locationDescriptionInput, ui.locationValueInput]);
        } else {
            if (ui.locationConfigError) ui.locationConfigError.textContent = "Both description and location value are required.";
        }
    },
    handleRemoveLocationFromEditList: (locationId) => {
        app.currentEditingLocations = app.currentEditingLocations.filter(loc => loc.id !== locationId);
        ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
    },
    handleSaveLocationConfig: async () => {
        const oldLocationIds = new Set(app.locations.map(l => l.id));
        app.locations = [...app.currentEditingLocations];
        const areTopicsDefined = app.topics && app.topics.length > 0;
        store.saveLocations(app.locations);
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.initialEditingLocationsString = JSON.stringify(app.locations);
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG);
        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) {
                console.log(`New location added: ${location.description}. Fetching initial data.`);
                // fetchAndCacheAiDataForLocation will handle its own loader increment/decrement
                await app.fetchAndCacheAiDataForLocation(location.id, true);
            }
        }
        app.updateGlobalRefreshButtonVisibility();
    },

    prepareEditTopic: (topicId) => {
        const topicToEdit = app.currentEditingTopics.find(t => t.id === topicId);
        if (topicToEdit) {
            app.editingTopicId = topicId;
            ui.topicDescriptionInput.value = topicToEdit.description;
            ui.topicAiQueryInput.value = topicToEdit.aiQuery;
            ui.addTopicBtn.textContent = 'Update Topic';
            ui.topicDescriptionInput.focus();
        }
    },
    handleAddOrUpdateTopicInEditList: () => {
        const description = ui.topicDescriptionInput.value.trim();
        const aiQuery = ui.topicAiQueryInput.value.trim();
        if (!description || !aiQuery) {
            if (ui.topicConfigError) ui.topicConfigError.textContent = "Both description and AI query are required.";
            return;
        }
        if (app.editingTopicId) {
            const topicToUpdate = app.currentEditingTopics.find(t => t.id === app.editingTopicId);
            if (topicToUpdate) { topicToUpdate.description = description; topicToUpdate.aiQuery = aiQuery; }
            app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
        } else {
            app.currentEditingTopics.push({ id: utils.generateId(), description, aiQuery });
        }
        if (ui.topicConfigError) ui.topicConfigError.textContent = '';
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
        ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
    },
    handleRemoveTopicFromEditList: (topicId) => {
        app.currentEditingTopics = app.currentEditingTopics.filter(topic => topic.id !== topicId);
        if (app.editingTopicId === topicId) {
            app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
        }
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
    },
    handleSaveTopicConfig: () => {
        const previousTopics = new Map(app.topics.map(t => [t.id, t]));
        const currentEditingTopicsMap = new Map(app.currentEditingTopics.map(t => [t.id, t]));

        const queryChangedTopicIds = new Set();
        const deletedTopicIds = new Set();

        app.currentEditingTopics.forEach(currentTopic => {
            const prevTopic = previousTopics.get(currentTopic.id);
            if (prevTopic && prevTopic.aiQuery !== currentTopic.aiQuery) {
                queryChangedTopicIds.add(currentTopic.id);
            }
        });

        previousTopics.forEach(prevTopic => {
            if (!currentEditingTopicsMap.has(prevTopic.id)) {
                deletedTopicIds.add(prevTopic.id);
            }
        });

        if (queryChangedTopicIds.size > 0 || deletedTopicIds.size > 0) {
            console.log("Query changed or topics deleted, selectively invalidating cache.");
            app.locations.forEach(location => {
                queryChangedTopicIds.forEach(topicId => {
                    store.flushAiCacheForLocationAndTopic(location.id, topicId);
                });
                deletedTopicIds.forEach(topicId => {
                    store.flushAiCacheForLocationAndTopic(location.id, topicId);
                });
            });
        }

        app.topics = [...app.currentEditingTopics];
        store.saveTopics(app.topics);
        app.initialEditingTopicsString = JSON.stringify(app.topics);
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG);
        app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.updateGlobalRefreshButtonVisibility();

        const newOrChangedQueryTopicIds = new Set();
        app.currentEditingTopics.forEach(currentTopic => {
            const prevTopic = previousTopics.get(currentTopic.id);
            if (!prevTopic) {
                newOrChangedQueryTopicIds.add(currentTopic.id);
            } else if (prevTopic.aiQuery !== currentTopic.aiQuery) {
                newOrChangedQueryTopicIds.add(currentTopic.id);
            }
        });

        if (newOrChangedQueryTopicIds.size > 0) {
            const idsToRefresh = Array.from(newOrChangedQueryTopicIds);
            console.log("Info Structure updated. Triggering specific refresh for EDITED/NEW items:", idsToRefresh);
            // fetchAndCacheAiDataForLocation will handle its own loader increment/decrement for each location
            app.locations.forEach(location => {
                app.fetchAndCacheAiDataForLocation(location.id, false, idsToRefresh);
            });
        } else {
            console.log("Info Structure updated. No AI queries changed, no new topics. No specific refresh needed from this operation.");
        }
    },
    handleLocationButtonClick: (locationId) => {
        if (app.topics && app.topics.length > 0) app.handleOpenLocationInfo(locationId);
        else { alert("Please define an Info Structure before viewing location information."); ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG); }
    },
    handleOpenLocationInfo: async (locationId) => {
        if (!app.config.apiKey) {
            if(ui.appConfigError) ui.appConfigError.textContent = "API Key is not configured. Please configure it first.";
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG); return;
        }
        app.currentLocationIdForInfoModal = locationId;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;

        if (ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go - Loading...`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = '<p>Accessing stored data...</p>';
        ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO);

        // Check for offline status immediately after opening the modal
        if (!window.isActuallyOnline) { // Use our reliable, tested flag
            console.log("Connectivity Test: App is offline. Displaying cached data immediately.");
            const cachedDataForLocation = {};
            app.topics.forEach(topic => {
                cachedDataForLocation[topic.id] = store.getAiCache(locationId, topic.id);
            });
            // Call displayInfoModal with isOffline = true and isCurrentlyFetching = false
            ui.displayInfoModal(location, app.topics, cachedDataForLocation, false, true);
            return; // Stop further execution to prevent network requests
        }

        const cachedDataForLocation = {};
        let needsOverallRefreshForModal = false;
        let isCurrentlyFetchingForThisLocation = app.fetchingStatus[locationId] === true;

        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            cachedDataForLocation[topic.id] = cacheEntry;
            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
            const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
            if (isStale || hasError) {
                needsOverallRefreshForModal = true;
            }
        });

        await ui.displayInfoModal(location, app.topics, cachedDataForLocation, isCurrentlyFetchingForThisLocation);

        if (needsOverallRefreshForModal && !isCurrentlyFetchingForThisLocation && ui.refreshInfoButton) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = locationId;
        } else if (ui.refreshInfoButton) {
            ui.refreshInfoButton.classList.add('hidden');
        }
    },

    handleRefreshLocationInfo: async (locationId) => {
        if (!locationId) locationId = app.currentLocationIdForInfoModal;
        if (!locationId) return;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;
        // Immediately hide the refresh button when a refresh is initiated.
        if (ui.refreshInfoButton) {
            ui.refreshInfoButton.classList.add('hidden');
        }
        if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go - Refreshing...`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = '<p>Fetching fresh data...</p>';
        await app.fetchAndCacheAiDataForLocation(locationId, false);
        // app.handleOpenLocationInfo(locationId); // This call is redundant as fetchAndCacheAiDataForLocation updates the modal if open.
    },

    fetchAndCacheAiDataForLocation: async (locationId, forceRefreshGeneral = false, specificTopicIdsToForce = null) => {
        if (!app.config.apiKey) {
            if (document.getElementById(APP_CONSTANTS.MODAL_IDS.APP_CONFIG).style.display !== 'block') {
                 if(ui.appConfigError) ui.appConfigError.textContent = "API Key is required to fetch data.";
                 ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            } return false;
        }
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return false;

        const areTopicsDefined = app.topics && app.topics.length > 0;

        const topicsToFetch = app.topics.filter(topic => {
            if (specificTopicIdsToForce && specificTopicIdsToForce.length > 0) {
                return specificTopicIdsToForce.includes(topic.id);
            } else {
                const cacheEntry = store.getAiCache(locationId, topic.id);
                const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
                const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                return forceRefreshGeneral || isStale || hasError;
            }
        });

        console.log(`[FADFL] Location: ${location.description}. Filtered topicsToFetch:`, topicsToFetch.map(t => ({id: t.id, desc: t.description})));

        if (topicsToFetch.length > 0) {
            app.fetchingStatus[locationId] = true;
            app.incrementActiveLoaders(); // AI fetch starting for this location
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        }

        console.log(`Fetching/Caching AI data for location: ${location.description}. GeneralForce: ${forceRefreshGeneral}, SpecificIDs: ${specificTopicIdsToForce ? specificTopicIdsToForce.join(', ') : 'None'}`);

        let completedCount = 0;
        const totalTopicsToFetch = topicsToFetch.length;

        if (document.getElementById(APP_CONSTANTS.MODAL_IDS.INFO).style.display === 'block' && app.currentLocationIdForInfoModal === locationId && totalTopicsToFetch > 0) {
            app.updateInfoModalLoadingMessage(location.description, completedCount, totalTopicsToFetch);
        }

        const fetchExecutionPromises = [];
        let anInvalidKeyErrorOccurred = false;

        if (totalTopicsToFetch === 0) {
            console.log(`No topics to fetch for ${location.description}.`);
            if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId && ui.infoModalTitle) {
                 if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go`;
            }
            // If fetchingStatus was set to true but no topics ended up being fetched, clear it.
            if (app.fetchingStatus[locationId]) {
                delete app.fetchingStatus[locationId];
                if (topicsToFetch.length > 0) app.decrementActiveLoaders(); // Ensure loader is decremented if it was incremented
                ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            }
            return true;
        }

        for (const topic of topicsToFetch) {
            console.log(`Preparing to fetch for topic: ${topic.description}`);
            const modifiedAiQuery = `${topic.aiQuery} Ensure the output is in markdown format.`;
            const promise = api.fetchAiData(app.config.apiKey, location.location, modifiedAiQuery)
                .then(aiData => {
                    store.saveAiCache(locationId, topic.id, aiData);
                    console.log(`Successfully fetched and cached data for ${location.description} - ${topic.description}`);
                    return { status: 'fulfilled', topicId: topic.id };
                })
                .catch(error => {
                    console.error(`Failed to fetch AI data for ${location.description} - ${topic.description}:`, error.message);
                    store.saveAiCache(locationId, topic.id, `Error: ${error.message}`);
                    if (error.message.toLowerCase().includes("invalid api key")) {
                        anInvalidKeyErrorOccurred = true;
                    }
                    return { status: 'rejected', topicId: topic.id, reason: error };
                })
                .finally(() => {
                    completedCount++;
                    if (document.getElementById(APP_CONSTANTS.MODAL_IDS.INFO).style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
                        app.updateInfoModalLoadingMessage(location.description, completedCount, totalTopicsToFetch);
                    }
                });
            fetchExecutionPromises.push(promise);
        }

        const results = await Promise.all(fetchExecutionPromises);
        let allIndividualFetchesSuccessful = true;
        results.forEach(result => {
            if (result.status === 'rejected') {
                allIndividualFetchesSuccessful = false;
            }
        });

        delete app.fetchingStatus[locationId];
        if (topicsToFetch.length > 0) app.decrementActiveLoaders(); // AI fetch ended for this location
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.updateGlobalRefreshButtonVisibility();

        if (anInvalidKeyErrorOccurred) {
            if(ui.infoModal) ui.closeModal(APP_CONSTANTS.MODAL_IDS.INFO);
            if(ui.appConfigError) ui.appConfigError.textContent = "Invalid API Key. Please check your configuration and save.";
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            return false;
        }

        if (document.getElementById(APP_CONSTANTS.MODAL_IDS.INFO).style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
            const currentCachedData = {};
            let isStillFetching = app.fetchingStatus[locationId] === true;
            let needsRefreshAfterFetch = false;

            app.topics.forEach(t => {
                const cacheEntry = store.getAiCache(locationId, t.id);
                currentCachedData[t.id] = cacheEntry;
                if (!isStillFetching && (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS || (cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:')))) {
                    needsRefreshAfterFetch = true;
                }
            });
            ui.displayInfoModal(location, app.topics, currentCachedData, isStillFetching);
           if (needsRefreshAfterFetch && ui.refreshInfoButton) {
                ui.refreshInfoButton.classList.remove('hidden');
            } else if (ui.refreshInfoButton) {
                ui.refreshInfoButton.classList.add('hidden');
            }
        }
        return allIndividualFetchesSuccessful;
    },

    updateInfoModalLoadingMessage: (locationDescription, completed, total) => {
        if (ui.infoModalTitle) {
            if (total === 0) {
                // Handled by displayInfoModal
            } else {
                ui.infoModalTitle.textContent = `${locationDescription} nfo2Go - Fetching (${completed}/${total})`;
                if (completed === total){
                     setTimeout(() => {
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal && app.locations.find(l=>l.id === app.currentLocationIdForInfoModal)?.description === locationDescription) {
                           // Final display update handled by calling function (e.g., handleOpenLocationInfo or the end of fetchAndCacheAiDataForLocation)
                        }
                    }, 50);
                }
            }
        }
    },

    refreshOutdatedQueries: async (forceAllStale = false) => {
        if (app.isRefreshingAllStale) { // Prevent re-entry if already running
            console.log("Global refresh already in progress. Skipping new request.");
            return;
        }
        if (!app.config.apiKey) return;

        const outdatedItemsToFetch = [];
        const sixtyMinutesAgo = Date.now() - APP_CONSTANTS.CACHE_EXPIRY_MS;

        for (const location of app.locations) {
            if (app.topics.length === 0) continue;
            for (const topic of app.topics) {
                const cacheEntry = store.getAiCache(location.id, topic.id);
                const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                const isStale = !cacheEntry || (cacheEntry.timestamp || 0) < sixtyMinutesAgo;
                
                if (forceAllStale || hasError || isStale) {
                    outdatedItemsToFetch.push({
                        locationId: location.id,
                        topicId: topic.id,
                        locationName: location.location, // For the prompt
                        topicDescription: topic.description, // For logging
                        aiQuery: topic.aiQuery // For the prompt
                    });
                }
            }
        }

        const totalOutdatedCount = outdatedItemsToFetch.length;

        if (totalOutdatedCount === 0) {
            console.log("Global refresh: No outdated items to fetch.");
            // app.isRefreshingAllStale is false here, so no need to reset
            app.updateGlobalRefreshButtonVisibility(); // Ensure button is hidden or text is correct
            return;
        }

        app.incrementActiveLoaders(); // Start global loading indicator
        console.log("Starting global refresh of outdated queries...");
        if (ui.globalRefreshButton) {
            app.isRefreshingAllStale = true; // Set flag now that we are starting the process
            ui.globalRefreshButton.classList.add('button-fetching');
            ui.globalRefreshButton.textContent = `Fetching (0/${totalOutdatedCount})...`;
        }

        let completedFetchCount = 0;
        const rpm = app.config.rpmLimit || 10;

        // Define the cleanup function
        function finishGlobalRefresh() {
            if (!app.isRefreshingAllStale) return; // Avoid double cleanup

            console.log("Global refresh of outdated queries finished or stopped.");
            app.decrementActiveLoaders(); // Global AI refresh ended
            app.isRefreshingAllStale = false;
            if (ui.globalRefreshButton) {
                ui.globalRefreshButton.classList.remove('button-fetching');
            }
            const areTopicsDefined = app.topics && app.topics.length > 0;
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            app.updateGlobalRefreshButtonVisibility();
        }

        async function processBatch(startIndex) {
            if (!app.isRefreshingAllStale) { // Check if refresh was cancelled/stopped
                console.log("Global refresh was stopped before processing a batch.");
                finishGlobalRefresh();
                return;
            }

            const batch = outdatedItemsToFetch.slice(startIndex, startIndex + rpm);
            if (batch.length === 0) { // All items processed
                finishGlobalRefresh();
                return;
            }

            if (ui.resumeHeaderIconLoading) ui.resumeHeaderIconLoading();
            if (ui.globalRefreshButton) ui.globalRefreshButton.textContent = `Fetching (${completedFetchCount}/${totalOutdatedCount})...`;

            const batchPromises = batch.map(item => {
                const modifiedAiQuery = `${item.aiQuery} Ensure the output is in markdown format.`;
                return api.fetchAiData(app.config.apiKey, item.locationName, modifiedAiQuery)
                    .then(aiData => {
                        store.saveAiCache(item.locationId, item.topicId, aiData);
                        console.log(`Global Refresh: Successfully fetched for ${item.locationName} - ${item.topicDescription}`);
                    })
                    .catch(error => {
                        console.error(`Global Refresh: Failed for ${item.locationName} - ${item.topicDescription}:`, error.message);
                        store.saveAiCache(item.locationId, item.topicId, `Error: ${error.message}`);
                    })
                    .finally(() => {
                        completedFetchCount++;
                        if (ui.globalRefreshButton && app.isRefreshingAllStale) {
                            ui.globalRefreshButton.textContent = `Fetching (${completedFetchCount}/${totalOutdatedCount})...`;
                        }
                    });
            });

            await Promise.allSettled(batchPromises);

            if (startIndex + rpm < totalOutdatedCount && app.isRefreshingAllStale) {
                if (ui.globalRefreshButton) ui.globalRefreshButton.textContent = `Waiting... (${completedFetchCount}/${totalOutdatedCount})`;
                if (ui.pauseHeaderIconLoading) ui.pauseHeaderIconLoading();
                setTimeout(() => processBatch(startIndex + rpm), 60 * 1000); // 1 minute timer
            } else { // All batches processed or refresh stopped
                finishGlobalRefresh();
            }
        }
        processBatch(0); // Start processing the first batch
    },

    updateGlobalRefreshButtonVisibility: () => {
        if (!ui.globalRefreshButton) return;
        let outdatedTopicsCount = 0;

        if (app.topics.length > 0) {
            for (const location of app.locations) {
                if (app.fetchingStatus && app.fetchingStatus[location.id]) continue;
                for (const topic of app.topics) {
                    const cacheEntry = store.getAiCache(location.id, topic.id);
                    const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                    const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS; // AI Cache expiry
                    if (hasError || isStale) {
                        outdatedTopicsCount++;
                    }
                }
            }
        }

        if (outdatedTopicsCount > 0) {
            ui.globalRefreshButton.textContent = `Refresh Outdated (${outdatedTopicsCount})`;
            ui.globalRefreshButton.classList.remove('hidden');
        } else {
            ui.globalRefreshButton.classList.add('hidden');
        }
    },

    updateOpenInfoModalWeather: async () => {
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal) {
            const locationForModal = app.locations.find(l => l.id === app.currentLocationIdForInfoModal);
            if (locationForModal) {
                console.log(`App: Triggering weather refresh for open info modal: ${locationForModal.description}`);
                // Call the UI function that specifically updates the weather part in the info modal
                await ui.refreshInfoModalWeatherOnly(locationForModal);
            }
        }
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

    validateAndDisplayGeminiKeyStatus: async (apiKeyToValidate, onOpen = false) => {
        if (!apiKeyToValidate) {
            ui.setApiKeyStatus('gemini', 'checking', 'Enter Key');
            return;
        }
        if (!onOpen) ui.setApiKeyStatus('gemini', 'checking', 'Checking...');
        const validationResult = await api.validateGeminiApiKey(apiKeyToValidate);
        if (validationResult.isValid) {
            ui.setApiKeyStatus('gemini', 'valid', 'Valid');
        } else {
            if (validationResult.reason === 'rate_limit') {
                ui.setApiKeyStatus('gemini', 'rate_limit', 'Rate Limit');
            } else { // Covers 'invalid', 'network_error', or any other reason
                ui.setApiKeyStatus('gemini', 'invalid', 'Invalid');
            }
        }
    },

    validateAndDisplayOwmKeyStatus: async (apiKeyToValidate, onOpen = false) => {
        if (!apiKeyToValidate) {
            ui.setApiKeyStatus('owm', 'checking', 'Enter Key');
            return;
        }
        if (!onOpen) ui.setApiKeyStatus('owm', 'checking', 'Checking...');
        const validationResult = await api.validateOwmApiKey(apiKeyToValidate);
        if (validationResult.isValid) {
            ui.setApiKeyStatus('owm', 'valid', 'Valid');
        } else {
            if (validationResult.reason === 'rate_limit') {
                ui.setApiKeyStatus('owm', 'rate_limit', 'Rate Limit');
            } else { // Covers 'invalid', 'network_error', or any other reason
                ui.setApiKeyStatus('owm', 'invalid', 'Invalid');
            }
        }
    },

    // Loader Management
    incrementActiveLoaders: () => {
        app.activeLoadingOperations++;
        if (app.activeLoadingOperations === 1 && ui.startHeaderIconLoading) {
            ui.startHeaderIconLoading();
        }
        // Ensure icon is running if it was paused
        if (app.activeLoadingOperations > 0 && ui.resumeHeaderIconLoading && ui.headerIcon && ui.headerIcon.style.animationPlayState === 'paused') {
            ui.resumeHeaderIconLoading();
        }
    },

    decrementActiveLoaders: () => {
        if (app.activeLoadingOperations > 0) {
            app.activeLoadingOperations--;
        }
        if (app.activeLoadingOperations === 0 && ui.stopHeaderIconLoading) {
            ui.stopHeaderIconLoading();
            // Also ensure the refresh button is not stuck on "Waiting..." if all ops are done
            if (ui.globalRefreshButton && ui.globalRefreshButton.textContent.startsWith("Waiting")) {
                app.updateGlobalRefreshButtonVisibility();
            }
        }
    },

    promptUserToUpdate: (worker) => { // Modified to show button instead of confirm
        console.log('App Update Prompt: Service Worker update available. Showing update button.');
        app.newWorkerForUpdate = worker; // Store the worker
        if (ui.btnAppUpdate) {
            console.log('App Update Prompt: Making update button visible.');
            ui.btnAppUpdate.classList.remove('hidden');
        } else {
             console.warn('App Update Prompt: Update button element (ui.btnAppUpdate) not found.');
        }
    },

    triggerUpdate: () => { // New function to be called by the button
        console.log('App Update Trigger: Triggering update process.');
        if (app.newWorkerForUpdate) {
            console.log('App Update Trigger: Sending SKIP_WAITING to new Service Worker.');
            app.userInitiatedUpdate = true; // Set the flag
            app.newWorkerForUpdate.postMessage({ type: APP_CONSTANTS.SW_MESSAGES.SKIP_WAITING });
            if (ui.btnAppUpdate) {
                console.log('App Update Trigger: Hiding update button after trigger.');
                ui.btnAppUpdate.classList.add('hidden'); // Hide button after click
            }
        } else {
            console.warn('App Update Trigger: Update button clicked, but no new worker found (app.newWorkerForUpdate is null).');
            if (ui.btnAppUpdate) {
                 console.log('App Update Trigger: Hiding update button as no worker was found.');
                 ui.btnAppUpdate.classList.add('hidden'); // Hide if no worker anyway
            }
        }
    },

    listenForControllerChange: () => {
        let refreshing;
        navigator.serviceWorker.addEventListener('controllerchange', () => {
            console.log('SW ControllerChange: Event fired. userInitiatedUpdate:', app.userInitiatedUpdate, 'refreshing:', refreshing);
            if (refreshing) {
                console.log('SW ControllerChange: Already refreshing page, exiting.');
                return;
            }
            // Only reload if our app triggered the skipWaiting process via the button
            if (app.userInitiatedUpdate) {
                console.log('SW ControllerChange: User initiated update detected. Reloading page now.');
                window.location.reload();
                refreshing = true; // Prevent multiple reloads
            } else {
                console.log('SW ControllerChange: Controller changed, but NOT user initiated by button click. Page will use new SW on next full navigation/reload.');
                // If the update button was visible, hide it now as the update has occurred.
                if (ui.btnAppUpdate && !ui.btnAppUpdate.classList.contains('hidden')) {
                    console.log('SW ControllerChange: Hiding update button as controller changed without user trigger.');
                    ui.btnAppUpdate.classList.add('hidden');
                    app.userInitiatedUpdate = false; // Reset flag
                    app.newWorkerForUpdate = null; // Clear the stored worker
                }
            }
        });
    },

    // Getter for OWM API Key, useful for other modules
    getOwmApiKey: () => {
        return app.config.owmApiKey;
    }
    ,
    // Weather Management Functions
     fetchAndCacheWeatherData: async (location) => {
        if (!app.config.owmApiKey) {
            console.log("OpenWeatherMap API key is not set. Skipping weather fetch.");
            return null; // Return null to indicate failure/skip
        }

        const coords = await utils.extractCoordinates(location.location); // Ensure this returns { lat, lon }
        if (!coords) {
            console.warn(`Could not extract coordinates from location: ${location.location}`);
            return null; // Return null if no coords
        }

        const cachedWeather = store.getWeatherCache(location.id);
        const isWeatherStale = !cachedWeather || (Date.now() - cachedWeather.timestamp) > APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS;

        if (!cachedWeather || isWeatherStale) {
            console.log(`Fetching fresh/stale weather data for ${location.description} at ${coords.lat}, ${coords.lon}`);

            const weatherData = await api.fetchWeatherData(coords.lat, coords.lon, app.config.owmApiKey);
            if (weatherData) {
                store.saveWeatherCache(location.id, weatherData);
                console.log(`Weather data updated for ${location.description}:`, weatherData);
                return weatherData;
            } else {
                console.error(`Failed to fetch or invalid weather data for ${location.description}`);
                return null;
            }
        } else {
            console.log(`Using cached weather data for ${location.description}`);
            return cachedWeather.data;
        }
    },

    getWeatherDisplayForLocation: async (location) => {
        if (!app.config.owmApiKey) return null; // Early exit if no API key
        
        const cachedWeather = store.getWeatherCache(location.id);
        let weatherData = cachedWeather?.data;

        const coords = await utils.extractCoordinates(location.location); // Ensure this returns { lat, lon }
        if (!coords) {
            console.warn(`Could not extract coordinates from location: ${location.location}`);
            return null;
        }
        // Check staleness based on the timestamp of the original cache entry
        const isWeatherStale = !cachedWeather || (Date.now() - cachedWeather.timestamp) > APP_CONSTANTS.CACHE_EXPIRY_WEATHER_MS;

        if (!weatherData || isWeatherStale) {
            console.log(`Stale or no weather data for ${location.description}. Attempting refresh.`);
            weatherData = await app.fetchAndCacheWeatherData(location); // This will return null if fetch fails
        }

        if (weatherData && weatherData.temp && weatherData.weather && weatherData.weather.length > 0) {
            const temp = Math.round(weatherData.temp);
            const iconCode = weatherData.weather[0].icon;
            const iconUrl = `https://openweathermap.org/img/wn/${iconCode}.png`;
            return `<span class="weather-info">${temp}°F <img src="${iconUrl}" alt="Weather Icon" class="weather-icon"></span>`;
        } else {
            // If weatherData is null (fetch error, geocoding error) or doesn't have the required properties, return null.
            return null;
        }
    },

    refreshOutdatedWeather: async () => {
        if (!app.config.owmApiKey) {
            console.log("OpenWeatherMap API key is not set. Skipping weather refresh.");
            return;
        }

        console.log("Checking and refreshing outdated weather data for all locations...");
        // Individual fetchAndCacheWeatherData calls don't use the global loader
        let refreshedSomething = false;
        const weatherFetchPromises = app.locations.map(async (location) => {
            const coords = await utils.extractCoordinates(location.location);
            if (!coords) {
                console.warn(`Could not extract coordinates for weather refresh: ${location.location}`);
                return;
            }
            // fetchAndCacheWeatherData already checks for staleness internally
            const newData = await app.fetchAndCacheWeatherData(location);
            if (newData) refreshedSomething = true;
        });

        await Promise.allSettled(weatherFetchPromises);
        console.log("Weather refresh check completed.", refreshedSomething ? "Some data was updated." : "No data needed update or failed to update.");
    }

};

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
});

document.addEventListener('DOMContentLoaded', app.init);
