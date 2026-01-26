# Map Download Tool

A web-based tool for downloading high-resolution map images from Google Maps. Search for locations, draw bounding boxes, select zoom levels, and download stitched map tiles as PNG images.

## Features

### Core Functionality
- **Location Search**: Search for any location worldwide using Google Places Autocomplete
- **Interactive Map**: Full Google Maps integration with pan, zoom, and map type selection
- **Bounding Box Drawing**: Draw rectangular selection areas on the map to define download regions
- **Zoom Control**: Adjustable zoom level (1-20) with real-time slider control
- **Map Type Selection**: Choose from Roadmap, Satellite, Hybrid, or Terrain map types
- **Current Location**: Quick access button (📍) to center map on your current GPS location at zoom 14
- **High-Resolution Downloads**: Downloads map images at full resolution with automatic tiling for large areas

### Technical Features
- **Automatic Tiling**: Large map areas are automatically split into multiple tiles and stitched together
- **Resolution Preservation**: Uses Google Static Maps API with `scale=2` for 1280x1280 pixel tiles
- **Mercator Projection**: Accurate coordinate conversion using Mercator projection calculations
- **Smart Cropping**: Extracts precise geographic areas from downloaded tiles with margin for detail preservation
- **Canvas Stitching**: Seamlessly combines multiple tiles into a single high-resolution image
- **Local Storage**: API key is saved in browser localStorage for convenience

## Requirements

- Modern web browser (Chrome, Firefox, Safari, Edge)
- Google Maps API key with the following APIs enabled:
  - Maps JavaScript API
  - Places API (for search/autocomplete)
  - Static Maps API (for image downloads)
  - Drawing Library (deprecated but still functional)

## Setup

### 1. Get a Google Maps API Key

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the required APIs:
   - Maps JavaScript API
   - Places API
   - Static Maps API
4. Create credentials (API Key)
5. (Optional) Restrict the API key to specific APIs and HTTP referrers for security

### 2. Configure the Tool

1. Open `index.html` in a web browser
2. Enter your Google Maps API key in the input field at the top
3. Click "Save Key" - the key will be stored in localStorage
4. The map will automatically load

## Usage

### Basic Workflow

1. **Set API Key** (first time only)
   - Enter your Google Maps API key in the header
   - Click "Save Key"

2. **Search for Location**
   - Type a location in the search box
   - Select from autocomplete suggestions
   - Or click the 📍 button to use your current location

3. **Adjust View**
   - Use the zoom slider to set desired zoom level (1-20)
   - Change map type using the dropdown (Roadmap, Satellite, Hybrid, Terrain)
   - Pan the map to find your area of interest

4. **Draw Bounding Box**
   - Click "Enable Drawing"
   - Click and drag on the map to draw a rectangle
   - The bounding box will appear in green
   - Only one bounding box can exist at a time (new drawings clear previous ones)

5. **Download Map**
   - Click "Download Map" button
   - A loading overlay will appear
   - The map image will download as a PNG file
   - File name format: `map_[timestamp].png`

### Advanced Features

#### Current Location Button (📍)
- Centers the map on your current GPS location
- Sets zoom level to 14
- Requires browser location permissions
- Falls back to IP-based location if GPS is unavailable

#### Zoom Levels
- **1-5**: World/continent view
- **6-10**: Country/state view
- **11-15**: City/neighborhood view
- **16-20**: Street/building detail view

#### Map Types
- **Roadmap**: Standard street map with labels
- **Satellite**: Aerial/satellite imagery
- **Hybrid**: Satellite imagery with road labels
- **Terrain**: Topographic map with elevation

## Technical Details

### Tiling System

For large map areas, the tool automatically:
1. Calculates if the area exceeds Google Static Maps API limits (640x640 base, 1280x1280 with scale=2)
2. Divides the bounding box into a grid of tiles
3. Downloads each tile from Google Static Maps API
4. Extracts the precise geographic portion from each tile
5. Stitches tiles together on an HTML5 canvas
6. Exports the final image as a PNG

### Resolution Handling

- **Single Tile**: Areas that fit in one 1280x1280 tile are downloaded directly
- **Multiple Tiles**: Large areas are split into a grid (e.g., 4x3 tiles)
- **Extraction**: Each tile extracts a portion with 50% margin to preserve detail
- **Stitching**: Tiles are precisely aligned using world coordinate calculations
- **Output**: Final image maintains full resolution of extracted portions

### Coordinate System

The tool uses:
- **Geographic Coordinates**: Latitude/longitude for user input and display
- **World Coordinates**: Mercator projection coordinates for calculations
- **Pixel Coordinates**: Canvas pixel positions for rendering
- **Conversion Functions**: `latToY()` and `yToLat()` for accurate Mercator projection

### API Usage

- **Maps JavaScript API**: Interactive map display, search, drawing
- **Places API**: Location search and autocomplete
- **Static Maps API**: High-resolution map image downloads
- **Drawing Library**: Rectangle drawing overlay (deprecated but functional)

## Browser Compatibility

- ✅ Chrome/Edge (recommended)
- ✅ Firefox
- ✅ Safari
- ✅ Mobile browsers (iOS Safari, Chrome Mobile)

**Note**: Requires modern JavaScript features (ES6+), Canvas API, and localStorage support.

## Limitations

1. **API Quotas**: Google Maps API has usage limits based on your billing plan
2. **Static Maps Limits**: Maximum 640x640 pixels per request (1280x1280 with scale=2)
3. **Large Downloads**: Very large areas may take significant time to download and stitch
4. **Drawing Library**: Google's Drawing Library is deprecated (scheduled for removal in May 2026)
5. **CORS**: Static Maps images require proper CORS headers (handled automatically by Google)

## File Structure

```
map/
├── index.html          # Main application file (all-in-one)
└── README.md          # This file
```

The tool is a single-file application containing:
- HTML structure
- CSS styling (dark theme)
- JavaScript functionality
- Google Maps integration
- Canvas-based image processing

## Troubleshooting

### Map Not Loading
- Verify API key is correct and saved
- Check that Maps JavaScript API is enabled
- Check browser console for error messages
- Ensure API key has proper restrictions (if any)

### Download Fails
- Verify Static Maps API is enabled
- Check API quota/billing status
- Ensure bounding box is drawn before downloading
- Check browser console for detailed error messages

### Images Are Misaligned
- This should be resolved in current version
- If issues persist, check browser console logs
- Verify zoom level is appropriate for the area size

### Current Location Not Working
- Grant browser location permissions
- Ensure HTTPS connection (required for geolocation)
- Check if GPS/location services are enabled on device

## Development Notes

### Key Functions

- `initMap()`: Initializes Google Maps and drawing tools
- `downloadMap()`: Main download orchestrator
- `downloadSingleTile()`: Handles single-tile downloads
- `downloadTiledMap()`: Handles multi-tile downloads and stitching
- `latToY()` / `yToLat()`: Mercator projection conversions
- `loadImage()`: Async image loading with CORS support

### Coordinate Calculations

The tool uses precise Mercator projection calculations:
- World size = `2^zoom * 256` pixels
- Latitude to Y: `Math.log(Math.tan((lat + 90) * Math.PI / 360)) / Math.PI * 180`
- Y to Latitude: Inverse calculation for accurate conversion

## License

This tool is provided as-is for personal and educational use. Google Maps API usage is subject to Google's Terms of Service.

## Credits

- Built with Google Maps JavaScript API
- Uses Google Static Maps API for image downloads
- Canvas API for image stitching and processing
