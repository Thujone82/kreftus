// IndexedDB wrapper for Habit Tracker

// Helper to get YYYY-MM-DD from a Date object in local time
function getLocalISODateString(date) {
  const year = date.getFullYear();
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');
  return `${year}-${month}-${day}`;
}
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
      await this.transaction('categories', 'readwrite', (store) => {
        orderedIds.forEach((id, i) => {
          const request = store.get(id);
          request.onsuccess = (event) => {
            const category = event.target.result;
            if (category) {
              category.order = i;
              store.put(category);
            }
          };
        });
      });
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
      const tasks = await this.transaction('tasks', 'readonly', (store) => {
        const index = store.index('categoryId');
        return index.getAll(IDBKeyRange.only(categoryId));
      });

      // Sort tasks by order after they are retrieved
      if (tasks) {
        tasks.sort((a, b) => (a.order || 0) - (b.order || 0));
      }

      return tasks;
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
      
      const result = await this.transaction('tasks', 'readwrite', (store) => {
        return store.add(task);
      });
      
      // Reset streak when a new task is added
      await this.resetStreakForToday();
      
      return result;
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

      // After deleting, check if today is now complete
      await this.updateStreak();
    } catch (error) {
      console.error('Error deleting task:', error);
      throw error;
    }
  }

  async updateTaskOrder(categoryId, orderedIds) {
    try {
      await this.transaction('tasks', 'readwrite', (store) => {
        orderedIds.forEach((id, i) => {
          const request = store.get(id);
          request.onsuccess = (event) => {
            const task = event.target.result;
            if (task) {
              task.order = i;
              store.put(task);
            }
          };
        });
      });
    } catch (error) {
      console.error('Error updating task order:', error);
      throw error;
    }
  }

  // Completion methods
  async getCompletionsForDate(date) {
    try {
      const dateStr = getLocalISODateString(new Date(date));
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
      const dateStr = getLocalISODateString(new Date(date));
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
      const dateStr = getLocalISODateString(new Date(date));
      
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
      
      // Reset streak when a completion is removed for today
      await this.resetStreakForToday();
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
      const todayStr = getLocalISODateString(today);
      
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
        const lastCompletionDateStr = settings.lastCompletionDate;
        
        if (lastCompletionDateStr) {
          // Check if the last completion was yesterday
          const yesterday = new Date(today);
          yesterday.setDate(yesterday.getDate() - 1);
          const yesterdayStr = getLocalISODateString(yesterday);
          
          if (lastCompletionDateStr === yesterdayStr) {
            // Continuing the streak
            settings.currentStreak++;
          } else if (lastCompletionDateStr !== todayStr) {
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
        // Check if streak record already exists for today
        const existingStreak = await this.transaction('streaks', 'readonly', (store) => {
          const index = store.index('date');
          return index.get(todayStr);
        });
        
        // Record the streak (update existing or create new)
        await this.transaction('streaks', 'readwrite', (store) => {
          const streakData = {
            date: todayStr,
            streak: settings.currentStreak,
            isRecord: settings.currentStreak === settings.recordStreak
          };
          
          if (existingStreak) {
            // Update existing record
            streakData.id = existingStreak.id;
            return store.put(streakData);
          } else {
            // Create new record
            return store.add(streakData);
          }
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

  async verifyAndResetStreak() {
    try {
      const settings = await this.getSettings();
      if (!settings.lastCompletionDate || settings.currentStreak === 0) {
        return; // Nothing to do if there's no streak
      }

      const today = new Date();
      const yesterday = new Date();
      yesterday.setDate(today.getDate() - 1);

      const todayStr = getLocalISODateString(today);
      const yesterdayStr = getLocalISODateString(yesterday);
      const lastCompletionStr = settings.lastCompletionDate;

      // If the last completion was not today and not yesterday, the streak is broken.
      if (lastCompletionStr !== todayStr && lastCompletionStr !== yesterdayStr) {
        settings.currentStreak = 0;
        await this.saveSettings(settings);
      }
    } catch (error) {
      console.error('Error verifying streak:', error);
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

  // Reset streak for today when tasks are unmarked or new tasks added
  async resetStreakForToday() {
    try {
      const settings = await this.getSettings();
      const today = new Date();
      const todayStr = getLocalISODateString(today);
      
      // Only reset if today was the last completion date
      if (settings.lastCompletionDate === todayStr) {
        // Check if all tasks are still completed
        const categories = await this.getCategories();
        const tasks = await this.getTasks();
        const completions = await this.getCompletionsForDate(today);
        const completedTaskIds = completions.map(c => c.taskId);
        
        const allCompleted = tasks.every(task => completedTaskIds.includes(task.id));
        
        if (!allCompleted || tasks.length === 0) {
          // Recalculate streak from scratch
          await this.recalculateStreak();
        }
      }
    } catch (error) {
      console.error('Error resetting streak for today:', error);
      throw error;
    }
  }

  // Recalculate streak from scratch by checking all completion days
  async recalculateStreak() {
    try {
      const settings = await this.getSettings();
      const tasks = await this.getTasks();
      
      if (tasks.length === 0) {
        settings.currentStreak = 0;
        settings.lastCompletionDate = null;
        await this.saveSettings(settings);
        return;
      }

      let currentStreak = 0;
      let lastCompletionDate = null;
      const today = new Date();
      
      // Check backwards from today to find the actual streak
      for (let i = 0; i < 365; i++) { // Check up to a year back
        const checkDate = new Date(today);
        checkDate.setDate(checkDate.getDate() - i);
        
        const completions = await this.getCompletionsForDate(checkDate);
        const completedTaskIds = completions.map(c => c.taskId);
        
        // Check if all tasks were completed on this date
        const allCompleted = tasks.every(task => completedTaskIds.includes(task.id));
        
        if (allCompleted) {
          currentStreak++;
          if (!lastCompletionDate) {
            lastCompletionDate = getLocalISODateString(checkDate);
          }
        } else {
          // Streak is broken, stop counting
          break;
        }
      }
      
      // Update settings with recalculated streak
      settings.currentStreak = currentStreak;
      settings.lastCompletionDate = lastCompletionDate;
      
      // Don't reduce record streak - it should remain the historical maximum
      // But if current streak somehow exceeds record (shouldn't happen), update it
      if (currentStreak > settings.recordStreak) {
        settings.recordStreak = currentStreak;
      }
      
      await this.saveSettings(settings);
      
      // Clean up streak records that are no longer valid
      await this.cleanupStreakRecords();
      
      return {
        currentStreak: settings.currentStreak,
        recordStreak: settings.recordStreak,
        isRecord: false // A recalculation should not trigger a new record notification
      };
    } catch (error) {
      console.error('Error recalculating streak:', error);
      throw error;
    }
  }

  // Clean up invalid streak records
  async cleanupStreakRecords() {
    try {
      const tasks = await this.getTasks();
      if (tasks.length === 0) return;

      // Get all streak records
      const streakRecords = await this.transaction('streaks', 'readonly', (store) => {
        return store.getAll();
      });

      // Check each record and remove invalid ones
      for (const record of streakRecords) {
        const recordDate = new Date(record.date);
        const completions = await this.getCompletionsForDate(recordDate);
        const completedTaskIds = completions.map(c => c.taskId);
        
        const allCompleted = tasks.every(task => completedTaskIds.includes(task.id));
        
        if (!allCompleted) {
          // This streak record is invalid, remove it
          await this.transaction('streaks', 'readwrite', (store) => {
            return store.delete(record.id);
          });
        }
      }
    } catch (error) {
      console.error('Error cleaning up streak records:', error);
      throw error;
    }
  }
}

// Create and export a single instance
const habitDB = new HabitDB();