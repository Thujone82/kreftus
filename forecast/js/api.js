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
            detectLocationByIP()
                .then(resolve)
                .catch(reject);
            return;
        }
        
        navigator.geolocation.getCurrentPosition(
            async (position) => {
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
                    resolve({ lat, lon, city: "Unknown", state: "US" });
                }
            },
            (error) => {
                // Geolocation denied or failed - fallback to IP-based
                detectLocationByIP()
                    .then(resolve)
                    .catch(reject);
            },
            { timeout: 10000, maximumAge: 600000 }
        );
    });
}

// Detect location using IP address (fallback)
async function detectLocationByIP() {
    try {
        const response = await fetch("https://ip-api.com/json/");
        if (!response.ok) {
            throw new Error("IP geolocation failed");
        }
        
        const data = await response.json();
        if (data.status === "success") {
            return {
                lat: data.lat,
                lon: data.lon,
                city: data.city,
                state: data.regionName
            };
        } else {
            throw new Error(data.message || "IP geolocation failed");
        }
    } catch (error) {
        throw new Error(`Unable to detect location: ${error.message}`);
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

