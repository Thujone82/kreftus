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

// Drag and drop functions
function handleDragMove(e) {
  // This function is referenced but not needed for HTML5 drag and drop
  // Keeping it empty to prevent errors
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
}

// Initialize draggable element
function initDraggable(element) {
  console.log('initDraggable called for:', element);
  const dragHandle = element.querySelector('.drag-handle');
  console.log('Found drag handle:', dragHandle);
  
  if (!dragHandle) return;
  
  // Make the entire element draggable
  element.draggable = true;
  console.log('Set element draggable to true');
  
  // Remove existing listeners to prevent duplicates
  element.removeEventListener('dragstart', handleDragStart);
  element.removeEventListener('dragend', handleDragEnd);
  
  // Add event listeners
  element.addEventListener('dragstart', handleDragStart);
  element.addEventListener('dragend', handleDragEnd);
  console.log('Added drag event listeners');
  
  // Add visual feedback on hover
  dragHandle.removeEventListener('mouseenter', dragHandle._mouseEnterHandler);
  dragHandle.removeEventListener('mouseleave', dragHandle._mouseLeaveHandler);
  
  dragHandle._mouseEnterHandler = () => {
    dragHandle.style.opacity = '1';
  };
  
  dragHandle._mouseLeaveHandler = () => {
    dragHandle.style.opacity = '0.5';
  };
  
  dragHandle.addEventListener('mouseenter', () => {
    dragHandle.style.opacity = '1';
  });
  
  dragHandle.addEventListener('mouseleave', () => {
    dragHandle.style.opacity = '0.5';
  });
}

// Handle drag start
function handleDragStart(e) {
  console.log('Drag start triggered for:', e.target);
  console.log('Event type:', e.type);
  console.log('DataTransfer available:', !!e.dataTransfer);
  draggingElement = e.target;
  dragSourceContainer = draggingElement.parentElement;
  
  // Set drag data
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', draggingElement.dataset.id);
  }
  
  // Add visual feedback
  setTimeout(() => {
    draggingElement.style.opacity = '0.5';
  }, 0);
  
  // Setup drop zones
  console.log('Setting up drop zones from drag start');
  setupDropZones();
}

// Setup drop zones
function setupDropZones() {
  // Find all containers that can accept drops
  const containers = [
    categoriesContainer,
    ...document.querySelectorAll('.tasks-container'),
    ...document.querySelectorAll('.inline-tasks-list')
  ];
  
  // Add config containers if they exist
  if (categoriesList && categoriesList.offsetParent !== null) {
    containers.push(categoriesList);
    console.log('Added categoriesList to drop zones');
  }
  if (tasksList && tasksList.offsetParent !== null) {
    containers.push(tasksList);
    console.log('Added tasksList to drop zones');
  }
  
  console.log('Setting up drop zones for containers:', containers.length);
  
  containers.forEach(container => {
    if (!container) return;
    
    console.log('Setting up drop zone for:', container.id || container.className);
    
    // Remove existing listeners to prevent duplicates
    container.removeEventListener('dragover', handleDragOver);
    container.removeEventListener('dragenter', handleDragEnter);
    container.removeEventListener('dragleave', handleDragLeave);
    container.removeEventListener('drop', handleDrop);
    
    container.addEventListener('dragover', handleDragOver);
    container.addEventListener('dragenter', handleDragEnter);
    container.addEventListener('dragleave', handleDragLeave);
    container.addEventListener('drop', handleDrop);
  });
}

// Handle drag over
function handleDragOver(e) {
  e.preventDefault();
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move';
  }
  
  const container = e.currentTarget;
  const afterElement = getDragAfterElement(container, e.clientY);
  showDropIndicator(container, afterElement);
  
}

// Handle drag enter
function handleDragEnter(e) {
  e.preventDefault();
  e.currentTarget.classList.add('drag-over');
}

// Handle drag leave
function handleDragLeave(e) {
  e.currentTarget.classList.remove('drag-over');
  hideDropIndicator();
}

// Get element after which to insert
function getDragAfterElement(container, y) {
  const draggableElements = [...container.children].filter(child => 
    child.classList.contains('draggable') && 
    child !== draggingElement &&
    !child.style.opacity.includes('0.5')
  );
  
  // If no elements, return null (drop at beginning)
  if (draggableElements.length === 0) {
    return null;
  }
  
  // Check if we're above the first element
  const firstElement = draggableElements[0];
  const firstBox = firstElement.getBoundingClientRect();
  if (y < firstBox.top + firstBox.height / 2) {
    return firstElement; // Insert before first element
  }
  
  // Check if we're below the last element
  const lastElement = draggableElements[draggableElements.length - 1];
  const lastBox = lastElement.getBoundingClientRect();
  if (y > lastBox.bottom) {
    return null; // Insert at end
  }
  
  // Find the element we're closest to
  let closestElement = null;
  let closestOffset = Number.NEGATIVE_INFINITY;
  
  for (const child of draggableElements) {
    const box = child.getBoundingClientRect();
    const offset = y - box.top - box.height / 2;
    
    if (offset < 0 && offset > closestOffset) {
      closestOffset = offset;
      closestElement = child;
    }
  }
  
  return closestElement;
}

// Show drop indicator
function showDropIndicator(container, afterElement) {
  hideDropIndicator();
  
  // Don't show indicator if we're dragging outside the valid container
  if (!container || !draggingElement) return;
  
  // Check if the container is a valid drop target for the dragging element
  const draggingElementType = draggingElement.classList.contains('category') ? 'category' : 'task';
  const isValidContainer = isValidDropContainer(container, draggingElementType);
  
  if (!isValidContainer) return;
  
  dropIndicator.className = 'drop-indicator';
  dropIndicator.classList.remove('hidden');
  
  if (afterElement === null) {
    // Insert at end of container, but within container bounds
    container.appendChild(dropIndicator);
  } else {
    // Insert before the specified element, ensuring it stays within container
    container.insertBefore(dropIndicator, afterElement);
  }
}

// Check if container is valid for the dragging element type
function isValidDropContainer(container, elementType) {
  if (elementType === 'category') {
    // Categories can only be dropped in categories container or categories list
    return container.id === 'categories-container' || container.id === 'categories-list';
  } else if (elementType === 'task') {
    // Tasks can be dropped in tasks containers, tasks list, or inline tasks list
    return container.classList.contains('tasks-container') || 
           container.id === 'tasks-list' || 
           container.classList.contains('inline-tasks-list');
  }
  return false;
}

// Handle drop
function handleDrop(e) {
  console.log('Drop triggered on:', e.currentTarget.id || e.currentTarget.className);
  e.preventDefault();
  e.stopPropagation();
  
  // Check if draggingElement is null and return early if so
  if (!draggingElement) {
    console.log('No dragging element found');
    cleanupDragState();
    return;
  }
  
  const container = e.currentTarget;
  console.log('Drop container:', container.id || container.className);
  
  // Prevent dropping categories inside other categories or tasks containers
  const draggingElementType = draggingElement.classList.contains('category') ? 'category' : 
                              (draggingElement.classList.contains('config-item') && 
                               container.id === 'categories-list') ? 'category' : 'task';
  
  console.log('Dragging element type:', draggingElementType);
  
  // Additional validation: prevent categories from being dropped in task containers
  if (draggingElementType === 'category' && container.classList.contains('tasks-container')) {
    console.log('Cannot drop category in task container');
    cleanupDragState();
    return;
  }
  
  // Additional validation: prevent tasks from being dropped in category containers (except task lists)
  if (draggingElementType === 'task' && container.id === 'categories-container') {
    console.log('Cannot drop task in categories container');
    cleanupDragState();
    return;
  }
  
  // Validate drop target
  if (!isValidDropContainer(container, draggingElementType)) {
    console.log('Invalid drop container');
    cleanupDragState();
    return;
  }
  
  const afterElement = getDragAfterElement(container, e.clientY);
  
  // Move the element to the correct position
  if (afterElement === null) {
    container.appendChild(draggingElement);
  } else {
    container.insertBefore(draggingElement, afterElement);
  }
  
  console.log('Element moved successfully');
  
  // Update database order
  updateOrderInDatabase(container);
  
  // Clean up
  cleanupDragState();
}

// Handle drag end
function handleDragEnd(e) {
  console.log('Drag end triggered');
  console.log('Event type:', e.type);
  cleanupDragState();
}

// Clean up drag state
function cleanupDragState() {
  if (draggingElement) {
    draggingElement.style.opacity = '';
  }
  
  // Remove drag-over classes
  document.querySelectorAll('.drag-over').forEach(el => {
    el.classList.remove('drag-over');
  });
  
  hideDropIndicator();
  
  draggingElement = null;
  dragSourceContainer = null;
}

// Hide drop indicator
function hideDropIndicator() {
  if (dropIndicator.parentNode) {
    dropIndicator.parentNode.removeChild(dropIndicator);
  }
  dropIndicator.classList.add('hidden');
}

// Update order in database
async function updateOrderInDatabase(container) {
  try {
    if (container.id === 'categories-list') {
      // Reordering categories in config
      const orderedCategoryIds = Array.from(container.querySelectorAll('.config-item'))
        .map(item => parseInt(item.dataset.id));
      await habitDB.updateCategoryOrder(orderedCategoryIds);
      
    } else if (container.id === 'tasks-list') {
      // Reordering tasks in config
      if (currentCategoryId) {
        const orderedTaskIds = Array.from(container.querySelectorAll('.config-item'))
          .map(item => parseInt(item.dataset.id));
        await habitDB.updateTaskOrder(currentCategoryId, orderedTaskIds);
      }
      
    } else if (container.classList.contains('inline-tasks-list')) {
      // Reordering tasks in inline editor
      const categoryElement = container.closest('.inline-task-editor').previousElementSibling;
      const categoryId = parseInt(categoryElement.dataset.id);
      const orderedTaskIds = Array.from(container.querySelectorAll('.config-item'))
        .map(item => parseInt(item.dataset.id));
      await habitDB.updateTaskOrder(categoryId, orderedTaskIds);
      
    } else if (container.id === 'categories-container') {
      // Reordering categories in main view
      const orderedCategoryIds = Array.from(container.querySelectorAll('.category'))
        .map(item => parseInt(item.dataset.id));
      await habitDB.updateCategoryOrder(orderedCategoryIds);
      
    } else if (container.classList.contains('tasks-container')) {
      // Reordering tasks within category
      const category = container.closest('.category');
      const categoryId = parseInt(category.dataset.id);
      const orderedTaskIds = Array.from(container.querySelectorAll('.task'))
        .map(item => parseInt(item.dataset.id));
      await habitDB.updateTaskOrder(categoryId, orderedTaskIds);
      
      // Update progress
      updateCategoryProgress(categoryId);
    }
  } catch (error) {
    console.error('Error updating order in database:', error);
  }
}

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
    
    // Reset background and text color to default
    taskElement.style.backgroundColor = '';
    taskElement.style.color = '';
    
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
    
    // Recalculate streak from scratch to ensure accuracy
    const streakInfo = await habitDB.recalculateStreak();
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
    categoryElement.style.backgroundColor = '';
    categoryElement.style.color = '';
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
    console.log('Loading config categories...');
    categoriesList.innerHTML = '';
    
    const categories = await habitDB.getCategories();
    console.log('Found categories:', categories);
    
    // Sort categories by order
    categories.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const category of categories) {
      const categoryElement = createConfigCategoryElement(category);
      console.log('Created category element:', categoryElement);
      categoriesList.appendChild(categoryElement);
    }
    
    // Set up drag and drop for all config items after they're in the DOM
    setTimeout(() => {
      const configItems = categoriesList.querySelectorAll('.config-item');
      console.log('Setting up drag for config items:', configItems.length);
      configItems.forEach(item => {
        console.log('Setting up drag for config item:', item.dataset.id);
        console.log('Item has drag handle:', !!item.querySelector('.drag-handle'));
        initDraggable(item);
      });
      
      // Setup drop zones initially
      console.log('Setting up initial drop zones');
      setupDropZones();
    }, 0);
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
  
  console.log('Created config category element:', category.id, categoryElement);
  
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
    
    // Check if inline editor already exists for this category
    const existingEditor = categoryElement.parentNode.querySelector('.inline-task-editor');
    if (existingEditor) {
      // Close the existing editor
      existingEditor.remove();
      return;
    }
    
    // Close any other open inline editors
    const otherEditors = document.querySelectorAll('.inline-task-editor');
    otherEditors.forEach(editor => editor.remove());
    
    currentCategoryId = categoryId;
    
    const categories = await habitDB.getCategories();
    const category = categories.find(c => c.id === categoryId);
    
    if (!category) return;
    
    currentCategoryName.textContent = category.name;
    
    // Load tasks
    await loadConfigTasks(categoryId);
    
    // Hide the main task editor and show inline editor
    taskEditor.classList.add('hidden');
    
    // Create or show inline task editor for this category
    showInlineTaskEditor(categoryElement, categoryId);
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
  // Hide all inline task editors
  const inlineEditors = document.querySelectorAll('.inline-task-editor');
  inlineEditors.forEach(editor => editor.remove());
  
  taskEditor.classList.add('hidden');
  currentCategoryId = null;
}

// Show inline task editor under specific category
function showInlineTaskEditor(categoryElement, categoryId) {
  // Remove any existing inline editors
  const existingEditors = document.querySelectorAll('.inline-task-editor');
  existingEditors.forEach(editor => editor.remove());
  
  // Create inline task editor
  const inlineEditor = document.createElement('div');
  inlineEditor.className = 'inline-task-editor';
  inlineEditor.innerHTML = `
    <div class="inline-editor-content">
      <h4>Tasks for ${categoryElement.querySelector('.item-text').textContent}</h4>
      <div class="inline-tasks-list"></div>
      <button class="inline-add-task-btn secondary-button">Add Task</button>
      <div class="inline-button-row">
        <button class="inline-save-tasks-btn primary-button">Save</button>
        <button class="inline-close-tasks-btn secondary-button">Close</button>
      </div>
    </div>
  `;
  
  // Insert after the category element
  categoryElement.parentNode.insertBefore(inlineEditor, categoryElement.nextSibling);
  
  // Load tasks into inline editor
  loadInlineConfigTasks(categoryId, inlineEditor);
  
  // Set up event listeners for inline editor
  const addTaskBtn = inlineEditor.querySelector('.inline-add-task-btn');
  const saveTasksBtn = inlineEditor.querySelector('.inline-save-tasks-btn');
  const closeTasksBtn = inlineEditor.querySelector('.inline-close-tasks-btn');
  
  addTaskBtn.addEventListener('click', () => addInlineTask(categoryId, inlineEditor));
  saveTasksBtn.addEventListener('click', () => saveInlineTasks(categoryId, inlineEditor));
  closeTasksBtn.addEventListener('click', () => inlineEditor.remove());
}

// Load tasks for inline configuration
async function loadInlineConfigTasks(categoryId, inlineEditor) {
  try {
    const tasksList = inlineEditor.querySelector('.inline-tasks-list');
    tasksList.innerHTML = '';
    
    const tasks = await habitDB.getTasksByCategory(categoryId);
    
    // Sort tasks by order
    tasks.sort((a, b) => (a.order || 0) - (b.order || 0));
    
    for (const task of tasks) {
      const taskElement = createInlineConfigTaskElement(task);
      tasksList.appendChild(taskElement);
    }
    
    // Set up drag and drop for inline config items
    const configItems = tasksList.querySelectorAll('.config-item');
    configItems.forEach(item => {
      initDraggable(item);
    });
  } catch (error) {
    console.error('Error loading inline config tasks:', error);
  }
}

// Create a task element for inline configuration
function createInlineConfigTaskElement(task) {
  const template = configTaskTemplate.content.cloneNode(true);
  const taskElement = template.querySelector('.config-item');
  
  taskElement.dataset.id = task.id;
  taskElement.querySelector('.item-text').textContent = task.name;
  
  return taskElement;
}

// Add a new task in inline editor
async function addInlineTask(categoryId, inlineEditor) {
  try {
    const taskName = prompt('Enter task name:');
    if (!taskName) return;
    
    const task = {
      name: taskName,
      categoryId: categoryId
    };
    
    const taskId = await habitDB.addTask(task);
    
    // Reload inline config tasks
    await loadInlineConfigTasks(categoryId, inlineEditor);
  } catch (error) {
    console.error('Error adding inline task:', error);
  }
}

// Save tasks from inline editor
async function saveInlineTasks(categoryId, inlineEditor) {
  try {
    const tasksList = inlineEditor.querySelector('.inline-tasks-list');
    
    // Save task order
    const orderedTaskIds = Array.from(tasksList.querySelectorAll('.config-item'))
      .map(item => parseInt(item.dataset.id));
    
    if (orderedTaskIds.length > 0) {
      await habitDB.updateTaskOrder(categoryId, orderedTaskIds);
    }
    
    // Reload tasks in main view
    await loadTasks(categoryId);
    
    // Close inline editor
    inlineEditor.remove();
  } catch (error) {
    console.error('Error saving inline tasks:', error);
  }
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
    document.documentElement.style.setProperty('--primary-color', color);
    
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
    
    // Update buttons
    const primaryButtons = document.querySelectorAll('.primary-button');
    primaryButtons.forEach(button => {
      button.style.backgroundColor = color;
    });
    
    // Update secondary buttons
    const secondaryButtons = document.querySelectorAll('.secondary-button');
    secondaryButtons.forEach(button => {
      button.style.borderColor = color;
      button.style.color = isDarkTheme ? '#ffffff' : color;
    });
    
    // Update streak record notification
    const streakNotification = document.getElementById('streak-record-notification');
    if (streakNotification) {
      streakNotification.style.backgroundColor = color;
    }
    
    // Update inline editor borders
    const inlineEditors = document.querySelectorAll('.inline-task-editor');
    inlineEditors.forEach(editor => {
      editor.style.borderLeftColor = color;
    });
    
    // Update inline editor headings
    const inlineHeadings = document.querySelectorAll('.inline-editor-content h4');
    inlineHeadings.forEach(heading => {
      heading.style.color = color;
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

// Initialize the app when the DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);

// Register the service worker
window.addEventListener('load', () => {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('service-worker.js')
      .then(registration => {
        console.log('Service Worker registered successfully with scope:', registration.scope);
      })
      .catch(err => {
        console.error('Service Worker registration failed:', err);
      });
  } else {
    console.log('Service Worker is not supported by this browser.');
  }
});