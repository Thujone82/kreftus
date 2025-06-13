console.log("app.js loaded");

const app = {
    config: null,
    locations: [],
    topics: [],
    currentEditingLocations: [],
    currentEditingTopics: [],
    editingTopicId: null, 
    currentLocationIdForInfoModal: null,
    fetchingStatus: {}, 

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
            ui.openModal('appConfigModal'); 
        }
        app.updateGlobalRefreshButtonVisibility(); // Initial check for button visibility
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
            if (locationId) app.handleRefreshLocationInfo(locationId);
        };
        if (ui.globalRefreshButton) {
            ui.globalRefreshButton.onclick = () => {
                console.log("Global refresh triggered.");
                app.refreshOutdatedQueries(true); 
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
        ui.closeModal('appConfigModal');
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
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined); 
        ui.closeModal('locationConfigModal');
        for (const location of app.locations) {
            if (!oldLocationIds.has(location.id)) {
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
        if (!description || !aiQuery) { alert("Both description and AI query are required."); return; }
        if (app.editingTopicId) { 
            const topicToUpdate = app.currentEditingTopics.find(t => t.id === app.editingTopicId);
            if (topicToUpdate) { topicToUpdate.description = description; topicToUpdate.aiQuery = aiQuery; }
            app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
        } else { 
            app.currentEditingTopics.push({ id: utils.generateId(), description, aiQuery });
        }
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
        app.topics = [...app.currentEditingTopics];
        store.saveTopics(app.topics);
        store.flushAllAiCache(); 
        ui.closeModal('infoCollectionConfigModal');
        app.editingTopicId = null; ui.addTopicBtn.textContent = 'Add Topic';
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined); 
        app.updateGlobalRefreshButtonVisibility(); 
    },

    handleLocationButtonClick: (locationId) => {
        if (app.topics && app.topics.length > 0) app.handleOpenLocationInfo(locationId);
        else { alert("Please define an Info Structure before viewing location information."); ui.openModal('infoCollectionConfigModal'); }
    },
    handleOpenLocationInfo: async (locationId) => {
        if (!app.config.apiKey) { 
            if(ui.appConfigError) ui.appConfigError.textContent = "API Key is not configured. Please configure it first."; 
            ui.openModal('appConfigModal'); return; 
        }
        app.currentLocationIdForInfoModal = locationId;
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return;

        // Initial loading message
        if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go - Loading...`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = '<p>Fetching data...</p>';
        ui.openModal('infoModal');

        // Fetch data. The fetch function will update the modal title with progress.
        await app.fetchAndCacheAiDataForLocation(locationId, false); 
        
        // After fetching (or if no fetching was needed), display the content.
        // The title might have been updated by fetchAndCacheAiDataForLocation if fetching occurred.
        // If no fetching occurred, or after it's done, we re-render the modal with actual data.
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
        ui.displayInfoModal(location, app.topics, cachedDataForLocation); // This sets the final title if not fetching
        
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
        await app.fetchAndCacheAiDataForLocation(locationId, true); 
        // After fetch, handleOpenLocationInfo will be called by updateInfoModalLoadingMessage's timeout
        // or directly if no fetches were made by fetchAndCacheAiDataForLocation.
        // For robustness, ensure it's called if the modal is still open for this location.
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
            app.handleOpenLocationInfo(locationId);
        }
    },

    fetchAndCacheAiDataForLocation: async (locationId, forceRefresh = false) => {
        if (!app.config.apiKey) {
            if (document.getElementById('appConfigModal').style.display !== 'block') {
                 if(ui.appConfigError) ui.appConfigError.textContent = "API Key is required to fetch data."; 
                 ui.openModal('appConfigModal');
            } return false; 
        }
        const location = app.locations.find(l => l.id === locationId);
        if (!location) return false;

        app.fetchingStatus[locationId] = true;
        const areTopicsDefined = app.topics && app.topics.length > 0;
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        
        console.log(`Fetching/Caching AI data for location: ${location.description}. Force refresh: ${forceRefresh}`);
        
        let completedCount = 0;
        const totalTopicsToFetchInitially = app.topics.filter(topic => {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);
            const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');
            return forceRefresh || isStale || hasError;
        }).length;

        // Update modal title immediately if it's open for this location
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId && totalTopicsToFetchInitially > 0) {
            app.updateInfoModalLoadingMessage(location.description, completedCount, totalTopicsToFetchInitially);
        }
        
        const fetchPromises = [];
        let anInvalidKeyErrorOccurred = false;

        for (const topic of app.topics) {
            const cacheEntry = store.getAiCache(locationId, topic.id);
            const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);
            const hasError = cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:');

            if (forceRefresh || isStale || hasError) {
                console.log(`Fetching for topic: ${topic.description} (Stale: ${isStale}, Force: ${forceRefresh}, Error: ${hasError})`);
                fetchPromises.push(
                    api.fetchAiData(app.config.apiKey, location.location, topic.aiQuery)
                        .then(aiData => ({ status: 'fulfilled', value: aiData, topicId: topic.id, topicDescription: topic.description }))
                        .catch(error => ({ status: 'rejected', reason: error, topicId: topic.id, topicDescription: topic.description }))
                );
            }
        }

        if (fetchPromises.length === 0) {
            console.log(`No topics to fetch for ${location.description}.`);
            delete app.fetchingStatus[locationId];
            ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
            app.updateGlobalRefreshButtonVisibility();
            // If modal is open for this location, ensure title is reset
            if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
                 if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${location.description} nfo2Go`;
                 // Potentially re-render modal content if it was showing "Fetching data..."
                 app.handleOpenLocationInfo(locationId);
            }
            return true; 
        }

        const results = await Promise.allSettled(fetchPromises);
        let allIndividualFetchesSuccessful = true;

        results.forEach(result => {
            if (result.status === 'fulfilled') {
                store.saveAiCache(locationId, result.value.topicId, result.value.value);
                completedCount++;
                console.log(`Successfully fetched and cached data for ${location.description} - ${result.value.topicDescription}`);
            } else { // status === 'rejected'
                allIndividualFetchesSuccessful = false;
                completedCount++; // Still counts as a settled promise for progress
                const error = result.reason.reason; 
                const topicId = result.reason.topicId;
                const topicDescription = result.reason.topicDescription;

                console.error(`Failed to fetch AI data for ${location.description} - ${topicDescription}:`, error.message);
                store.saveAiCache(locationId, topicId, `Error: ${error.message}`);
                
                if (error.message.toLowerCase().includes("invalid api key")) {
                    anInvalidKeyErrorOccurred = true;
                }
            }
            // Update progress if modal is open for this location
            if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId) {
                app.updateInfoModalLoadingMessage(location.description, completedCount, fetchPromises.length);
            }
        });

        delete app.fetchingStatus[locationId];
        ui.renderLocationButtons(app.locations, app.handleLocationButtonClick, areTopicsDefined);
        app.updateGlobalRefreshButtonVisibility();

        if (anInvalidKeyErrorOccurred) {
            if(ui.infoModal) ui.closeModal('infoModal'); 
            if(ui.appConfigError) ui.appConfigError.textContent = "Invalid API Key. Please check your configuration and save.";
            ui.openModal('appConfigModal');
            return false; 
        }
        
        // If all fetches are done and the modal was showing progress, ensure it's updated to final state
        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === locationId && completedCount === fetchPromises.length) {
             // The timeout in updateInfoModalLoadingMessage will handle final display update
        }


        return allIndividualFetchesSuccessful;
    },
    
    updateInfoModalLoadingMessage: (locationDescription, completed, total) => {
        if (ui.infoModalTitle) {
            if (total === 0) { // Handle case where no topics were to be fetched
                 ui.infoModalTitle.textContent = `${locationDescription} nfo2Go`;
                 // Potentially call handleOpenLocationInfo here if needed to refresh content
                 if (app.currentLocationIdForInfoModal) {
                    // Small delay to allow UI to settle before re-rendering content
                    setTimeout(() => app.handleOpenLocationInfo(app.currentLocationIdForInfoModal), 50);
                 }
            } else {
                ui.infoModalTitle.textContent = `${locationDescription} nfo2Go - Fetching (${completed}/${total})`;
                if (completed === total){
                     setTimeout(() => {
                        // Check if modal is still open for this location before updating
                        if (ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal && app.locations.find(l=>l.id === app.currentLocationIdForInfoModal)?.description === locationDescription) {
                           if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${locationDescription} nfo2Go`;
                           // Re-render the modal content now that fetching is complete
                           app.handleOpenLocationInfo(app.currentLocationIdForInfoModal);
                        }
                    }, 150); // Increased delay slightly
                }
            }
        }
    },

    refreshOutdatedQueries: async (forceAllStale = false) => {
        if (!app.config.apiKey) return;
        const sixtyMinutesAgo = Date.now() - (60 * 60 * 1000);
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
                app.fetchAndCacheAiDataForLocation(location.id, true).then(success => {
                    // If the info modal was open for this specific location and fetch was successful, refresh its content
                    if (success && ui.infoModal.style.display === 'block' && app.currentLocationIdForInfoModal === location.id) {
                        // The fetchAndCacheAiDataForLocation itself will trigger a re-render of the modal
                        // via updateInfoModalLoadingMessage's timeout mechanism.
                    } else if (!success) {
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
                    const isStale = !cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000);
                    if (hasError || isStale) { anyStaleOrError = true; break; }
                }
                if (anyStaleOrError) break;
            }
        }
        if (anyStaleOrError) ui.globalRefreshButton.classList.remove('hidden');
        else ui.globalRefreshButton.classList.add('hidden');
    }
};

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled Promise Rejection:', event.reason);
});

document.addEventListener('DOMContentLoaded', app.init);
