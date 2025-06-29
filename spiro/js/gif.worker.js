var GIFEncoder = (function() {
  // private property
  var private_property = true;

  // constructor
  var GIFEncoder = function() {
    this.width = null;
    this.height = null;
    this.transparent = null;
    this.transIndex = 0;
    this.repeat = -1;
    this.delay = 0;
    this.image = null;
    this.pixels = null;
    this.indexedPixels = null;
    this.colorDepth = null;
    this.colorTab = null;
    this.neuQuant = null;
    this.usedEntry = new Array();
    this.palSize = 7;
    this.dispose = -1;
    this.firstFrame = true;
    this.sample = 10;
    this.dither = false;
    this.globalPalette = false;
    this.out = new ByteArray();
  };

  GIFEncoder.prototype.setDelay = function(milliseconds) {
    this.delay = Math.round(milliseconds / 10);
  };

  GIFEncoder.prototype.setFrameRate = function(fps) {
    this.delay = Math.round(100 / fps);
  };

  GIFEncoder.prototype.setDispose = function(disposalCode) {
    if (disposalCode >= 0) this.dispose = disposalCode;
  };

  GIFEncoder.prototype.setRepeat = function(repeat) {
    this.repeat = repeat;
  };

  GIFEncoder.prototype.setTransparent = function(color) {
    this.transparent = color;
  };

  GIFEncoder.prototype.addFrame = function(imageData) {
    this.image = imageData;
    this.colorTab = (this.globalPalette && this.globalPalette.slice) ? this.globalPalette : null;
    this.getImagePixels();
    this.analyzePixels();
    if (this.globalPalette === true) this.globalPalette = this.colorTab;
    if (this.firstFrame) {
      this.writeLSD();
      this.writePalette();
      if (this.repeat >= 0) {
        this.writeNetscapeExt();
      }
    }
    this.writeGraphicCtrlExt();
    this.writeImageDesc();
    if (!this.firstFrame && !this.globalPalette) this.writePalette();
    this.writePixels();
    this.firstFrame = false;
  };

  GIFEncoder.prototype.finish = function() {
    this.out.writeByte(0x3b); // gif trailer
  };

  GIFEncoder.prototype.setQuality = function(quality) {
    if (quality < 1) quality = 1;
    this.sample = quality;
  };

  GIFEncoder.prototype.setDither = function(dither) {
    if (dither === true) dither = 'FloydSteinberg';
    this.dither = dither;
  };

  GIFEncoder.prototype.setGlobalPalette = function(palette) {
    this.globalPalette = palette;
  };

  GIFEncoder.prototype.getGlobalPalette = function() {
    return (this.globalPalette && this.globalPalette.slice) ? this.globalPalette : null;
  };

  GIFEncoder.prototype.writeHeader = function() {
    this.out.writeUTFBytes('GIF89a');
  };

  GIFEncoder.prototype.analyzePixels = function() {
    if (!this.colorTab) {
      this.neuQuant = new NeuQuant(this.pixels, this.sample);
      this.neuQuant.buildColormap(); // create reduced palette
      this.colorTab = this.neuQuant.getColormap();
    }

    // map image pixels to new palette
    if (this.dither) {
      this.ditherPixels(this.dither.replace('-serpentine', ''), this.dither.match(/-serpentine/) !== null);
    } else {
      this.indexPixels();
    }

    this.pixels = null;
    this.colorDepth = 8;
    this.palSize = 7;

    // get closest match to transparent color if specified
    if (this.transparent !== null) {
      this.transIndex = this.findClosest(this.transparent, true);
    }
  };

  GIFEncoder.prototype.indexPixels = function(imgq) {
    var nPix = this.pixels.length / 3;
    this.indexedPixels = new Uint8Array(nPix);
    var k = 0;
    for (var j = 0; j < nPix; j++) {
      var index = this.findClosest(this.pixels[k++] & 0xff, this.pixels[k++] & 0xff, this.pixels[k++] & 0xff);
      this.usedEntry[index] = true;
      this.indexedPixels[j] = index;
    }
  };

  GIFEncoder.prototype.ditherPixels = function(kernel, serpentine) {
    var kernels = {
      'FloydSteinberg': [
        [7 / 16, 1, 0],
        [3 / 16, -1, 1],
        [5 / 16, 0, 1],
        [1 / 16, 1, 1]
      ],
      'FalseFloydSteinberg': [
        [3 / 8, 1, 0],
        [3 / 8, -1, 1],
        [2 / 8, 0, 1]
      ],
      'Stucki': [
        [8 / 42, 1, 0],
        [4 / 42, 2, 0],
        [2 / 42, -2, 1],
        [4 / 42, -1, 1],
        [8 / 42, 0, 1],
        [4 / 42, 1, 1],
        [2 / 42, 2, 1]
      ],
      'Atkinson': [
        [1 / 8, 1, 0],
        [1 / 8, 2, 0],
        [1 / 8, -1, 1],
        [1 / 8, 0, 1],
        [1 / 8, 1, 1],
        [1 / 8, 0, 2]
      ],
      'Jarvis': [
        [7 / 48, 1, 0],
        [5 / 48, 2, 0],
        [3 / 48, -2, 1],
        [5 / 48, -1, 1],
        [7 / 48, 0, 1],
        [5 / 48, 1, 1],
        [3 / 48, 2, 1],
        [1 / 48, -2, 2],
        [3 / 48, -1, 2],
        [5 / 48, 0, 2],
        [3 / 48, 1, 2],
        [1 / 48, 2, 2]
      ],
      'Burkes': [
        [8 / 32, 1, 0],
        [4 / 32, 2, 0],
        [2 / 32, -2, 1],
        [4 / 32, -1, 1],
        [8 / 32, 0, 1],
        [4 / 32, 1, 1],
        [2 / 32, 2, 1]
      ],
      'Sierra': [
        [5 / 32, 1, 0],
        [3 / 32, 2, 0],
        [2 / 32, -2, 1],
        [4 / 32, -1, 1],
        [5 / 32, 0, 1],
        [4 / 32, 1, 1],
        [2 / 32, 2, 1]
      ],
      'TwoRowSierra': [
        [4 / 16, 1, 0],
        [3 / 16, 2, 0],
        [1 / 16, -2, 1],
        [2 / 16, -1, 1],
        [3 / 16, 0, 1],
        [2 / 16, 1, 1],
        [1 / 16, 2, 1]
      ],
      'SierraLite': [
        [2 / 4, 1, 0],
        [1 / 4, -1, 1],
        [1 / 4, 0, 1]
      ]
    };

    if (!kernel || !kernels[kernel]) {
      throw 'Unknown dithering kernel: ' + kernel;
    }

    var selectedKernel = kernels[kernel];
    var nPix = this.pixels.length / 3;
    this.indexedPixels = new Uint8Array(nPix);

    var lum = new Uint8Array(nPix);
    for (var i = 0; i < nPix; i++) {
      lum[i] = Math.round(this.pixels[i * 3] * 0.299 + this.pixels[i * 3 + 1] * 0.587 + this.pixels[i * 3 + 2] * 0.114);
    }

    var data = new Int16Array(nPix * 3);
    for (var i = 0; i < nPix; i++) {
      data[i * 3] = this.pixels[i * 3];
      data[i * 3 + 1] = this.pixels[i * 3 + 1];
      data[i * 3 + 2] = this.pixels[i * 3 + 2];
    }

    var width = this.width;
    var height = this.height;
    var dir = serpentine ? -1 : 1;

    for (var y = 0; y < height; y++) {
      if (serpentine) dir *= -1;
      for (var x = (dir == 1 ? 0 : width - 1), x_stop = (dir == 1 ? width : -1); x !== x_stop; x += dir) {
        var idx = y * width + x;
        var r = data[idx * 3];
        var g = data[idx * 3 + 1];
        var b = data[idx * 3 + 2];

        var index = this.findClosest(r, g, b);
        this.usedEntry[index] = true;
        this.indexedPixels[idx] = index;

        var err_r = r - this.colorTab[index * 3];
        var err_g = g - this.colorTab[index * 3 + 1];
        var err_b = b - this.colorTab[index * 3 + 2];

        for (var j = (dir == 1 ? 0 : selectedKernel.length - 1), j_stop = (dir == 1 ? selectedKernel.length : -1); j !== j_stop; j += dir) {
          var k = selectedKernel[j];
          var err_idx = (y + k[2]) * width + (x + k[1]);
          if (err_idx >= 0 && err_idx < nPix) {
            data[err_idx * 3] += err_r * k[0];
            data[err_idx * 3 + 1] += err_g * k[0];
            data[err_idx * 3 + 2] += err_b * k[0];
          }
        }
      }
    }
  };

  GIFEncoder.prototype.findClosest = function(r, g, b, used) {
    return this.findClosestRGB(r, g, b, used);
  };

  GIFEncoder.prototype.findClosestRGB = function(r, g, b, used) {
    if (this.colorTab === null) return -1;
    if (this.neuQuant && !used) {
      return this.neuQuant.lookupRGB(r, g, b);
    }
    var c = b | (g << 8) | (r << 16);
    var minpos = 0;
    var dmin = 256 * 256 * 256;
    var len = this.colorTab.length;

    for (var i = 0, index = 0; i < len; index++) {
      var dr = r - (this.colorTab[i++] & 0xff);
      var dg = g - (this.colorTab[i++] & 0xff);
      var db = b - (this.colorTab[i] & 0xff);
      var d = dr * dr + dg * dg + db * db;
      if ((!used || this.usedEntry[index]) && (d < dmin)) {
        dmin = d;
        minpos = index;
      }
      i++;
    }
    return minpos;
  };

  GIFEncoder.prototype.getImagePixels = function() {
    var w = this.width;
    var h = this.height;
    this.pixels = new Uint8Array(w * h * 3);
    var data = this.image;
    var src = data.data;
    var dst = 0;
    var len = w * h * 4;
    for (var i = 0; i < len; i += 4) {
      this.pixels[dst++] = src[i];
      this.pixels[dst++] = src[i + 1];
      this.pixels[dst++] = src[i + 2];
    }
    if (this.globalPalette === true) {
      this.globalPalette = this.colorTab;
    }
  };

  GIFEncoder.prototype.writeGraphicCtrlExt = function() {
    this.out.writeByte(0x21); // extension introducer
    this.out.writeByte(0xf9); // GCE label
    this.out.writeByte(4); // data block size

    var transp, disp;
    if (this.transparent === null) {
      transp = 0;
      disp = 0; // dispose = no action
    } else {
      transp = 1;
      disp = 2; // force clear if using transparent color
    }

    if (this.dispose >= 0) {
      disp = this.dispose & 7; // user override
    }
    disp <<= 2;

    // packed fields
    this.out.writeByte(0 | // 1:3 reserved
      disp | // 4:6 disposal
      0 | // 7 user input - 0 = no
      transp); // 8 transparency flag

    this.writeShort(this.delay); // delay x 1/100 sec
    this.out.writeByte(this.transIndex); // transparent color index
    this.out.writeByte(0); // block terminator
  };

  GIFEncoder.prototype.writeImageDesc = function() {
    this.out.writeByte(0x2c); // image separator
    this.writeShort(0); // image position x,y = 0,0
    this.writeShort(0);
    this.writeShort(this.width); // image size
    this.writeShort(this.height);

    // packed fields
    if (this.firstFrame || this.globalPalette) {
      // no LCT - GCT is used for first (or only) frame
      this.out.writeByte(0);
    } else {
      // specify normal LCT
      this.out.writeByte(0x80 | // 1 local color table 1=yes
        0 | // 2 interlace - 0=no
        0 | // 3 sorted - 0=no
        0 | // 4-5 reserved
        this.palSize); // 6-8 size of LCT
    }
  };

  GIFEncoder.prototype.writeLSD = function() {
    // logical screen size
    this.writeShort(this.width);
    this.writeShort(this.height);

    // packed fields
    this.out.writeByte(0x80 | // 1 : global color table flag = 1 (gct used)
      0x70 | // 2-4 : color resolution = 7
      0x00 | // 5 : gct sort flag = 0
      this.palSize); // 6-8 : gct size

    this.out.writeByte(0); // background color index
    this.out.writeByte(0); // pixel aspect ratio - assume 1:1
  };

  GIFEncoder.prototype.writeNetscapeExt = function() {
    this.out.writeByte(0x21); // extension introducer
    this.out.writeByte(0xff); // app extension label
    this.out.writeByte(11); // block size
    this.out.writeUTFBytes('NETSCAPE2.0'); // app id + auth code
    this.out.writeByte(3); // sub-block size
    this.out.writeByte(1); // loop sub-block id
    this.writeShort(this.repeat); // loop count (extra iterations, 0=repeat forever)
    this.out.writeByte(0); // block terminator
  };

  GIFEncoder.prototype.writePalette = function() {
    this.out.writeBytes(this.colorTab);
    var n = (3 * 256) - this.colorTab.length;
    for (var i = 0; i < n; i++) this.out.writeByte(0);
  };

  GIFEncoder.prototype.writeShort = function(pValue) {
    this.out.writeByte(pValue & 0xFF);
    this.out.writeByte((pValue >> 8) & 0xFF);
  };

  GIFEncoder.prototype.writePixels = function() {
    var enc = new LZWEncoder(this.width, this.height, this.indexedPixels, this.colorDepth);
    enc.encode(this.out);
  };

  GIFEncoder.prototype.stream = function() {
    return this.out;
  };

  var NeuQuant = (function() {
    var ncycles = 100; // number of learning cycles
    var netsize = 256; // number of colors used
    var maxnetpos = netsize - 1;

    // defs for freq and bias
    var netbiasshift = 4; // bias for colour values
    var intbiasshift = 16; // bias for fractions
    var intbias = (1 << intbiasshift);
    var gammashift = 10;
    var gamma = (1 << gammashift);
    var betashift = 10;
    var beta = (intbias >> betashift); /* beta = 1/1024 */
    var betagamma = (intbias << (gammashift - betashift));

    // defs for decreasing radius factor
    var initrad = (netsize >> 3); // for 256 cols, radius starts
    var radiusbiasshift = 6; // at 32.0 biased by 6 bits
    var radiusbias = (1 << radiusbiasshift);
    var initradius = (initrad * radiusbias); // and decreases by a
    var radiusdec = 30; // factor of 1/30 each cycle

    // defs for decreasing alpha factor
    var alphabiasshift = 10; // alpha starts at 1.0
    var initalpha = (1 << alphabiasshift);
    var alphadec; // biased by 10 bits

    /* radbias and alpharadbias used for radpower calculation */
    var radbiasshift = 8;
    var radbias = (1 << radbiasshift);
    var alpharadbshift = (alphabiasshift + radbiasshift);
    var alpharadbias = (1 << alpharadbshift);

    var NeuQuant = function(pixels, samplefac) {
      var network; // int[netsize][4]
      var netindex; // for network lookup - really 256

      var bias; // bias and freq arrays for learning
      var freq;
      var radpower;

      this.pixels = pixels;
      this.samplefac = samplefac;

      this.network = network = new Array(netsize);
      this.netindex = netindex = new Int32Array(256);
      this.bias = bias = new Int32Array(netsize);
      this.freq = freq = new Int32Array(netsize);
      this.radpower = radpower = new Int32Array(netsize >> 3);

      var i, v;
      for (i = 0; i < netsize; i++) {
        v = (i << (netbiasshift + 8)) / netsize;
        network[i] = new Float64Array(4);
        network[i][0] = network[i][1] = network[i][2] = v;
        freq[i] = intbias / netsize;
        bias[i] = 0;
      }
    };

    NeuQuant.prototype.buildColormap = function() {
      this.learn();
      this.unbiasnet();
      this.inxbuild();
    };

    NeuQuant.prototype.learn = function() {
      var i, j, b, g, r;
      var lengthcount = this.pixels.length;
      var alphadec = 30 + ((this.samplefac - 1) / 3);
      var samplepixels = lengthcount / (3 * this.samplefac);
      var delta = ~~(samplepixels / ncycles);
      var alpha = initalpha;
      var radius = initradius;

      var rad = radius >> radiusbiasshift;
      if (rad <= 1) rad = 0;

      for (i = 0; i < rad; i++) {
        radpower[i] = alpha * (((rad * rad - i * i) * radbias) / (rad * rad));
      }

      var step;
      if (lengthcount < (1509 * 3)) {
        this.samplefac = 1;
        step = 3;
      } else if ((lengthcount % 499) !== 0) {
        step = 3 * 499;
      } else if ((lengthcount % 491) !== 0) {
        step = 3 * 491;
      } else if ((lengthcount % 487) !== 0) {
        step = 3 * 487;
      } else {
        step = 3 * 499;
      }

      var pix = this.pixels;
      var i = 0;
      var j = 0;
      var b = 0;
      var g = 0;
      var r = 0;

      while (i < samplepixels) {
        b = (pix[j] & 0xff) << netbiasshift;
        g = (pix[j + 1] & 0xff) << netbiasshift;
        r = (pix[j + 2] & 0xff) << netbiasshift;

        var p = this.contest(b, g, r);
        this.altersingle(alpha, p, b, g, r);
        if (rad !== 0) this.alterneigh(rad, p, b, g, r);

        j += step;
        if (j >= lengthcount) j -= lengthcount;

        i++;
        if (delta === 0) delta = 1;
        if (i % delta === 0) {
          alpha -= alpha / alphadec;
          radius -= radius / radiusdec;
          rad = radius >> radiusbiasshift;
          if (rad <= 1) rad = 0;
          for (p = 0; p < rad; p++) {
            radpower[p] = alpha * (((rad * rad - p * p) * radbias) / (rad * rad));
          }
        }
      }
    };

    NeuQuant.prototype.unbiasnet = function() {
      var i, j;
      for (i = 0; i < netsize; i++) {
        this.network[i][0] >>= netbiasshift;
        this.network[i][1] >>= netbiasshift;
        this.network[i][2] >>= netbiasshift;
        this.network[i][3] = i; // record color number
      }
    };

    NeuQuant.prototype.altersingle = function(alpha, i, b, g, r) {
      // Move neuron i towards sample color
      var n = this.network[i];
      n[0] -= (alpha * (n[0] - b)) / initalpha;
      n[1] -= (alpha * (n[1] - g)) / initalpha;
      n[2] -= (alpha * (n[2] - r)) / initalpha;
    };

    NeuQuant.prototype.alterneigh = function(rad, i, b, g, r) {
      var lo = Math.abs(i - rad);
      var hi = Math.min(i + rad, netsize);

      var j = i + 1;
      var k = i - 1;
      var m = 1;

      var p, a;
      while ((j < hi) || (k > lo)) {
        a = this.radpower[m++];
        if (j < hi) {
          p = this.network[j++];
          p[0] -= (a * (p[0] - b)) / alpharadbias;
          p[1] -= (a * (p[1] - g)) / alpharadbias;
          p[2] -= (a * (p[2] - r)) / alpharadbias;
        }
        if (k > lo) {
          p = this.network[k--];
          p[0] -= (a * (p[0] - b)) / alpharadbias;
          p[1] -= (a * (p[1] - g)) / alpharadbias;
          p[2] -= (a * (p[2] - r)) / alpharadbias;
        }
      }
    };

    NeuQuant.prototype.contest = function(b, g, r) {
      var bestd = ~(1 << 31);
      var bestbiasd = bestd;
      var bestpos = -1;
      var bestbiaspos = bestpos;

      var i, n, dist, biasdist, betafreq;
      for (i = 0; i < netsize; i++) {
        n = this.network[i];

        dist = n[0] - b;
        if (dist < 0) dist = -dist;
        var a = n[1] - g;
        if (a < 0) a = -a;
        dist += a;
        a = n[2] - r;
        if (a < 0) a = -a;
        dist += a;

        if (dist < bestd) {
          bestd = dist;
          bestpos = i;
        }

        biasdist = dist - ((this.bias[i]) >> (intbiasshift - netbiasshift));
        if (biasdist < bestbiasd) {
          bestbiasd = biasdist;
          bestbiaspos = i;
        }
        betafreq = (this.freq[i] >> betashift);
        this.freq[i] -= betafreq;
        this.bias[i] += (betafreq << gammashift);
      }
      this.freq[bestpos] += beta;
      this.bias[bestpos] -= betagamma;
      return bestpos;
    };

    NeuQuant.prototype.inxbuild = function() {
      var i, j, p, q, smallpos, smallval;
      var previouscol = 0;
      var startpos = 0;
      for (i = 0; i < netsize; i++) {
        p = this.network[i];
        smallpos = i;
        smallval = p[1]; // index on g
        // find smallest in i..netsize-1
        for (j = i + 1; j < netsize; j++) {
          q = this.network[j];
          if (q[1] < smallval) { // index on g
            smallpos = j;
            smallval = q[1];
          }
        }
        q = this.network[smallpos];
        // swap p (i) and q (smallpos) entries
        if (i != smallpos) {
          j = q[0];
          q[0] = p[0];
          p[0] = j;
          j = q[1];
          q[1] = p[1];
          p[1] = j;
          j = q[2];
          q[2] = p[2];
          p[2] = j;
          j = q[3];
          q[3] = p[3];
          p[3] = j;
        }
        // smallval entry is now in position i
        if (smallval != previouscol) {
          this.netindex[previouscol] = (startpos + i) >> 1;
          for (j = previouscol + 1; j < smallval; j++) this.netindex[j] = i;
          previouscol = smallval;
          startpos = i;
        }
      }
      this.netindex[previouscol] = (startpos + maxnetpos) >> 1;
      for (j = previouscol + 1; j < 256; j++) this.netindex[j] = maxnetpos; // really 256
    };

    NeuQuant.prototype.getColormap = function() {
      var map = new Uint8Array(netsize * 3);
      var index = new Uint8Array(netsize);
      for (var i = 0; i < netsize; i++) {
        index[this.network[i][3]] = i;
      }
      var k = 0;
      for (var l = 0; l < netsize; l++) {
        var j = index[l];
        map[k++] = this.network[j][0];
        map[k++] = this.network[j][1];
        map[k++] = this.network[j][2];
      }
      return map;
    };

    NeuQuant.prototype.lookupRGB = function(b, g, r) {
      var a, p, dist;
      var bestd = 1000; // biggest possible dist is 256*3
      var best = -1;
      var i = this.netindex[g]; // index on g
      var j = i - 1; // start at netindex[g] and work outwards

      while ((i < netsize) || (j >= 0)) {
        if (i < netsize) {
          p = this.network[i];
          dist = p[1] - g; // inx key
          if (dist >= bestd) i = netsize; // stop iter
          else {
            i++;
            if (dist < 0) dist = -dist;
            a = p[0] - b;
            if (a < 0) a = -a;
            dist += a;
            if (dist < bestd) {
              a = p[2] - r;
              if (a < 0) a = -a;
              dist += a;
              if (dist < bestd) {
                bestd = dist;
                best = p[3];
              }
            }
          }
        }
        if (j >= 0) {
          p = this.network[j];
          dist = g - p[1]; // inx key - reverse dif
          if (dist >= bestd) j = -1; // stop iter
          else {
            j--;
            if (dist < 0) dist = -dist;
            a = p[0] - b;
            if (a < 0) a = -a;
            dist += a;
            if (dist < bestd) {
              a = p[2] - r;
              if (a < 0) a = -a;
              dist += a;
              if (dist < bestd) {
                bestd = dist;
                best = p[3];
              }
            }
          }
        }
      }
      return best;
    };

    return NeuQuant;
  }());

  var LZWEncoder = (function() {
    var EOF = -1;
    var BITS = 12;
    var HSIZE = 5003; // 80% occupancy
    var masks = [0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767, 65535];

    var LZWEncoder = function(width, height, pixels, colorDepth) {
      var initCodeSize = Math.max(2, colorDepth);

      var accum = new Uint8Array(256);
      var htab = new Int32Array(HSIZE);
      var codetab = new Int32Array(HSIZE);

      var cur_accum, cur_bits = 0;
      var a_count;
      var free_ent = 0; // first unused entry
      var maxcode;

      // block compression parameters -- after all codes are used up,
      // and compression rate changes, start over.
      var clear_flg = false;

      // Algorithm: use open addressing double hashing (no chaining) on the
      // prefix code / next character combination. We do a variant of Knuth's
      // algorithm D (vol. 3, sec. 6.4) along with G. Knott's relatively-prime
      // secondary probe. Here, the modular division first probe is gives way
      // to a faster exclusive-or manipulation. Also do block compression with
      // an adaptive reset, whereby the code table is cleared when the compression
      // ratio decreases, but after the table fills. The variable-length output
      // codes are re-sized at this point, and a special CLEAR code is generated
      // for the decompressor. Late addition: construct the table according to
      // file size for small files. Also the stack simplifies maintenance.

      // output
      //
      // Output the given code.
      // Inputs:
      //      code:   A n_bits-bit integer. If == -1, then EOF. This assumes
      //              that n_bits =< wordsize - 1.
      // Outputs:
      //      Outputs code to the file.
      // Assumptions:
      //      Chars are 8 bits long.
      // Algorithm:
      //      Maintain a BITS character long buffer (so that 8 codes will
      // fit in it exactly). Use the VAX VMS standard names for the variables.
      //
      var g_init_bits, ClearCode, EOFCode;

      // Number of characters so far in this 'packet'
      var a_count;

      // Define the storage for the packet accumulator
      var accum = new Uint8Array(256);

      var char_out = function(c, outs) {
        accum[a_count++] = c;
        if (a_count >= 254) flush_char(outs);
      };

      var cl_block = function(outs) { // compress a block of pixels
        cl_hash(HSIZE);
        free_ent = ClearCode + 2;
        clear_flg = true;
        output(ClearCode, outs);
      };

      var cl_hash = function(hsize) { // reset code table
        for (var i = 0; i < hsize; ++i) htab[i] = -1;
      };

      var compress = function(init_bits, outs) {
        var fcode, c, i, ent, disp, hsize_reg, hshift;

        // Set up the globals: g_init_bits - initial number of bits
        g_init_bits = init_bits;

        // Set up the necessary values
        clear_flg = false;
        n_bits = g_init_bits;
        maxcode = MAXCODE(n_bits);

        ClearCode = 1 << (init_bits - 1);
        EOFCode = ClearCode + 1;
        free_ent = ClearCode + 2;

        a_count = 0; // clear packet

        ent = next_pixel();

        hshift = 0;
        for (fcode = HSIZE; fcode < 65536; fcode *= 2) ++hshift;
        hshift = 8 - hshift; // set hash code range bound

        hsize_reg = HSIZE;
        cl_hash(hsize_reg); // clear hash table

        output(ClearCode, outs);

        outer_loop: while ((c = next_pixel()) != EOF) {
          fcode = (c << BITS) + ent;
          i = (c << hshift) ^ ent; // xor hashing

          if (htab[i] == fcode) {
            ent = codetab[i];
            continue;
          } else if (htab[i] >= 0) { // non-empty slot
            disp = hsize_reg - i; // find secondary hash code
            if (i === 0) disp = 1;
            do {
              if ((i -= disp) < 0) i += hsize_reg;

              if (htab[i] == fcode) {
                ent = codetab[i];
                continue outer_loop;
              }
            } while (htab[i] >= 0);
          }

          output(ent, outs);
          ent = c;
          if (free_ent < (1 << BITS)) {
            codetab[i] = free_ent++; // code -> hashtable
            htab[i] = fcode;
          } else {
            cl_block(outs);
          }
        }
        // Put out the final code.
        output(ent, outs);
        output(EOFCode, outs);
      };

      // ----------------------------------------------------------------------------
      // Flush the packet to disk, and reset the accumulator
      var flush_char = function(outs) {
        if (a_count > 0) {
          outs.writeByte(a_count);
          outs.writeBytes(accum, 0, a_count);
          a_count = 0;
        }
      };

      var MAXCODE = function(n_bits) {
        return (1 << n_bits) - 1;
      };

      // ----------------------------------------------------------------------------
      // Return the next pixel from the image
      // ----------------------------------------------------------------------------
      var remaining = width * height;
      var curPixel = 0;

      var next_pixel = function() {
        if (remaining === 0) return EOF;

        --remaining;

        var pix = pixels[curPixel++];

        return pix & 0xff;
      };

      var output = function(code, outs) {
        cur_accum &= masks[cur_bits];

        if (cur_bits > 0) cur_accum |= (code << cur_bits);
        else cur_accum = code;

        cur_bits += n_bits;

        while (cur_bits >= 8) {
          char_out(cur_accum & 0xff, outs);
          cur_accum >>= 8;
          cur_bits -= 8;
        }

        // If the next entry is going to be too big for the code size,
        // then increase it, if possible.
        if (free_ent > maxcode || clear_flg) {
          if (clear_flg) {
            maxcode = MAXCODE(n_bits = g_init_bits);
            clear_flg = false;
          } else {
            ++n_bits;
            if (n_bits == BITS) maxcode = (1 << BITS);
            else maxcode = MAXCODE(n_bits);
          }
        }

        if (code == EOFCode) {
          // At EOF, write the rest of the buffer.
          while (cur_bits > 0) {
            char_out(cur_accum & 0xff, outs);
            cur_accum >>= 8;
            cur_bits -= 8;
          }

          flush_char(outs);
        }
      };

      this.encode = function(outs) {
        outs.writeByte(initCodeSize); // write "initial code size" byte
        remaining = width * height;
        curPixel = 0;
        compress(initCodeSize + 1, outs); // compress and write the pixel data
        outs.writeByte(0); // write block terminator
      };
    };

    var ByteArray = function() {
      this.page = -1;
      this.pages = [];
      this.newPage();
    };

    ByteArray.pageSize = 4096;
    ByteArray.charMap = {};
    for (var i = 0; i < 256; i++) {
      ByteArray.charMap[i] = String.fromCharCode(i);
    }

    ByteArray.prototype.newPage = function() {
      this.pages[++this.page] = new Uint8Array(ByteArray.pageSize);
      this.cursor = 0;
    };

    ByteArray.prototype.getData = function() {
      var rv = '';
      for (var i = 0; i < this.pages.length; i++) {
        rv += ByteArray.charMap[this.pages[i]];
      }
      return rv;
    };

    ByteArray.prototype.writeByte = function(val) {
      if (this.cursor >= ByteArray.pageSize) this.newPage();
      this.pages[this.page][this.cursor++] = val;
    };

    ByteArray.prototype.writeUTFBytes = function(string) {
      for (var l = string.length, i = 0; i < l; i++) {
        this.writeByte(string.charCodeAt(i));
      }
    };

    ByteArray.prototype.writeBytes = function(array, offset, length) {
      for (var l = length || array.length, i = offset || 0; i < l; i++) {
        this.writeByte(array[i]);
      }
    };

    return GIFEncoder;
  }());

  var renderFrame = function(frame) {
    var encoder = new GIFEncoder(frame.width, frame.height);
    if (frame.index === 0) {
      encoder.writeHeader();
    } else {
      encoder.firstFrame = false;
    }
    encoder.setTransparent(frame.transparent);
    encoder.setRepeat(frame.repeat);
    encoder.setDelay(frame.delay);
    encoder.setQuality(frame.quality);
    encoder.setDither(frame.dither);
    encoder.setGlobalPalette(frame.globalPalette);
    encoder.addFrame(frame.data);
    if (frame.last) {
      encoder.finish();
    }
    if (frame.globalPalette === true) {
      frame.globalPalette = encoder.getGlobalPalette();
    }
    var stream = encoder.stream();
    frame.data = stream.pages;
    self.postMessage(frame, [frame.data.buffer]);
  };

  self.onmessage = function(e) {
    renderFrame(e.data);
  };
})();