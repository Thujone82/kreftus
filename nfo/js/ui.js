console.log("ui.js loaded");

const ui = {
    // DOM Elements
    locationsSection: document.getElementById('locationsSection'),
    locationButtonsContainer: document.getElementById('locationButtons'),
    btnLocationsConfig: document.getElementById('btnLocationsConfig'),
    btnInfoCollectionConfig: document.getElementById('btnInfoCollectionConfig'),
    btnAppConfig: document.getElementById('btnAppConfig'),

    // App Config Modal
    appConfigModal: document.getElementById('appConfigModal'),
    appConfigError: document.getElementById('appConfigError'),
    apiKeyInput: document.getElementById('apiKey'),
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
            textColor = '#E0E0E0';
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 5); 
            if (pageBackgroundColor === contentBackgroundColor) { 
                 pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 10) === contentBackgroundColor ? '#121212' : ui.adjustColorBrightness(contentBackgroundColor, 10);
            }
        } else {
            textColor = '#121212';
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, -5);
             if (pageBackgroundColor === contentBackgroundColor) { 
                 pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, -10) === contentBackgroundColor ? '#FAFAFA' : ui.adjustColorBrightness(contentBackgroundColor, -10);
            }
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
        ui.apiKeyInput.value = settings.apiKey || '';
        ui.primaryColorInput.value = settings.primaryColor;
        ui.backgroundColorInput.value = settings.backgroundColor; 
        ui.appConfigError.textContent = ''; 
        console.log("App config form loaded with settings:", settings);
    },
    showAppConfigError: (message) => {
        ui.appConfigError.textContent = message;
    },
    toggleConfigButtons: (enabled) => {
        ui.btnLocationsConfig.disabled = !enabled;
        ui.btnInfoCollectionConfig.disabled = !enabled;
        console.log(`Config buttons ${enabled ? 'enabled' : 'disabled'}`);
    },

    renderLocationButtons: (locations, onLocationClickCallback) => {
        ui.locationButtonsContainer.innerHTML = ''; 
        if (locations && locations.length > 0) {
            ui.locationsSection.classList.remove('hidden');
            locations.forEach(location => {
                const button = document.createElement('button');
                button.textContent = location.description;
                button.dataset.locationId = location.id;
                button.onclick = () => onLocationClickCallback(location.id);
                ui.locationButtonsContainer.appendChild(button);
            });
        } else {
            ui.locationsSection.classList.add('hidden');
        }
        console.log("Location buttons rendered:", locations);
    },

    renderConfigList: (items, listElement, type, onRemoveCallback, onEditCallback) => { // Added onEditCallback
        listElement.innerHTML = ''; 
        items.forEach((item, index) => {
            const li = document.createElement('li');
            const textSpan = document.createElement('span');
            textSpan.textContent = `${item.description} ${type === 'location' ? '(' + item.location + ')' : '(' + item.aiQuery + ')'}`;
            li.appendChild(textSpan);

            li.dataset.id = item.id;
            li.dataset.index = index;
            li.draggable = true;

            const buttonContainer = document.createElement('div'); // Container for buttons
            buttonContainer.classList.add('config-item-buttons');

            if (type === 'topic' && onEditCallback) {
                const editBtn = document.createElement('button');
                editBtn.textContent = 'Edit';
                editBtn.onclick = (e) => {
                    e.stopPropagation();
                    onEditCallback(item.id);
                };
                buttonContainer.appendChild(editBtn);
            }

            const removeBtn = document.createElement('button');
            removeBtn.textContent = 'Remove';
            removeBtn.classList.add('danger');
            removeBtn.onclick = (e) => {
                e.stopPropagation(); 
                onRemoveCallback(item.id);
            };
            buttonContainer.appendChild(removeBtn);
            li.appendChild(buttonContainer);
            listElement.appendChild(li);
        });
        console.log(`${type} list rendered with ${items.length} items`);
    },

    enableDragAndDrop: (listElement, onSortCallback) => {
        let draggedItem = null;
        listElement.addEventListener('dragstart', (e) => {
            if (e.target.tagName === 'LI') { // Ensure drag starts on LI
                draggedItem = e.target;
                setTimeout(() => {
                    if(e.target.style) e.target.style.opacity = '0.5';
                }, 0);
            }
        });
        listElement.addEventListener('dragend', (e) => {
            if (draggedItem) {
                setTimeout(() => {
                    if(e.target.style) e.target.style.opacity = '1';
                    draggedItem = null;
                }, 0);
                onSortCallback(Array.from(listElement.children).map(li => li.dataset.id));
            }
        });
        listElement.addEventListener('dragover', (e) => {
            e.preventDefault();
            if (draggedItem) {
                const afterElement = getDragAfterElement(listElement, e.clientY);
                if (afterElement == null) {
                    listElement.appendChild(draggedItem);
                } else {
                    listElement.insertBefore(draggedItem, afterElement);
                }
            }
        });
        function getDragAfterElement(container, y) {
            const draggableElements = [...container.querySelectorAll('li:not(.dragging)')];
            return draggableElements.reduce((closest, child) => {
                const box = child.getBoundingClientRect();
                const offset = y - box.top - box.height / 2;
                if (offset < 0 && offset > closest.offset) {
                    return { offset: offset, element: child };
                } else {
                    return closest;
                }
            }, { offset: Number.NEGATIVE_INFINITY }).element;
        }
    },

    displayInfoModal: (location, topics, cachedData) => {
        const locationData = store.getLocations().find(l => l.id === location.id);
        if (!locationData) {
            console.error("Location data not found for info modal:", location.id);
            return;
        }
        ui.infoModalTitle.textContent = `${locationData.description} Info2Go`;
        ui.infoModalContent.innerHTML = ''; 
        ui.refreshInfoButton.classList.add('hidden'); 
        let oldestTimestamp = Date.now();
        let needsRefresh = false;
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
                if (content.style.display === "block") {
                    content.style.display = "none";
                } else {
                    content.style.display = "block";
                }
            };
            sectionDiv.appendChild(titleH3);
            const contentContainer = document.createElement('div');
            contentContainer.classList.add('ai-topic-content', 'collapsible-content');
            if (cacheEntry && cacheEntry.data) {
                contentContainer.innerHTML = utils.formatAiResponseToHtml(cacheEntry.data);
                if (cacheEntry.timestamp && cacheEntry.timestamp < oldestTimestamp) {
                    oldestTimestamp = cacheEntry.timestamp;
                }
                if (!cacheEntry.timestamp || (Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000)) {
                    needsRefresh = true;
                }
            } else {
                contentContainer.innerHTML = "<p>No data available. Try refreshing.</p>";
                needsRefresh = true; 
            }
            sectionDiv.appendChild(contentContainer);
            ui.infoModalContent.appendChild(sectionDiv);
        });
        const overallAge = (topics.length > 0 && oldestTimestamp !== Date.now()) ? oldestTimestamp : null;
        ui.infoModalUpdated.textContent = `Updated ${utils.formatTimeAgo(overallAge)}`;
        if (needsRefresh) {
            ui.refreshInfoButton.classList.remove('hidden');
            ui.refreshInfoButton.dataset.locationId = location.id;
        }
        ui.openModal('infoModal');
        console.log("Info modal displayed for location:", locationData.description);
    },

    displayInfoModalError: (topicDescription, errorMessage) => {
        const errorP = document.createElement('p');
        errorP.classList.add('error-message');
        errorP.textContent = `Error loading ${topicDescription}: ${errorMessage}`;
        ui.infoModalContent.appendChild(errorP);
        console.error(`Error displayed in info modal for ${topicDescription}: ${errorMessage}`);
    },

    clearInputFields: (fields) => {
        fields.forEach(field => {
            if (field) field.value = '';
        });
    },

    init: () => {
        ui.initModalCloseButtons();
        // Drag and drop setup is now handled in app.js setupEventListeners
        console.log("UI initialized");
    }
};
ui.init();
