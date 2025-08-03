# Gemini Project File

## Project: Habit Tracker

**Author:** Kreft&Gemini[Gemini 2.5 Pro (preview)]
**Date:** 2025-08-02
**Version:** 1.0

---

### Description

Habit Tracker is a client-side, mobile-first Progressive Web App (PWA) designed to help users build and maintain positive habits. The application allows for the creation of tasks organized into customizable categories. It provides visual feedback on daily progress and tracks the user's completion streak to encourage consistency.

Built with vanilla JavaScript, HTML, and CSS, the app is fully self-contained and runs entirely in the browser. It uses IndexedDB for all data storage, ensuring that user data is persistent and private. The interface is highly interactive, featuring drag-and-drop for reordering tasks and categories.

### Key Functionality

-   **Categorized Task Management:** Users can create custom categories (e.g., "Morning Routine," "Fitness") and add specific tasks to each.
-   **Daily Progress Tracking:** The main screen displays progress bars for each category and an overall completion bar for the day. Tasks are marked as complete with a single tap.
-   **Streak System:** The app tracks the current consecutive-day completion streak and the all-time record streak, providing motivation to maintain consistency. A notification appears when a new record is set.
-   **Interactive Configuration:** A comprehensive setup menu allows users to:
    -   Add, edit, delete, and reorder categories.
    -   Add, edit, delete, and reorder tasks within each category.
-   **Customizable Themes:** Users can switch between a light and dark mode and select a custom color for completed tasks.
-   **Drag-and-Drop Interface:** Both categories on the main screen and items in the setup menu can be reordered via drag-and-drop.
-   **Offline First (PWA):** As a PWA, the app can be installed on a device's home screen and works offline. All data is stored locally using IndexedDB.
-   **First-Time User Onboarding:** A welcome message guides new users to the setup screen to configure their initial set of habits.

### How to Run

The application is a static web app. To run it, open the `index.html` file in a modern web browser that supports IndexedDB (e.g., Chrome, Firefox, Edge, Safari).

### Dependencies

-   A modern web browser.

### File Structure

-   `index.html`: The main HTML file containing the structure of the app and templates for dynamic elements.
-   `styles.css`: The stylesheet for the application.
-   `manifest.json`: The PWA manifest file.
-   `service-worker.js`: The service worker for offline capabilities.
-   `app.js`: The core application logic, handling UI, events, and state management.
-   `db.js`: A dedicated module for all IndexedDB operations (saving, loading, updating data).
-   `icons/`: Directory containing the PWA icons.
