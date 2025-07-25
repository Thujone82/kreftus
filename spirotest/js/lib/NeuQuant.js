/*
* NeuQuant Neural-Net Quantization Algorithm
* ------------------------------------------
*
* Copyright (c) 1994 Anthony Dekker
*
* NEUQUANT Neural-Net quantization algorithm by Anthony Dekker, 1994. See
* "Kohonen neural networks for optimal colour quantization" in "Network:
* Computation in Neural Systems" Vol. 5 (1994) pp 351-367. for a discussion of
* the algorithm.
*
* Any party obtaining a copy of these files from the author, directly or
* indirectly, is granted, free of charge, a full and unrestricted irrevocable,
* world-wide, paid up, royalty-free, nonexclusive right and license to deal in
* this software and documentation files (the "Software"), including without
* limitation the rights to use, copy, modify, merge, publish, distribute,
* sublicense, and/or sell copies of the Software, and to permit persons who
* receive copies from any such party to do so, with the only requirement being
* that this copyright notice remain intact.
*/

/*
* This class handles Neural-Net quantization algorithm
* @author Kevin Weiner (original Java version - kweiner@fmsware.com)
* @author Thibault Imbert (AS3 version - bytearray.org)
* @version 0.1 AS3 implementation
* @version 0.2 JS->AS3 "translation" by antimatter15
* @version 0.3 JS clean up + using modern JS idioms by sole - http://soledadpenades.com
* Also implement fix in color conversion described at http://stackoverflow.com/questions/16371712/neuquant-js-javascript-color-quantization-hidden-bug-in-js-conversion
*/

var NeuQuant = (function() {
    function NeuQuant() {

    var netsize = 256; // number of colours used

    // four primes near 500 - assume no image has a length so large
    // that it is divisible by all four primes
    var prime1 = 499;
    var prime2 = 491;
    var prime3 = 487;
    var prime4 = 503;

    // minimum size for input image
    var minpicturebytes = (3 * prime4);

    // Network Definitions

    var maxnetpos = (netsize - 1);
    var netbiasshift = 4; // bias for colour values
    var ncycles = 100; // no. of learning cycles

    // defs for freq and bias
    var intbiasshift = 16; // bias for fractions
    var intbias = (1 << intbiasshift);
    var gammashift = 10; // gamma = 1024
    var gamma = (1 << gammashift);
    var betashift = 10;
    var beta = (intbias >> betashift); // beta = 1/1024
    var betagamma = (intbias << (gammashift - betashift));

    // defs for decreasing radius factor
    // For 256 colors, radius starts at 32.0 biased by 6 bits
    // and decreases by a factor of 1/30 each cycle
    var initrad = (netsize >> 3);
    var radiusbiasshift = 6;
    var radiusbias = (1 << radiusbiasshift);
    var initradius = (initrad * radiusbias);
    var radiusdec = 30;

    // defs for decreasing alpha factor
    // Alpha starts at 1.0 biased by 10 bits
    var alphabiasshift = 10;
    var initalpha = (1 << alphabiasshift);
    var alphadec;

    // radbias and alpharadbias used for radpower calculation
    var radbiasshift = 8;
    var radbias = (1 << radbiasshift);
    var alpharadbshift = (alphabiasshift + radbiasshift);
    var alpharadbias = (1 << alpharadbshift);


    // Input image
    var thepicture;
    // Height * Width * 3
    var lengthcount;
    // Sampling factor 1..30
    var samplefac;

    // The network itself
    var network;
    var netindex = [];

    // for network lookup - really 256
    var bias = [];

    // bias and freq arrays for learning
    var freq = [];
    var radpower = [];

    function NeuQuantConstructor(thepic, len, sample) {

        var i;
        var p;

        thepicture = thepic;
        lengthcount = len;
        samplefac = sample;

        network = new Array(netsize);

        for (i = 0; i < netsize; i++) {
            network[i] = new Array(4);
            p = network[i];
            p[0] = p[1] = p[2] = ((i << (netbiasshift + 8)) / netsize) | 0;
            freq[i] = (intbias / netsize) | 0; // 1 / netsize
            bias[i] = 0;
        }

    }

    function colorMap() {
        var map = [];
        var index = new Array(netsize);
        for (var i = 0; i < netsize; i++)
            index[network[i][3]] = i;
        var k = 0;
        for (var l = 0; l < netsize; l++) {
            var j = index[l];
            map[k++] = (network[j][0]);
            map[k++] = (network[j][1]);
            map[k++] = (network[j][2]);
        }
        return map;
    }

    // Insertion sort of network and building of netindex[0..255]
    // (to do after unbias)
    function inxbuild() {
        var i;
        var j;
        var smallpos;
        var smallval;
        var p;
        var q;
        var previouscol;
        var startpos;

        previouscol = 0;
        startpos = 0;

        for (i = 0; i < netsize; i++)
        {

            p = network[i];
            smallpos = i;
            smallval = p[1]; // index on g
            // find smallest in i..netsize-1
            for (j = i + 1; j < netsize; j++) {

                q = network[j];

                if (q[1] < smallval) { // index on g
                    smallpos = j;
                    smallval = q[1]; // index on g
                }
            }

            q = network[smallpos];

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

                netindex[previouscol] = (startpos + i) >> 1;

                for (j = previouscol + 1; j < smallval; j++) {
                    netindex[j] = i;
                }

                previouscol = smallval;
                startpos = i;

            }

        }

        netindex[previouscol] = (startpos + maxnetpos) >> 1;
        for (j = previouscol + 1; j < 256; j++) {
            netindex[j] = maxnetpos; // really 256
        }

    }


    // Main Learning Loop

    function learn() {
        var i;
        var j;
        var b;
        var g;
        var r;
        var radius;
        var rad;
        var alpha;
        var step;
        var delta;
        var samplepixels;
        var p;
        var pix;
        var lim;

        if (lengthcount < minpicturebytes) {
            samplefac = 1;
        }

        alphadec = 30 + ((samplefac - 1) / 3);
        p = thepicture;
        pix = 0;
        lim = lengthcount;
        samplepixels = lengthcount / (3 * samplefac);
        delta = (samplepixels / ncycles) | 0;
        alpha = initalpha;
        radius = initradius;

        rad = radius >> radiusbiasshift;
        if (rad <= 1) {
            rad = 0;
        }

        for (i = 0; i < rad; i++) {
            radpower[i] = alpha * (((rad * rad - i * i) * radbias) / (rad * rad));
        }


        if (lengthcount < minpicturebytes) {
            step = 3;
        } else if ((lengthcount % prime1) !== 0) {
            step = 3 * prime1;
        } else {

            if ((lengthcount % prime2) !== 0) {
                step = 3 * prime2;
            } else {
                if ((lengthcount % prime3) !== 0) {
                    step = 3 * prime3;
                } else {
                    step = 3 * prime4;
                }
            }

        }

        i = 0;

        while (i < samplepixels) {

            b = (p[pix + 0] & 0xff) << netbiasshift;
            g = (p[pix + 1] & 0xff) << netbiasshift;
            r = (p[pix + 2] & 0xff) << netbiasshift;
            j = contest(b, g, r);

            altersingle(alpha, j, b, g, r);

            if (rad !== 0) {
                // Alter neighbours
                alterneigh(rad, j, b, g, r);
            }

            pix += step;

            if (pix >= lim) {
                pix -= lengthcount;
            }

            i++;

            if (delta === 0) {
                delta = 1;
            }

            if (i % delta === 0) {
                alpha -= alpha / alphadec;
                radius -= radius / radiusdec;
                rad = radius >> radiusbiasshift;

                if (rad <= 1) {
                    rad = 0;
                }

                for (j = 0; j < rad; j++) {
                    radpower[j] = alpha * (((rad * rad - j * j) * radbias) / (rad * rad));
                }

            }

        }

    }

    // Search for BGR values 0..255 (after net is unbiased) and return colour index
    function map(b, g, r) {
        var i;
        var j;
        var dist;
        var a;
        var bestd;
        var p;
        var best;

        // Biggest possible distance is 256 * 3
        bestd = 1000;
        best = -1;
        i = netindex[g]; // index on g
        j = i - 1; // start at netindex[g] and work outwards

        while ((i < netsize) || (j >= 0)) {

            if (i < netsize) {

                p = network[i];

                dist = p[1] - g; // inx key

                if (dist >= bestd) {
                    i = netsize; // stop iter
                } else {

                    i++;

                    if (dist < 0) {
                        dist = -dist;
                    }

                    a = p[0] - b;

                    if (a < 0) {
                        a = -a;
                    }

                    dist += a;

                    if (dist < bestd) {
                        a = p[2] - r;

                        if (a < 0) {
                            a = -a;
                        }

                        dist += a;

                        if (dist < bestd) {
                            bestd = dist;
                            best = p[3];
                        }
                    }

                }

            }

            if (j >= 0) {

                p = network[j];

                dist = g - p[1]; // inx key - reverse dif

                if (dist >= bestd) {
                    j = -1; // stop iter
                } else {

                    j--;
                    if (dist < 0) {
                        dist = -dist;
                    }
                    a = p[0] - b;
                    if (a < 0) {
                        a = -a;
                    }
                    dist += a;

                    if (dist < bestd) {
                        a = p[2] - r;
                        if (a < 0) {
                            a = -a;
                        }
                        dist += a;
                        if (dist < bestd) {
                            bestd = dist;
                            best = p[3];
                        }
                    }

                }

            }

        }

        return (best);

    }

    function process() {
        learn();
        unbiasnet();
        inxbuild();
        return colorMap();
    }

    // Unbias network to give byte values 0..255 and record position i
    // to prepare for sort
    function unbiasnet() {
        var i;
        var j;

        for (i = 0; i < netsize; i++) {
            network[i][0] >>= netbiasshift;
            network[i][1] >>= netbiasshift;
            network[i][2] >>= netbiasshift;
            network[i][3] = i; // record colour no
        }
    }

    // Move adjacent neurons by precomputed alpha*(1-((i-j)^2/[r]^2))
    // in radpower[|i-j|]
    function alterneigh(rad, i, b, g, r) {

        var j;
        var k;
        var lo;
        var hi;
        var a;
        var m;

        var p;

        lo = i - rad;
        if (lo < -1) {
            lo = -1;
        }

        hi = i + rad;

        if (hi > netsize) {
            hi = netsize;
        }

        j = i + 1;
        k = i - 1;
        m = 1;

        while ((j < hi) || (k > lo)) {

            a = radpower[m++];

            if (j < hi) {

                p = network[j++];

                try {

                    p[0] -= ((a * (p[0] - b)) / alpharadbias) | 0;
                    p[1] -= ((a * (p[1] - g)) / alpharadbias) | 0;
                    p[2] -= ((a * (p[2] - r)) / alpharadbias) | 0;

                } catch (e) {}

            }

            if (k > lo) {

                p = network[k--];

                try {

                    p[0] -= ((a * (p[0] - b)) / alpharadbias) | 0;
                    p[1] -= ((a * (p[1] - g)) / alpharadbias) | 0;
                    p[2] -= ((a * (p[2] - r)) / alpharadbias) | 0;

                } catch (e) {}

            }

        }

    }


    // Move neuron i towards biased (b,g,r) by factor alpha
    function altersingle(alpha, i, b, g, r) {

        // alter hit neuron
        var n = network[i];
        var alphaMult = alpha / initalpha;
        n[0] -= ((alphaMult * (n[0] - b))) | 0;
        n[1] -= ((alphaMult * (n[1] - g))) | 0;
        n[2] -= ((alphaMult * (n[2] - r))) | 0;

    }

    // Search for biased BGR values
    function contest(b, g, r) {

        // finds closest neuron (min dist) and updates freq
        // finds best neuron (min dist-bias) and returns position
        // for frequently chosen neurons, freq[i] is high and bias[i] is negative
        // bias[i] = gamma*((1/netsize)-freq[i])

        var i;
        var dist;
        var a;
        var biasdist;
        var betafreq;
        var bestpos;
        var bestbiaspos;
        var bestd;
        var bestbiasd;
        var n;

        bestd = ~(1 << 31);
        bestbiasd = bestd;
        bestpos = -1;
        bestbiaspos = bestpos;

        for (i = 0; i < netsize; i++) {

            n = network[i];
            dist = n[0] - b;

            if (dist < 0) {
                dist = -dist;
            }

            a = n[1] - g;

            if (a < 0) {
                a = -a;
            }

            dist += a;

            a = n[2] - r;

            if (a < 0) {
                a = -a;
            }

            dist += a;

            if (dist < bestd) {
                bestd = dist;
                bestpos = i;
            }

            biasdist = dist - ((bias[i]) >> (intbiasshift - netbiasshift));

            if (biasdist < bestbiasd) {
                bestbiasd = biasdist;
                bestbiaspos = i;
            }

            betafreq = (freq[i] >> betashift);
            freq[i] -= betafreq;
            bias[i] += (betafreq << gammashift);

        }

        freq[bestpos] += beta;
        bias[bestpos] -= betagamma;
        return (bestbiaspos);

    }

    NeuQuantConstructor.apply(this, arguments);

    var exports = {};
    exports.map = map;
    exports.process = process;

    return exports;
    }
    return NeuQuant;
})();