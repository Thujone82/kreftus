console.log("app.js loaded");

const app = {
    config: null,
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    currentLocationIdForInfoModal: null,

    init: () => {
        console.log("App initializing...");
        app.loadAndApplyAppSettings();
        app.loadLocations();
        app.loadTopics();
        app.registerServiceWorker();
        app.setupEventListeners();

        if (app.config && app.config.apiKey) {
            ui.toggleConfigButtons(true);
            app.refreshOutdatedQueries();
        } else {
            ui.toggleConfigButtons(false);
            ui.openModal('appConfigModal'); // Force app config if no API key
        }
        console.log("App initialized.");
    },

    registerServiceWorker: () => {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
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
        ui.renderLocationButtons(app.locations, app.handleOpenLocationInfo);
        console.log("Locations loaded:", app.locations);
    },

    loadTopics: () => {
        app.topics = store.getTopics();
        console.log("Topics loaded:", app.topics);
    },

    setupEventListeners: () => {
        // Main page config buttons
        ui.btnAppConfig.onclick = () => {
            ui.loadAppConfigForm(app.config); // Ensure form has latest data
            ui.openModal('appConfigModal');
        };
        ui.btnLocationsConfig.onclick = () => {
            app.currentEditingLocations = JSON.parse(JSON.stringify(app.locations)); // Deep copy for editing
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.openModal('locationConfigModal');
        };
        ui.btnInfoCollectionConfig.onclick = () => {
            app.currentEditingTopics = JSON.parse(JSON.stringify(app.topics)); // Deep copy
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList);
            ui.openModal('infoCollectionConfigModal');
        };

        // App Config Modal
        ui.saveAppConfigBtn.onclick = app.handleSaveAppSettings;

        // Location Config Modal
        ui.addLocationBtn.onclick = app.handleAddLocationToEditList;
        ui.saveLocationConfigBtn.onclick = app.handleSaveLocationConfig;
        ui.enableDragAndDrop(ui.locationsListUI, app.handleSortLocationsInEditList);


        // Info Collection Config Modal
        ui.addTopicBtn.onclick = app.handleAddTopicToEditList;
        ui.saveTopicConfigBtn.onclick = app.handleSaveTopicConfig;
        ui.enableDragAndDrop(ui.topicsListUI, app.handleSortTopicsInEditList);

        // Info Modal
        ui.refreshInfoButton.onclick = () => {
            const locationId = ui.refreshInfoButton.dataset.locationId;
            if (locationId) {
                app.handleRefreshLocationInfo(locationId);
            }
        };
        console.log("Event listeners set up.");
    },

    // --- App Config Logic ---
    handleSaveAppSettings: () => {
        const newApiKey = ui.apiKeyInput.value.trim();
        const newPrimaryColor = ui.primaryColorInput.value;
        const newBackgroundColor = ui.backgroundColorInput.value;

        if (!newApiKey) {
            ui.showAppConfigError("API Key is required.");
            return;
        }
        ui.showAppConfigError(""); // Clear error

        app.config.apiKey = newApiKey;
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config);
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.toggleConfigButtons(true); // Enable other config buttons
        ui.closeModal('appConfigModal');
        console.log("App settings saved. API Key present.");
        // Potentially trigger a refresh of all data if API key changed significantly
        // For now, assume user will manually refresh or new data will be fetched as needed.
    },

    // --- Location Config Logic ---
    handleAddLocationToEditList: () => {
        const description = ui.locationDescriptionInput.value.trim();
        const locationVal = ui.locationValueInput.value.trim();
        if (description && locationVal) {
            const newLocation = { id: utils.generateId(), description, location: locationVal };
            app.currentEditingLocations.push(newLocation);
            ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
            ui.clearInputFields([ui.locationDescriptionInput, ui.locationValueInput]);
            console.log("Location added to edit list:", newLocation);
        } else {
            alert("Both description and location value are required.");
        }
    },
    handleRemoveLocationFromEditList: (locationId) => {
        app.currentEditingLocations = app.currentEditingLocations.filter(loc => loc.id !== locationId);
        ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
        console.log("Location removed from edit list:", locationId);
    },
    handleSortLocationsInEditList: (newOrderIds) => {
        const reorderedLocations = newOrderIds.map(id => app.currentEditingLocations.find(loc => loc.id === id));
        app.currentEditingLocations = reorderedLocations.filter(loc => loc !== undefined); // Filter out any undefined if IDs mismatch
        console.log("Locations sorted in edit list.");
        // The list is visually updated by drag-and-drop, this updates the array.
        // Actual save happens on "Save Changes".
    },
    handleSaveLocationConfig: async () => {
        // Update app.locations with the sorted list from currentEditingLocations
        const newLocationIds = new Set(app.currentEditingLocations.map(l => l.id));
        const oldLocationIds = new Set(app.locations.map(l => l.id));

        app.locations = [...app.currentEditingLocations]; // Save the edited list (includes new, removed, sorted)
        store.saveLocations(app.locations);
        ui.renderLocationButtons(app.locations, app.handleOpenLocationInfo); // Update main page buttons
        ui.closeModal('locationConfigModal');
        console.log("Location configuration saved:", app.locations);

        // Process AI queries for newly added locations
        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) { // If it's a new location
                console.log(`New location added: ${location.description}. Fetching initial data.`);
                await app.fetchAndCacheAiDataForLocation(location.id, true); // Force fetch for new locations
            }
        }
    },

    // --- Topic Config Logic ---
    handleAddTopicToEditList: () => {
        const description = ui.topicDescriptionInput.value.trim();
        const aiQuery = ui.topicAiQueryInput.value.trim();
        if (description && aiQuery) {
            const newTopic = { id: utils.generateId(), description, aiQuery };
            app.currentEditingTopics.push(newTopic);
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList);
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
            console.log("Topic added to edit list:", newTopic);
        } else {
            alert("Both description and AI query are required.");
        }
    },
    handleRemoveTopicFromEditList: (topicId) => {
        app.currentEditingTopics = app.currentEditingTopics.filter(topic => topic.id !== topicId);
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList);
        console.log("Topic removed from edit list:", topicId);
    },
    handleSortTopicsInEditList: (newOrderIds) => {
        const reorderedTopics = newOrderIds.map(id => app.currentEditingTopics.find(topic => topic.id === id));
        app.currentEditingTopics = reorderedTopics.filter(topic => topic !== undefined);
        console.log("Topics sorted in edit list.");
    },
    handleSaveTopicConfig: () => {
        app.topics = [...app.currentEditingTopics];
        store.saveTopics(app.topics);
        store.flushAllAiCache(); // Flush all AI cache as topic queries might have changed
        ui.closeModal('infoCollectionConfigModal');
        console.log("Topic configuration saved. AI Cache flushed.", app.topics);
        // Optionally, re-fetch data for all locations if desired, or let it happen on demand/next refresh.
        // For now, we'll let outdated queries handle it or manual refresh.
    },

    // --- Info Modal Logic ---
    handleOpenLocationInfo: async (locationId) => {
        if (!app.config.apiKey) {
            ui.showAppConfigError("API Key is not configured. Please configure it first.");
            ui.openModal('appConfigModal');
            return;
        }

        app.currentLocationIdForInfoModal = locationId;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) {
            console.error("Location not found:", locationId);
            return;
        }

        console.log(`Opening info for location: ${location.description}`);
        ui.infoModalTitle.textContent = `${location.description} Info2Go - Loading...`;
        ui.infoModalContent.innerHTML = '<p>Fetching data...</p>';
        ui.openModal('infoModal');

        // Ensure data is fetched if not available or outdated (though refreshOutdatedQueries should handle some of this)
        // For simplicity, we can just try to fetch/refresh here too, or rely on existing cache.
        // Let's ensure data is fresh enough or fetch it.
        await app.fetchAndCacheAiDataForLocation(locationId, false); // false = don't force if not stale

        const cachedDataForLocation = {};
        let oldestTimestamp = Date.now();
        let needsOverallRefresh = false;

        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            cachedDataForLocation[topic.id] = cacheEntry;
            if (cacheEntry) {
                if (cacheEntry.timestamp < oldestTimestamp) {
                    oldestTimestamp = cacheEntry.timestamp;
                }
                if ((Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000)) { // 60 minutes
                    needsOverallRefresh = true;
                }
            } else {
                needsOverallRefresh = true; // Missing data also means refresh is needed
            }
        });

        ui.displayInfoModal(location, app.topics, cachedDataForLocation);

        if (needsOverallRefresh) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = locationId;
        } else {
            ui.refreshInfoButton.classList.add('hidden');
        }
    },

    handleRefreshLocationInfo: async (locationId) => {
        if (!locationId) locationId = app.currentLocationIdForInfoModal;
        if (!locationId) {
            console.error("No location ID available to refresh.");
            return;
        }

        const location = app.locations.find(l => l.id === locationId);
        if (!location) {
            console.error("Location not found for refresh:", locationId);
            return;
        }

        console.log(`Refreshing info for location: ${location.description}`);
        ui.infoModalTitle.textContent = `${location.description} Info2Go - Refreshing...`;
        ui.infoModalContent.innerHTML = '<p>Fetching fresh data...</p>';
        // No need to open modal, it should already be open.

        await app.fetchAndCacheAiDataForLocation(locationId, true); // true = force refresh

        // After refresh, re-display the modal content
        const cachedDataForLocation = {};
        app.topics.forEach(topic => {
            cachedDataForLocation[topic.id] = store.getAiCache(locationId, topic.id);
        });
        ui.displayInfoModal(location, app.topics, cachedDataForLocation);
        ui.refreshInfoButton.classList.add('hidden'); // Hide refresh button after successful refresh
        console.log("Info modal refreshed for:", location.description);
    },

    // --- AI Data Fetching and Caching ---
    fetchAndCacheAiDataForLocation: async (locationId, forceRefresh = false) => {
        if (!app.config.apiKey) {
            console.warn("Cannot fetch AI data: API Key is not configured.");
            // Optionally, redirect to app config or show a persistent error
            if (document.getElementById('appConfigModal').style.display !== 'block') {
                 ui.showAppConfigError("API Key is required to fetch data.");
                 ui.openModal('appConfigModal');
            }
            return;
        }

        const location = app.locations.find(l => l.id === locationId);
        if (!location) {
            console.error("Location not found for AI fetch:", locationId);
            return;
        }

        console.log(`Fetching/Caching AI data for location: ${location.description}. Force refresh: ${forceRefresh}`);
        let allSuccessful = true;

        for (const topic of app.topics) {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            const isStale = !cacheEntry || (Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000); // 60 minutes

            if (forceRefresh || isStale) {
                console.log(`Fetching for topic: ${topic.description} (Stale: ${isStale}, Force: ${forceRefresh})`);
                try {
                    const aiData = await api.fetchAiData(app.config.apiKey, location.location, topic.aiQuery);
                    store.saveAiCache(locationId, topic.id, aiData);
                    console.log(`Successfully fetched and cached data for ${location.description} - ${topic.description}`);
                } catch (error) {
                    allSuccessful = false;
                    console.error(`Failed to fetch AI data for ${location.description} - ${topic.description}:`, error.message);
                    store.saveAiCache(locationId, topic.id, `Error: ${error.message}`); // Cache the error message
                    
                    if (error.message.toLowerCase().includes("invalid api key")) {
                        ui.closeModal('infoModal'); // Close info modal if open
                        ui.showAppConfigError("Invalid API Key. Please check your configuration and save.");
                        ui.openModal('appConfigModal');
                        return; // Stop further processing for this location if API key is bad
                    }
                    // If info modal is currently open for this location, display error there
                    if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
                        // This requires ui.displayInfoModal to be able to show errors per topic
                        // For now, the cached error will be shown on next displayInfoModal call
                    }
                }
            } else {
                console.log(`Skipping fetch for topic: ${topic.description} (Data is fresh)`);
            }
        }
        return allSuccessful;
    },

    refreshOutdatedQueries: async () => {
        if (!app.config.apiKey) {
            console.log("Skipping outdated query refresh: No API key.");
            return;
        }
        console.log("Checking for outdated AI queries...");
        const sixtyMinutesAgo = Date.now() - (60 * 60 * 1000);

        for (const location of app.locations) {
            let needsRefreshForLocation = false;
            if (app.topics.length === 0) continue; // No topics to check against

            for (const topic of app.topics) {
                const cacheEntry = store.getAiCache(location.id, topic.id);
                if (!cacheEntry || cacheEntry.timestamp < sixtyMinutesAgo) {
                    needsRefreshForLocation = true;
                    break; 
                }
            }

            if (needsRefreshForLocation) {
                console.log(`Data for location ${location.description} is outdated or missing. Refreshing...`);
                // Don't await here to allow background refresh for multiple locations
                app.fetchAndCacheAiDataForLocation(location.id, true).then(success => {
                    if (success) {
                        console.log(`Background refresh for ${location.description} completed.`);
                        // If this location's info modal is open, update it
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === location.id) {
                            app.handleOpenLocationInfo(location.id); // Re-opens and re-renders with fresh data
                        }
                    } else {
                        console.warn(`Background refresh for ${location.description} encountered errors.`);
                    }
                });
            } else {
                console.log(`Data for location ${location.description} is fresh.`);
            }
        }
    },
};

// Global error handler for unhandled promise rejections (e.g. network errors during fetch)
window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
    // You could display a generic error message to the user here if desired
    // For example: ui.showGlobalError("An unexpected error occurred. Please try again.");
});


document.addEventListener('DOMContentLoaded', app.init);


/**
 * Small helper for drag and drop reordering of config lists.
 * The ui.enableDragAndDrop calls the provided callback with an array of IDs in the new order.
 * This function then reorders the actual data array (app.currentEditingLocations or app.currentEditingTopics).
 */
function reorderArrayByIds(originalArray, idOrderArray) {
    const itemMap = new Map(originalArray.map(item => [item.id, item]));
    return idOrderArray.map(id => itemMap.get(id)).filter(item => item !== undefined);
}

// Modify drag and drop callbacks in setupEventListeners to use the reorderArrayByIds
app.setupEventListeners = () => { // Redefine to include updated drag-drop handlers
    // ... (keep existing event listeners)
     // Main page config buttons
    ui.btnAppConfig.onclick = () => {
        ui.loadAppConfigForm(app.config); // Ensure form has latest data
        ui.openModal('appConfigModal');
    };
    ui.btnLocationsConfig.onclick = () => {
        app.currentEditingLocations = JSON.parse(JSON.stringify(app.locations)); // Deep copy for editing
        ui.renderConfigList(app.currentEditingLocations, ui.locationsListUI, 'location', app.handleRemoveLocationFromEditList);
        ui.openModal('locationConfigModal');
    };
    ui.btnInfoCollectionConfig.onclick = () => {
        app.currentEditingTopics = JSON.parse(JSON.stringify(app.topics)); // Deep copy
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList);
        ui.openModal('infoCollectionConfigModal');
    };

    // App Config Modal
    ui.saveAppConfigBtn.onclick = app.handleSaveAppSettings;

    // Location Config Modal
    ui.addLocationBtn.onclick = app.handleAddLocationToEditList;
    ui.saveLocationConfigBtn.onclick = app.handleSaveLocationConfig;
    ui.enableDragAndDrop(ui.locationsListUI, (newOrderIds) => {
        app.currentEditingLocations = reorderArrayByIds(app.currentEditingLocations, newOrderIds);
        // No need to re-render here, drag-drop already updated UI. List is saved on "Save Changes".
        console.log("Locations reordered in edit list (data array updated).");
    });

    // Info Collection Config Modal
    ui.addTopicBtn.onclick = app.handleAddTopicToEditList;
    ui.saveTopicConfigBtn.onclick = app.handleSaveTopicConfig;
    ui.enableDragAndDrop(ui.topicsListUI, (newOrderIds) => {
        app.currentEditingTopics = reorderArrayByIds(app.currentEditingTopics, newOrderIds);
        console.log("Topics reordered in edit list (data array updated).");
    });

    // Info Modal
    ui.refreshInfoButton.onclick = () => {
        const locationId = ui.refreshInfoButton.dataset.locationId || app.currentLocationIdForInfoModal;
        if (locationId) {
            app.handleRefreshLocationInfo(locationId);
        }
    };
    console.log("Event listeners set up (v2).");
};