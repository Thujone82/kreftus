// API integration for NWS, geocoding, and location detection

// Headers for NWS API requests
const NWS_HEADERS = {
    "Accept": "application/geo+json",
    "User-Agent": "GetForecast/1.0 (081625PDX)"
};

// Exponential backoff retry logic
async function fetchWithRetry(url, options = {}, maxRetries = 10) {
    const baseDelay = 1000; // 1 second
    
    for (let retryCount = 0; retryCount < maxRetries; retryCount++) {
        try {
            const response = await fetch(url, options);
            
            if (response.status === 503) {
                // Service unavailable - retry with exponential backoff
                if (retryCount < maxRetries - 1) {
                    const delay = Math.min(baseDelay * Math.pow(2, retryCount), 512000); // Cap at 512 seconds
                    await new Promise(resolve => setTimeout(resolve, delay));
                    continue;
                }
            }
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            if (retryCount < maxRetries - 1) {
                const delay = Math.min(baseDelay * Math.pow(2, retryCount), 512000);
                await new Promise(resolve => setTimeout(resolve, delay));
                continue;
            }
            throw error;
        }
    }
    
    throw new Error("Service unavailable after maximum retries");
}

// Geocode location using OpenStreetMap Nominatim API
async function geocodeLocation(location) {
    try {
        // Prepare location for API query
        const locationForApi = location.includes(",") ? location : `${location},US`;
        const encodedLocation = encodeURIComponent(locationForApi);
        const geoUrl = `https://nominatim.openstreetmap.org/search?q=${encodedLocation}&format=json&limit=1&countrycodes=us`;
        
        const geoData = await fetch(geoUrl);
        if (!geoData.ok) {
            throw new Error(`Geocoding failed: ${geoData.statusText}`);
        }
        
        const results = await geoData.json();
        if (!results || results.length === 0) {
            throw new Error(`No geocoding results found for '${location}'`);
        }
        
        const result = results[0];
        const lat = parseFloat(result.lat);
        const lon = parseFloat(result.lon);
        
        // Extract city and state
        let city = result.name;
        let state = "US";
        
        if (result.type === "postcode") {
            // For zipcodes, try address object first (most reliable)
            const displayName = result.display_name;
            
            // Try to get city from address object first (most reliable)
            if (result.address) {
                // Check various city fields in order of preference
                if (result.address.city) {
                    city = result.address.city;
                } else if (result.address.town) {
                    city = result.address.town;
                } else if (result.address.village) {
                    city = result.address.village;
                } else if (result.address.municipality) {
                    city = result.address.municipality;
                }
            }
            
            // If address object didn't work (city is still the zipcode), parse from display_name
            if (city === result.name) {
                // Extract city from display_name
                // Format varies: "99502, Anchorage, Alaska, United States" or "97219, Multnomah, Portland, Multnomah County, Oregon, United States"
                // Strategy: Find the element that comes before the state name
                const firstElementMatch = displayName.match(/^\d{5}, ([^,]+),/);
                if (firstElementMatch) {
                    const firstElement = firstElementMatch[1].trim();
                    // Check if first element is likely a city (not a state name)
                    const stateNames = ["Alabama", "Alaska", "Arizona", "Arkansas", "California", "Colorado", "Connecticut", 
                                       "Delaware", "Florida", "Georgia", "Hawaii", "Idaho", "Illinois", "Indiana", "Iowa",
                                       "Kansas", "Kentucky", "Louisiana", "Maine", "Maryland", "Massachusetts", "Michigan", 
                                       "Minnesota", "Mississippi", "Missouri", "Montana", "Nebraska", "Nevada", "New Hampshire", 
                                       "New Jersey", "New Mexico", "New York", "North Carolina", "North Dakota", "Ohio",
                                       "Oklahoma", "Oregon", "Pennsylvania", "Rhode Island", "South Carolina", "South Dakota", 
                                       "Tennessee", "Texas", "Utah", "Vermont", "Virginia", "Washington", "West Virginia", 
                                       "Wisconsin", "Wyoming"];
                    
                    if (stateNames.includes(firstElement)) {
                        // First element is a state, try second element
                        const cityMatch = displayName.match(/^\d{5}, [^,]+,\s*([^,]+),/);
                        if (cityMatch) {
                            city = cityMatch[1].trim();
                        }
                    } else {
                        // First element is likely the city
                        city = firstElement;
                    }
                }
            }
            
            // Extract state abbreviation
            const stateMatch = displayName.match(/, ([A-Z]{2}), United States$/);
            if (stateMatch) {
                state = stateMatch[1];
            } else {
                // Try to extract from full state name
                const stateMap = {
                    "Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR", "California": "CA",
                    "Colorado": "CO", "Connecticut": "CT", "Delaware": "DE", "Florida": "FL", "Georgia": "GA",
                    "Hawaii": "HI", "Idaho": "ID", "Illinois": "IL", "Indiana": "IN", "Iowa": "IA",
                    "Kansas": "KS", "Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD",
                    "Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS", "Missouri": "MO",
                    "Montana": "MT", "Nebraska": "NE", "Nevada": "NV", "New Hampshire": "NH", "New Jersey": "NJ",
                    "New Mexico": "NM", "New York": "NY", "North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH",
                    "Oklahoma": "OK", "Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC",
                    "South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT", "Vermont": "VT",
                    "Virginia": "VA", "Washington": "WA", "West Virginia": "WV", "Wisconsin": "WI", "Wyoming": "WY"
                };
                
                for (const [stateName, stateAbbr] of Object.entries(stateMap)) {
                    if (displayName.includes(`, ${stateName}, United States`)) {
                        state = stateAbbr;
                        break;
                    }
                }
            }
        } else {
            // For city/state queries, try to extract state from address object first (most reliable)
            if (result.address) {
                // Check for state_code (2-letter abbreviation) first
                if (result.address.state_code && result.address.state_code.length === 2) {
                    state = result.address.state_code.toUpperCase();
                }
                // If no state_code, try to map full state name to abbreviation
                else if (result.address.state) {
                    const stateMap = {
                        "Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR", "California": "CA",
                        "Colorado": "CO", "Connecticut": "CT", "Delaware": "DE", "Florida": "FL", "Georgia": "GA",
                        "Hawaii": "HI", "Idaho": "ID", "Illinois": "IL", "Indiana": "IN", "Iowa": "IA",
                        "Kansas": "KS", "Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD",
                        "Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS", "Missouri": "MO",
                        "Montana": "MT", "Nebraska": "NE", "Nevada": "NV", "New Hampshire": "NH", "New Jersey": "NJ",
                        "New Mexico": "NM", "New York": "NY", "North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH",
                        "Oklahoma": "OK", "Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC",
                        "South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT", "Vermont": "VT",
                        "Virginia": "VA", "Washington": "WA", "West Virginia": "WV", "Wisconsin": "WI", "Wyoming": "WY",
                        "District of Columbia": "DC"
                    };
                    if (stateMap[result.address.state]) {
                        state = stateMap[result.address.state];
                    }
                }
            }
            
            // Fallback: Parse state from display_name if address object didn't work
            if (!state || state === "US") {
                const displayName = result.display_name;
                const stateMatch = displayName.match(/, ([A-Z]{2})(?:,|$)/);
                if (stateMatch) {
                    state = stateMatch[1];
                } else {
                    // Try to extract full state name from display_name and map it
                    const stateMap = {
                        "Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR", "California": "CA",
                        "Colorado": "CO", "Connecticut": "CT", "Delaware": "DE", "Florida": "FL", "Georgia": "GA",
                        "Hawaii": "HI", "Idaho": "ID", "Illinois": "IL", "Indiana": "IN", "Iowa": "IA",
                        "Kansas": "KS", "Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD",
                        "Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS", "Missouri": "MO",
                        "Montana": "MT", "Nebraska": "NE", "Nevada": "NV", "New Hampshire": "NH", "New Jersey": "NJ",
                        "New Mexico": "NM", "New York": "NY", "North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH",
                        "Oklahoma": "OK", "Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC",
                        "South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT", "Vermont": "VT",
                        "Virginia": "VA", "Washington": "WA", "West Virginia": "WV", "Wisconsin": "WI", "Wyoming": "WY",
                        "District of Columbia": "DC"
                    };
                    for (const [stateName, stateAbbr] of Object.entries(stateMap)) {
                        if (displayName.includes(`, ${stateName},`)) {
                            state = stateAbbr;
                            break;
                        }
                    }
                }
            }
        }
        
        return { lat, lon, city, state };
    } catch (error) {
        throw new Error(`Geocoding error: ${error.message}`);
    }
}

// Detect current location using browser geolocation API
async function detectCurrentLocation() {
    return new Promise((resolve, reject) => {
        if (!navigator.geolocation) {
            // Fallback to IP-based geolocation
            console.log('Geolocation API not available, using IP-based geolocation');
            detectLocationByIP()
                .then(resolve)
                .catch(reject);
            return;
        }
        
        console.log('Attempting browser geolocation...');
        navigator.geolocation.getCurrentPosition(
            async (position) => {
                console.log('Geolocation successful:', position.coords);
                const lat = position.coords.latitude;
                const lon = position.coords.longitude;
                
                // Reverse geocode to get city and state
                try {
                    const reverseGeoUrl = `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lon}&format=json&addressdetails=1`;
                    const response = await fetch(reverseGeoUrl);
                    const data = await response.json();
                    
                    const city = data.address?.city || data.address?.town || data.address?.village || "Unknown";
                    const state = data.address?.state_code || data.address?.state || "US";
                    
                    resolve({ lat, lon, city, state });
                } catch (error) {
                    // If reverse geocoding fails, still return coordinates
                    console.warn('Reverse geocoding failed, using coordinates only:', error);
                    resolve({ lat, lon, city: "Unknown", state: "US" });
                }
            },
            (error) => {
                // Geolocation denied or failed - fallback to IP-based
                console.log('Geolocation failed, falling back to IP-based geolocation:', error.code, error.message);
                detectLocationByIP()
                    .then(resolve)
                    .catch((ipError) => {
                        console.error('IP geolocation also failed:', ipError);
                        reject(ipError);
                    });
            },
            { 
                timeout: 10000, 
                maximumAge: 600000,
                enableHighAccuracy: false // Don't require high accuracy, faster fallback
            }
        );
    });
}

// Detect location using IP address (fallback)
async function detectLocationByIP() {
    try {
        console.log('Attempting IP-based geolocation...');
        // Try multiple IP geolocation services for better browser compatibility
        // ip-api.com free tier has CORS restrictions for browser requests
        // Try ipapi.co first (better CORS support), then fallback to ip-api.com
        
        const services = [
            {
                name: 'ipapi.co',
                url: 'https://ipapi.co/json/',
                parse: (data) => ({
                    lat: data.latitude,
                    lon: data.longitude,
                    city: data.city,
                    state: data.region_code || data.region
                })
            },
            {
                name: 'ip-api.com (HTTPS)',
                url: 'https://ip-api.com/json/',
                parse: (data) => {
                    if (data.status === "success") {
                        return {
                            lat: data.lat,
                            lon: data.lon,
                            city: data.city,
                            state: data.regionName
                        };
                    }
                    throw new Error(data.message || "IP geolocation failed");
                }
            },
            {
                name: 'ip-api.com (HTTP)',
                url: 'http://ip-api.com/json/',
                parse: (data) => {
                    if (data.status === "success") {
                        return {
                            lat: data.lat,
                            lon: data.lon,
                            city: data.city,
                            state: data.regionName
                        };
                    }
                    throw new Error(data.message || "IP geolocation failed");
                }
            }
        ];
        
        for (const service of services) {
            try {
                console.log(`Trying ${service.name}:`, service.url);
                const response = await fetch(service.url, {
                    method: 'GET',
                    headers: {
                        'Accept': 'application/json'
                    }
                });
                
                if (!response.ok) {
                    console.log(`${service.name} returned error:`, response.status, response.statusText);
                    continue; // Try next service
                }
                
                const data = await response.json();
                console.log(`${service.name} response:`, data);
                
                const result = service.parse(data);
                console.log(`Successfully detected location using ${service.name}:`, result);
                return result;
            } catch (error) {
                console.log(`${service.name} failed:`, error.message);
                // Continue to next service
                continue;
            }
        }
        
        // All services failed
        throw new Error('All IP geolocation services failed');
    } catch (error) {
        console.error('IP geolocation error:', error);
        // Re-throw with more context
        if (error.message.includes('fetch') || error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
            throw new Error(`Network error: Unable to reach IP geolocation service. Please check your internet connection.`);
        } else if (error.message.includes('CORS') || error.message.includes('Mixed Content') || error.message.includes('blocked')) {
            throw new Error(`CORS error: IP geolocation service is not accessible. Please try entering a location manually.`);
        } else {
            throw new Error(`Unable to detect location: ${error.message}`);
        }
    }
}

// Fetch NWS points data
async function fetchNWSPoints(lat, lon) {
    const url = `https://api.weather.gov/points/${lat},${lon}`;
    return await fetchWithRetry(url, { headers: NWS_HEADERS });
}

// Fetch NWS forecast data
async function fetchNWSForecast(forecastUrl) {
    return await fetchWithRetry(forecastUrl, { headers: NWS_HEADERS });
}

// Fetch NWS hourly forecast data
async function fetchNWSHourly(hourlyUrl) {
    return await fetchWithRetry(hourlyUrl, { headers: NWS_HEADERS });
}

// Fetch NWS alerts
async function fetchNWSAlerts(lat, lon) {
    try {
        const url = `https://api.weather.gov/alerts/active?point=${lat},${lon}`;
        const response = await fetch(url, { headers: NWS_HEADERS });
        
        if (!response.ok) {
            // Alerts are optional, return null on error
            return null;
        }
        
        return await response.json();
    } catch (error) {
        // Alerts are optional, return null on error
        return null;
    }
}

// Fetch NWS observation stations
async function fetchNWSObservationStations(pointsData) {
    try {
        const observationStationsUrl = pointsData.properties.observationStations;
        if (!observationStationsUrl) {
            console.log('No observation stations URL found in points data');
            return null;
        }
        
        console.log('Fetching observation stations from:', observationStationsUrl);
        const response = await fetch(observationStationsUrl, { headers: NWS_HEADERS });
        
        if (!response.ok) {
            console.error('Failed to fetch observation stations:', response.status, response.statusText);
            return null;
        }
        
        const stationsData = await response.json();
        
        if (!stationsData.features || stationsData.features.length === 0) {
            console.log('No observation stations found');
            return null;
        }
        
        const stationId = stationsData.features[0].properties.stationIdentifier;
        console.log('Using observation station:', stationId);
        return stationId;
    } catch (error) {
        console.error('Error fetching observation stations:', error);
        return null;
    }
}

// Fetch NWS observations
async function fetchNWSObservations(stationId, timeZone) {
    try {
        if (!stationId) {
            return null;
        }
        
        // Calculate time range (last 7 days)
        const endTime = new Date();
        const startTime = new Date();
        startTime.setDate(startTime.getDate() - 7);
        
        // Format times in ISO 8601 format (UTC)
        const startTimeStr = startTime.toISOString();
        const endTimeStr = endTime.toISOString();
        
        const observationsUrl = `https://api.weather.gov/stations/${stationId}/observations?start=${startTimeStr}&end=${endTimeStr}`;
        console.log('Fetching observations from:', observationsUrl);
        
        // Collect all observations from all pages
        const allFeatures = [];
        let currentUrl = observationsUrl;
        let pageCount = 0;
        const maxPages = 50;  // Safety limit to prevent infinite loops
        
        while (currentUrl && pageCount < maxPages) {
            pageCount++;
            console.log(`Fetching observations page ${pageCount}:`, currentUrl);
            
            const response = await fetch(currentUrl, { headers: NWS_HEADERS });
            
            if (!response.ok) {
                console.error(`Failed to fetch observations page ${pageCount}:`, response.status, response.statusText);
                break;
            }
            
            const observationsData = await response.json();
            
            // Add features from this page to our collection
            if (observationsData.features && Array.isArray(observationsData.features)) {
                allFeatures.push(...observationsData.features);
                console.log(`Collected ${observationsData.features.length} observations from page ${pageCount} (total: ${allFeatures.length})`);
            }
            
            // Check for next page
            currentUrl = null;
            if (observationsData.pagination && observationsData.pagination.next) {
                currentUrl = observationsData.pagination.next;
                console.log('Found pagination link for next page');
            }
        }
        
        if (allFeatures.length === 0) {
            console.log('No observations collected from any page');
            return null;
        }
        
        console.log(`Collected total of ${allFeatures.length} observations from ${pageCount} page(s)`);
        
        // Create a combined observations data object
        const combinedObservationsData = {
            type: 'FeatureCollection',
            features: allFeatures
        };
        
        return combinedObservationsData;
    } catch (error) {
        console.error('Error fetching observations:', error);
        return null;
    }
}

// Calculate distance between two coordinates using Haversine formula
function calculateDistanceMiles(lat1, lon1, lat2, lon2) {
    // Earth radius in miles
    const R = 3959;
    
    // Convert degrees to radians
    const lat1Rad = lat1 * Math.PI / 180;
    const lon1Rad = lon1 * Math.PI / 180;
    const lat2Rad = lat2 * Math.PI / 180;
    const lon2Rad = lon2 * Math.PI / 180;
    
    // Calculate differences
    const dLat = lat2Rad - lat1Rad;
    const dLon = lon2Rad - lon1Rad;
    
    // Haversine formula
    const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
              Math.cos(lat1Rad) * Math.cos(lat2Rad) *
              Math.sin(dLon / 2) * Math.sin(dLon / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    const distance = R * c;
    
    return distance;
}

// Cache duration for stations.json: 1 week (7 days)
const STATIONS_CACHE_DURATION_MS = 7 * 24 * 60 * 60 * 1000; // 7 days in milliseconds
const STATIONS_CACHE_KEY = 'forecastNoaaStations';
const STATIONS_CACHE_TIMESTAMP_KEY = 'forecastNoaaStationsTimestamp';

// Load cached stations.json if available and fresh
function loadCachedStations() {
    try {
        const cachedData = localStorage.getItem(STATIONS_CACHE_KEY);
        const cachedTimestamp = localStorage.getItem(STATIONS_CACHE_TIMESTAMP_KEY);
        
        if (!cachedData || !cachedTimestamp) {
            return null;
        }
        
        const timestamp = parseInt(cachedTimestamp, 10);
        const age = Date.now() - timestamp;
        
        if (age < STATIONS_CACHE_DURATION_MS) {
            const stations = JSON.parse(cachedData);
            console.log('Using cached stations.json (age:', Math.round(age / (60 * 60 * 1000)), 'hours,', stations.length, 'stations)');
            return stations;
        } else {
            console.log('Cached stations.json is stale (age:', Math.round(age / (24 * 60 * 60 * 1000)), 'days), will refresh');
            // Clear stale cache
            localStorage.removeItem(STATIONS_CACHE_KEY);
            localStorage.removeItem(STATIONS_CACHE_TIMESTAMP_KEY);
            return null;
        }
    } catch (error) {
        console.error('Error loading cached stations.json:', error);
        return null;
    }
}

// Save stations.json to cache
function saveStationsToCache(stations) {
    try {
        if (stations && Array.isArray(stations)) {
            localStorage.setItem(STATIONS_CACHE_KEY, JSON.stringify(stations));
            localStorage.setItem(STATIONS_CACHE_TIMESTAMP_KEY, Date.now().toString());
            console.log('Saved stations.json to cache (', stations.length, 'stations)');
        }
    } catch (error) {
        console.error('Error saving stations.json to cache:', error);
    }
}

// Fetch NOAA stations.json (can be called in parallel)
// Uses cache if available and fresh (1 week), otherwise fetches from API
async function fetchNoaaStationsJson() {
    try {
        // Check cache first
        const cachedStations = loadCachedStations();
        if (cachedStations) {
            return cachedStations;
        }
        
        // Cache miss or stale - fetch from API
        const apiUrl = 'https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json';
        console.log('Fetching NOAA stations.json from API:', apiUrl);
        
        const response = await fetch(apiUrl, {
            method: 'GET',
            headers: {
                'Accept': 'application/json'
            }
        });
        
        if (!response.ok) {
            console.log('NOAA stations API returned status:', response.status);
            return null;
        }
        
        const apiResponse = await response.json();
        
        if (apiResponse && apiResponse.stations) {
            const stations = apiResponse.stations;
            console.log('Fetched', stations.length, 'stations from NOAA API');
            
            // Save to cache
            saveStationsToCache(stations);
            
            return stations;
        }
        
        return null;
    } catch (error) {
        console.error('Error fetching NOAA stations.json:', error);
        return null;
    }
}

// Search NOAA tide stations by coordinates using CO-OPS Metadata API
// Can accept pre-fetched stations data for optimization
async function fetchNoaaTideStation(lat, lon, preFetchedStations = null) {
    try {
        console.log('Searching NOAA tide stations for coordinates:', lat, lon);
        
        let stations = preFetchedStations;
        
        // Use pre-fetched stations if provided, otherwise try cache, then fetch from API
        if (!stations) {
            // Try to use cached stations first (will check freshness internally)
            stations = await fetchNoaaStationsJson();
            
            // If cache returned null (stale or missing), fetchNoaaStationsJson will have fetched fresh data
            // So stations should now be populated if available
            if (!stations) {
                console.log('No stations available from cache or API');
                return null;
            }
        } else {
            console.log('Using pre-fetched stations data (' + stations.length + ' stations)');
        }
        
        let closestStation = null;
        let minDistance = 1000000;
        const maxDistanceMiles = 100;
        const allNearbyStations = [];  // Track all stations within 100 miles for logging
        
        for (const station of stations) {
            // API uses 'lng' for longitude, not 'lon'
            if (station.lat && station.lng) {
                const stationLat = parseFloat(station.lat);
                const stationLon = parseFloat(station.lng);
                
                const distance = calculateDistanceMiles(lat, lon, stationLat, stationLon);
                
                if (distance <= maxDistanceMiles) {
                    // Track all stations within 100 miles
                    allNearbyStations.push({
                        stationId: station.id.toString(),
                        name: station.name,
                        lat: stationLat,
                        lon: stationLon,
                        distance: distance
                    });
                    
                    // Update closest station if this one is closer
                    if (distance < minDistance) {
                        minDistance = distance;
                        closestStation = {
                            stationId: station.id.toString(),
                            name: station.name,
                            lat: stationLat,
                            lon: stationLon,
                            distance: distance
                        };
                    }
                }
            }
        }
        
        // Log the closest stations (sorted by distance)
        if (allNearbyStations.length > 0) {
            // Sort by distance (ascending - shortest first)
            const sortedStations = allNearbyStations.sort((a, b) => a.distance - b.distance);
            const topStations = sortedStations.slice(0, 5);
            const stationCount = topStations.length;
            
            const headerText = stationCount === 1 
                ? 'Top 1 closest NOAA station within 100 miles:'
                : `Top ${stationCount} closest NOAA stations within 100 miles:`;
            console.log(headerText);
            
            topStations.forEach(station => {
                const isSelected = (closestStation && station.stationId === closestStation.stationId);
                const marker = isSelected ? ' [SELECTED]' : '';
                console.log(`  ${station.name} (${station.stationId}) at ${station.distance.toFixed(2)} miles${marker}`);
            });
        }
        
        if (closestStation) {
            console.log('Found closest NOAA station via API:', closestStation.name, `(${closestStation.stationId}) at ${closestStation.distance.toFixed(2)} miles`);
            
            // Check for water level support via products endpoint (most reliable method)
            // Use cache if available and fresh
            const cachedWaterLevelSupport = loadCachedWaterLevelSupport(closestStation.stationId);
            if (cachedWaterLevelSupport !== null) {
                closestStation.supportsWaterLevels = cachedWaterLevelSupport;
                console.log('Using cached water level support for station', closestStation.stationId + ':', cachedWaterLevelSupport);
            } else {
                // Cache miss or stale - fetch from API
                try {
                    const productsUrl = `https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations/${closestStation.stationId}/products.json`;
                    console.log('Checking water levels support via products endpoint:', productsUrl);
                    const productsResponse = await fetch(productsUrl, {
                        method: 'GET',
                        headers: { 'Accept': 'application/json' }
                    });
                    
                    if (productsResponse.ok) {
                        const products = await productsResponse.json();
                        if (products.products && Array.isArray(products.products)) {
                            // Check if any product name contains "Water Level" or "Water Levels"
                            const waterLevelProducts = products.products.filter(product => 
                                product.name && product.name.match(/Water Level/i)
                            );
                            closestStation.supportsWaterLevels = waterLevelProducts.length > 0;
                            console.log('Water levels support from products endpoint:', closestStation.supportsWaterLevels);
                        } else {
                            closestStation.supportsWaterLevels = false;
                            console.log('No products found, assuming no water levels support');
                        }
                    } else {
                        closestStation.supportsWaterLevels = false;
                        console.log('Products endpoint returned error status:', productsResponse.status);
                    }
                    
                    // Save to cache
                    saveWaterLevelSupportToCache(closestStation.stationId, closestStation.supportsWaterLevels);
                } catch (error) {
                    console.error('Could not fetch products endpoint, assuming no water levels support:', error);
                    closestStation.supportsWaterLevels = false;
                    // Save false to cache to avoid repeated failed attempts
                    saveWaterLevelSupportToCache(closestStation.stationId, false);
                }
            }
            
            return closestStation;
        } else {
            console.log('No stations found within 100 miles via API');
            return null;
        }
    } catch (error) {
        console.error('Error searching NOAA tide stations:', error);
        return null;
    }
}

// Fetch NOAA tide predictions for a specific date
async function fetchNoaaTidePredictionsForDate(stationId, date) {
    try {
        const apiUrl = `https://api.tidesandcurrents.noaa.gov/api/prod/datagetter?product=predictions&datum=mllw&station=${stationId}&date=${date}&interval=hilo&format=json&units=english&time_zone=lst_ldt`;
        console.log(`Fetching tide predictions for date '${date}':`, apiUrl);
        
        const response = await fetch(apiUrl, {
            method: 'GET',
            headers: { 'Accept': 'application/json' }
        });
        
        if (!response.ok) {
            console.log(`Tide predictions API returned status ${response.status} for date '${date}'`);
            return null;
        }
        
        const data = await response.json();
        
        if (!data.predictions || data.predictions.length === 0) {
            console.log(`No tide predictions returned for date '${date}'`);
            return null;
        }
        
        return data.predictions;
    } catch (error) {
        console.error(`Error fetching tide predictions for date '${date}':`, error);
        return null;
    }
}

// Cache duration for tide predictions: 10 minutes (same as weather data)
const TIDE_CACHE_DURATION_MS = 10 * 60 * 1000; // 10 minutes in milliseconds

// Cache duration for water level support: 1 week (static property, rarely changes)
const WATER_LEVEL_SUPPORT_CACHE_DURATION_MS = 7 * 24 * 60 * 60 * 1000; // 7 days in milliseconds

// Load cached tide predictions if available and fresh
function loadCachedTidePredictions(stationId) {
    try {
        const cacheKey = `forecastTidePredictions_${stationId}`;
        const timestampKey = `forecastTidePredictionsTimestamp_${stationId}`;
        
        const cachedData = localStorage.getItem(cacheKey);
        const cachedTimestamp = localStorage.getItem(timestampKey);
        
        if (!cachedData || !cachedTimestamp) {
            return null;
        }
        
        const timestamp = parseInt(cachedTimestamp, 10);
        const age = Date.now() - timestamp;
        
        if (age < TIDE_CACHE_DURATION_MS) {
            const tideData = JSON.parse(cachedData);
            // Restore Date objects from strings
            if (tideData.lastTide && tideData.lastTide.time) {
                tideData.lastTide.time = new Date(tideData.lastTide.time);
            }
            if (tideData.nextTide && tideData.nextTide.time) {
                tideData.nextTide.time = new Date(tideData.nextTide.time);
            }
            console.log('Using cached tide predictions for station', stationId, '(age:', Math.round(age / 1000), 'seconds)');
            return tideData;
        } else {
            console.log('Cached tide predictions are stale (age:', Math.round(age / 60000), 'minutes), will refresh');
            // Clear stale cache
            localStorage.removeItem(cacheKey);
            localStorage.removeItem(timestampKey);
            return null;
        }
    } catch (error) {
        console.error('Error loading cached tide predictions:', error);
        return null;
    }
}

// Save tide predictions to cache
function saveTidePredictionsToCache(stationId, tideData) {
    try {
        if (tideData && stationId) {
            const cacheKey = `forecastTidePredictions_${stationId}`;
            const timestampKey = `forecastTidePredictionsTimestamp_${stationId}`;
            localStorage.setItem(cacheKey, JSON.stringify(tideData));
            localStorage.setItem(timestampKey, Date.now().toString());
            console.log('Saved tide predictions to cache for station', stationId);
        }
    } catch (error) {
        console.error('Error saving tide predictions to cache:', error);
    }
}

// Load cached water level support if available and fresh
function loadCachedWaterLevelSupport(stationId) {
    try {
        const cacheKey = `forecastWaterLevelSupport_${stationId}`;
        const timestampKey = `forecastWaterLevelSupportTimestamp_${stationId}`;
        
        const cachedValue = localStorage.getItem(cacheKey);
        const cachedTimestamp = localStorage.getItem(timestampKey);
        
        if (cachedValue === null || !cachedTimestamp) {
            return null;
        }
        
        const timestamp = parseInt(cachedTimestamp, 10);
        const age = Date.now() - timestamp;
        
        if (age < WATER_LEVEL_SUPPORT_CACHE_DURATION_MS) {
            const supportsWaterLevels = cachedValue === 'true';
            console.log('Using cached water level support for station', stationId, '(age:', Math.round(age / (60 * 60 * 1000)), 'hours):', supportsWaterLevels);
            return supportsWaterLevels;
        } else {
            console.log('Cached water level support is stale (age:', Math.round(age / (24 * 60 * 60 * 1000)), 'days), will refresh');
            // Clear stale cache
            localStorage.removeItem(cacheKey);
            localStorage.removeItem(timestampKey);
            return null;
        }
    } catch (error) {
        console.error('Error loading cached water level support:', error);
        return null;
    }
}

// Save water level support to cache
function saveWaterLevelSupportToCache(stationId, supportsWaterLevels) {
    try {
        if (stationId !== null && stationId !== undefined) {
            const cacheKey = `forecastWaterLevelSupport_${stationId}`;
            const timestampKey = `forecastWaterLevelSupportTimestamp_${stationId}`;
            localStorage.setItem(cacheKey, supportsWaterLevels ? 'true' : 'false');
            localStorage.setItem(timestampKey, Date.now().toString());
            console.log('Saved water level support to cache for station', stationId + ':', supportsWaterLevels);
        }
    } catch (error) {
        console.error('Error saving water level support to cache:', error);
    }
}

// Fetch NOAA tide predictions
// Uses cache if available and fresh (10 minutes), otherwise fetches from API
async function fetchNoaaTidePredictions(stationId, timeZone) {
    try {
        // Check cache first
        const cachedTideData = loadCachedTidePredictions(stationId);
        if (cachedTideData) {
            return cachedTideData;
        }
        
        // Cache miss or stale - fetch from API
        const now = new Date();
        
        // First, fetch today's predictions
        const todayPredictions = await fetchNoaaTidePredictionsForDate(stationId, 'today');
        
        if (!todayPredictions || todayPredictions.length === 0) {
            console.log('No tide predictions returned from API for today');
            return null;
        }
        
        // Parse today's predictions and find last and next tide
        let lastTide = null;
        let nextTide = null;
        
        for (const prediction of todayPredictions) {
            const timeStr = prediction.t;
            const tideTime = new Date(timeStr);
            
            if (isNaN(tideTime.getTime())) {
                continue;
            }
            
            const height = parseFloat(prediction.v);
            const type = prediction.type;
            
            if (tideTime <= now) {
                if (!lastTide || tideTime > lastTide.time) {
                    lastTide = {
                        time: tideTime,
                        height: height,
                        type: type
                    };
                }
            } else {
                if (!nextTide || tideTime < nextTide.time) {
                    nextTide = {
                        time: tideTime,
                        height: height,
                        type: type
                    };
                }
            }
        }
        
        // If no next tide found, fetch tomorrow's predictions
        if (!nextTide) {
            console.log('No next tide found for today, fetching tomorrow\'s predictions');
            const tomorrow = new Date(now);
            tomorrow.setDate(tomorrow.getDate() + 1);
            const tomorrowDateStr = tomorrow.toISOString().slice(0, 10).replace(/-/g, '');
            const tomorrowPredictions = await fetchNoaaTidePredictionsForDate(stationId, tomorrowDateStr);
            
            if (tomorrowPredictions && tomorrowPredictions.length > 0) {
                // Get the first tide from tomorrow (earliest future tide)
                let firstTomorrowTide = null;
                for (const prediction of tomorrowPredictions) {
                    const timeStr = prediction.t;
                    const tideTime = new Date(timeStr);
                    
                    if (isNaN(tideTime.getTime())) {
                        continue;
                    }
                    
                    if (tideTime > now) {
                        if (!firstTomorrowTide || tideTime < firstTomorrowTide.time) {
                            firstTomorrowTide = {
                                time: tideTime,
                                height: parseFloat(prediction.v),
                                type: prediction.type
                            };
                        }
                    }
                }
                
                if (firstTomorrowTide) {
                    nextTide = firstTomorrowTide;
                    console.log(`Found next tide from tomorrow: ${firstTomorrowTide.time} ${firstTomorrowTide.type}`);
                }
            }
        }
        
        // If no last tide found, fetch yesterday's predictions
        if (!lastTide) {
            console.log('No last tide found for today, fetching yesterday\'s predictions');
            const yesterday = new Date(now);
            yesterday.setDate(yesterday.getDate() - 1);
            const yesterdayDateStr = yesterday.toISOString().slice(0, 10).replace(/-/g, '');
            const yesterdayPredictions = await fetchNoaaTidePredictionsForDate(stationId, yesterdayDateStr);
            
            if (yesterdayPredictions && yesterdayPredictions.length > 0) {
                // Get the last tide from yesterday (latest past tide)
                let lastYesterdayTide = null;
                for (const prediction of yesterdayPredictions) {
                    const timeStr = prediction.t;
                    const tideTime = new Date(timeStr);
                    
                    if (isNaN(tideTime.getTime())) {
                        continue;
                    }
                    
                    if (tideTime <= now) {
                        if (!lastYesterdayTide || tideTime > lastYesterdayTide.time) {
                            lastYesterdayTide = {
                                time: tideTime,
                                height: parseFloat(prediction.v),
                                type: prediction.type
                            };
                        }
                    }
                }
                
                if (lastYesterdayTide) {
                    lastTide = lastYesterdayTide;
                    console.log(`Found last tide from yesterday: ${lastYesterdayTide.time} ${lastYesterdayTide.type}`);
                }
            }
        }
        
        // Return data if we have at least one tide (last or next)
        if (lastTide || nextTide) {
            if (lastTide) {
                console.log('Found last tide:', lastTide.time, lastTide.type === 'H' ? 'High' : 'Low', lastTide.height + 'ft');
            } else {
                console.log('No last tide found (all tides are in the future)');
            }
            
            if (nextTide) {
                console.log('Found next tide:', nextTide.time, nextTide.type === 'H' ? 'High' : 'Low', nextTide.height + 'ft');
            } else {
                console.log('No next tide found (all tides are in the past)');
            }
            
            const tideData = {
                lastTide: lastTide,
                nextTide: nextTide
            };
            
            // Save to cache
            saveTidePredictionsToCache(stationId, tideData);
            
            return tideData;
        }
        
        console.log('Could not determine any tide from predictions');
        return null;
    } catch (error) {
        console.error('Error fetching tide predictions:', error);
        return null;
    }
}

// Fetch all weather data for a location
async function fetchWeatherData(location) {
    let lat, lon, city, state;
    
    // Geocode location or detect current location
    if (location.toLowerCase() === "here") {
        const locationData = await detectCurrentLocation();
        lat = locationData.lat;
        lon = locationData.lon;
        city = locationData.city;
        state = locationData.state;
    } else {
        const locationData = await geocodeLocation(location);
        lat = locationData.lat;
        lon = locationData.lon;
        city = locationData.city;
        state = locationData.state;
    }
    
    // Start fetching NOAA stations.json in parallel (doesn't depend on NWS data)
    const noaaStationsPromise = fetchNoaaStationsJson();
    
    // Fetch NWS points data
    const pointsData = await fetchNWSPoints(lat, lon);
    
    // Extract forecast URLs and metadata
    const forecastUrl = pointsData.properties.forecast;
    const hourlyUrl = pointsData.properties.forecastHourly;
    const office = pointsData.properties.cwa;
    const gridX = pointsData.properties.gridX;
    const gridY = pointsData.properties.gridY;
    const timeZone = pointsData.properties.timeZone;
    const radarStation = pointsData.properties.radarStation;
    
    // Fetch forecast and hourly data concurrently
    const [forecastData, hourlyData, alertsData, preFetchedStations] = await Promise.all([
        fetchNWSForecast(forecastUrl),
        fetchNWSHourly(hourlyUrl),
        fetchNWSAlerts(lat, lon),
        noaaStationsPromise  // NOAA stations fetch in parallel
    ]);
    
    // Extract elevation from forecast data
    const elevationMeters = forecastData.properties.elevation?.value || 0;
    const elevationFeet = Math.round(elevationMeters * 3.28084);
    
    // Fetch NOAA tide station data using pre-fetched stations (non-blocking, don't fail if it errors)
    let noaaStation = null;
    try {
        noaaStation = await fetchNoaaTideStation(lat, lon, preFetchedStations);
        
        // If we have a station, fetch tide predictions (products already fetched in fetchNoaaTideStation)
        if (noaaStation) {
            try {
                noaaStation.tideData = await fetchNoaaTidePredictions(noaaStation.stationId, timeZone);
            } catch (error) {
                console.error('Error fetching tide predictions:', error);
                // Continue without tide data
            }
        }
    } catch (error) {
        console.error('Error fetching NOAA station data:', error);
        // Continue without NOAA data
    }
    
    return {
        location: { lat, lon, city, state, timeZone, radarStation, elevationFeet },
        points: pointsData,
        forecast: forecastData,
        hourly: hourlyData,
        alerts: alertsData,
        noaaStation: noaaStation,
        fetchTime: new Date()
    };
}

