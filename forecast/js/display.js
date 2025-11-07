// Display mode rendering functions

// Display current conditions
function displayCurrentConditions(weather, location) {
    const { current, location: loc } = weather;
    const { sunrise, sunset, moonPhase } = loc;
    
    let html = '<div class="current-conditions">';
    html += `<div class="section-header">${location.city}, ${location.state} Current Conditions</div>`;
    
    html += '<div class="condition-row">';
    html += `<span class="condition-label">Currently:</span>`;
    html += `<span class="condition-value">${current.icon} ${current.conditions}</span>`;
    html += '</div>';
    
    html += '<div class="condition-row">';
    html += `<span class="condition-label">Temperature:</span>`;
    html += `<span class="condition-value ${getTempColor(current.temp)}">${current.temp}°F</span>`;
    
    if (current.windChill) {
        html += ` <span class="temp-cold">[${current.windChill}°F]</span>`;
    } else if (current.heatIndex) {
        html += ` <span class="temp-hot">[${current.heatIndex}°F]</span>`;
    }
    
    if (current.trend) {
        html += ` ${getTrendIcon(current.trend)}`;
    }
    html += '</div>';
    
    html += '<div class="condition-row">';
    html += `<span class="condition-label">Wind:</span>`;
    html += `<span class="condition-value ${getWindColor(getWindSpeed(current.wind))}">${current.wind} ${current.windDir}</span>`;
    if (current.windGust) {
        html += ` <span class="wind-strong">(gusts to ${current.windGust} mph)</span>`;
    }
    html += '</div>';
    
    html += '<div class="condition-row">';
    html += `<span class="condition-label">Humidity:</span>`;
    html += `<span class="condition-value ${getHumidityColor(current.humidity)}">${current.humidity}%</span>`;
    html += '</div>';
    
    if (current.dewPoint !== null) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Dew Point:</span>`;
        html += `<span class="condition-value ${getDewPointColor(current.dewPoint)}">${current.dewPoint}°F</span>`;
        html += '</div>';
    }
    
    if (current.precipProb > 0) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Precipitation:</span>`;
        html += `<span class="condition-value ${getPrecipColor(current.precipProb)}">${current.precipProb}% chance</span>`;
        html += '</div>';
    }
    
    if (sunrise) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Sunrise:</span>`;
        html += `<span class="condition-value">${formatTime(sunrise, location.timeZone)}</span>`;
        html += '</div>';
    }
    
    if (sunset) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Sunset:</span>`;
        html += `<span class="condition-value">${formatTime(sunset, location.timeZone)}</span>`;
        html += '</div>';
    }
    
    if (moonPhase && moonPhase.emoji) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Moon Phase:</span>`;
        html += `<span class="condition-value">${moonPhase.emoji} ${moonPhase.name}</span>`;
        html += '</div>';
        
        if (moonPhase.showNextFullMoon && moonPhase.nextFullMoon) {
            html += '<div class="condition-row">';
            html += `<span class="condition-label">Next Full Moon:</span>`;
            html += `<span class="condition-value">${moonPhase.nextFullMoon}</span>`;
            html += '</div>';
        }
        
        if (moonPhase.showNextNewMoon && moonPhase.nextNewMoon) {
            html += '<div class="condition-row">';
            html += `<span class="condition-label">Next New Moon:</span>`;
            html += `<span class="condition-value">${moonPhase.nextNewMoon}</span>`;
            html += '</div>';
        }
    }
    
    html += '<div class="condition-row">';
    html += `<span class="condition-label">Updated:</span>`;
    html += `<span class="condition-value">${formatDateTime(current.time, location.timeZone)}</span>`;
    html += '</div>';
    
    html += '</div>';
    
    return html;
}

// Display forecast text
function displayForecastText(title, text) {
    const wrappedLines = wrapText(text, 80);
    let html = `<div class="section-header">${title}</div>`;
    html += '<div class="forecast-text">';
    html += wrappedLines.join('\n');
    html += '</div>';
    return html;
}

// Display hourly forecast
function displayHourlyForecast(weather, location, startIndex = 0, maxHours = 12, showNavigation = true) {
    const { hourly } = weather;
    const periods = hourly.periods;
    const totalHours = Math.min(periods.length, 48);
    const endIndex = Math.min(startIndex + maxHours, totalHours);
    
    const cityName = truncateCityName(location.city, 20);
    let html = `<div class="section-header">${cityName} Hourly</div>`;
    
    // Only show navigation buttons if showNavigation is true (for hourly mode)
    if (showNavigation) {
        if (startIndex > 0) {
            html += '<button class="hourly-nav-btn" data-action="scroll-up" style="color: yellow; margin-bottom: 0.5rem; background: none; border: none; cursor: pointer; text-decoration: underline; padding: 0;">↑ Previous hours available</button><br>';
        }
        if (endIndex < totalHours) {
            html += '<button class="hourly-nav-btn" data-action="scroll-down" style="color: yellow; margin-bottom: 0.5rem; background: none; border: none; cursor: pointer; text-decoration: underline; padding: 0;">↓ More hours available</button>';
        }
        html += `<div style="color: cyan; margin-bottom: 1rem;">Showing hours ${startIndex + 1}-${endIndex} of ${totalHours}</div>`;
    }
    
    html += '<div class="hourly-forecast">';
    html += '<table class="hourly-table">';
    html += '<thead><tr><th>Time</th><th>Temp</th><th>Wind</th><th>Precip</th><th>Forecast</th></tr></thead>';
    html += '<tbody>';
    
    for (let i = startIndex; i < endIndex; i++) {
        const period = periods[i];
        const periodTime = new Date(period.startTime);
        const hourDisplay = formatTime(periodTime, location.timeZone);
        const temp = period.temperature;
        const wind = period.windSpeed;
        const windDir = period.windDirection;
        const precipProb = period.probabilityOfPrecipitation?.value || 0;
        const shortForecast = period.shortForecast;
        
        // Determine if daytime
        const isPeriodDaytime = period.isDaytime !== undefined 
            ? period.isDaytime 
            : (periodTime.getHours() >= 6 && periodTime.getHours() < 18);
        
        const periodIcon = getWeatherIcon(period.icon, isPeriodDaytime, precipProb);
        
        // Calculate windchill or heat index
        const tempNum = parseFloat(temp);
        const windSpeedNum = getWindSpeed(wind);
        let windchillHeatIndex = "";
        let windchillHeatIndexClass = "";
        
        if (tempNum <= 50) {
            const windChill = calculateWindChill(tempNum, windSpeedNum);
            if (windChill && Math.abs(tempNum - windChill) > 1) {
                windchillHeatIndex = ` [${windChill}°F]`;
                windchillHeatIndexClass = "temp-cold";
            }
        } else if (tempNum >= 80) {
            const humidityNum = period.relativeHumidity?.value || 0;
            const heatIndex = calculateHeatIndex(tempNum, humidityNum);
            if (heatIndex && Math.abs(heatIndex - tempNum) > 1) {
                windchillHeatIndex = ` [${heatIndex}°F]`;
                windchillHeatIndexClass = "temp-hot";
            }
        }
        
        html += '<tr>';
        html += `<td>${hourDisplay}</td>`;
        html += `<td>${periodIcon} <span class="${getTempColor(temp)}">${temp}°F</span>${windchillHeatIndex ? `<span class="${windchillHeatIndexClass}">${windchillHeatIndex}</span>` : ''}</td>`;
        html += `<td class="${getWindColor(windSpeedNum)}">${wind} ${windDir}</td>`;
        html += `<td class="${getPrecipColor(precipProb)}">${precipProb > 0 ? `${precipProb}%` : ''}</td>`;
        html += `<td>${shortForecast}</td>`;
        html += '</tr>';
    }
    
    html += '</tbody></table></div>';
    
    return html;
}

// Display 7-day forecast
function displaySevenDayForecast(weather, location, enhanced = false) {
    const { forecast } = weather;
    const periods = forecast.periods;
    const cityName = truncateCityName(location.city, 20);
    
    let html = `<div class="section-header">${cityName} ${enhanced ? '7-Day Forecast' : '7-Day Summary'}</div>`;
    
    const processedDays = {};
    let dayCount = 0;
    const maxDays = 7;
    
    for (const period of periods) {
        if (dayCount >= maxDays) break;
        
        const periodTime = new Date(period.startTime);
        const dayName = enhanced ? getDayName(periodTime, false) : getDayName(periodTime, true);
        
        // Skip if we've already processed this day
        if (processedDays[dayName]) continue;
        
        const temp = period.temperature;
        const shortForecast = period.shortForecast;
        const precipProb = period.probabilityOfPrecipitation?.value || 0;
        const periodIcon = getWeatherIcon(period.icon, true, precipProb);
        
        // Find night period for same day
        const currentDay = formatDate(periodTime);
        let nightTemp = null;
        let nightDetailedForecast = null;
        
        for (const nightPeriod of periods) {
            const nightTime = new Date(nightPeriod.startTime);
            const nightDay = formatDate(nightTime);
            
            if (nightDay === currentDay && nightTime > periodTime) {
                nightTemp = nightPeriod.temperature;
                nightDetailedForecast = nightPeriod.detailedForecast;
                break;
            }
        }
        
        html += '<div class="daily-item">';
        
        if (enhanced) {
            // Enhanced mode
            const windSpeed = getWindSpeed(period.windSpeed);
            const windColor = windSpeed >= 16 ? "wind-strong" : "";
            const windDisplay = period.windSpeed.replace(/\s+mph/, 'mph');
            
            // Calculate windchill or heat index
            const tempNum = parseFloat(temp);
            let windChillHeatIndex = "";
            let windChillHeatIndexClass = "";
            
            if (tempNum <= 50) {
                const windChill = calculateWindChill(tempNum, windSpeed);
                if (windChill && Math.abs(tempNum - windChill) > 1) {
                    windChillHeatIndex = ` <span class="temp-cold">[${windChill}°F]</span>`;
                }
            } else if (tempNum >= 80) {
                const humidityNum = period.relativeHumidity?.value || 0;
                const heatIndex = calculateHeatIndex(tempNum, humidityNum);
                if (heatIndex && Math.abs(heatIndex - tempNum) > 1) {
                    windChillHeatIndex = ` <span class="temp-hot">[${heatIndex}°F]</span>`;
                }
            }
            
            // Temperature row
            html += `<div class="daily-temp-row">${dayName}: <span class="${getTempColor(temp)}">H:${temp}°F</span>${windChillHeatIndex || ' '}`;
            if (nightTemp) {
                html += `<span class="${getTempColor(nightTemp)}">L:${nightTemp}°F</span>`;
            }
            html += '</div>';
            
            // Additional info row
            html += `<div class="daily-info-row"><span class="${windColor}">${windDisplay} ${period.windDirection}</span>`;
            if (precipProb > 0) {
                html += ` <span class="${getPrecipColor(precipProb)}">${precipProb}%☔️</span>`;
            }
            html += '</div>';
            
            // Day and night detailed forecasts
            if (nightDetailedForecast) {
                const dayForecastText = period.detailedForecast || "No detailed forecast available";
                
                html += `<div class="daily-details">${periodIcon} Day: ${dayForecastText}</div>`;
                
                const moonPhaseInfo = calculateMoonPhase(new Date());
                
                html += `<div class="daily-details">${moonPhaseInfo.emoji} Night: ${nightDetailedForecast}</div>`;
            } else {
                const currentHour = periodTime.getHours();
                const isCurrentPeriodNight = (currentHour >= 18 || currentHour < 6);
                const singlePeriodLabel = isCurrentPeriodNight ? `${calculateMoonPhase(new Date()).emoji} Night: ` : `${periodIcon} Day: `;
                const singlePeriodText = period.detailedForecast || "No detailed forecast available";
                
                html += `<div class="daily-details">${singlePeriodLabel}${singlePeriodText}</div>`;
            }
        } else {
            // Standard mode
            // Temperature row
            html += `<div class="daily-temp-row">${dayName}: ${periodIcon} <span class="${getTempColor(temp)}">H:${temp}°F</span>`;
            if (nightTemp) {
                html += ` <span class="${getTempColor(nightTemp)}">L:${nightTemp}°F</span>`;
            }
            html += '</div>';
            
            // Additional info row
            html += `<div class="daily-info-row">${shortForecast}`;
            if (precipProb > 0) {
                html += ` <span class="${getPrecipColor(precipProb)}">${precipProb}%☔️</span>`;
            }
            html += '</div>';
        }
        
        html += '</div>';
        
        processedDays[dayName] = true;
        dayCount++;
    }
    
    return html;
}

// Display rain forecast
function displayRainForecast(weather, location) {
    const { hourly } = weather;
    const periods = hourly.periods;
    const totalHours = Math.min(periods.length, 96);
    const cityName = truncateCityName(location.city, 20);
    
    let html = `<div class="section-header">${cityName} Rain Outlook</div>`;
    
    // Group periods by day
    const daysData = groupHourlyByDay(periods.slice(0, totalHours), location.timeZone);
    const sortedDays = Object.keys(daysData).sort();
    let dayCount = 0;
    
    for (const dayKey of sortedDays) {
        if (dayCount >= 5) break;
        
        const [year, month, day] = dayKey.split('-').map(Number);
        const periodTime = new Date(year, month - 1, day);
        const dayName = getDayName(periodTime, true);
        const dayData = daysData[dayKey];
        
        // Find max rain percentage for this day
        let maxRainPercent = 0;
        for (const hour in dayData) {
            const period = dayData[hour];
            const rainPercent = period.probabilityOfPrecipitation?.value || 0;
            if (rainPercent > maxRainPercent) {
                maxRainPercent = rainPercent;
            }
        }
        
        // Cap at 99% to prevent alignment issues
        if (maxRainPercent === 100) maxRainPercent = 99;
        
        // Get color for max percentage
        const maxRainColor = maxRainPercent <= 10 ? "precip-low" :
                            maxRainPercent <= 33 ? "humidity-low" :
                            maxRainPercent <= 44 ? "humidity-normal" :
                            maxRainPercent <= 80 ? "precip-medium" : "precip-high";
        
        html += '<div class="rain-day">';
        html += `${dayName} <span class="rain-percent ${maxRainColor}">${maxRainPercent < 10 ? ' ' : ''}${maxRainPercent}%</span> `;
        html += '<div class="rain-grid">';
        
        // Build CSS Grid with colored blocks for 24 hours
        for (let hour = 0; hour < 24; hour++) {
            if (dayData[hour]) {
                const period = dayData[hour];
                const rainPercent = period.probabilityOfPrecipitation?.value || 0;
                
                // Create block with background color based on percentage
                if (rainPercent === 0) {
                    // Empty space for 0%
                    html += '<div class="rain-block rain-block-empty"></div>';
                } else {
                    // Colored block with gradient - top 40% background, bottom percentage% filled
                    const fillPercent = 100 - rainPercent; // Top portion that's transparent
                    const blockColor = getRainBlockColor(rainPercent);
                    html += `<div class="rain-block" style="background: linear-gradient(to bottom, transparent ${fillPercent}%, ${blockColor} ${fillPercent}%);"></div>`;
                }
            } else {
                // Empty space for missing data
                html += '<div class="rain-block rain-block-empty"></div>';
            }
        }
        
        html += '</div></div>';
        dayCount++;
    }
    
    return html;
}

// Display wind forecast
function displayWindForecast(weather, location) {
    const { hourly } = weather;
    const periods = hourly.periods;
    const totalHours = Math.min(periods.length, 96);
    const cityName = truncateCityName(location.city, 20);
    
    let html = `<div class="section-header">${cityName} Wind Outlook</div>`;
    
    // Group periods by day
    const daysData = groupHourlyByDay(periods.slice(0, totalHours), location.timeZone);
    const sortedDays = Object.keys(daysData).sort();
    let dayCount = 0;
    
    for (const dayKey of sortedDays) {
        if (dayCount >= 5) break;
        
        const [year, month, day] = dayKey.split('-').map(Number);
        const periodTime = new Date(year, month - 1, day);
        const dayName = getDayName(periodTime, true);
        const dayData = daysData[dayKey];
        
        // Find max wind speed for this day
        let maxWindSpeed = 0;
        for (const hour in dayData) {
            const period = dayData[hour];
            const windSpeed = getWindSpeed(period.windSpeed);
            if (windSpeed > maxWindSpeed) {
                maxWindSpeed = windSpeed;
            }
        }
        
        // Get color for max wind speed
        const maxWindColor = maxWindSpeed <= 5 ? "wind-calm" :
                            maxWindSpeed <= 9 ? "wind-light" :
                            maxWindSpeed <= 14 ? "wind-moderate" : "wind-strong";
        
        html += '<div class="wind-day">';
        html += `${dayName} <span class="wind-speed ${maxWindColor}">${maxWindSpeed < 10 ? ' ' : ''}${maxWindSpeed}mph</span> `;
        html += '<span class="wind-grid">';
        
        // Build wind glyphs for 24 hours (on same line)
        for (let hour = 0; hour < 24; hour++) {
            if (dayData[hour]) {
                const period = dayData[hour];
                const windSpeed = getWindSpeed(period.windSpeed);
                const windDirection = period.windDirection;
                const windGlyphData = getWindGlyph(windDirection, windSpeed);
                
                // Check if this hour matches peak wind speed
                if (windSpeed === maxWindSpeed) {
                    html += `<span class="wind-char wind-glyph-peak" style="background-color: ${windGlyphData.color}; color: black;">${windGlyphData.char}</span>`;
                } else {
                    html += `<span class="wind-char" style="color: ${windGlyphData.color};">${windGlyphData.char}</span>`;
                }
            } else {
                html += '<span class="wind-char"> </span>';
            }
        }
        
        html += '</span></div>';
        dayCount++;
    }
    
    return html;
}

// Display weather alerts
function displayWeatherAlerts(alerts, showDetails = true) {
    if (!alerts || alerts.length === 0) {
        return '';
    }
    
    let html = '<div class="section-header">Active Weather Alerts</div>';
    
    alerts.forEach((alert, index) => {
        const props = alert.properties;
        const event = props.event;
        const headline = props.headline;
        const description = props.description;
        const effective = new Date(props.effective);
        const expires = new Date(props.expires);
        
        html += '<div class="alert-item">';
        html += `<div class="alert-title">${event}</div>`;
        html += `<div class="alert-description">${headline}</div>`;
        
        if (showDetails && description) {
            const wrappedDescription = wrapText(description, 80);
            html += '<div class="alert-description">';
            html += wrappedDescription.join('\n');
            html += '</div>';
        }
        
        html += '<div class="alert-times">';
        html += `Effective: ${effective.toLocaleString()}<br>`;
        html += `Expires: ${expires.toLocaleString()}`;
        html += '</div>';
        html += '</div>';
    });
    
    return html;
}

// Display location information
function displayLocationInfo(location) {
    let html = '<div class="location-info">';
    html += '<div class="section-header">Location Information</div>';
    html += `<div class="location-info-item">Time Zone: ${location.timeZone}</div>`;
    html += `<div class="location-info-item">Coordinates: ${location.lat}, ${location.lon}</div>`;
    html += `<div class="location-info-item">Elevation: ${location.elevationFeet}ft</div>`;
    
    html += '<div class="location-info-item">NWS Resources: ';
    const forecastUrl = `https://forecast.weather.gov/MapClick.php?lat=${location.lat}&lon=${location.lon}`;
    const graphUrl = `https://forecast.weather.gov/MapClick.php?lat=${location.lat}&lon=${location.lon}&unit=0&lg=english&FcstType=graphical`;
    const radarUrl = `https://radar.weather.gov/ridge/standard/${location.radarStation}_loop.gif`;
    
    html += `<a href="${forecastUrl}" target="_blank" class="location-info-link">Forecast</a>`;
    html += `<a href="${graphUrl}" target="_blank" class="location-info-link">Graph</a>`;
    html += `<a href="${radarUrl}" target="_blank" class="location-info-link">Radar</a>`;
    html += '</div>';
    
    html += '</div>';
    
    return html;
}

// Display full weather report
function displayFullWeatherReport(weather, location) {
    let html = '';
    html += displayCurrentConditions(weather, location);
    html += displayForecastText(weather.forecast.today.name, weather.forecast.today.text);
    if (weather.forecast.tomorrow.text) {
        html += displayForecastText(weather.forecast.tomorrow.name, weather.forecast.tomorrow.text);
    }
    html += displayHourlyForecast(weather, location, 0, 12, false);
    html += displaySevenDayForecast(weather, location, false);
    html += displayWeatherAlerts(weather.alerts, true);
    html += displayLocationInfo(location);
    return html;
}

// Display terse mode
function displayTerseMode(weather, location) {
    let html = '';
    html += displayCurrentConditions(weather, location);
    html += displayForecastText(weather.forecast.today.name, weather.forecast.today.text);
    html += displayWeatherAlerts(weather.alerts, false);
    return html;
}

