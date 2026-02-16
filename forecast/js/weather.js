// Weather data processing and calculations

// Process weather data from API responses
function processWeatherData(weatherData) {
    const { location, forecast, hourly, alerts } = weatherData;
    
    // Extract current conditions from first hourly period
    const currentPeriod = hourly.properties.periods[0];
    const nextHourPeriod = hourly.properties.periods[1];
    
    const currentTemp = currentPeriod.temperature;
    const currentConditions = currentPeriod.shortForecast;
    const currentWind = currentPeriod.windSpeed;
    const currentWindDir = currentPeriod.windDirection;
    const currentHumidity = currentPeriod.relativeHumidity?.value || 0;
    const currentPrecipProb = currentPeriod.probabilityOfPrecipitation?.value || 0;
    const currentIcon = currentPeriod.icon;
    
    // Extract dew point if available
    let currentDewPoint = null;
    if (currentPeriod.dewpoint?.value !== null && currentPeriod.dewpoint?.value !== undefined) {
        const dewPointCelsius = currentPeriod.dewpoint.value;
        currentDewPoint = Math.round(dewPointCelsius * 9/5 + 32 * 10) / 10; // Convert to Fahrenheit
    }
    
    // Calculate temperature trend
    const nextHourTemp = nextHourPeriod ? nextHourPeriod.temperature : currentTemp;
    const currentTempTrend = calculateTemperatureTrend(currentTemp, nextHourTemp);
    
    // Extract wind gust information
    let windGust = null;
    let windSpeedOnly = currentWind;
    const windGustMatch = currentWind.match(/(\d+)\s*to\s*(\d+)\s*mph/);
    if (windGustMatch) {
        windGust = windGustMatch[2];
        windSpeedOnly = `${windGustMatch[1]} mph`;
    }
    
    // Determine if currently daytime
    const isCurrentlyDaytime = currentPeriod.isDaytime !== undefined 
        ? currentPeriod.isDaytime 
        : (new Date().getHours() >= 6 && new Date().getHours() < 18);
    
    // Get weather icon
    const weatherIcon = getWeatherIcon(currentIcon, isCurrentlyDaytime, currentPrecipProb);
    
    // Calculate sunrise and sunset
    // Use the location's current date (not the viewer's local date) for accuracy
    const locationToday = convertToTimeZone(new Date(), location.timeZone);
    // Create date in UTC to avoid timezone issues when extracting year/month/day
    const locationDate = new Date(Date.UTC(
        locationToday.getFullYear(),
        locationToday.getMonth(),
        locationToday.getDate()
    ));
    const sunTimes = calculateSunriseSunset(
        location.lat,
        location.lon,
        locationDate,
        location.timeZone
    );
    
    // Calculate moon phase
    const moonPhaseInfo = calculateMoonPhase(new Date());
    
    // Calculate wind chill or heat index
    const tempNum = parseFloat(currentTemp);
    const windSpeedNum = getWindSpeed(windSpeedOnly);
    let windChill = null;
    let heatIndex = null;
    
    if (tempNum <= 50) {
        windChill = calculateWindChill(tempNum, windSpeedNum);
        if (windChill && Math.abs(tempNum - windChill) <= 1) {
            windChill = null; // Only show if difference > 1°F
        }
    } else if (tempNum >= 80) {
        heatIndex = calculateHeatIndex(tempNum, currentHumidity);
        if (heatIndex && Math.abs(heatIndex - tempNum) <= 1) {
            heatIndex = null; // Only show if difference > 1°F
        }
    }
    
    // Extract today's and tomorrow's forecasts
    const todayPeriod = forecast.properties.periods[0];
    const tomorrowPeriod = forecast.properties.periods[1];
    
    const todayForecast = todayPeriod.detailedForecast;
    const todayPeriodName = todayPeriod.name;
    const tomorrowForecast = tomorrowPeriod ? tomorrowPeriod.detailedForecast : "";
    const tomorrowPeriodName = tomorrowPeriod ? tomorrowPeriod.name : "";
    
    return {
        current: {
            temp: currentTemp,
            conditions: currentConditions,
            wind: windSpeedOnly,
            windDir: currentWindDir,
            windGust: windGust,
            humidity: currentHumidity,
            dewPoint: currentDewPoint,
            precipProb: currentPrecipProb,
            icon: weatherIcon,
            trend: currentTempTrend,
            windChill: windChill,
            heatIndex: heatIndex,
            time: weatherData.fetchTime
        },
        forecast: {
            today: {
                name: todayPeriodName,
                text: todayForecast
            },
            tomorrow: {
                name: tomorrowPeriodName,
                text: tomorrowForecast
            },
            periods: forecast.properties.periods
        },
        hourly: {
            periods: hourly.properties.periods
        },
        alerts: alerts?.features || [],
        location: {
            ...location,
            sunrise: sunTimes.sunrise,
            sunset: sunTimes.sunset,
            isPolarNight: sunTimes.isPolarNight,
            isPolarDay: sunTimes.isPolarDay,
            moonPhase: moonPhaseInfo
        },
        noaaStation: weatherData.noaaStation || null
    };
}

// Group hourly periods by day (converting to location timezone)
function groupHourlyByDay(periods, timeZoneId) {
    const daysData = {};
    
    periods.forEach(period => {
        const periodTime = new Date(period.startTime);
        
        // Convert to location timezone
        let localTime = periodTime;
        if (timeZoneId) {
            try {
                const formatter = new Intl.DateTimeFormat('en-US', {
                    timeZone: timeZoneId,
                    year: 'numeric',
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                    hour12: false
                });
                
                const parts = formatter.formatToParts(periodTime);
                const year = parseInt(parts.find(p => p.type === 'year').value);
                const month = parseInt(parts.find(p => p.type === 'month').value) - 1;
                const day = parseInt(parts.find(p => p.type === 'day').value);
                const hour = parseInt(parts.find(p => p.type === 'hour').value);
                const minute = parseInt(parts.find(p => p.type === 'minute').value);
                
                localTime = new Date(year, month, day, hour, minute);
            } catch (error) {
                // Fallback to UTC time
                localTime = periodTime;
            }
        }
        
        const dayKey = `${localTime.getFullYear()}-${String(localTime.getMonth() + 1).padStart(2, '0')}-${String(localTime.getDate()).padStart(2, '0')}`;
        const hour = localTime.getHours();
        
        if (!daysData[dayKey]) {
            daysData[dayKey] = {};
        }
        
        daysData[dayKey][hour] = period;
    });
    
    return daysData;
}

// Get day name from date
function getDayName(date, short = false) {
    const days = short 
        ? ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
        : ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    
    return days[date.getDay()];
}

// Convert wind direction degrees to cardinal direction
function getCardinalDirection(degrees) {
    if (degrees === null || degrees === undefined) {
        return '';
    }
    
    const directions = ['N', 'NNE', 'NE', 'ENE', 'E', 'ESE', 'SE', 'SSE', 'S', 'SSW', 'SW', 'WSW', 'W', 'WNW', 'NW', 'NNW'];
    const index = Math.round(degrees / 22.5) % 16;
    return directions[index];
}

// Process observations data from NWS API
// Returns { dailyByDate, timeZoneId, displayList } so the UI can always show "last 7 days from current date" at display time, not from cache time
function processObservationsData(observationsData, timeZoneId) {
    if (!observationsData || !observationsData.features || observationsData.features.length === 0) {
        return { dailyByDate: {}, timeZoneId: timeZoneId || null, displayList: [] };
    }
    
    const dailyData = {};
    
    // Process each observation
    observationsData.features.forEach(observation => {
        try {
            // Parse the observation timestamp (API returns in UTC)
            const obsTime = new Date(observation.properties.timestamp);
            
            // Convert to local timezone if provided
            let localTime = obsTime;
            if (timeZoneId) {
                try {
                    const formatter = new Intl.DateTimeFormat('en-US', {
                        timeZone: timeZoneId,
                        year: 'numeric',
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: false
                    });
                    
                    const parts = formatter.formatToParts(obsTime);
                    const year = parseInt(parts.find(p => p.type === 'year').value);
                    const month = parseInt(parts.find(p => p.type === 'month').value) - 1;
                    const day = parseInt(parts.find(p => p.type === 'day').value);
                    const hour = parseInt(parts.find(p => p.type === 'hour').value);
                    const minute = parseInt(parts.find(p => p.type === 'minute').value);
                    
                    localTime = new Date(year, month, day, hour, minute);
                } catch (error) {
                    // Fallback to UTC if timezone conversion fails
                    localTime = obsTime;
                }
            }
            
            const obsDate = `${localTime.getFullYear()}-${String(localTime.getMonth() + 1).padStart(2, '0')}-${String(localTime.getDate()).padStart(2, '0')}`;
            
            if (!dailyData[obsDate]) {
                dailyData[obsDate] = {
                    date: obsDate,
                    temperatures: [],
                    windSpeeds: [],
                    windGusts: [],
                    windDirections: [],
                    humidities: [],
                    precipitations: [],
                    conditions: [],
                    pressures: [],
                    cloudSummaryStrings: []
                };
            }
            
            const props = observation.properties;
            
            // Extract temperature (convert from Celsius to Fahrenheit if needed)
            if (props.temperature && props.temperature.value !== null && props.temperature.value !== undefined) {
                const tempC = props.temperature.value;
                const tempF = (tempC * 9/5) + 32;
                dailyData[obsDate].temperatures.push(tempF);
            }
            
            // Extract wind speed (convert from km/h to mph - NWS observations API provides all wind values in km/h)
            if (props.windSpeed && props.windSpeed.value !== null && props.windSpeed.value !== undefined) {
                // NWS observations API always provides wind speeds in km/h
                const windSpeedKmh = props.windSpeed.value;
                const windSpeedMph = windSpeedKmh * 0.621371;
                
                // Filter out unrealistic values (>200 mph is likely bad data)
                if (windSpeedMph <= 200) {
                    dailyData[obsDate].windSpeeds.push(windSpeedMph);
                } else {
                    console.warn('Filtered unrealistic wind speed:', windSpeedMph, 'mph (raw value:', windSpeedKmh, 'km/h)');
                }
            }
            
            // Extract wind gust (peak wind, convert from km/h to mph - NWS observations API provides all wind values in km/h)
            if (props.windGust && props.windGust.value !== null && props.windGust.value !== undefined) {
                // NWS observations API always provides wind gusts in km/h
                const windGustKmh = props.windGust.value;
                const windGustMph = windGustKmh * 0.621371;
                
                // Trust all gust values from the API - no filtering
                dailyData[obsDate].windGusts.push(windGustMph);
            }
            
            // Extract wind direction
            if (props.windDirection && props.windDirection.value !== null && props.windDirection.value !== undefined) {
                dailyData[obsDate].windDirections.push(props.windDirection.value);
            }
            
            // Extract sea-level pressure (NWS API returns Pascals; convert to inHg: inHg = Pa / 3386.389)
            if (props.seaLevelPressure && props.seaLevelPressure.value != null) {
                const pressureInHg = props.seaLevelPressure.value / 3386.389;
                dailyData[obsDate].pressures.push(pressureInHg);
            }
            
            // Extract humidity
            if (props.relativeHumidity && props.relativeHumidity.value !== null && props.relativeHumidity.value !== undefined) {
                dailyData[obsDate].humidities.push(props.relativeHumidity.value);
            }
            
            // Extract precipitation - try multiple fields for better accuracy
            // precipitationLastHour is most common, but also check other time periods
            // NWS API returns these values in millimeters, so we convert to inches
            let precipValue = null;
            if (props.precipitationLastHour && props.precipitationLastHour.value !== null && props.precipitationLastHour.value !== undefined) {
                // Convert from millimeters to inches (1 mm = 0.0393701 inches)
                precipValue = props.precipitationLastHour.value * 0.0393701;
            } else if (props.precipitationLast3Hours && props.precipitationLast3Hours.value !== null && props.precipitationLast3Hours.value !== undefined) {
                // Convert from millimeters to inches and divide by 3 to get hourly equivalent
                precipValue = (props.precipitationLast3Hours.value * 0.0393701) / 3;
            } else if (props.precipitationLast6Hours && props.precipitationLast6Hours.value !== null && props.precipitationLast6Hours.value !== undefined) {
                // Convert from millimeters to inches and divide by 6 to get hourly equivalent
                precipValue = (props.precipitationLast6Hours.value * 0.0393701) / 6;
            }
            
            if (precipValue !== null) {
                dailyData[obsDate].precipitations.push(precipValue);
            }
            
            // Extract conditions
            if (props.textDescription) {
                dailyData[obsDate].conditions.push(props.textDescription);
            }
            
            // Extract cloud layers summary (amount + base height in ft)
            if (props.cloudLayers && Array.isArray(props.cloudLayers) && props.cloudLayers.length > 0) {
                const parts = props.cloudLayers.map(layer => {
                    const amount = (layer.amount || '').trim() || '?';
                    const baseM = layer.base && layer.base.value != null ? layer.base.value : null;
                    const baseFt = baseM != null ? Math.round(baseM * 3.28084) : null;
                    const ftStr = baseFt != null ? baseFt.toLocaleString() + ' ft' : '? ft';
                    return amount + ' ' + ftStr;
                });
                const summary = parts.join(', ');
                if (summary) dailyData[obsDate].cloudSummaryStrings.push(summary);
            }
        } catch (error) {
            console.error('Error processing observation:', error);
            // Continue processing other observations
        }
    });
    
    // Helper: get date string (yyyy-MM-dd) in the location's timezone for a given instant
    const getDateStringInTz = (instant, tzId) => {
        if (!tzId) {
            const d = new Date(instant);
            return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
        }
        try {
            const formatter = new Intl.DateTimeFormat('en-US', {
                timeZone: tzId,
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour12: false
            });
            const parts = formatter.formatToParts(new Date(instant));
            const year = parseInt(parts.find(p => p.type === 'year').value);
            const month = parseInt(parts.find(p => p.type === 'month').value);
            const day = parseInt(parts.find(p => p.type === 'day').value);
            return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
        } catch (e) {
            const d = new Date(instant);
            return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
        }
    };

    // Build last 7 days from current time in location TZ (so display can rebuild from "today" at render time)
    const result = [];
    for (let i = 6; i >= 0; i--) {
        const instant = Date.now() - i * 24 * 60 * 60 * 1000;
        const date = getDateStringInTz(instant, timeZoneId);
        
        if (dailyData[date]) {
            const dayData = dailyData[date];
            
            // Calculate aggregates
            const highTemp = dayData.temperatures.length > 0 
                ? Math.round(Math.max(...dayData.temperatures) * 10) / 10 
                : null;
            const lowTemp = dayData.temperatures.length > 0 
                ? Math.round(Math.min(...dayData.temperatures) * 10) / 10 
                : null;
            const avgWindSpeed = dayData.windSpeeds.length > 0 
                ? Math.round(dayData.windSpeeds.reduce((a, b) => a + b, 0) / dayData.windSpeeds.length * 10) / 10 
                : null;
            const maxWindSpeed = dayData.windSpeeds.length > 0 
                ? Math.round(Math.max(...dayData.windSpeeds) * 10) / 10 
                : null;
            const maxWindGust = dayData.windGusts.length > 0 
                ? Math.round(Math.max(...dayData.windGusts) * 10) / 10 
                : null;
            // Use the larger of maxWindGust or maxWindSpeed for max wind value
            const maxWind = (maxWindGust !== null && maxWindSpeed !== null) 
                ? Math.max(maxWindGust, maxWindSpeed)
                : (maxWindGust !== null ? maxWindGust : maxWindSpeed);
            const windDirection = dayData.windDirections.length > 0 
                ? Math.round(dayData.windDirections.reduce((a, b) => a + b, 0) / dayData.windDirections.length) 
                : null;
            const avgHumidity = dayData.humidities.length > 0 
                ? Math.round(dayData.humidities.reduce((a, b) => a + b, 0) / dayData.humidities.length * 10) / 10 
                : null;
            const totalPrecipitation = dayData.precipitations.length > 0 
                ? Math.round(dayData.precipitations.reduce((a, b) => a + b, 0) * 100) / 100 
                : 0;
            
            // Get most common condition
            let conditions = 'N/A';
            if (dayData.conditions.length > 0) {
                const conditionCounts = {};
                dayData.conditions.forEach(cond => {
                    conditionCounts[cond] = (conditionCounts[cond] || 0) + 1;
                });
                const sortedConditions = Object.entries(conditionCounts).sort((a, b) => b[1] - a[1]);
                conditions = sortedConditions[0][0];
            }
            
            const pressure = dayData.pressures && dayData.pressures.length > 0
                ? Math.round(dayData.pressures.reduce((a, b) => a + b, 0) / dayData.pressures.length * 100) / 100
                : null;
            let cloudSummary = null;
            if (dayData.cloudSummaryStrings && dayData.cloudSummaryStrings.length > 0) {
                const nonEmpty = dayData.cloudSummaryStrings.filter(s => s && String(s).trim());
                if (nonEmpty.length > 0) {
                    const counts = {};
                    nonEmpty.forEach(s => { counts[s] = (counts[s] || 0) + 1; });
                    const sorted = Object.entries(counts).sort((a, b) => b[1] - a[1]);
                    cloudSummary = sorted[0][0];
                }
            }
            
            result.push({
                date: date,
                highTemp: highTemp,
                lowTemp: lowTemp,
                avgWindSpeed: avgWindSpeed,
                maxWindSpeed: maxWindSpeed,
                maxWindGust: maxWindGust,
                maxWind: maxWind, // The larger of maxWindGust or maxWindSpeed
                windDirection: windDirection,
                avgHumidity: avgHumidity,
                totalPrecipitation: totalPrecipitation,
                conditions: conditions,
                pressure: pressure,
                cloudSummary: cloudSummary
            });
        } else {
            // No data for this day - add empty entry
            result.push({
                date: date,
                highTemp: null,
                lowTemp: null,
                avgWindSpeed: null,
                maxWindSpeed: null,
                maxWindGust: null,
                maxWind: null,
                windDirection: null,
                avgHumidity: null,
                totalPrecipitation: 0,
                conditions: 'N/A',
                pressure: null,
                cloudSummary: null
            });
        }
    }
    
    // Filter out days with no actual data for displayList
    const filtered = result.filter(day => day.highTemp !== null || day.lowTemp !== null || day.avgWindSpeed !== null);
    return { dailyByDate: dailyData, timeZoneId: timeZoneId || null, displayList: filtered };
}

// Convert legacy cached observations (array of day objects) into { dailyByDate, timeZoneId } so the UI can build "last 7 days" from current date when restoring old cache
function migrateLegacyObservationsCache(observationsArray, timeZoneId) {
    if (!Array.isArray(observationsArray) || observationsArray.length === 0) return null;
    const dailyByDate = {};
    observationsArray.forEach(day => {
        if (!day || !day.date) return;
        const d = day;
        dailyByDate[d.date] = {
            temperatures: [d.highTemp, d.lowTemp].filter(x => x != null),
            windSpeeds: [d.avgWindSpeed, d.maxWindSpeed].filter(x => x != null),
            windGusts: d.maxWindGust != null ? [d.maxWindGust] : [],
            windDirections: d.windDirection != null ? [d.windDirection] : [],
            humidities: d.avgHumidity != null ? [d.avgHumidity] : [],
            precipitations: (d.totalPrecipitation != null && d.totalPrecipitation > 0) ? [d.totalPrecipitation] : [],
            conditions: (d.conditions && d.conditions !== 'N/A') ? [d.conditions] : [],
            pressures: d.pressure != null ? [d.pressure] : [],
            cloudSummaryStrings: d.cloudSummary != null && String(d.cloudSummary).trim() ? [d.cloudSummary] : []
        };
    });
    return { dailyByDate, timeZoneId: timeZoneId || null, displayList: observationsArray };
}

// Build the "last 7 days" display list from current time and cached daily aggregates (used at render time so History always shows last 7 days from today, not cache time)
function getObservationsDisplayList(observationsData) {
    if (!observationsData) return [];
    if (Array.isArray(observationsData)) return observationsData;
    const { dailyByDate, timeZoneId, displayList } = observationsData;
    if (!dailyByDate || typeof dailyByDate !== 'object') return displayList || [];
    const tzId = timeZoneId || null;
    const getDateStringInTz = (instant, tzId) => {
        if (!tzId) {
            const d = new Date(instant);
            return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
        }
        try {
            const formatter = new Intl.DateTimeFormat('en-US', { timeZone: tzId, year: 'numeric', month: '2-digit', day: '2-digit', hour12: false });
            const parts = formatter.formatToParts(new Date(instant));
            const year = parseInt(parts.find(p => p.type === 'year').value);
            const month = parseInt(parts.find(p => p.type === 'month').value);
            const day = parseInt(parts.find(p => p.type === 'day').value);
            return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
        } catch (e) {
            const d = new Date(instant);
            return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`;
        }
    };
    const result = [];
    for (let i = 6; i >= 0; i--) {
        const instant = Date.now() - i * 24 * 60 * 60 * 1000;
        const date = getDateStringInTz(instant, tzId);
        const dayData = dailyByDate[date];
        if (dayData) {
            const highTemp = dayData.temperatures.length > 0 ? Math.round(Math.max(...dayData.temperatures) * 10) / 10 : null;
            const lowTemp = dayData.temperatures.length > 0 ? Math.round(Math.min(...dayData.temperatures) * 10) / 10 : null;
            const avgWindSpeed = dayData.windSpeeds.length > 0 ? Math.round(dayData.windSpeeds.reduce((a, b) => a + b, 0) / dayData.windSpeeds.length * 10) / 10 : null;
            const maxWindSpeed = dayData.windSpeeds.length > 0 ? Math.round(Math.max(...dayData.windSpeeds) * 10) / 10 : null;
            const maxWindGust = dayData.windGusts.length > 0 ? Math.round(Math.max(...dayData.windGusts) * 10) / 10 : null;
            const maxWind = (maxWindGust != null && maxWindSpeed != null) ? Math.max(maxWindGust, maxWindSpeed) : (maxWindGust != null ? maxWindGust : maxWindSpeed);
            const windDirection = dayData.windDirections.length > 0 ? Math.round(dayData.windDirections.reduce((a, b) => a + b, 0) / dayData.windDirections.length) : null;
            const avgHumidity = dayData.humidities.length > 0 ? Math.round(dayData.humidities.reduce((a, b) => a + b, 0) / dayData.humidities.length * 10) / 10 : null;
            const totalPrecipitation = dayData.precipitations.length > 0 ? Math.round(dayData.precipitations.reduce((a, b) => a + b, 0) * 100) / 100 : 0;
            let conditions = 'N/A';
            if (dayData.conditions.length > 0) {
                const conditionCounts = {};
                dayData.conditions.forEach(cond => { conditionCounts[cond] = (conditionCounts[cond] || 0) + 1; });
                conditions = Object.entries(conditionCounts).sort((a, b) => b[1] - a[1])[0][0];
            }
            const pressure = dayData.pressures && dayData.pressures.length > 0
                ? Math.round(dayData.pressures.reduce((a, b) => a + b, 0) / dayData.pressures.length * 100) / 100
                : null;
            let cloudSummary = null;
            if (dayData.cloudSummaryStrings && dayData.cloudSummaryStrings.length > 0) {
                const nonEmpty = dayData.cloudSummaryStrings.filter(s => s && String(s).trim());
                if (nonEmpty.length > 0) {
                    const counts = {};
                    nonEmpty.forEach(s => { counts[s] = (counts[s] || 0) + 1; });
                    cloudSummary = Object.entries(counts).sort((a, b) => b[1] - a[1])[0][0];
                }
            }
            result.push({ date, highTemp, lowTemp, avgWindSpeed, maxWindSpeed, maxWindGust, maxWind, windDirection, avgHumidity, totalPrecipitation, conditions, pressure, cloudSummary });
        } else {
            result.push({ date, highTemp: null, lowTemp: null, avgWindSpeed: null, maxWindSpeed: null, maxWindGust: null, maxWind: null, windDirection: null, avgHumidity: null, totalPrecipitation: 0, conditions: 'N/A', pressure: null, cloudSummary: null });
        }
    }
    const filtered = result.filter(day => day.highTemp !== null || day.lowTemp !== null || day.avgWindSpeed !== null);
    if (filtered.length === 0 && displayList && displayList.length > 0) return displayList;
    return filtered;
}

