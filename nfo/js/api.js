console.log("api.js loaded");
const api = {
    fetchAiData: async (apiKey, locationName, topicQuery) => {
        const modelName = "gemini-2.5-flash-preview-05-20"; // Corrected model name as per Gemini documentation
        const GEMINI_API_URL = `https://generativelanguage.googleapis.com/v1beta/models/${modelName}:generateContent?key=${apiKey}`;
        
        // The prompt structure for Gemini API is typically a "parts" array with "text".
        // We'll combine location and topic query into a single text prompt.
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
                    tools: [
                        {
                            "googleSearch": {}
                        }
                    ]
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                console.error("API Error Response:", errorData);
                let errorMessage = `API request failed with status ${response.status}`;
                if (errorData.error && errorData.error.message) {
                    errorMessage += `: ${errorData.error.message}`;
                }
                // Specific check for API key issues
                if (response.status === 400 && errorData.error && errorData.error.message.toLowerCase().includes("api key not valid")) {
                    throw new Error("Invalid API Key. Please check your configuration.");
                }
                throw new Error(errorMessage);
            }

            const data = await response.json();
            console.log("AI API Response Data:", data);

            // Extract text from the Gemini response
            // Adjust this based on the actual structure of the Gemini API response
            return data.candidates[0].content.parts[0].text;
        } catch (error) {
            console.error('Error fetching AI data:', error);
            throw error; // Re-throw to be caught by the caller
        }
    }
};