console.log("app.js loaded");

const APP_CONSTANTS = {
    CACHE_EXPIRY_MS: 60 * 60 * 1000, // 1 hour
    MODAL_IDS: {
        APP_CONFIG: 'appConfigModal',
        LOCATION_CONFIG: 'locationConfigModal',
        INFO_COLLECTION_CONFIG: 'infoCollectionConfigModal',
        INFO: 'infoModal'
    },
    SW_MESSAGES: {
        SKIP_WAITING: 'SKIP_WAITING'
    }
};

const app = {
    config: null,
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    editingTopicId: null, 
    currentLocationIdForInfoModal: null,
    fetchingStatus: {}, 
    initialEditingLocationsString: '', // For unsaved changes check
    initialEditingTopicsString: '',    // For unsaved changes check

    init: () => {
        console.log("App initializing...");
        app.loadAndApplyAppSettings();
        app.loadLocations(); 
        app.loadTopics();    
        app.registerServiceWorker();
        app.setupEventListeners(); 

        if (app.config && app.config.apiKey) {
            ui.toggleConfigButtons(true);
            // Automatic refresh on init is removed. User will use the button.
        } else {
            ui.toggleConfigButtons(false);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG); 
        }
        app.updateGlobalRefreshButtonVisibility(); // Initial check for button visibility
        console.log("App initialized.");
    },

    registerServiceWorker: () => {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/nfo/sw.js', { scope: '/nfo/' })
                .then(registration => {
                    console.log('Service Worker registered with scope:', registration.scope);

                    // Check for an existing waiting worker on page load.
                    // This handles cases where a new SW was installed, but the user closed all tabs
                    // before it could activate, and then reopened the app.
                    if (registration.waiting) {
                        app.promptUserToUpdate(registration.waiting);
                    }

                    // Listen for new worker installing
                    registration.onupdatefound = () => {
                        console.log('New service worker found installing.');
                        const installingWorker = registration.installing;
                        if (installingWorker) {
                            installingWorker.onstatechange = () => {
                                console.log('Service worker state changed:', installingWorker.state);
                                if (installingWorker.state === 'installed' && navigator.serviceWorker.controller) {
                                    // New worker is installed and waiting (because there's an active controller)
                                    if (registration.waiting) {
                                        app.promptUserToUpdate(registration.waiting);
                                    }
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
        app.config = store.getAppSettings();
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
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
    },

    setupEventListeners: () => {
        ui.btnAppConfig.onclick = () => {
            ui.loadAppConfigForm(app.config); 
            if (ui.appConfigError) ui.appConfigError.textContent = ''; // Clear previous errors
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
        };
        ui.btnLocationsConfig.onclick = () => {
            app.currentEditingLocations = JSON.parse(JSON.stringify(app.locations));
            app.initialEditingLocationsString = JSON.stringify(app.currentEditingLocations); // Store initial state
            if (ui.locationConfigError) ui.locationConfigError.textContent = ''; // Clear previous errors
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG);
        };
        ui.btnInfoCollectionConfig.onclick = () => {
            app.currentEditingTopics = JSON.parse(JSON.stringify(app.topics));
            app.initialEditingTopicsString = JSON.stringify(app.currentEditingTopics); // Store initial state
            app.editingTopicId = null; 
            ui.addTopicBtn.textContent = 'Add Topic'; 
            if (ui.topicConfigError) ui.topicConfigError.textContent = ''; // Clear previous errors
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]); 
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
            ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG);

        };

        ui.saveAppConfigBtn.onclick = app.handleSaveAppSettings;
        ui.addLocationBtn.onclick = app.handleAddLocationToEditList;
        ui.saveLocationConfigBtn.onclick = app.handleSaveLocationConfig;
        ui.enableDragAndDrop(ui.locationsListUI, (newOrderIds) => {
            app.currentEditingLocations = app.reorderArrayByIds(app.currentEditingLocations, newOrderIds);
            // Note: Dragging itself is a change, app.hasUnsavedChanges will detect this.
        });

        ui.addTopicBtn.onclick = app.handleAddOrUpdateTopicInEditList;
        ui.saveTopicConfigBtn.onclick = app.handleSaveTopicConfig;
        ui.enableDragAndDrop(ui.topicsListUI, (newOrderIds) => {
            app.currentEditingTopics = app.reorderArrayByIds(app.currentEditingTopics, newOrderIds);
            // Note: Dragging itself is a change, app.hasUnsavedChanges will detect this.
        });

        ui.refreshInfoButton.onclick = () => {
            const locationId = ui.refreshInfoButton.dataset.locationId || app.currentLocationIdForInfoModal;
            if (locationId) app.handleRefreshLocationInfo(locationId);
        };
        if (ui.globalRefreshButton) {
            ui.globalRefreshButton.onclick = () => {
                console.log("Global refresh triggered.");
                app.refreshOutdatedQueries(false); // Changed to false for targeted refresh
            };
        }
        console.log("Event listeners set up.");
    },

    reorderArrayByIds: (originalArray, idOrderArray) => {
        const itemMap = new Map(originalArray.map(item => [item.id, item]));
        return idOrderArray.map(id => itemMap.get(id)).filter(item => item !== undefined);
    },

    handleSaveAppSettings: () => {
        const newApiKey = ui.apiKeyInput.value.trim();
        const newPrimaryColor = ui.primaryColorInput.value;
        const newBackgroundColor = ui.backgroundColorInput.value;

        if (!newApiKey) {
            if(ui.appConfigError) ui.appConfigError.textContent = "API Key is required.";
            if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.remove('hidden'); 
            return;
        }
        if(ui.appConfigError) ui.appConfigError.textContent = "";
        if (ui.getApiKeyLinkContainer) ui.getApiKeyLinkContainer.classList.add('hidden'); 

        app.config.apiKey = newApiKey;
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config);
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor); 
        ui.toggleConfigButtons(true);
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
        console.log("App settings saved. API Key present.");
        app.updateGlobalRefreshButtonVisibility(); 
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
        app.initialEditingLocationsString = JSON.stringify(app.locations); // Update baseline after save
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG);
        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) {
                console.log(`New location added: ${location.description}. Fetching initial data.`);
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

        // Identify topics with changed queries
        app.currentEditingTopics.forEach(currentTopic => {
            const prevTopic = previousTopics.get(currentTopic.id);
            if (prevTopic && prevTopic.aiQuery !== currentTopic.aiQuery) {
                queryChangedTopicIds.add(currentTopic.id);
            }
            // New topics will naturally be fetched as they have no cache.
        });

        // Identify deleted topics
        previousTopics.forEach(prevTopic => {
            if (!currentEditingTopicsMap.has(prevTopic.id)) {
                deletedTopicIds.add(prevTopic.id);
            }
        });

        // Selectively invalidate cache
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
        app.initialEditingTopicsString = JSON.stringify(app.topics); // Update baseline after save
        ui.closeModal(APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG);
        app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined); 
        app.updateGlobalRefreshButtonVisibility(); 

        // Determine which topics were newly added or had their AI query changed.
        const newOrChangedQueryTopicIds = new Set();
        app.currentEditingTopics.forEach(currentTopic => {
            const prevTopic = previousTopics.get(currentTopic.id);
            if (!prevTopic) { // New topic
                newOrChangedQueryTopicIds.add(currentTopic.id);
            } else if (prevTopic.aiQuery !== currentTopic.aiQuery) { // Query changed
                newOrChangedQueryTopicIds.add(currentTopic.id);
            }
        });

        if (newOrChangedQueryTopicIds.size > 0) {
            const idsToRefresh = Array.from(newOrChangedQueryTopicIds);
            console.log("Info Structure updated. Triggering specific refresh for EDITED/NEW items:", idsToRefresh);
            app.locations.forEach(location => {
                // Call fetchAndCacheAiDataForLocation to refresh only these specific topics for this location.
                // The second argument (forceRefreshGeneral) is false, as we are not doing a general force refresh.
                // The third argument provides the specific list of topic IDs to target.
                app.fetchAndCacheAiDataForLocation(location.id, false, idsToRefresh);
            });
        } else {
            console.log("Info Structure updated. No AI queries changed, no new topics. No specific refresh needed from this operation.");
            // If only reordering or deletion of non-queried topics occurred,
            // the global refresh button visibility is still updated.
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

        // Initial loading message
        if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go - Loading...`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = '<p>Accessing stored data...</p>'; // More accurate initial message
        ui.openModal(APP_CONSTANTS.MODAL_IDS.INFO);

        // Directly gather all cached data for the location without an initial fetch.
        // Stale data will be displayed as is.
        const cachedDataForLocation = {};
        let needsOverallRefreshForModal = false;

        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            cachedDataForLocation[topic.id] = cacheEntry;

            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
            const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');

            if (isStale || hasError) {
                needsOverallRefreshForModal = true;
            }
        });

        // Display the gathered data (fresh, stale, or error)
        ui.displayInfoModal(location, app.topics, cachedDataForLocation); // This will now set the final title and content
        
        // Show/hide the modal's refresh button based on the state of the displayed data
        if (needsOverallRefreshForModal && ui.refreshInfoButton) { 
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
        if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go - Refreshing...`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = '<p>Fetching fresh data...</p>';
        await app.fetchAndCacheAiDataForLocation(locationId, false); // Only refresh stale/error topics
        // After fetching, call handleOpenLocationInfo again to re-render with fresh data
        app.handleOpenLocationInfo(locationId); 
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

        app.fetchingStatus[locationId] = true;
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        
        console.log(`Fetching/Caching AI data for location: ${location.description}. GeneralForce: ${forceRefreshGeneral}, SpecificIDs: ${specificTopicIdsToForce ? specificTopicIdsToForce.join(', ') : 'None'}`);
        
        let completedCount = 0;

        const topicsToFetch = app.topics.filter(topic => {
            if (specificTopicIdsToForce && specificTopicIdsToForce.length > 0) {
                // If a specific list is provided, only consider topics in that list for fetching,
                // and they are always fetched regardless of cache status.
                return specificTopicIdsToForce.includes(topic.id);
            } else {
                // No specific list, use general logic for all topics of the location.
                const cacheEntry = store.getAiCache(locationId, topic.id);
                const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
                const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                return forceRefreshGeneral || isStale || hasError;
            }
        });


        const totalTopicsToFetch = topicsToFetch.length;

        // Update modal title immediately if it's open for this location
        if (document.getElementById(APP_CONSTANTS.MODAL_IDS.INFO).style.display === 'block' && app.currentLocationIdForInfoModal === locationId && totalTopicsToFetch > 0) {
            app.updateInfoModalLoadingMessage(location.description, completedCount, totalTopicsToFetch);
        }
        
        const fetchExecutionPromises = [];
        let anInvalidKeyErrorOccurred = false;


        if (totalTopicsToFetch === 0) {
            console.log(`No topics to fetch for ${location.description}.`);
            delete app.fetchingStatus[locationId];
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            app.updateGlobalRefreshButtonVisibility();
            // If the modal is open for this location, ensure its title is set correctly,
            // but avoid a recursive call to handleOpenLocationInfo. //TODO: Check this logic
            // The original caller of fetchAndCacheAiDataForLocation (handleOpenLocationInfo) will handle the UI update.
            if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId && ui.infoModalTitle) {
                 if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go`;
            }
            return true; 
        }

        for (const topic of topicsToFetch) {
            console.log(`Preparing to fetch for topic: ${topic.description}`);
            const promise = api.fetchAiData(app.config.apiKey, location.location, topic.aiQuery)
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

        const results = await Promise.all(fetchExecutionPromises); // Wait for all fetches (and their finally blocks)
        let allIndividualFetchesSuccessful = true;
        results.forEach(result => {
            if (result.status === 'rejected') {
                allIndividualFetchesSuccessful = false;
                // anInvalidKeyErrorOccurred is already handled in the catch block
            }
        });

        delete app.fetchingStatus[locationId];
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.updateGlobalRefreshButtonVisibility();

        if (anInvalidKeyErrorOccurred) {
            if(ui.infoModal) ui.closeModal(APP_CONSTANTS.MODAL_IDS.INFO); 
            if(ui.appConfigError) ui.appConfigError.textContent = "Invalid API Key. Please check your configuration and save.";
            ui.openModal(APP_CONSTANTS.MODAL_IDS.APP_CONFIG);
            return false; 
        }

        // If the modal is still open after all fetches, ensure its content is fully updated.
        if (document.getElementById(APP_CONSTANTS.MODAL_IDS.INFO).style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
            const currentCachedData = {};
            app.topics.forEach(t => {
                currentCachedData[t.id] = store.getAiCache(locationId, t.id);
            });
            ui.displayInfoModal(location, app.topics, currentCachedData);
        }
        
        return allIndividualFetchesSuccessful;
    },
    
    updateInfoModalLoadingMessage: (locationDescription, completed, total) => {
        if (ui.infoModalTitle) {
            if (total === 0) { 
                // Title will be set by displayInfoModal in handleOpenLocationInfo
            } else {
                ui.infoModalTitle.textContent = `${locationDescription} nfo2Go - Fetching (${completed}/${total})`;
                if (completed === total){
                     setTimeout(() => {
                        // Check if modal is still open for this location before attempting to update title
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal && app.locations.find(l=>l.id === app.currentLocationIdForInfoModal)?.description === locationDescription) {
                           // The calling function (handleOpenLocationInfo) will handle the final display update.
                        }
                    }, 50); // Short delay for the last progress update to be visible
                }
            }
        }
    },

    refreshOutdatedQueries: async (forceAllStale = false) => {
        if (!app.config.apiKey) return;
        const sixtyMinutesAgo = Date.now() - APP_CONSTANTS.CACHE_EXPIRY_MS;
        let anyFetchesInitiated = false;
        for (const location of app.locations) {
            let needsRefreshForLocation = false;
            if (app.topics.length === 0) continue;
            if (forceAllStale) needsRefreshForLocation = true;
            else {
                for (const topic of app.topics) {
                    const cacheEntry = store.getAiCache(location.id, topic.id);
                    if (!cacheEntry || (cacheEntry.timestamp || 0) < sixtyMinutesAgo ||
                        (cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:'))) {
                        needsRefreshForLocation = true; break; 
                    }
                }
            }
            if (needsRefreshForLocation) {
                anyFetchesInitiated = true;
                // Don't await here, let them run in parallel across locations too
                app.fetchAndCacheAiDataForLocation(location.id, forceAllStale, null).then(success => { // Pass null for specificTopicIds
                    if (!success) {
                         console.warn(`Background refresh for ${location.description} encountered errors or an API key issue.`);
                    }
                });
            }
        }
        if (!anyFetchesInitiated) { // If no fetches were started by this loop, ensure UI is up-to-date
            const areTopicsDefined = app.topics && app.topics.length > 0;
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            app.updateGlobalRefreshButtonVisibility();
        }
    },

    updateGlobalRefreshButtonVisibility: () => {
        if (!ui.globalRefreshButton) return;
        let anyStaleOrError = false;
        if (app.topics.length > 0) {
            for (const location of app.locations) {
                if (app.fetchingStatus && app.fetchingStatus[location.id]) continue;
                for (const topic of app.topics) {
                    const cacheEntry = store.getAiCache(location.id, topic.id);
                    const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                    const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS;
                    if (hasError || isStale) { anyStaleOrError = true; break; }
                }
                if (anyStaleOrError) break;
            }
        }
        if (anyStaleOrError) ui.globalRefreshButton.classList.remove('hidden');
        else ui.globalRefreshButton.classList.add('hidden');
    }
    ,

    hasUnsavedChanges: (modalId) => {
        if (modalId === APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG) {
            return JSON.stringify(app.currentEditingLocations) !== app.initialEditingLocationsString;
        }
        if (modalId === APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG) {
            return JSON.stringify(app.currentEditingTopics) !== app.initialEditingTopicsString;
        }
        return false; // Not a settings modal we're tracking
    },

    // --- Service Worker Update Logic ---
    
    promptUserToUpdate: (worker) => {
        // For a real application, you would use a more sophisticated UI element
        // like a banner or a toast notification instead of a confirm dialog.
        console.log('Prompting user to update to new version.');
        if (confirm("A new version of nfo2Go is available. Refresh to update?")) {
            worker.postMessage({ type: APP_CONSTANTS.SW_MESSAGES.SKIP_WAITING });
        }
    },

    listenForControllerChange: () => {
        let refreshing;
        navigator.serviceWorker.addEventListener('controllerchange', () => {
            if (refreshing) return;
            console.log('Controller changed. New service worker has activated. Reloading page.');
            window.location.reload();
            refreshing = true;
        });
    }
};

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
});

document.addEventListener('DOMContentLoaded', app.init);
