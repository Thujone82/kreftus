package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/fatih/color"
	"golang.org/x/term"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/ini.v1"
)

const (
	version = "1.5"
	date    = "2025-08-07@1430"
)

// Configuration structure
type Config struct {
	Settings struct {
		ApiKey string `ini:"ApiKey"`
	} `ini:"Settings"`
}

// API response structure
type APIResponse struct {
	Rate float64 `json:"rate"`
}

// Mode settings
type ModeSettings struct {
	duration int
	interval int
	spinner  []string
}

// Global variables
var (
	apiKey            string
	currentBtcPrice   float64
	monitorStartPrice float64
	monitorStartTime  time.Time
	previousPrice     float64
	previousColor     string
	priceHistory      []float64
	soundEnabled      bool
	sparklineEnabled  bool
)

// Mode configurations
var modeSettings = map[string]ModeSettings{
	"go": {
		duration: 900, // 15 minutes
		interval: 5,
		spinner:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	},
	"golong": {
		duration: 86400, // 24 hours
		interval: 20,
		spinner:  []string{"*"},
	},
}

// Command line arguments structure
type Args struct {
	goMode         bool
	golongMode     bool
	sound          bool
	sparkline      bool
	help           bool
	conversionMode string
	conversionVal  float64
}

func main() {
	// Ensure Windows console uses UTF-8 and supports ANSI (for spinners, sparklines, colors)
	configureWindowsConsole()
	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		// Restore terminal state and exit cleanly
		fmt.Print("\033[?25h") // Show cursor
		os.Exit(0)
	}()

	// Parse command line arguments
	args := parseArgs()

	// Initialize configuration
	if err := initConfig(); err != nil {
		color.Red("Failed to initialize configuration: %v", err)
		os.Exit(1)
	}

	// Handle help
	if args.help {
		printHelp()
		return
	}

	// Handle conversion modes
	if args.conversionMode != "" {
		handleConversion(args)
		return
	}

	// Get initial price - show appropriate message based on mode
	if args.goMode || args.golongMode {
		clearScreen()
		fmt.Print("\r")
		color.Cyan("Fetching initial price...")
	} else {
		clearScreen()
		color.Cyan("Fetching initial price...")
	}

	if err := fetchInitialPrice(); err != nil {
		color.Red("Failed to fetch initial price: %v", err)
		os.Exit(1)
	}

	// Handle monitoring modes
	if args.goMode || args.golongMode {
		runMonitoringMode(args)
	} else {
		// Interactive mode - no screen clear here (matches PowerShell)
		runInteractiveMode(args)
	}
}

func parseArgs() Args {
	args := Args{}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-go":
			args.goMode = true
		case "-golong":
			args.golongMode = true
		case "-s":
			args.sound = true
		case "-h":
			args.sparkline = true
		case "-help":
			args.help = true
		case "-bu":
			if i+1 < len(os.Args) {
				if val, err := strconv.ParseFloat(os.Args[i+1], 64); err == nil {
					args.conversionMode = "bu"
					args.conversionVal = val
					i++
				}
			}
		case "-ub":
			if i+1 < len(os.Args) {
				if val, err := strconv.ParseFloat(os.Args[i+1], 64); err == nil {
					args.conversionMode = "ub"
					args.conversionVal = val
					i++
				}
			}
		case "-us":
			if i+1 < len(os.Args) {
				if val, err := strconv.ParseFloat(os.Args[i+1], 64); err == nil {
					args.conversionMode = "us"
					args.conversionVal = val
					i++
				}
			}
		case "-su":
			if i+1 < len(os.Args) {
				if val, err := strconv.ParseFloat(os.Args[i+1], 64); err == nil {
					args.conversionMode = "su"
					args.conversionVal = val
					i++
				}
			}
		}
	}

	return args
}

func initConfig() error {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)

	// Try bmon.ini first
	bmonPath := filepath.Join(exeDir, "bmon.ini")
	if cfg, err := loadConfig(bmonPath); err == nil && cfg.Settings.ApiKey != "" {
		apiKey = cfg.Settings.ApiKey
		return nil
	}

	// Try vbtc.ini as fallback
	vbtcPath := filepath.Join(exeDir, "vbtc.ini")
	if cfg, err := loadConfig(vbtcPath); err == nil && cfg.Settings.ApiKey != "" {
		apiKey = cfg.Settings.ApiKey
		return nil
	}

	// No valid config found, start onboarding
	return runOnboarding(bmonPath)
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{}

	iniFile, err := ini.Load(path)
	if err != nil {
		return cfg, err
	}

	if err := iniFile.MapTo(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func saveConfig(path string, apiKey string) error {
	cfg := ini.Empty()
	cfg.Section("Settings").Key("ApiKey").SetValue(apiKey)
	return cfg.SaveTo(path)
}

func runOnboarding(configPath string) error {
	color.Yellow("*** bmon First Time Setup ***")
	color.White("A LiveCoinWatch API key is required to monitor prices.")
	color.Green("Get a free key at: https://www.livecoinwatch.com/tools/api")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Please enter your LiveCoinWatch API Key (or press Enter to exit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return fmt.Errorf("setup cancelled by user")
		}

		if testAPIKey(input) {
			if err := saveConfig(configPath, input); err != nil {
				color.Red("API Key was valid, but failed to save to %s. Please check file permissions.", configPath)
				fmt.Print("Press Enter to exit.")
				reader.ReadString('\n')
				return err
			}

			apiKey = input
			color.Green("API Key is valid and has been saved to bmon.ini.")
			fmt.Print("Press Enter to start monitoring.")
			reader.ReadString('\n')
			return nil
		} else {
			color.Yellow("Invalid API Key. Please try again.")
		}
	}
}

func testAPIKey(key string) bool {
	if key == "" {
		return false
	}

	url := "https://api.livecoinwatch.com/coins/single"
	payload := map[string]interface{}{
		"currency": "USD",
		"code":     "BTC",
		"meta":     false,
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func fetchInitialPrice() error {
	price, err := getBtcPrice()
	if err != nil {
		return err
	}

	currentBtcPrice = price
	priceHistory = append(priceHistory, price)
	return nil
}

func getBtcPrice() (float64, error) {
	if apiKey == "" {
		return 0, fmt.Errorf("API key is null or empty")
	}

	url := "https://api.livecoinwatch.com/coins/single"
	payload := map[string]interface{}{
		"currency": "USD",
		"code":     "BTC",
		"meta":     false,
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}

	// Retry logic
	maxAttempts := 5
	baseDelay := 2 * time.Second
	warningShown := false

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			if attempt >= maxAttempts {
				return 0, fmt.Errorf("API call failed after %d attempts: %v", maxAttempts, err)
			}

			// Exponential backoff with jitter
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
			jitter := time.Duration(time.Now().UnixNano()%1000) * time.Millisecond
			sleepTime := backoff + jitter

			warningMsg := fmt.Sprintf("API call failed. Retrying in %.1f seconds... (Retry %d of %d)",
				sleepTime.Seconds(), attempt, maxAttempts-1)
			// Draw warning in-place on a single line without advancing
			width := getConsoleWidth()
			padded := fmt.Sprintf("%-*s", width, "\x1b[33m"+warningMsg+"\x1b[0m")
			fmt.Printf("\r%s\r", padded)
			warningShown = true

			time.Sleep(sleepTime)
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return 0, err
		}

		if apiResp.Rate <= 0 {
			warningMsg := fmt.Sprintf("API returned invalid price: '%.2f'", apiResp.Rate)
			width := getConsoleWidth()
			padded := fmt.Sprintf("%-*s", width, "\x1b[33m"+warningMsg+"\x1b[0m")
			fmt.Printf("\r%s\r", padded)
			warningShown = true
			return 0, fmt.Errorf("invalid price returned")
		}

		// Clear screen after a successful retry to ensure no messages remain
		if warningShown {
			clearScreen()
		} else {
			clearLine()
		}
		return apiResp.Rate, nil
	}

	return 0, fmt.Errorf("failed to get price after all attempts")
}

func getConsoleWidth() int {
	// Try to get console width, default to 80 if we can't
	if runtime.GOOS == "windows" {
		// On Windows, we can try to get the console width
		cmd := exec.Command("cmd", "/c", "mode con | findstr Columns")
		output, err := cmd.Output()
		if err == nil {
			// Expect something like: "   Columns:         120"
			fields := strings.Fields(string(output))
			for i := 0; i < len(fields); i++ {
				if strings.HasPrefix(fields[i], "Columns") && i+1 < len(fields) {
					if width, err := strconv.Atoi(fields[i+1]); err == nil {
						return width
					}
				}
			}
		}
	} else {
		// On Linux, try to get terminal width
		cmd := exec.Command("sh", "-c", "stty size 2>/dev/null | awk '{print $2}'")
		output, err := cmd.Output()
		if err == nil {
			trim := strings.TrimSpace(string(output))
			if width, err := strconv.Atoi(trim); err == nil && width > 0 {
				return width
			}
		}
	}
	return 80 // Default fallback
}

func clearLine() {
	width := getConsoleWidth()
	fmt.Print("\r")
	// Clear the line by printing spaces
	fmt.Print(strings.Repeat(" ", width))
	fmt.Print("\r")
}

func getSparkline(history []float64) string {
	if len(history) < 2 {
		return "‖            ‖"
	}

	// Choose a charset that renders reliably. On Windows when launched
	// directly in classic conhost (no WT_SESSION), prefer CP437 shading.
	sparkChars := getSparkChars()

	// Find min and max
	minPrice := history[0]
	maxPrice := history[0]
	for _, price := range history {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	priceRange := maxPrice - minPrice
	if priceRange < 0.00000001 {
		return "‖            ‖"
	}

	sparkline := ""
	for _, price := range history {
		normalized := (price - minPrice) / priceRange
		charIndex := int(normalized * float64(len(sparkChars)-1))
		if charIndex >= len(sparkChars) {
			charIndex = len(sparkChars) - 1
		}
		sparkline += string(sparkChars[charIndex])
	}

	// Ensure sparkline content is exactly 12 characters (ASCII spaces for pad)
	// First truncate if too long (keep the most recent data on the right)
	if len(sparkline) > 12 {
		sparkline = sparkline[len(sparkline)-12:]
	}
	// Then pad to exactly 12 characters on the left (like PowerShell)
	// Use ASCII spaces; ensure no stray runes get into the buffer
	sparkline = fmt.Sprintf("%12s", sparkline)

	// Return with wrapper - should always be exactly 14 characters
	result := "‖" + sparkline + "‖"

	return result
}

// getSparkChars selects characters for the sparkline that the current
// terminal can reliably render. Falls back to CP437 shading on classic conhost.
func getSparkChars() []rune {
	if runtime.GOOS == "windows" {
		// Windows Terminal sets WT_SESSION. Classic conhost typically does not.
		if os.Getenv("WT_SESSION") == "" {
			return []rune{' ', '░', '▒', '▓', '█'}
		}
	}
	// Default: finer Unicode block elements
	return []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
}

func playSound(frequency int, duration int) {
	if runtime.GOOS == "windows" {
		exec.Command("powershell", "-c", fmt.Sprintf("[console]::beep(%d, %d)", frequency, duration)).Run()
	} else {
		// For Linux, use a different approach
		exec.Command("echo", "-e", "\\a").Run()
	}
}

func handleConversion(args Args) {
	price, err := getBtcPrice()
	if err != nil {
		color.Red("Could not retrieve Bitcoin price. Cannot perform conversion.")
		os.Exit(1)
	}

	switch args.conversionMode {
	case "bu":
		usdValue := args.conversionVal * price
		fmt.Printf("$%s\n", formatUSD(usdValue))
	case "ub":
		if price <= 0.00000001 {
			color.Red("Bitcoin price is too low or zero, cannot divide.")
			os.Exit(1)
		}
		btcValue := args.conversionVal / price
		fmt.Printf("B%.8f\n", btcValue)
	case "us":
		if price <= 0.00000001 {
			color.Red("Bitcoin price is too low or zero, cannot divide.")
			os.Exit(1)
		}
		satoshiValue := (args.conversionVal / price) * 100000000
		fmt.Printf("%.0fs\n", satoshiValue)
	case "su":
		usdValue := (args.conversionVal / 100000000) * price
		fmt.Printf("$%s\n", formatUSD(usdValue))
	}
}

func runMonitoringMode(args Args) {
	// Determine mode
	mode := "go"
	if args.golongMode {
		mode = "golong"
	}

	settings := modeSettings[mode]

	// Initialize state
	monitorStartPrice = currentBtcPrice
	previousPrice = currentBtcPrice
	previousColor = "White"
	monitorStartTime = time.Now()
	soundEnabled = args.sound
	sparklineEnabled = args.sparkline

	// Start keyboard listener
	keyChan := startKeyboardListener()

	// Clear screen and hide cursor (matches PowerShell behavior)
	clearScreen()
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	spinnerIndex := 0

	// Main monitoring loop
	for {
		// Check if duration exceeded
		if time.Since(monitorStartTime).Seconds() >= float64(settings.duration) {
			break
		}

		// Calculate price change and color
		priceChange := currentBtcPrice - monitorStartPrice
		priceColor := "White"
		changeString := ""

		if priceChange >= 0.01 {
			priceColor = "Green"
			changeString = fmt.Sprintf(" [+$%.2f]", priceChange)
		} else if priceChange <= -0.01 {
			priceColor = "Red"
			changeString = fmt.Sprintf(" [$%.2f]", priceChange)
		}

		// Determine if flash is needed
		flashNeeded := false
		if priceColor != "White" && priceColor != previousColor {
			flashNeeded = true
		} else if (priceColor == "Green" && currentBtcPrice > previousPrice) ||
			(priceColor == "Red" && currentBtcPrice < previousPrice) {
			flashNeeded = true
		}

		// Wait for interval with keyboard polling (like PowerShell script)
		waitStart := time.Now()
		refreshed := false
		modeSwitched := false
		isFirstTick := true
		previousWasFlash := false
		for time.Since(waitStart).Seconds() < float64(settings.interval) {
			spinnerChar := settings.spinner[spinnerIndex%len(settings.spinner)]
			sparklineString := ""
			if sparklineEnabled {
				sparklineString = " " + getSparkline(priceHistory)
			} else {
				sparklineString = " Bitcoin (USD):"
			}

			restOfLine := fmt.Sprintf("%s $%s%s", sparklineString, formatUSD(currentBtcPrice), changeString)

			if mode == "go" {
				spinnerIndex = (spinnerIndex + 1) % len(settings.spinner)
			}

			fullLine := spinnerChar + restOfLine

			// Handle flash: hold for one 500ms tick, then draw normal line on next tick
			if flashNeeded && isFirstTick {
				width := getConsoleWidth()
				paddedLine := fmt.Sprintf("%-*s", width, fullLine)
				switch priceColor {
				case "Green":
					fmt.Printf("\r\033[42;30m%s\033[0m\r", paddedLine)
				case "Red":
					fmt.Printf("\r\033[41;30m%s\033[0m\r", paddedLine)
				default:
					fmt.Printf("\r%s\r", paddedLine)
				}
				// Hold the flash for one tick and continue
				time.Sleep(500 * time.Millisecond)
				isFirstTick = false
				previousWasFlash = true
				continue
			} else {
				// Normal display - matches PowerShell exactly
				// Build the complete line with colors
				var coloredLine string
				switch priceColor {
				case "Green":
					coloredLine = fmt.Sprintf("\033[32m%s\033[0m", restOfLine)
				case "Red":
					coloredLine = fmt.Sprintf("\033[31m%s\033[0m", restOfLine)
				default:
					coloredLine = restOfLine
				}

				// Print the complete line with spinner and padding
				width := getConsoleWidth()
				if previousWasFlash {
					// Clear any lingering background color from prior flash
					fmt.Printf("\r%s\r", strings.Repeat(" ", width))
					previousWasFlash = false
				}
				paddedLine := fmt.Sprintf("%-*s", width, spinnerChar+coloredLine)
				fmt.Printf("\r%s\r", paddedLine)
			}

			isFirstTick = false

			// Check for key press (non-blocking)
			if key, pressed := checkKeyPress(keyChan); pressed {
				switch key {
				case 'r':
					monitorStartPrice = currentBtcPrice
					monitorStartTime = time.Now()
					refreshed = true
				case 'm':
					if mode == "go" {
						mode = "golong"
					} else {
						mode = "go"
					}
					settings = modeSettings[mode]
					monitorStartTime = time.Now()
					monitorStartPrice = currentBtcPrice
					modeSwitched = true
					spinnerIndex = 0
				case 's':
					soundEnabled = !soundEnabled
					if soundEnabled {
						playSound(1200, 350)
					} else {
						playSound(400, 350)
					}
				case 'h':
					sparklineEnabled = !sparklineEnabled
				}
			}
			if refreshed || modeSwitched {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if refreshed || modeSwitched {
			continue
		}

		// Update previous values
		previousPrice = currentBtcPrice
		previousColor = priceColor

		// Final display update (matches PowerShell exactly)
		if mode == "go" {
			spinnerIndex = (spinnerIndex + 1) % len(settings.spinner)
		}
		spinnerChar := settings.spinner[spinnerIndex%len(settings.spinner)]
		sparklineString := ""
		if sparklineEnabled {
			sparklineString = " " + getSparkline(priceHistory)
		} else {
			sparklineString = " Bitcoin (USD):"
		}
		restOfLine := fmt.Sprintf("%s $%s%s", sparklineString, formatUSD(currentBtcPrice), changeString)

		// Final display - matches PowerShell exactly
		// Build the complete line with colors
		var coloredLine string
		switch priceColor {
		case "Green":
			coloredLine = fmt.Sprintf("\033[32m%s\033[0m", restOfLine)
		case "Red":
			coloredLine = fmt.Sprintf("\033[31m%s\033[0m", restOfLine)
		default:
			coloredLine = restOfLine
		}

		// Print the complete line with cyan spinner and padding
		cyanSpinner := fmt.Sprintf("\033[36m%s\033[0m", spinnerChar)
		width := getConsoleWidth()
		paddedLine := fmt.Sprintf("%-*s", width, cyanSpinner+coloredLine)
		fmt.Printf("\r%s\r", paddedLine)

		// Fetch new price
		if newPrice, err := getBtcPrice(); err == nil {
			if soundEnabled {
				if newPrice >= currentBtcPrice+0.01 {
					playSound(1200, 150)
				} else if newPrice <= currentBtcPrice-0.01 {
					playSound(400, 150)
				}
			}
			currentBtcPrice = newPrice
			priceHistory = append(priceHistory, newPrice)
			if len(priceHistory) > 12 {
				priceHistory = priceHistory[1:]
			}
		}
	}
}

func runInteractiveMode(args Args) {
	soundEnabled = args.sound
	sparklineEnabled = args.sparkline

	for {
		// Main screen - clear screen first (matches PowerShell)
		clearScreen()
		color.Yellow("*** BTC Monitor ***")
		fmt.Printf("Bitcoin (USD): $%s\n", formatUSD(currentBtcPrice))
		white := color.New(color.FgWhite)
		cyan := color.New(color.FgCyan)
		white.Print("Start[")
		cyan.Print("Space")
		white.Print("], Exit[")
		cyan.Print("Ctrl+C")
		white.Print("]")

		// Wait for space bar
		fmt.Print("\nPress Space to start monitoring...")
		for {
			key := getKeyPress()
			if key == ' ' {
				break
			}
		}

		color.Cyan("\nStarting monitoring...")

		// Get new price (no screen clear after this - matches PowerShell)
		if newPrice, err := getBtcPrice(); err == nil {
			currentBtcPrice = newPrice
		}

		// Initialize monitoring state
		monitorStartPrice = currentBtcPrice
		monitorStartTime = time.Now()
		priceHistory = []float64{currentBtcPrice}
		previousPrice = currentBtcPrice
		previousColor = "White"
		monitorDuration := 300 // 5 minutes

		// Start keyboard listener for interactive mode
		keyChan := startKeyboardListener()

		// Monitoring loop
		for time.Since(monitorStartTime).Seconds() < float64(monitorDuration) {
			// Calculate price change and color
			priceChange := currentBtcPrice - monitorStartPrice
			priceColor := "White"
			changeString := ""

			if priceChange >= 0.01 {
				priceColor = "Green"
				changeString = fmt.Sprintf(" [+$%.2f]", priceChange)
			} else if priceChange <= -0.01 {
				priceColor = "Red"
				changeString = fmt.Sprintf(" [$%.2f]", priceChange)
			}

			// Determine if flash is needed
			flashNeeded := false
			if priceColor != "White" && priceColor != previousColor {
				flashNeeded = true
			} else if (priceColor == "Green" && currentBtcPrice > previousPrice) ||
				(priceColor == "Red" && currentBtcPrice < previousPrice) {
				flashNeeded = true
			}

			// Handle flash (matches PowerShell exactly)
			if flashNeeded {
				drawScreen(true, changeString, priceColor)
				time.Sleep(500 * time.Millisecond)
			}

			// Draw normal screen
			drawScreen(false, changeString, priceColor)

			// Wait for 5 seconds with keyboard polling (like PowerShell script)
			waitStart := time.Now()
			paused := false
			refreshed := false
			for time.Since(waitStart).Seconds() < 5 {
				// Check for key press (non-blocking)
				if key, pressed := checkKeyPress(keyChan); pressed {
					switch key {
					case ' ':
						paused = true
					case 'r':
						refreshed = true
					case 's':
						soundEnabled = !soundEnabled
						if soundEnabled {
							playSound(1200, 350)
						} else {
							playSound(400, 350)
						}
					case 'h':
						sparklineEnabled = !sparklineEnabled
						drawScreen(false, changeString, priceColor)
					}
				}
				if paused || refreshed {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if paused {
				break // Return to main screen
			}
			if refreshed {
				monitorStartPrice = currentBtcPrice
				monitorStartTime = time.Now()
				continue
			}

			// Update previous values
			previousPrice = currentBtcPrice
			previousColor = priceColor

			// Fetch new price
			if newPrice, err := getBtcPrice(); err == nil {
				if soundEnabled {
					if newPrice >= currentBtcPrice+0.01 {
						playSound(1200, 150)
					} else if newPrice <= currentBtcPrice-0.01 {
						playSound(400, 150)
					}
				}
				currentBtcPrice = newPrice
				priceHistory = append(priceHistory, newPrice)
				if len(priceHistory) > 12 {
					priceHistory = priceHistory[1:]
				}
			}
		}
	}
}

func drawScreen(invertColors bool, changeString, priceColor string) {
	clearScreen()
	color.Yellow("*** BTC Monitor ***")

	sparklineString := ""
	if sparklineEnabled {
		sparklineString = getSparkline(priceHistory)
	} else {
		sparklineString = "Bitcoin (USD):"
	}

	priceLine := fmt.Sprintf("%s $%s%s", sparklineString, formatUSD(currentBtcPrice), changeString)

	if invertColors {
		switch priceColor {
		case "Green":
			color.New(color.BgGreen, color.FgBlack).Println(priceLine)
		case "Red":
			color.New(color.BgRed, color.FgBlack).Println(priceLine)
		default:
			color.White(priceLine)
		}
	} else {
		switch priceColor {
		case "Green":
			color.Green(priceLine)
		case "Red":
			color.Red(priceLine)
		default:
			color.White(priceLine)
		}
	}

	// Control line - matches PowerShell exactly (no newline)
	white := color.New(color.FgWhite)
	cyan := color.New(color.FgCyan)
	white.Print("Pause[")
	cyan.Print("Space")
	white.Print("], Reset[")
	cyan.Print("R")
	white.Print("], Exit[")
	cyan.Print("Ctrl+C")
	white.Print("]")
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Ensure UTF-8 code page for proper box-drawing and braille characters
		exec.Command("cmd", "/c", "chcp 65001 >nul").Run()
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// configureWindowsConsole ensures the Windows console uses UTF-8 code page
// so Unicode characters (spinners, sparklines, wrappers) render correctly.
func configureWindowsConsole() {
	if runtime.GOOS != "windows" {
		return
	}
	// Switch to UTF-8 code page quietly for legacy APIs
	_ = exec.Command("cmd", "/c", "chcp 65001 >nul").Run()

	// Also set code page via Win32 to help Go's console writes
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")
	setConsoleCP := kernel32.NewProc("SetConsoleCP")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	const (
		CP_UTF8                  = 65001
		ENABLE_VTP_OUTPUT uint32 = 0x0004 // ENABLE_VIRTUAL_TERMINAL_PROCESSING
		ENABLE_VTP_INPUT  uint32 = 0x0200 // ENABLE_VIRTUAL_TERMINAL_INPUT
	)

	// Set UTF-8 code pages
	_, _, _ = setConsoleOutputCP.Call(uintptr(CP_UTF8))
	_, _, _ = setConsoleCP.Call(uintptr(CP_UTF8))

	// Enable VT processing on output
	// Use DWORD for handle constants to avoid negative literal to uintptr
	outHandle, _, _ := getStdHandle.Call(uintptr(^uint32(11) + 1)) // STD_OUTPUT_HANDLE = -11
	if outHandle != 0 {
		var mode uint32
		_, _, _ = getConsoleMode.Call(outHandle, uintptr(unsafe.Pointer(&mode)))
		mode |= ENABLE_VTP_OUTPUT
		_, _, _ = setConsoleMode.Call(outHandle, uintptr(mode))
	}

	// Enable VT on input (optional, harmless)
	inHandle, _, _ := getStdHandle.Call(uintptr(^uint32(10) + 1)) // STD_INPUT_HANDLE = -10
	if inHandle != 0 {
		var mode uint32
		_, _, _ = getConsoleMode.Call(inHandle, uintptr(unsafe.Pointer(&mode)))
		mode |= ENABLE_VTP_INPUT
		_, _, _ = setConsoleMode.Call(inHandle, uintptr(mode))
	}
}

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Key rune
	Err error
}

// startKeyboardListener starts a goroutine that listens for keyboard input
func startKeyboardListener() chan KeyEvent {
	keyChan := make(chan KeyEvent, 1)

	go func() {
		// Set terminal to raw mode
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			keyChan <- KeyEvent{Err: err}
			return
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		for {
			// Read a single character
			var buf [1]byte
			_, err := os.Stdin.Read(buf[:])
			if err != nil {
				keyChan <- KeyEvent{Err: err}
				return
			}

			// Handle special keys
			var key rune
			switch buf[0] {
			case 3: // Ctrl+C
				os.Exit(0)
			case 32: // Space
				key = ' '
			case 114: // 'r'
				key = 'r'
			case 109: // 'm'
				key = 'm'
			case 115: // 's'
				key = 's'
			case 104: // 'h'
				key = 'h'
			default:
				key = rune(buf[0])
			}

			select {
			case keyChan <- KeyEvent{Key: key}:
			default:
				// Channel is full, skip this key
			}
		}
	}()

	return keyChan
}

// checkKeyPress checks if a key is available (non-blocking)
func checkKeyPress(keyChan chan KeyEvent) (rune, bool) {
	select {
	case event := <-keyChan:
		if event.Err != nil {
			return 0, false
		}
		return event.Key, true
	default:
		return 0, false
	}
}

// getKeyPress reads a single key press (blocking)
func getKeyPress() rune {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read a single character
	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	if err != nil {
		return 0
	}

	// Handle special keys
	switch buf[0] {
	case 3: // Ctrl+C
		os.Exit(0)
	case 32: // Space
		return ' '
	case 114: // 'r'
		return 'r'
	case 109: // 'm'
		return 'm'
	case 115: // 's'
		return 's'
	case 104: // 'h'
		return 'h'
	}

	return rune(buf[0])
}

func printHelp() {
	color.Yellow("Bitcoin Monitor (bmon) - Version %s", version)
	color.White("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	color.Cyan("USAGE:")
	white := color.New(color.FgWhite)
	gray := color.New(color.FgWhite) // Using white as gray equivalent
	white.Print("    ./bmon              ")
	gray.Println("# Interactive mode")
	white.Print("    ./bmon -go          ")
	gray.Println("# Monitor for 15 minutes")
	white.Print("    ./bmon -golong      ")
	gray.Println("# Monitor for 24 hours")
	white.Print("    ./bmon -s           ")
	gray.Println("# Enable sound alerts")
	white.Print("    ./bmon -h           ")
	gray.Println("# Enable history sparkline")
	white.Print("    ./bmon -bu 0.5      ")
	gray.Println("# 0.5 BTC to USD")
	white.Print("    ./bmon -ub 50000    ")
	gray.Println("# $50,000 to BTC")
	white.Print("    ./bmon -us 100      ")
	gray.Println("# $100 to satoshis")
	white.Print("    ./bmon -su 1000000  ")
	gray.Println("# 1M satoshis to USD")
	fmt.Println()

	color.Green("MONITORING MODES:")
	white.Print("    Interactive: ")
	gray.Println("Press Space to start/pause, R to reset, Ctrl+C to exit")
	white.Print("    Go Mode: ")
	gray.Println("15-minute monitoring with 5-second updates")
	white.Print("    Long Go Mode: ")
	gray.Println("24-hour monitoring with 20-second updates")
	fmt.Println()

	color.Magenta("CONTROLS (during monitoring):")
	white.Print("    R - ")
	gray.Println("Reset baseline price and timer")
	white.Print("    M - ")
	gray.Println("Switch between go/golong modes")
	white.Print("    S - ")
	gray.Println("Toggle sound alerts")
	white.Print("    H - ")
	gray.Println("Toggle history sparkline")
	fmt.Println()

	color.Blue("FEATURES:")
	yellow := color.New(color.FgYellow)
	yellow.Print("    • ")
	gray.Println("Real-time Bitcoin price monitoring")
	yellow.Print("    • ")
	gray.Println("Price change indicators (green/red)")
	yellow.Print("    • ")
	gray.Println("Visual price flash alerts")
	yellow.Print("    • ")
	gray.Println("Sound alerts for price movements")
	yellow.Print("    • ")
	gray.Println("Historical price sparkline")
	yellow.Print("    • ")
	gray.Println("BTC/USD conversion tools")
	yellow.Print("    • ")
	gray.Println("Satoshi conversion tools")
	yellow.Print("    • ")
	gray.Println("Automatic API key management")
	fmt.Println()

	color.Red("API KEY:")
	white.Print("    Get a free API key from: ")
	white.Print("    ")
	color.Cyan("https://www.livecoinwatch.com/tools/api")
	white.Print("    The script will guide you through setup on first run.")
	fmt.Println()
	color.White("═══════════════════════════════════════════════════════════════")
}

// formatUSD formats a float with thousands separators and two decimals, like 116,802.19
func formatUSD(v float64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%0.2f", v)
}
