// Main application code for Habit Tracker

// DOM Elements
const categoriesContainer = document.getElementById('categories-container');
const setupButton = document.getElementById('setup-button');
const configOverlay = document.getElementById('config-overlay');
const saveConfigBtn = document.getElementById('save-config-btn');
const closeConfigBtn = document.getElementById('close-config-btn');
const themeSelect = document.getElementById('theme-select');
const colorPicker = document.getElementById('color-picker');
const categoriesList = document.getElementById('categories-list');
const addCategoryBtn = document.getElementById('add-category-btn');
const taskEditor = document.getElementById('task-editor');
const currentCategoryName = document.getElementById('current-category-name');
const tasksList = document.getElementById('tasks-list');
const addTaskBtn = document.getElementById('add-task-btn');
const saveTasksBtn = document.getElementById('save-tasks-btn');
const closeTasksBtn = document.getElementById('close-tasks-btn');
const firstTimeSetup = document.getElementById('first-time-setup');
const streakContainer = document.getElementById('streak-container');
const currentStreakEl = document.getElementById('current-streak');
const recordStreakEl = document.getElementById('record-streak');
const streakRecordNotification = document.getElementById('streak-record-notification');
const progressBar = document.getElementById('progress-bar');
const dropIndicator = document.getElementById('drop-indicator');

// Templates
const categoryTemplate = document.getElementById('category-template');
const taskTemplate = document.getElementById('task-template');
const configCategoryTemplate = document.getElementById('config-category-template');
const configTaskTemplate = document.getElementById('config-task-template');

// Global state
let currentCategoryId = null;
let draggingElement = null;
let dragPreview = null;
let dragSourceContainer = null;
let dragSourceIndex = null;
let dragTargetIndex = null;
let completionColor = '#4CAF50';
let isDarkTheme = true;

// Initialize the application
async function initApp() {
  try {
    // Load settings
    const settings = await habitDB.getSettings();
    updateTheme(settings.theme);
    updateCompletionColor(settings.completionColor);
    
    // Update streak display
    updateStreakDisplay();
    
    // Load categories and tasks
    await loadCategories();
    
    // Show first time setup if needed
    const categories = await habitDB.getCategories();
    if (categories.length === 0) {
      firstTimeSetup.classList.remove('hidden');
      openConfig();
    }
    
    // Set up event listeners
    setupEventListeners();
    
    // Update progress
    updateAllProgress();
  } catch (error) {
    console.error('Error initializing app:', error);
  }
}

// Load categories and their tasks
async function loadCategories() {
  try {
    categoriesContainer.innerHTML = '';
    
    const categories = await habitDB.getCategories();
    
    if (categories.length === 0) {
      return;
    }
    
    // Sort categories by order
    categories.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const category of categories) {
      const categoryElement = createCategoryElement(category);
      categoriesContainer.appendChild(categoryElement);
      
      // Load tasks for this category
      await loadTasks(category.id);
    }
    
    // Show streak info if we have categories
    if (categories.length > 0) {
      streakContainer.classList.remove('hidden');
    }
  } catch (error) {
    console.error('Error loading categories:', error);
  }
}

// Create a category element
function createCategoryElement(category) {
  const template = categoryTemplate.content.cloneNode(true);
  const categoryElement = template.querySelector('.category');
  
  categoryElement.dataset.id = category.id;
  categoryElement.querySelector('h3').textContent = category.name;
  
  // Set up drag and drop
  initDraggable(categoryElement);
  
  return categoryElement;
}

// Load tasks for a category
async function loadTasks(categoryId) {
  try {
    const categoryElement = document.querySelector(`.category[data-id="${categoryId}"]`);
    if (!categoryElement) return;
    
    const tasksContainer = categoryElement.querySelector('.tasks-container');
    tasksContainer.innerHTML = '';
    
    const tasks = await habitDB.getTasksByCategory(categoryId);
    
    // Sort tasks by order
    tasks.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const task of tasks) {
      const taskElement = createTaskElement(task);
      tasksContainer.appendChild(taskElement);
      
      // Check if task is completed today
      const completion = await habitDB.getCompletionForTask(task.id, new Date());
      if (completion) {
        markTaskCompleted(taskElement);
      }
    }
    
    // Update category progress
    updateCategoryProgress(categoryId);
  } catch (error) {
    console.error('Error loading tasks:', error);
  }
}

// Create a task element
function createTaskElement(task) {
  const template = taskTemplate.content.cloneNode(true);
  const taskElement = template.querySelector('.task');
  
  taskElement.dataset.id = task.id;
  taskElement.querySelector('.task-text').textContent = task.name;
  
  // Set up drag and drop
  initDraggable(taskElement);
  
  return taskElement;
}

// Set up event listeners
function setupEventListeners() {
  // Setup button
  setupButton.addEventListener('click', openConfig);
  
  // Config overlay
  saveConfigBtn.addEventListener('click', saveConfig);
  closeConfigBtn.addEventListener('click', closeConfig);
  
  // Theme and color
  themeSelect.addEventListener('change', () => {
    updateTheme(themeSelect.value);
  });
  
  colorPicker.addEventListener('input', () => {
    updateCompletionColor(colorPicker.value);
  });
  
  // Category management
  addCategoryBtn.addEventListener('click', addNewCategory);
  
  // Task management
  addTaskBtn.addEventListener('click', addNewTask);
  saveTasksBtn.addEventListener('click', saveTasks);
  closeTasksBtn.addEventListener('click', closeTasks);
  
  // Category toggle
  categoriesContainer.addEventListener('click', (e) => {
    if (e.target.classList.contains('category-toggle')) {
      const category = e.target.closest('.category');
      toggleCategory(category);
    }
  });
  
  // Task completion
  categoriesContainer.addEventListener('click', (e) => {
    // Only handle clicks on the task text, not on buttons or drag handles
    if (e.target.classList.contains('task-text')) {
      const task = e.target.closest('.task');
      toggleTaskCompletion(task);
    }
  });
  
  // Long press to undo completion
  categoriesContainer.addEventListener('mousedown', handleLongPress);
  categoriesContainer.addEventListener('touchstart', handleLongPress, { passive: true });
  
  // Edit buttons
  categoriesContainer.addEventListener('click', (e) => {
    if (e.target.classList.contains('edit-category-btn')) {
      const category = e.target.closest('.category');
      editCategory(category);
    } else if (e.target.classList.contains('edit-task-btn')) {
      const task = e.target.closest('.task');
      editTask(task);
    }
  });
  
  // Config list actions
  categoriesList.addEventListener('click', (e) => {
    if (e.target.classList.contains('edit-item-btn')) {
      const item = e.target.closest('.config-item');
      editConfigItem(item, 'category');
    } else if (e.target.classList.contains('delete-item-btn')) {
      const item = e.target.closest('.config-item');
      deleteConfigItem(item, 'category');
    } else if (e.target.classList.contains('tasks-item-btn')) {
      const item = e.target.closest('.config-item');
      openTasksForCategory(item);
    }
  });
  
  // Tasks list actions
  tasksList.addEventListener('click', (e) => {
    if (e.target.classList.contains('edit-item-btn')) {
      const item = e.target.closest('.config-item');
      editConfigItem(item, 'task');
    } else if (e.target.classList.contains('delete-item-btn')) {
      const item = e.target.closest('.config-item');
      deleteConfigItem(item, 'task');
    }
  });
  
  // Document-wide event listeners for drag and drop
  document.addEventListener('mousemove', handleDragMove);
  document.addEventListener('touchmove', handleDragMove, { passive: false });
  document.addEventListener('mouseup', handleDragEnd);
  document.addEventListener('touchend', handleDragEnd);
}

// Handle long press for undoing completion
function handleLongPress(e) {
  // Only handle long press on category headers or tasks
  const target = e.target.closest('.category-header, .task');
  if (!target) return;
  
  // Don't handle long press on interactive elements
  if (e.target.closest('button, input, .drag-handle')) return;
  
  const startTime = Date.now();
  const longPressTimeout = 1000; // 1 second
  
  const element = target;
  let longPressTimer;
  
  const cancelLongPress = () => {
    clearTimeout(longPressTimer);
    element.removeEventListener('mouseup', cancelLongPress);
    element.removeEventListener('mouseleave', cancelLongPress);
    element.removeEventListener('touchend', cancelLongPress);
    element.removeEventListener('touchcancel', cancelLongPress);
  };
  
  const handleLongPressEnd = () => {
    cancelLongPress();
    
    if (element.classList.contains('category-header')) {
      const category = element.closest('.category');
      undoCategoryCompletion(category);
    } else if (element.classList.contains('task') && element.classList.contains('completed')) {
      undoTaskCompletion(element);
    }
  };
  
  longPressTimer = setTimeout(handleLongPressEnd, longPressTimeout);
  
  element.addEventListener('mouseup', cancelLongPress);
  element.addEventListener('mouseleave', cancelLongPress);
  element.addEventListener('touchend', cancelLongPress);
  element.addEventListener('touchcancel', cancelLongPress);
}

// Toggle category expansion
function toggleCategory(category) {
  const tasksContainer = category.querySelector('.tasks-container');
  const toggleButton = category.querySelector('.category-toggle');
  
  if (tasksContainer.style.display === 'none') {
    tasksContainer.style.display = 'block';
    toggleButton.textContent = '▼';
  } else {
    tasksContainer.style.display = 'none';
    toggleButton.textContent = '▶';
  }
}

// Toggle task completion
async function toggleTaskCompletion(taskElement) {
  try {
    const taskId = parseInt(taskElement.dataset.id);
    
    if (taskElement.classList.contains('completed')) {
      // Task is already completed, do nothing on click
      // (use long press to undo)
      return;
    }
    
    // Mark as completed
    await habitDB.addCompletion(taskId);
    markTaskCompleted(taskElement);
    
    // Update progress
    const categoryElement = taskElement.closest('.category');
    const categoryId = parseInt(categoryElement.dataset.id);
    updateCategoryProgress(categoryId);
    updateOverallProgress();
    
    // Update streak
    const streakInfo = await habitDB.updateStreak();
    updateStreakDisplay(streakInfo);
    
    // Check if all tasks in the category are completed
    const allCategoryTasksCompleted = await checkCategoryCompletion(categoryId);
    if (allCategoryTasksCompleted) {
      markCategoryCompleted(categoryElement);
    }
  } catch (error) {
    console.error('Error toggling task completion:', error);
  }
}

// Undo task completion
async function undoTaskCompletion(taskElement) {
  try {
    const taskId = parseInt(taskElement.dataset.id);
    
    // Remove completion
    await habitDB.removeCompletion(taskId);
    taskElement.classList.remove('completed');
    
    // Update progress
    const categoryElement = taskElement.closest('.category');
    const categoryId = parseInt(categoryElement.dataset.id);
    
    // Undo category completion if it was completed
    if (categoryElement.classList.contains('completed')) {
      categoryElement.classList.remove('completed');
      const tasksContainer = categoryElement.querySelector('.tasks-container');
      tasksContainer.style.display = 'block';
    }
    
    updateCategoryProgress(categoryId);
    updateOverallProgress();
    
    // Update streak
    const streakInfo = await habitDB.updateStreak();
    updateStreakDisplay(streakInfo);
  } catch (error) {
    console.error('Error undoing task completion:', error);
  }
}

// Undo all completions for a category
async function undoCategoryCompletion(categoryElement) {
  try {
    const categoryId = parseInt(categoryElement.dataset.id);
    const taskElements = categoryElement.querySelectorAll('.task');
    
    for (const taskElement of taskElements) {
      if (taskElement.classList.contains('completed')) {
        await undoTaskCompletion(taskElement);
      }
    }
    
    // Make sure category is not marked as completed
    categoryElement.classList.remove('completed');
    const tasksContainer = categoryElement.querySelector('.tasks-container');
    tasksContainer.style.display = 'block';
    
    // Update progress
    updateCategoryProgress(categoryId);
    updateOverallProgress();
  } catch (error) {
    console.error('Error undoing category completion:', error);
  }
}

// Mark a task as completed
function markTaskCompleted(taskElement) {
  taskElement.classList.add('completed');
  taskElement.style.backgroundColor = completionColor;
  
  // Set text color based on background color for contrast
  const textColor = getContrastColor(completionColor);
  taskElement.style.color = textColor;
}

// Mark a category as completed
function markCategoryCompleted(categoryElement) {
  categoryElement.classList.add('completed');
  
  // Hide tasks
  const tasksContainer = categoryElement.querySelector('.tasks-container');
  tasksContainer.style.display = 'none';
  
  // Update toggle button
  const toggleButton = categoryElement.querySelector('.category-toggle');
  toggleButton.textContent = '▶';
  
  // Set background color
  categoryElement.style.backgroundColor = completionColor;
  
  // Set text color based on background color for contrast
  const textColor = getContrastColor(completionColor);
  categoryElement.style.color = textColor;
  
  // Animate completion
  categoryElement.classList.add('category-completed-animation');
  setTimeout(() => {
    categoryElement.classList.remove('category-completed-animation');
  }, 1000);
}

// Check if all tasks in a category are completed
async function checkCategoryCompletion(categoryId) {
  try {
    const tasks = await habitDB.getTasksByCategory(categoryId);
    if (tasks.length === 0) return false;
    
    const completions = await habitDB.getCompletionsForDate(new Date());
    const completedTaskIds = completions.map(c => c.taskId);
    
    return tasks.every(task => completedTaskIds.includes(task.id));
  } catch (error) {
    console.error('Error checking category completion:', error);
    return false;
  }
}

// Update progress for a category
async function updateCategoryProgress(categoryId) {
  try {
    const categoryElement = document.querySelector(`.category[data-id="${categoryId}"]`);
    if (!categoryElement) return;
    
    const progressBar = categoryElement.querySelector('.progress-bar');
    const tasks = await habitDB.getTasksByCategory(categoryId);
    
    if (tasks.length === 0) {
      progressBar.style.width = '0%';
      return;
    }
    
    const completions = await habitDB.getCompletionsForDate(new Date());
    const completedTaskIds = completions.map(c => c.taskId);
    
    const categoryTasks = tasks.filter(task => task.categoryId === categoryId);
    const completedCount = categoryTasks.filter(task => completedTaskIds.includes(task.id)).length;
    
    const progressPercent = (completedCount / categoryTasks.length) * 100;
    progressBar.style.width = `${progressPercent}%`;
    
    // Update color
    progressBar.style.backgroundColor = completionColor;
  } catch (error) {
    console.error('Error updating category progress:', error);
  }
}

// Update overall progress
async function updateOverallProgress() {
  try {
    const tasks = await habitDB.getTasks();
    if (tasks.length === 0) {
      progressBar.style.width = '0%';
      return;
    }
    
    const completions = await habitDB.getCompletionsForDate(new Date());
    const completedTaskIds = completions.map(c => c.taskId);
    
    const completedCount = tasks.filter(task => completedTaskIds.includes(task.id)).length;
    const progressPercent = (completedCount / tasks.length) * 100;
    
    progressBar.style.width = `${progressPercent}%`;
    progressBar.style.backgroundColor = completionColor;
  } catch (error) {
    console.error('Error updating overall progress:', error);
  }
}

// Update all progress bars
async function updateAllProgress() {
  try {
    // Update category progress bars
    const categories = await habitDB.getCategories();
    for (const category of categories) {
      await updateCategoryProgress(category.id);
    }
    
    // Update overall progress
    await updateOverallProgress();
  } catch (error) {
    console.error('Error updating all progress:', error);
  }
}

// Update streak display
async function updateStreakDisplay(streakInfo) {
  try {
    if (!streakInfo) {
      streakInfo = await habitDB.getStreakInfo();
    }
    
    currentStreakEl.textContent = streakInfo.currentStreak;
    recordStreakEl.textContent = streakInfo.recordStreak;
    
    // Show record notification if needed
    if (streakInfo.isRecord && streakInfo.currentStreak > 1) {
      streakRecordNotification.classList.remove('hidden');
      
      // Show confetti for new records
      if (streakInfo.currentStreak >= 7) {
        showConfetti(streakInfo.currentStreak);
      }
      
      // Hide after 5 seconds
      setTimeout(() => {
        streakRecordNotification.classList.add('hidden');
      }, 5000);
    } else {
      streakRecordNotification.classList.add('hidden');
    }
  } catch (error) {
    console.error('Error updating streak display:', error);
  }
}

// Show confetti celebration
function showConfetti(streakCount) {
  // Create confetti container
  const confettiContainer = document.createElement('div');
  confettiContainer.className = 'confetti-container';
  document.body.appendChild(confettiContainer);
  
  // Determine intensity based on streak
  let particleCount = 50; // Default
  
  if (streakCount >= 30) {
    particleCount = 200; // Monthly streak
  } else if (streakCount >= 7) {
    particleCount = 100; // Weekly streak
  }
  
  // Create confetti particles
  for (let i = 0; i < particleCount; i++) {
    const confetti = document.createElement('div');
    confetti.className = 'confetti';
    
    // Random properties
    const size = Math.random() * 10 + 5;
    const color = `hsl(${Math.random() * 360}, 80%, 60%)`;
    const left = Math.random() * 100;
    const animationDuration = Math.random() * 3 + 2;
    const animationDelay = Math.random() * 2;
    
    // Apply styles
    confetti.style.width = `${size}px`;
    confetti.style.height = `${size}px`;
    confetti.style.backgroundColor = color;
    confetti.style.left = `${left}%`;
    confetti.style.animationDuration = `${animationDuration}s`;
    confetti.style.animationDelay = `${animationDelay}s`;
    
    confettiContainer.appendChild(confetti);
  }
  
  // Remove after animation completes
  setTimeout(() => {
    confettiContainer.remove();
  }, 5000);
}

// Open configuration overlay
function openConfig() {
  loadConfigData();
  configOverlay.classList.remove('hidden');
  taskEditor.classList.add('hidden');
}

// Close configuration overlay
function closeConfig() {
  configOverlay.classList.add('hidden');
  firstTimeSetup.classList.add('hidden');
}

// Load configuration data
async function loadConfigData() {
  try {
    // Load settings
    const settings = await habitDB.getSettings();
    themeSelect.value = settings.theme;
    colorPicker.value = settings.completionColor;
    
    // Load categories
    await loadConfigCategories();
  } catch (error) {
    console.error('Error loading config data:', error);
  }
}

// Load categories for configuration
async function loadConfigCategories() {
  try {
    categoriesList.innerHTML = '';
    
    const categories = await habitDB.getCategories();
    
    // Sort categories by order
    categories.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const category of categories) {
      const categoryElement = createConfigCategoryElement(category);
      categoriesList.appendChild(categoryElement);
    }
    
    // Set up drag and drop for config items
    const configItems = categoriesList.querySelectorAll('.config-item');
    configItems.forEach(item => {
      initDraggable(item);
    });
  } catch (error) {
    console.error('Error loading config categories:', error);
  }
}

// Create a category element for configuration
function createConfigCategoryElement(category) {
  const template = configCategoryTemplate.content.cloneNode(true);
  const categoryElement = template.querySelector('.config-item');
  
  categoryElement.dataset.id = category.id;
  categoryElement.querySelector('.item-text').textContent = category.name;
  
  return categoryElement;
}

// Save configuration
async function saveConfig() {
  try {
    // Save settings
    const settings = await habitDB.getSettings();
    settings.theme = themeSelect.value;
    settings.completionColor = colorPicker.value;
    await habitDB.saveSettings(settings);
    
    // Update UI
    updateTheme(settings.theme);
    updateCompletionColor(settings.completionColor);
    
    // Save category order
    const orderedCategoryIds = Array.from(categoriesList.querySelectorAll('.config-item'))
      .map(item => parseInt(item.dataset.id));
    
    if (orderedCategoryIds.length > 0) {
      await habitDB.updateCategoryOrder(orderedCategoryIds);
    }
    
    // Reload categories
    await loadCategories();
    
    // Close config
    closeConfig();
  } catch (error) {
    console.error('Error saving config:', error);
  }
}

// Add a new category
async function addNewCategory() {
  try {
    const categoryName = prompt('Enter category name:');
    if (!categoryName) return;
    
    const category = {
      name: categoryName
    };
    
    const categoryId = await habitDB.addCategory(category);
    
    // Reload config categories
    await loadConfigCategories();
  } catch (error) {
    console.error('Error adding category:', error);
  }
}

// Edit a category
async function editCategory(categoryElement) {
  try {
    const categoryId = parseInt(categoryElement.dataset.id);
    const categories = await habitDB.getCategories();
    const category = categories.find(c => c.id === categoryId);
    
    if (!category) return;
    
    const newName = prompt('Enter new category name:', category.name);
    if (!newName || newName === category.name) return;
    
    category.name = newName;
    await habitDB.updateCategory(category);
    
    // Update UI
    categoryElement.querySelector('h3').textContent = newName;
  } catch (error) {
    console.error('Error editing category:', error);
  }
}

// Edit a task
async function editTask(taskElement) {
  try {
    const taskId = parseInt(taskElement.dataset.id);
    const tasks = await habitDB.getTasks();
    const task = tasks.find(t => t.id === taskId);
    
    if (!task) return;
    
    const newName = prompt('Enter new task name:', task.name);
    if (!newName || newName === task.name) return;
    
    task.name = newName;
    await habitDB.updateTask(task);
    
    // Update UI
    taskElement.querySelector('.task-text').textContent = newName;
  } catch (error) {
    console.error('Error editing task:', error);
  }
}

// Edit a config item
async function editConfigItem(itemElement, type) {
  try {
    const itemId = parseInt(itemElement.dataset.id);
    
    if (type === 'category') {
      const categories = await habitDB.getCategories();
      const category = categories.find(c => c.id === itemId);
      
      if (!category) return;
      
      const newName = prompt('Enter new category name:', category.name);
      if (!newName || newName === category.name) return;
      
      category.name = newName;
      await habitDB.updateCategory(category);
      
      // Update UI
      itemElement.querySelector('.item-text').textContent = newName;
    } else if (type === 'task') {
      const tasks = await habitDB.getTasks();
      const task = tasks.find(t => t.id === itemId);
      
      if (!task) return;
      
      const newName = prompt('Enter new task name:', task.name);
      if (!newName || newName === task.name) return;
      
      task.name = newName;
      await habitDB.updateTask(task);
      
      // Update UI
      itemElement.querySelector('.item-text').textContent = newName;
    }
  } catch (error) {
    console.error('Error editing config item:', error);
  }
}

// Delete a config item
async function deleteConfigItem(itemElement, type) {
  try {
    const itemId = parseInt(itemElement.dataset.id);
    
    if (!confirm('Are you sure you want to delete this item?')) {
      return;
    }
    
    if (type === 'category') {
      await habitDB.deleteCategory(itemId);
    } else if (type === 'task') {
      await habitDB.deleteTask(itemId);
    }
    
    // Remove from UI
    itemElement.remove();
  } catch (error) {
    console.error('Error deleting config item:', error);
  }
}

// Open tasks editor for a category
async function openTasksForCategory(categoryElement) {
  try {
    const categoryId = parseInt(categoryElement.dataset.id);
    currentCategoryId = categoryId;
    
    const categories = await habitDB.getCategories();
    const category = categories.find(c => c.id === categoryId);
    
    if (!category) return;
    
    currentCategoryName.textContent = category.name;
    
    // Load tasks
    await loadConfigTasks(categoryId);
    
    // Show task editor
    taskEditor.classList.remove('hidden');
  } catch (error) {
    console.error('Error opening tasks for category:', error);
  }
}

// Load tasks for configuration
async function loadConfigTasks(categoryId) {
  try {
    tasksList.innerHTML = '';
    
    const tasks = await habitDB.getTasksByCategory(categoryId);
    
    // Sort tasks by order
    tasks.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const task of tasks) {
      const taskElement = createConfigTaskElement(task);
      tasksList.appendChild(taskElement);
    }
    
    // Set up drag and drop for config items
    const configItems = tasksList.querySelectorAll('.config-item');
    configItems.forEach(item => {
      initDraggable(item);
    });
  } catch (error) {
    console.error('Error loading config tasks:', error);
  }
}

// Create a task element for configuration
function createConfigTaskElement(task) {
  const template = configTaskTemplate.content.cloneNode(true);
  const taskElement = template.querySelector('.config-item');
  
  taskElement.dataset.id = task.id;
  taskElement.querySelector('.item-text').textContent = task.name;
  
  return taskElement;
}

// Add a new task
async function addNewTask() {
  try {
    if (!currentCategoryId) return;
    
    const taskName = prompt('Enter task name:');
    if (!taskName) return;
    
    const task = {
      name: taskName,
      categoryId: currentCategoryId
    };
    
    const taskId = await habitDB.addTask(task);
    
    // Reload config tasks
    await loadConfigTasks(currentCategoryId);
  } catch (error) {
    console.error('Error adding task:', error);
  }
}

// Save tasks
async function saveTasks() {
  try {
    // Save task order
    const orderedTaskIds = Array.from(tasksList.querySelectorAll('.config-item'))
      .map(item => parseInt(item.dataset.id));
    
    if (orderedTaskIds.length > 0 && currentCategoryId) {
      await habitDB.updateTaskOrder(currentCategoryId, orderedTaskIds);
    }
    
    // Reload tasks
    await loadTasks(currentCategoryId);
    
    // Close task editor
    closeTasks();
  } catch (error) {
    console.error('Error saving tasks:', error);
  }
}

// Close task editor
function closeTasks() {
  taskEditor.classList.add('hidden');
  currentCategoryId = null;
}

// Update theme
async function updateTheme(theme) {
  try {
    isDarkTheme = theme === 'dark';
    document.body.classList.toggle('dark-theme', isDarkTheme);
    document.body.classList.toggle('light-theme', !isDarkTheme);
    
    // Update settings
    const settings = await habitDB.getSettings();
    settings.theme = theme;
    await habitDB.saveSettings(settings);
  } catch (error) {
    console.error('Error updating theme:', error);
  }
}

// Update completion color
async function updateCompletionColor(color) {
  try {
    completionColor = color;
    document.documentElement.style.setProperty('--completion-color', color);
    
    // Update settings
    const settings = await habitDB.getSettings();
    settings.completionColor = color;
    await habitDB.saveSettings(settings);
    
    // Update completed tasks
    const completedTasks = document.querySelectorAll('.task.completed');
    completedTasks.forEach(task => {
      task.style.backgroundColor = color;
      task.style.color = getContrastColor(color);
    });
    
    // Update completed categories
    const completedCategories = document.querySelectorAll('.category.completed');
    completedCategories.forEach(category => {
      category.style.backgroundColor = color;
      category.style.color = getContrastColor(color);
    });
    
    // Update progress bars
    const progressBars = document.querySelectorAll('.progress-bar');
    progressBars.forEach(bar => {
      bar.style.backgroundColor = color;
    });
  } catch (error) {
    console.error('Error updating completion color:', error);
  }
}

// Get contrast color (black or white) based on background color
function getContrastColor(hexColor) {
  // Convert hex to RGB
  const r = parseInt(hexColor.substr(1, 2), 16);
  const g = parseInt(hexColor.substr(3, 2), 16);
  const b = parseInt(hexColor.substr(5, 2), 16);
  
  // Calculate luminance
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  
  // Return black for light colors, white for dark colors
  return luminance > 0.5 ? '#000000' : '#ffffff';
}

// Initialize draggable element
function initDraggable(element) {
  const dragHandle = element.querySelector('.drag-handle');
  
  if (!dragHandle) return;
  
  dragHandle.addEventListener('mousedown', handleDragStart);
  dragHandle.addEventListener('touchstart', handleDragStart, { passive: false });
}

// Handle drag start
function handleDragStart(e) {
  // Prevent default only for touch events
  if (e.type === 'touchstart') {
    e.preventDefault();
  }
  
  // Don't start drag on input elements
  if (e.target.closest('input, textarea, select, button:not(.drag-handle)')) {
    return;
  }
  
  const dragHandle = e.target.closest('.drag-handle');
  if (!dragHandle) return;
  
  draggingElement = dragHandle.closest('.draggable');
  if (!draggingElement) return;
  
  // Store the source container and index
  dragSourceContainer = draggingElement.parentElement;
  dragSourceIndex = Array.from(dragSourceContainer.children).indexOf(draggingElement);
  
  // Create drag preview
  createDragPreview(draggingElement, e);
  
  // Add dragging class
  draggingElement.classList.add('dragging');
  
  // Prevent text selection during drag
  document.body.style.userSelect = 'none';
}

// Create drag preview
function createDragPreview(element, e) {
  // Remove any existing preview
  if (dragPreview) {
    dragPreview.remove();
  }
  
  // Create preview element
  dragPreview = element.cloneNode(true);
  dragPreview.classList.add('drag-preview');
  dragPreview.style.width = `${element.offsetWidth}px`;
  
  // Remove any nested draggable elements from the preview
  const nestedDraggables = dragPreview.querySelectorAll('.draggable');
  nestedDraggables.forEach(nested => {
    if (nested !== dragPreview) {
      nested.remove();
    }
  });
  
  // Position the preview
  const clientX = e.type === 'touchstart' ? e.touches[0].clientX : e.clientX;
  const clientY = e.type === 'touchstart' ? e.touches[0].clientY : e.clientY;
  
  dragPreview.style.left = `${clientX}px`;
  dragPreview.style.top = `${clientY}px`;
  dragPreview.style.transform = 'translate(-50%, -50%)';
  
  // Add to document
  document.body.appendChild(dragPreview);
}

// Handle drag move
function handleDragMove(e) {
  if (!draggingElement || !dragPreview) return;
  
  // Prevent default to stop scrolling on touch devices
  if (e.type === 'touchmove') {
    e.preventDefault();
  }
  
  // Update preview position
  const clientX = e.type === 'touchmove' ? e.touches[0].clientX : e.clientX;
  const clientY = e.type === 'touchmove' ? e.touches[0].clientY : e.clientY;
  
  dragPreview.style.left = `${clientX}px`;
  dragPreview.style.top = `${clientY}px`;
  
  // Find the target container and position
  const targetContainer = findDropTarget(clientX, clientY);
  if (!targetContainer) return;
  
  // Find the target index
  const targetIndex = findDropIndex(targetContainer, clientY);
  
  // Show drop indicator
  showDropIndicator(targetContainer, targetIndex);
  
  // Store target index
  dragTargetIndex = targetIndex;
}

// Find drop target container
function findDropTarget(x, y) {
  // Check if we're in the categories list
  const categoriesListRect = categoriesList.getBoundingClientRect();
  if (
    x >= categoriesListRect.left &&
    x <= categoriesListRect.right &&
    y >= categoriesListRect.top &&
    y <= categoriesListRect.bottom
  ) {
    return categoriesList;
  }
  
  // Check if we're in the tasks list
  const tasksListRect = tasksList.getBoundingClientRect();
  if (
    x >= tasksListRect.left &&
    x <= tasksListRect.right &&
    y >= tasksListRect.top &&
    y <= tasksListRect.bottom
  ) {
    return tasksList;
  }
  
  // Check if we're in a category's tasks container
  const categories = document.querySelectorAll('.category');
  for (const category of categories) {
    const tasksContainer = category.querySelector('.tasks-container');
    if (tasksContainer.style.display === 'none') continue;
    
    const rect = tasksContainer.getBoundingClientRect();
    if (
      x >= rect.left &&
      x <= rect.right &&
      y >= rect.top &&
      y <= rect.bottom
    ) {
      return tasksContainer;
    }
  }
  
  // Check if we're in the categories container
  const categoriesContainerRect = categoriesContainer.getBoundingClientRect();
  if (
    x >= categoriesContainerRect.left &&
    x <= categoriesContainerRect.right &&
    y >= categoriesContainerRect.top &&
    y <= categoriesContainerRect.bottom
  ) {
    return categoriesContainer;
  }
  
  return null;
}

// Find drop index
function findDropIndex(container, y) {
  const items = Array.from(container.children).filter(
    child => child !== draggingElement && !child.classList.contains('drop-indicator')
  );
  
  if (items.length === 0) return 0;
  
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    const rect = item.getBoundingClientRect();
    const itemMiddle = rect.top + rect.height / 2;
    
    if (y < itemMiddle) {
      return i;
    }
  }
  
  return items.length;
}

// Show drop indicator
function showDropIndicator(container, index) {
  // Remove any existing indicators
  hideDropIndicator();
  
  // Get the element before which to insert the indicator
  const items = Array.from(container.children).filter(
    child => !child.classList.contains('drop-indicator')
  );
  
  // Clone the indicator
  const indicator = dropIndicator.cloneNode(true);
  indicator.classList.remove('hidden');
  
  // Insert at the correct position
  if (index >= items.length) {
    container.appendChild(indicator);
  } else {
    container.insertBefore(indicator, items[index]);
  }
}

// Hide drop indicator
function hideDropIndicator() {
  const indicators = document.querySelectorAll('.drop-indicator:not(.hidden)');
  indicators.forEach(indicator => {
    indicator.remove();
  });
}

// Handle drag end
async function handleDragEnd(e) {
  if (!draggingElement) return;
  
  // Remove dragging class
  draggingElement.classList.remove('dragging');
  
  // Remove drag preview
  if (dragPreview) {
    dragPreview.remove();
    dragPreview = null;
  }
  
  // Hide drop indicator
  hideDropIndicator();
  
  // Get the target container
  const targetContainer = findDropTarget(
    e.type === 'touchend' ? e.changedTouches[0].clientX : e.clientX,
    e.type === 'touchend' ? e.changedTouches[0].clientY : e.clientY
  );
  
  // If we have a valid target and index, move the element
  if (targetContainer && dragTargetIndex !== null) {
    await moveElement(targetContainer, dragTargetIndex);
  }
  
  // Reset drag state
  draggingElement = null;
  dragSourceContainer = null;
  dragSourceIndex = null;
  dragTargetIndex = null;
  
  // Restore text selection
  document.body.style.userSelect = '';
}

// Move element to new position
async function moveElement(targetContainer, targetIndex) {
  try {
    // Don't do anything if source and target are the same
    if (
      targetContainer === dragSourceContainer &&
      targetIndex === dragSourceIndex
    ) {
      return;
    }
    
    // Handle different container types
    if (targetContainer === categoriesList && dragSourceContainer === categoriesList) {
      // Reordering categories in config
      moveElementInDOM(targetContainer, targetIndex);
      
    } else if (targetContainer === tasksList && dragSourceContainer === tasksList) {
      // Reordering tasks in config
      moveElementInDOM(targetContainer, targetIndex);
      
    } else if (targetContainer === categoriesContainer && dragSourceContainer === categoriesContainer) {
      // Reordering categories in main view
      moveElementInDOM(targetContainer, targetIndex);
      
      // Update category order in database
      const orderedCategoryIds = Array.from(categoriesContainer.querySelectorAll('.category'))
        .map(item => parseInt(item.dataset.id));
      
      await habitDB.updateCategoryOrder(orderedCategoryIds);
      
    } else if (
      targetContainer.classList.contains('tasks-container') &&
      dragSourceContainer.classList.contains('tasks-container')
    ) {
      // Moving task between categories or reordering within category
      const sourceCategory = dragSourceContainer.closest('.category');
      const targetCategory = targetContainer.closest('.category');
      
      const sourceCategoryId = parseInt(sourceCategory.dataset.id);
      const targetCategoryId = parseInt(targetCategory.dataset.id);
      
      // Move in DOM
      moveElementInDOM(targetContainer, targetIndex);
      
      if (sourceCategoryId === targetCategoryId) {
        // Reordering within same category
        const orderedTaskIds = Array.from(targetContainer.querySelectorAll('.task'))
          .map(item => parseInt(item.dataset.id));
        
        await habitDB.updateTaskOrder(targetCategoryId, orderedTaskIds);
      } else {
        // Moving between categories
        const taskId = parseInt(draggingElement.dataset.id);
        const tasks = await habitDB.getTasks();
        const task = tasks.find(t => t.id === taskId);
        
        if (task) {
          // Update task's category
          task.categoryId = targetCategoryId;
          await habitDB.updateTask(task);
          
          // Update order in both categories
          const sourceOrderedTaskIds = Array.from(sourceCategory.querySelectorAll('.task'))
            .map(item => parseInt(item.dataset.id));
          
          const targetOrderedTaskIds = Array.from(targetCategory.querySelectorAll('.task'))
            .map(item => parseInt(item.dataset.id));
          
          await habitDB.updateTaskOrder(sourceCategoryId, sourceOrderedTaskIds);
          await habitDB.updateTaskOrder(targetCategoryId, targetOrderedTaskIds);
          
          // Update progress
          updateCategoryProgress(sourceCategoryId);
          updateCategoryProgress(targetCategoryId);
        }
      }
    }
  } catch (error) {
    console.error('Error moving element:', error);
  }
}

// Move element in the DOM
function moveElementInDOM(targetContainer, targetIndex) {
  // Get all items except the dragging element
  const items = Array.from(targetContainer.children).filter(
    child => child !== draggingElement && !child.classList.contains('drop-indicator')
  );
  
  // Insert at the correct position
  if (targetIndex >= items.length) {
    targetContainer.appendChild(draggingElement);
  } else {
    targetContainer.insertBefore(draggingElement, items[targetIndex]);
  }
}

// Initialize the app when the DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);