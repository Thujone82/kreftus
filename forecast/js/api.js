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
            // For zipcodes, parse from display_name
            const displayName = result.display_name;
            // Format: "97219, Multnomah, Portland, Multnomah County, Oregon, United States"
            const cityMatch = displayName.match(/^\d{5}, [^,]+,\s*([^,]+),/);
            if (cityMatch) {
                city = cityMatch[1].trim();
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
            // For city/state queries, parse state from display_name
            const displayName = result.display_name;
            const stateMatch = displayName.match(/, ([A-Z]{2})(?:,|$)/);
            if (stateMatch) {
                state = stateMatch[1];
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
        
        const response = await fetch(observationsUrl, { headers: NWS_HEADERS });
        
        if (!response.ok) {
            console.error('Failed to fetch observations:', response.status, response.statusText);
            return null;
        }
        
        const observationsData = await response.json();
        console.log('Fetched observations:', observationsData.features?.length || 0, 'observations');
        return observationsData;
    } catch (error) {
        console.error('Error fetching observations:', error);
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
    const [forecastData, hourlyData, alertsData] = await Promise.all([
        fetchNWSForecast(forecastUrl),
        fetchNWSHourly(hourlyUrl),
        fetchNWSAlerts(lat, lon)
    ]);
    
    // Extract elevation from forecast data
    const elevationMeters = forecastData.properties.elevation?.value || 0;
    const elevationFeet = Math.round(elevationMeters * 3.28084);
    
    return {
        location: { lat, lon, city, state, timeZone, radarStation, elevationFeet },
        points: pointsData,
        forecast: forecastData,
        hourly: hourlyData,
        alerts: alertsData,
        fetchTime: new Date()
    };
}

