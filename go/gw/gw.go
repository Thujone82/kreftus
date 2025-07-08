package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v3/process"
	"gopkg.in/ini.v1"
)

const (
	appName            = "gw"     // Changed from "goweather"
	configFileName     = "gw.ini" // More specific name
	defaultApiSection  = "openweathermap"
	defaultApiKeyName  = "apikey"
	defaultPermissions = 0600 // Read/write for user only for config file

	geoZipURL    = "http://api.openweathermap.org/geo/1.0/zip"
	geoDirectURL = "http://api.openweathermap.org/geo/1.0/direct"
	oneCallURL   = "https://api.openweathermap.org/data/3.0/onecall"
	overviewURL  = "https://api.openweathermap.org/data/3.0/onecall/overview"
)

var (
	// Colors - attempting to match PowerShell intent
	colorAlert   = color.New(color.FgRed)
	colorTitle   = color.New(color.FgGreen)
	colorInfo    = color.New(color.FgHiBlue)
	colorSun     = color.New(color.FgYellow)  // PowerShell DarkYellow
	colorMoon    = color.New(color.FgHiBlack) // PowerShell DarkGray
	colorDefault = color.New(color.FgCyan)    // PowerShell DarkCyan

	// For welcome banner and help text, specific colors from PS script
	psColorYellow = color.New(color.FgYellow)
	psColorCyan   = color.New(color.FgCyan)
	psColorGreen  = color.New(color.FgGreen)
	psColorBlue   = color.New(color.FgBlue)

	// Regex for zipcode
	zipCodeRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

// Geocoding structs
type GeoZipResponse struct {
	Zip     string  `json:"zip"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
}

type GeoDirectResponse struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state,omitempty"`
}

// Weather API structs
type WeatherData struct {
	Lat     float64         `json:"lat"`
	Lon     float64         `json:"lon"`
	Current CurrentWeather  `json:"current"`
	Hourly  []HourlyWeather `json:"hourly,omitempty"`
	Daily   []DailyWeather  `json:"daily"`
	Alerts  []Alert         `json:"alerts,omitempty"`
}

type CurrentWeather struct {
	Dt        int64              `json:"dt"`
	Sunrise   int64              `json:"sunrise"`
	Sunset    int64              `json:"sunset"`
	Temp      float64            `json:"temp"`
	Humidity  int                `json:"humidity"`
	UVI       float64            `json:"uvi"`
	WindSpeed float64            `json:"wind_speed"`
	WindDeg   int                `json:"wind_deg"`
	WindGust  float64            `json:"wind_gust,omitempty"`
	Weather   []WeatherCondition `json:"weather"`
	Rain      *RainSnowInfo      `json:"rain,omitempty"`
	Snow      *RainSnowInfo      `json:"snow,omitempty"`
}

type RainSnowInfo struct {
	OneH float64 `json:"1h,omitempty"`
}

type WeatherCondition struct {
	Main string `json:"main"`
}

type HourlyWeather struct {
	Dt   int64   `json:"dt"`
	Temp float64 `json:"temp"`
}

type DailyWeather struct {
	Dt        int64              `json:"dt"`
	Sunrise   int64              `json:"sunrise"` // Daily sunrise/sunset might differ slightly from current
	Sunset    int64              `json:"sunset"`
	Moonrise  int64              `json:"moonrise"`
	Moonset   int64              `json:"moonset"`
	MoonPhase float64            `json:"moon_phase"`
	Summary   string             `json:"summary"`
	Temp      DailyTemp          `json:"temp"`
	Weather   []WeatherCondition `json:"weather"`
}

type DailyTemp struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Alert struct {
	SenderName  string `json:"sender_name"`
	Event       string `json:"event"`
	Start       int64  `json:"start"`
	End         int64  `json:"end"`
	Description string `json:"description"`
}

type OverviewData struct {
	WeatherOverview string `json:"weather_overview"`
}

func clearScreen() {
	// This is a simple way to clear the screen on different OSes.
	fmt.Print("\033[H\033[2J")
}

// setup will be the new entry point for configuration loading.
func setup() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", fmt.Errorf("error determining config path: %w", err)
	}

	apiKey, err := loadAPIKey(configPath)
	if err != nil {
		return "", err // The error from loadAPIKey is already descriptive.
	}
	return apiKey, nil
}

func getConfigPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory if user config dir is not available
		usr, usrErr := user.Current()
		if usrErr != nil {
			return "", fmt.Errorf("failed to get user config directory (%v) and home directory (%w)", err, usrErr)
		}
		cfgDir = filepath.Join(usr.HomeDir, "."+appName) // e.g., ~/.gw
	} else {
		cfgDir = filepath.Join(cfgDir, appName) // e.g., ~/.config/gw
	}

	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", cfgDir, err)
	}
	return filepath.Join(cfgDir, configFileName), nil
}

// testApiKey will check if the provided key is valid by making a lightweight API call.
func testApiKey(apiKey string) bool {
	if apiKey == "" {
		return false
	}
	// Use a simple geocoding request to a known valid location to test the key.
	testURL := fmt.Sprintf("%s?zip=90210,us&appid=%s", geoZipURL, apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(testURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// A 200 OK response means the key is valid and active.
	return resp.StatusCode == http.StatusOK
}

// showFirstRunSetup handles the interactive prompt for the API key.
func showFirstRunSetup(configPath string) (string, error) {
	clearScreen()
	psColorYellow.Println("*** First Time Setup ***")
	psColorGreen.Println("Get Free One Call API 3.0 Key: https://openweathermap.org/api")

	reader := bufio.NewReader(os.Stdin)
	for {
		color.White("Please enter your API Key:")
		apiKey, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read API key from input: %w", err)
		}
		apiKey = strings.TrimSpace(apiKey)

		if testApiKey(apiKey) {
			cfg := ini.Empty()
			cfg.Section(defaultApiSection).Key(defaultApiKeyName).SetValue(apiKey)

			dir := filepath.Dir(configPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory for config file %s: %w", dir, err)
			}
			if err := cfg.SaveToIndent(configPath, "  "); err != nil {
				return "", fmt.Errorf("failed to save API key to %s: %w", configPath, err)
			}
			if err := os.Chmod(configPath, defaultPermissions); err != nil {
				log.Printf("Warning: could not set permissions for config file %s: %v", configPath, err)
			}

			color.Green("API Key is valid and has been saved to %s", configPath)
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
			return apiKey, nil
		} else {
			color.Red("Invalid API Key. Please try again.")
		}
	}
}

func loadAPIKey(configPath string) (string, error) {
	cfg, err := ini.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "cannot find file") {
			return showFirstRunSetup(configPath)
		}
		return "", fmt.Errorf("failed to load config file %s: %w", configPath, err)
	}

	apiKey := cfg.Section(defaultApiSection).Key(defaultApiKeyName).String()
	if apiKey == "" || !testApiKey(apiKey) {
		if apiKey != "" {
			color.Yellow("Your previously saved API key is no longer valid.")
		}
		return showFirstRunSetup(configPath)
	}
	return apiKey, nil
}

func showHelp() {
	psColorGreen.Println("Usage: gw [ZipCode | \"City, State\"]") // Changed from goweather
	psColorCyan.Println(" • Provide a 5-digit zipcode or a City, State (e.g., 'Portland, OR').")
	fmt.Println()
	psColorBlue.Println("This script retrieves weather info from OpenWeatherMap One Call API 3.0 and outputs:")
	psColorCyan.Println(" • Location (City, Country/State)")
	psColorCyan.Println(" • Overview")
	psColorCyan.Println(" • Conditions")
	psColorCyan.Println(" • Temperature with forecast range (red if <33°F or >89°F)")
	psColorCyan.Println(" • Humidity")
	psColorCyan.Println(" • Wind (with gust if available; red if wind speed >=16 mph)")
	psColorCyan.Println(" • UV Index (red if >=6)")
	psColorCyan.Println(" • Sunrise and Sunset times")
	psColorCyan.Println(" • Moonrise and Moonset times")
	psColorCyan.Println(" • Weather Report")
	psColorCyan.Println(" • Observation timestamp")
	fmt.Println()
	psColorBlue.Println("Examples:")
	psColorCyan.Println("  gw 97219")            // Changed from goweather
	psColorCyan.Println("  gw \"Portland, OR\"") // Changed from goweather
	psColorCyan.Println("  gw -h")               // Changed from goweather
}

func showWelcomeBanner() {
	psColorYellow.Print("    \\|/     ")
	psColorCyan.Println("    .-~~~~~~-.")
	psColorYellow.Print("  -- O --   ")
	psColorCyan.Println("   /_)      ( \\")
	psColorYellow.Print("    /|\\     ")
	psColorCyan.Println("  (   ( )    ( )")
	psColorYellow.Print("            ")
	psColorCyan.Println("   `-~~~~~~~~~-`")
	psColorGreen.Print("  Welcome   ")
	psColorCyan.Println("     ''    ''")
	psColorGreen.Print("     to     ")
	psColorCyan.Println("    ''    ''")
	psColorGreen.Print("  GetWeather")
	psColorCyan.Println("  ________________") // Changed from GoWeather
	psColorYellow.Print("            ")
	psColorCyan.Println("~~~~~~~~~~~~~~~~~~~~")
	fmt.Println()
}

func makeAPIRequest(url string, target interface{}) error {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", appName+"/1.0") // appName is now "gw"

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request to %s failed with status %s: %s", url, resp.Status, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body from %s: %w", url, err)
	}
	if len(body) == 0 {
		return fmt.Errorf("empty response from API: %s", url)
	}

	err = json.Unmarshal(body, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s (body: %s): %w", url, string(body), err)
	}
	return nil
}

func getGeoCoordinates(locationInput, apiKey string) (lat, lon float64, city, countryOrState string, err error) {
	if zipCodeRegex.MatchString(locationInput) {
		geoURL := fmt.Sprintf("%s?zip=%s,us&appid=%s", geoZipURL, url.QueryEscape(locationInput), apiKey)
		var geoResp GeoZipResponse
		if err = makeAPIRequest(geoURL, &geoResp); err != nil {
			return 0, 0, "", "", fmt.Errorf("geocoding by zip failed for '%s': %w", locationInput, err)
		}
		if geoResp.Name == "" {
			return 0, 0, "", "", fmt.Errorf("no geocoding results for zipcode '%s'", locationInput)
		}
		return geoResp.Lat, geoResp.Lon, geoResp.Name, geoResp.Country, nil
	} else {
		// For non-zip inputs, append ",us" if a comma isn't already present.
		loc := strings.TrimSpace(locationInput)
		if !strings.Contains(loc, ",") {
			loc += ",us"
		}
		geoURL := fmt.Sprintf("%s?q=%s&limit=1&appid=%s", geoDirectURL, url.QueryEscape(loc), apiKey)
		var geoRespArr []GeoDirectResponse
		if err = makeAPIRequest(geoURL, &geoRespArr); err != nil {
			return 0, 0, "", "", fmt.Errorf("geocoding by city failed for '%s': %w", locationInput, err)
		}
		if len(geoRespArr) == 0 {
			return 0, 0, "", "", fmt.Errorf("no geocoding results found for '%s'", locationInput)
		}
		geoResp := geoRespArr[0]
		resolvedLocation := geoResp.Country
		if geoResp.State != "" {
			resolvedLocation = geoResp.State + ", " + geoResp.Country
		}
		return geoResp.Lat, geoResp.Lon, geoResp.Name, resolvedLocation, nil
	}
}

func getWeatherData(lat, lon float64, apiKey string) (*WeatherData, error) {
	weatherURL := fmt.Sprintf("%s?lat=%f&lon=%f&appid=%s&units=imperial&lang=en&exclude=minutely",
		oneCallURL, lat, lon, apiKey)
	var data WeatherData
	if err := makeAPIRequest(weatherURL, &data); err != nil {
		return nil, err
	}
	if data.Current.Dt == 0 {
		return nil, fmt.Errorf("weather API returned incomplete 'current' data")
	}
	if len(data.Daily) == 0 {
		return nil, fmt.Errorf("weather API returned no 'daily' forecast data")
	}
	return &data, nil
}

func getWeatherOverview(lat, lon float64, apiKey string) (*OverviewData, error) {
	overviewAPIURL := fmt.Sprintf("%s?lat=%f&lon=%f&appid=%s&units=imperial&lang=en",
		overviewURL, lat, lon, apiKey)
	var data OverviewData
	if err := makeAPIRequest(overviewAPIURL, &data); err != nil {
		return nil, err
	}
	if data.WeatherOverview == "" {
		return nil, fmt.Errorf("weather overview API returned empty 'weather_overview' data")
	}
	return &data, nil
}

func formatUnixTimeLocal(unixTime int64, format string) string {
	if unixTime == 0 {
		return "N/A"
	}
	return time.Unix(unixTime, 0).Local().Format(format)
}

func getCardinalDirection(deg int) string {
	val := int(math.Floor((float64(deg) / 22.5) + 0.5))
	directions := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return directions[val%16]
}

func getMoonPhaseDescription(phase float64) string {
	switch {
	case phase < 0.0625 || phase >= 0.9375:
		return "New Moon"
	case phase < 0.1875:
		return "Waxing Crescent"
	case phase < 0.3125:
		return "First Quarter"
	case phase < 0.4375:
		return "Waxing Gibbous"
	case phase < 0.5625:
		return "Full Moon"
	case phase < 0.6875:
		return "Waning Gibbous"
	case phase < 0.8125:
		return "Third Quarter"
	default:
		return "Waning Crescent"
	}
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return lines
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

func displayWeather(city, countryOrState string, weather *WeatherData, overview *OverviewData) {
	current := weather.Current
	dailyToday := weather.Daily[0] // Assumes at least one day is present, checked in getWeatherData

	tempC := colorDefault
	if current.Temp < 33 || current.Temp > 89 {
		tempC = colorAlert
	}

	windC := colorDefault
	if current.WindSpeed >= 16 {
		windC = colorAlert
	}

	uvC := colorDefault
	if current.UVI >= 6 {
		uvC = colorAlert
	}

	tempIndicator := ""
	if len(weather.Hourly) > 2 { // PowerShell used index 2 for next hour's temp (dt + 2 hours)
		nextHourTemp := weather.Hourly[2].Temp
		tempDiff := nextHourTemp - current.Temp
		if tempDiff >= 0.67 {
			tempIndicator = "(Rising)"
		}
		if tempDiff <= -0.67 {
			tempIndicator = "(Falling)"
		}
	}

	conditions := current.Weather[0].Main
	if current.Rain != nil && current.Rain.OneH > 0 {
		conditions = fmt.Sprintf("%s [%.1f mm/H Rain]", conditions, current.Rain.OneH)
	}
	if current.Snow != nil && current.Snow.OneH > 0 {
		conditions = fmt.Sprintf("%s [%.1f mm/H Snow]", conditions, current.Snow.OneH)
	}

	windDisplay := fmt.Sprintf("%.1f mph %s", current.WindSpeed, getCardinalDirection(current.WindDeg))
	windLabel := "Wind:"
	if current.WindGust > 0 {
		windLabel = "Wind[Gust]:"
		windDisplay = fmt.Sprintf("%.1f mph [%.1f mph] %s", current.WindSpeed, current.WindGust, getCardinalDirection(current.WindDeg))
	}

	colorTitle.Printf("*** %s, %s Current Conditions ***\n", city, countryOrState)
	colorInfo.Printf("Forecast: %s\n", dailyToday.Summary)
	colorDefault.Printf("Currently: %s\n", conditions)
	tempC.Printf("Temp [L/H]: %.0f°F%s [%.0f°F/%.0f°F]\n", current.Temp, tempIndicator, dailyToday.Temp.Min, dailyToday.Temp.Max)
	colorDefault.Printf("Humidity: %d%%\n", current.Humidity)
	uvC.Printf("UV Index: %.1f\n", current.UVI)
	windC.Printf("%s %s\n", windLabel, windDisplay)

	if len(weather.Daily) > 1 {
		psColorCyan.Printf("Tomorrow: %s\n", weather.Daily[1].Summary)
	}

	colorSun.Printf("Sunrise: %s\n", formatUnixTimeLocal(current.Sunrise, "3:04 PM"))
	colorSun.Printf("Sunset: %s\n", formatUnixTimeLocal(current.Sunset, "3:04 PM"))
	colorMoon.Printf("Moonrise: %s\n", formatUnixTimeLocal(dailyToday.Moonrise, "3:04 PM"))
	colorMoon.Printf("Moonset: %s\n", formatUnixTimeLocal(dailyToday.Moonset, "3:04 PM"))
	colorMoon.Printf("Moon Phase: %s\n", getMoonPhaseDescription(dailyToday.MoonPhase))
	colorInfo.Printf("Observed: %s\n", formatUnixTimeLocal(current.Dt, "Jan 2, 2006 3:04 PM"))
	fmt.Println()

	colorTitle.Printf("*** %s, %s Weather Report ***\n", city, countryOrState)
	wrappedReport := wrapText(overview.WeatherOverview, 80) // Assuming 80 char width for console
	for _, line := range wrappedReport {
		colorDefault.Println(line)
	}
	fmt.Println()
	psColorCyan.Printf("https://forecast.weather.gov/MapClick.php?lat=%f&lon=%f\n", weather.Lat, weather.Lon)

	if len(weather.Alerts) > 0 {
		for _, alert := range weather.Alerts {
			fmt.Println()
			colorAlert.Printf("*** %s - %s ***\n", alert.Event, alert.SenderName)
			wrappedAlertDesc := wrapText(alert.Description, 80)
			for _, line := range wrappedAlertDesc {
				colorDefault.Println(line)
			}
			colorInfo.Printf("Starts: %s\n", formatUnixTimeLocal(alert.Start, "Jan 2, 2006 3:04 PM MST"))
			colorInfo.Printf("Ends: %s\n", formatUnixTimeLocal(alert.End, "Jan 2, 2006 3:04 PM MST"))
		}
	}
}

func main() {
	clearScreen()

	log.SetFlags(0) // No timestamps or prefixes for cleaner error messages from log.Fatal

	helpFlag := flag.Bool("h", false, "Display help information")
	helpLongFlag := flag.Bool("help", false, "Display help information")
	flag.Parse()

	if *helpFlag || *helpLongFlag {
		showHelp()
		return
	}

	// --- API Key Handling (Moved Up) ---
	apiKey, err := setup()
	if err != nil {
		log.Fatalf("Configuration setup failed: %v", err)
	}

	// --- Location Input & Geocoding Loop ---
	var lat, lon float64
	var city, countryOrState string
	args := flag.Args()
	isInteractive := len(args) == 0
	var locationInput string
	if !isInteractive {
		locationInput = strings.Join(args, " ")
	}

	for {
		if isInteractive && locationInput == "" {
			clearScreen()
			showWelcomeBanner()
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a location (Zip Code or City, State): ")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Error reading location input: %v", err)
			}
			locationInput = strings.TrimSpace(input)
			if locationInput == "" {
				return // User hit enter on an empty line, exit cleanly.
			}
		}

		var geoErr error
		lat, lon, city, countryOrState, geoErr = getGeoCoordinates(locationInput, apiKey)
		if geoErr != nil {
			color.Red("Location not found, try again")
			if !isInteractive {
				os.Exit(1)
			}
			locationInput = "" // This will cause the interactive prompt to show again
			time.Sleep(1 * time.Second)
			continue
		}
		break // Geocoding was successful, exit the loop.
	}

	weatherData, err := getWeatherData(lat, lon, apiKey)
	if err != nil {
		log.Fatalf("Error fetching weather data: %v", err)
	}

	overviewData, err := getWeatherOverview(lat, lon, apiKey)
	if err != nil {
		log.Fatalf("Error fetching weather overview: %v", err)
	}

	// Clear screen if we prompted for location input before showing weather.
	// This is done again here to ensure a clean display if the API key prompt occurred
	// and then the location prompt followed.
	if len(args) == 0 {
		clearScreen()
	}

	displayWeather(city, countryOrState, weatherData, overviewData)

	// --- Pause Before Exit Logic ---
	// Replicate PowerShell script's "pause before exit" logic
	// Pause if no arguments were passed, unless run from a known terminal that keeps the window open.
	if len(args) == 0 {
		shouldPause := true // Default to pause

		ppid := int32(os.Getppid())
		parentProc, pErr := process.NewProcess(ppid) // Renamed err to pErr to avoid conflict
		if pErr == nil {
			parentName, errName := parentProc.Name()
			if errName == nil {
				parentNameLower := strings.ToLower(parentName)

				// List of parent processes that typically keep their windows open
				excludedParents := make(map[string]bool)
				if runtime.GOOS == "windows" {
					excludedParents["powershell.exe"] = true
					excludedParents["cmd.exe"] = true
					excludedParents["wt.exe"] = true // Windows Terminal
					// explorer.exe is not in this list, so if it's the parent (double-click), it will pause.
				} else {
					// Common terminal emulators/shells on Linux/macOS
					excludedParents["bash"] = true
					excludedParents["zsh"] = true
					excludedParents["sh"] = true
					excludedParents["fish"] = true
					excludedParents["gnome-terminal-server"] = true // Often the parent of gnome-terminal
					excludedParents["konsole"] = true
					excludedParents["xterm"] = true
				}

				if _, isExcluded := excludedParents[parentNameLower]; isExcluded {
					shouldPause = false
				}
			} // If parentName can't be determined, default to pausing
		} // If parentProc can't be determined, default to pausing

		if shouldPause {
			fmt.Println() // Ensure the prompt is on a new line
			psColorYellow.Print("Press Enter to exit...")
			bufio.NewReader(os.Stdin).ReadBytes('\n') // Wait for Enter key
		}
	}
}
