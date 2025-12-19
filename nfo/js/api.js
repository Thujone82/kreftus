console.log("api.js loaded");

const OWM_API_BASE_URL = 'https://api.openweathermap.org/data/3.0/onecall';

const api = {
    fetchAiData: async (provider, apiKey, locationName, topicQuery, model = null) => {
        if (provider === 'openrouter') {
            return api.fetchAiDataOpenRouter(apiKey, locationName, topicQuery, model);
        } else {
            return api.fetchAiDataGoogle(apiKey, locationName, topicQuery);
        }
    },

    fetchAiDataGoogle: async (apiKey, locationName, topicQuery) => {
        const modelName = "gemini-2.5-flash"; 
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
                    "tools": [
                        {
                            "googleSearch": {}
                        }
                    ]
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

    fetchAiDataOpenRouter: async (apiKey, locationName, topicQuery, model) => {
        if (!model || model === '') {
            throw new Error("OpenRouter model is not selected. Please select a model in settings.");
        }
        
        const OPENROUTER_API_URL = 'https://openrouter.ai/api/v1/chat/completions';
        const promptText = `${locationName}: ${topicQuery}`;
        console.log(`Fetching AI data for: ${promptText} using OpenRouter model ${model}`);

        try {
            const response = await fetch(OPENROUTER_API_URL, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${apiKey}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    model: model,
                    messages: [
                        {
                            role: 'user',
                            content: promptText
                        }
                    ]
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ error: { message: response.statusText } }));
                console.error("OpenRouter API Error Response:", errorData);
                let errorMessage = `OpenRouter API request failed with status ${response.status}`;
                if (errorData.error && errorData.error.message) {
                    errorMessage += `: ${errorData.error.message}`;
                }
                if (response.status === 401 || response.status === 403) {
                    throw new Error("Invalid OpenRouter API Key. Please check your configuration.");
                }
                throw new Error(errorMessage);
            }

            const data = await response.json();
            console.log("OpenRouter API Response Data:", data);

            if (data.choices && data.choices.length > 0 &&
                data.choices[0].message && data.choices[0].message.content) {
                return data.choices[0].message.content;
            } else {
                console.warn("OpenRouter API response did not contain expected text data structure:", data);
                throw new Error("OpenRouter API response did not contain usable text content.");
            }
        } catch (error) {
            console.error('Error fetching AI data from OpenRouter:', error);
            throw error;
        }
    },

    fetchOpenRouterModels: async () => {
        const OPENROUTER_MODELS_URL = 'https://openrouter.ai/api/v1/models';
        console.log('Fetching OpenRouter models from API...');

        try {
            const response = await fetch(OPENROUTER_MODELS_URL);
            if (!response.ok) {
                throw new Error(`Failed to fetch models: ${response.status}`);
            }

            const data = await response.json();
            if (!data.data || !Array.isArray(data.data)) {
                throw new Error('Invalid response format from OpenRouter models API');
            }

            // Process and sort models
            const models = data.data.map(model => {
                const isFree = model.id.endsWith(':free') || model.pricing?.prompt === '0';
                return {
                    id: model.id,
                    name: model.name || model.id,
                    isFree: isFree
                };
            });

            // Sort: free models first, then alphabetically
            models.sort((a, b) => {
                if (a.isFree && !b.isFree) return -1;
                if (!a.isFree && b.isFree) return 1;
                return a.name.localeCompare(b.name);
            });

            console.log(`Fetched ${models.length} OpenRouter models (${models.filter(m => m.isFree).length} free)`);
            return models;
        } catch (error) {
            console.error('Error fetching OpenRouter models:', error);
            throw error;
        }
    },

    validateOpenRouterModel: async (selectedModel) => {
        if (!selectedModel || selectedModel === '') {
            return { isValid: false, availableModels: [] };
        }

        try {
            const models = await api.fetchOpenRouterModels();
            const modelExists = models.some(model => model.id === selectedModel);
            return { isValid: modelExists, availableModels: models };
        } catch (error) {
            console.error('Error validating OpenRouter model:', error);
            return { isValid: false, availableModels: [] };
        }
    },

    /**
     * Fetches current weather data from OpenWeatherMap One Call API
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
    },

    validateGeminiApiKey: async (apiKey) => {
        if (!apiKey) return false;
        // Use a lightweight call, like listing models, to validate the key.
        // Using a specific model known to exist to ensure the endpoint is valid for a key check.
        const modelName = "gemini-2.5-flash";
        const VALIDATE_URL = `https://generativelanguage.googleapis.com/v1beta/models/${modelName}?key=${apiKey}`;
        try {
            const response = await fetch(VALIDATE_URL);
            // A 200 OK response, even with an empty list or specific models, indicates the key is valid for access.
            // A 400 or 403 would indicate an invalid key or permission issue.
            if (response.ok) {
                return { isValid: true };
            } else if (response.status === 429) {
                console.warn('Gemini API key validation: Rate limit hit (429).');
                return { isValid: false, reason: 'rate_limit' };
            }
            // For other errors (400, 401, 403, etc.)
            console.warn(`Gemini API key validation failed with status: ${response.status}`);
            return { isValid: false, reason: 'invalid' };
        } catch (error) {
            console.error('Error validating Gemini API key:', error);
            return { isValid: false, reason: 'network_error' };
        }
    },

    validateOwmApiKey: async (apiKey) => {
        if (!apiKey) return false;
        // Make a minimal call to a protected endpoint.
        // Using current weather for a fixed coordinate (e.g., London) just to check the key.
        // We don't need to process the weather data itself.
        const VALIDATE_URL = `${OWM_API_BASE_URL}?lat=51.5074&lon=0.1278&appid=${apiKey}&cnt=1&exclude=minutely,hourly,daily,alerts`;
        try {
            const response = await fetch(VALIDATE_URL);
            // OWM returns 401 for invalid API key.
            if (response.ok) {
                return { isValid: true };
            } else if (response.status === 429) {
                console.warn('OpenWeatherMap API key validation: Rate limit hit (429).');
                return { isValid: false, reason: 'rate_limit' };
            }
            // For other errors (e.g., 401 for invalid key)
            console.warn(`OpenWeatherMap API key validation failed with status: ${response.status}`);
            return { isValid: false, reason: 'invalid' };
        } catch (error) {
            console.error('Error validating OpenWeatherMap API key:', error);
            return { isValid: false, reason: 'network_error' };
        }
    },

    validateOpenRouterApiKey: async (apiKey) => {
        if (!apiKey) return { isValid: false, reason: 'missing' };
        // Use a lightweight call to validate the key - fetch models endpoint
        const VALIDATE_URL = 'https://openrouter.ai/api/v1/models';
        try {
            const response = await fetch(VALIDATE_URL, {
                headers: {
                    'Authorization': `Bearer ${apiKey}`
                }
            });
            if (response.ok) {
                return { isValid: true };
            } else if (response.status === 429) {
                console.warn('OpenRouter API key validation: Rate limit hit (429).');
                return { isValid: false, reason: 'rate_limit' };
            } else if (response.status === 401 || response.status === 403) {
                console.warn('OpenRouter API key validation: Invalid key.');
                return { isValid: false, reason: 'invalid' };
            }
            console.warn(`OpenRouter API key validation failed with status: ${response.status}`);
            return { isValid: false, reason: 'invalid' };
        } catch (error) {
            console.error('Error validating OpenRouter API key:', error);
            return { isValid: false, reason: 'network_error' };
        }
    }
};
