var GIF = (function() {
  var GIF = function(options) {
    this.options = options || {};
    this.frames = [];
    this.freeWorkers = [];
    this.activeWorkers = [];
    this.imageParts = null;
    this.finishedFrames = 0;
    this.nextFrame = 0;
    this.running = false;
    this.options.workerScript = this.options.workerScript || 'gif.worker.js';
    this.options.workers = this.options.workers || 2;
    this.options.quality = this.options.quality || 10;
    this.options.repeat = this.options.repeat || 0;
    this.options.background = this.options.background || '#fff';
    this.options.transparent = this.options.transparent || null;
    this.options.width = this.options.width || null;
    this.options.height = this.options.height || null;
    this.options.dither = this.options.dither || false;
  };

  GIF.prototype.on = function(event, cb) {
    this.listeners = this.listeners || {};
    this.listeners[event] = this.listeners[event] || [];
    this.listeners[event].push(cb);
  };

  GIF.prototype.emit = function(event, arg) {
    if (this.listeners && this.listeners[event]) {
      this.listeners[event].forEach(function(cb) {
        cb(arg);
      });
    }
  };

  GIF.prototype.addFrame = function(image, options) {
    var frame = {};
    frame.transparent = this.options.transparent;
    for (var key in options) {
      frame[key] = options[key];
    }
    if (this.options.width === null) {
      this.options.width = image.width;
    }
    if (this.options.height === null) {
      this.options.height = image.height;
    }
    if (typeof image.getImageData === 'function') {
      frame.data = image.getImageData(0, 0, this.options.width, this.options.height).data;
    } else if (image.nodeName === 'CANVAS') {
      frame.data = image.getContext('2d').getImageData(0, 0, this.options.width, this.options.height).data;
    } else if (image.nodeName === 'IMG') {
      var canvas = document.createElement('canvas');
      canvas.width = this.options.width;
      canvas.height = this.options.height;
      var ctx = canvas.getContext('2d');
      ctx.drawImage(image, 0, 0);
      frame.data = ctx.getImageData(0, 0, this.options.width, this.options.height).data;
    } else {
      throw new Error('Invalid image');
    }
    this.frames.push(frame);
  };

  GIF.prototype.render = function() {
    if (this.running) {
      throw new Error('Already running');
    }
    if (this.options.width === null || this.options.height === null) {
      throw new Error('Width and height must be set prior to rendering');
    }
    this.running = true;
    this.nextFrame = 0;
    this.finishedFrames = 0;
    this.imageParts = new Array(this.frames.length);
    var numWorkers = this.spawnWorkers();
    for (var i = 0; i < numWorkers; i++) {
      this.renderNextFrame();
    }
    this.emit('start');
    this.emit('progress', 0);
  };

  GIF.prototype.abort = function() {
    var worker;
    while (true) {
      worker = this.activeWorkers.shift();
      if (worker == null) {
        break;
      }
      worker.terminate();
    }
    this.running = false;
    this.emit('abort');
  };

  GIF.prototype.spawnWorkers = function() {
    var numWorkers = Math.min(this.options.workers, this.frames.length);
    for (var i = this.freeWorkers.length; i < numWorkers; i++) {
      var worker = new Worker(this.options.workerScript);
      worker.onmessage = (function(e) {
        this.activeWorkers.splice(this.activeWorkers.indexOf(worker), 1);
        this.freeWorkers.push(worker);
        this.frameFinished(e.data);
      }).bind(this);
      this.freeWorkers.push(worker);
    }
    return numWorkers;
  };

  GIF.prototype.frameFinished = function(frame) {
    this.finishedFrames++;
    this.emit('progress', this.finishedFrames / this.frames.length);
    this.imageParts[frame.index] = frame;
    if (this.finishedFrames === this.frames.length) {
      this.finishRendering();
    } else {
      this.renderNextFrame();
    }
  };

  GIF.prototype.finishRendering = function() {
    var len = 0;
    for (var i = 0; i < this.imageParts.length; i++) {
      len += this.imageParts[i].data.length;
    }
    var data = new Uint8Array(len);
    var offset = 0;
    for (var i = 0; i < this.imageParts.length; i++) {
      data.set(this.imageParts[i].data, offset);
      offset += this.imageParts[i].data.length;
    }
    var image = new Blob([data], {type: 'image/gif'});
    this.emit('finished', image, data);
    this.running = false;
  };

  GIF.prototype.renderNextFrame = function() {
    if (this.freeWorkers.length === 0) {
      return;
    }
    if (this.nextFrame >= this.frames.length) {
      return;
    }
    var worker = this.freeWorkers.shift();
    var task = this.frames[this.nextFrame++];
    task.index = this.nextFrame - 1;
    task.last = this.nextFrame === this.frames.length;
    task.width = this.options.width;
    task.height = this.options.height;
    task.quality = this.options.quality;
    task.dither = this.options.dither;
    task.globalPalette = this.options.globalPalette;
    task.repeat = this.options.repeat;
    task.canTransfer = true;
    this.activeWorkers.push(worker);
    worker.postMessage(task);
  };

  return GIF;
})();
