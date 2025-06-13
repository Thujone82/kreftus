console.log("ui.js loaded");

const ui = {
    // DOM Elements
    locationsSection: document.getElementById('locationsSection'),
    locationButtonsContainer: document.getElementById('locationButtons'),
    btnLocationsConfig: document.getElementById('btnLocationsConfig'),
    btnInfoCollectionConfig: document.getElementById('btnInfoCollectionConfig'),
    btnAppConfig: document.getElementById('btnAppConfig'),
    globalRefreshButton: document.getElementById('globalRefreshButton'), 

    // App Config Modal
    appConfigModal: document.getElementById('appConfigModal'),
    appConfigError: document.getElementById('appConfigError'),
    apiKeyInput: document.getElementById('apiKey'),
    getApiKeyLinkContainer: document.getElementById('getApiKeyLinkContainer'), 
    primaryColorInput: document.getElementById('primaryColor'),
    backgroundColorInput: document.getElementById('backgroundColor'),
    saveAppConfigBtn: document.getElementById('saveAppConfig'),

    // Location Config Modal
    locationConfigModal: document.getElementById('locationConfigModal'),
    locationDescriptionInput: document.getElementById('locationDescription'),
    locationValueInput: document.getElementById('locationValue'),
    addLocationBtn: document.getElementById('addLocation'),
    locationsListUI: document.getElementById('locationsList'),
    saveLocationConfigBtn: document.getElementById('saveLocationConfig'),

    // Info Collection Config Modal
    infoCollectionConfigModal: document.getElementById('infoCollectionConfigModal'),
    topicDescriptionInput: document.getElementById('topicDescription'),
    topicAiQueryInput: document.getElementById('topicAiQuery'),
    addTopicBtn: document.getElementById('addTopic'),
    topicsListUI: document.getElementById('topicsList'),
    saveTopicConfigBtn: document.getElementById('saveTopicConfig'),

    // Info Modal
    infoModal: document.getElementById('infoModal'),
    infoModalTitle: document.getElementById('infoModalTitle'),
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
            button.onclick = () => {
                ui.closeModal(button.dataset.modalId);
            };
        });
        window.onclick = (event) => {
            if (event.target.classList.contains('modal')) {
                ui.closeModal(event.target.id);
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
        if(ui.primaryColorInput) ui.primaryColorInput.value = settings.primaryColor;
        if(ui.backgroundColorInput) ui.backgroundColorInput.value = settings.backgroundColor; 
        if(ui.appConfigError) ui.appConfigError.textContent = ''; 

        if (settings.apiKey && ui.getApiKeyLinkContainer) {
            ui.getApiKeyLinkContainer.classList.add('hidden');
        } else if (ui.getApiKeyLinkContainer) {
            ui.getApiKeyLinkContainer.classList.remove('hidden');
        }
        console.log("App config form loaded with settings:", settings);
    },

    toggleConfigButtons: (enabled) => {
        if(ui.btnLocationsConfig) ui.btnLocationsConfig.disabled = !enabled;
        if(ui.btnInfoCollectionConfig) ui.btnInfoCollectionConfig.disabled = !enabled;
        console.log(`Config buttons ${enabled ? 'enabled' : 'disabled'}`);
    },

    renderLocationButtons: (locations, onLocationClickCallback, areTopicsDefined) => {
        if(!ui.locationButtonsContainer) return;
        ui.locationButtonsContainer.innerHTML = ''; 
        if (locations && locations.length > 0) {
            if(ui.locationsSection) ui.locationsSection.classList.remove('hidden');
            locations.forEach(location => {
                const button = document.createElement('button');
                button.textContent = location.description;
                button.dataset.locationId = location.id;
                button.onclick = () => onLocationClickCallback(location.id);

                button.classList.remove('location-button-fresh', 'location-button-fetching', 'location-button-error', 'needs-info-structure');

                if (!areTopicsDefined) {
                    button.classList.add('needs-info-structure');
                    button.title = "Info Structure not defined. Click to configure.";
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
                                if (!cacheEntry || (Date.now() - (cacheEntry.timestamp || 0)) > (60 * 60 * 1000)) {
                                    allTopicsFresh = false; 
                                }
                            }
                        }
                        if (hasError) locationStatus = 'error';
                        else if (allTopicsFresh) locationStatus = 'fresh';
                    }
                    if (locationStatus === 'fresh') button.classList.add('location-button-fresh');
                    else if (locationStatus === 'fetching') button.classList.add('location-button-fetching');
                    else if (locationStatus === 'error') button.classList.add('location-button-error');
                }              
                ui.locationButtonsContainer.appendChild(button);
            });
        } else {
            if(ui.locationsSection) ui.locationsSection.classList.add('hidden');
        }
        
        if (typeof app.updateGlobalRefreshButtonVisibility === 'function') {
            app.updateGlobalRefreshButtonVisibility();
        }
        console.log("Location buttons rendered:", locations);
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

    displayInfoModal: (location, topics, cachedData) => {
        const locationData = store.getLocations().find(l => l.id === location.id);
        if (!locationData) return;
        if(ui.infoModalTitle) ui.infoModalTitle.textContent = `${locationData.description} nfo2Go`;
        if(ui.infoModalContent) ui.infoModalContent.innerHTML = ''; 
        if(ui.refreshInfoButton) ui.refreshInfoButton.classList.add('hidden'); 
        
        let oldestTimestamp = Date.now(), needsRefresh = false;
        topics.forEach(topic => {
            const cacheEntry = cachedData[topic.id];
            const sectionDiv = document.createElement('div');
            sectionDiv.classList.add('topic-section');
            const titleH3 = document.createElement('h3');
            titleH3.textContent = topic.description;
            titleH3.classList.add('collapsible-title');
            titleH3.onclick = function() {
                this.classList.toggle('active');
                const content = this.nextElementSibling;
                if(content) content.style.display = (content.style.display === "block") ? "none" : "block";
            };
            sectionDiv.appendChild(titleH3);
            const contentContainer = document.createElement('div');
            contentContainer.classList.add('ai-topic-content', 'collapsible-content');
            if (cacheEntry && cacheEntry.data) {
                contentContainer.innerHTML = utils.formatAiResponseToHtml(cacheEntry.data);
                if (cacheEntry.timestamp && cacheEntry.timestamp < oldestTimestamp) oldestTimestamp = cacheEntry.timestamp;
                if (!cacheEntry.timestamp || (Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000)) needsRefresh = true;
            } else {
                contentContainer.innerHTML = "<p>No data available. Try refreshing.</p>";
                needsRefresh = true; 
            }
            const topicFooter = document.createElement('div');
            topicFooter.classList.add('topic-content-footer');
            const hr = document.createElement('hr');
            topicFooter.appendChild(hr);
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
        if(ui.infoModalUpdated) ui.infoModalUpdated.textContent = `Updated ${utils.formatTimeAgo(overallAge)}`;
        if (needsRefresh && ui.refreshInfoButton) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = location.id;
        }
        ui.openModal('infoModal');
    },

    displayInfoModalError: (topicDescription, errorMessage) => {
        const errorP = document.createElement('p');
        errorP.classList.add('error-message');
        errorP.textContent = `Error loading ${topicDescription}: ${errorMessage}`;
        if(ui.infoModalContent) ui.infoModalContent.appendChild(errorP);
    },

    clearInputFields: (fields) => {
        fields.forEach(field => { if (field) field.value = ''; });
    },

    init: () => {
        ui.initModalCloseButtons();
        console.log("UI initialized");
    }
};
ui.init();
