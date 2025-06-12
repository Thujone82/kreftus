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
    backgroundColorInput: document.getElementById('backgroundColor'), // This is for content area
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

    // Helper to determine if a color is dark
    isColorDark: (hexColor) => {
        const color = (hexColor.charAt(0) === '#') ? hexColor.substring(1, 7) : hexColor;
        const r = parseInt(color.substring(0, 2), 16); // hexToR
        const g = parseInt(color.substring(2, 4), 16); // hexToG
        const b = parseInt(color.substring(4, 6), 16); // hexToB
        // Calculate luminance
        const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        return luminance < 0.5;
    },

    // Helper to adjust brightness (basic)
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

    // Theme Application
    applyTheme: (primaryColor, contentBackgroundColor) => {
        document.documentElement.style.setProperty('--primary-color', primaryColor);
        document.documentElement.style.setProperty('--content-background-color', contentBackgroundColor);

        let pageBackgroundColor;
        let textColor;

        if (ui.isColorDark(contentBackgroundColor)) {
            textColor = '#E0E0E0'; // Light text for dark content background
            // Make page background slightly lighter than content, or a fixed very dark if content is almost black
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 5); 
            if (pageBackgroundColor === contentBackgroundColor) { // if content is already very dark
                 pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, 10) === contentBackgroundColor ? '#121212' : ui.adjustColorBrightness(contentBackgroundColor, 10);
            }
        } else {
            textColor = '#121212'; // Dark text for light content background
            // Make page background slightly darker than content
            pageBackgroundColor = ui.adjustColorBrightness(contentBackgroundColor, -5);
             if (pageBackgroundColor === contentBackgroundColor) { // if content is already very light
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

    // App Config UI
    loadAppConfigForm: (settings) => {
        ui.apiKeyInput.value = settings.apiKey || '';
        ui.primaryColorInput.value = settings.primaryColor;
        ui.backgroundColorInput.value = settings.backgroundColor; // This is content area BG
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

    // Location Buttons on Main Page
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

    // Generic List Rendering for Config Modals (Locations & Topics)
    renderConfigList: (items, listElement, type, onRemoveCallback) => {
        listElement.innerHTML = ''; 
        items.forEach((item, index) => {
            const li = document.createElement('li');
            li.textContent = `${item.description} ${type === 'location' ? '(' + item.location + ')' : '(' + item.aiQuery + ')'}`;
            li.dataset.id = item.id;
            li.dataset.index = index;
            li.draggable = true;

            const removeBtn = document.createElement('button');
            removeBtn.textContent = 'Remove';
            removeBtn.classList.add('danger');
            removeBtn.onclick = (e) => {
                e.stopPropagation(); 
                onRemoveCallback(item.id);
            };
            li.appendChild(removeBtn);
            listElement.appendChild(li);
        });
        console.log(`${type} list rendered with ${items.length} items`);
    },

    // Drag and Drop for Sortable Lists
    enableDragAndDrop: (listElement, onSortCallback) => {
        let draggedItem = null;

        listElement.addEventListener('dragstart', (e) => {
            draggedItem = e.target;
            setTimeout(() => {
                if(e.target.style) e.target.style.opacity = '0.5';
            }, 0);
            console.log("Drag started:", draggedItem.dataset.id);
        });

        listElement.addEventListener('dragend', (e) => {
            setTimeout(() => {
                if(e.target.style) e.target.style.opacity = '1';
                draggedItem = null;
            }, 0);
            onSortCallback(Array.from(listElement.children).map(li => li.dataset.id));
            console.log("Drag ended");
        });

        listElement.addEventListener('dragover', (e) => {
            e.preventDefault();
            const afterElement = getDragAfterElement(listElement, e.clientY);
            if (draggedItem) { // Ensure draggedItem is not null
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
        console.log(`Drag and drop enabled for list:`, listElement.id);
    },

    // Info Modal UI
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
            const titleH3 = document.createElement('h3');
            titleH3.textContent = topic.description;
            sectionDiv.appendChild(titleH3);

            const contentContainer = document.createElement('div');
            contentContainer.classList.add('ai-topic-content'); // Add class for styling

            if (cacheEntry && cacheEntry.data) {
                contentContainer.innerHTML = utils.formatAiResponseToHtml(cacheEntry.data);
                if (cacheEntry.timestamp < oldestTimestamp) {
                    oldestTimestamp = cacheEntry.timestamp;
                }
                if ((Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000)) { // 60 minutes
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

    // Clear input fields
    clearInputFields: (fields) => {
        fields.forEach(field => {
            if (field) field.value = '';
        });
    },

    // Initial setup
    init: () => {
        ui.initModalCloseButtons();

        const locationsList = document.getElementById('locationsList');
        if (locationsList) {
            ui.enableDragAndDrop(locationsList, (newOrderIds) => {
                // app.js will handle saving on "Save Changes" button click.
                // The callback in app.js should update app.currentEditingLocations
            });
        }

        const topicsList = document.getElementById('topicsList');
        if (topicsList) {
            ui.enableDragAndDrop(topicsList, (newOrderIds) => {
                 // app.js will handle saving on "Save Changes" button click.
                 // The callback in app.js should update app.currentEditingTopics
            });
        }
        console.log("UI initialized");
    }
};

ui.init();
