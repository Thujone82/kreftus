// Display mode rendering functions

// Get time ago string (helper function, also defined in app.js)
function getTimeAgo(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (seconds < 60) {
        return 'just now';
    } else if (minutes < 60) {
        return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
    } else if (hours < 24) {
        return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
    } else {
        return date.toLocaleString();
    }
}

// Display current conditions
function displayCurrentConditions(weather, location) {
    const { current, location: loc } = weather;
    const { sunrise, sunset, moonPhase } = loc;
    
    let html = '<div class="current-conditions">';
    const locationDisplayName = formatLocationDisplayName(location.city, location.state);
    html += `<div class="section-header">${locationDisplayName} Current Conditions</div>`;
    
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
        // Format as date/time (MM/dd HH:mm) if polar night/day, otherwise time (12-hour format)
        // During polar night: shows next sunrise; during polar day: shows last sunrise
        const sunriseDisplay = (loc.isPolarNight || loc.isPolarDay) ? formatSunriseDate(sunrise, location.timeZone) : formatTime(sunrise, location.timeZone);
        html += `<span class="condition-value">${sunriseDisplay}</span>`;
        html += '</div>';
    }
    
    if (sunset) {
        html += '<div class="condition-row">';
        html += `<span class="condition-label">Sunset:</span>`;
        // Format as date/time (MM/dd HH:mm) if polar night/day, otherwise time (12-hour format)
        // During polar night: shows last sunset; during polar day: shows next sunset
        const sunsetDisplay = (loc.isPolarNight || loc.isPolarDay) ? formatSunriseDate(sunset, location.timeZone) : formatTime(sunset, location.timeZone);
        html += `<span class="condition-value">${sunsetDisplay}</span>`;
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
    
    // Get time ago and check if stale
    // Access appState from the global scope (it's defined in app.js)
    // Use appState.lastFetchTime (when data was actually fetched) instead of current.time
    // Format in viewer's local timezone, not destination timezone
    let timeAgoHtml = '';
    let updatedTimeHtml = '';
    
    if (typeof appState !== 'undefined' && appState.lastFetchTime) {
        const fetchTime = appState.lastFetchTime;
        const timeAgo = getTimeAgo(fetchTime);
        
        // Check if data is stale (>10 minutes)
        const now = new Date();
        const diff = now - fetchTime;
        const isStale = diff > 600000; // 10 minutes in milliseconds
        const staleClass = isStale ? 'stale-data' : '';
        timeAgoHtml = ` <span class="updated-timestamp ${staleClass}">[${timeAgo}]</span>`;
        
        // Format in viewer's local timezone (don't pass timeZoneId, or pass undefined/null)
        // This will use the browser's local timezone
        updatedTimeHtml = formatDateTime(fetchTime, null);
    } else if (current.time) {
        // Fallback to current.time if lastFetchTime not available (shouldn't happen, but safety)
        // Still format in viewer's local timezone
        updatedTimeHtml = formatDateTime(current.time, null);
    }
    
    html += `<span class="condition-value">${updatedTimeHtml}${timeAgoHtml}</span>`;
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
    
    // Get sunrise/sunset from location for hour label color coding
    // Ensure they are Date objects (they may be Date objects, ISO strings, or null/undefined)
    let sunrise = null;
    let sunset = null;
    if (location.sunrise) {
        sunrise = location.sunrise instanceof Date ? location.sunrise : new Date(location.sunrise);
        // Validate the date is not invalid
        if (isNaN(sunrise.getTime())) {
            sunrise = null;
        }
    }
    if (location.sunset) {
        sunset = location.sunset instanceof Date ? location.sunset : new Date(location.sunset);
        // Validate the date is not invalid
        if (isNaN(sunset.getTime())) {
            sunset = null;
        }
    }
    
    // If sunrise/sunset are not available, calculate them now
    // This ensures hour label coloring always works, even if location data is incomplete
    if ((!sunrise || !sunset) && location.lat && location.lon && location.timeZone) {
        // Use the location's current date (not the viewer's local date) for accuracy
        const locationToday = convertToTimeZone(new Date(), location.timeZone);
        const locationDate = new Date(locationToday.getFullYear(), locationToday.getMonth(), locationToday.getDate());
        const sunTimes = calculateSunriseSunset(
            location.lat,
            location.lon,
            locationDate,
            location.timeZone
        );
        if (sunTimes.sunrise && !sunrise) {
            sunrise = sunTimes.sunrise instanceof Date ? sunTimes.sunrise : new Date(sunTimes.sunrise);
        }
        if (sunTimes.sunset && !sunset) {
            sunset = sunTimes.sunset instanceof Date ? sunTimes.sunset : new Date(sunTimes.sunset);
        }
    }
    
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
        
        // Get hour label color class (yellow for daytime, white for nighttime)
        const hourLabelColorClass = getHourLabelColor(periodTime, sunrise, sunset, location.timeZone);
        
        html += '<tr>';
        html += `<td class="${hourLabelColorClass}">${hourDisplay}</td>`;
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
        
        // Format date as MM/DD for day label
        const month = String(periodTime.getMonth() + 1).padStart(2, '0');
        const day = String(periodTime.getDate()).padStart(2, '0');
        const dateStr = `${month}/${day}`;
        const dayNameWithDate = `${dayName} (${dateStr})`;
        
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
        let nightPeriodTime = null;
        
        for (const nightPeriod of periods) {
            const nightTime = new Date(nightPeriod.startTime);
            const nightDay = formatDate(nightTime);
            
            if (nightDay === currentDay && nightTime > periodTime) {
                nightTemp = nightPeriod.temperature;
                nightDetailedForecast = nightPeriod.detailedForecast;
                nightPeriodTime = nightTime;
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
            
            // Calculate sunrise/sunset for this specific day (use date only, not time)
            let sunriseStr = "";
            let sunsetStr = "";
            let dayLengthStr = "";
            if (location.lat && location.lon && location.timeZone) {
            // Use the location's date (not viewer local) for accurate sunrise/sunset calculation
            const periodLocal = location.timeZone ? convertToTimeZone(periodTime, location.timeZone) : periodTime;
            const dayDate = new Date(periodLocal.getFullYear(), periodLocal.getMonth(), periodLocal.getDate());
                const daySunTimes = calculateSunriseSunset(
                    location.lat,
                    location.lon,
                    dayDate,
                    location.timeZone
                );
                if (daySunTimes.sunrise) {
                    // Format sunrise: date/time (MM/dd HH:mm) if polar night/day, otherwise time (24-hour format)
                    // During polar night: shows next sunrise; during polar day: shows last sunrise
                    sunriseStr = (daySunTimes.isPolarNight || daySunTimes.isPolarDay) ? formatSunriseDate(daySunTimes.sunrise, location.timeZone) : formatTime24(daySunTimes.sunrise, location.timeZone);
                    // Show sunset if available (during polar night: last sunset; during polar day: next sunset)
                    if (daySunTimes.sunset) {
                        sunsetStr = (daySunTimes.isPolarNight || daySunTimes.isPolarDay) ? formatSunriseDate(daySunTimes.sunset, location.timeZone) : formatTime24(daySunTimes.sunset, location.timeZone);
                        // Only show day length if not polar night/day (normal day)
                        if (!daySunTimes.isPolarNight && !daySunTimes.isPolarDay) {
                            dayLengthStr = formatDayLength(daySunTimes.sunrise, daySunTimes.sunset);
                        }
                    }
                }
            }
            
            // Display day label on its own line (slightly larger)
            // Then sunrise/sunset/day length on the next line
            if (sunriseStr) {
                html += `<div class="daily-day-label">${dayNameWithDate}:</div>`;
                if (sunsetStr) {
                    if (dayLengthStr) {
                        // Normal case: show sunrise, sunset, and day length
                        html += `<div class="condition-row">Sunrise: <span class="forecast-text">${sunriseStr}</span> Sunset: <span class="forecast-text">${sunsetStr}</span> Day Length: <span class="forecast-text">${dayLengthStr}</span></div>`;
                    } else {
                        // Polar night/day: show sunrise and sunset (with date/time), no day length
                        html += `<div class="condition-row">Sunrise: <span class="forecast-text">${sunriseStr}</span> Sunset: <span class="forecast-text">${sunsetStr}</span></div>`;
                    }
                } else {
                    // Only sunrise available
                    html += `<div class="condition-row">Sunrise: <span class="forecast-text">${sunriseStr}</span></div>`;
                }
                html += '<div></div>'; // Line feed after day length (or sunrise/sunset)
            }
            
            // Temperature and info row (combined) - day name only if no sunrise/sunset
            if (sunriseStr && sunsetStr && dayLengthStr) {
                // No day name here, it's on the sunrise line
                html += `<div class="daily-temp-row"> <span class="${getTempColor(temp)}">H:${temp}°F</span>${windChillHeatIndex || ' '}`;
            } else {
                // If no sunrise/sunset, show day name on temperature line
                html += `<div class="daily-temp-row">${dayNameWithDate}:<span class="${getTempColor(temp)}">H:${temp}°F</span>${windChillHeatIndex || ' '}`;
                html += '<div></div>'; // Line feed after day label for narrow mode
            }
            if (nightTemp) {
                html += `<span class="${getTempColor(nightTemp)}">L:${nightTemp}°F</span>`;
            }
            html += ` <span class="${windColor}">${windDisplay} ${period.windDirection}</span>`;
            if (precipProb > 0) {
                html += ` <span class="${getPrecipColor(precipProb)}">${precipProb}%☔️</span>`;
            }
            html += '</div>';
            
            // Day and night detailed forecasts
            if (nightDetailedForecast) {
                const dayForecastText = period.detailedForecast || "No detailed forecast available";
                
                html += `<div class="daily-details">${periodIcon} Day: ${dayForecastText}</div>`;
                
                // Calculate moon phase for the night period's date
                // Use the night period's date if available, otherwise use the day period's date
                const nightDate = nightPeriodTime || periodTime;
                const moonPhaseInfo = calculateMoonPhase(nightDate);
                
                html += `<div class="daily-details">${moonPhaseInfo.emoji} Night: ${nightDetailedForecast}</div>`;
            } else {
                const currentHour = periodTime.getHours();
                const isCurrentPeriodNight = (currentHour >= 18 || currentHour < 6);
                // Calculate moon phase for this period's date
                const moonPhaseInfo = calculateMoonPhase(periodTime);
                const singlePeriodLabel = isCurrentPeriodNight ? `${moonPhaseInfo.emoji} Night: ` : `${periodIcon} Day: `;
                const singlePeriodText = period.detailedForecast || "No detailed forecast available";
                
                html += `<div class="daily-details">${singlePeriodLabel}${singlePeriodText}</div>`;
            }
        } else {
            // Standard mode
            // Temperature and info row (combined)
            html += `<div class="daily-temp-row">${dayName}: ${periodIcon} <span class="${getTempColor(temp)}">H:${temp}°F</span>`;
            if (nightTemp) {
                html += ` <span class="${getTempColor(nightTemp)}">L:${nightTemp}°F</span>`;
            }
            html += ` ${shortForecast}`;
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
    
    let html = `<div class="section-header">${cityName} <span class="section-header-forecast">Forecast</span> Rain Outlook</div>`;
    
    // Add hour header row
    html += '<div class="rain-day">';
    html += '<div class="rain-day-header">Hour:</div>';
    html += '<div class="rain-grid">';
    for (let hour = 0; hour < 24; hour++) {
        // Skip even double-digit numbers (10, 12, 14, 16, 18, 20, 22) to reduce clutter
        if (hour >= 10 && hour % 2 === 0) {
            html += '<div class="rain-block rain-hour-header"></div>';
        } else {
            html += `<div class="rain-block rain-hour-header">${hour}</div>`;
        }
    }
    html += '</div></div>';
    
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
        html += `<div class="rain-day-header">${dayName} <span class="rain-percent ${maxRainColor}">${maxRainPercent < 10 ? ' ' : ''}${maxRainPercent}%</span></div>`;
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
    
    let html = `<div class="section-header">${cityName} <span class="section-header-forecast">Forecast</span> Wind Outlook</div>`;
    
    // Add hour header row
    html += '<div class="wind-day">';
    html += '<div class="wind-day-header">Hour:</div>';
    html += '<div class="wind-hour-grid">';
    for (let hour = 0; hour < 24; hour++) {
        // Skip even double-digit numbers (10, 12, 14, 16, 18, 20, 22) to reduce clutter
        if (hour >= 10 && hour % 2 === 0) {
            html += '<div class="wind-hour-cell"></div>';
        } else {
            html += `<div class="wind-hour-cell">${hour}</div>`;
        }
    }
    html += '</div></div>';
    
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
        html += `<div class="wind-day-header">${dayName} <span class="wind-speed ${maxWindColor}">${maxWindSpeed < 10 ? ' ' : ''}${maxWindSpeed}mph</span></div>`;
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
function displayWeatherAlerts(alerts, showDetails = true, timeZoneId = null) {
    if (!alerts || alerts.length === 0) {
        return '';
    }
    
    let html = '<div class="section-header">Active Weather Alerts</div>';
    
    alerts.forEach((alert, index) => {
        const props = alert.properties;
        const event = props.event;
        const headline = props.headline;
        const description = props.description;
        
        // Parse dates - NWS API returns ISO 8601 format, typically UTC (with 'Z' suffix)
        // We need to ensure they're treated as UTC and then converted to location's timezone
        // Note: 'expires' is when the alert message expires, 'ends' is when the event actually ends
        // We use 'ends' if available to match the description text, otherwise fall back to 'expires'
        const effective = new Date(props.effective);
        const eventEnd = props.ends ? new Date(props.ends) : new Date(props.expires);
        
        // Format times in location's timezone to match the alert description text
        // The description text already contains times in local time, so we must match that
        const targetTimeZone = timeZoneId || Intl.DateTimeFormat().resolvedOptions().timeZone;
        const effectiveFormatted = formatDateTime24(effective, targetTimeZone);
        const expiresFormatted = formatDateTime24(eventEnd, targetTimeZone);
        
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
        html += `Effective: ${effectiveFormatted}<br>`;
        html += `Expires: ${expiresFormatted}`;
        html += '</div>';
        html += '</div>';
    });
    
    return html;
}

// Get UTC offset string for a timezone (e.g., "UTC-8" or "UTC+5")
function getUtcOffsetString(timeZoneId) {
    if (!timeZoneId) {
        return '';
    }
    
    try {
        const now = new Date();
        
        // Get UTC time components
        const utcHour = now.getUTCHours();
        const utcMinute = now.getUTCMinutes();
        const utcTime = utcHour * 60 + utcMinute; // minutes since midnight UTC
        
        // Get timezone time components for the same moment
        const tzFormatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        });
        
        const tzParts = tzFormatter.formatToParts(now);
        const tzHour = parseInt(tzParts.find(p => p.type === 'hour').value, 10);
        const tzMinute = parseInt(tzParts.find(p => p.type === 'minute').value, 10);
        const tzTime = tzHour * 60 + tzMinute; // minutes since midnight in timezone
        
        // Calculate offset in minutes, then convert to hours
        let offsetMinutes = tzTime - utcTime;
        
        // Handle day rollover (if timezone is ahead/behind by more than 12 hours)
        if (offsetMinutes > 12 * 60) {
            offsetMinutes -= 24 * 60; // Subtract a full day
        } else if (offsetMinutes < -12 * 60) {
            offsetMinutes += 24 * 60; // Add a full day
        }
        
        const offsetHours = offsetMinutes / 60;
        const sign = offsetHours >= 0 ? '+' : '';
        
        return ` (UTC${sign}${Math.round(offsetHours)})`;
    } catch (error) {
        console.error('Error calculating UTC offset:', error);
        return '';
    }
}

// Display location information
function displayLocationInfo(location, noaaStation = null) {
    let html = '<div class="location-info">';
    html += '<div class="section-header">Location Information</div>';
    
    const utcOffsetStr = getUtcOffsetString(location.timeZone);
    html += `<div class="location-info-item">Time Zone: ${location.timeZone}${utcOffsetStr}</div>`;
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
    
    // Display NOAA Station and Resources if a tide station is found within 100 miles
    if (noaaStation) {
        // Display NOAA Station information first with clickable station ID
        html += '<div class="location-info-item">';
        const stationHomeUrl = `https://tidesandcurrents.noaa.gov/stationhome.html?id=${noaaStation.stationId}`;
        
        // Calculate bearing and cardinal direction from location to station
        const bearing = calculateBearing(location.lat, location.lon, noaaStation.lat, noaaStation.lon);
        const cardinalDir = getCardinalDirection(bearing);
        const distanceStr = `${noaaStation.distance.toFixed(2)}mi`;
        
        html += `NOAA Station: <span class="noaa-station-name">${noaaStation.name} (<a href="${stationHomeUrl}" target="_blank" class="location-info-link no-margin">${noaaStation.stationId}</a>)</span> ${distanceStr} ${cardinalDir}`;
        html += '</div>';
        
        // Display NOAA Resources
        html += '<div class="location-info-item">NOAA Resources: ';
        const tideUrl = `https://tidesandcurrents.noaa.gov/noaatidepredictions.html?id=${noaaStation.stationId}`;
        const datumsUrl = `https://tidesandcurrents.noaa.gov/datums.html?id=${noaaStation.stationId}`;
        
        html += `<a href="${tideUrl}" target="_blank" class="location-info-link">Tide Prediction</a>`;
        html += `<a href="${datumsUrl}" target="_blank" class="location-info-link">Datums</a>`;
        
        if (noaaStation.supportsWaterLevels) {
            const waterLevelsUrl = `https://tidesandcurrents.noaa.gov/waterlevels.html?id=${noaaStation.stationId}`;
            html += `<a href="${waterLevelsUrl}" target="_blank" class="location-info-link">Levels</a>`;
        }
        html += '</div>';
        
        // Display tide predictions if available
        if (noaaStation.tideData) {
            const lastTide = noaaStation.tideData.lastTide;
            const nextTide = noaaStation.tideData.nextTide;
            
            html += '<div class="location-info-item">';
            html += '<span class="location-info-label">Tides: </span>';
            html += '<span class="noaa-tide-info">';
            
            // Display last tide if available
            if (lastTide) {
                const lastHeight = `${lastTide.height.toFixed(2)}ft`;
                const lastHour = String(lastTide.time.getHours()).padStart(2, '0');
                const lastMin = String(lastTide.time.getMinutes()).padStart(2, '0');
                const lastTime = `${lastHour}${lastMin}`;
                const lastArrow = lastTide.type === 'L' ? '↓' : '↑';
                html += `Last${lastArrow}: ${lastHeight}@${lastTime}`;
            }
            
            // Add space between last and next if both are present
            if (lastTide && nextTide) {
                html += ' ';
            }
            
            // Display next tide if available
            if (nextTide) {
                const nextHeight = `${nextTide.height.toFixed(2)}ft`;
                const nextHour = String(nextTide.time.getHours()).padStart(2, '0');
                const nextMin = String(nextTide.time.getMinutes()).padStart(2, '0');
                const nextTime = `${nextHour}${nextMin}`;
                const nextArrow = nextTide.type === 'H' ? '↑' : '↓';
                html += `Next${nextArrow}: ${nextHeight}@${nextTime}`;
            }
            
            html += '</span>';
            html += '</div>';
        }
    }
    
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
    html += displayWeatherAlerts(weather.alerts, true, location.timeZone);
    html += displayLocationInfo(location, weather.noaaStation);
    return html;
}

// Display observations (history) mode
function displayObservations(observationsData, location) {
    if (!observationsData || observationsData.length === 0) {
        return '<div class="error-message">No historical observations available.</div>';
    }
    
    const cityName = truncateCityName(location.city, 20);
    let html = `<div class="section-header">${cityName} Observations</div>`;
    
    // Reverse the array so most recent observations appear first
    const reversedData = [...observationsData].reverse();
    
    reversedData.forEach(dayData => {
        const [year, month, day] = dayData.date.split('-').map(Number);
        const date = new Date(year, month - 1, day);
        const dayName = getDayName(date, false);
        const dateStr = `${String(month).padStart(2, '0')}/${String(day).padStart(2, '0')}`;
        
        // Calculate moon phase for this day
        const moonPhaseInfo = calculateMoonPhase(date);
        const moonEmoji = moonPhaseInfo.emoji;
        
        html += '<div class="daily-item">';
        html += `<div class="daily-header">${dayName} (${dateStr}):</div>`;
        
        // Calculate sunrise/sunset for this specific observation date (use date only, not time)
        let sunriseStr = "";
        let sunsetStr = "";
        let dayLengthStr = "";
        if (location.lat && location.lon && location.timeZone) {
        // Use the location's date (not viewer local) for accurate sunrise/sunset calculation
        const dateLocal = location.timeZone ? convertToTimeZone(date, location.timeZone) : date;
        const dayDate = new Date(dateLocal.getFullYear(), dateLocal.getMonth(), dateLocal.getDate());
            const daySunTimes = calculateSunriseSunset(
                location.lat,
                location.lon,
                dayDate,
                location.timeZone
            );
            if (daySunTimes.sunrise) {
                // Format sunrise: date/time (MM/dd HH:mm) if polar night/day, otherwise time (24-hour format)
                // During polar night: shows next sunrise; during polar day: shows last sunrise
                sunriseStr = (daySunTimes.isPolarNight || daySunTimes.isPolarDay) ? formatSunriseDate(daySunTimes.sunrise, location.timeZone) : formatTime24(daySunTimes.sunrise, location.timeZone);
                // Show sunset if available (during polar night: last sunset; during polar day: next sunset)
                if (daySunTimes.sunset) {
                    sunsetStr = (daySunTimes.isPolarNight || daySunTimes.isPolarDay) ? formatSunriseDate(daySunTimes.sunset, location.timeZone) : formatTime24(daySunTimes.sunset, location.timeZone);
                    // Only show day length if not polar night/day (normal day)
                    if (!daySunTimes.isPolarNight && !daySunTimes.isPolarDay) {
                        dayLengthStr = formatDayLength(daySunTimes.sunrise, daySunTimes.sunset);
                    }
                }
            }
        }
        
        // Display sunrise/sunset/day length if available
        if (sunriseStr) {
            if (sunsetStr) {
                if (dayLengthStr) {
                    // Normal case: show sunrise, sunset, and day length
                    html += `<div class="condition-row"> Sunrise: <span class="forecast-text">${sunriseStr}</span> Sunset: <span class="forecast-text">${sunsetStr}</span> Day Length: <span class="forecast-text">${dayLengthStr}</span></div>`;
                } else {
                    // Polar night/day: show sunrise and sunset (with date/time), no day length
                    html += `<div class="condition-row"> Sunrise: <span class="forecast-text">${sunriseStr}</span> Sunset: <span class="forecast-text">${sunsetStr}</span></div>`;
                }
            } else {
                // Only sunrise available
                html += `<div class="condition-row"> Sunrise: <span class="forecast-text">${sunriseStr}</span></div>`;
            }
            html += '<div></div>'; // Line feed after day length (or sunrise/sunset)
        }
        
        // Temp line: H:{high}°F L:{low}°F with windchill/heat index if applicable
        html += '<div class="condition-row">';
        html += ' <span class="condition-label">Temp:</span>';
        
        if (dayData.highTemp !== null) {
            html += `<span class="condition-value ${getTempColor(dayData.highTemp)}">H:${dayData.highTemp}°F</span>`;
        } else {
            html += '<span class="condition-value">H:N/A</span>';
        }
        
        // Calculate windchill or heat index
        let windChill = null;
        let heatIndex = null;
        if (dayData.highTemp !== null && dayData.avgWindSpeed !== null) {
            const tempNum = dayData.highTemp;
            const windSpeedNum = dayData.avgWindSpeed;
            
            if (tempNum <= 50) {
                windChill = calculateWindChill(tempNum, windSpeedNum);
                if (windChill && Math.abs(tempNum - windChill) <= 1) {
                    windChill = null; // Only show if difference > 1°F
                }
            } else if (tempNum >= 80 && dayData.avgHumidity !== null) {
                heatIndex = calculateHeatIndex(tempNum, dayData.avgHumidity);
                if (heatIndex && Math.abs(heatIndex - tempNum) <= 1) {
                    heatIndex = null; // Only show if difference > 1°F
                }
            }
        }
        
        if (windChill) {
            html += ` <span class="condition-value">Windchill[<span class="temp-cold">${windChill}°F</span>]</span>`;
        } else if (heatIndex) {
            html += ` <span class="condition-value">HeatIndex[<span class="temp-hot">${heatIndex}°F</span>]</span>`;
        }
        
        if (dayData.lowTemp !== null) {
            html += ` <span class="condition-value ${getTempColor(dayData.lowTemp)}">L:${dayData.lowTemp}°F</span>`;
        } else {
            html += ' <span class="condition-value">L:N/A</span>';
        }
        
        html += '</div>';
        
        // Wind line: Color code avg and gust separately
        html += '<div class="condition-row">';
        html += '<span class="condition-label">Wind:</span>';
        
        const windDir = dayData.windDirection !== null ? getCardinalDirection(dayData.windDirection) : '';
        
        if (dayData.avgWindSpeed !== null) {
            const avgWindSpeedNum = Math.round(dayData.avgWindSpeed);
            const avgWindColor = getWindColor(avgWindSpeedNum);
            
            // Build wind display with separate color coding for avg and gust
            if (dayData.maxWindGust !== null) {
                // Show average with gust (preferred - most accurate)
                const maxWindGustNum = Math.round(dayData.maxWindGust);
                const gustWindColor = getWindColor(maxWindGustNum);
                html += `<span class="condition-value ${avgWindColor}">avg ${avgWindSpeedNum}mph</span>`;
                html += ` <span class="condition-value ${gustWindColor}">gust ${maxWindGustNum}mph</span>`;
                if (windDir) {
                    html += ` <span class="condition-value">${windDir}</span>`;
                }
            } else if (dayData.maxWindSpeed !== null) {
                // Show average with max sustained wind (fallback if no gust data)
                const maxWindSpeedNum = Math.round(dayData.maxWindSpeed);
                if (Math.abs(dayData.maxWindSpeed - dayData.avgWindSpeed) > 1) {
                    // Only show max if it differs significantly from avg
                    const maxWindColor = getWindColor(maxWindSpeedNum);
                    html += `<span class="condition-value ${avgWindColor}">avg ${avgWindSpeedNum}mph</span>`;
                    html += ` <span class="condition-value ${maxWindColor}">max ${maxWindSpeedNum}mph</span>`;
                    if (windDir) {
                        html += ` <span class="condition-value">${windDir}</span>`;
                    }
                } else {
                    // If max and avg are similar, just show avg
                    html += `<span class="condition-value ${avgWindColor}">avg ${avgWindSpeedNum}mph</span>`;
                    if (windDir) {
                        html += ` <span class="condition-value">${windDir}</span>`;
                    }
                }
            } else {
                // Just show average if no max/gust data
                html += `<span class="condition-value ${avgWindColor}">avg ${avgWindSpeedNum}mph</span>`;
                if (windDir) {
                    html += ` <span class="condition-value">${windDir}</span>`;
                }
            }
        } else if (dayData.maxWindSpeed !== null) {
            // Fallback to max if avg not available
            const maxWindSpeedNum = Math.round(dayData.maxWindSpeed);
            const maxWindColor = getWindColor(maxWindSpeedNum);
            html += `<span class="condition-value ${maxWindColor}">max ${maxWindSpeedNum}mph</span>`;
            if (windDir) {
                html += ` <span class="condition-value">${windDir}</span>`;
            }
        } else if (dayData.maxWindGust !== null) {
            // Fallback to gust if available
            const maxWindGustNum = Math.round(dayData.maxWindGust);
            const gustWindColor = getWindColor(maxWindGustNum);
            html += `<span class="condition-value ${gustWindColor}">gust ${maxWindGustNum}mph</span>`;
            if (windDir) {
                html += ` <span class="condition-value">${windDir}</span>`;
            }
        } else {
            html += '<span class="condition-value">N/A</span>';
        }
        
        html += '</div>';
        
        // Precipitation line: Precip: {total}" if > 0
        if (dayData.totalPrecipitation > 0) {
            html += '<div class="condition-row">';
            html += '<span class="condition-label">Precip:</span>';
            html += `<span class="condition-value ${getPrecipColor(dayData.totalPrecipitation * 100)}">${dayData.totalPrecipitation}"</span>`;
            html += '</div>';
        }
        
        // Humidity line: Humidity: {avg}% RH
        if (dayData.avgHumidity !== null) {
            html += '<div class="condition-row">';
            html += '<span class="condition-label">Humidity:</span>';
            html += `<span class="condition-value ${getHumidityColor(dayData.avgHumidity)}">${Math.round(dayData.avgHumidity)}% RH</span>`;
            html += '</div>';
        }
        
        // Conditions line: {moonEmoji} Conditions: {description}
        html += '<div></div>'; // Line feed before moon phase icon
        html += '<div class="condition-row">';
        html += `<span class="condition-label">${moonEmoji} Conditions:</span>`;
        html += `<span class="condition-value forecast-text">${dayData.conditions}</span>`;
        html += '</div>';
        
        html += '</div>';
    });
    
    return html;
}

