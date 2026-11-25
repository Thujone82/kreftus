console.log("api.js loaded");

const OWM_API_BASE_URL = 'https://api.openweathermap.org/data/3.0/onecall';
const ONEMIN_PROXY_URL = './api/1min.php';

const api = {
    fetchAiData: async (provider, apiKey, locationName, topicQuery, model = null) => {
        if (provider === 'openrouter') {
            return api.fetchAiDataOpenRouter(apiKey, locationName, topicQuery, model);
        } else if (provider === '1min') {
            return api.fetchAiDataOneMin(apiKey, locationName, topicQuery, model);
        } else {
            return api.fetchAiDataGoogle(apiKey, locationName, topicQuery, model);
        }
    },

    fetchAiDataGoogle: async (apiKey, locationName, topicQuery, model = "gemini-3.5-flash") => {
        const modelName = model || "gemini-3.5-flash"; 
        const GEMINI_API_URL = `https://generativelanguage.googleapis.com/v1beta/models/${modelName}:generateContent?key=${apiKey}`;
        
        const promptText = `${locationName}: ${topicQuery}`;
        console.log(`Fetching AI data for: ${promptText} using model ${modelName}`);

        // Build request body
        const requestBody = {
            contents: [{
                parts: [{
                    text: promptText
                }]
            }]
        };

        // Add thinkingConfig for Gemini 3 models
        if (modelName.startsWith("gemini-3")) {
            requestBody.generationConfig = {
                thinkingConfig: {
                    thinkingLevel: "low"
                }
            };
        } else {
            // Only include googleSearch tool for non-Gemini 3 models to avoid 429 errors
            requestBody.tools = [
                {
                    "googleSearch": {}
                }
            ];
        }

        try {
            const response = await fetch(GEMINI_API_URL, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody)
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
        console.log(`Fetching AI data for: ${promptText} using OpenRouter model ${model} with web search grounding`);

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
                    ],
                    plugins: [
                        {
                            id: 'web'
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

    fetchAiDataOneMin: async (apiKey, locationName, topicQuery, model) => {
        if (!model || model === '') {
            throw new Error("1min.AI model is not selected. Please select a model in settings.");
        }

        const now = new Date();
        const todayLong = now.toLocaleDateString('en-US', {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
        const todayIso = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
        const promptText = `Current date: ${todayLong} (${todayIso}). Treat this as today's date when searching the web and reporting current news or events.\n\n${locationName}: ${topicQuery}`;

        console.log(`Fetching AI data for: ${locationName}: ${topicQuery} using 1min.AI model ${model} with web search`);

        try {
            const response = await fetch(`${ONEMIN_PROXY_URL}?action=chat`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'API-KEY': apiKey
                },
                body: JSON.stringify({
                    type: 'UNIFY_CHAT_WITH_AI',
                    model: model,
                    promptObject: {
                        prompt: promptText,
                        settings: {
                            webSearchSettings: {
                                webSearch: true,
                                numOfSite: 3,
                                maxWord: 1000
                            },
                            historySettings: {
                                isMixed: false
                            },
                            withMemories: false
                        }
                    }
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ message: response.statusText }));
                console.error("1min.AI API Error Response:", errorData);
                if (
                    response.status === 401 ||
                    response.status === 403 ||
                    errorData.errorCode === 'INVALID_API_KEY' ||
                    errorData.error?.code === 'UNAUTHORIZED'
                ) {
                    throw new Error("Invalid 1min.AI API Key. Please check your configuration.");
                }
                const detail = errorData.error?.message || errorData.message || '';
                throw new Error(`1min.AI API request failed with status ${response.status}${detail ? `: ${detail}` : ''}`);
            }

            const data = await response.json();
            console.log("1min.AI API Response Data:", data);

            if (data?.aiRecord?.status === 'FAILED') {
                throw new Error("1min.AI request failed to generate a response.");
            }

            const resultObject = data?.aiRecord?.aiRecordDetail?.resultObject;
            if (Array.isArray(resultObject)) {
                const combined = resultObject
                    .filter(item => typeof item === 'string' && item.trim() !== '')
                    .join('\n\n')
                    .trim();
                if (combined) {
                    return combined;
                }
            } else if (typeof resultObject === 'string' && resultObject.trim() !== '') {
                return resultObject.trim();
            }

            console.warn("1min.AI API response did not contain expected text data structure:", data);
            throw new Error("1min.AI API response did not contain usable text content.");
        } catch (error) {
            console.error('Error fetching AI data from 1min.AI:', error);
            throw error;
        }
    },

    fetchOneMinModels: async () => {
        console.log('Fetching 1min.AI UNIFY_CHAT_WITH_AI models via proxy...');

        const providerLabels = {
            perplexity: 'Perplexity (Native Web Search)',
            openai: 'OpenAI',
            'aws-bedrock': 'Anthropic (Claude)',
            anthropic: 'Anthropic (Claude)',
            googleai: 'Google (Gemini)',
            google: 'Google (Gemini)',
            xai: 'xAI (Grok)',
            deepseek: 'DeepSeek',
            alibaba: 'Alibaba (Qwen)',
            mistralai: 'Mistral AI',
            mistral: 'Mistral AI',
            cohere: 'Cohere',
            openrouter: 'OpenRouter',
            replicate: 'Meta / Replicate',
            zai: 'Z.AI (GLM)'
        };

        try {
            const response = await fetch(`${ONEMIN_PROXY_URL}?action=models`, { cache: 'no-store' });
            if (!response.ok) {
                throw new Error(`Failed to fetch 1min.AI models: ${response.status}`);
            }

            const data = await response.json();
            if (!data || !Array.isArray(data.models)) {
                throw new Error('Invalid response format from 1min.AI models API');
            }

            const nowMs = Date.now();
            const seenIds = new Set();
            const models = [];

            for (const model of data.models) {
                if (!model || !model.modelId || model.modelId === 'NOT_APPLICABLE') continue;
                if (model.status && model.status !== 'ACTIVE') continue;

                if (model.deprecationDate) {
                    const depTs = Date.parse(model.deprecationDate);
                    if (!isNaN(depTs) && depTs <= nowMs) continue;
                }

                const idLower = model.modelId.toLowerCase();
                if (idLower.endsWith('-codex') || idLower.endsWith('-code')) continue;

                if (seenIds.has(model.modelId)) continue;
                seenIds.add(model.modelId);

                const providerKey = (model.provider || 'other').toLowerCase();
                const providerLabel = providerLabels[providerKey] || (model.provider ? model.provider : 'Other');
                models.push({
                    id: model.modelId,
                    name: model.name || model.modelId,
                    provider: providerKey,
                    providerLabel: providerLabel,
                    isNativeWebSearch: providerKey === 'perplexity'
                });
            }

            models.sort((a, b) => {
                if (a.isNativeWebSearch && !b.isNativeWebSearch) return -1;
                if (!a.isNativeWebSearch && b.isNativeWebSearch) return 1;
                const providerCmp = a.providerLabel.localeCompare(b.providerLabel);
                if (providerCmp !== 0) return providerCmp;
                return a.name.localeCompare(b.name);
            });

            console.log(`Fetched ${models.length} active web-search-capable 1min.AI models`);
            return models;
        } catch (error) {
            console.error('Error fetching 1min.AI models:', error);
            throw error;
        }
    },

    validateOneMinModel: async (selectedModel) => {
        if (!selectedModel || selectedModel === '') {
            return { isValid: false, availableModels: [] };
        }

        try {
            const models = await api.fetchOneMinModels();
            const modelExists = models.some(model => model.id === selectedModel);
            return { isValid: modelExists, availableModels: models };
        } catch (error) {
            console.error('Error validating 1min.AI model:', error);
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
        const modelName = "gemini-3.5-flash";
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
    },

    validateOneMinApiKey: async (apiKey) => {
        if (!apiKey) return { isValid: false, reason: 'missing' };
        try {
            const response = await fetch(`${ONEMIN_PROXY_URL}?action=validate`, {
                headers: {
                    'API-KEY': apiKey
                },
                cache: 'no-store'
            });
            if (response.ok) {
                return { isValid: true };
            } else if (response.status === 429) {
                console.warn('1min.AI API key validation: Rate limit hit (429).');
                return { isValid: false, reason: 'rate_limit' };
            } else if (response.status === 401 || response.status === 403) {
                console.warn('1min.AI API key validation: Invalid key.');
                return { isValid: false, reason: 'invalid' };
            }
            console.warn(`1min.AI API key validation failed with status: ${response.status}`);
            return { isValid: false, reason: 'invalid' };
        } catch (error) {
            console.error('Error validating 1min.AI API key:', error);
            return { isValid: false, reason: 'network_error' };
        }
    }
};
