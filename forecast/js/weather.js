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

