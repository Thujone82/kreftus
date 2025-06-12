console.log("app.js loaded");

const app = {
    config: null,
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    editingTopicId: null, 
    currentLocationIdForInfoModal: null,
    fetchingStatus: {}, // { locationId: boolean }

    init: () => {
        console.log("App initializing...");
        app.loadAndApplyAppSettings();
        app.loadLocations(); 
        app.loadTopics();    
        app.registerServiceWorker();
        app.setupEventListeners(); 

        if (app.config && app.config.apiKey) {
            ui.toggleConfigButtons(true);
            app.refreshOutdatedQueries(); // This will also update button states and global refresh visibility
        } else {
            ui.toggleConfigButtons(false);
            ui.openModal('appConfigModal'); 
        }
        app.updateGlobalRefreshButtonVisibility(); // Initial check
        console.log("App initialized.");
    },

    registerServiceWorker: () => {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/nfo/sw.js', { scope: '/nfo/' })
                .then(registration => console.log('Service Worker registered with scope:', registration.scope))
                .catch(error => console.error('Service Worker registration failed:', error));
        }
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
            ui.openModal('appConfigModal');
        };
        ui.btnLocationsConfig.onclick = () => {
            app.currentEditingLocations = JSON.parse(JSON.stringify(app.locations));
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.openModal('locationConfigModal');
        };
        ui.btnInfoCollectionConfig.onclick = () => {
            app.currentEditingTopics = JSON.parse(JSON.stringify(app.topics));
            app.editingTopicId = null; 
            ui.addTopicBtn.textContent = 'Add Topic'; 
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]); 
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
            ui.openModal('infoCollectionConfigModal');
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
            if (locationId) {
                app.handleRefreshLocationInfo(locationId);
            }
        };
        if (ui.globalRefreshButton) {
            ui.globalRefreshButton.onclick = () => {
                console.log("Global refresh triggered.");
                app.refreshOutdatedQueries(true); // Pass true to force refresh all stale/error items
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
            ui.showAppConfigError("API Key is required.");
            return;
        }
        ui.showAppConfigError("");

        app.config.apiKey = newApiKey;
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config);
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.toggleConfigButtons(true);
        ui.closeModal('appConfigModal');
        console.log("App settings saved. API Key present.");
        app.refreshOutdatedQueries(); // Refresh data if API key was just added/changed
    },

    handleAddLocationToEditList: () => {
        const description = ui.locationDescriptionInput.value.trim();
        const locationVal = ui.locationValueInput.value.trim();
        if (description && locationVal) {
            const newLocation = { id: utils.generateId(), description, location: locationVal };
            app.currentEditingLocations.push(newLocation);
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.clearInputFields([ui.locationDescriptionInput, ui.locationValueInput]);
        } else {
            alert("Both description and location value are required.");
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
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined); // This will also call updateGlobalRefreshButtonVisibility
        ui.closeModal('locationConfigModal');
        console.log("Location configuration saved:", app.locations);

        let newLocationsFetched = false;
        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) {
                console.log(`New location added: ${location.description}. Fetching initial data.`);
                await app.fetchAndCacheAiDataForLocation(location.id, true);
                newLocationsFetched = true;
            }
        }
        // If new locations were fetched, their individual fetches updated UI.
        // Now, run refreshOutdatedQueries for existing locations.
        // If no new locations, still run it to ensure everything is checked.
        app.refreshOutdatedQueries();
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
            alert("Both description and AI query are required.");
            return;
        }

        if (app.editingTopicId) { 
            const topicToUpdate = app.currentEditingTopics.find(t => t.id === app.editingTopicId);
            if (topicToUpdate) {
                topicToUpdate.description = description;
                topicToUpdate.aiQuery = aiQuery;
            }
            app.editingTopicId = null;
            ui.addTopicBtn.textContent = 'Add Topic';
        } else { 
            const newTopic = { id: utils.generateId(), description, aiQuery };
            app.currentEditingTopics.push(newTopic);
        }
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
        ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
    },
    handleRemoveTopicFromEditList: (topicId) => {
        app.currentEditingTopics = app.currentEditingTopics.filter(topic => topic.id !== topicId);
        if (app.editingTopicId === topicId) {
            app.editingTopicId = null;
            ui.addTopicBtn.textContent = 'Add Topic';
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
        }
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
    },
    handleSaveTopicConfig: () => {
        app.topics = [...app.currentEditingTopics];
        store.saveTopics(app.topics);
        store.flushAllAiCache(); // This means all data will be re-fetched
        ui.closeModal('infoCollectionConfigModal');
        app.editingTopicId = null;
        ui.addTopicBtn.textContent = 'Add Topic';

        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined); // This will also call updateGlobalRefreshButtonVisibility
        console.log("Topic configuration saved. AI Cache flushed.", app.topics);
        app.refreshOutdatedQueries(true); // Force refresh all since cache was flushed
    },

    handleLocationButtonClick: (locationId) => {
        if (app.topics && app.topics.length > 0) {
            app.handleOpenLocationInfo(locationId);
        } else {
            alert("Please define an Info Structure before viewing location information.");
            ui.openModal('infoCollectionConfigModal');
        }
    },
    handleOpenLocationInfo: async (locationId) => {
        if (!app.config.apiKey) {
            ui.showAppConfigError("API Key is not configured. Please configure it first.");
            ui.openModal('appConfigModal');
            return;
        }
        app.currentLocationIdForInfoModal = locationId;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;

        ui.infoModalTitle.textContent = `${location.description} Info2Go - Loading...`;
        ui.infoModalContent.innerHTML = '<p>Fetching data...</p>';
        ui.openModal('infoModal');

        // Fetch data if needed (stale or forced), then display
        await app.fetchAndCacheAiDataForLocation(locationId, false); 

        const cachedDataForLocation = {};
        let needsOverallRefreshForModal = false;
        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            cachedDataForLocation[topic.id] = cacheEntry;
            if (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000) ||
                (cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:'))) {
                needsOverallRefreshForModal = true;
            }
        });
        ui.displayInfoModal(location, app.topics, cachedDataForLocation);
        if (needsOverallRefreshForModal) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = locationId;
        } else {
            ui.refreshInfoButton.classList.add('hidden');
        }
    },

    handleRefreshLocationInfo: async (locationId) => {
        if (!locationId) locationId = app.currentLocationIdForInfoModal;
        if (!locationId) return;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;

        ui.infoModalTitle.textContent = `${location.description} Info2Go - Refreshing...`;
        ui.infoModalContent.innerHTML = '<p>Fetching fresh data...</p>';
        await app.fetchAndCacheAiDataForLocation(locationId, true); // Force refresh
        // Re-open/re-render the modal content
        app.handleOpenLocationInfo(locationId);
    },

    fetchAndCacheAiDataForLocation: async (locationId, forceRefresh = false) => {
        if (!app.config.apiKey) {
            console.warn("Cannot fetch AI data: API Key is not configured.");
            if (document.getElementById('appConfigModal').style.display !== 'block') {
                 ui.showAppConfigError("API Key is required to fetch data.");
                 ui.openModal('appConfigModal');
            }
            return false; 
        }
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return false;

        app.fetchingStatus[locationId] = true;
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);

        console.log(`Fetching/Caching AI data for location: ${location.description}. Force refresh: ${forceRefresh}`);
        let allSuccessful = true;

        for (const topic of app.topics) {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);
            const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');

            if (forceRefresh || isStale || hasError) { // Also refresh if there was a previous error
                console.log(`Fetching for topic: ${topic.description} (Stale: ${isStale}, Force: ${forceRefresh}, Error: ${hasError})`);
                try {
                    const aiData = await api.fetchAiData(app.config.apiKey, location.location, topic.aiQuery);
                    store.saveAiCache(locationId, topic.id, aiData);
                } catch (error) {
                    allSuccessful = false;
                    console.error(`Failed to fetch AI data for ${location.description} - ${topic.description}:`, error.message);
                    store.saveAiCache(locationId, topic.id, `Error: ${error.message}`);
                    if (error.message.toLowerCase().includes("invalid api key")) {
                        ui.closeModal('infoModal');
                        ui.showAppConfigError("Invalid API Key. Please check your configuration and save.");
                        ui.openModal('appConfigModal');
                        // Clean up fetching status for this location as we are stopping
                        delete app.fetchingStatus[locationId];
                        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
                        app.updateGlobalRefreshButtonVisibility();
                        return false; 
                    }
                }
            }
        }
        delete app.fetchingStatus[locationId];
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.updateGlobalRefreshButtonVisibility();
        return allSuccessful;
    },

    refreshOutdatedQueries: async (forceAllStale = false) => {
        if (!app.config.apiKey) {
            console.log("Skipping outdated query refresh: No API key.");
            return;
        }
        console.log("Checking for outdated AI queries...", forceAllStale ? "(Forcing all stale)" : "");
        const sixtyMinutesAgo = Date.now() - (60 * 60 * 1000);
        let anyFetchesInitiated = false;

        for (const location of app.locations) {
            let needsRefreshForLocation = false;
            if (app.topics.length === 0) continue;

            if (forceAllStale) { // If global refresh button or topic save triggered
                needsRefreshForLocation = true;
            } else {
                for (const topic of app.topics) {
                    const cacheEntry = store.getAiCache(location.id, topic.id);
                    if (!cacheEntry || (cacheEntry.timestamp || 0) < sixtyMinutesAgo ||
                        (cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:'))) {
                        needsRefreshForLocation = true;
                        break; 
                    }
                }
            }

            if (needsRefreshForLocation) {
                anyFetchesInitiated = true;
                console.log(`Data for location ${location.description} needs refresh. Refreshing...`);
                // fetchAndCacheAiDataForLocation will update UI for its specific button and global button
                app.fetchAndCacheAiDataForLocation(location.id, true).then(success => { // true to force refresh this location
                    if (success) {
                        console.log(`Background refresh for ${location.description} completed.`);
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === location.id) {
                            app.handleOpenLocationInfo(location.id); 
                        }
                    } else {
                        console.warn(`Background refresh for ${location.description} encountered errors.`);
                    }
                });
            }
        }
        if (!anyFetchesInitiated) { // If no fetches were started, ensure UI is up-to-date
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
                if (app.fetchingStatus && app.fetchingStatus[location.id]) {
                    // If actively fetching, consider it as needing attention for the button
                    // Or, decide if fetching state should hide the button. For now, let's assume fetching means it might become stale.
                    // Let's refine: if it's fetching, it's not yet "stale" for the purpose of this button.
                    // The button is for user-initiated refresh of already settled stale/error states.
                    continue;
                }
                for (const topic of app.topics) {
                    const cacheEntry = store.getAiCache(location.id, topic.id);
                    const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
                    const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);

                    if (hasError || isStale) {
                        anyStaleOrError = true;
                        break;
                    }
                }
                if (anyStaleOrError) break;
            }
        }
        if (anyStaleOrError) {
            ui.globalRefreshButton.classList.remove('hidden');
        } else {
            ui.globalRefreshButton.classList.add('hidden');
        }
    }
};

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
});

document.addEventListener('DOMContentLoaded', app.init);
