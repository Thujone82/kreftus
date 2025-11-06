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
	"sync"
	"time"

	bspinner "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
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

// Global variables
var (
	apiKey          string
	currentBtcPrice float64
)

// (legacy mode settings removed; TUI handles timing and spinners)

// Command line arguments structure
type Args struct {
	goMode         bool
	golongMode     bool
	kMode          bool
	sound          bool
	sparkline      bool
	help           bool
	conversionMode string
	conversionVal  float64
}

func main() {
	// Ensure Windows console uses UTF-8 and supports ANSI (no-op on non-Windows)
	configureWindowsConsole()
	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

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
	if args.goMode || args.golongMode || args.kMode {
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

	// Handle monitoring modes via Bubble Tea TUI
	runTUI(args)
}

func parseArgs() Args {
	args := Args{}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-go", "-g":
			args.goMode = true
		case "-golong", "-gl":
			args.golongMode = true
		case "-k":
			args.kMode = true
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
	price, err := getBtcPriceWithContext(true)
	if err != nil {
		return err
	}

	currentBtcPrice = price
	return nil
}

func getBtcPrice() (float64, error) {
	return getBtcPriceWithContext(false)
}

func getBtcPriceWithContext(isInitialFetch bool) (float64, error) {
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

	client := &http.Client{Timeout: 10 * time.Second}

	// Retry logic
	maxAttempts := 5
	baseDelay := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create a fresh request each attempt (request bodies are one-shot)
		req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
		if err != nil {
			return 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			if attempt >= maxAttempts {
				// Final failure: show red '5' indicator for TUI
				setRetryIndicator("5", "1", true)
				return 0, fmt.Errorf("API call failed after %d attempts: %v", maxAttempts, err)
			}

			// Show timeout message for initial fetch on first retry
			if isInitialFetch && attempt == 1 {
				fmt.Print("\r")
				color.Yellow("  Timeout, retrying...")
			}

			// Exponential backoff with jitter
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
			jitter := time.Duration(time.Now().UnixNano()%1000) * time.Millisecond
			sleepTime := backoff + jitter

			// Show yellow digit for current attempt (1-4)
			setRetryIndicator(strconv.Itoa(attempt), "11", true)

			time.Sleep(sleepTime)

			// Change to cyan before retry attempt (like spinner does before fetch)
			setRetryIndicator(strconv.Itoa(attempt), "6", true)
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
			if attempt >= maxAttempts {
				setRetryIndicator("5", "1", true)
				return 0, fmt.Errorf("invalid price returned")
			}
			// treat as transient; set yellow digit and retry with backoff
			setRetryIndicator(strconv.Itoa(attempt), "11", true)
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
			jitter := time.Duration(time.Now().UnixNano()%1000) * time.Millisecond
			time.Sleep(backoff + jitter)

			// Change to cyan before retry attempt (like spinner does before fetch)
			setRetryIndicator(strconv.Itoa(attempt), "6", true)
			continue
		}

		// Success: clear indicator so spinner resumes
		clearRetryIndicator()
		return apiResp.Rate, nil
	}

	return 0, fmt.Errorf("failed to get price after all attempts")
}

// (legacy line-warning flag removed; retry indicator handles UI signaling)

// Retry indicator shared state for TUI
var (
	retryMu     sync.RWMutex
	retryActive bool
	retryDigit  string
	retryColor  string
)

func setRetryIndicator(digit string, color string, active bool) {
	retryMu.Lock()
	retryActive = active
	retryDigit = digit
	retryColor = color
	retryMu.Unlock()
}

func clearRetryIndicator() {
	setRetryIndicator("", "", false)
}

func getRetryIndicator() (bool, string, string) {
	retryMu.RLock()
	defer retryMu.RUnlock()
	return retryActive, retryDigit, retryColor
}

// (legacy helpers removed)

// (legacy console width and line clearing helpers removed)

func getSparkline(history []float64) string {
	if len(history) < 2 {
		return strings.Repeat(" ", 14)
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
		return strings.Repeat(" ", 14)
	}

	// Build as runes to measure by glyph count, not bytes
	var sparkRunes []rune
	for _, price := range history {
		normalized := (price - minPrice) / priceRange
		charIndex := int(normalized * float64(len(sparkChars)-1))
		if charIndex >= len(sparkChars) {
			charIndex = len(sparkChars) - 1
		}
		sparkRunes = append(sparkRunes, sparkChars[charIndex])
	}

	// Ensure exactly 14 glyphs (truncate keeping most recent on the right)
	if len(sparkRunes) > 14 {
		sparkRunes = sparkRunes[len(sparkRunes)-14:]
	}
	if len(sparkRunes) < 14 {
		pad := make([]rune, 14-len(sparkRunes))
		for i := range pad {
			pad[i] = ' '
		}
		sparkRunes = append(pad, sparkRunes...)
	}
	sparkline := string(sparkRunes)

	return sparkline
}

// getSparkChars selects characters for the sparkline that the current
// terminal can reliably render. Falls back to CP437 shading on classic conhost.
func getSparkChars() []rune {
	// Use block elements including the lowest bar to avoid blank/glyph issues
	return []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
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
	price, err := getBtcPriceWithContext(true)
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

// (legacy CLI monitoring mode removed; Bubble Tea TUI handles monitoring)

// (legacy interactive mode removed; Bubble Tea TUI provides interactive flow)

// (legacy drawScreen removed)

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

// non-Windows stubs live in console_other.go; Windows implementations live in console_windows.go

// (legacy keyboard input helpers removed; Bubble Tea handles input)

func printHelp() {
	color.Yellow("Bitcoin Monitor (bmon) - Version %s (%s)", version, date)
	color.White("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	color.Cyan("USAGE:")
	white := color.New(color.FgWhite)
	gray := color.New(color.FgWhite) // Using white as gray equivalent
	white.Print("    ./bmon              ")
	gray.Println("# Interactive mode")
	white.Print("    ./bmon -go [Alias -g]")
	gray.Println("# Monitor for 15 minutes")
	white.Print("    ./bmon -golong [Alias -gl]")
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
	white.Print("    Go Long Mode: ")
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

// ------------- Bubble Tea TUI -------------

// tea messages
type tickMsg struct{}
type priceMsg struct {
	price float64
	err   error
}
type fetchStartMsg struct{}

// session modes
const (
	modeLanding     = "landing"
	modeInteractive = "interactive"
	modeGo          = "go"
	modeGoLong      = "golong"
	modeK           = "k"
)

type tuiModel struct {
	// flags
	args Args

	// screen
	width  int
	height int

	// components
	spinner bspinner.Model

	// state
	mode              string
	sessionStartTime  time.Time
	monitorStartPrice float64
	previousPrice     float64
	previousColor     string
	flashUntil        time.Time
	fetchingNow       bool
	soundEnabled      bool
	sparklineEnabled  bool
	history           []float64
	fetchError        error // Track fetch errors to display on exit
}

func newTUIModel(args Args) tuiModel {
	sp := bspinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // white by default

	m := tuiModel{
		args:             args,
		spinner:          sp,
		soundEnabled:     args.sound,
		sparklineEnabled: args.sparkline || args.kMode, // Enable sparkline when -k is used
		history:          []float64{},
		previousColor:    "White",
	}
	// choose start mode (prioritize k, then golong, then go) and set spinner accordingly
	if args.kMode {
		m.mode = modeK
		sp.Spinner = bspinner.Spinner{Frames: []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█", "▉", "▊", "▋", "▌", "▍", "▎"}, FPS: 500 * time.Millisecond}
	} else if args.golongMode {
		m.mode = modeGoLong
		sp.Spinner = bspinner.Spinner{Frames: []string{"▚", "▚", "▚", "▚", "▚", "▚", "▞", "▞", "▞", "▞", "▞", "▞"}, FPS: 500 * time.Millisecond}
	} else if args.goMode {
		m.mode = modeGo
		sp.Spinner = bspinner.Spinner{Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, FPS: 500 * time.Millisecond}
	} else {
		m.mode = modeLanding
		// Default spinner for landing mode (will be used if user switches to go mode)
		sp.Spinner = bspinner.Spinner{Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, FPS: 500 * time.Millisecond}
	}
	m.spinner = sp
	// seed price/history from globals populated earlier
	if currentBtcPrice > 0 {
		m.monitorStartPrice = currentBtcPrice
		m.previousPrice = currentBtcPrice
		m.history = append(m.history, currentBtcPrice)
	}
	m.sessionStartTime = time.Now()
	return m
}

func (m tuiModel) Init() tea.Cmd {
	// Set spinner based on mode
	switch m.mode {
	case modeGo:
		m.spinner.Spinner = bspinner.Spinner{Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, FPS: 500 * time.Millisecond}
	case modeGoLong:
		m.spinner.Spinner = bspinner.Spinner{Frames: []string{"▚", "▚", "▚", "▚", "▚", "▚", "▞", "▞", "▞", "▞", "▞", "▞"}, FPS: 500 * time.Millisecond}
	case modeK:
		m.spinner.Spinner = bspinner.Spinner{Frames: []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█", "▉", "▊", "▋", "▌", "▍", "▎"}, FPS: 500 * time.Millisecond}
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // white by default

	cmds := []tea.Cmd{m.spinner.Tick, tickEvery(500 * time.Millisecond)}
	// if monitoring, schedule first price fetch according to mode interval
	if m.mode == modeGo || m.mode == modeGoLong || m.mode == modeK || m.mode == modeInteractive {
		cmds = append(cmds, fetchPriceCmdAfter(m.currentInterval()))
	}
	return tea.Batch(cmds...)
}

func (m tuiModel) currentInterval() time.Duration {
	switch m.mode {
	case modeGo:
		return 5 * time.Second
	case modeGoLong:
		return 20 * time.Second
	case modeK:
		return 4 * time.Second
	case modeInteractive:
		return 5 * time.Second
	default:
		return 5 * time.Second
	}
}

func (m tuiModel) sessionDuration() time.Duration {
	switch m.mode {
	case modeGo:
		return 15 * time.Minute
	case modeGoLong:
		return 24 * time.Hour
	case modeK:
		return 30 * time.Minute
	case modeInteractive:
		return 5 * time.Minute
	default:
		return 0
	}
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func fetchPriceCmd() tea.Cmd {
	return func() tea.Msg {
		p, err := getBtcPrice()
		return priceMsg{price: p, err: err}
	}
}

func fetchPriceCmdAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return fetchStartMsg{} })
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	// Always let spinner process messages so it can animate
	var sc tea.Cmd
	m.spinner, sc = m.spinner.Update(msg)
	if sc != nil {
		cmds = append(cmds, sc)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case " ":
			switch m.mode {
			case modeLanding:
				m.mode = modeInteractive
				m.sessionStartTime = time.Now()
				m.monitorStartPrice = currentBtcPrice
				m.previousPrice = currentBtcPrice
				cmds = append(cmds, fetchPriceCmd())
			case modeInteractive:
				// pause/return to landing
				m.mode = modeLanding
			}
		case "e":
			// Extend session timeout without changing comparison baseline
			if m.mode == modeGo || m.mode == modeGoLong || m.mode == modeK || m.mode == modeInteractive {
				m.sessionStartTime = time.Now()

				// Visual feedback: flash the screen
				m.flashUntil = time.Now().Add(300 * time.Millisecond)

				// Audio feedback: brief beep if sound is enabled
				if m.soundEnabled {
					playSound(800, 200)
				}
			}
		case "g":
			if m.mode == modeLanding {
				m.mode = modeGo
				m.sessionStartTime = time.Now()
				m.monitorStartPrice = currentBtcPrice
				m.previousPrice = currentBtcPrice
				cmds = append(cmds, fetchPriceCmd())
			}
		case "r":
			if m.mode == modeGo || m.mode == modeGoLong || m.mode == modeK || m.mode == modeInteractive {
				m.monitorStartPrice = currentBtcPrice
				m.sessionStartTime = time.Now()
			}
		case "k":
			// Switch to k mode from go/golong modes
			if m.mode == modeGo || m.mode == modeGoLong {
				m.mode = modeK
				m.sparklineEnabled = true
				m.sessionStartTime = time.Now()
				m.monitorStartPrice = currentBtcPrice
				// Update spinner for k mode
				m.spinner.Spinner = bspinner.Spinner{Frames: []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█", "▉", "▊", "▋", "▌", "▍", "▎"}, FPS: 500 * time.Millisecond}
				cmds = append(cmds, m.spinner.Tick)
			}
		case "m":
			if m.mode == modeGo || m.mode == modeGoLong {
				if m.mode == modeGo {
					m.mode = modeGoLong
					m.spinner.Spinner = bspinner.Spinner{Frames: []string{"▚", "▚", "▚", "▚", "▚", "▚", "▞", "▞", "▞", "▞", "▞", "▞"}, FPS: 500 * time.Millisecond}
				} else {
					m.mode = modeGo
					m.spinner.Spinner = bspinner.Spinner{Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, FPS: 500 * time.Millisecond}
				}
				m.sessionStartTime = time.Now()
				m.monitorStartPrice = currentBtcPrice
				cmds = append(cmds, m.spinner.Tick)
			}
		case "s":
			m.soundEnabled = !m.soundEnabled
			if m.soundEnabled {
				playSound(1200, 350)
			} else {
				playSound(400, 350)
			}
		case "h":
			m.sparklineEnabled = !m.sparklineEnabled
		case "i":
			if m.mode == modeGo || m.mode == modeGoLong || m.mode == modeK {
				m.mode = modeInteractive
				m.sessionStartTime = time.Now()
				m.monitorStartPrice = currentBtcPrice
			}
		}

	case tickMsg:
		// periodic maintenance: duration checks
		// end-of-session logic
		dur := m.sessionDuration()
		if dur > 0 && time.Since(m.sessionStartTime) >= dur {
			switch m.mode {
			case modeInteractive:
				// return to landing after 5 minutes
				m.mode = modeLanding
			case modeK, modeGo, modeGoLong:
				return m, tea.Quit
			}
		}
		// schedule next UI tick
		cmds = append(cmds, tickEvery(500*time.Millisecond))

	case fetchStartMsg:
		m.fetchingNow = true
		cmds = append(cmds, fetchPriceCmd())

	case priceMsg:
		if msg.err != nil {
			// After all retries failed, store error and exit
			m.fetchingNow = false
			m.fetchError = msg.err
			clearRetryIndicator()
			// Exit TUI - error will be displayed after exit
			return m, tea.Quit
		}
		if msg.price > 0 {
			newPrice := msg.price
			// sound cues
			if m.soundEnabled {
				if newPrice >= currentBtcPrice+0.01 {
					playSound(1200, 150)
				} else if newPrice <= currentBtcPrice-0.01 {
					playSound(400, 150)
				}
			}
			currentBtcPrice = newPrice
			// history
			m.history = append(m.history, newPrice)
			if len(m.history) > 14 {
				m.history = m.history[1:]
			}
			// flash logic
			priceChange := newPrice - m.monitorStartPrice
			priceColor := "White"
			if priceChange >= 0.01 {
				priceColor = "Green"
			} else if priceChange <= -0.01 {
				priceColor = "Red"
			}
			flashNeeded := false
			if priceColor != "White" && priceColor != m.previousColor {
				flashNeeded = true
			} else if (priceColor == "Green" && newPrice > m.previousPrice) ||
				(priceColor == "Red" && newPrice < m.previousPrice) {
				flashNeeded = true
			}
			if flashNeeded {
				m.flashUntil = time.Now().Add(500 * time.Millisecond)
			}
			m.previousPrice = newPrice
			m.previousColor = priceColor
			// schedule next fetch
			cmds = append(cmds, fetchPriceCmdAfter(m.currentInterval()))
		}
		m.fetchingNow = false
	}

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	// landing view
	if m.mode == modeLanding {
		title := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("*** BTC Monitor ***") // yellow
		priceLine := fmt.Sprintf("Bitcoin (USD): $%s", formatUSD(currentBtcPrice))
		controls := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Start[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("Space") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("], Go Mode[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("G") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("], Exit[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("Ctrl+C") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("]")
		prompt := "Press Space to start monitoring..."
		return strings.Join([]string{title, priceLine, controls, prompt}, "\n")
	}

	// interactive mode view - multi-line like PS version
	if m.mode == modeInteractive {
		title := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("*** BTC Monitor ***") // yellow

		// Build price line with sparkline and change indicator
		priceChange := currentBtcPrice - m.monitorStartPrice
		priceColor := lipgloss.Color("15") // white
		changeString := ""
		if priceChange >= 0.01 {
			priceColor = lipgloss.Color("2") // green
			changeString = fmt.Sprintf(" [+$%0.2f]", priceChange)
		} else if priceChange <= -0.01 {
			priceColor = lipgloss.Color("1") // red
			changeString = fmt.Sprintf(" [$%0.2f]", priceChange)
		}

		var sparklineOrLabel string
		if m.sparklineEnabled {
			sparklineOrLabel = getSparkline(m.history)
		} else {
			sparklineOrLabel = "Bitcoin (USD):"
		}

		priceLine := fmt.Sprintf("%s $%s%s", sparklineOrLabel, formatUSD(currentBtcPrice), changeString)

		// Apply color and flash effect
		var styledPriceLine string
		if time.Now().Before(m.flashUntil) && (priceChange >= 0.01 || priceChange <= -0.01) {
			// Inverted colors for flash
			styledPriceLine = lipgloss.NewStyle().Background(priceColor).Foreground(lipgloss.Color("0")).Render(priceLine)
		} else {
			styledPriceLine = lipgloss.NewStyle().Foreground(priceColor).Render(priceLine)
		}

		controls := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Pause[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("Space") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("], Reset[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("R") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("], Exit[") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("Ctrl+C") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("]")

		return strings.Join([]string{title, styledPriceLine, controls}, "\n")
	}

	// go/golong mode views (single-line)
	priceChange := currentBtcPrice - m.monitorStartPrice
	priceColor := "White"
	changeString := ""
	if priceChange >= 0.01 {
		priceColor = "Green"
		changeString = fmt.Sprintf(" [+$%0.2f]", priceChange)
	} else if priceChange <= -0.01 {
		priceColor = "Red"
		changeString = fmt.Sprintf(" [$%0.2f]", priceChange)
	}

	var left string
	if m.sparklineEnabled {
		// simple unicode sparkline to match PS feel, relying on VT support
		left = " " + getSparkline(m.history)
	} else {
		left = " Bitcoin (USD):"
	}

	// spinner char or retry indicator
	spinnerChar := ""
	// If a retry is active, show the indicator digit in color; else show spinner
	active, digit, colorCode := getRetryIndicator()
	if active && digit != "" {
		// map retry colors: "11" (yellow) or "1" (red). Only replace the spinner glyph itself.
		spinnerChar = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCode)).Render(digit)
	} else {
		// spinner color: white by default; cyan only on fetch ticks
		if m.fetchingNow {
			m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		} else {
			m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
		}
		spinnerChar = m.spinner.View()
	}

	rest := fmt.Sprintf("%s $%s%s", left, formatUSD(currentBtcPrice), changeString)

	// colorize/invert
	var styledRest string
	if time.Now().Before(m.flashUntil) && (priceColor == "Green" || priceColor == "Red") {
		bg := lipgloss.Color("2") // green
		if priceColor == "Red" {
			bg = lipgloss.Color("1")
		}
		styledRest = lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("0")).Render(rest)
	} else {
		switch priceColor {
		case "Green":
			styledRest = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(rest)
		case "Red":
			styledRest = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(rest)
		default:
			styledRest = rest
		}
	}

	line := spinnerChar + styledRest
	// pad to width
	if m.width > 0 {
		pad := m.width - lipgloss.Width(line)
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		}
	}
	return line + "\n"
}

func runTUI(args Args) {
	m := newTUIModel(args)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModelInterface, _ := p.Run()
	// Type assert to tuiModel to access fetchError field
	finalModel, ok := finalModelInterface.(tuiModel)
	// Clear screen on exit
	clearScreen()
	// If there was a fetch error, show error message
	if ok && finalModel.fetchError != nil {
		color.Red("Failed to fetch price. Check API key or network.")
		os.Exit(1)
	}
}
