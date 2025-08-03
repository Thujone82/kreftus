# Gemini Project File

## Project: nfo2Go v.2

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 2.0

---

### Description

`nfo2Go` is a highly configurable, mobile-first Progressive Web App (PWA) designed to provide customized, AI-generated information briefings for specified locations. It acts as a personal dashboard where users can define multiple locations and a flexible structure of "topics" to query. For each location, the app uses the Google Gemini API to generate reports based on user-defined prompts (e.g., "Top 5 local news stories") and the OpenWeatherMap API to fetch current weather conditions.

The application is built with vanilla JavaScript, HTML, and CSS, and it leverages modern browser features like IndexedDB for persistent storage and Service Workers for offline capabilities and app updates.

### Key Functionality

-   **Dynamic Information Structure:** Users can define a list of "topics," each with a custom prompt for the Gemini AI. This allows for a fully personalized information feed for any location.
-   **Customizable Locations:** Users can add, remove, and reorder a list of geographic locations (e.g., "Portland, OR", "90210").
-   **Dual API Integration:**
    -   **Google Gemini:** Powers the core information retrieval, answering user-defined queries for each topic.
    -   **OpenWeatherMap:** Fetches and displays real-time weather data for the selected location.
-   **Offline First (PWA):** As a Progressive Web App, it can be "installed" on a home screen. The service worker enables offline access by serving cached data, displaying an "OFFLINE CACHE VIEW" when the network is unavailable.
-   **Configuration Management:** An extensive in-app configuration menu allows users to:
    -   Securely enter and validate API keys for both Gemini and OpenWeatherMap.
    -   Set a custom Rate-Per-Minute (RPM) limit for the Gemini API to manage usage.
    -   Customize the application's primary and background colors.
-   **Data Persistence:** All configurations (API keys, locations, topics, colors) are stored locally in the browser's IndexedDB via the `stor.js` module.
-   **Manual & Automatic Refresh:** Users can manually refresh data for a specific location or use a global "Refresh All Stale" button to update all locations with outdated caches.

### How to Run

The application is a static web app. To run it, simply open the `index.html` file in a modern web browser that supports Service Workers and IndexedDB (e.g., Chrome, Firefox, Edge, Safari).

For full functionality, API keys must be configured in the "App" settings menu.

### Dependencies

-   A modern web browser.
-   An active internet connection (for initial data fetching and updates).
-   A free API key from [Google AI Studio (Gemini)](https://aistudio.google.com/app/apikey).
-   A free API key from [OpenWeatherMap](https://home.openweathermap.org/api_keys) (with billing info added to enable the One Call API, though it remains free under the 1,000 calls/day limit).

### File Structure

-   `index.html`: The main entry point and structure of the application.
-   `manifest.json`: PWA manifest for installation and app metadata.
-   `sw.js`: Service worker script for offline caching and app updates.
-   `css/style.css`: All styling for the application.
-   `icons/`: Contains the application icons for the PWA manifest and favicon.
-   `js/`:
    -   `app.js`: Main application logic, event listeners, and initialization.
    -   `api.js`: Handles all API requests to Gemini and OpenWeatherMap.
    -   `stor.js`: A module for managing all interactions with the browser's IndexedDB.
    -   `ui.js`: Controls all UI manipulations, including modals, button states, and dynamic content rendering.
    -   `utils.js`: Contains helper and utility functions used across the application.
