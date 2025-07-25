// IndexedDB wrapper for Habit Tracker
class HabitDB {
  constructor() {
    this.dbName = 'habitTrackerDB';
    this.dbVersion = 1;
    this.db = null;
    this.isReady = false;
    this.readyPromise = this.init();
  }

  // Initialize the database
  async init() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, this.dbVersion);

      request.onupgradeneeded = (event) => {
        const db = event.target.result;

        // Create object stores if they don't exist
        if (!db.objectStoreNames.contains('settings')) {
          const settingsStore = db.createObjectStore('settings', { keyPath: 'id' });
        }

        if (!db.objectStoreNames.contains('categories')) {
          const categoriesStore = db.createObjectStore('categories', { keyPath: 'id', autoIncrement: true });
          categoriesStore.createIndex('order', 'order', { unique: false });
        }

        if (!db.objectStoreNames.contains('tasks')) {
          const tasksStore = db.createObjectStore('tasks', { keyPath: 'id', autoIncrement: true });
          tasksStore.createIndex('categoryId', 'categoryId', { unique: false });
          tasksStore.createIndex('order', 'order', { unique: false });
        }

        if (!db.objectStoreNames.contains('completions')) {
          const completionsStore = db.createObjectStore('completions', { keyPath: 'id', autoIncrement: true });
          completionsStore.createIndex('taskId', 'taskId', { unique: false });
          completionsStore.createIndex('date', 'date', { unique: false });
        }

        if (!db.objectStoreNames.contains('streaks')) {
          const streaksStore = db.createObjectStore('streaks', { keyPath: 'id', autoIncrement: true });
          streaksStore.createIndex('date', 'date', { unique: true });
        }
      };

      request.onsuccess = (event) => {
        this.db = event.target.result;
        this.isReady = true;
        resolve();
      };

      request.onerror = (event) => {
        console.error('IndexedDB error:', event.target.error);
        reject(event.target.error);
      };
    });
  }

  // Ensure database is ready before operations
  async ready() {
    if (this.isReady) return Promise.resolve();
    return this.readyPromise;
  }

  // Generic method to perform a transaction
  async transaction(storeName, mode, callback) {
    await this.ready();
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(storeName, mode);
      const store = transaction.objectStore(storeName);

      const request = callback(store);

      if (request) {
        request.onsuccess = (event) => resolve(event.target.result);
        request.onerror = (event) => reject(event.target.error);
      }

      transaction.oncomplete = () => resolve();
      transaction.onerror = (event) => reject(event.target.error);
    });
  }

  // Settings methods
  async getSettings() {
    try {
      const settings = await this.transaction('settings', 'readonly', (store) => {
        return store.get('user-settings');
      });

      if (!settings) {
        // Default settings
        const defaultSettings = {
          id: 'user-settings',
          theme: 'dark',
          completionColor: '#4CAF50',
          currentStreak: 0,
          recordStreak: 0,
          lastCompletionDate: null
        };
        await this.saveSettings(defaultSettings);
        return defaultSettings;
      }

      return settings;
    } catch (error) {
      console.error('Error getting settings:', error);
      throw error;
    }
  }

  async saveSettings(settings) {
    try {
      await this.transaction('settings', 'readwrite', (store) => {
        return store.put(settings);
      });
    } catch (error) {
      console.error('Error saving settings:', error);
      throw error;
    }
  }

  // Category methods
  async getCategories() {
    try {
      return await this.transaction('categories', 'readonly', (store) => {
        const index = store.index('order');
        return index.getAll();
      });
    } catch (error) {
      console.error('Error getting categories:', error);
      throw error;
    }
  }

  async addCategory(category) {
    try {
      // Get the highest order value
      const categories = await this.getCategories();
      const maxOrder = categories.length > 0 
        ? Math.max(...categories.map(c => c.order || 0)) 
        : -1;
      
      category.order = maxOrder + 1;
      
      return await this.transaction('categories', 'readwrite', (store) => {
        return store.add(category);
      });
    } catch (error) {
      console.error('Error adding category:', error);
      throw error;
    }
  }

  async updateCategory(category) {
    try {
      await this.transaction('categories', 'readwrite', (store) => {
        return store.put(category);
      });
    } catch (error) {
      console.error('Error updating category:', error);
      throw error;
    }
  }

  async deleteCategory(id) {
    try {
      // First delete all tasks in this category
      const tasks = await this.getTasksByCategory(id);
      for (const task of tasks) {
        await this.deleteTask(task.id);
      }

      // Then delete the category
      await this.transaction('categories', 'readwrite', (store) => {
        return store.delete(id);
      });
    } catch (error) {
      console.error('Error deleting category:', error);
      throw error;
    }
  }

  async updateCategoryOrder(orderedIds) {
    try {
      for (let i = 0; i < orderedIds.length; i++) {
        await this.transaction('categories', 'readwrite', (store) => {
          const request = store.get(orderedIds[i]);
          request.onsuccess = (event) => {
            const category = event.target.result;
            if (category) {
              category.order = i;
              store.put(category);
            }
          };
        });
      }
    } catch (error) {
      console.error('Error updating category order:', error);
      throw error;
    }
  }

  // Task methods
  async getTasks() {
    try {
      return await this.transaction('tasks', 'readonly', (store) => {
        return store.getAll();
      });
    } catch (error) {
      console.error('Error getting tasks:', error);
      throw error;
    }
  }

  async getTasksByCategory(categoryId) {
    try {
      return await this.transaction('tasks', 'readonly', (store) => {
        const index = store.index('categoryId');
        const request = index.getAll(IDBKeyRange.only(categoryId));
        
        request.onsuccess = (event) => {
          const tasks = event.target.result;
          // Sort by order
          tasks.sort((a, b) => (a.order || 0) - (b.order || 0));
          return tasks;
        };
        
        return request;
      });
    } catch (error) {
      console.error('Error getting tasks by category:', error);
      throw error;
    }
  }

  async addTask(task) {
    try {
      // Get the highest order value for this category
      const tasks = await this.getTasksByCategory(task.categoryId);
      const maxOrder = tasks.length > 0 
        ? Math.max(...tasks.map(t => t.order || 0)) 
        : -1;
      
      task.order = maxOrder + 1;
      
      return await this.transaction('tasks', 'readwrite', (store) => {
        return store.add(task);
      });
    } catch (error) {
      console.error('Error adding task:', error);
      throw error;
    }
  }

  async updateTask(task) {
    try {
      await this.transaction('tasks', 'readwrite', (store) => {
        return store.put(task);
      });
    } catch (error) {
      console.error('Error updating task:', error);
      throw error;
    }
  }

  async deleteTask(id) {
    try {
      // First delete all completions for this task
      await this.transaction('completions', 'readwrite', (store) => {
        const index = store.index('taskId');
        const request = index.openCursor(IDBKeyRange.only(id));
        
        request.onsuccess = (event) => {
          const cursor = event.target.result;
          if (cursor) {
            cursor.delete();
            cursor.continue();
          }
        };
      });

      // Then delete the task
      await this.transaction('tasks', 'readwrite', (store) => {
        return store.delete(id);
      });
    } catch (error) {
      console.error('Error deleting task:', error);
      throw error;
    }
  }

  async updateTaskOrder(categoryId, orderedIds) {
    try {
      for (let i = 0; i < orderedIds.length; i++) {
        await this.transaction('tasks', 'readwrite', (store) => {
          const request = store.get(orderedIds[i]);
          request.onsuccess = (event) => {
            const task = event.target.result;
            if (task) {
              task.order = i;
              store.put(task);
            }
          };
        });
      }
    } catch (error) {
      console.error('Error updating task order:', error);
      throw error;
    }
  }

  // Completion methods
  async getCompletionsForDate(date) {
    try {
      const dateStr = new Date(date).toISOString().split('T')[0];
      return await this.transaction('completions', 'readonly', (store) => {
        const index = store.index('date');
        return index.getAll(IDBKeyRange.only(dateStr));
      });
    } catch (error) {
      console.error('Error getting completions for date:', error);
      throw error;
    }
  }

  async getCompletionForTask(taskId, date) {
    try {
      const dateStr = new Date(date).toISOString().split('T')[0];
      const completions = await this.transaction('completions', 'readonly', (store) => {
        const index = store.index('taskId');
        return index.getAll(IDBKeyRange.only(taskId));
      });
      
      return completions.find(c => c.date === dateStr);
    } catch (error) {
      console.error('Error getting completion for task:', error);
      throw error;
    }
  }

  async addCompletion(taskId, date = new Date()) {
    try {
      const dateStr = new Date(date).toISOString().split('T')[0];
      
      // Check if completion already exists
      const existing = await this.getCompletionForTask(taskId, date);
      if (existing) return existing.id;
      
      const completion = {
        taskId,
        date: dateStr,
        timestamp: new Date().toISOString()
      };
      
      return await this.transaction('completions', 'readwrite', (store) => {
        return store.add(completion);
      });
    } catch (error) {
      console.error('Error adding completion:', error);
      throw error;
    }
  }

  async removeCompletion(taskId, date = new Date()) {
    try {
      const completion = await this.getCompletionForTask(taskId, date);
      if (!completion) return;
      
      await this.transaction('completions', 'readwrite', (store) => {
        return store.delete(completion.id);
      });
    } catch (error) {
      console.error('Error removing completion:', error);
      throw error;
    }
  }

  // Streak methods
  async updateStreak() {
    try {
      const settings = await this.getSettings();
      const today = new Date();
      const todayStr = today.toISOString().split('T')[0];
      
      // Get all tasks and categories
      const categories = await this.getCategories();
      const tasks = await this.getTasks();
      
      // Get completions for today
      const completions = await this.getCompletionsForDate(today);
      const completedTaskIds = completions.map(c => c.taskId);
      
      // Check if all tasks are completed
      const allCompleted = tasks.every(task => completedTaskIds.includes(task.id));
      
      if (allCompleted && tasks.length > 0) {
        // All tasks completed for today
        const lastCompletionDate = settings.lastCompletionDate 
          ? new Date(settings.lastCompletionDate) 
          : null;
        
        if (lastCompletionDate) {
          // Check if the last completion was yesterday
          const yesterday = new Date(today);
          yesterday.setDate(yesterday.getDate() - 1);
          const yesterdayStr = yesterday.toISOString().split('T')[0];
          
          if (lastCompletionDate.toISOString().split('T')[0] === yesterdayStr) {
            // Continuing the streak
            settings.currentStreak++;
          } else if (lastCompletionDate.toISOString().split('T')[0] !== todayStr) {
            // Broke the streak, starting a new one
            settings.currentStreak = 1;
          }
        } else {
          // First completion
          settings.currentStreak = 1;
        }
        
        // Update record streak if needed
        if (settings.currentStreak > settings.recordStreak) {
          settings.recordStreak = settings.currentStreak;
        }
        
        settings.lastCompletionDate = todayStr;
        
        // Save updated settings
        await this.saveSettings(settings);
        
        // Record the streak
        await this.transaction('streaks', 'readwrite', (store) => {
          return store.put({
            date: todayStr,
            streak: settings.currentStreak,
            isRecord: settings.currentStreak === settings.recordStreak
          });
        });
        
        return {
          currentStreak: settings.currentStreak,
          recordStreak: settings.recordStreak,
          isRecord: settings.currentStreak === settings.recordStreak
        };
      }
      
      return {
        currentStreak: settings.currentStreak,
        recordStreak: settings.recordStreak,
        isRecord: false
      };
    } catch (error) {
      console.error('Error updating streak:', error);
      throw error;
    }
  }

  async getStreakInfo() {
    try {
      const settings = await this.getSettings();
      return {
        currentStreak: settings.currentStreak,
        recordStreak: settings.recordStreak
      };
    } catch (error) {
      console.error('Error getting streak info:', error);
      throw error;
    }
  }
}

// Create and export a single instance
const habitDB = new HabitDB();