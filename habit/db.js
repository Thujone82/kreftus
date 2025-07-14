// IndexedDB wrapper for habit tracking
class HabitDB {
    constructor() {
        this.dbName = 'HabitTrackerDB';
        this.version = 1;
        this.db = null;
    }

    async init() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(this.dbName, this.version);

            request.onerror = () => reject(request.error);
            request.onsuccess = () => {
                this.db = request.result;
                resolve();
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;

                // Categories store
                if (!db.objectStoreNames.contains('categories')) {
                    const categoriesStore = db.createObjectStore('categories', { keyPath: 'id' });
                    categoriesStore.createIndex('order', 'order', { unique: false });
                }

                // Tasks store
                if (!db.objectStoreNames.contains('tasks')) {
                    const tasksStore = db.createObjectStore('tasks', { keyPath: 'id' });
                    tasksStore.createIndex('categoryId', 'categoryId', { unique: false });
                    tasksStore.createIndex('order', 'order', { unique: false });
                }

                // Completions store (daily task completions)
                if (!db.objectStoreNames.contains('completions')) {
                    const completionsStore = db.createObjectStore('completions', { keyPath: 'id' });
                    completionsStore.createIndex('date', 'date', { unique: false });
                    completionsStore.createIndex('taskId', 'taskId', { unique: false });
                }

                // Settings store
                if (!db.objectStoreNames.contains('settings')) {
                    db.createObjectStore('settings', { keyPath: 'key' });
                }

                // Streaks store
                if (!db.objectStoreNames.contains('streaks')) {
                    db.createObjectStore('streaks', { keyPath: 'id' });
                }
            };
        });
    }

    async getSettings() {
        const transaction = this.db.transaction(['settings'], 'readonly');
        const store = transaction.objectStore('settings');
        
        const theme = await this.getFromStore(store, 'theme') || 'dark';
        const completionColor = await this.getFromStore(store, 'completionColor') || '#22c55e';
        
        return { theme, completionColor };
    }

    async saveSetting(key, value) {
        const transaction = this.db.transaction(['settings'], 'readwrite');
        const store = transaction.objectStore('settings');
        await store.put({ key, value });
    }

    async getCategories() {
        const transaction = this.db.transaction(['categories'], 'readonly');
        const store = transaction.objectStore('categories');
        const index = store.index('order');
        
        return new Promise((resolve, reject) => {
            const request = index.getAll();
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    async saveCategory(category) {
        const transaction = this.db.transaction(['categories'], 'readwrite');
        const store = transaction.objectStore('categories');
        await store.put(category);
    }

    async deleteCategory(categoryId) {
        const transaction = this.db.transaction(['categories', 'tasks'], 'readwrite');
        const categoriesStore = transaction.objectStore('categories');
        const tasksStore = transaction.objectStore('tasks');
        
        // Delete category
        await categoriesStore.delete(categoryId);
        
        // Delete all tasks in category
        const tasks = await this.getTasksByCategory(categoryId);
        for (const task of tasks) {
            await tasksStore.delete(task.id);
        }
    }

    async getTasks() {
        const transaction = this.db.transaction(['tasks'], 'readonly');
        const store = transaction.objectStore('tasks');
        
        return new Promise((resolve, reject) => {
            const request = store.getAll();
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    async getTasksByCategory(categoryId) {
        const transaction = this.db.transaction(['tasks'], 'readonly');
        const store = transaction.objectStore('tasks');
        const index = store.index('categoryId');
        
        return new Promise((resolve, reject) => {
            const request = index.getAll(categoryId);
            request.onsuccess = () => {
                const tasks = request.result;
                tasks.sort((a, b) => a.order - b.order);
                resolve(tasks);
            };
            request.onerror = () => reject(request.error);
        });
    }

    async saveTask(task) {
        const transaction = this.db.transaction(['tasks'], 'readwrite');
        const store = transaction.objectStore('tasks');
        await store.put(task);
    }

    async deleteTask(taskId) {
        const transaction = this.db.transaction(['tasks'], 'readwrite');
        const store = transaction.objectStore('tasks');
        await store.delete(taskId);
    }

    async getCompletions(date) {
        const transaction = this.db.transaction(['completions'], 'readonly');
        const store = transaction.objectStore('completions');
        const index = store.index('date');
        
        return new Promise((resolve, reject) => {
            const request = index.getAll(date);
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    async saveCompletion(completion) {
        const transaction = this.db.transaction(['completions'], 'readwrite');
        const store = transaction.objectStore('completions');
        await store.put(completion);
    }

    async deleteCompletion(completionId) {
        const transaction = this.db.transaction(['completions'], 'readwrite');
        const store = transaction.objectStore('completions');
        await store.delete(completionId);
    }

    async getStreaks() {
        const transaction = this.db.transaction(['streaks'], 'readonly');
        const store = transaction.objectStore('streaks');
        
        return new Promise((resolve, reject) => {
            const request = store.get('current');
            request.onsuccess = () => {
                const result = request.result || { id: 'current', current: 0, record: 0 };
                resolve(result);
            };
            request.onerror = () => reject(request.error);
        });
    }

    async saveStreaks(streaks) {
        const transaction = this.db.transaction(['streaks'], 'readwrite');
        const store = transaction.objectStore('streaks');
        await store.put({ id: 'current', ...streaks });
    }

    async getFromStore(store, key) {
        return new Promise((resolve, reject) => {
            const request = store.get(key);
            request.onsuccess = () => resolve(request.result?.value);
            request.onerror = () => reject(request.error);
        });
    }

    generateId() {
        return Date.now().toString(36) + Math.random().toString(36).substr(2);
    }

    getTodayString() {
        return new Date().toISOString().split('T')[0];
    }
}