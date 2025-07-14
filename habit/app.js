// Main application logic
class HabitTracker {
    constructor() {
        this.db = new HabitDB();
        this.categories = [];
        this.tasks = [];
        this.completions = [];
        this.settings = {};
        this.streaks = { current: 0, record: 0 };
        this.currentEditingCategory = null;
        this.longPressTimer = null;
        this.longPressActive = false;
        
        this.init();
    }

    async init() {
        await this.db.init();
        await this.loadData();
        this.setupEventListeners();
        this.checkFirstTimeSetup();
        this.render();
        
        // Register service worker
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/service-worker.js');
        }
    }

    async loadData() {
        this.settings = await this.db.getSettings();
        this.categories = await this.db.getCategories();
        this.tasks = await this.db.getTasks();
        this.completions = await this.db.getCompletions(this.db.getTodayString());
        this.streaks = await this.db.getStreaks();
        
        // Apply theme
        document.body.className = this.settings.theme === 'light' ? 'light-mode' : 'dark-mode';
        
        // Apply completion color
        document.documentElement.style.setProperty('--completion-color', this.settings.completionColor);
    }

    checkFirstTimeSetup() {
        if (this.categories.length === 0) {
            this.showConfigView();
            document.getElementById('first-time-setup').classList.remove('hidden');
        }
    }

    setupEventListeners() {
        // Main view events
        document.getElementById('setup-button').addEventListener('click', () => this.showConfigView());
        
        // Config view events
        document.getElementById('theme-selector').addEventListener('change', (e) => {
            this.settings.theme = e.target.value;
            document.body.className = e.target.value === 'light' ? 'light-mode' : 'dark-mode';
        });
        
        document.getElementById('completion-color').addEventListener('input', (e) => {
            this.settings.completionColor = e.target.value;
            document.documentElement.style.setProperty('--completion-color', e.target.value);
            document.getElementById('color-preview').style.backgroundColor = e.target.value;
        });
        
        document.getElementById('add-category-btn').addEventListener('click', () => this.addCategory());
        document.getElementById('save-config-btn').addEventListener('click', () => this.saveConfig());
        document.getElementById('close-config-btn').addEventListener('click', () => this.closeConfig());
        
        // Task manager events
        document.getElementById('add-task-btn').addEventListener('click', () => this.addTask());
        document.getElementById('save-tasks-btn').addEventListener('click', () => this.saveTasks());
        document.getElementById('close-tasks-btn').addEventListener('click', () => this.closeTaskManager());
        
        // Theme selector setup
        document.getElementById('theme-selector').value = this.settings.theme;
        document.getElementById('completion-color').value = this.settings.completionColor;
        document.getElementById('color-preview').style.backgroundColor = this.settings.completionColor;
    }

    render() {
        this.renderStreaks();
        this.renderCategories();
        this.renderCategoriesConfig();
    }

    renderStreaks() {
        const streakText = document.getElementById('streak-text');
        const streakProgress = document.getElementById('streak-progress');
        
        if (this.streaks.current > 0 || this.streaks.record > 0) {
            streakText.textContent = `Streak: ${this.streaks.current} Record: ${this.streaks.record}`;
            streakText.parentElement.style.display = 'block';
        } else {
            streakText.parentElement.style.display = 'none';
        }
        
        // Update progress bar based on daily completion
        const progress = this.getDailyProgress();
        streakProgress.style.width = `${progress}%`;
    }

    renderCategories() {
        const container = document.getElementById('categories-container');
        container.innerHTML = '';
        
        this.categories.forEach(category => {
            const categoryTasks = this.tasks.filter(task => task.categoryId === category.id);
            const categoryElement = this.createCategoryElement(category, categoryTasks);
            container.appendChild(categoryElement);
        });
    }

    createCategoryElement(category, tasks) {
        const categoryDiv = document.createElement('div');
        categoryDiv.className = 'category';
        categoryDiv.dataset.categoryId = category.id;
        
        const completedTasks = tasks.filter(task => 
            this.completions.some(comp => comp.taskId === task.id)
        );
        
        const isCompleted = tasks.length > 0 && completedTasks.length === tasks.length;
        
        if (isCompleted) {
            categoryDiv.classList.add('completed');
        }
        
        const headerDiv = document.createElement('div');
        headerDiv.className = 'category-header';
        headerDiv.tabIndex = 0;
        
        const progressDiv = document.createElement('div');
        progressDiv.className = 'category-progress';
        progressDiv.style.width = tasks.length > 0 ? `${(completedTasks.length / tasks.length) * 100}%` : '0%';
        
        const nameDiv = document.createElement('div');
        nameDiv.className = 'category-name';
        nameDiv.textContent = category.name;
        
        headerDiv.appendChild(progressDiv);
        headerDiv.appendChild(nameDiv);
        
        const tasksContainer = document.createElement('div');
        tasksContainer.className = 'tasks-container';
        
        if (isCompleted) {
            tasksContainer.classList.add('collapsed');
        }
        
        tasks.forEach(task => {
            const taskElement = this.createTaskElement(task);
            tasksContainer.appendChild(taskElement);
        });
        
        categoryDiv.appendChild(headerDiv);
        categoryDiv.appendChild(tasksContainer);
        
        // Add event listeners
        this.addCategoryEventListeners(headerDiv, category, tasks);
        
        return categoryDiv;
    }

    createTaskElement(task) {
        const taskDiv = document.createElement('div');
        taskDiv.className = 'task';
        taskDiv.dataset.taskId = task.id;
        taskDiv.textContent = task.name;
        taskDiv.tabIndex = 0;
        
        const isCompleted = this.completions.some(comp => comp.taskId === task.id);
        
        if (isCompleted) {
            taskDiv.classList.add('completed');
            taskDiv.style.color = this.getContrastColor(this.settings.completionColor);
        }
        
        // Add event listeners
        this.addTaskEventListeners(taskDiv, task);
        
        return taskDiv;
    }

    addCategoryEventListeners(headerDiv, category, tasks) {
        // Long press to uncomplete category
        headerDiv.addEventListener('mousedown', (e) => this.startLongPress(e, 'category', category.id));
        headerDiv.addEventListener('touchstart', (e) => this.startLongPress(e, 'category', category.id));
        headerDiv.addEventListener('mouseup', () => this.endLongPress());
        headerDiv.addEventListener('mouseleave', () => this.endLongPress());
        headerDiv.addEventListener('touchend', () => this.endLongPress());
        headerDiv.addEventListener('touchcancel', () => this.endLongPress());
        
        // Keyboard support
        headerDiv.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.startLongPress(e, 'category', category.id);
            }
        });
        
        headerDiv.addEventListener('keyup', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                this.endLongPress();
            }
        });
    }

    addTaskEventListeners(taskDiv, task) {
        // Click to toggle completion
        taskDiv.addEventListener('click', () => this.toggleTask(task.id));
        
        // Long press to uncomplete task
        taskDiv.addEventListener('mousedown', (e) => this.startLongPress(e, 'task', task.id));
        taskDiv.addEventListener('touchstart', (e) => this.startLongPress(e, 'task', task.id));
        taskDiv.addEventListener('mouseup', () => this.endLongPress());
        taskDiv.addEventListener('mouseleave', () => this.endLongPress());
        taskDiv.addEventListener('touchend', () => this.endLongPress());
        taskDiv.addEventListener('touchcancel', () => this.endLongPress());
        
        // Keyboard support
        taskDiv.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.toggleTask(task.id);
            }
        });
    }

    startLongPress(e, type, id) {
        e.preventDefault();
        this.longPressActive = true;
        
        const element = e.target.closest(type === 'category' ? '.category-header' : '.task');
        element.classList.add('long-press-active');
        
        this.longPressTimer = setTimeout(() => {
            if (this.longPressActive) {
                if (type === 'category') {
                    this.uncompleteCategory(id);
                } else {
                    this.uncompleteTask(id);
                }
            }
        }, 1000);
    }

    endLongPress() {
        this.longPressActive = false;
        
        if (this.longPressTimer) {
            clearTimeout(this.longPressTimer);
            this.longPressTimer = null;
        }
        
        // Remove long press visual feedback
        document.querySelectorAll('.long-press-active').forEach(el => {
            el.classList.remove('long-press-active');
        });
    }

    async toggleTask(taskId) {
        const existingCompletion = this.completions.find(comp => comp.taskId === taskId);
        
        if (existingCompletion) {
            // Uncomplete task
            await this.db.deleteCompletion(existingCompletion.id);
            this.completions = this.completions.filter(comp => comp.id !== existingCompletion.id);
        } else {
            // Complete task
            const completion = {
                id: this.db.generateId(),
                taskId: taskId,
                date: this.db.getTodayString(),
                timestamp: new Date().toISOString()
            };
            
            await this.db.saveCompletion(completion);
            this.completions.push(completion);
        }
        
        await this.updateStreaks();
        this.render();
    }

    async uncompleteTask(taskId) {
        const existingCompletion = this.completions.find(comp => comp.taskId === taskId);
        
        if (existingCompletion) {
            await this.db.deleteCompletion(existingCompletion.id);
            this.completions = this.completions.filter(comp => comp.id !== existingCompletion.id);
            await this.updateStreaks();
            this.render();
        }
    }

    async uncompleteCategory(categoryId) {
        const categoryTasks = this.tasks.filter(task => task.categoryId === categoryId);
        
        for (const task of categoryTasks) {
            await this.uncompleteTask(task.id);
        }
    }

    async updateStreaks() {
        const totalTasks = this.tasks.length;
        const completedTasks = this.completions.length;
        const isCompleteDay = totalTasks > 0 && completedTasks === totalTasks;
        
        if (isCompleteDay) {
            this.streaks.current++;
            
            if (this.streaks.current > this.streaks.record) {
                this.streaks.record = this.streaks.current;
                this.showCelebration();
            }
        } else {
            this.streaks.current = 0;
        }
        
        await this.db.saveStreaks(this.streaks);
    }

    showCelebration() {
        const container = document.getElementById('celebration-container');
        
        // Create celebration text
        const celebrationText = document.createElement('div');
        celebrationText.className = 'celebration-text';
        celebrationText.textContent = 'New Streak Record!';
        celebrationText.style.color = this.settings.completionColor;
        
        container.appendChild(celebrationText);
        
        // Create confetti
        this.createConfetti(container);
        
        // Clean up after animation
        setTimeout(() => {
            container.innerHTML = '';
        }, 3000);
    }

    createConfetti(container) {
        const colors = [this.settings.completionColor, '#fbbf24', '#f59e0b', '#d97706'];
        
        for (let i = 0; i < 50; i++) {
            const confetti = document.createElement('div');
            confetti.className = 'confetti';
            confetti.style.left = Math.random() * 100 + '%';
            confetti.style.backgroundColor = colors[Math.floor(Math.random() * colors.length)];
            confetti.style.animationDelay = Math.random() * 3 + 's';
            confetti.style.animationDuration = (Math.random() * 3 + 2) + 's';
            container.appendChild(confetti);
        }
    }

    getDailyProgress() {
        if (this.tasks.length === 0) return 0;
        return (this.completions.length / this.tasks.length) * 100;
    }

    getContrastColor(hexColor) {
        // Convert hex to RGB
        const r = parseInt(hexColor.slice(1, 3), 16);
        const g = parseInt(hexColor.slice(3, 5), 16);
        const b = parseInt(hexColor.slice(5, 7), 16);
        
        // Calculate relative luminance
        const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        
        // Return black or white based on luminance
        return luminance > 0.5 ? '#000000' : '#ffffff';
    }

    // Configuration methods
    showConfigView() {
        document.getElementById('main-view').classList.add('hidden');
        document.getElementById('config-view').classList.remove('hidden');
        this.renderCategoriesConfig();
    }

    closeConfig() {
        document.getElementById('config-view').classList.add('hidden');
        document.getElementById('main-view').classList.remove('hidden');
        document.getElementById('first-time-setup').classList.add('hidden');
    }

    async saveConfig() {
        await this.db.saveSetting('theme', this.settings.theme);
        await this.db.saveSetting('completionColor', this.settings.completionColor);
        
        this.closeConfig();
    }

    renderCategoriesConfig() {
        const container = document.getElementById('categories-config');
        container.innerHTML = '';
        
        this.categories.forEach((category, index) => {
            const categoryElement = this.createCategoryConfigElement(category, index);
            container.appendChild(categoryElement);
        });
        
        this.setupDragAndDrop(container, 'category');
    }

    createCategoryConfigElement(category, index) {
        const div = document.createElement('div');
        div.className = 'category-config';
        div.dataset.categoryId = category.id;
        div.dataset.order = index;
        
        div.innerHTML = `
            <div class="drag-handle">≡</div>
            <input type="text" value="${category.name}" data-field="name" />
            <button class="delete-btn" data-action="delete-category">Delete</button>
            <button class="add-btn" data-action="manage-tasks">Tasks</button>
        `;
        
        // Add event listeners
        const input = div.querySelector('input');
        input.addEventListener('input', (e) => {
            category.name = e.target.value;
        });
        
        const deleteBtn = div.querySelector('[data-action="delete-category"]');
        deleteBtn.addEventListener('click', () => this.deleteCategory(category.id));
        
        const tasksBtn = div.querySelector('[data-action="manage-tasks"]');
        tasksBtn.addEventListener('click', () => this.showTaskManager(category));
        
        return div;
    }

    addCategory() {
        const category = {
            id: this.db.generateId(),
            name: 'New Category',
            order: this.categories.length
        };
        
        this.categories.push(category);
        this.renderCategoriesConfig();
    }

    async deleteCategory(categoryId) {
        if (confirm('Are you sure you want to delete this category and all its tasks?')) {
            await this.db.deleteCategory(categoryId);
            this.categories = this.categories.filter(cat => cat.id !== categoryId);
            this.tasks = this.tasks.filter(task => task.categoryId !== categoryId);
            this.renderCategoriesConfig();
        }
    }

    // Task management methods
    showTaskManager(category) {
        this.currentEditingCategory = category;
        document.getElementById('task-manager-title').textContent = `Manage Tasks - ${category.name}`;
        document.getElementById('task-manager-modal').classList.remove('hidden');
        this.renderTasksConfig();
    }

    closeTaskManager() {
        document.getElementById('task-manager-modal').classList.add('hidden');
        this.currentEditingCategory = null;
    }

    renderTasksConfig() {
        const container = document.getElementById('tasks-config');
        container.innerHTML = '';
        
        const categoryTasks = this.tasks.filter(task => task.categoryId === this.currentEditingCategory.id);
        categoryTasks.sort((a, b) => a.order - b.order);
        
        categoryTasks.forEach((task, index) => {
            const taskElement = this.createTaskConfigElement(task, index);
            container.appendChild(taskElement);
        });
        
        this.setupDragAndDrop(container, 'task');
    }

    createTaskConfigElement(task, index) {
        const div = document.createElement('div');
        div.className = 'task-config';
        div.dataset.taskId = task.id;
        div.dataset.order = index;
        
        div.innerHTML = `
            <div class="drag-handle">≡</div>
            <input type="text" value="${task.name}" data-field="name" />
            <button class="delete-btn" data-action="delete-task">Delete</button>
        `;
        
        // Add event listeners
        const input = div.querySelector('input');
        input.addEventListener('input', (e) => {
            task.name = e.target.value;
        });
        
        const deleteBtn = div.querySelector('[data-action="delete-task"]');
        deleteBtn.addEventListener('click', () => this.deleteTask(task.id));
        
        return div;
    }

    addTask() {
        const task = {
            id: this.db.generateId(),
            name: 'New Task',
            categoryId: this.currentEditingCategory.id,
            order: this.tasks.filter(t => t.categoryId === this.currentEditingCategory.id).length
        };
        
        this.tasks.push(task);
        this.renderTasksConfig();
    }

    async deleteTask(taskId) {
        if (confirm('Are you sure you want to delete this task?')) {
            await this.db.deleteTask(taskId);
            this.tasks = this.tasks.filter(task => task.id !== taskId);
            this.renderTasksConfig();
        }
    }

    async saveTasks() {
        // Save all categories and tasks
        for (const category of this.categories) {
            await this.db.saveCategory(category);
        }
        
        for (const task of this.tasks) {
            await this.db.saveTask(task);
        }
        
        this.closeTaskManager();
        this.render();
    }

    // Drag and drop functionality
    setupDragAndDrop(container, type) {
        let draggedElement = null;
        let draggedIndex = null;
        
        container.addEventListener('dragstart', (e) => {
            if (e.target.classList.contains(`${type}-config`)) {
                draggedElement = e.target;
                draggedIndex = parseInt(e.target.dataset.order);
                e.target.classList.add('dragging');
            }
        });
        
        container.addEventListener('dragend', (e) => {
            e.target.classList.remove('dragging');
            draggedElement = null;
            draggedIndex = null;
        });
        
        container.addEventListener('dragover', (e) => {
            e.preventDefault();
        });
        
        container.addEventListener('drop', (e) => {
            e.preventDefault();
            
            if (!draggedElement) return;
            
            const dropTarget = e.target.closest(`.${type}-config`);
            if (!dropTarget || dropTarget === draggedElement) return;
            
            const dropIndex = parseInt(dropTarget.dataset.order);
            
            // Update order in arrays
            if (type === 'category') {
                const item = this.categories.splice(draggedIndex, 1)[0];
                this.categories.splice(dropIndex, 0, item);
                
                // Update order values
                this.categories.forEach((cat, index) => {
                    cat.order = index;
                });
                
                this.renderCategoriesConfig();
            } else {
                const categoryTasks = this.tasks.filter(task => task.categoryId === this.currentEditingCategory.id);
                const item = categoryTasks.splice(draggedIndex, 1)[0];
                categoryTasks.splice(dropIndex, 0, item);
                
                // Update order values
                categoryTasks.forEach((task, index) => {
                    task.order = index;
                });
                
                this.renderTasksConfig();
            }
        });
        
        // Make elements draggable
        const items = container.querySelectorAll(`.${type}-config`);
        items.forEach(item => {
            item.draggable = true;
        });
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HabitTracker();
});