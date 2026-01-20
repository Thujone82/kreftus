// Utility functions for weather calculations and formatting

// Convert degrees to radians
function toRadians(deg) {
    return Math.PI * deg / 180.0;
}

// Convert radians to degrees
function toDegrees(rad) {
    return 180.0 * rad / Math.PI;
}

// Calculate sunrise and sunset times using NOAA astronomical algorithms
function calculateSunriseSunset(latitude, longitude, date, timeZoneId) {
    const zenithDegrees = 90.833; // Includes standard atmospheric refraction
    
    const latRad = toRadians(latitude);
    const dayOfYear = getDayOfYear(date);
    
    // Fractional year (radians) for day N at 12:00
    const gamma = 2.0 * Math.PI * (dayOfYear - 1) / 365.0;
    
    // Equation of time (minutes) - NOAA approximation
    const equationOfTime = 229.18 * (
        0.000075 + 
        0.001868 * Math.cos(gamma) - 
        0.032077 * Math.sin(gamma) - 
        0.014615 * Math.cos(2 * gamma) - 
        0.040849 * Math.sin(2 * gamma)
    );
    
    // Solar declination (radians) - NOAA series
    const declination = 0.006918 - 
        0.399912 * Math.cos(gamma) + 
        0.070257 * Math.sin(gamma) - 
        0.006758 * Math.cos(2 * gamma) + 
        0.000907 * Math.sin(2 * gamma) - 
        0.002697 * Math.cos(3 * gamma) + 
        0.00148 * Math.sin(3 * gamma);
    
    // Hour angle for the sun at sunrise/sunset
    const cosH = (Math.cos(toRadians(zenithDegrees)) - Math.sin(latRad) * Math.sin(declination)) / 
                 (Math.cos(latRad) * Math.cos(declination));
    
    if (cosH > 1) {
        // Polar night - no sunrise today, find next sunrise and last sunset
        let nextSunrise = null;
        let lastSunset = null;
        const maxDaysToCheck = 180; // Check up to 6 months ahead/back
        
        // Find next sunrise (forward)
        for (let dayOffset = 1; dayOffset <= maxDaysToCheck; dayOffset++) {
            const checkDate = new Date(date);
            checkDate.setDate(checkDate.getDate() + dayOffset);
            const checkDayOfYear = getDayOfYear(checkDate);
            const checkGamma = 2.0 * Math.PI * (checkDayOfYear - 1) / 365.0;
            const checkDeclination = 0.006918 - 
                0.399912 * Math.cos(checkGamma) + 
                0.070257 * Math.sin(checkGamma) - 
                0.006758 * Math.cos(2 * checkGamma) + 
                0.000907 * Math.sin(2 * checkGamma) - 
                0.002697 * Math.cos(3 * checkGamma) + 
                0.00148 * Math.sin(3 * checkGamma);
            const checkCosH = (Math.cos(toRadians(zenithDegrees)) - Math.sin(latRad) * Math.sin(checkDeclination)) / 
                             (Math.cos(latRad) * Math.cos(checkDeclination));
            
            if (checkCosH <= 1) {
                // Found a day with sunrise
                const checkH = Math.acos(Math.min(1.0, Math.max(-1.0, checkCosH)));
                const checkHdeg = toDegrees(checkH);
                const checkEquationOfTime = 229.18 * (
                    0.000075 + 
                    0.001868 * Math.cos(checkGamma) - 
                    0.032077 * Math.sin(checkGamma) - 
                    0.014615 * Math.cos(2 * checkGamma) - 
                    0.040849 * Math.sin(2 * checkGamma)
                );
                const checkSolarNoonUtcMin = 720.0 - 4.0 * longitude - checkEquationOfTime;
                let checkSunriseUtcMin = checkSolarNoonUtcMin - 4.0 * checkHdeg;
                
                while (checkSunriseUtcMin < 0) checkSunriseUtcMin += 1440;
                while (checkSunriseUtcMin >= 1440) checkSunriseUtcMin -= 1440;
                
                const checkUtcMidnight = new Date(Date.UTC(checkDate.getUTCFullYear(), checkDate.getUTCMonth(), checkDate.getUTCDate(), 0, 0, 0));
                const checkSunriseUtc = new Date(checkUtcMidnight.getTime() + checkSunriseUtcMin * 60000);
                // Keep as UTC; display helpers format with the target timeZone
                nextSunrise = checkSunriseUtc;
                break;
            }
        }
        
        // Find last sunset (backward)
        for (let dayOffset = -1; dayOffset >= -maxDaysToCheck; dayOffset--) {
            const checkDate = new Date(date);
            checkDate.setDate(checkDate.getDate() + dayOffset);
            const checkDayOfYear = getDayOfYear(checkDate);
            const checkGamma = 2.0 * Math.PI * (checkDayOfYear - 1) / 365.0;
            const checkDeclination = 0.006918 - 
                0.399912 * Math.cos(checkGamma) + 
                0.070257 * Math.sin(checkGamma) - 
                0.006758 * Math.cos(2 * checkGamma) + 
                0.000907 * Math.sin(2 * checkGamma) - 
                0.002697 * Math.cos(3 * checkGamma) + 
                0.00148 * Math.sin(3 * checkGamma);
            const checkCosH = (Math.cos(toRadians(zenithDegrees)) - Math.sin(latRad) * Math.sin(checkDeclination)) / 
                             (Math.cos(latRad) * Math.cos(checkDeclination));
            
            if (checkCosH <= 1) {
                // Found a day with sunset
                const checkH = Math.acos(Math.min(1.0, Math.max(-1.0, checkCosH)));
                const checkHdeg = toDegrees(checkH);
                const checkEquationOfTime = 229.18 * (
                    0.000075 + 
                    0.001868 * Math.cos(checkGamma) - 
                    0.032077 * Math.sin(checkGamma) - 
                    0.014615 * Math.cos(2 * checkGamma) - 
                    0.040849 * Math.sin(2 * checkGamma)
                );
                const checkSolarNoonUtcMin = 720.0 - 4.0 * longitude - checkEquationOfTime;
                let checkSunsetUtcMin = checkSolarNoonUtcMin + 4.0 * checkHdeg;
                
                while (checkSunsetUtcMin < 0) checkSunsetUtcMin += 1440;
                while (checkSunsetUtcMin >= 1440) checkSunsetUtcMin -= 1440;
                
                const checkUtcMidnight = new Date(Date.UTC(checkDate.getUTCFullYear(), checkDate.getUTCMonth(), checkDate.getUTCDate(), 0, 0, 0));
                const checkSunsetUtc = new Date(checkUtcMidnight.getTime() + checkSunsetUtcMin * 60000);
                // Keep as UTC; display helpers format with the target timeZone
                lastSunset = checkSunsetUtc;
                break;
            }
        }
        
        return { sunrise: nextSunrise, sunset: lastSunset, isPolarNight: true, isPolarDay: false };
    }
    if (cosH < -1) {
        // Polar day - no sunset today, find next sunset and last sunrise
        let nextSunset = null;
        let lastSunrise = null;
        const maxDaysToCheck = 180; // Check up to 6 months ahead/back
        
        // Find next sunset (forward)
        for (let dayOffset = 1; dayOffset <= maxDaysToCheck; dayOffset++) {
            const checkDate = new Date(date);
            checkDate.setDate(checkDate.getDate() + dayOffset);
            const checkDayOfYear = getDayOfYear(checkDate);
            const checkGamma = 2.0 * Math.PI * (checkDayOfYear - 1) / 365.0;
            const checkDeclination = 0.006918 - 
                0.399912 * Math.cos(checkGamma) + 
                0.070257 * Math.sin(checkGamma) - 
                0.006758 * Math.cos(2 * checkGamma) + 
                0.000907 * Math.sin(2 * checkGamma) - 
                0.002697 * Math.cos(3 * checkGamma) + 
                0.00148 * Math.sin(3 * checkGamma);
            const checkCosH = (Math.cos(toRadians(zenithDegrees)) - Math.sin(latRad) * Math.sin(checkDeclination)) / 
                             (Math.cos(latRad) * Math.cos(checkDeclination));
            
            if (checkCosH <= 1) {
                // Found a day with sunset
                const checkH = Math.acos(Math.min(1.0, Math.max(-1.0, checkCosH)));
                const checkHdeg = toDegrees(checkH);
                const checkEquationOfTime = 229.18 * (
                    0.000075 + 
                    0.001868 * Math.cos(checkGamma) - 
                    0.032077 * Math.sin(checkGamma) - 
                    0.014615 * Math.cos(2 * checkGamma) - 
                    0.040849 * Math.sin(2 * checkGamma)
                );
                const checkSolarNoonUtcMin = 720.0 - 4.0 * longitude - checkEquationOfTime;
                let checkSunsetUtcMin = checkSolarNoonUtcMin + 4.0 * checkHdeg;
                
                while (checkSunsetUtcMin < 0) checkSunsetUtcMin += 1440;
                while (checkSunsetUtcMin >= 1440) checkSunsetUtcMin -= 1440;
                
                const checkUtcMidnight = new Date(Date.UTC(checkDate.getUTCFullYear(), checkDate.getUTCMonth(), checkDate.getUTCDate(), 0, 0, 0));
                const checkSunsetUtc = new Date(checkUtcMidnight.getTime() + checkSunsetUtcMin * 60000);
                // Keep as UTC; display helpers format with the target timeZone
                nextSunset = checkSunsetUtc;
                break;
            }
        }
        
        // Find last sunrise (backward)
        for (let dayOffset = -1; dayOffset >= -maxDaysToCheck; dayOffset--) {
            const checkDate = new Date(date);
            checkDate.setDate(checkDate.getDate() + dayOffset);
            const checkDayOfYear = getDayOfYear(checkDate);
            const checkGamma = 2.0 * Math.PI * (checkDayOfYear - 1) / 365.0;
            const checkDeclination = 0.006918 - 
                0.399912 * Math.cos(checkGamma) + 
                0.070257 * Math.sin(checkGamma) - 
                0.006758 * Math.cos(2 * checkGamma) + 
                0.000907 * Math.sin(2 * checkGamma) - 
                0.002697 * Math.cos(3 * checkGamma) + 
                0.00148 * Math.sin(3 * checkGamma);
            const checkCosH = (Math.cos(toRadians(zenithDegrees)) - Math.sin(latRad) * Math.sin(checkDeclination)) / 
                             (Math.cos(latRad) * Math.cos(checkDeclination));
            
            if (checkCosH <= 1) {
                // Found a day with sunrise
                const checkH = Math.acos(Math.min(1.0, Math.max(-1.0, checkCosH)));
                const checkHdeg = toDegrees(checkH);
                const checkEquationOfTime = 229.18 * (
                    0.000075 + 
                    0.001868 * Math.cos(checkGamma) - 
                    0.032077 * Math.sin(checkGamma) - 
                    0.014615 * Math.cos(2 * checkGamma) - 
                    0.040849 * Math.sin(2 * checkGamma)
                );
                const checkSolarNoonUtcMin = 720.0 - 4.0 * longitude - checkEquationOfTime;
                let checkSunriseUtcMin = checkSolarNoonUtcMin - 4.0 * checkHdeg;
                
                while (checkSunriseUtcMin < 0) checkSunriseUtcMin += 1440;
                while (checkSunriseUtcMin >= 1440) checkSunriseUtcMin -= 1440;
                
                const checkUtcMidnight = new Date(Date.UTC(checkDate.getUTCFullYear(), checkDate.getUTCMonth(), checkDate.getUTCDate(), 0, 0, 0));
                const checkSunriseUtc = new Date(checkUtcMidnight.getTime() + checkSunriseUtcMin * 60000);
                // Keep as UTC; display helpers format with the target timeZone
                lastSunrise = checkSunriseUtc;
                break;
            }
        }
        
        return { sunrise: lastSunrise, sunset: nextSunset, isPolarNight: false, isPolarDay: true };
    }
    
    const H = Math.acos(Math.min(1.0, Math.max(-1.0, cosH)));
    const Hdeg = toDegrees(H);
    
    // Solar noon in minutes from UTC midnight
    const solarNoonUtcMin = 720.0 - 4.0 * longitude - equationOfTime;
    
    // Sunrise/Sunset in minutes from UTC midnight
    const sunriseUtcMin = solarNoonUtcMin - 4.0 * Hdeg;
    const sunsetUtcMin = solarNoonUtcMin + 4.0 * Hdeg;
    
    // Normalize to 0..1440 range
    let sunriseMin = sunriseUtcMin;
    let sunsetMin = sunsetUtcMin;
    while (sunriseMin < 0) sunriseMin += 1440;
    while (sunriseMin >= 1440) sunriseMin -= 1440;
    while (sunsetMin < 0) sunsetMin += 1440;
    while (sunsetMin >= 1440) sunsetMin -= 1440;
    
    // Build UTC DateTimes
    // Use UTC methods to ensure consistent calculation regardless of input date timezone
    const utcMidnight = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate(), 0, 0, 0));
    const sunriseUtc = new Date(utcMidnight.getTime() + sunriseMin * 60000);
    const sunsetUtc = new Date(utcMidnight.getTime() + sunsetMin * 60000);
    
    return {
        // Keep as UTC; downstream formatters apply the correct timeZone for display
        sunrise: sunriseUtc,
        sunset: sunsetUtc,
        isPolarDay: false,
        isPolarNight: false
    };
}

// Get day of year (1-365/366)
// Works correctly with both UTC and local dates
function getDayOfYear(date) {
    // Use UTC methods to ensure consistent calculation regardless of timezone
    const year = date.getUTCFullYear();
    const start = new Date(Date.UTC(year, 0, 0));
    const diff = date - start;
    return Math.floor(diff / (1000 * 60 * 60 * 24));
}

// Convert UTC date to timezone (simplified - uses Intl.DateTimeFormat)
function convertToTimeZone(date, timeZoneId) {
    if (!timeZoneId) return date;
    
    try {
        const formatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false
        });
        
        const parts = formatter.formatToParts(date);
        const year = parseInt(parts.find(p => p.type === 'year').value);
        const month = parseInt(parts.find(p => p.type === 'month').value) - 1;
        const day = parseInt(parts.find(p => p.type === 'day').value);
        const hour = parseInt(parts.find(p => p.type === 'hour').value);
        const minute = parseInt(parts.find(p => p.type === 'minute').value);
        const second = parseInt(parts.find(p => p.type === 'second').value);
        
        return new Date(year, month, day, hour, minute, second);
    } catch (error) {
        return date;
    }
}

// Calculate moon phase using astronomical method
function calculateMoonPhase(date) {
    const knownNewMoon = new Date(Date.UTC(2000, 0, 6, 18, 14, 0));
    const lunarCycle = 29.53058867;
    
    // Calculate phase (0-1 range)
    const daysSince = (date.getTime() - knownNewMoon.getTime()) / (1000 * 60 * 60 * 24);
    const currentCycle = daysSince % lunarCycle;
    const phase = currentCycle / lunarCycle;
    
    // Determine phase name and emoji
    let phaseName = "";
    let emoji = "";
    
    if (phase < 0.125) {
        phaseName = "New Moon";
        emoji = "🌑";
    } else if (phase < 0.25) {
        phaseName = "Waxing Crescent";
        emoji = "🌒";
    } else if (phase < 0.375) {
        phaseName = "First Quarter";
        emoji = "🌓";
    } else if (phase < 0.48) {
        phaseName = "Waxing Gibbous";
        emoji = "🌔";
    } else if (phase < 0.52) {
        phaseName = "Full Moon";
        emoji = "🌕";
    } else if (phase < 0.75) {
        phaseName = "Waning Gibbous";
        emoji = "🌖";
    } else if (phase < 0.875) {
        phaseName = "Last Quarter";
        emoji = "🌗";
    } else {
        phaseName = "Waning Crescent";
        emoji = "🌘";
    }
    
    // Calculate next full moon
    const daysUntilNextFullMoon = (14.77 - currentCycle) % lunarCycle;
    const nextFullMoonDays = daysUntilNextFullMoon <= 0 ? daysUntilNextFullMoon + lunarCycle : daysUntilNextFullMoon;
    const nextFullMoonDate = new Date(date.getTime() + nextFullMoonDays * 24 * 60 * 60 * 1000);
    
    // Calculate next new moon
    const daysUntilNextNewMoon = lunarCycle - currentCycle;
    const nextNewMoonDate = new Date(date.getTime() + daysUntilNextNewMoon * 24 * 60 * 60 * 1000);
    
    return {
        name: phaseName,
        emoji: emoji,
        isFullMoon: phase >= 0.48 && phase < 0.52,
        isNewMoon: phase < 0.125,
        showNextFullMoon: phase < 0.48,
        showNextNewMoon: phase >= 0.52,
        nextFullMoon: formatDate(nextFullMoonDate),
        nextNewMoon: formatDate(nextNewMoonDate)
    };
}

// Format date as MM/dd/yyyy
function formatDate(date) {
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const year = date.getFullYear();
    return `${month}/${day}/${year}`;
}

// Extract numeric wind speed from wind speed string
function getWindSpeed(windString) {
    if (!windString) return 0;
    const match = windString.match(/(\d+)/);
    return match ? parseInt(match[1]) : 0;
}

// Calculate wind chill using NWS formula
function calculateWindChill(tempF, windSpeedMph) {
    // Wind chill only applies when temp <= 50°F and wind speed >= 3 mph
    if (tempF > 50 || windSpeedMph < 3) {
        return null;
    }
    
    // NWS Wind Chill Formula
    const windChill = 35.74 + 
        (0.6215 * tempF) - 
        (35.75 * Math.pow(windSpeedMph, 0.16)) + 
        (0.4275 * tempF * Math.pow(windSpeedMph, 0.16));
    
    return Math.round(windChill);
}

// Calculate heat index using NWS Rothfusz regression
function calculateHeatIndex(tempF, humidity) {
    // Heat index only applies when temp >= 80°F
    if (tempF < 80) {
        return null;
    }
    
    const T = tempF;
    const RH = humidity;
    
    // Simple formula for initial estimate
    let HI = 0.5 * (T + 61.0 + ((T - 68.0) * 1.2) + (RH * 0.094));
    
    // If >= 80°F, use full Rothfusz regression
    if (HI >= 80) {
        HI = -42.379 + 
            (2.04901523 * T) + 
            (10.14333127 * RH) - 
            (0.22475541 * T * RH) - 
            (0.00683783 * T * T) - 
            (0.05481717 * RH * RH) + 
            (0.00122874 * T * T * RH) + 
            (0.00085282 * T * RH * RH) - 
            (0.00000199 * T * T * RH * RH);
        
        // Adjustments for low/high RH
        if (RH < 13 && T >= 80 && T <= 112) {
            HI = HI - ((13 - RH) / 4) * Math.sqrt((17 - Math.abs(T - 95)) / 17);
        } else if (RH > 85 && T >= 80 && T <= 87) {
            HI = HI + ((RH - 85) / 10) * ((87 - T) / 5);
        }
    }
    
    return Math.round(HI);
}

// Get weather icon emoji from NWS icon URL
function getWeatherIcon(iconUrl, isDaytime = true, precipProb = 0) {
    if (!iconUrl) return "";
    
    // Extract weather condition from NWS icon URL
    const match = iconUrl.match(/\/([^/]+)\?/);
    if (!match) return isDaytime ? "☁️" : "🌙";
    
    const condition = match[1];
    
    // Prioritize precipitation-related conditions
    if (condition.match(/tsra/)) return "⛈️";  // Thunderstorm
    if (condition.match(/rain/) && precipProb >= 50) return "🌧️";  // Rain
    if (condition.match(/snow/)) return "❄️";  // Snow
    if (condition.match(/fzra/)) return "🧊";  // Freezing rain
    
    // Other weather conditions
    if (condition.match(/fog|haze/)) return "🌫️";  // Fog/Haze
    if (condition.match(/smoke|dust|wind/)) return "💨";  // Smoke/Dust/Wind
    
    // Cloud conditions
    if (condition.match(/ovc/)) return "☁️";  // Overcast
    if (condition.match(/bkn/)) return "☁️";  // Broken clouds
    if (condition.match(/sct/)) return isDaytime ? "⛅" : "☁️";  // Scattered clouds
    if (condition.match(/few/)) return isDaytime ? "🌤️" : "🌙";  // Few clouds
    if (condition.match(/skc/)) return isDaytime ? "☀️" : "🌙";  // Clear
    
    // Generic fallbacks
    if (condition.match(/cloud|shower|drizzle/)) return "☁️";
    
    // Default fallback
    return isDaytime ? "☁️" : "🌙";
}

// Get rain sparkline character and color
function getRainSparkline(rainPercent) {
    if (rainPercent === 0) return { char: " ", color: "white" };
    if (rainPercent <= 10) return { char: "▁", color: "white" };
    if (rainPercent <= 33) return { char: "▂", color: "cyan" };
    if (rainPercent <= 44) return { char: "▃", color: "green" };
    if (rainPercent <= 66) return { char: "▄", color: "yellow" };
    if (rainPercent <= 80) return { char: "▅", color: "yellow" };
    return { char: "▇", color: "red" };
}

// Get background color for rain block based on percentage
function getRainBlockColor(rainPercent) {
    if (rainPercent === 0) return "transparent";
    if (rainPercent <= 10) return "#ffffff"; // white
    if (rainPercent <= 33) return "#00ffff"; // cyan
    if (rainPercent <= 44) return "#00ff00"; // green
    if (rainPercent <= 80) return "#ffff00"; // yellow
    return "#ff0000"; // red
}

// Get wind direction glyph and color
function getWindGlyph(windDirection, windSpeed) {
    const directionMap = {
        "N": 0, "NNE": 0, "NNW": 7,
        "NE": 1, "ENE": 1,
        "E": 2, "ESE": 2,
        "SE": 3, "SSE": 3,
        "S": 4, "SSW": 4,
        "SW": 5, "WSW": 5,
        "W": 6, "WNW": 6,
        "NW": 7
    };
    
    const dirIndex = directionMap[windDirection] || 0;
    
    // Choose glyph set based on wind speed
    const glyphs = windSpeed < 7 
        ? ["▽", "◺", "◁", "◸", "△", "◹", "▷", "◿"]
        : ["▼", "◣", "◀", "◤", "▲", "◥", "▶", "◢"];
    
    const glyph = glyphs[dirIndex];
    
    // Get color based on wind speed
    let color = "white";
    if (windSpeed > 14) color = "magenta";
    else if (windSpeed > 9) color = "red";
    else if (windSpeed > 5) color = "yellow";
    
    return { char: glyph, color: color };
}

// Wrap text to specified width
function wrapText(text, width) {
    const words = text.split(/\s+/);
    const lines = [];
    let currentLine = '';
    
    words.forEach(word => {
        if (currentLine.length === 0) {
            currentLine = word;
        } else if (currentLine.length + 1 + word.length <= width) {
            currentLine += ' ' + word;
        } else {
            lines.push(currentLine);
            currentLine = word;
        }
    });
    
    if (currentLine) {
        lines.push(currentLine);
    }
    
    return lines;
}

// Format time in location timezone
// Helper function to format day length as "Xh Ym"
function formatDayLength(sunrise, sunset) {
    if (!sunrise || !sunset) {
        return "N/A";
    }
    
    // Calculate day length: simply subtract sunrise from sunset
    // If sunset is earlier than sunrise, add 24 hours (sunset is next day)
    let durationMs = sunset.getTime() - sunrise.getTime();
    if (durationMs < 0) {
        // Sunset is next day, add 24 hours
        durationMs += 24 * 60 * 60 * 1000;
    }
    
    const totalMinutes = Math.round(durationMs / (1000 * 60));
    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;
    
    return `${hours}h ${minutes}m`;
}

function formatTime(date, timeZoneId) {
    if (!date) return "";
    
    try {
        const formatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            hour: 'numeric',
            minute: '2-digit',
            hour12: true
        });
        return formatter.format(date);
    } catch (error) {
        // Fallback: format manually using timezone conversion
        if (timeZoneId) {
            try {
                const converted = convertToTimeZone(date, timeZoneId);
                return converted.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true });
            } catch (e) {
                // If conversion fails, fall through to local time
            }
        }
        // Final fallback: use local time (not ideal, but better than error)
        return date.toLocaleTimeString();
    }
}

// Format time in 24-hour format (HH:mm)
function formatTime24(date, timeZoneId) {
    if (!date) return "";
    
    try {
        const formatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        });
        return formatter.format(date);
    } catch (error) {
        // Fallback: format manually using timezone conversion
        if (timeZoneId) {
            try {
                const converted = convertToTimeZone(date, timeZoneId);
                const hours = converted.getHours().toString().padStart(2, '0');
                const minutes = converted.getMinutes().toString().padStart(2, '0');
                return `${hours}:${minutes}`;
            } catch (e) {
                // If conversion fails, fall through to local time
            }
        }
        // Final fallback: use local time (not ideal, but better than error)
        const hours = date.getHours().toString().padStart(2, '0');
        const minutes = date.getMinutes().toString().padStart(2, '0');
        return `${hours}:${minutes}`;
    }
}

// Format date/time in specified timezone (or viewer's local timezone if timeZoneId is null/undefined)
function formatDateTime(date, timeZoneId) {
    if (!date) return "";
    
    try {
        const options = {
            month: '2-digit',
            day: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            hour12: true
        };
        
        // Only set timeZone if provided (null/undefined means use viewer's local timezone)
        if (timeZoneId) {
            options.timeZone = timeZoneId;
        }
        
        const formatter = new Intl.DateTimeFormat('en-US', options);
        return formatter.format(date);
    } catch (error) {
        return date.toLocaleString();
    }
}

// Format date/time in location timezone (24-hour format: MM/dd/yyyy HH:mm)
function formatDateTime24(date, timeZoneId) {
    if (!date) return "";
    
    try {
        if (!timeZoneId) {
            // If no timezone specified, use browser's local timezone
            timeZoneId = Intl.DateTimeFormat().resolvedOptions().timeZone;
        }
        
        const formatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            month: '2-digit',
            day: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        });
        
        // Format to match PowerShell format: MM/dd/yyyy HH:mm
        const parts = formatter.formatToParts(date);
        const month = parts.find(p => p.type === 'month').value;
        const day = parts.find(p => p.type === 'day').value;
        const year = parts.find(p => p.type === 'year').value;
        const hour = parts.find(p => p.type === 'hour').value;
        const minute = parts.find(p => p.type === 'minute').value;
        
        return `${month}/${day}/${year} ${hour}:${minute}`;
    } catch (error) {
        // Fallback: convert to timezone manually if possible, otherwise use local time
        try {
            if (timeZoneId) {
                const converted = convertToTimeZone(date, timeZoneId);
                const month = String(converted.getMonth() + 1).padStart(2, '0');
                const day = String(converted.getDate()).padStart(2, '0');
                const year = converted.getFullYear();
                const hour = String(converted.getHours()).padStart(2, '0');
                const min = String(converted.getMinutes()).padStart(2, '0');
                return `${month}/${day}/${year} ${hour}:${min}`;
            }
        } catch (e) {
            // If conversion fails, fall through to local time
        }
        // Final fallback: format manually using local time
        const d = new Date(date);
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        const year = d.getFullYear();
        const hour = String(d.getHours()).padStart(2, '0');
        const min = String(d.getMinutes()).padStart(2, '0');
        return `${month}/${day}/${year} ${hour}:${min}`;
    }
}

// Format sunrise date/time for polar night (MM/dd HH:mm format)
function formatSunriseDate(date, timeZoneId) {
    if (!date) return "";
    
    try {
        if (!timeZoneId) {
            timeZoneId = Intl.DateTimeFormat().resolvedOptions().timeZone;
        }
        
        const formatter = new Intl.DateTimeFormat('en-US', {
            timeZone: timeZoneId,
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        });
        
        const parts = formatter.formatToParts(date);
        const month = parts.find(p => p.type === 'month').value;
        const day = parts.find(p => p.type === 'day').value;
        const hour = parts.find(p => p.type === 'hour').value;
        const minute = parts.find(p => p.type === 'minute').value;
        
        return `${month}/${day} ${hour}:${minute}`;
    } catch (error) {
        // Fallback: convert to timezone manually if possible, otherwise use local time
        try {
            if (timeZoneId) {
                const converted = convertToTimeZone(date, timeZoneId);
                const month = String(converted.getMonth() + 1).padStart(2, '0');
                const day = String(converted.getDate()).padStart(2, '0');
                const hour = String(converted.getHours()).padStart(2, '0');
                const min = String(converted.getMinutes()).padStart(2, '0');
                return `${month}/${day} ${hour}:${min}`;
            }
        } catch (e) {
            // If conversion fails, fall through to local time
        }
        // Final fallback: format manually using local time
        const d = new Date(date);
        const month = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        const hour = String(d.getHours()).padStart(2, '0');
        const min = String(d.getMinutes()).padStart(2, '0');
        return `${month}/${day} ${hour}:${min}`;
    }
}

// Get temperature color class
function getTempColor(temp) {
    if (temp < 33) return "temp-cold";
    if (temp > 89) return "temp-hot";
    return "temp-normal";
}

// Get wind color class
function getWindColor(windSpeed) {
    if (windSpeed > 14) return "wind-strong";
    if (windSpeed > 9) return "wind-moderate";
    if (windSpeed > 5) return "wind-light";
    return "wind-calm";
}

// Get precipitation color class
function getPrecipColor(precipProb) {
    if (precipProb > 50) return "precip-high";
    if (precipProb > 20) return "precip-medium";
    return "precip-low";
}

// Get humidity color class
function getHumidityColor(humidity) {
    if (humidity > 70) return "humidity-high";
    if (humidity > 60) return "humidity-elevated";
    if (humidity < 30) return "humidity-low";
    return "humidity-normal";
}

// Get dew point color class
function getDewPointColor(dewPointF) {
    if (dewPointF >= 65) return "dewpoint-oppressive";
    if (dewPointF >= 55) return "dewpoint-sticky";
    if (dewPointF < 40) return "dewpoint-low";
    return "dewpoint-normal";
}

// Check if hour midpoint is during daytime
// Returns true if the majority of the hour (HH:30) falls between sunrise and sunset
function isHourMidpointDaytime(periodTime, sunrise, sunset, timeZone) {
    if (!sunrise || !sunset) {
        // Fallback to simple time-based heuristic if sunrise/sunset unavailable
        const hour = periodTime.getHours();
        return hour >= 6 && hour < 18;
    }
    
    // Ensure sunrise and sunset are Date objects
    const sunriseDate = sunrise instanceof Date ? sunrise : new Date(sunrise);
    const sunsetDate = sunset instanceof Date ? sunset : new Date(sunset);
    
    // Calculate hour midpoint (HH:30) - use the period's date but set to HH:30
    const hourMidpoint = new Date(periodTime);
    hourMidpoint.setMinutes(30, 0, 0);
    
    // Normalize sunrise/sunset to the same date as the period for accurate comparison
    // This ensures we're comparing times on the same day
    const periodDate = new Date(periodTime.getFullYear(), periodTime.getMonth(), periodTime.getDate());
    const normalizedSunrise = new Date(periodDate);
    normalizedSunrise.setHours(sunriseDate.getHours(), sunriseDate.getMinutes(), sunriseDate.getSeconds(), sunriseDate.getMilliseconds());
    const normalizedSunset = new Date(periodDate);
    normalizedSunset.setHours(sunsetDate.getHours(), sunsetDate.getMinutes(), sunsetDate.getSeconds(), sunsetDate.getMilliseconds());
    
    // Extract time-of-day portions for comparison (minutes since midnight)
    const hourMidpointTime = hourMidpoint.getHours() * 60 + hourMidpoint.getMinutes();
    const sunriseTime = normalizedSunrise.getHours() * 60 + normalizedSunrise.getMinutes();
    const sunsetTime = normalizedSunset.getHours() * 60 + normalizedSunset.getMinutes();
    
    // Handle cases where sunset is the next day (after midnight)
    if (sunsetTime < sunriseTime) {
        // Sunset is the next day, so daytime is from sunrise to midnight OR midnight to sunset
        return (hourMidpointTime >= sunriseTime) || (hourMidpointTime < sunsetTime);
    } else {
        // Normal case: sunset is same day as sunrise
        // Check if hour midpoint is between sunrise and sunset (exclusive of sunset)
        return hourMidpointTime >= sunriseTime && hourMidpointTime < sunsetTime;
    }
}

// Get hour label color class
function getHourLabelColor(periodTime, sunrise, sunset, timeZone) {
    if (isHourMidpointDaytime(periodTime, sunrise, sunset, timeZone)) {
        return "hour-label-daytime";
    }
    return "hour-label-nighttime";
}

// Calculate temperature trend
function calculateTemperatureTrend(currentTemp, nextHourTemp) {
    const tempDiff = nextHourTemp - currentTemp;
    
    if (tempDiff > 0.1) return "rising";
    if (tempDiff < -0.1) return "falling";
    return "steady";
}

// Get trend icon
function getTrendIcon(trend) {
    switch (trend) {
        case "rising": return "↗️";
        case "falling": return "↘️";
        case "steady": return "→";
        default: return "";
    }
}

// Truncate city name to fit within max length
function truncateCityName(cityName, maxLength = 20) {
    if (!cityName || cityName.length <= maxLength) return cityName;
    
    const words = cityName.split(' ');
    let result = words[0];
    
    if (result.length > maxLength) return result;
    
    for (let i = 1; i < words.length; i++) {
        const nextWord = words[i];
        const potentialLength = result.length + 1 + nextWord.length;
        
        if (potentialLength <= maxLength) {
            result += " " + nextWord;
        } else {
            break;
        }
    }
    
    return result;
}

// Format location display name - removes ", US" and formats as "City, ST"
function formatLocationDisplayName(city, state) {
    if (!city) return '';
    
    // If state is "US" or empty, just return city name
    if (!state || state.toUpperCase() === 'US') {
        return city;
    }
    
    // Return "City, ST" format
    return `${city}, ${state}`;
}

// Calculate bearing (direction) from point 1 to point 2 in degrees (0-360)
function calculateBearing(lat1, lon1, lat2, lon2) {
    const lat1Rad = toRadians(lat1);
    const lat2Rad = toRadians(lat2);
    const dLon = toRadians(lon2 - lon1);
    
    const y = Math.sin(dLon) * Math.cos(lat2Rad);
    const x = Math.cos(lat1Rad) * Math.sin(lat2Rad) - 
              Math.sin(lat1Rad) * Math.cos(lat2Rad) * Math.cos(dLon);
    
    let bearing = Math.atan2(y, x);
    bearing = toDegrees(bearing);
    bearing = (bearing + 360) % 360; // Normalize to 0-360
    
    return bearing;
}

