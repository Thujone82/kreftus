importScripts('NeuQuant.js');

function channelizePalette( palette ) {
    var channelizedPalette = [];

    for(var i = 0; i < palette.length; i++) {
        var color = palette[i];

        var r = (color & 0xFF0000) >> 16;
        var g = (color & 0x00FF00) >>  8;
        var b = (color & 0x0000FF);

        channelizedPalette.push([ r, g, b, color ]);
    }

    return channelizedPalette;

}


function dataToRGB( data, width, height ) {
    var i = 0;
    var length = width * height * 4;
    var rgb = [];

    while(i < length) {
        rgb.push( data[i++] );
        rgb.push( data[i++] );
        rgb.push( data[i++] );
        i++; // for the alpha channel which we don't care about
    }

    return rgb;
}


function componentizedPaletteToArray(paletteRGB) {

    var paletteArray = [];

    for(var i = 0; i < paletteRGB.length; i += 3) {
        var r = paletteRGB[ i ];
        var g = paletteRGB[ i + 1 ];
        var b = paletteRGB[ i + 2 ];
        paletteArray.push(r << 16 | g << 8 | b);
    }

    return paletteArray;
}


// This is the "traditional" Animated_GIF style of going from RGBA to indexed color frames
function processFrameWithQuantizer(imageData, width, height, sampleInterval) {

    var rgbComponents = dataToRGB( imageData, width, height );
    var nq = new NeuQuant(rgbComponents, rgbComponents.length, sampleInterval);
    var paletteRGB = nq.process();
    var paletteArray = new Uint32Array(componentizedPaletteToArray(paletteRGB));

    var numberPixels = width * height;
    var indexedPixels = new Uint8Array(numberPixels);

    var k = 0;
    for(var i = 0; i < numberPixels; i++) {
        r = rgbComponents[k++];
        g = rgbComponents[k++];
        b = rgbComponents[k++];
        indexedPixels[i] = nq.map(r, g, b);
    }

    return {
        pixels: indexedPixels,
        palette: paletteArray
    };

}





// ~~~

function run(frame) {
    var width = frame.width;
    var height = frame.height;
    var imageData = frame.data;
    var sampleInterval = frame.sampleInterval;

    return processFrameWithQuantizer(imageData, width, height, sampleInterval);
}


self.onmessage = function(ev) {
    var data = ev.data;
    var response = run(data);
    postMessage(response);
};
