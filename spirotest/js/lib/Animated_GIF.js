var Animated_GIF = (function() {
    'use strict';

    function GifWriter(buf, width, height, gopts) {
      var p = 0;

      var gopts = gopts === undefined ? { } : gopts;
      var loop_count = gopts.loop === undefined ? null : gopts.loop;
      var global_palette = gopts.palette === undefined ? null : gopts.palette;

      if (width <= 0 || height <= 0 || width > 65535 || height > 65535)
        throw new Error("Width/Height invalid.");

      function check_palette_and_num_colors(palette) {
        var num_colors = palette.length;
        if (num_colors < 2 || num_colors > 256 ||  num_colors & (num_colors-1)) {
          throw new Error(
              "Invalid code/color length, must be power of 2 and 2 .. 256.");
        }
        return num_colors;
      }

      // - Header.
      buf[p++] = 0x47; buf[p++] = 0x49; buf[p++] = 0x46;  // GIF
      buf[p++] = 0x38; buf[p++] = 0x39; buf[p++] = 0x61;  // 89a

      // Handling of Global Color Table (palette) and background index.
      var gp_num_colors_pow2 = 0;
      var background = 0;
      if (global_palette !== null) {
        var gp_num_colors = check_palette_and_num_colors(global_palette);
        while (gp_num_colors >>= 1) ++gp_num_colors_pow2;
        gp_num_colors = 1 << gp_num_colors_pow2;
        --gp_num_colors_pow2;
        if (gopts.background !== undefined) {
          background = gopts.background;
          if (background >= gp_num_colors)
            throw new Error("Background index out of range.");
          if (background === 0)
            throw new Error("Background index explicitly passed as 0.");
        }
      }

      // - Logical Screen Descriptor.
      buf[p++] = width & 0xff; buf[p++] = width >> 8 & 0xff;
      buf[p++] = height & 0xff; buf[p++] = height >> 8 & 0xff;
      buf[p++] = (global_palette !== null ? 0x80 : 0) |
                 gp_num_colors_pow2;
      buf[p++] = background;
      buf[p++] = 0;

      // - Global Color Table
      if (global_palette !== null) {
        for (var i = 0, il = global_palette.length; i < il; ++i) {
          var rgb = global_palette[i];
          buf[p++] = rgb >> 16 & 0xff;
          buf[p++] = rgb >> 8 & 0xff;
          buf[p++] = rgb & 0xff;
        }
      }

      if (loop_count !== null) {
        if (loop_count < 0 || loop_count > 65535)
          throw new Error("Loop count invalid.")
        buf[p++] = 0x21; buf[p++] = 0xff; buf[p++] = 0x0b;
        buf[p++] = 0x4e; buf[p++] = 0x45; buf[p++] = 0x54; buf[p++] = 0x53;
        buf[p++] = 0x43; buf[p++] = 0x41; buf[p++] = 0x50; buf[p++] = 0x45;
        buf[p++] = 0x32; buf[p++] = 0x2e; buf[p++] = 0x30;
        buf[p++] = 0x03; buf[p++] = 0x01;
        buf[p++] = loop_count & 0xff; buf[p++] = loop_count >> 8 & 0xff;
        buf[p++] = 0x00;
      }


      var ended = false;

      this.addFrame = function(x, y, w, h, indexed_pixels, opts) {
        if (ended === true) { --p; ended = false; }

        opts = opts === undefined ? { } : opts;

        if (x < 0 || y < 0 || x > 65535 || y > 65535)
          throw new Error("x/y invalid.")

        if (w <= 0 || h <= 0 || w > 65535 || h > 65535)
          throw new Error("Width/Height invalid.")

        if (indexed_pixels.length < w * h)
          throw new Error("Not enough pixels for the frame size.");

        var using_local_palette = true;
        var palette = opts.palette;
        if (palette === undefined || palette === null) {
          using_local_palette = false;
          palette = global_palette;
        }

        if (palette === undefined || palette === null)
          throw new Error("Must supply either a local or global palette.");

        var num_colors = check_palette_and_num_colors(palette);

        var min_code_size = 0;
        while (num_colors >>= 1) ++min_code_size;
        num_colors = 1 << min_code_size;

        var delay = opts.delay === undefined ? 0 : opts.delay;

        var disposal = opts.disposal === undefined ? 0 : opts.disposal;
        if (disposal < 0 || disposal > 3)
          throw new Error("Disposal out of range.");

        var use_transparency = false;
        var transparent_index = 0;
        if (opts.transparent !== undefined && opts.transparent !== null) {
          use_transparency = true;
          transparent_index = opts.transparent;
          if (transparent_index < 0 || transparent_index >= num_colors)
            throw new Error("Transparent color index.");
        }

        if (disposal !== 0 || use_transparency || delay !== 0) {
          buf[p++] = 0x21; buf[p++] = 0xf9;
          buf[p++] = 4;

          buf[p++] = disposal << 2 | (use_transparency === true ? 1 : 0);
          buf[p++] = delay & 0xff; buf[p++] = delay >> 8 & 0xff;
          buf[p++] = transparent_index;
          buf[p++] = 0;
        }

        buf[p++] = 0x2c;
        buf[p++] = x & 0xff; buf[p++] = x >> 8 & 0xff;
        buf[p++] = y & 0xff; buf[p++] = y >> 8 & 0xff;
        buf[p++] = w & 0xff; buf[p++] = w >> 8 & 0xff;
        buf[p++] = h & 0xff; buf[p++] = h >> 8 & 0xff;
        buf[p++] = using_local_palette === true ? (0x80 | (min_code_size-1)) : 0;

        if (using_local_palette === true) {
          for (var i = 0, il = palette.length; i < il; ++i) {
            var rgb = palette[i];
            buf[p++] = rgb >> 16 & 0xff;
            buf[p++] = rgb >> 8 & 0xff;
            buf[p++] = rgb & 0xff;
          }
        }

        p = GifWriterOutputLZWCodeStream(
                buf, p, min_code_size < 2 ? 2 : min_code_size, indexed_pixels);

        return p;
      };

      this.end = function() {
        if (ended === false) {
          buf[p++] = 0x3b;
          ended = true;
        }
        return p;
      };

      this.getOutputBuffer = function() { return buf; };
      this.setOutputBuffer = function(v) { buf = v; };
      this.getOutputBufferPosition = function() { return p; };
      this.setOutputBufferPosition = function(v) { p = v; };
    }

    function GifWriterOutputLZWCodeStream(buf, p, min_code_size, index_stream) {
      buf[p++] = min_code_size;
      var cur_subblock = p++;

      var clear_code = 1 << min_code_size;
      var code_mask = clear_code - 1;
      var eoi_code = clear_code + 1;
      var next_code = eoi_code + 1;

      var cur_code_size = min_code_size + 1;
      var cur_shift = 0;
      var cur = 0;

      function emit_bytes_to_buffer(bit_block_size) {
        while (cur_shift >= bit_block_size) {
          buf[p++] = cur & 0xff;
          cur >>= 8; cur_shift -= 8;
          if (p === cur_subblock + 256) {
            buf[cur_subblock] = 255;
            cur_subblock = p++;
          }
        }
      }

      function emit_code(c) {
        cur |= c << cur_shift;
        cur_shift += cur_code_size;
        emit_bytes_to_buffer(8);
      }

      var ib_code = index_stream[0] & code_mask;
      var code_table = { };

      emit_code(clear_code);

      for (var i = 1, il = index_stream.length; i < il; ++i) {
        var k = index_stream[i] & code_mask;
        var cur_key = ib_code << 8 | k;
        var cur_code = code_table[cur_key];

        if (cur_code === undefined) {
          cur |= ib_code << cur_shift;
          cur_shift += cur_code_size;
          while (cur_shift >= 8) {
            buf[p++] = cur & 0xff;
            cur >>= 8; cur_shift -= 8;
            if (p === cur_subblock + 256) {
              buf[cur_subblock] = 255;
              cur_subblock = p++;
            }
          }

          if (next_code === 4096) {
            emit_code(clear_code);
            next_code = eoi_code + 1;
            cur_code_size = min_code_size + 1;
            code_table = { };
          } else {
            if (next_code >= (1 << cur_code_size)) ++cur_code_size;
            code_table[cur_key] = next_code++;
          }

          ib_code = k;
        } else {
          ib_code = cur_code;
        }
      }

      emit_code(ib_code);
      emit_code(eoi_code);

      emit_bytes_to_buffer(1);

      if (cur_subblock + 1 === p) {
        buf[cur_subblock] = 0;
      } else {
        buf[cur_subblock] = p - cur_subblock - 1;
        buf[p++] = 0;
      }
      return p;
    }

    function Animated_GIF(options) {
        options = options || {};

        var width = options.width || 160;
        var height = options.height || 120;
        var dithering = options.dithering || null;
        var palette = options.palette || null;
        var delay = options.delay !== undefined ? (options.delay * 0.1) : 250;
        var canvas = null, ctx = null, repeat = 0;
        var frames = [];
        var numRenderedFrames = 0;
        var onRenderCompleteCallback = function() {};
        var onRenderProgressCallback = function() {};
        var sampleInterval;
        var workers = [], availableWorkers = [], numWorkers;
        var generatingGIF = false;

        if(palette) {

            if(!(palette instanceof Array)) {

                throw('Palette MUST be an array but it is: ', palette);

            } else {

                if(palette.length < 2 || palette.length > 256) {
                    console.error('Palette must hold only between 2 and 256 colours');

                    while(palette.length < 2) {
                        palette.push(0x000000);
                    }

                    if(palette.length > 256) {
                        palette = palette.slice(0, 256);
                    }
                }

                if(!powerOfTwo(palette.length)) {
                    console.error('Palette must have a power of two number of colours');

                    while(!powerOfTwo(palette.length)) {
                        palette.splice(palette.length - 1, 1);
                    }
                }

            }

        }

        options = options || {};
        sampleInterval = options.sampleInterval || 10;
        numWorkers = options.numWorkers || 2;

        for(var i = 0; i < numWorkers; i++) {
            var w = new Worker('js/lib/Animated_GIF.worker.js');
            workers.push(w);
            availableWorkers.push(w);
        }

        function getWorker() {
            if(availableWorkers.length === 0) {
                throw ('No workers left!');
            }

            return availableWorkers.pop();
        }

        function freeWorker(worker) {
            availableWorkers.push(worker);
        }

        var bufferToString = (function() {
            var byteMap = [];
            for(var i = 0; i < 256; i++) {
                byteMap[i] = String.fromCharCode(i);
            }

            return (function(buffer) {
                var numberValues = buffer.length;
                var str = '';

                for(var i = 0; i < numberValues; i++) {
                    str += byteMap[ buffer[i] ];
                }

                return str;
            });
        })();

        function startRendering(completeCallback) {
            var numFrames = frames.length;

            onRenderCompleteCallback = completeCallback;

            for(var i = 0; i < numWorkers && i < frames.length; i++) {
                processFrame(i);
            }
        }

        function processFrame(position) {
            var frame;
            var worker;

            frame = frames[position];

            if(frame.beingProcessed || frame.done) {
                console.error('Frame already being processed or done!', frame.position);
                onFrameFinished();
                return;
            }

            frame.sampleInterval = sampleInterval;
            frame.beingProcessed = true;

            worker = getWorker();

            worker.onmessage = function(ev) {
                var data = ev.data;

                delete(frame.data);

                frame.pixels = Array.prototype.slice.call(data.pixels);
                frame.palette = Array.prototype.slice.call(data.palette);
                frame.done = true;
                frame.beingProcessed = false;

                freeWorker(worker);

                onFrameFinished();
            };

            worker.postMessage(frame);
        }

        function processNextFrame() {

            var position = -1;

            for(var i = 0; i < frames.length; i++) {
                var frame = frames[i];
                if(!frame.done && !frame.beingProcessed) {
                    position = i;
                    break;
                }
            }

            if(position >= 0) {
                processFrame(position);
            }
        }


        function onFrameFinished() {

            var allDone = frames.every(function(frame) {
                return !frame.beingProcessed && frame.done;
            });

            numRenderedFrames++;
            onRenderProgressCallback(numRenderedFrames * 0.75 / frames.length);

            if(allDone) {
                if(!generatingGIF) {
                    generateGIF(frames, onRenderCompleteCallback);
                }
            } else {
                setTimeout(processNextFrame, 1);
            }

        }

        function generateGIF(frames, callback) {

            var buffer = [];
            var globalPalette;
            var gifOptions = { loop: repeat };

            if(dithering !== null && palette !== null) {
                globalPalette = palette;
                gifOptions.palette = globalPalette;
            }

            var gifWriter = new GifWriter(buffer, width, height, gifOptions);

            generatingGIF = true;

            frames.forEach(function(frame, index) {

                var framePalette;

                if(!globalPalette) {
                   framePalette = frame.palette;
                }

                onRenderProgressCallback(0.75 + 0.25 * frame.position * 1.0 / frames.length);
                gifWriter.addFrame(0, 0, width, height, frame.pixels, {
                    palette: framePalette,
                    delay: frame.delay,
                });
            });

            gifWriter.end();
            onRenderProgressCallback(1.0);

            frames = [];
            generatingGIF = false;

            callback(buffer);
        }


        function powerOfTwo(value) {
            return (value !== 0) && ((value & (value - 1)) === 0);
        }

        this.setSize = function(w, h) {
            width = w;
            height = h;
            canvas = document.createElement('canvas');
            canvas.width = w;
            canvas.height = h;
            ctx = canvas.getContext('2d');
        };

        this.setDelay = function(seconds) {
            delay = seconds * 0.1;
        };

        this.setRepeat = function(r) {
            repeat = r;
        };

        this.addFrame = function(element, opts) {

            if(ctx === null) {
                this.setSize(width, height);
            }

            ctx.drawImage(element, 0, 0, width, height);
            var imageData = ctx.getImageData(0, 0, width, height);

            this.addFrameImageData(imageData, opts);
        };

        this.addFrameImageData = function(imageData, opts) {
            opts = opts || {};

            var dataLength = imageData.length,
                imageDataArray = new Uint8Array(imageData.data);

            frames.push({
                data: imageDataArray,
                width: imageData.width,
                height: imageData.height,
                delay: opts.delay !== undefined ? (opts.delay * 0.1) : delay,
                palette: palette,
                dithering: dithering,
                done: false,
                beingProcessed: false,
                position: frames.length
            });
        };

        this.onRenderProgress = function(callback) {
            onRenderProgressCallback = callback;
        };

        this.isRendering = function() {
            return generatingGIF;
        };

        this.getBase64GIF = function(completeCallback) {

            var onRenderComplete = function(buffer) {
                var str = bufferToString(buffer);
                var gif = 'data:image/gif;base64,' + btoa(str);
                completeCallback(gif);
            };

            startRendering(onRenderComplete);

        };


        this.getBlobGIF = function(completeCallback) {

            var onRenderComplete = function(buffer) {
                var array = new Uint8Array(buffer);
                var blob = new Blob([ array ], { type: 'image/gif' });
                completeCallback(blob);
            };

            startRendering(onRenderComplete);

        };

        this.destroy = function() {

            workers.forEach(function(w) {
                w.terminate();
            });

        };

    }

    return Animated_GIF;
})();