# Gemini Project File

## Project: nfo2Go v.3

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-01-25
**Version:** 3.0

---

### Description

`nfo2Go` is a highly configurable, mobile-first Progressive Web App (PWA) designed to provide customized, AI-generated information briefings for specified locations. It acts as a personal dashboard where users can define multiple locations and a flexible structure of "topics" to query. For each location, the app uses AI APIs to generate reports based on user-defined prompts (e.g., "Top 5 local news stories") and the OpenWeatherMap API to fetch current weather conditions.

The application supports multiple AI providers:
- **Google Gemini API:** The default provider, offering direct access to Google's Gemini models
- **OpenRouter API:** An alternative provider that provides access to hundreds of AI models through a unified interface, including free tier options

The application is built with vanilla JavaScript, HTML, and CSS, and it leverages modern browser features like localStorage for persistent storage and Service Workers for offline capabilities and app updates.

### Key Functionality

-   **Dynamic Information Structure:** Users can define a list of "topics," each with a custom prompt for the selected AI provider. This allows for a fully personalized information feed for any location.
-   **Customizable Locations:** Users can add, remove, and reorder a list of geographic locations (e.g., "Portland, OR", "90210").
-   **Multi-Provider AI Integration:**
    -   **Google Gemini:** The default AI provider, offering direct access to Google's Gemini models with Google Search integration.
    -   **OpenRouter:** An alternative AI provider that provides access to hundreds of AI models through a unified API, including free tier models. Features dynamic model selection with free models highlighted and sorted first.
    -   **OpenWeatherMap:** Fetches and displays real-time weather data for the selected location.
-   **Offline First (PWA):** As a Progressive Web App, it can be "installed" on a home screen. The service worker enables offline access by serving cached data, displaying an "OFFLINE CACHE VIEW" when the network is unavailable.
-   **Configuration Management:** An extensive in-app configuration menu allows users to:
    -   Select AI provider (Google Gemini or OpenRouter) with a dropdown interface.
    -   Securely enter and validate API keys for the selected AI provider and OpenWeatherMap (stored separately for each provider).
    -   For OpenRouter: Dynamically fetch and select from available AI models, with free models highlighted in green and sorted first.
    -   Set custom Rate-Per-Minute (RPM) limits separately for each AI provider to manage usage.
    -   Customize the application's primary and background colors.
    -   Model validation on app launch ensures selected models are still available.
-   **Data Persistence:** All configurations (API keys, provider settings, model selections, locations, topics, colors) are stored locally in the browser's localStorage via the `stor.js` module. Settings are maintained separately for each provider to allow easy switching.
-   **Manual & Automatic Refresh:** Users can manually refresh data for a specific location or use a global "Refresh All Stale" button to update all locations with outdated caches.

### How to Run

The application is a static web app. To run it, simply open the `index.html` file in a modern web browser that supports Service Workers and localStorage (e.g., Chrome, Firefox, Edge, Safari).

For full functionality, API keys must be configured in the "App" settings menu.

### Dependencies

-   A modern web browser.
-   An active internet connection (for initial data fetching and updates).
-   **AI Provider API Key (choose one):**
    -   A free API key from [Google AI Studio (Gemini)](https://aistudio.google.com/app/apikey) for Google Gemini provider.
    -   OR an API key from [OpenRouter](https://openrouter.ai/keys) for OpenRouter provider (supports free tier models).
-   A free API key from [OpenWeatherMap](https://home.openweathermap.org/api_keys) (with billing info added to enable the One Call API, though it remains free under the 1,000 calls/day limit).

### File Structure

-   `index.html`: The main entry point and structure of the application.
-   `manifest.json`: PWA manifest for installation and app metadata.
-   `sw.js`: Service worker script for offline caching and app updates.
-   `css/style.css`: All styling for the application.
-   `icons/`: Contains the application icons for the PWA manifest and favicon.
-   `js/`:
    -   `app.js`: Main application logic, event listeners, and initialization.
    -   `api.js`: Handles all API requests to AI providers (Google Gemini and OpenRouter) and OpenWeatherMap.
    -   `stor.js`: A module for managing all interactions with the browser's localStorage for persistent data storage.
    -   `ui.js`: Controls all UI manipulations, including modals, button states, and dynamic content rendering.
    -   `utils.js`: Contains helper and utility functions used across the application.
