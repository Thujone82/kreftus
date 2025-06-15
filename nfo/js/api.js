console.log("api.js loaded");

const OWM_API_BASE_URL = 'https://api.openweathermap.org/data/3.0/onecall';

const api = {
    fetchAiData: async (apiKey, locationName, topicQuery) => {
        const modelName = "gemini-1.5-flash-preview-0514"; // Updated model name
        const GEMINI_API_URL = `https://generativelanguage.googleapis.com/v1beta/models/${modelName}:generateContent?key=${apiKey}`;
        
        const promptText = `${locationName}: ${topicQuery}`;
        console.log(`Fetching AI data for: ${promptText} using model ${modelName}`);

        try {
            const response = await fetch(GEMINI_API_URL, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    contents: [{
                        parts: [{
                            text: promptText
                        }]
                    }],
                    // "tools": [ // Consider if Google Search tool is always needed or configurable
                    //     {
                    //         "googleSearch": {}
                    //     }
                    // ]
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ error: { message: response.statusText } })); // Graceful error parsing
                console.error("API Error Response:", errorData);
                let errorMessage = `API request failed with status ${response.status}`;
                if (errorData.error && errorData.error.message) {
                    errorMessage += `: ${errorData.error.message}`;
                }
                if (response.status === 400 && errorData.error && errorData.error.message.toLowerCase().includes("api key not valid")) {
                    throw new Error("Invalid API Key. Please check your configuration.");
                }
                throw new Error(errorMessage);
            }

            const data = await response.json();
            console.log("AI API Response Data:", data);

            if (data.candidates && data.candidates.length > 0 &&
                data.candidates[0].content && data.candidates[0].content.parts &&
                data.candidates[0].content.parts.length > 0 && data.candidates[0].content.parts[0].text) {
                return data.candidates[0].content.parts[0].text;
            } else {
                console.warn("AI API response did not contain expected text data structure:", data);
                // Check for safety ratings or other reasons for empty content
                if (data.candidates && data.candidates[0] && data.candidates[0].finishReason) {
                    throw new Error(`AI generation finished with reason: ${data.candidates[0].finishReason}. No content available.`);
                }
                throw new Error("AI response did not contain usable text content.");
            }
        } catch (error) {
            console.error('Error fetching AI data:', error);
            throw error;
        }
    },

    /**
     * Fetches current weather data from OpenWeatherMap One Call API.
     * @param {number} lat Latitude of the location.
     * @param {number} lon Longitude of the location.
     * @param {string} apiKey Your OpenWeatherMap API key.
     * @returns {Promise<Object|null>} A promise that resolves to the 'current' weather data object or null if an error occurs.
     */
    fetchWeatherData: async (lat, lon, apiKey) => {
        if (!apiKey) {
            console.error('OpenWeatherMap API key is missing. Cannot fetch weather data.');
            // Consider returning a specific error object or status
            return null;
        }
        const units = 'imperial'; // For Fahrenheit
        const exclude = 'minutely,hourly,daily,alerts'; // Exclude data we don't need
        const url = `${OWM_API_BASE_URL}?lat=${lat}&lon=${lon}&appid=${apiKey}&units=${units}&exclude=${exclude}`;

        console.log(`Fetching weather data from: ${url}`);

        try {
            const response = await fetch(url);
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ message: response.statusText }));
                console.error(`Error fetching weather data: ${response.status}`, errorData);
                // You might want to throw a more specific error or return a structured error
                return null;
            }
            const data = await response.json();
            console.log("OpenWeatherMap API Response Data:", data);
            return data.current; // We are interested in the 'current' weather block.
        } catch (error) {
            console.error('Network error or other issue fetching weather data:', error);
            return null;
        }
    }
};
