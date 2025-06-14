console.log("stor.js loaded");

const STORE_PREFIX = 'info2go_';

const APP_SETTINGS_KEY = `${STORE_PREFIX}appSettings`;
const LOCATIONS_KEY = `${STORE_PREFIX}locations`;
const TOPICS_KEY = `${STORE_PREFIX}topics`;
const AI_CACHE_PREFIX = `${STORE_PREFIX}aiCache_`;
const TOPIC_COLLAPSED_STATE_PREFIX = `${STORE_PREFIX}topicCollapsed_`;

const store = {
    // Application Settings (API Key, Colors)
    getAppSettings: () => {
        const settings = localStorage.getItem(APP_SETTINGS_KEY);
        // Default to a dark theme content background
        return settings ? JSON.parse(settings) : { 
            apiKey: '', 
            primaryColor: '#029ec5', 
            backgroundColor: '#1E1E1E' // Default content area background
        };
    },
    saveAppSettings: (settings) => {
        localStorage.setItem(APP_SETTINGS_KEY, JSON.stringify(settings));
        console.log("App settings saved:", settings);
    },

    // Locations
    getLocations: () => {
        const locations = localStorage.getItem(LOCATIONS_KEY);
        return locations ? JSON.parse(locations) : [];
    },
    saveLocations: (locations) => {
        localStorage.setItem(LOCATIONS_KEY, JSON.stringify(locations));
        console.log("Locations saved:", locations);
    },

    // Topics
    getTopics: () => {
        const topics = localStorage.getItem(TOPICS_KEY);
        return topics ? JSON.parse(topics) : [];
    },
    saveTopics: (topics) => {
        localStorage.setItem(TOPICS_KEY, JSON.stringify(topics));
        console.log("Topics saved:", topics);
    },

    // AI Query Cache
    getAiCache: (locationId, topicId) => {
        const cacheKey = `${AI_CACHE_PREFIX}${locationId}_${topicId}`;
        const cachedItem = localStorage.getItem(cacheKey);
        return cachedItem ? JSON.parse(cachedItem) : null;
    },
    saveAiCache: (locationId, topicId, data) => {
        const cacheKey = `${AI_CACHE_PREFIX}${locationId}_${topicId}`;
        const itemToCache = {
            timestamp: Date.now(),
            data: data
        };
        localStorage.setItem(cacheKey, JSON.stringify(itemToCache));
        console.log(`AI Cache saved for ${locationId} - ${topicId}`);
    },
    flushAiCacheForLocation: (locationId) => {
        console.log(`Flushing AI cache for location ID: ${locationId}`);
        for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key && key.startsWith(`${AI_CACHE_PREFIX}${locationId}_`)) {
                localStorage.removeItem(key);
                console.log(`Removed cache item: ${key}`);
            }
        }
    },
    flushAllAiCache: () => {
        console.log("Flushing all AI cache");
        for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key && key.startsWith(AI_CACHE_PREFIX)) {
                localStorage.removeItem(key);
                i--; // Adjust index as localStorage length changes
            }
        }
    },

    // Topic Collapsed State
    getTopicCollapsedState: (locationId, topicId) => {
        const key = `${TOPIC_COLLAPSED_STATE_PREFIX}${locationId}_${topicId}`;
        const state = localStorage.getItem(key);
        return state ? JSON.parse(state) : null; // null means no preference, default to expanded
    },
    saveTopicCollapsedState: (locationId, topicId, isCollapsed) => {
        const key = `${TOPIC_COLLAPSED_STATE_PREFIX}${locationId}_${topicId}`;
        localStorage.setItem(key, JSON.stringify(isCollapsed));
    }
};
