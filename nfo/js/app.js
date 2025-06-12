console.log("app.js loaded");

const app = {
    config: null,
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    editingTopicId: null, // To keep track of the topic being edited
    currentLocationIdForInfoModal: null,

    init: () => {
        console.log("App initializing...");
        app.loadAndApplyAppSettings();
        app.loadLocations();
        app.loadTopics();
        app.registerServiceWorker();
        app.setupEventListeners(); // Single setup

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
            app.editingTopicId = null; // Reset editing state
            ui.addTopicBtn.textContent = 'Add Topic'; // Reset button text
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]); // Clear form
            ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
            ui.openModal('infoCollectionConfigModal');
        };

        // App Config Modal
        ui.saveAppConfigBtn.onclick = app.handleSaveAppSettings;

        // Location Config Modal
        ui.addLocationBtn.onclick = app.handleAddLocationToEditList;
        ui.saveLocationConfigBtn.onclick = app.handleSaveLocationConfig;
        ui.enableDragAndDrop(ui.locationsListUI, (newOrderIds) => {
            app.currentEditingLocations = app.reorderArrayByIds(app.currentEditingLocations, newOrderIds);
            console.log("Locations reordered in edit list (data array updated).");
        });

        // Info Collection Config Modal
        ui.addTopicBtn.onclick = app.handleAddOrUpdateTopicInEditList; // Changed from handleAddTopicToEditList
        ui.saveTopicConfigBtn.onclick = app.handleSaveTopicConfig;
        ui.enableDragAndDrop(ui.topicsListUI, (newOrderIds) => {
            app.currentEditingTopics = app.reorderArrayByIds(app.currentEditingTopics, newOrderIds);
            console.log("Topics reordered in edit list (data array updated).");
        });

        // Info Modal
        ui.refreshInfoButton.onclick = () => {
            const locationId = ui.refreshInfoButton.dataset.locationId || app.currentLocationIdForInfoModal;
            if (locationId) {
                app.handleRefreshLocationInfo(locationId);
            }
        };
        console.log("Event listeners set up.");
    },

    reorderArrayByIds: (originalArray, idOrderArray) => {
        const itemMap = new Map(originalArray.map(item => [item.id, item]));
        return idOrderArray.map(id => itemMap.get(id)).filter(item => item !== undefined);
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
        ui.showAppConfigError("");

        app.config.apiKey = newApiKey;
        app.config.primaryColor = newPrimaryColor;
        app.config.backgroundColor = newBackgroundColor;

        store.saveAppSettings(app.config);
        ui.applyTheme(app.config.primaryColor, app.config.backgroundColor);
        ui.toggleConfigButtons(true);
        ui.closeModal('appConfigModal');
        console.log("App settings saved. API Key present.");
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
    handleSaveLocationConfig: async () => {
        const oldLocationIds = new Set(app.locations.map(l => l.id));
        app.locations = [...app.currentEditingLocations];
        store.saveLocations(app.locations);
        ui.renderLocationButtons(app.locations, app.handleOpenLocationInfo);
        ui.closeModal('locationConfigModal');
        console.log("Location configuration saved:", app.locations);

        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) {
                console.log(`New location added: ${location.description}. Fetching initial data.`);
                await app.fetchAndCacheAiDataForLocation(location.id, true);
            }
        }
    },

    // --- Topic Config Logic ---
    prepareEditTopic: (topicId) => {
        const topicToEdit = app.currentEditingTopics.find(t => t.id === topicId);
        if (topicToEdit) {
            app.editingTopicId = topicId;
            ui.topicDescriptionInput.value = topicToEdit.description;
            ui.topicAiQueryInput.value = topicToEdit.aiQuery;
            ui.addTopicBtn.textContent = 'Update Topic';
            ui.topicDescriptionInput.focus(); // Focus on the first field
        }
    },
    handleAddOrUpdateTopicInEditList: () => {
        const description = ui.topicDescriptionInput.value.trim();
        const aiQuery = ui.topicAiQueryInput.value.trim();

        if (!description || !aiQuery) {
            alert("Both description and AI query are required.");
            return;
        }

        if (app.editingTopicId) { // Update existing
            const topicToUpdate = app.currentEditingTopics.find(t => t.id === app.editingTopicId);
            if (topicToUpdate) {
                topicToUpdate.description = description;
                topicToUpdate.aiQuery = aiQuery;
            }
            console.log("Topic updated in edit list:", app.editingTopicId);
            app.editingTopicId = null;
            ui.addTopicBtn.textContent = 'Add Topic';
        } else { // Add new
            const newTopic = { id: utils.generateId(), description, aiQuery };
            app.currentEditingTopics.push(newTopic);
            console.log("Topic added to edit list:", newTopic);
        }

        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
        ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
    },
    handleRemoveTopicFromEditList: (topicId) => {
        app.currentEditingTopics = app.currentEditingTopics.filter(topic => topic.id !== topicId);
        // If the removed topic was being edited, reset the form
        if (app.editingTopicId === topicId) {
            app.editingTopicId = null;
            ui.addTopicBtn.textContent = 'Add Topic';
            ui.clearInputFields([ui.topicDescriptionInput, ui.topicAiQueryInput]);
        }
        ui.renderConfigList(app.currentEditingTopics, ui.topicsListUI, 'topic', app.handleRemoveTopicFromEditList, app.prepareEditTopic);
        console.log("Topic removed from edit list:", topicId);
    },
    handleSaveTopicConfig: () => {
        app.topics = [...app.currentEditingTopics];
        store.saveTopics(app.topics);
        store.flushAllAiCache();
        ui.closeModal('infoCollectionConfigModal');
        // Reset editing state for next time modal opens
        app.editingTopicId = null;
        ui.addTopicBtn.textContent = 'Add Topic';
        console.log("Topic configuration saved. AI Cache flushed.", app.topics);
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

        await app.fetchAndCacheAiDataForLocation(locationId, false);

        const cachedDataForLocation = {};
        let needsOverallRefresh = false;

        app.topics.forEach(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            cachedDataForLocation[topic.id] = cacheEntry;
            if (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000)) {
                needsOverallRefresh = true;
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

        await app.fetchAndCacheAiDataForLocation(locationId, true);

        const cachedDataForLocation = {};
        app.topics.forEach(topic => {
            cachedDataForLocation[topic.id] = store.getAiCache(locationId, topic.id);
        });
        ui.displayInfoModal(location, app.topics, cachedDataForLocation);
        ui.refreshInfoButton.classList.add('hidden');
        console.log("Info modal refreshed for:", location.description);
    },

    // --- AI Data Fetching and Caching ---
    fetchAndCacheAiDataForLocation: async (locationId, forceRefresh = false) => {
        if (!app.config.apiKey) {
            console.warn("Cannot fetch AI data: API Key is not configured.");
            if (document.getElementById('appConfigModal').style.display !== 'block') {
                 ui.showAppConfigError("API Key is required to fetch data.");
                 ui.openModal('appConfigModal');
            }
            return false; // Indicate failure
        }

        const location = app.locations.find(l => l.id === locationId);
        if (!location) {
            console.error("Location not found for AI fetch:", locationId);
            return false; // Indicate failure
        }

        console.log(`Fetching/Caching AI data for location: ${location.description}. Force refresh: ${forceRefresh}`);
        let allSuccessful = true;

        for (const topic of app.topics) {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);

            if (forceRefresh || isStale) {
                console.log(`Fetching for topic: ${topic.description} (Stale: ${isStale}, Force: ${forceRefresh})`);
                try {
                    const aiData = await api.fetchAiData(app.config.apiKey, location.location, topic.aiQuery);
                    store.saveAiCache(locationId, topic.id, aiData);
                    console.log(`Successfully fetched and cached data for ${location.description} - ${topic.description}`);
                } catch (error) {
                    allSuccessful = false;
                    console.error(`Failed to fetch AI data for ${location.description} - ${topic.description}:`, error.message);
                    store.saveAiCache(locationId, topic.id, `Error: ${error.message}`);
                    
                    if (error.message.toLowerCase().includes("invalid api key")) {
                        ui.closeModal('infoModal');
                        ui.showAppConfigError("Invalid API Key. Please check your configuration and save.");
                        ui.openModal('appConfigModal');
                        return false; // Stop further processing
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
            if (app.topics.length === 0) continue;

            for (const topic of app.topics) {
                const cacheEntry = store.getAiCache(location.id, topic.id);
                if (!cacheEntry || (cacheEntry.timestamp || 0) < sixtyMinutesAgo) {
                    needsRefreshForLocation = true;
                    break; 
                }
            }

            if (needsRefreshForLocation) {
                console.log(`Data for location ${location.description} is outdated or missing. Refreshing...`);
                app.fetchAndCacheAiDataForLocation(location.id, true).then(success => {
                    if (success) {
                        console.log(`Background refresh for ${location.description} completed.`);
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === location.id) {
                            app.handleOpenLocationInfo(location.id);
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

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
});

document.addEventListener('DOMContentLoaded', app.init);
