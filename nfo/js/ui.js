console.log("ui.js loaded");

const ui = {
    // DOM Elements
    locationsSection: document.getElementById('locationsSection'),
    headerIcon: document.querySelector('.header-icon'), // Added
    locationButtonsContainer: document.getElementById('locationButtons'),
    btnLocationsConfig: document.getElementById('btnLocationsConfig'),
    btnInfoCollectionConfig: document.getElementById('btnInfoCollectionConfig'),
    btnAppConfig: document.getElementById('btnAppConfig'),
    btnAppUpdate: document.getElementById('btnAppUpdate'), // Added Update Button
    globalRefreshButton: document.getElementById('globalRefreshButton'), 

    // App Config Modal
    appConfigModal: document.getElementById('appConfigModal'),
    appConfigError: document.getElementById('appConfigError'),
    apiKeyInput: document.getElementById('apiKey'),
    rpmLimitInput: document.getElementById('rpmLimit'), // Added
    geminiApiKeyStatusUI: document.getElementById('geminiApiKeyStatus'), // Added
    getApiKeyLinkContainer: document.getElementById('getApiKeyLinkContainer'),
    primaryColorInput: document.getElementById('primaryColor'),
    owmApiKeyInput: document.getElementById('owmApiKey'), // Added
    owmApiKeyStatusUI: document.getElementById('owmApiKeyStatus'), // Added
    getOwmApiKeyLinkContainer: document.getElementById('getOwmApiKeyLinkContainer'), // Added
    backgroundColorInput: document.getElementById('backgroundColor'),
    saveAppConfigBtn: document.getElementById('saveAppConfig'),
    offlineStatus: document.getElementById('offlineStatus'),

    // Location Config Modal
    locationConfigModal: document.getElementById('locationConfigModal'),
    locationDescriptionInput: document.getElementById('locationDescription'),
    locationConfigError: document.getElementById('locationConfigError'),
    locationValueInput: document.getElementById('locationValue'),
    addLocationBtn: document.getElementById('addLocation'),
    locationsListUI: document.getElementById('locationsList'),
    saveLocationConfigBtn: document.getElementById('saveLocationConfig'),

    // Info Collection Config Modal
    infoCollectionConfigModal: document.getElementById('infoCollectionConfigModal'),
    topicDescriptionInput: document.getElementById('topicDescription'),
    topicAiQueryInput: document.getElementById('topicAiQuery'),
    topicConfigError: document.getElementById('topicConfigError'),
    addTopicBtn: document.getElementById('addTopic'),
    topicsListUI: document.getElementById('topicsList'),
    saveTopicConfigBtn: document.getElementById('saveTopicConfig'),

    // Info Modal
    infoModal: document.getElementById('infoModal'),
    infoModalTitle: document.getElementById('infoModalTitle'),
    infoModalWeather: document.getElementById('infoModalWeather'), // Added
    infoModalUpdated: document.getElementById('infoModalUpdated'),
    refreshInfoButton: document.getElementById('refreshInfoButton'),
    infoModalContent: document.getElementById('infoModalContent'),

    // Modal Management
    openModal: (modalId) => {
        const modal = document.getElementById(modalId);
        if (modal) modal.style.display = 'block';
        console.log(`Modal ${modalId} opened`);
    },
    closeModal: (modalId) => {
        const modal = document.getElementById(modalId);
        if (modal) modal.style.display = 'none';
        console.log(`Modal ${modalId} closed`);
    },
    initModalCloseButtons: () => {
        document.querySelectorAll('.close-button').forEach(button => {
            button.onclick = (e) => {
                const modalId = e.currentTarget.dataset.modalId;
                if (app.hasUnsavedChanges && app.hasUnsavedChanges(modalId)) {
                    if (confirm("You have unsaved changes. Save them now? \nOK = Save, Cancel = Discard")) {
                        if (modalId === APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG && typeof app.handleSaveLocationConfig === 'function') {
                            app.handleSaveLocationConfig(); // This also closes the modal
                        } else if (modalId === APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG && typeof app.handleSaveTopicConfig === 'function') {
                            app.handleSaveTopicConfig(); // This also closes the modal
                        } else {
                            ui.closeModal(modalId); // Fallback if save handler not found
                        }
                    } else {
                        ui.closeModal(modalId); // Discard changes
                    }
                } else {
                    ui.closeModal(modalId); // No unsaved changes
                }
            };
        });
        window.onclick = (event) => {
            if (event.target.classList.contains('modal')) {
                const modalId = event.target.id;
                if (app.hasUnsavedChanges && app.hasUnsavedChanges(modalId)) {
                     if (confirm("You have unsaved changes. Save them now? \nOK = Save, Cancel = Discard")) {
                        if (modalId === APP_CONSTANTS.MODAL_IDS.LOCATION_CONFIG && typeof app.handleSaveLocationConfig === 'function') {
                            app.handleSaveLocationConfig();
                        } else if (modalId === APP_CONSTANTS.MODAL_IDS.INFO_COLLECTION_CONFIG && typeof app.handleSaveTopicConfig === 'function') {
                            app.handleSaveTopicConfig();
                        } else { ui.closeModal(modalId); }
                    } else { ui.closeModal(modalId); }
                } else { ui.closeModal(modalId); }
            }
        };
    },

    isColorDark: (hexColor) => {
        const color = (hexColor.charAt(0) === '#') ? hexColor.substring(1, 7) : hexColor;
        const r = parseInt(color.substring(0, 2), 16);
        const g = parseInt(color.substring(2, 4), 16);
        const b = parseInt(color.substring(4, 6), 16);
        const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        return luminance < 0.5;
    },

    adjustColorBrightness: (hexColor, percent) => {
        let R = parseInt(hexColor.substring(1,3),16);
        let G = parseInt(hexColor.substring(3,5),16);
        let B = parseInt(hexColor.substring(5,7),16);
        R = parseInt(R * (100 + percent) / 100);
        G = parseInt(G * (100 + percent) / 100);
        B = parseInt(B * (100 + percent) / 100);
        R = (R<255)?R:255;  
        G = (G<255)?G:255;  
        B = (B<255)?B:255;  
        R = (R>0)?R:0;
        G = (G>0)?G:0;
        B = (B>0)?B:0;
        const RR = ((R.toString(16).length==1)?"0"+R.toString(16):R.toString(16));
        const GG = ((G.toString(16).length==1)?"0"+G.toString(16):G.toString(16));
        const BB = ((B.toString(16).length==1)?"0"+B.toString(16):B.toString(16));
        return "#"+RR+GG+BB;
    },

    applyTheme: (primaryColor, contentBackgroundColor) => {
        document.documentElement.style.setProperty('--primary-color', primaryColor);
        document.documentElement.style.setProperty('--content-background-color', contentBackgroundColor);
        let pageBackgroundColor;
        let textColor;

        if (ui.isColorDark(contentBackgroundColor)) {
            // Dark theme settings
            textColor = '#E0E0E0';
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 5); 
            if (pageBackgroundColor === contentBackgroundColor) { 
                 pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 10) === contentBackgroundColor ? '#121212' : ui.adjustColorBrightness(contentBackgroundColor, 10);
            }
            // Set variables for dark theme inputs/lists
            document.documentElement.style.setProperty('--input-bg-color', '#333');
            document.documentElement.style.setProperty('--input-border-color', '#555');
            document.documentElement.style.setProperty('--list-item-bg-color', '#2a2a2a');
            document.documentElement.style.setProperty('--list-item-border-color', '#444');
        } else {
            // Light theme settings
            textColor = '#121212';
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, -5);
             if (pageBackgroundColor === contentBackgroundColor) { 
                 pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, -10) === contentBackgroundColor ? '#FAFAFA' : ui.adjustColorBrightness(contentBackgroundColor, -10);
            }
            // Set variables for light theme inputs/lists
            document.documentElement.style.setProperty('--input-bg-color', '#F0F0F0');
            document.documentElement.style.setProperty('--input-border-color', '#CCCCCC');
            document.documentElement.style.setProperty('--list-item-bg-color', '#FAFAFA');
            document.documentElement.style.setProperty('--list-item-border-color', '#E0E0E0');
        }
        document.documentElement.style.setProperty('--text-color', textColor);
        document.body.style.backgroundColor = pageBackgroundColor;
        
        let themeColorMeta = document.querySelector('meta[name="theme-color"]');
        if (themeColorMeta) {
            themeColorMeta.setAttribute('content', primaryColor);
        } else {
            themeColorMeta = document.createElement('meta');
            themeColorMeta.name = "theme-color";
            themeColorMeta.content = primaryColor;
            document.getElementsByTagName('head')[0].appendChild(themeColorMeta);
        }
        console.log(`Theme applied: Primary=${primaryColor}, ContentBG=${contentBackgroundColor}, PageBG=${pageBackgroundColor}, Text=${textColor}`);
    },

    loadAppConfigForm: (settings) => {
        if(ui.apiKeyInput) ui.apiKeyInput.value = settings.apiKey || '';
        if(ui.rpmLimitInput) ui.rpmLimitInput.value = settings.rpmLimit || 10; // Added
        if(ui.owmApiKeyInput) ui.owmApiKeyInput.value = settings.owmApiKey || ''; // Added
        if(ui.primaryColorInput) ui.primaryColorInput.value = settings.primaryColor;
        if(ui.backgroundColorInput) ui.backgroundColorInput.value = settings.backgroundColor; 
        if(ui.appConfigError) ui.appConfigError.textContent = ''; 

        // Toggle visibility of "Get API Key" links
        if (ui.getApiKeyLinkContainer) {
            ui.getApiKeyLinkContainer.classList.toggle('hidden', !!settings.apiKey);
        }
        if (ui.getOwmApiKeyLinkContainer) { // Added
            ui.getOwmApiKeyLinkContainer.classList.toggle('hidden', !!settings.owmApiKey); // Added
        }
        console.log("App config form loaded with settings:", settings);
    },

    toggleConfigButtons: (enabled) => {
        if(ui.btnLocationsConfig) ui.btnLocationsConfig.disabled = !enabled;
        if(ui.btnInfoCollectionConfig) ui.btnInfoCollectionConfig.disabled = !enabled;
        console.log(`Config buttons ${enabled ? 'enabled' : 'disabled'}`);
    },

    startHeaderIconLoading: () => {
        if (ui.headerIcon) {
            ui.headerIcon.classList.add('header-icon-loading');
        }
    },

    stopHeaderIconLoading: () => {
        if (ui.headerIcon) {
            ui.headerIcon.classList.remove('header-icon-loading');
            ui.headerIcon.style.animationPlayState = 'running'; // Ensure it's not paused
        }
    },

    pauseHeaderIconLoading: () => {
        if (ui.headerIcon && ui.headerIcon.classList.contains('header-icon-loading')) {
            ui.headerIcon.style.animationPlayState = 'paused';
        }
    },

    resumeHeaderIconLoading: () => {
        if (ui.headerIcon && ui.headerIcon.classList.contains('header-icon-loading')) {
            ui.headerIcon.style.animationPlayState = 'running';
        }
    },

    setApiKeyStatus: (apiKeyType, status, message = '') => { // No longer async itself
        let statusElement;
        if (apiKeyType === 'gemini') {
            statusElement = ui.geminiApiKeyStatusUI;
        } else if (apiKeyType === 'owm') {
            statusElement = ui.owmApiKeyStatusUI;
        }

        if (statusElement) {
            statusElement.textContent = message || status.charAt(0).toUpperCase() + status.slice(1);
            statusElement.className = 'api-key-status'; // Reset classes
            if (status === 'valid') {
                statusElement.classList.add('status-valid');
            } else if (status === 'invalid') {
                statusElement.classList.add('status-invalid');
            } else if (status === 'checking') {
                statusElement.classList.add('status-checking');
            } else if (status === 'rate_limit') {
                statusElement.classList.add('status-rate-limit');
            }
        }
    },

    renderLocationButtons: (locations, onLocationClickCallback, areTopicsDefined) => { // No longer async itself
        if(!ui.locationButtonsContainer) return;
        ui.locationButtonsContainer.innerHTML = ''; 
        if (locations && locations.length > 0) {
            if(ui.locationsSection) ui.locationsSection.classList.remove('hidden');

            locations.forEach(location => {
                const button = document.createElement('button');
                let buttonHTML = '';
                button.dataset.locationId = location.id;
                // button.dataset.locationString = location.location; // Not strictly needed if location object is passed
                button.onclick = () => onLocationClickCallback(location.id);

                button.classList.remove('location-button-fresh', 'location-button-fetching', 'location-button-error', 'needs-info-structure');

                if (!areTopicsDefined) {
                    // Structure for button content: Name on one line, weather on another (if available)
                    const nameSpan = `<span class="location-button-name">${location.description}</span>`;
                    buttonHTML = `${nameSpan}<span class="location-button-weather"></span>`; // Placeholder for weather
                    button.classList.add('needs-info-structure');
                    button.title = "Info Structure not defined. Weather disabled.";
                } else if (app.fetchingStatus && app.fetchingStatus[location.id]) {
                    button.classList.add('location-button-fetching');
                    const nameSpan = `<span class="location-button-name">${location.description}</span>`;
                    buttonHTML = `${nameSpan}<span class="location-button-weather"></span>`; // Placeholder
                    button.title = "Fetching AI data..."; // Initial title
                } else {
                    let locationStatus = 'stale'; 
                    if (app.fetchingStatus && app.fetchingStatus[location.id]) {
                        locationStatus = 'fetching';
                    } else {
                        let hasError = false;
                        let allTopicsFresh = app.topics.length > 0; 
                        if (app.topics.length === 0) { 
                            allTopicsFresh = false; 
                        } else {
                            for (const topic of app.topics) {
                                const cacheEntry = store.getAiCache(location.id, topic.id);
                                if (cacheEntry && typeof cacheEntry.data === 'string' && cacheEntry.data.toLowerCase().startsWith('error:')) {
                                    hasError = true;
                                    break; 
                                }
                                if (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > APP_CONSTANTS.CACHE_EXPIRY_MS) {
                                    allTopicsFresh = false; 
                                }
                            }
                        }
                        if (hasError) locationStatus = 'error';
                        else if (allTopicsFresh) locationStatus = 'fresh';
                    }

                    const nameSpan = `<span class="location-button-name">${location.description}</span>`;
                    buttonHTML = `${nameSpan}<span class="location-button-weather"></span>`; // Placeholder

                    if (locationStatus === 'fresh') button.classList.add('location-button-fresh');
                    else if (locationStatus === 'fetching') { // Should be caught by the earlier check, but as a fallback
                        button.classList.add('location-button-fetching');
                        button.title = "Fetching AI data...";
                    } else if (locationStatus === 'error') {
                        button.classList.add('location-button-error');
                        button.title = "One or more topics have an error.";
                    } else { // Default for 'stale'
                        button.title = "AI Data is stale or not yet loaded.";
                    }
                }              
                button.innerHTML = buttonHTML.trim();
                ui.locationButtonsContainer.appendChild(button);

                // Asynchronously update weather for this button
                if (app.config && app.config.owmApiKey) {
                    ui.updateButtonWeatherDisplay(button, location);
                }
            });
        } else {
            if(ui.locationsSection) ui.locationsSection.classList.add('hidden');
        }
        
        if (typeof app.updateGlobalRefreshButtonVisibility === 'function') {
            app.updateGlobalRefreshButtonVisibility();
        }
        console.log("Location buttons rendered:", locations);
    },

    updateButtonWeatherDisplay: async (buttonElement, location) => {
        // Ensure this function is robust against the button being removed from DOM
        if (!buttonElement || !document.body.contains(buttonElement)) {
            return;
        }
        const weatherDisplayHtml = await app.getWeatherDisplayForLocation(location);
        const weatherSpan = buttonElement.querySelector('.location-button-weather');
        if (weatherSpan) { // Check if weatherSpan still exists (button might have been re-rendered)
            if (weatherDisplayHtml) {
                weatherSpan.innerHTML = weatherDisplayHtml;
                // Update title intelligently based on existing title and weather presence
                const currentTitle = buttonElement.title || "";
                if (currentTitle.includes("Fetching AI data") && !currentTitle.includes("current weather")) buttonElement.title = "Fetching AI data, showing current weather...";
                else if (currentTitle.includes("AI Data is stale") && !currentTitle.includes("Weather shown")) buttonElement.title = "AI Data is stale or not yet loaded. Weather shown.";
                else if (currentTitle.includes("topics have an error") && !currentTitle.includes("Weather shown")) buttonElement.title = "One or more topics have an error. Weather shown.";
            } else {
                weatherSpan.innerHTML = ''; // Clear if no weather
            }
        }
    },

    renderConfigList: (items, listElement, type, onRemoveCallback, onEditCallback) => {
        if(!listElement) return;
        listElement.innerHTML = ''; 
        items.forEach((item, index) => {
            const li = document.createElement('li');
            const textSpan = document.createElement('span');
            textSpan.textContent = `${item.description} ${type === 'location' ? '(' + item.location + ')' : '(' + item.aiQuery + ')'}`;
            li.appendChild(textSpan);
            li.dataset.id = item.id;
            li.dataset.index = index;
            li.draggable = true;
            const buttonContainer = document.createElement('div');
            buttonContainer.classList.add('config-item-buttons');
            if (type === 'topic' && onEditCallback) {
                const editBtn = document.createElement('button');
                editBtn.textContent = 'Edit';
                editBtn.onclick = (e) => { e.stopPropagation(); onEditCallback(item.id); };
                buttonContainer.appendChild(editBtn);
            }
            const removeBtn = document.createElement('button');
            removeBtn.textContent = 'Remove';
            removeBtn.classList.add('danger');
            removeBtn.onclick = (e) => { e.stopPropagation(); onRemoveCallback(item.id); };
            buttonContainer.appendChild(removeBtn);
            li.appendChild(buttonContainer);
            listElement.appendChild(li);
        });
        console.log(`${type} list rendered with ${items.length} items`);
    },

    enableDragAndDrop: (listElement, onSortCallback) => {
        if(!listElement) return;
        let draggedItem = null;
        const getLiTarget = (eventTarget) => {
            let target = eventTarget;
            while (target && target.tagName !== 'LI') {
                if (!target.parentElement || target === listElement) return null;
                target = target.parentElement;
            }
            return target;
        };
        listElement.addEventListener('dragstart', (e) => {
            if (draggedItem) return; 
            const targetLi = getLiTarget(e.target); 
            if (targetLi && targetLi.draggable) { 
                draggedItem = targetLi;
                setTimeout(() => { if (draggedItem) draggedItem.classList.add('dragging'); }, 0);
            }
        });
        listElement.addEventListener('dragend', (e) => {
            if (draggedItem && e.target === draggedItem) {
                draggedItem.classList.remove('dragging');
                onSortCallback(Array.from(listElement.children).map(li => li.dataset.id));
                draggedItem = null; 
            }
        });
        listElement.addEventListener('dragover', (e) => {
            if (draggedItem) {
                e.preventDefault(); 
                const afterElement = getDragAfterElement(listElement, e.clientY);
                if (afterElement === draggedItem) return; 
                if (afterElement == null) listElement.appendChild(draggedItem);
                else listElement.insertBefore(draggedItem, afterElement);
            }
        });
        const handleTouchStart = (e) => {
            if (draggedItem) return; 
            const targetLi = getLiTarget(e.targetTouches[0].target);
            if (targetLi && targetLi.draggable) { 
                draggedItem = targetLi;
                draggedItem.classList.add('dragging'); 
                document.addEventListener('touchmove', handleTouchMove, { passive: false });
                document.addEventListener('touchend', handleTouchEnd, { passive: true });
                document.addEventListener('touchcancel', handleTouchEnd, { passive: true }); 
            }
        };
        const handleTouchMove = (e) => {
            if (!draggedItem) return;
            e.preventDefault(); 
            const touch = e.touches[0];
            const afterElement = getDragAfterElement(listElement, touch.clientY);
            if (afterElement === draggedItem) return; 
            if (afterElement == null) listElement.appendChild(draggedItem);
            else listElement.insertBefore(draggedItem, afterElement);
        };
        const handleTouchEnd = (e) => {
            if (!draggedItem) return;
            draggedItem.classList.remove('dragging');
            onSortCallback(Array.from(listElement.children).map(li => li.dataset.id));
            draggedItem = null;
            document.removeEventListener('touchmove', handleTouchMove, { passive: false });
            document.removeEventListener('touchend', handleTouchEnd, { passive: true });
            document.removeEventListener('touchcancel', handleTouchEnd, { passive: true });
        };
        listElement.addEventListener('touchstart', handleTouchStart, { passive: true }); 
        function getDragAfterElement(container, y) {
            const draggableElements = [...container.querySelectorAll('li:not(.dragging)')];
            return draggableElements.reduce((closest, child) => {
                const box = child.getBoundingClientRect();
                const offset = y - box.top - box.height / 2;
                if (offset < 0 && offset > closest.offset) return { offset: offset, element: child };
                else return closest;
            }, { offset: Number.NEGATIVE_INFINITY }).element;
        }
    },

    displayInfoModal: async (location, topics, cachedData, isCurrentlyFetching, isOffline) => {
        const locationData = store.getLocations().find(l => l.id === location.id);
        if (!locationData) return;

        // If a fetch is NOT active for this modal, set the title to its final state.
        // If a fetch IS active, app.updateInfoModalLoadingMessage is managing the title (e.g., "Fetching (X/Y)...").
        if (!isCurrentlyFetching && ui.infoModalTitle) {
            ui.infoModalTitle.textContent = `${locationData.description} nfo2Go`;
        }
        if (ui.infoModalContent) ui.infoModalContent.innerHTML = '';
         if (isOffline && ui.offlineStatus) {
            ui.offlineStatus.classList.remove('hidden');
        } else if (ui.offlineStatus) {
            ui.offlineStatus.classList.add('hidden');
        }
        if(ui.refreshInfoButton) ui.refreshInfoButton.classList.add('hidden');

        let oldestTimestamp = Date.now(), needsRefreshOverall = false;
        // Fetch and display weather in its dedicated spot
        if(ui.infoModalWeather) ui.infoModalWeather.innerHTML = ''; // Clear previous weather
        if (app.config && app.config.owmApiKey) {
            const weatherDisplayHtml = await app.getWeatherDisplayForLocation(location);
            if (weatherDisplayHtml && ui.infoModalWeather) {
                ui.infoModalWeather.innerHTML = weatherDisplayHtml;
            }
        }

        topics.forEach(topic => {
            const cacheEntry = cachedData[topic.id];
            const sectionDiv = document.createElement('div');
            sectionDiv.classList.add('topic-section');
            const titleH3 = document.createElement('h3'); // This is the topic title, not the modal title
            titleH3.textContent = topic.description;
            titleH3.classList.add('collapsible-title');
            
            sectionDiv.appendChild(titleH3);
            const contentContainer = document.createElement('div');
            contentContainer.classList.add('ai-topic-content', 'collapsible-content');

            // Load and apply collapsed state
            let isCollapsed = store.getTopicCollapsedState(location.id, topic.id);
            if (isCollapsed === null) { // No saved state
                isCollapsed = true; // Default to collapsed
            }
            if (!isCollapsed) { // If expanded
                titleH3.classList.add('active');
                contentContainer.style.display = "block";
            } else { // If collapsed
                contentContainer.style.display = "none";
            }

            titleH3.onclick = function() {
                this.classList.toggle('active');
                const content = this.nextElementSibling;
                const currentlyCollapsed = (content.style.display === "none" || content.style.display === "");
                content.style.display = currentlyCollapsed ? "block" : "none";

                // If we just collapsed the item (it was previously not collapsed)
                if (!currentlyCollapsed) {
                    // Use a minimal timeout to allow the DOM to reflow after the display change.
                    // This ensures our measurements of the modal's position are accurate.
                    setTimeout(() => {
                        const modalContentEl = ui.infoModal.querySelector('.modal-content');
                        if (!modalContentEl) return;

                        const modalRect = modalContentEl.getBoundingClientRect();
                        const viewportHeight = window.innerHeight;

                        // If the bottom of the modal content is now above the bottom of the viewport,
                        // it means the modal has jumped up. We need to scroll it back down.
                        if (modalRect.bottom < viewportHeight) {
                            const scrollOffset = viewportHeight - modalRect.bottom;
                            ui.infoModal.scrollTop -= scrollOffset;
                        }
                    }, 0);
                }

                store.saveTopicCollapsedState(location.id, topic.id, !currentlyCollapsed);
            };

            // Swipe to collapse logic
            let touchStartX = 0;
            let touchStartY = 0;
            const SWIPE_THRESHOLD = 50; // Minimum horizontal distance for a swipe
            const SWIPE_VERTICAL_MAX = 75; // Maximum vertical distance to still be considered a horizontal swipe

            sectionDiv.addEventListener('touchstart', (e) => {
                // Only track single touches for swipe
                if (e.touches.length === 1) {
                    touchStartX = e.touches[0].clientX;
                    touchStartY = e.touches[0].clientY;
                }
            }, { passive: true });

            sectionDiv.addEventListener('touchend', (e) => {
                if (e.changedTouches.length === 1) { // Ensure it's the end of a single touch
                    const touchEndX = e.changedTouches[0].clientX;
                    const touchEndY = e.changedTouches[0].clientY;

                    const deltaX = touchEndX - touchStartX;
                    const deltaY = touchEndY - touchStartY;

                    // Check for a right swipe on an expanded section
                    if (deltaX > SWIPE_THRESHOLD && Math.abs(deltaY) < SWIPE_VERTICAL_MAX) {
                        if (titleH3.classList.contains('active')) { // If currently expanded
                            console.log(`UI: Right swipe detected on expanded topic "${topic.description}". Collapsing.`);
                            titleH3.click(); // Simulate a click on the title to collapse it
                            // e.preventDefault(); // Optional: if click was also being triggered
                        }
                    }
                }
                // Reset for next touch
                touchStartX = 0; touchStartY = 0;
            }, { passive: true });

            if (cacheEntry && cacheEntry.data) {
                const formattedHtml = utils.formatAiResponseToHtml(cacheEntry.data);
                // console.log(`HTML for topic "${topic.description}" (Location: ${location.id}):\n`, formattedHtml); // DEBUG LOG
                contentContainer.innerHTML = formattedHtml;
                if (cacheEntry.timestamp && cacheEntry.timestamp < oldestTimestamp) {
                    oldestTimestamp = cacheEntry.timestamp;
                }
                if (!isCurrentlyFetching && (!cacheEntry.timestamp || (Date.now() - cacheEntry.timestamp) > APP_CONSTANTS.CACHE_EXPIRY_MS)) {
                     needsRefreshOverall = true;
                }
            } else {
                contentContainer.innerHTML = isCurrentlyFetching ? "<p>Fetching data...</p>" : "<p>No data available. Try refreshing.</p>";
                if (!isCurrentlyFetching) needsRefreshOverall = true; 
            }
            const topicFooter = document.createElement('div');
            topicFooter.classList.add('topic-content-footer');
            const hr = document.createElement('hr');
            topicFooter.appendChild(hr);

            const copyButton = document.createElement('button');
            copyButton.textContent = 'Copy';
            copyButton.classList.add('copy-topic-button');
            copyButton.onclick = function() {
                // Clone the content container to manipulate it without affecting the displayed version
                const tempContainer = contentContainer.cloneNode(true);
                // Find and remove the footer from the cloned container
                const footerToRemove = tempContainer.querySelector('.topic-content-footer');
                if (footerToRemove) {
                    footerToRemove.parentNode.removeChild(footerToRemove);
                }
                // Get text content from the modified clone
                const contentToCopy = tempContainer.innerText || tempContainer.textContent;
                navigator.clipboard.writeText(contentToCopy).then(() => {
                    copyButton.textContent = 'Copied!';
                    setTimeout(() => {
                        copyButton.textContent = 'Copy';
                    }, 2000);
                }).catch(err => {
                    console.error('Failed to copy text: ', err);
                    copyButton.textContent = 'Error!';
                    setTimeout(() => { copyButton.textContent = 'Copy'; }, 2000);
                });
            };
            topicFooter.appendChild(copyButton);

            const collapseHint = document.createElement('span');
            collapseHint.classList.add('collapse-hint');
            collapseHint.innerHTML = 'Collapse &#9650;'; 
            collapseHint.onclick = function() {
                const currentTitleH3 = this.closest('.topic-section').querySelector('.collapsible-title');
                if (currentTitleH3) currentTitleH3.click();
            };
            topicFooter.appendChild(collapseHint);
            contentContainer.appendChild(topicFooter); 
            sectionDiv.appendChild(contentContainer);
            if(ui.infoModalContent) ui.infoModalContent.appendChild(sectionDiv);
        });
        const overallAge = (topics.length > 0 && oldestTimestamp !== Date.now()) ? oldestTimestamp : null;
        if(ui.infoModalUpdated) {
            ui.infoModalUpdated.textContent = isCurrentlyFetching ? "Fetching latest AI data..." : `AI Data Updated ${utils.formatTimeAgo(overallAge)}`;
        }

        if (needsRefreshOverall && !isCurrentlyFetching && ui.refreshInfoButton) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = location.id;
        }
    },

    displayInfoModalError: (topicDescription, errorMessage) => {
        const errorP = document.createElement('p');
        errorP.classList.add('error-message');
        errorP.textContent = `Error loading ${topicDescription}: ${errorMessage}`;
        if(ui.infoModalContent) ui.infoModalContent.appendChild(errorP);
    },

    refreshInfoModalWeatherOnly: async (location) => {
        if (!ui.infoModalWeather || !app.config || !app.config.owmApiKey || ui.infoModal.style.display !== 'block') {
            // Only proceed if the modal is visible, weather element exists, and OWM key is configured
            return;
        }

        // ui.infoModalWeather.innerHTML = '<p>Refreshing weather...</p>'; // Optional: Add a temporary loading indicator
        const weatherDisplayHtml = await app.getWeatherDisplayForLocation(location); // This will fetch if stale or use cache
        
        if (weatherDisplayHtml && ui.infoModalWeather) { // Ensure element still exists
            ui.infoModalWeather.innerHTML = weatherDisplayHtml;
        } else if (ui.infoModalWeather) {
            ui.infoModalWeather.innerHTML = ''; // Clear if no weather data
        }
        console.log(`UI: Weather in info modal attempted refresh for ${location.description}`);
    },
    clearInputFields: (fields) => {
        fields.forEach(field => { if (field) field.value = ''; });
    },

    init: () => {
        const DEBOUNCE_DELAY_MS = 750; // Delay in milliseconds for debouncing API key validation

        ui.initModalCloseButtons();

        // Debounced API Key Validation for Gemini
        if (ui.apiKeyInput) {
            ui.apiKeyInput.addEventListener('input', utils.debounce((event) => {
                const key = event.target.value.trim();
                app.validateAndDisplayGeminiKeyStatus(key, false); // false for onOpen to show "Checking..."
            }, DEBOUNCE_DELAY_MS));
        }

        // Debounced API Key Validation for OpenWeatherMap
        if (ui.owmApiKeyInput) {
            ui.owmApiKeyInput.addEventListener('input', utils.debounce((event) => {
                const key = event.target.value.trim();
                app.validateAndDisplayOwmKeyStatus(key, false); // false for onOpen to show "Checking..."
            }, DEBOUNCE_DELAY_MS));
        }

        console.log("UI initialized");
    }
};
ui.init();
