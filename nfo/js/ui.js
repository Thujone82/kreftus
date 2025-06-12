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

    // Theme Application
    applyTheme: (primaryColor, backgroundColor) => {
        document.documentElement.style.setProperty('--primary-color', primaryColor);
        document.documentElement.style.setProperty('--background-color', backgroundColor);
        // Update manifest theme-color dynamically if possible (complex, usually static)
        // For simplicity, we'll rely on initial manifest and PWA install time.
        // Meta theme-color can be updated:
        let themeColorMeta = document.querySelector('meta[name="theme-color"]');
        if (themeColorMeta) {
            themeColorMeta.setAttribute('content', primaryColor);
        } else {
            themeColorMeta = document.createElement('meta');
            themeColorMeta.name = "theme-color";
            themeColorMeta.content = primaryColor;
            document.getElementsByTagName('head')[0].appendChild(themeColorMeta);
        }
        console.log(`Theme applied: Primary=${primaryColor}, Background=${backgroundColor}`);
    },

    // App Config UI
    loadAppConfigForm: (settings) => {
        ui.apiKeyInput.value = settings.apiKey || '';
        ui.primaryColorInput.value = settings.primaryColor || '#4A90E2';
        ui.backgroundColorInput.value = settings.backgroundColor || '#F0F0F0';
        ui.appConfigError.textContent = ''; // Clear previous errors
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
        ui.locationButtonsContainer.innerHTML = ''; // Clear existing buttons
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
        listElement.innerHTML = ''; // Clear existing items
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
                e.stopPropagation(); // Prevent drag start
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
                e.target.style.opacity = '0.5';
            }, 0);
            console.log("Drag started:", draggedItem.dataset.id);
        });

        listElement.addEventListener('dragend', (e) => {
            setTimeout(() => {
                e.target.style.opacity = '1';
                draggedItem = null;
            }, 0);
            onSortCallback(Array.from(listElement.children).map(li => li.dataset.id));
            console.log("Drag ended");
        });

        listElement.addEventListener('dragover', (e) => {
            e.preventDefault();
            const afterElement = getDragAfterElement(listElement, e.clientY);
            if (afterElement == null) {
                listElement.appendChild(draggedItem);
            } else {
                listElement.insertBefore(draggedItem, afterElement);
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
        ui.infoModalContent.innerHTML = ''; // Clear previous content
        ui.refreshInfoButton.classList.add('hidden'); // Hide by default

        let oldestTimestamp = Date.now();
        let needsRefresh = false;

        topics.forEach(topic => {
            const cacheEntry = cachedData[topic.id];
            const sectionDiv = document.createElement('div');
            const titleH3 = document.createElement('h3');
            titleH3.textContent = topic.description;
            sectionDiv.appendChild(titleH3);

            const contentP = document.createElement('p');
            if (cacheEntry && cacheEntry.data) {
                contentP.textContent = cacheEntry.data;
                if (cacheEntry.timestamp < oldestTimestamp) {
                    oldestTimestamp = cacheEntry.timestamp;
                }
                if ((Date.now() - cacheEntry.timestamp) > (60 * 60 * 1000)) { // 60 minutes
                    needsRefresh = true;
                }
            } else {
                contentP.textContent = "No data available. Try refreshing.";
                needsRefresh = true; // If any topic is missing data, allow refresh
            }
            sectionDiv.appendChild(contentP);
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
        // This function is intended to be called if a specific topic fails during refresh
        // It would find the specific topic's P tag and update it.
        // For simplicity in this initial build, a full refresh might be easier.
        // Or, we can append a general error message to the modal.
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

        // Setup drag and drop for sortable lists
        const locationsList = document.getElementById('locationsList');
        if (locationsList) {
            ui.enableDragAndDrop(locationsList, (newOrderIds) => {
                // This callback will be connected in app.js to update the store
                console.log("Locations sorted, new order IDs:", newOrderIds);
                // In a real app, you'd call a function here to reorder and save.
                // For now, app.js will handle saving on "Save Changes" button click.
            });
        }

        const topicsList = document.getElementById('topicsList');
        if (topicsList) {
            ui.enableDragAndDrop(topicsList, (newOrderIds) => {
                // This callback will be connected in app.js to update the store
                console.log("Topics sorted, new order IDs:", newOrderIds);
                // Similar to locations, app.js will handle saving.
            });
        }
        console.log("UI initialized");
    }
};

// Initialize UI components on script load
ui.init();