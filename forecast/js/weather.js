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
    const sunTimes = calculateSunriseSunset(
        location.lat,
        location.lon,
        new Date(),
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
            moonPhase: moonPhaseInfo
        }
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
function processObservationsData(observationsData, timeZoneId) {
    if (!observationsData || !observationsData.features || observationsData.features.length === 0) {
        return [];
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
                    conditions: []
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
                
                // Filter out unrealistic gust values (>100 mph gusts are extremely rare)
                // Also ensure gust is reasonable compared to wind speed
                const currentWindSpeed = dailyData[obsDate].windSpeeds.length > 0 
                    ? dailyData[obsDate].windSpeeds[dailyData[obsDate].windSpeeds.length - 1]
                    : null;
                if (windGustMph <= 150 && (currentWindSpeed === null || windGustMph <= currentWindSpeed * 3)) {
                    dailyData[obsDate].windGusts.push(windGustMph);
                } else {
                    console.warn('Filtered unrealistic wind gust:', windGustMph, 'mph (raw value:', windGustKmh, 'km/h, windSpeed:', currentWindSpeed, 'mph)');
                }
            }
            
            // Extract wind direction
            if (props.windDirection && props.windDirection.value !== null && props.windDirection.value !== undefined) {
                dailyData[obsDate].windDirections.push(props.windDirection.value);
            }
            
            // Extract humidity
            if (props.relativeHumidity && props.relativeHumidity.value !== null && props.relativeHumidity.value !== undefined) {
                dailyData[obsDate].humidities.push(props.relativeHumidity.value);
            }
            
            // Extract precipitation - try multiple fields for better accuracy
            let precipValue = null;
            if (props.precipitationLastHour && props.precipitationLastHour.value !== null && props.precipitationLastHour.value !== undefined) {
                precipValue = props.precipitationLastHour.value;
            } else if (props.precipitationLast3Hours && props.precipitationLast3Hours.value !== null && props.precipitationLast3Hours.value !== undefined) {
                // If 3-hour value exists, divide by 3 to get hourly equivalent
                precipValue = props.precipitationLast3Hours.value / 3;
            } else if (props.precipitationLast6Hours && props.precipitationLast6Hours.value !== null && props.precipitationLast6Hours.value !== undefined) {
                // If 6-hour value exists, divide by 6 to get hourly equivalent
                precipValue = props.precipitationLast6Hours.value / 6;
            }
            
            if (precipValue !== null) {
                dailyData[obsDate].precipitations.push(precipValue);
            }
            
            // Extract conditions
            if (props.textDescription) {
                dailyData[obsDate].conditions.push(props.textDescription);
            }
        } catch (error) {
            console.error('Error processing observation:', error);
            // Continue processing other observations
        }
    });
    
    // Calculate daily aggregates
    const result = [];
    
    // Get current date in the target timezone (same as observations)
    let nowInLocalTz = new Date();
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
            
            const parts = formatter.formatToParts(nowInLocalTz);
            const year = parseInt(parts.find(p => p.type === 'year').value);
            const month = parseInt(parts.find(p => p.type === 'month').value) - 1;
            const day = parseInt(parts.find(p => p.type === 'day').value);
            const hour = parseInt(parts.find(p => p.type === 'hour').value);
            const minute = parseInt(parts.find(p => p.type === 'minute').value);
            
            nowInLocalTz = new Date(year, month, day, hour, minute);
        } catch (error) {
            // Fallback to current time
            nowInLocalTz = new Date();
        }
    }
    
    // Process last 7 days
    for (let i = 6; i >= 0; i--) {
        const targetDate = new Date(nowInLocalTz);
        targetDate.setDate(targetDate.getDate() - i);
        const date = `${targetDate.getFullYear()}-${String(targetDate.getMonth() + 1).padStart(2, '0')}-${String(targetDate.getDate()).padStart(2, '0')}`;
        
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
                conditions: conditions
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
                conditions: 'N/A'
            });
        }
    }
    
    // Filter out days with no actual data
    return result.filter(day => day.highTemp !== null || day.lowTemp !== null || day.avgWindSpeed !== null);
}

