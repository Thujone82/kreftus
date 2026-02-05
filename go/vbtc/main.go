package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/Knetic/govaluate"
	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/term"
	"gopkg.in/ini.v1"
)

const (
	startingCapital = 1000.00
	iniFilePath     = "vbtc.ini"
	ledgerFilePath  = "ledger.csv"
)

var (
	sessionStartTime           = time.Now().UTC()
	sessionStartPortfolioValue float64
	initialSessionBtcPrice     float64
	cfg                        *ini.File
	apiData                    *ApiDataResponse
)

// Structs for API responses
type ApiDataResponse struct {
	Rate   float64 `json:"rate"`
	Volume float64 `json:"volume"`
	Delta  struct {
		Day float64 `json:"day"`
	} `json:"delta"`
	FetchTime               time.Time
	Rate24hAgo              float64
	Rate24hHigh             float64
	Rate24hLow              float64
	Rate24hHighTime         time.Time
	Rate24hLowTime          time.Time
	Volatility24h           float64
	Volatility12h           float64
	Volatility12h_old       float64
	Sma1h                   float64
	HistoricalDataFetchTime time.Time
	ApiError                string `json:"-"`
	ApiErrorCode            int    `json:"-"`
}

type HistoryResponse struct {
	History []struct {
		Date int64   `json:"date"`
		Rate float64 `json:"rate"`
	} `json:"history"`
}

// A struct to hold parsed ledger data for easier handling
type LedgerEntry struct {
	TX       string
	USD      float64
	BTC      float64
	BTCPrice float64
	UserBTC  float64
	Time     string
	DateTime time.Time
}

// LedgerSummary holds aggregated data from ledger entries.
type LedgerSummary struct {
	TotalBuyUSD      float64
	TotalSellUSD     float64
	TotalBuyBTC      float64
	TotalSellBTC     float64
	AvgBuyPrice      float64
	AvgSalePrice     float64
	BuyTransactions  int
	SellTransactions int
	MinUSD           float64
	MaxUSD           float64
	FirstTime        time.Time
	LastTime         time.Time
}

// ApiKeyError is a custom error for invalid API key responses (401, 403).
type ApiKeyError struct {
	StatusCode int
}

func (e *ApiKeyError) Error() string {
	return fmt.Sprintf("invalid api key (status %d)", e.StatusCode)
}

// ProviderDownError is a custom error for API provider issues (like 5xx status codes)
// that we want to treat as network errors for UI purposes.
type ProviderDownError struct {
	StatusCode int
	Message    string
}

func (e *ProviderDownError) Error() string {
	return fmt.Sprintf("provider down: %s (status %d)", e.Message, e.StatusCode)
}

// --- Main Application ---
func main() {
	// Check for help flag
	if len(os.Args) > 1 && (os.Args[1] == "-help" || os.Args[1] == "-h" || os.Args[1] == "--help") {
		showHelpScreen(nil)
		return
	}
	// Check for config flag (open config and exit, e.g. to fix API key)
	if len(os.Args) > 1 && (os.Args[1] == "-config" || os.Args[1] == "--config") {
		reader := bufio.NewReader(os.Stdin)
		setup(reader)
		showConfigScreen(reader)
		return
	}

	reader := bufio.NewReader(os.Stdin) // Create the single, authoritative reader.
	setup(reader)
	mainLoop(reader)
}

func setup(reader *bufio.Reader) {
	var err error
	cfg, err = ini.Load(iniFilePath)
	if err != nil {
		fmt.Println("Failed to read ini file, creating a new one.")
		cfg = ini.Empty()
		cfg.Section("Settings").Key("ApiKey").SetValue("")
		cfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", startingCapital))
		cfg.Section("Portfolio").Key("PlayerBTC").SetValue("0.0")
		cfg.Section("Portfolio").Key("PlayerInvested").SetValue("0.0")
		cfg.SaveTo(iniFilePath)
	}

	if cfg.Section("Settings").Key("ApiKey").String() == "" {
		showFirstRunSetup(reader)
	}

	// Perform the initial data fetch to get a complete data object.
	apiData = updateApiData(false)

	// Only after the data is fully fetched, set the session-start values.
	// This prevents the race condition and ensures the initial price is correct.
	if apiData != nil {
		initialSessionBtcPrice = apiData.Rate
	}
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	sessionStartPortfolioValue = getPortfolioValue(playerUSD, playerBTC, apiData)
}

func mainLoop(reader *bufio.Reader) {
	commands := map[string]string{
		"b": "buy", "buy": "buy",
		"s": "sell", "sell": "sell",
		"l": "ledger", "ledger": "ledger",
		"r": "refresh", "refresh": "refresh",
		"c": "config", "config": "config",
		"h": "help", "help": "help",
		"e": "exit", "exit": "exit",
	}

	for {
		showMainScreen()
		fmt.Print("Enter command: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		commandInput := strings.ToLower(parts[0])
		var amount string
		if len(parts) > 1 {
			amount = parts[1]
		}

		var matchedCommands []string
		for _, long := range commands {
			if strings.HasPrefix(long, commandInput) {
				// Avoid adding duplicates
				found := false
				for _, mc := range matchedCommands {
					if mc == long {
						found = true
						break
					}
				}
				if !found {
					matchedCommands = append(matchedCommands, long)
				}
			}
		}

		if len(matchedCommands) == 1 {
			command := matchedCommands[0]
			switch command {
			case "buy":
				// The invokeTrade function now returns the latest data it fetched.
				returnedApiData := invokeTrade(reader, "Buy", amount)
				if returnedApiData != nil {
					apiData = returnedApiData
				}
				// After returning from trade, always reload config to ensure the main screen is perfectly in sync.
				reloadedCfg, err := ini.Load(iniFilePath)
				if err != nil {
					color.Red("Error reloading portfolio from vbtc.ini: %v", err)
					fmt.Println("Press Enter to continue.")
					reader.ReadString('\n')
				} else {
					cfg = reloadedCfg
				}
				// Stale check: refresh API data before showing main screen if >15 minutes old.
				if isApiDataStale() {
					apiData = updateApiData(false)
				}
			case "sell":
				returnedApiData := invokeTrade(reader, "Sell", amount)
				if returnedApiData != nil {
					apiData = returnedApiData
				}
				// After returning from trade, always reload config to ensure the main screen is perfectly in sync.
				reloadedCfg, err := ini.Load(iniFilePath)
				if err != nil {
					color.Red("Error reloading portfolio from vbtc.ini: %v", err)
					fmt.Println("Press Enter to continue.")
					reader.ReadString('\n')
				} else {
					cfg = reloadedCfg
				}
				// Stale check: refresh API data before showing main screen if >15 minutes old.
				if isApiDataStale() {
					apiData = updateApiData(false)
				}
			case "ledger":
				showLedgerScreen(reader)
			case "refresh":
				// Reload config from disk to sync with other potential clients
				reloadedCfg, err := ini.Load(iniFilePath)
				if err != nil {
					color.Red("Error reloading portfolio from vbtc.ini: %v", err)
					fmt.Println("Press Enter to continue.")
					reader.ReadString('\n')
				} else {
					cfg = reloadedCfg
				}
				apiData = updateApiData(false)
			case "config":
				showConfigScreen(reader)
			case "help":
				showHelpScreen(reader)
			case "exit":
				showExitScreen(reader)
				return
			}
		} else if len(matchedCommands) > 1 {
			color.Yellow("Ambiguous command. Did you mean: %s?", strings.Join(matchedCommands, ", "))
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
		} else {
			color.Red("Invalid command. Type 'help' for a list of commands.")
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
		}
	}
}

// --- UI Functions ---

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func showLoadingScreen() {
	clearScreen()
	color.Yellow("Loading Data...")
}

func showMainScreen() {
	clearScreen()

	isNetworkError := apiData != nil && apiData.ApiError == "NetworkError"
	if isNetworkError {
		errorMessage := "API Provider Problem"
		if apiData.ApiErrorCode > 0 {
			errorMessage = fmt.Sprintf("%s (%d)", errorMessage, apiData.ApiErrorCode)
		}
		errorMessage += " - Try again later"
		color.Red(errorMessage)
	}

	// Market Data
	color.New(color.FgYellow).Println("*** Bitcoin Market ***")

	isDataAvailable := apiData != nil && apiData.Rate > 0

	if !isDataAvailable && !isNetworkError {
		color.Red("Could not retrieve market data. Please check your API key in the Config menu.")
	}

	if isDataAvailable {
		priceColor24h := color.New(color.FgWhite)
		if apiData.Rate > apiData.Rate24hAgo {
			priceColor24h = color.New(color.FgGreen)
		} else if apiData.Rate < apiData.Rate24hAgo {
			priceColor24h = color.New(color.FgRed)
		}

		priceColorSession := color.New(color.FgWhite)
		if initialSessionBtcPrice > 0 {
			if apiData.Rate > initialSessionBtcPrice {
				priceColorSession = color.New(color.FgGreen)
			} else if apiData.Rate < initialSessionBtcPrice {
				priceColorSession = color.New(color.FgRed)
			}
		}

		percentChange := 0.0
		if apiData.Rate24hAgo != 0 {
			percentChange = ((apiData.Rate - apiData.Rate24hAgo) / apiData.Rate24hAgo) * 100
		}

		writeAlignedLine("Bitcoin (USD):", fmt.Sprintf("$%s", formatFloat(apiData.Rate, 2)), priceColorSession)

		if apiData.Sma1h > 0 {
			smaColor := color.New(color.FgWhite)
			if apiData.Rate > apiData.Sma1h {
				smaColor = color.New(color.FgGreen)
			} else if apiData.Rate < apiData.Sma1h {
				smaColor = color.New(color.FgRed)
			}
			writeAlignedLine("1H SMA:", fmt.Sprintf("$%s", formatFloat(apiData.Sma1h, 2)), smaColor)
		}

		writeAlignedLine("24H Ago:", fmt.Sprintf("$%s [%+.2f%%]", formatFloat(apiData.Rate24hAgo, 2), percentChange), priceColor24h)

		highDisplay := formatFloat(apiData.Rate24hHigh, 2)
		if !apiData.Rate24hHighTime.IsZero() {
			highDisplay += " (at " + apiData.Rate24hHighTime.Local().Format("15:04") + ")"
		}
		lowDisplay := formatFloat(apiData.Rate24hLow, 2)
		if !apiData.Rate24hLowTime.IsZero() {
			lowDisplay += " (at " + apiData.Rate24hLowTime.Local().Format("15:04") + ")"
		}

		writeAlignedLine("24H High:", fmt.Sprintf("$%s", highDisplay), color.New(color.FgWhite))
		writeAlignedLine("24H Low:", fmt.Sprintf("$%s", lowDisplay), color.New(color.FgWhite))
		if apiData.Volatility24h > 0 {
			volatilityColor := color.New(color.FgWhite)
			if apiData.Volatility12h > apiData.Volatility12h_old {
				volatilityColor = color.New(color.FgGreen)
			} else if apiData.Volatility12h < apiData.Volatility12h_old {
				volatilityColor = color.New(color.FgRed)
			}
			writeAlignedLine("Volatility:", fmt.Sprintf("%.2f%%", apiData.Volatility24h), volatilityColor)
		}
		writeAlignedLine("24H Volume:", fmt.Sprintf("$%s", formatFloat(apiData.Volume, 0)), color.New(color.FgWhite))
		// Time: shows when the (historical) API data was fetched, not when the main modal was loaded.
		dataTime := apiData.FetchTime
		if !apiData.HistoricalDataFetchTime.IsZero() {
			dataTime = apiData.HistoricalDataFetchTime
		}
		writeAlignedLine("Time:", dataTime.Local().Format("010206@150405"), color.New(color.FgCyan))
	}

	// Portfolio
	fmt.Println()
	color.New(color.FgYellow).Println("*** Portfolio ***")
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	playerInvested, _ := cfg.Section("Portfolio").Key("PlayerInvested").Float64()
	portfolioValue := getPortfolioValue(playerUSD, playerBTC, apiData)

	portfolioColor := color.New(color.FgWhite)
	if portfolioValue > startingCapital {
		portfolioColor = color.New(color.FgGreen)
	} else if portfolioValue < startingCapital {
		portfolioColor = color.New(color.FgRed)
	}

	if playerBTC > 0 {
		btcValueDisplay := ""
		if apiData != nil {
			btcValue := playerBTC * apiData.Rate
			btcValueDisplay = fmt.Sprintf(" ($%s)", formatFloat(btcValue, 2))
		}
		writeAlignedLine("Bitcoin:", fmt.Sprintf("%.8f%s", playerBTC, btcValueDisplay), color.New(color.FgWhite))

		investedChange := 0.0
		if playerInvested > 0 && apiData != nil {
			btcValue := playerBTC * apiData.Rate
			investedChange = ((btcValue - playerInvested) / playerInvested) * 100
		}
		investedColor := color.New(color.FgWhite)
		if investedChange > 0.005 { // Add a small tolerance for floating point
			investedColor = color.New(color.FgGreen)
		} else if investedChange < 0 {
			investedColor = color.New(color.FgRed)
		}
		writeAlignedLine("Invested:", fmt.Sprintf("$%s [%+.2f%%]", formatFloat(playerInvested, 2), investedChange), investedColor)
	}

	writeAlignedLine("Cash:", fmt.Sprintf("$%s", formatFloat(playerUSD, 2)), color.New(color.FgWhite))
	writeAlignedLine("Value (USD):", fmt.Sprintf("$%s", formatFloat(portfolioValue, 2)), portfolioColor)

	if sessionStartPortfolioValue > 0 {
		sessionChange := portfolioValue - sessionStartPortfolioValue
		var sessionPercent float64
		if sessionStartPortfolioValue != 0 {
			sessionPercent = (sessionChange / sessionStartPortfolioValue) * 100
		}

		// Round portfolio values for color comparison to avoid floating point inaccuracies.
		roundedCurrentValue := math.Round(portfolioValue*100) / 100
		roundedStartValue := math.Round(sessionStartPortfolioValue*100) / 100

		sessionColor := color.New(color.FgWhite)
		if roundedCurrentValue > roundedStartValue {
			sessionColor = color.New(color.FgGreen)
		} else if roundedCurrentValue < roundedStartValue {
			sessionColor = color.New(color.FgRed)
		}
		sessionDisplay := fmt.Sprintf("%s [%s]", formatProfitLoss(sessionChange, ""), fmt.Sprintf("%+.2f%%", sessionPercent))
		writeAlignedLine("Session P/L:", sessionDisplay, sessionColor)
	}

	fmt.Println()
	color.New(color.FgYellow).Print("Commands: ")
	color.New(color.FgGreen).Print("Buy ")
	color.New(color.FgRed).Print("Sell ")
	color.New(color.FgYellow).Print("Ledger ")
	color.New(color.FgCyan).Print("Refresh ")
	color.New(color.FgHiBlack).Print("Config ")
	color.New(color.FgBlue).Print("Help ")
	color.New(color.FgYellow).Println("Exit")
}

func showFirstRunSetup(reader *bufio.Reader) {
	clearScreen()
	color.Yellow("*** First Time Setup ***")
	color.Green("Get Free Key: https://www.livecoinwatch.com/tools/api")
	for {
		fmt.Print("Please enter your LiveCoinWatch API Key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
		if testApiKey(apiKey) {
			cfg.Section("Settings").Key("ApiKey").SetValue(apiKey)
			cfg.SaveTo(iniFilePath)
			color.Green("API Key saved. Welcome!")
			fmt.Println("Press Enter to start.")
			reader.ReadString('\n')
			break
		} else {
			color.Red("Invalid API Key. Please try again.")
		}
	}
}

func showConfigScreen(reader *bufio.Reader) {
	for {
		clearScreen()
		color.Yellow("*** Configuration ***")
		fmt.Println("1. Update API Key")
		fmt.Println("2. Reset Portfolio")
		fmt.Println("3. Archive Ledger")
		fmt.Println("4. Merge Archived Ledgers")
		fmt.Println("5. Return to Main Screen")
		fmt.Print("Enter your choice (Number 1-5): ")

		// --- Raw Terminal Input Setup ---
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			// Fallback to simple input if not a terminal
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)
			if handleConfigChoice(choice, reader) {
				return
			}
			continue
		}

		oldState, err := term.GetState(fd)
		if err != nil {
			// Fallback to simple input if we can't get terminal state
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)
			if handleConfigChoice(choice, reader) {
				return
			}
			continue
		}

		done := make(chan struct{})
		var wg sync.WaitGroup
		restoreNeeded := true

		defer func() {
			if restoreNeeded {
				close(done)
				wg.Wait()
				term.Restore(fd, oldState)
				reader.Reset(os.Stdin)
			}
		}()

		_, err = term.MakeRaw(fd)
		if err != nil {
			// Fallback to simple input if we can't set raw mode
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)
			if handleConfigChoice(choice, reader) {
				return
			}
			continue
		}

		inputChan := make(chan byte)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(inputChan)
			for {
				b, err := cancellableRead(done)
				if err != nil {
					return
				}
				select {
				case inputChan <- b:
				case <-done:
					return
				}
			}
		}()

		// Wait for user input
		b, ok := <-inputChan
		if !ok {
			return
		}

		// Handle Ctrl+C gracefully in raw mode
		if b == 3 {
			term.Restore(fd, oldState)
			os.Exit(1)
		}

		// Handle Esc key (ASCII 27) to return
		if b == 27 {
			restoreNeeded = false
			close(done)
			wg.Wait()
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)
			return
		}

		// Handle Enter key (empty input = return)
		if b == 13 || b == 10 {
			restoreNeeded = false
			close(done)
			wg.Wait()
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)
			return
		}

		// Handle numeric keys 1-5
		choice := string(b)
		if choice >= "1" && choice <= "5" {
			fmt.Println(choice)
			restoreNeeded = false
			close(done)
			wg.Wait()
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)
			shouldReturn := handleConfigChoice(choice, reader)
			if shouldReturn {
				return
			}
			continue
		}

		// Invalid input, restore and continue loop
		restoreNeeded = false
		close(done)
		wg.Wait()
		term.Restore(fd, oldState)
		reader.Reset(os.Stdin)
		color.Red("Invalid choice. Please try again.")
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
	}
}

func handleConfigChoice(choice string, reader *bufio.Reader) bool {
	switch choice {
	case "1":
		currentKey := cfg.Section("Settings").Key("ApiKey").String()
		if currentKey == "" {
			currentKey = "(not set)"
		}
		color.New(color.FgCyan).Printf("Current API Key: %s\n", currentKey)
		fmt.Print("Enter your new LiveCoinWatch API Key: ")
		newApiKey, _ := reader.ReadString('\n')
		newApiKey = strings.TrimSpace(newApiKey)
		if testApiKey(newApiKey) {
			cfg.Section("Settings").Key("ApiKey").SetValue(newApiKey)
			cfg.SaveTo(iniFilePath)
			color.Green("API Key updated successfully.")
		} else {
			color.Red("The new API Key is invalid. It has not been saved.")
		}
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return false
	case "2":
		color.New(color.FgRed).Print("Are you sure you want to reset your portfolio? This cannot be undone. Type 'YES' to confirm: ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(confirm) == "YES" { // This comparison is already case-sensitive
			cfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", startingCapital))
			cfg.Section("Portfolio").Key("PlayerBTC").SetValue("0.0")
			cfg.Section("Portfolio").Key("PlayerInvested").SetValue("0.0")
			os.Remove(ledgerFilePath)
			cfg.SaveTo(iniFilePath)
			color.Green("Portfolio has been reset.")
		} else {
			fmt.Println("Portfolio reset cancelled.")
		}
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return false
	case "3":
		invokeLedgerArchive(reader)
		return false
	case "4":
		invokeLedgerMerge(reader)
		return false
	case "5", "": // Default to returning if input is empty
		return true
	default:
		color.Red("Invalid choice. Please try again.")
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return false
	}
}

func showHelpScreen(reader *bufio.Reader) {
	clearScreen()
	color.Yellow("Virtual Bitcoin Trader (vBTC) - Version 1.5")
	color.New(color.FgHiBlack).Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	color.New(color.FgCyan).Println("COMMANDS:")
	color.New(color.FgWhite).Print("    buy [amount]     ")
	color.New(color.FgHiBlack).Println("Purchase a specific USD amount of Bitcoin")
	color.New(color.FgWhite).Print("    sell [amount]    ")
	color.New(color.FgHiBlack).Println("Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s)")
	color.New(color.FgWhite).Print("    ledger           ")
	color.New(color.FgHiBlack).Println("View a history of all your transactions")
	color.New(color.FgWhite).Print("    refresh          ")
	color.New(color.FgHiBlack).Println("Manually update the market data")
	color.New(color.FgWhite).Print("    config           ")
	color.New(color.FgHiBlack).Println("Access the configuration menu")
	color.New(color.FgWhite).Print("    help             ")
	color.New(color.FgHiBlack).Println("Show this help screen")
	color.New(color.FgWhite).Print("    exit             ")
	color.New(color.FgHiBlack).Println("Exit the application")
	fmt.Println()

	color.New(color.FgGreen).Println("TIPS:")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("Commands may be shortened (e.g. 'b 10' to buy $10 of BTC)")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("Use 'p' for percentage trades (e.g., '50p' for 50%, '100/3p' for 33.3%)")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("Volatility shows the price swing (High vs Low) over the last 24 hours")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("1H SMA is the average price over the last hour. Green = price is above average")
	fmt.Println()

	color.New(color.FgBlue).Println("REQUIREMENTS:")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("Go runtime")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("An internet connection")
	color.New(color.FgYellow).Print("    • ")
	color.New(color.FgHiBlack).Println("A free API key from https://www.livecoinwatch.com/tools/api")
	fmt.Println()
	color.New(color.FgCyan).Println("COMMAND LINE:")
	color.New(color.FgWhite).Print("    -help, -h, --help  ")
	color.New(color.FgHiBlack).Println("Show this help and exit")
	color.New(color.FgWhite).Print("    -config, --config  ")
	color.New(color.FgHiBlack).Println("Open configuration (e.g. to fix API key) and exit")
	fmt.Println()
	color.New(color.FgHiBlack).Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	if reader != nil {
		fmt.Println("Press Enter or Esc to return to the Main Screen.")

		// --- Raw Terminal Input Setup ---
		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			// Fallback to simple input if not a terminal
			reader.ReadString('\n')
			return
		}

		oldState, err := term.GetState(fd)
		if err != nil {
			// Fallback to simple input if we can't get terminal state
			reader.ReadString('\n')
			return
		}

		done := make(chan struct{})
		var wg sync.WaitGroup

		defer func() {
			close(done)
			wg.Wait()
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)
		}()

		_, err = term.MakeRaw(fd)
		if err != nil {
			// Fallback to simple input if we can't set raw mode
			reader.ReadString('\n')
			return
		}

		inputChan := make(chan byte)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(inputChan)
			for {
				b, err := cancellableRead(done)
				if err != nil {
					return
				}
				select {
				case inputChan <- b:
				case <-done:
					return
				}
			}
		}()

		// Wait for user input
		for {
			b, ok := <-inputChan
			if !ok {
				return
			}

			// Handle Ctrl+C gracefully in raw mode
			if b == 3 {
				term.Restore(fd, oldState)
				os.Exit(1)
			}

			// Handle Enter key (13 is Carriage Return, 10 is Line Feed)
			if b == 13 || b == 10 {
				return
			}

			// Handle Esc key (ASCII 27) to return
			if b == 27 {
				return
			}
		}
	}
}

func showLedgerScreen(reader *bufio.Reader) {
	clearScreen()
	color.Yellow("*** Ledger ***")

	// All data (current + archives) for summary; current log only for table
	allEntries, err := readAllLedgerEntries()
	if err != nil {
		color.Red("Error reading historical ledger data: %v", err)
		fmt.Println("\nPress Enter to return to Main screen")
		reader.ReadString('\n')
		return
	}
	hasAnyData := len(allEntries) > 0
	ledgerEntries, _ := readAndParseLedger() // current log only (for table)
	currentHasRows := len(ledgerEntries) > 0

	if !hasAnyData {
		fmt.Println("You have not made any transactions yet.")
		fmt.Println("\nPress Enter to return to Main screen")
		reader.ReadString('\n')
		return
	}

	if currentHasRows {
		// Sort entries by date and time to ensure chronological order for display.
		sort.Slice(ledgerEntries, func(i, j int) bool {
			return ledgerEntries[i].DateTime.Before(ledgerEntries[j].DateTime)
		})

		// 2. Dynamically calculate column widths for proper alignment.
		columnOrder := []string{"TX", "USD", "BTC", "BTC(USD)", "User BTC", "Time"}
		widths := map[string]int{
			"TX": len("TX"), "USD": len("USD"), "BTC": len("BTC"),
			"BTC(USD)": len("BTC(USD)"), "User BTC": len("User BTC"), "Time": len("Time"),
		}

		for _, entry := range ledgerEntries {
			if len(entry.TX) > widths["TX"] {
				widths["TX"] = len(entry.TX)
			}
			if len(formatFloat(entry.USD, 2)) > widths["USD"] {
				widths["USD"] = len(formatFloat(entry.USD, 2))
			}
			if len(fmt.Sprintf("%.8f", entry.BTC)) > widths["BTC"] {
				widths["BTC"] = len(fmt.Sprintf("%.8f", entry.BTC))
			}
			if len(formatFloat(entry.BTCPrice, 2)) > widths["BTC(USD)"] {
				widths["BTC(USD)"] = len(formatFloat(entry.BTCPrice, 2))
			}
			if len(fmt.Sprintf("%.8f", entry.UserBTC)) > widths["User BTC"] {
				widths["User BTC"] = len(fmt.Sprintf("%.8f", entry.UserBTC))
			}
			if len(entry.Time) > widths["Time"] {
				widths["Time"] = len(entry.Time)
			}
		}

		// 3. Create header and separator strings based on dynamic widths.
		var headerParts []string
		for _, colName := range columnOrder {
			width := widths[colName]
			headerParts = append(headerParts, fmt.Sprintf("%-*s", width, colName))
		}
		header := strings.Join(headerParts, "  ")
		separator := strings.Repeat("-", len(header))
		fmt.Println(header)
		fmt.Println(separator)

		// 4. Print data rows
		sessionStarted := false
		// Truncate the session start time to the second. This is crucial for a correct comparison,
		// as the timestamps stored in the ledger only have second-level precision.
		sessionStartTruncated := sessionStartTime.Truncate(time.Second)
		for _, entry := range ledgerEntries {
			// Compare the entry's time (which has second-level precision) with the truncated session start time.
			// This prevents issues where a trade in the same second as launch would be considered "before".
			if !sessionStarted && !entry.DateTime.IsZero() && !entry.DateTime.Before(sessionStartTruncated) {
				totalWidth := len(separator)
				sessionText := "*** Current Session Start ***"
				paddingLength := 0
				if totalWidth > len(sessionText) {
					paddingLength = (totalWidth - len(sessionText)) / 2
				}
				centeredText := strings.Repeat(" ", paddingLength) + sessionText
				color.White(centeredText)
				sessionStarted = true
			}

			rowColor := color.New(color.FgGreen)
			if entry.TX == "Sell" {
				rowColor = color.New(color.FgRed)
			}

			// Build the row dynamically with correct alignment.
			rowParts := []string{
				fmt.Sprintf("%-*s", widths["TX"], entry.TX),                  // Left-align TX
				fmt.Sprintf("%*s", widths["USD"], formatFloat(entry.USD, 2)), // Right-align numbers
				fmt.Sprintf("%*s", widths["BTC"], fmt.Sprintf("%.8f", entry.BTC)),
				fmt.Sprintf("%*s", widths["BTC(USD)"], formatFloat(entry.BTCPrice, 2)),
				fmt.Sprintf("%*s", widths["User BTC"], fmt.Sprintf("%.8f", entry.UserBTC)),
				fmt.Sprintf("%*s", widths["Time"], entry.Time),
			}
			row := strings.Join(rowParts, "  ")
			rowColor.Println(row)
		}
	} else {
		fmt.Println("Log Empty")
	}

	// Ledger Summary from all data (current + archives)
	summary := getLedgerTotals(allEntries)
	sessionSummary := getSessionSummary()
	fmt.Println()
	color.Yellow("*** Ledger Summary ***")
	summaryValueStartColumn := 22 // Align with portfolio summary

	// Portfolio Summary Section
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	portfolioValue := getPortfolioValue(playerUSD, playerBTC, apiData)

	portfolioColor := color.New(color.FgWhite)
	if portfolioValue > startingCapital {
		portfolioColor = color.New(color.FgGreen)
	} else if portfolioValue < startingCapital {
		portfolioColor = color.New(color.FgRed)
	}

	// Portfolio Value with session delta in [] (green if up, red if down)
	fmt.Print("Portfolio Value:")
	fmt.Print(strings.Repeat(" ", summaryValueStartColumn-len("Portfolio Value:")))
	portfolioColor.Print(fmt.Sprintf("$%s", formatFloat(portfolioValue, 2)))
	if sessionStartPortfolioValue > 0 {
		sessionPortfolioDelta := portfolioValue - sessionStartPortfolioValue
		deltaStr := fmt.Sprintf(" [%+.2f]", sessionPortfolioDelta)
		deltaColor := color.New(color.FgWhite)
		if sessionPortfolioDelta > 0 {
			deltaColor = color.New(color.FgGreen)
		} else if sessionPortfolioDelta < 0 {
			deltaColor = color.New(color.FgRed)
		}
		deltaColor.Println(deltaStr)
	} else {
		fmt.Println()
	}

	// Trading Statistics Section (all-time with session in [])
	if summary.TotalBuyUSD > 0 {
		v := fmt.Sprintf("$%s", formatFloat(summary.TotalBuyUSD, 2))
		if sessionSummary != nil {
			v += fmt.Sprintf(" [$%s]", formatFloat(sessionSummary.TotalBuyUSD, 2))
		}
		writeAlignedLine("Total Bought (USD):", v, color.New(color.FgGreen), summaryValueStartColumn)
		btcVal := fmt.Sprintf("%.8f", summary.TotalBuyBTC)
		if sessionSummary != nil {
			btcVal += fmt.Sprintf(" [%.8f]", sessionSummary.TotalBuyBTC)
		}
		writeAlignedLine("Total Bought (BTC):", btcVal, color.New(color.FgGreen), summaryValueStartColumn)
	}

	// Display additional statistics
	totalTransactions := summary.BuyTransactions + summary.SellTransactions
	if totalTransactions > 0 {
		txVal := fmt.Sprintf("%d", totalTransactions)
		if sessionSummary != nil {
			sessionTx := sessionSummary.BuyTransactions + sessionSummary.SellTransactions
			txVal += fmt.Sprintf(" [%d]", sessionTx)
		}
		writeAlignedLine("Transaction Count:", txVal, color.New(color.FgWhite), summaryValueStartColumn)
	}

	if summary.AvgBuyPrice > 0 {
		v := fmt.Sprintf("$%s", formatFloat(summary.AvgBuyPrice, 2))
		if sessionSummary != nil && sessionSummary.AvgBuyPrice > 0 {
			v += fmt.Sprintf(" [$%s]", formatFloat(sessionSummary.AvgBuyPrice, 2))
		} else if sessionSummary != nil {
			v += " [$0.00]"
		}
		writeAlignedLine("Average Purchase:", v, color.New(color.FgGreen), summaryValueStartColumn)
	}

	if summary.AvgSalePrice > 0 {
		v := fmt.Sprintf("$%s", formatFloat(summary.AvgSalePrice, 2))
		if sessionSummary != nil && sessionSummary.AvgSalePrice > 0 {
			v += fmt.Sprintf(" [$%s]", formatFloat(sessionSummary.AvgSalePrice, 2))
		} else if sessionSummary != nil {
			v += " [$0.00]"
		}
		writeAlignedLine("Average Sale:", v, color.New(color.FgRed), summaryValueStartColumn)
	}
	if totalTransactions > 0 && summary.MaxUSD >= summary.MinUSD {
		writeAlignedLine("Tx Range:", fmt.Sprintf("$%s - $%s", formatFloat(summary.MinUSD, 2), formatFloat(summary.MaxUSD, 2)), color.New(color.FgWhite), summaryValueStartColumn)
		if sessionSummary != nil && sessionSummary.MaxUSD >= sessionSummary.MinUSD {
			writeAlignedLine("Session Tx Range:", fmt.Sprintf("$%s - $%s", formatFloat(sessionSummary.MinUSD, 2), formatFloat(sessionSummary.MaxUSD, 2)), color.New(color.FgWhite), summaryValueStartColumn)
		}
	}
	totalLen := formatDuration(summary.FirstTime, summary.LastTime)
	if totalLen != "" {
		timeVal := totalLen
		if sessionSummary != nil {
			sessionLen := formatDuration(sessionStartTime, time.Now().UTC())
			if sessionLen != "" {
				timeVal += " [" + sessionLen + "]"
			}
		}
		writeAlignedLine("Time:", timeVal, color.New(color.FgWhite), summaryValueStartColumn)
	}
	// Cadence: time per trade (ledger and session); slower = red, quicker = green
	if totalTransactions > 0 && !summary.FirstTime.IsZero() && summary.LastTime.After(summary.FirstTime) {
		ledgerDuration := summary.LastTime.Sub(summary.FirstTime)
		ledgerCadenceDur := ledgerDuration / time.Duration(totalTransactions)
		ledgerCadenceStr := formatCadence(ledgerCadenceDur)
		sessionCadenceStr := ""
		var sessionCadenceDur time.Duration
		sessionTx := 0
		if sessionSummary != nil {
			sessionTx = sessionSummary.BuyTransactions + sessionSummary.SellTransactions
			if sessionTx > 0 {
				// Session cadence: session time (app start to now) / session trades
				sessionDuration := time.Now().UTC().Sub(sessionStartTime)
				sessionCadenceDur = sessionDuration / time.Duration(sessionTx)
				sessionCadenceStr = formatCadence(sessionCadenceDur)
			}
		}
		if ledgerCadenceStr != "" {
			ledgerIsSlower := sessionCadenceStr == "" || ledgerCadenceDur >= sessionCadenceDur
			writeAlignedLineCadence("Cadence:", ledgerCadenceStr, sessionCadenceStr, ledgerIsSlower, summaryValueStartColumn)
		}
	}

	fmt.Println("\nPress Enter to return to Main screen, or R to refresh")

	// --- Raw Terminal Input Setup ---
	// Get the file descriptor for standard input.
	fd := int(os.Stdin.Fd())

	// Check if we are in a terminal, which is required for raw mode.
	if !term.IsTerminal(fd) {
		// Fallback to simple input if not a terminal
		reader.ReadString('\n')
		return
	}

	// Save the original terminal state so we can restore it later.
	oldState, err := term.GetState(fd)
	if err != nil {
		// Fallback to simple input if we can't get terminal state
		reader.ReadString('\n')
		return
	}

	done := make(chan struct{}) // Channel to signal the goroutine to stop.
	var wg sync.WaitGroup
	restoreNeeded := true // Flag to track if we need to restore terminal

	// CRITICAL: Ensure the goroutine is stopped, terminal is restored,
	// and the input buffer is reset when the function exits.
	defer func() {
		if restoreNeeded {
			close(done) // 1. Signal the input goroutine to stop.
			wg.Wait()   // 2. Wait for the input goroutine to finish before proceeding.
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)
		}
	}()

	// Put the terminal into raw mode using the original descriptor.
	_, err = term.MakeRaw(fd)
	if err != nil {
		// Fallback to simple input if we can't set raw mode
		reader.ReadString('\n')
		return
	}

	// Create a channel to receive input from a non-blocking goroutine.
	inputChan := make(chan byte)
	wg.Add(1) // Increment the WaitGroup counter before starting the goroutine.
	go func() {
		defer wg.Done()
		defer close(inputChan)
		for {
			// cancellableRead is a platform-aware, non-blocking read.
			b, err := cancellableRead(done)
			if err != nil {
				// This error is expected when 'done' is closed, so we just exit.
				return
			}
			// We got a character, try to send it but don't block.
			select {
			case inputChan <- b:
			case <-done:
				return
			}
		}
	}()

	// Wait for user input
	for {
		b, ok := <-inputChan
		if !ok {
			// Channel closed, exit loop.
			return
		}

		// Handle Ctrl+C gracefully in raw mode.
		if b == 3 {
			term.Restore(fd, oldState) // Restore terminal state BEFORE exiting.
			os.Exit(1)
		}

		// Handle Enter key (13 is Carriage Return, 10 is Line Feed)
		if b == 13 || b == 10 {
			return // Return to main screen
		}

		// Handle Esc key (ASCII 27) - could be Esc or start of arrow key sequence
		if b == 27 {
			// Check if this is an arrow key sequence (ESC [ A/B/C/D)
			arrowDetected := false

			// Use a very short timeout to check for arrow key sequence
			select {
			case nextByte, ok := <-inputChan:
				if ok && nextByte == '[' {
					// This looks like an arrow key sequence, read the direction
					select {
					case arrowByte, ok := <-inputChan:
						if ok {
							switch arrowByte {
							case 'C': // Right arrow = R (refresh)
								arrowDetected = true
								b = 'r'
							}
						}
					case <-time.After(10 * time.Millisecond):
						// Timeout - not an arrow key, treat as plain Esc
					}
				}
			case <-time.After(10 * time.Millisecond):
				// Timeout - treat as plain Esc
			}

			if !arrowDetected {
				// Plain Esc key pressed - return to main screen
				return
			}
		}

		// Handle 'R' or 'r' for refresh (or Right Arrow which was converted to 'r' above)
		if b == 'R' || b == 'r' {
			// Close the input goroutine and restore terminal before recursive call
			restoreNeeded = false // Prevent defer from restoring again
			close(done)
			wg.Wait()
			term.Restore(fd, oldState)
			reader.Reset(os.Stdin)

			// Reload config from disk (for portfolio values)
			reloadedCfg, err := ini.Load(iniFilePath)
			if err != nil {
				color.Red("Error reloading portfolio from vbtc.ini: %v", err)
				fmt.Println("Press Enter to continue.")
				reader.ReadString('\n')
				return
			}
			cfg = reloadedCfg

			// Stale check: refresh API data if >15 minutes old so Portfolio Value on summary is current.
			if isApiDataStale() {
				apiData = updateApiData(false)
			}

			// Recursively call showLedgerScreen to redraw with fresh ledger data
			showLedgerScreen(reader)
			return
		}
	}
}

func showExitScreen(reader *bufio.Reader) {
	clearScreen()
	color.Yellow("*** Portfolio Summary ***")
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	finalValue := getPortfolioValue(playerUSD, playerBTC, apiData)
	profit := finalValue - startingCapital

	profitColor := color.New(color.FgWhite)
	if profit > 0 {
		profitColor = color.New(color.FgGreen)
	} else if profit < 0 {
		profitColor = color.New(color.FgRed)
	}

	writeAlignedLine("Portfolio Value:", fmt.Sprintf("$%s", formatFloat(finalValue, 2)), profitColor)

	// --- Session Summary ---
	fmt.Println()
	color.Yellow("*** Session Summary ***")
	sessionValueStartColumn := 22 // Use a consistent start column for this block

	summary := getSessionSummary()
	if summary != nil {
		totalTransactions := summary.BuyTransactions + summary.SellTransactions
		writeAlignedLine("Transactions:", fmt.Sprintf("%d", totalTransactions), color.New(color.FgWhite), sessionValueStartColumn)
	}

	finalBtcPrice := initialSessionBtcPrice
	if apiData != nil {
		finalBtcPrice = apiData.Rate
	}

	// Round to 2 decimal places for "to the penny" comparison
	roundedInitial := math.Round(initialSessionBtcPrice*100) / 100
	roundedFinal := math.Round(finalBtcPrice*100) / 100
	sessionPriceColor := color.New(color.FgWhite)
	if roundedFinal > roundedInitial {
		sessionPriceColor = color.New(color.FgGreen)
	} else if roundedFinal < roundedInitial {
		sessionPriceColor = color.New(color.FgRed)
	}

	writeAlignedLine("Start BTC(USD):", fmt.Sprintf("$%s", formatFloat(initialSessionBtcPrice, 2)), color.New(color.FgWhite), sessionValueStartColumn)
	writeAlignedLine("End BTC(USD):", fmt.Sprintf("$%s", formatFloat(finalBtcPrice, 2)), sessionPriceColor, sessionValueStartColumn)

	if sessionStartPortfolioValue > 0 {
		sessionChange := finalValue - sessionStartPortfolioValue
		var sessionPercent float64
		if sessionStartPortfolioValue != 0 {
			sessionPercent = (sessionChange / sessionStartPortfolioValue) * 100
		}

		// Round portfolio values for color comparison to avoid floating point inaccuracies.
		roundedFinalValue := math.Round(finalValue*100) / 100
		roundedStartValue := math.Round(sessionStartPortfolioValue*100) / 100

		sessionColor := color.New(color.FgWhite)
		if roundedFinalValue > roundedStartValue {
			sessionColor = color.New(color.FgGreen)
		} else if roundedFinalValue < roundedStartValue {
			sessionColor = color.New(color.FgRed)
		}
		sessionDisplay := fmt.Sprintf("%s [%s]", formatProfitLoss(sessionChange, ""), fmt.Sprintf("%+.2f%%", sessionPercent))
		writeAlignedLine("P/L:", sessionDisplay, sessionColor, sessionValueStartColumn)
	}

	if summary != nil {
		if summary.TotalBuyUSD > 0 {
			writeAlignedLine("Total Bought (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalBuyUSD, 2)), color.New(color.FgGreen), sessionValueStartColumn)
			writeAlignedLine("Total Bought (BTC):", fmt.Sprintf("%.8f", summary.TotalBuyBTC), color.New(color.FgGreen), sessionValueStartColumn)
		}
		if summary.TotalSellUSD > 0 {
			writeAlignedLine("Total Sold (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalSellUSD, 2)), color.New(color.FgRed), sessionValueStartColumn)
			writeAlignedLine("Total Sold (BTC):", fmt.Sprintf("%.8f", summary.TotalSellBTC), color.New(color.FgRed), sessionValueStartColumn)
		}
		if summary.AvgBuyPrice > 0 {
			writeAlignedLine("Average Purchase:", fmt.Sprintf("$%s", formatFloat(summary.AvgBuyPrice, 2)), color.New(color.FgGreen), sessionValueStartColumn)
		}
		if summary.AvgSalePrice > 0 {
			writeAlignedLine("Average Sale:", fmt.Sprintf("$%s", formatFloat(summary.AvgSalePrice, 2)), color.New(color.FgRed), sessionValueStartColumn)
		}
		if summary.MaxUSD >= summary.MinUSD {
			writeAlignedLine("Session Tx Range:", fmt.Sprintf("$%s - $%s", formatFloat(summary.MinUSD, 2), formatFloat(summary.MaxUSD, 2)), color.New(color.FgWhite), sessionValueStartColumn)
		}
		sessionLen := formatDuration(sessionStartTime, time.Now().UTC())
		if sessionLen != "" {
			writeAlignedLine("Time:", sessionLen, color.New(color.FgWhite), sessionValueStartColumn)
		}
		sessionTx := summary.BuyTransactions + summary.SellTransactions
		if sessionTx > 0 {
			sessionDuration := time.Now().UTC().Sub(sessionStartTime)
			sessionCadenceDur := sessionDuration / time.Duration(sessionTx)
			sessionCadenceStr := formatCadence(sessionCadenceDur)
			if sessionCadenceStr != "" {
				writeAlignedLine("Cadence:", sessionCadenceStr, color.New(color.FgWhite), sessionValueStartColumn)
			}
		}
	}

	// --- Comprehensive Ledger Summary ---
	fmt.Println()
	color.Yellow("*** Trading History Summary ***")
	allEntries, err := readAllLedgerEntries()
	if err == nil && len(allEntries) > 0 {
		allTimeSummary := getLedgerTotals(allEntries)
		ledgerValueStartColumn := 22 // Use consistent column alignment

		// Display transaction counts
		totalTransactions := allTimeSummary.BuyTransactions + allTimeSummary.SellTransactions
		if totalTransactions > 0 {
			writeAlignedLine("Total Transactions:", fmt.Sprintf("%d", totalTransactions), color.New(color.FgWhite), ledgerValueStartColumn)
		}

		// Display totals
		if allTimeSummary.TotalBuyUSD > 0 {
			writeAlignedLine("Total Bought (USD):", fmt.Sprintf("$%s", formatFloat(allTimeSummary.TotalBuyUSD, 2)), color.New(color.FgGreen), ledgerValueStartColumn)
			writeAlignedLine("Total Bought (BTC):", fmt.Sprintf("%.8f", allTimeSummary.TotalBuyBTC), color.New(color.FgGreen), ledgerValueStartColumn)
		}
		if allTimeSummary.TotalSellUSD > 0 {
			writeAlignedLine("Total Sold (USD):", fmt.Sprintf("$%s", formatFloat(allTimeSummary.TotalSellUSD, 2)), color.New(color.FgRed), ledgerValueStartColumn)
			writeAlignedLine("Total Sold (BTC):", fmt.Sprintf("%.8f", allTimeSummary.TotalSellBTC), color.New(color.FgRed), ledgerValueStartColumn)
		}

		// Display average prices
		if allTimeSummary.AvgBuyPrice > 0 {
			writeAlignedLine("Average Purchase:", fmt.Sprintf("$%s", formatFloat(allTimeSummary.AvgBuyPrice, 2)), color.New(color.FgGreen), ledgerValueStartColumn)
		}
		if allTimeSummary.AvgSalePrice > 0 {
			writeAlignedLine("Average Sale:", fmt.Sprintf("$%s", formatFloat(allTimeSummary.AvgSalePrice, 2)), color.New(color.FgRed), ledgerValueStartColumn)
		}
		exitTxCount := allTimeSummary.BuyTransactions + allTimeSummary.SellTransactions
		if exitTxCount > 0 && allTimeSummary.MaxUSD >= allTimeSummary.MinUSD {
			writeAlignedLine("Tx Range:", fmt.Sprintf("$%s - $%s", formatFloat(allTimeSummary.MinUSD, 2), formatFloat(allTimeSummary.MaxUSD, 2)), color.New(color.FgWhite), ledgerValueStartColumn)
		}
		exitTimeLen := formatDuration(allTimeSummary.FirstTime, allTimeSummary.LastTime)
		if exitTimeLen != "" {
			writeAlignedLine("Time:", exitTimeLen, color.New(color.FgWhite), ledgerValueStartColumn)
		}

		// Net BTC Position
		netBTC := allTimeSummary.TotalBuyBTC - allTimeSummary.TotalSellBTC
		netBTCColor := color.New(color.FgWhite)
		if netBTC > 0 {
			netBTCColor = color.New(color.FgGreen)
		} else if netBTC < 0 {
			netBTCColor = color.New(color.FgRed)
		}
		writeAlignedLine("Net BTC Position:", fmt.Sprintf("%.8f", netBTC), netBTCColor, ledgerValueStartColumn)

		// Net Profit/Loss USD
		netProfitLoss := allTimeSummary.TotalSellUSD - allTimeSummary.TotalBuyUSD
		netPLColor := color.New(color.FgWhite)
		if netProfitLoss > 0 {
			netPLColor = color.New(color.FgGreen)
		} else if netProfitLoss < 0 {
			netPLColor = color.New(color.FgRed)
		}
		writeAlignedLine("Net Trading P/L (USD):", fmt.Sprintf("$%s", formatFloat(netProfitLoss, 2)), netPLColor, ledgerValueStartColumn)
	} else {
		color.New(color.FgCyan).Println("No trading history found.")
	}

	// Pause the screen if the application was likely run by double-clicking.
	// We do this by checking if the parent process is a known interactive shell.
	parentName, err := getParentProcessName()
	isInteractiveShell := false
	if err == nil {
		parentName = strings.ToLower(strings.TrimSuffix(parentName, ".exe"))
		// List of shells where we should NOT pause.
		interactiveShells := []string{"cmd", "powershell", "pwsh", "wt", "alacritty", "bash", "zsh", "fish"}
		for _, shell := range interactiveShells {
			if parentName == shell {
				isInteractiveShell = true
				break
			}
		}
	} else {
		fmt.Printf("\nCould not determine parent process: %v. Pausing by default.", err)
	}
	shouldPause := !isInteractiveShell

	if shouldPause {
		fmt.Println("\nPress Enter to exit.")
		reader.ReadString('\n')
	}
}

func writeAlignedLine(label, value string, c *color.Color, startColumn ...int) {
	valueStartColumn := 22 // Default value
	if len(startColumn) > 0 {
		valueStartColumn = startColumn[0]
	}
	padding := valueStartColumn - len(label)
	if padding < 0 {
		padding = 0
	}
	fmt.Print(label)
	fmt.Print(strings.Repeat(" ", padding))
	c.Println(value)
}

// --- API and Data Functions ---

func fetchCurrentPriceData(apiKey string) (*ApiDataResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is empty")
	}
	jsonData := map[string]string{"currency": "USD", "code": "BTC", "meta": "false"}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json for current price: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.livecoinwatch.com/coins/single", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for current price: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for current price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return nil, &ProviderDownError{StatusCode: resp.StatusCode, Message: "API provider returned server error"}
	}

	// Check for user-fixable API key errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &ApiKeyError{StatusCode: resp.StatusCode}
	}

	if resp.StatusCode != http.StatusOK {
		// Treat any other non-OK status as a provider problem, so the code is displayed.
		return nil, &ProviderDownError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("API provider returned non-OK status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for current price: %w", err)
	}

	var data ApiDataResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for current price: %w", err)
	}
	data.FetchTime = time.Now().UTC()
	return &data, nil
}

func getHistoricalData(apiKey string, start, end int64) (*HistoryResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is empty")
	}

	jsonData := map[string]interface{}{"currency": "USD", "code": "BTC", "start": start, "end": end, "meta": false}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json for historical price: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.livecoinwatch.com/coins/single/history", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for historical price: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for historical price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return nil, &ProviderDownError{StatusCode: resp.StatusCode, Message: "API provider returned server error for history"}
	}

	// Check for user-fixable API key errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &ApiKeyError{StatusCode: resp.StatusCode}
	}

	if resp.StatusCode != http.StatusOK {
		// Treat any other non-OK status as a provider problem, so the code is displayed.
		return nil, &ProviderDownError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("API provider for history returned non-OK status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for historical price: %w", err)
	}

	var history HistoryResponse
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for historical price: %w", err)
	}
	return &history, nil
}

// isApiDataStale returns true if apiData is nil or older than 15 minutes (so we should refresh before showing main screen).
func isApiDataStale() bool {
	if apiData == nil {
		return true
	}
	return time.Since(apiData.HistoricalDataFetchTime).Minutes() > 15
}

func updateApiData(skipHistorical bool) *ApiDataResponse {
	showLoadingScreen()
	apiKey := cfg.Section("Settings").Key("ApiKey").String()

	// 1. Always fetch the latest current price data.
	newData, err := fetchCurrentPriceData(apiKey)
	if err != nil {
		fmt.Printf("Error fetching current price data: %v\n", err)

		var apiKeyErr *ApiKeyError
		// If it's an API key error, we want the generic "Could not retrieve..." message to show.
		// We achieve this by returning nil (or old data without an error flag).
		if errors.As(err, &apiKeyErr) {
			if apiData != nil {
				apiData.ApiError = "" // Clear any previous network error
			}
			return apiData // Return old data, no error flag.
		}

		// For any other error, we assume it's a provider/network issue.
		// We return the old data but with the ApiError flag set.
		if apiData != nil {
			apiData.ApiError = "NetworkError"
			var providerDownErr *ProviderDownError
			if errors.As(err, &providerDownErr) {
				apiData.ApiErrorCode = providerDownErr.StatusCode
			} else {
				apiData.ApiErrorCode = 0 // Reset if it's a different network error
			}
			return apiData
		}

		// No old data to return, so create a new object just to hold the error flag.
		newErrorData := &ApiDataResponse{ApiError: "NetworkError"}
		var providerDownErr *ProviderDownError
		if errors.As(err, &providerDownErr) {
			newErrorData.ApiErrorCode = providerDownErr.StatusCode
		}
		return newErrorData
	}

	if !skipHistorical {
		// 2. Check if historical data needs to be updated (stale if nil or > 15 mins old).
		isStale := false
		if apiData == nil {
			isStale = true
		} else {
			// apiData is not nil here, so we can safely access its fields.
			if time.Since(apiData.HistoricalDataFetchTime).Minutes() > 15 {
				isStale = true
			} else if newData.Rate > apiData.Rate24hHigh || newData.Rate < apiData.Rate24hLow {
				// Also mark as stale if the current price breaks the known 24h high/low.
				isStale = true
			}
		}

		if isStale {
			color.Yellow("Fetching updated historical data...")
			time.Sleep(1 * time.Second) // Let user see the message

			end := time.Now().UTC()
			start := end.Add(-24 * time.Hour)
			history, historyErr := getHistoricalData(apiKey, start.UnixMilli(), end.UnixMilli())

			if historyErr == nil && history != nil && len(history.History) > 0 {
				// Successfully fetched new historical data.
				minRate24h, maxRate24h := math.MaxFloat64, 0.0
				minRate12hRecent, maxRate12hRecent := math.MaxFloat64, 0.0
				minRate12hOld, maxRate12hOld := math.MaxFloat64, 0.0
				var highTime, lowTime int64
				var closestRate float64
				minDiff := int64(math.MaxInt64)

				now := time.Now().UTC()
				startTs := now.Add(-24 * time.Hour).UnixMilli()
				midpointTs := now.Add(-12 * time.Hour).UnixMilli()

				// Sort history by date to ensure correct order for SMA calculation
				sort.Slice(history.History, func(i, j int) bool {
					return history.History[i].Date < history.History[j].Date
				})

				for _, p := range history.History {
					// Overall 24h stats
					if p.Rate > maxRate24h {
						maxRate24h = p.Rate
						highTime = p.Date
					}
					if p.Rate < minRate24h {
						minRate24h = p.Rate
						lowTime = p.Date
					}

					// Split for 12h volatility stats
					if p.Date >= midpointTs { // Recent 12 hours
						if p.Rate > maxRate12hRecent {
							maxRate12hRecent = p.Rate
						}
						if p.Rate < minRate12hRecent {
							minRate12hRecent = p.Rate
						}
					} else { // Older 12 hours (12-24h ago)
						if p.Rate > maxRate12hOld {
							maxRate12hOld = p.Rate
						}
						if p.Rate < minRate12hOld {
							minRate12hOld = p.Rate
						}
					}

					// Find rate from 24h ago
					diff := int64(math.Abs(float64(p.Date - startTs)))
					if diff < minDiff {
						minDiff = diff
						closestRate = p.Rate
					}
				}
				newData.Rate24hHigh = maxRate24h
				newData.Rate24hLow = minRate24h
				if newData.Rate24hLow > 0 {
					newData.Volatility24h = ((maxRate24h - minRate24h) / newData.Rate24hLow) * 100
				}
				if minRate12hRecent < math.MaxFloat64 && minRate12hRecent > 0 {
					newData.Volatility12h = ((maxRate12hRecent - minRate12hRecent) / minRate12hRecent) * 100
				}
				if minRate12hOld < math.MaxFloat64 && minRate12hOld > 0 {
					newData.Volatility12h_old = ((maxRate12hOld - minRate12hOld) / minRate12hOld) * 100
				}
				// Calculate 1H SMA from the most recent points
				smaPoints := 12 // ~1 hour of data (12 * 5 mins)
				if len(history.History) > 0 {
					startIndex := 0
					if len(history.History) > smaPoints {
						startIndex = len(history.History) - smaPoints
					}
					smaHistory := history.History[startIndex:]
					var smaSum float64
					for _, p := range smaHistory {
						smaSum += p.Rate
					}
					if len(smaHistory) > 0 {
						newData.Sma1h = smaSum / float64(len(smaHistory))
					}
				}
				if highTime > 0 {
					newData.Rate24hHighTime = time.UnixMilli(highTime)
				}
				if lowTime > 0 {
					newData.Rate24hLowTime = time.UnixMilli(lowTime)
				}
				newData.Rate24hAgo = closestRate
				newData.HistoricalDataFetchTime = time.Now().UTC()
			} else {
				// Historical fetch failed, use fallback.
				if historyErr != nil {
					var apiKeyErr *ApiKeyError
					// If it's NOT an API key error, flag it as a network error.
					// If it IS an API key error, we just let it fail silently and use fallbacks,
					// because the main data fetch would have already succeeded.
					if !errors.As(historyErr, &apiKeyErr) {
						newData.ApiError = "NetworkError"
						var providerDownErr *ProviderDownError
						if errors.As(historyErr, &providerDownErr) {
							newData.ApiErrorCode = providerDownErr.StatusCode
						} else {
							newData.ApiErrorCode = 0
						}
						fmt.Printf("Warning: could not fetch 24h history data due to a network error. Using fallbacks.\n")
					} else {
						// The error is an API key error, but we already have current data, so we don't need to shout about it.
						fmt.Printf("Warning: could not fetch 24h history data (API key issue?). Using fallbacks.\n")
					}
				}
				// Try to use old historical data first.
				if apiData != nil {
					copyHistoricalData(apiData, newData)
				} else {
					// No old data, use the delta fallback
					newData.Rate24hHigh = newData.Rate
					newData.Rate24hLow = newData.Rate
					newData.Volatility24h = 0
					newData.Volatility12h = 0
					newData.Volatility12h_old = 0
					newData.Sma1h = 0
					if newData.Delta.Day != 0 {
						newData.Rate24hAgo = newData.Rate / (1 + (newData.Delta.Day / 100))
					} else {
						newData.Rate24hAgo = newData.Rate
					}
				}
			}
		} else {
			// Historical data is fresh, just copy it over.
			copyHistoricalData(apiData, newData)
		}
	} else {
		// Skipping historical check, just copy old data
		copyHistoricalData(apiData, newData)
	}

	return newData
}

func testApiKey(apiKey string) bool {
	jsonData := map[string]string{"currency": "USD", "code": "BTC", "meta": "false"}
	jsonValue, _ := json.Marshal(jsonData)
	req, _ := http.NewRequest("POST", "https://api.livecoinwatch.com/coins/single", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	return err == nil && resp.StatusCode == 200
}

func getPortfolioValue(playerUSD, playerBTC float64, apiData *ApiDataResponse) float64 {
	if apiData != nil {
		return playerUSD + (playerBTC * apiData.Rate)
	}
	return playerUSD
}

func copyHistoricalData(source, dest *ApiDataResponse) {
	if source == nil || dest == nil {
		return
	}
	dest.Rate24hAgo = source.Rate24hAgo
	dest.Rate24hHigh = source.Rate24hHigh
	dest.Rate24hLow = source.Rate24hLow
	dest.Rate24hHighTime = source.Rate24hHighTime
	dest.Rate24hLowTime = source.Rate24hLowTime
	dest.Volatility24h = source.Volatility24h
	dest.Volatility12h = source.Volatility12h
	dest.Volatility12h_old = source.Volatility12h_old
	dest.Sma1h = source.Sma1h
	dest.HistoricalDataFetchTime = source.HistoricalDataFetchTime
}

func readAndParseLedger() ([]LedgerEntry, error) {
	file, err := os.Open(ledgerFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error, just no ledger yet
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, nil // No records or just header
	}

	var ledgerEntries []LedgerEntry
	for _, record := range records[1:] { // Skip header
		usd, _ := strconv.ParseFloat(strings.ReplaceAll(record[1], ",", ""), 64)
		btc, _ := strconv.ParseFloat(strings.ReplaceAll(record[2], ",", ""), 64)
		btcPrice, _ := strconv.ParseFloat(strings.ReplaceAll(record[3], ",", ""), 64)
		userBTC, _ := strconv.ParseFloat(strings.ReplaceAll(record[4], ",", ""), 64)
		dateTime, err := time.ParseInLocation("010206@150405", record[5], time.UTC)
		if err != nil {
			fmt.Printf("\nWarning: Could not parse timestamp '%s' in ledger.csv. Ignoring for calculation.\n", record[5])
		}
		ledgerEntries = append(ledgerEntries, LedgerEntry{
			TX: record[0], USD: usd, BTC: btc,
			BTCPrice: btcPrice, UserBTC: userBTC, Time: record[5], DateTime: dateTime,
		})
	}
	return ledgerEntries, nil
}

func readAllLedgerEntries() ([]LedgerEntry, error) {
	var allEntries []LedgerEntry
	processedTimestamps := make(map[string]struct{})

	// Resolve ledger directory so we find archives regardless of CWD
	ledgerAbs, _ := filepath.Abs(ledgerFilePath)
	ledgerDir := filepath.Dir(ledgerAbs)

	// 1. Read current ledger
	currentEntries, err := readAndParseLedger()
	if err != nil {
		return nil, fmt.Errorf("failed to read current ledger: %w", err)
	}
	for _, entry := range currentEntries {
		if _, exists := processedTimestamps[entry.Time]; !exists {
			processedTimestamps[entry.Time] = struct{}{}
			allEntries = append(allEntries, entry)
		}
	}

	// 2. Add merged ledger if present (historical bulk)
	mergedLedgerPath := filepath.Join(ledgerDir, "vBTC - Ledger_Merged.csv")
	if _, err := os.Stat(mergedLedgerPath); err == nil {
		mergedEntries, err := readLedgerFromFile(mergedLedgerPath)
		if err == nil {
			for _, entry := range mergedEntries {
				if _, exists := processedTimestamps[entry.Time]; !exists {
					processedTimestamps[entry.Time] = struct{}{}
					allEntries = append(allEntries, entry)
				}
			}
		}
	}

	// 3. Always add all unmerged archives (more recent; may be multiple) — dedup by Time
	// Match both legacy (MMddyy.csv) and new (MMddyy@HHmmss.csv) so multiple archives per day are included
	archivePattern := filepath.Join(ledgerDir, "vBTC - Ledger_*.csv")
	archiveFiles, err := filepath.Glob(archivePattern)
	if err == nil {
		// Exclude merged ledger from archive list
		var unmergedArchives []string
		for _, p := range archiveFiles {
			base := filepath.Base(p)
			if base != "vBTC - Ledger_Merged.csv" {
				unmergedArchives = append(unmergedArchives, p)
			}
		}
		// Sort by date then time (legacy: MMddyy; new: MMddyy@HHmmss)
		sortArchiveFilesByDateAndTime(unmergedArchives)

		for _, archivePath := range unmergedArchives {
			archiveEntries, err := readLedgerFromFile(archivePath)
			if err == nil {
				for _, entry := range archiveEntries {
					if _, exists := processedTimestamps[entry.Time]; !exists {
						processedTimestamps[entry.Time] = struct{}{}
						allEntries = append(allEntries, entry)
					}
				}
			}
		}
	}

	// 4. Sort all entries by DateTime chronologically
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].DateTime.Before(allEntries[j].DateTime)
	})

	return allEntries, nil
}

// sortArchiveFilesByDateAndTime sorts archive paths by date then time (legacy: MMddyy; new: MMddyy@HHmmss).
func sortArchiveFilesByDateAndTime(paths []string) {
	const dateLayout = "010206"
	const dateTimeLayout = "010206@150405"
	sort.Slice(paths, func(i, j int) bool {
		suffixI := extractArchiveSuffix(paths[i])
		suffixJ := extractArchiveSuffix(paths[j])
		var tI, tJ time.Time
		if strings.Contains(suffixI, "@") {
			tI, _ = time.Parse(dateTimeLayout, suffixI)
		} else {
			tI, _ = time.Parse(dateLayout, suffixI)
		}
		if strings.Contains(suffixJ, "@") {
			tJ, _ = time.Parse(dateTimeLayout, suffixJ)
		} else {
			tJ, _ = time.Parse(dateLayout, suffixJ)
		}
		return tI.Before(tJ)
	})
}

func extractArchiveSuffix(path string) string {
	base := filepath.Base(path)
	// vBTC - Ledger_012926.csv or vBTC - Ledger_012926@143022.csv
	if strings.HasPrefix(base, "vBTC - Ledger_") && strings.HasSuffix(base, ".csv") {
		return base[len("vBTC - Ledger_") : len(base)-len(".csv")]
	}
	return ""
}

func readLedgerFromFile(filePath string) ([]LedgerEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, nil // No records or just header
	}

	var ledgerEntries []LedgerEntry
	for _, record := range records[1:] { // Skip header
		usd, _ := strconv.ParseFloat(strings.ReplaceAll(record[1], ",", ""), 64)
		btc, _ := strconv.ParseFloat(strings.ReplaceAll(record[2], ",", ""), 64)
		btcPrice, _ := strconv.ParseFloat(strings.ReplaceAll(record[3], ",", ""), 64)
		userBTC, _ := strconv.ParseFloat(strings.ReplaceAll(record[4], ",", ""), 64)
		dateTime, err := time.ParseInLocation("010206@150405", record[5], time.UTC)
		if err != nil {
			fmt.Printf("\nWarning: Could not parse timestamp '%s' in %s. Ignoring for calculation.\n", record[5], filePath)
		}
		ledgerEntries = append(ledgerEntries, LedgerEntry{
			TX: record[0], USD: usd, BTC: btc,
			BTCPrice: btcPrice, UserBTC: userBTC, Time: record[5], DateTime: dateTime,
		})
	}
	return ledgerEntries, nil
}

func readAndParseLedgerRaw() ([][]string, error) {
	file, err := os.Open(ledgerFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error, just no ledger yet
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 { // Can be empty even if it exists
		return nil, nil
	}
	return records, nil
}

func writeLedgerRaw(header []string, dataRecords [][]string) error {
	file, err := os.Create(ledgerFilePath) // Create truncates the file
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		return err
	}
	if len(dataRecords) > 0 {
		if err := writer.WriteAll(dataRecords); err != nil {
			return err
		}
	}
	return nil
}

func getLedgerTotals(entries []LedgerEntry) *LedgerSummary {
	summary := &LedgerSummary{}
	summary.MinUSD = math.MaxFloat64
	summary.MaxUSD = -math.MaxFloat64
	var totalWeightedBuyPrice, totalWeightedSellPrice float64

	for _, entry := range entries {
		if !entry.DateTime.IsZero() {
			if summary.FirstTime.IsZero() || entry.DateTime.Before(summary.FirstTime) {
				summary.FirstTime = entry.DateTime
			}
			if entry.DateTime.After(summary.LastTime) {
				summary.LastTime = entry.DateTime
			}
		}
		// Tx Range = min/max Bitcoin price (USD per BTC) at time of any transaction, not total tx value
		if entry.BTCPrice > 0 {
			if entry.BTCPrice < summary.MinUSD {
				summary.MinUSD = entry.BTCPrice
			}
			if entry.BTCPrice > summary.MaxUSD {
				summary.MaxUSD = entry.BTCPrice
			}
		}
		switch entry.TX {
		case "Buy":
			summary.TotalBuyUSD += entry.USD
			summary.TotalBuyBTC += entry.BTC
			summary.BuyTransactions++
			totalWeightedBuyPrice += entry.BTCPrice * entry.BTC
		case "Sell":
			summary.TotalSellUSD += entry.USD
			summary.TotalSellBTC += entry.BTC
			summary.SellTransactions++
			totalWeightedSellPrice += entry.BTCPrice * entry.BTC
		}
	}
	if summary.MinUSD > summary.MaxUSD {
		summary.MinUSD = 0
		summary.MaxUSD = 0
	}

	// Calculate average prices (weighted by BTC amount)
	if summary.TotalBuyBTC > 0 {
		summary.AvgBuyPrice = totalWeightedBuyPrice / summary.TotalBuyBTC
	}
	if summary.TotalSellBTC > 0 {
		summary.AvgSalePrice = totalWeightedSellPrice / summary.TotalSellBTC
	}

	return summary
}

func getSessionSummary() *LedgerSummary {
	// Use all ledger data (current + archives) so session stats stay correct if user archived during session.
	allEntries, err := readAllLedgerEntries()
	if err != nil || allEntries == nil {
		return nil
	}
	// Ledger timestamps are whole seconds; truncate session start so trades in the same second are included.
	sessionStartTruncated := sessionStartTime.Truncate(time.Second)
	var sessionEntries []LedgerEntry
	for _, entry := range allEntries {
		if !entry.DateTime.IsZero() && !entry.DateTime.Before(sessionStartTruncated) {
			sessionEntries = append(sessionEntries, entry)
		}
	}
	if len(sessionEntries) == 0 {
		return nil
	}
	return getLedgerTotals(sessionEntries)
}

func invokeLedgerArchive(reader *bufio.Reader) {
	// Check if ledger exists
	if _, err := os.Stat(ledgerFilePath); os.IsNotExist(err) {
		color.Yellow("Ledger file not found. No action taken.")
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}

	// Get user input
	var linesToKeep int
	for {
		fmt.Print("Keep X Recent Lines? [0]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			linesToKeep = 0
			break
		}
		val, err := strconv.Atoi(input)
		if err == nil && val >= 0 {
			linesToKeep = val
			break
		}
		color.Red("Invalid input. Please enter a non-negative integer.")
	}

	// Archive the file (same directory as ledger); include time so multiple archives per day don't overwrite
	archiveFileName := fmt.Sprintf("vBTC - Ledger_%s.csv", time.Now().Format("010206@150405"))
	ledgerAbs, _ := filepath.Abs(ledgerFilePath)
	archivePath := filepath.Join(filepath.Dir(ledgerAbs), archiveFileName)
	sourceFile, err := os.Open(ledgerFilePath)
	if err != nil {
		color.Red("Error opening ledger file: %v", err)
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return
	}
	defer sourceFile.Close()

	destFile, err := os.Create(archivePath)
	if err != nil {
		color.Red("Error creating archive file: %v", err)
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		color.Red("Error copying to archive file: %v", err)
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return
	}
	color.Green("Ledger successfully backed up to '%s'.", archiveFileName)

	// Purge the original file
	allRecords, err := readAndParseLedgerRaw()
	if err != nil || len(allRecords) == 0 {
		color.Red("Error reading records from ledger for purging: %v", err)
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return
	}

	header := allRecords[0]
	dataRecords := allRecords[1:]

	var recordsToKeep [][]string
	if linesToKeep > 0 && linesToKeep < len(dataRecords) {
		startIndex := len(dataRecords) - linesToKeep
		recordsToKeep = dataRecords[startIndex:]
	} else if linesToKeep >= len(dataRecords) {
		recordsToKeep = dataRecords // Keep all if number is >= total records
	}

	err = writeLedgerRaw(header, recordsToKeep)
	if err != nil {
		color.Red("Error purging ledger file: %v", err)
	} else {
		color.Green("Original ledger has been purged, keeping the last %d transaction(s).", linesToKeep)
	}

	fmt.Println("Press Enter to continue.")
	reader.ReadString('\n')
}

func readCsvFileRecords(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return [][]string{}, nil // Not an error, just no file
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		if err == io.EOF {
			return [][]string{}, nil
		}
		return nil, err
	}

	if len(records) > 1 {
		return records[1:], nil // Skip header
	}
	return [][]string{}, nil // Only header or empty
}

func invokeLedgerMerge(reader *bufio.Reader) {
	clearScreen()
	color.Yellow("*** Merge Archived Ledgers ***")

	// Resolve ledger directory so we find archives regardless of CWD
	ledgerAbs, _ := filepath.Abs(ledgerFilePath)
	ledgerDir := filepath.Dir(ledgerAbs)
	archivePattern := filepath.Join(ledgerDir, "vBTC - Ledger_*.csv")

	// 1. Find archives (legacy MMddyy.csv and new MMddyy@HHmmss.csv)
	fmt.Println("\nSearching for archives...")
	archiveFiles, err := filepath.Glob(archivePattern)
	if err != nil {
		color.New(color.FgRed).Printf("Error finding archives: %v\n", err)
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}

	// Filter out the merged ledger itself
	var filteredArchives []string
	for _, file := range archiveFiles {
		if filepath.Base(file) != "vBTC - Ledger_Merged.csv" {
			filteredArchives = append(filteredArchives, file)
		}
	}

	if len(filteredArchives) == 0 {
		color.Yellow("No ledger archives found to merge.")
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}

	// 2. Sort by date then time
	sortArchiveFilesByDateAndTime(filteredArchives)

	color.Green("Found %d archive(s) to process.", len(filteredArchives))

	// 3. Load existing data for deduplication
	mergedLedgerPath := filepath.Join(ledgerDir, "vBTC - Ledger_Merged.csv")
	processedTimestamps := make(map[string]struct{})
	var existingRecords [][]string
	if _, err := os.Stat(mergedLedgerPath); err == nil {
		var readErr error
		existingRecords, readErr = readCsvFileRecords(mergedLedgerPath)
		if readErr != nil {
			color.New(color.FgRed).Printf("Could not read existing merged ledger at '%s'. It may be corrupt: %v\n", mergedLedgerPath, readErr)
			fmt.Println("\nPress Enter to continue.")
			reader.ReadString('\n')
			return
		}
		for _, record := range existingRecords {
			if len(record) > 5 { // Ensure timestamp column exists
				processedTimestamps[record[5]] = struct{}{}
			}
		}
		fmt.Printf("Existing merged ledger contains %d transactions.\n", len(existingRecords))
	} else {
		fmt.Println("No existing 'vBTC - Ledger_Merged.csv' found. A new one would be created.")
	}

	// 4. Process archives and count new unique transactions
	var newUniqueRecords [][]string
	totalScannedTxCount := 0

	fmt.Println("Analyzing archives for new transactions...")
	for _, archivePath := range filteredArchives {
		records, err := readCsvFileRecords(archivePath)
		if err != nil {
			color.Yellow("Could not read or parse '%s'. Skipping this file.", archivePath)
			continue
		}
		totalScannedTxCount += len(records)
		for _, record := range records {
			if len(record) > 5 { // Ensure timestamp column exists
				timestamp := record[5]
				if _, exists := processedTimestamps[timestamp]; !exists {
					processedTimestamps[timestamp] = struct{}{} // Add to set to prevent counting duplicates across archives
					newUniqueRecords = append(newUniqueRecords, record)
				}
			}
		}
	}

	// 5. Display summary report
	fmt.Println()
	color.Yellow("*** Merge Summary ***")
	writeAlignedLine("Archives Found:", fmt.Sprintf("%d", len(filteredArchives)), color.New(color.FgWhite))
	writeAlignedLine("Transactions in Archives:", fmt.Sprintf("%d", totalScannedTxCount), color.New(color.FgWhite))
	writeAlignedLine("Existing Merged TXs:", fmt.Sprintf("%d", len(existingRecords)), color.New(color.FgWhite))
	writeAlignedLine("New Unique TXs to Add:", fmt.Sprintf("%d", len(newUniqueRecords)), color.New(color.FgGreen))
	expectedTotal := len(existingRecords) + len(newUniqueRecords)
	writeAlignedLine("New Total TXs:", fmt.Sprintf("%d", expectedTotal), color.New(color.FgWhite))

	fmt.Println()

	if len(newUniqueRecords) == 0 {
		color.Cyan("No new transactions to merge.")
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}

	// 6. Get confirmation
	fmt.Print("Proceed with merge? This will create/update 'vBTC - Ledger_Merged.csv'. (y/n): ")
	confirm, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		color.Yellow("Merge cancelled.")
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}

	// 7. Perform safe write and verification
	fmt.Println("Merging transactions...")
	finalRecords := append(existingRecords, newUniqueRecords...)

	// Sort the final list by date to ensure chronological order
	sort.Slice(finalRecords, func(i, j int) bool {
		if len(finalRecords[i]) <= 5 || len(finalRecords[j]) <= 5 {
			return false
		}
		t1, _ := time.Parse("010206@150405", finalRecords[i][5])
		t2, _ := time.Parse("010206@150405", finalRecords[j][5])
		return t1.Before(t2)
	})

	// Use a temporary file for a safer write operation
	tempFile, err := os.CreateTemp("", "ledger-merge-*.csv")
	if err != nil {
		color.Red("Error creating temporary file: %v", err)
		fmt.Println("\nPress Enter to continue.")
		reader.ReadString('\n')
		return
	}
	defer os.Remove(tempFile.Name()) // Ensure temp file is cleaned up on exit

	writer := csv.NewWriter(tempFile)
	header := []string{"TX", "USD", "BTC", "BTC(USD)", "User BTC", "Time"}
	if err := writer.Write(header); err != nil {
		color.Red("Error writing header to temp file: %v", err)
		tempFile.Close()
		return
	}
	if err := writer.WriteAll(finalRecords); err != nil {
		color.Red("Error writing records to temp file: %v", err)
		tempFile.Close()
		return
	}
	writer.Flush()
	tempFile.Close()

	// Verify temp file before replacing the original
	verificationData, err := readCsvFileRecords(tempFile.Name())
	if err != nil || len(verificationData) != expectedTotal {
		color.Red("Verification failed. Expected %d rows, but found %d in the temporary file. Aborting.", expectedTotal, len(verificationData))
		return
	}

	// If verification passes, move the temp file to the final destination
	if err := os.Rename(tempFile.Name(), mergedLedgerPath); err != nil {
		color.Red("Error moving temporary file to final destination: %v", err)
		return
	}

	// 8. Automated Cleanup
	color.Cyan("Verification successful. Cleaning up source archives...")
	for _, archivePath := range filteredArchives {
		if err := os.Remove(archivePath); err != nil {
			color.Yellow("Warning: Could not delete archive file: %s. Please remove it manually.", archivePath)
		}
	}

	color.Green("Merge successful. '%s' has been updated.", mergedLedgerPath)
	color.Green("%d source archive(s) have been deleted.", len(filteredArchives))

	fmt.Println("\nPress Enter to continue.")
	reader.ReadString('\n')
}

func addLedgerEntry(txType string, usdAmount, btcAmount, btcPrice, userBtcAfter float64) error {
	file, err := os.OpenFile(ledgerFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Return the error to be handled by the caller, which is aware of the terminal state (raw/cooked)
		return fmt.Errorf("failed to open ledger file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	info, _ := file.Stat()
	if info.Size() == 0 {
		writer.Write([]string{"TX", "USD", "BTC", "BTC(USD)", "User BTC", "Time"})
	}

	err = writer.Write([]string{
		txType,
		fmt.Sprintf("%.2f", usdAmount),
		fmt.Sprintf("%.8f", btcAmount),
		fmt.Sprintf("%.2f", btcPrice),
		fmt.Sprintf("%.8f", userBtcAfter),
		time.Now().UTC().Format("010206@150405"),
	})
	if err != nil {
		return fmt.Errorf("failed to write record to ledger: %w", err)
	}
	return nil
}

// waitForEnter consumes from the raw input channel until an Enter key is pressed.
// It's used for "Press Enter to continue" prompts while in raw mode to avoid
// corrupting the main bufio.Reader. It also handles Ctrl+C.
func waitForEnter(inputChan chan byte, fd int, oldState *term.State) {
	for {
		b, ok := <-inputChan
		if !ok {
			return // Channel closed, exit loop.
		}
		// Enter is carriage return (13) in raw mode on Windows, or line feed (10) on Unix.
		if b == 13 || b == 10 {
			return
		}
		// Handle Ctrl+C (ASCII 3) gracefully.
		if b == 3 {
			term.Restore(fd, oldState)
			os.Exit(1)
		}
	}
}

func invokeTrade(reader *bufio.Reader, txType, amountString string) *ApiDataResponse {
	// For the most accurate UI prompt, we should read the latest config from disk here too.
	// This prevents showing the user a stale "Max" amount if another client has made a trade.
	promptCfg, err := ini.Load(iniFilePath)
	if err != nil {
		// If the read fails, fall back to the in-memory config for the prompt.
		// The critical "read-before-write" is still performed later.
		color.Yellow("Warning: could not read latest portfolio for prompt, using cached value: %v", err)
		promptCfg = cfg
	}
	playerUSD, _ := promptCfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := promptCfg.Section("Portfolio").Key("PlayerBTC").Float64()

	var maxAmount float64
	var prompt string
	if txType == "Buy" {
		maxAmount = playerUSD
		prompt = fmt.Sprintf("Amount in USD [Max $%s]:", formatFloat(maxAmount, 2))
	} else {
		maxAmount = playerBTC
		prompt = fmt.Sprintf("Amount in BTC [Max %.8f] (or use 's' for satoshis):", maxAmount)
	}

	var tradeAmount float64

	for {
		clearScreen()
		color.Yellow("*** %s Bitcoin ***", txType)

		userInput := amountString
		if userInput == "" {
			fmt.Print(prompt + " ")
			userInput, _ = reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)
			if userInput == "" {
				return apiData // Cancel
			}
		}

		parsedAmount, parseSuccess := parseTradeAmount(userInput, maxAmount, txType)
		if !parseSuccess {
			color.Red("Invalid amount or expression.")
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
			amountString = "" // Reset to re-prompt
			continue
		}

		if parsedAmount <= 0 {
			color.Red("Please enter a positive number.")
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
			amountString = ""
			continue
		}
		if parsedAmount > maxAmount {
			color.Red("Amount exceeds your balance.")
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
			amountString = ""
			continue
		}

		tradeAmount = parsedAmount
		break
	}

	// Confirmation Loop
	offerExpired := false

	// --- Raw Terminal Input Setup ---
	// Get the file descriptor for standard input.
	fd := int(os.Stdin.Fd())

	// Check if we are in a terminal, which is required for raw mode.
	if !term.IsTerminal(fd) {
		color.Red("Error: Standard input is not a terminal. Cannot enter raw mode.")
		color.Red("Trade cancelled.")
		time.Sleep(2 * time.Second)
		return apiData
	}

	// Save the original terminal state so we can restore it later.
	oldState, err := term.GetState(fd)
	if err != nil {
		color.Red("Error: Could not get terminal state: %v", err)
		time.Sleep(2 * time.Second)
		return apiData
	}

	done := make(chan struct{}) // Channel to signal the goroutine to stop.
	var wg sync.WaitGroup

	// CRITICAL: Ensure the goroutine is stopped, terminal is restored,
	// and the input buffer is reset when the function exits.
	defer func() {
		close(done) // 1. Signal the input goroutine to stop.
		wg.Wait()   // 2. Wait for the input goroutine to finish before proceeding.
		term.Restore(fd, oldState)
		reader.Reset(os.Stdin)
	}()

	// Put the terminal into raw mode using the original descriptor.
	_, err = term.MakeRaw(fd)
	if err != nil {
		color.Red("Error: Could not set terminal to raw mode: %v", err)
		time.Sleep(2 * time.Second)
		return apiData
	}

	// Create a channel to receive input from a non-blocking goroutine.
	inputChan := make(chan byte)
	wg.Add(1) // Increment the WaitGroup counter before starting the goroutine.
	go func() {
		defer wg.Done()
		defer close(inputChan)
		for {
			// cancellableRead is a platform-aware, non-blocking read.
			b, err := cancellableRead(done)
			if err != nil {
				// This error is expected when 'done' is closed, so we just exit.
				return
			}
			// We got a character, try to send it but don't block.
			select {
			case inputChan <- b:
			case <-done:
				return
			}
		}
	}()

	for {
		apiData = updateApiData(true)
		if apiData != nil && apiData.ApiError == "NetworkError" {
			errorMessage := "\nAPI Provider Problem"
			if apiData.ApiErrorCode > 0 {
				errorMessage = fmt.Sprintf("%s (%d)", errorMessage, apiData.ApiErrorCode)
			}
			errorMessage += " - Try again later"
			color.Red("\n%s", errorMessage)
			fmt.Println("\nPress Enter to return to the main menu.")
			waitForEnter(inputChan, fd, oldState)
			return apiData // Return to main menu
		}
		if apiData == nil || apiData.Rate == 0 {
			color.Red("\nError fetching price. Press Enter to continue.")
			waitForEnter(inputChan, fd, oldState)
			return apiData
		}
		offerTimestamp := time.Now() // Record the time the offer is presented.

		clearScreen()
		color.Yellow("*** %s Bitcoin ***", txType)
		if offerExpired {
			color.Yellow("\nOffer expired. A new price has been fetched.")
			offerExpired = false // Reset the flag after showing the message
		}

		var usdAmount, btcAmount float64
		if txType == "Buy" {
			usdAmount = tradeAmount
			btcAmount = math.Floor((usdAmount/apiData.Rate)*1e8) / 1e8
		} else { // Sell
			btcAmount = tradeAmount
			usdAmount = math.Floor((btcAmount*apiData.Rate)*100) / 100
		}

		priceColor := color.New(color.FgWhite)
		if apiData.Rate > apiData.Rate24hAgo {
			priceColor = color.New(color.FgGreen)
		} else if apiData.Rate < apiData.Rate24hAgo {
			priceColor = color.New(color.FgRed)
		}

		fmt.Println("\nYou have 2 minutes to accept this offer.")
		priceColor.Printf("Market Rate: $%s\n", formatFloat(apiData.Rate, 2))

		var confirmPrompt string
		if txType == "Buy" {
			confirmPrompt = fmt.Sprintf("Purchase %.8f BTC for $%s? ", btcAmount, formatFloat(usdAmount, 2))
		} else {
			confirmPrompt = fmt.Sprintf("Sell %.8f BTC for $%s? ", btcAmount, formatFloat(usdAmount, 2))
		}

		fmt.Print(confirmPrompt)
		color.New(color.FgWhite).Print("[")
		color.New(color.FgGreen).Print("y")
		color.New(color.FgWhite).Print("/")
		color.New(color.FgCyan).Print("r")
		color.New(color.FgWhite).Print("/")
		color.New(color.FgRed).Print("n")
		color.New(color.FgWhite).Println("]")

		// Create a new ticker for each offer to handle the countdown.
		ticker := time.NewTicker(250 * time.Millisecond)

		displayState := "" // e.g., "Initial", "OneMinute", "ThirtySeconds", "Expired"
		redrawTradeScreen(txType, offerExpired, apiData, tradeAmount, displayState)

	EventLoop:
		for {
			select {
			case <-ticker.C:
				secondsRemaining := (2 * time.Minute) - time.Since(offerTimestamp)
				requiredState := "Initial"
				if secondsRemaining <= 0 {
					requiredState = "Expired"
				} else if secondsRemaining <= 30*time.Second {
					requiredState = "ThirtySeconds"
				} else if secondsRemaining <= 1*time.Minute {
					requiredState = "OneMinute"
				}

				if requiredState != displayState {
					displayState = requiredState
					redrawTradeScreen(txType, offerExpired, apiData, tradeAmount, displayState)
				}
			case b, ok := <-inputChan:
				if !ok {
					// Input channel was closed, likely due to an error.
					color.Red("\nInput listener closed unexpectedly. Cancelling trade.")
					time.Sleep(1 * time.Second)
					return apiData
				}

				// Handle Ctrl+C gracefully in raw mode.
				if b == 3 {
					term.Restore(fd, oldState) // Restore terminal state BEFORE exiting.
					os.Exit(1)
				}

				// Handle Esc key (ASCII 27) - could be Esc or start of arrow key sequence
				if b == 27 {
					// Check if this is an arrow key sequence (ESC [ A/B/C/D)
					// Try to read the next bytes quickly to detect arrow keys
					arrowDetected := false

					// Use a very short timeout to check for arrow key sequence
					select {
					case nextByte, ok := <-inputChan:
						if ok && nextByte == '[' {
							// This looks like an arrow key sequence, read the direction
							select {
							case arrowByte, ok := <-inputChan:
								if ok {
									switch arrowByte {
									case 'A': // Up arrow = Y (accept)
										arrowDetected = true
										b = 'y'
									case 'B': // Down arrow = N (cancel)
										arrowDetected = true
										b = 'n'
									case 'C': // Right arrow = R (refresh)
										arrowDetected = true
										b = 'r'
									case 'D': // Left arrow = Esc (cancel)
										arrowDetected = true
										b = 'n'
									}
								}
							case <-time.After(10 * time.Millisecond):
								// Timeout - not an arrow key, treat as plain Esc
							}
						}
					case <-time.After(10 * time.Millisecond):
						// Timeout - treat as plain Esc
					}

					if !arrowDetected {
						// Plain Esc key pressed - treat as cancel
						fmt.Printf("\n\n%s cancelled.\n", txType)
						time.Sleep(1 * time.Second)
						ticker.Stop()
						return apiData
					}
				}

				var rawInput string
				if b == 13 { // 13 is the ASCII code for Carriage Return (Enter key in raw mode)
					rawInput = "\n"
				} else {
					rawInput = string(b)
				}

				input := strings.ToLower(strings.TrimSpace(rawInput))

				if displayState == "Expired" {
					if input == "" { // Enter on expired screen returns to main menu.
						ticker.Stop()
						return apiData
					}
					if input != "r" {
						// On expired screen, ignore any key other than 'r', Enter, or Esc (already handled above).
						continue EventLoop
					}
					// If input is 'r', it will fall through to the handler below.
				}

				if input == "y" {
					// Check if the offer has expired *at the moment of acceptance*.
					if time.Since(offerTimestamp).Minutes() >= 2 {
						offerExpired = true
						ticker.Stop()
						break EventLoop // The offer is stale, break inner loop to get a new price.
					}

					// Reload config from disk to get the absolute latest portfolio state before committing the trade.
					tradeCfg, err := ini.Load(iniFilePath)
					if err != nil {
						color.Red("\nCritical Error: Could not read portfolio file '%s' to finalize trade.", iniFilePath)
						color.Red("Error: %v", err)
						color.Red("Your trade has been CANCELLED to prevent data loss.")
						fmt.Println("\nPress Enter to continue.")
						ticker.Stop()
						waitForEnter(inputChan, fd, oldState)
						return apiData // Cancel the trade
					}

					// Get the most up-to-date portfolio values
					currentPlayerUSD, _ := tradeCfg.Section("Portfolio").Key("PlayerUSD").Float64()
					currentPlayerBTC, _ := tradeCfg.Section("Portfolio").Key("PlayerBTC").Float64()
					currentPlayerInvested, _ := tradeCfg.Section("Portfolio").Key("PlayerInvested").Float64()

					// Verify if the trade is still possible with the latest balance
					if txType == "Buy" && usdAmount > currentPlayerUSD {
						color.Red("\nTrade cancelled. Your USD balance has changed since the trade was initiated.")
						color.Red("Your current balance is $%s, but the trade required $%s.", formatFloat(currentPlayerUSD, 2), formatFloat(usdAmount, 2))
						fmt.Println("\nPress Enter to continue.")
						ticker.Stop()
						waitForEnter(inputChan, fd, oldState)
						return apiData
					}
					if txType == "Sell" && btcAmount > currentPlayerBTC {
						color.Red("\nTrade cancelled. Your BTC balance has changed since the trade was initiated.")
						color.Red("Your current balance is %.8f BTC, but the trade required %.8f BTC.", currentPlayerBTC, btcAmount)
						fmt.Println("\nPress Enter to continue.")
						ticker.Stop()
						waitForEnter(inputChan, fd, oldState)
						return apiData
					}

					var newUserBtc, newInvested float64
					if txType == "Buy" {
						tradeCfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", currentPlayerUSD-usdAmount))
						newUserBtc = currentPlayerBTC + btcAmount
						newInvested = currentPlayerInvested + usdAmount
					} else { // Sell
						newUserBtc = currentPlayerBTC - btcAmount
						if newUserBtc < 1e-9 { // Tolerance for float comparison
							newUserBtc = 0
							newInvested = 0
						} else if currentPlayerBTC > 0 {
							newInvested = currentPlayerInvested * (newUserBtc / currentPlayerBTC)
						}
						tradeCfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", currentPlayerUSD+usdAmount))
					}
					tradeCfg.Section("Portfolio").Key("PlayerBTC").SetValue(fmt.Sprintf("%.8f", newUserBtc))
					tradeCfg.Section("Portfolio").Key("PlayerInvested").SetValue(fmt.Sprintf("%.2f", newInvested))
					err = tradeCfg.SaveTo(iniFilePath)
					if err != nil {
						color.Red("\nTrade failed: Could not save portfolio update to vbtc.ini.")
						color.Red("Error: %v", err)
						fmt.Println("\nPlease check file permissions and try again.")
						fmt.Println("\nPress Enter to continue.")
						ticker.Stop()
						waitForEnter(inputChan, fd, oldState)
					} else {
						cfg = tradeCfg // Update the global config to reflect the new state
						err := addLedgerEntry(txType, usdAmount, btcAmount, apiData.Rate, newUserBtc)
						if err != nil {
							color.Red("\nTransaction complete, but failed to write to ledger.csv.")
							color.Red("Error: %v", err)
							fmt.Println("\nPlease ensure the file is not open in another program.")
							fmt.Println("\nPress Enter to acknowledge.")
							waitForEnter(inputChan, fd, oldState)
						}
						// Print success message without a newline, sleep, then overwrite with a processing message.
						fmt.Printf("\n\n%s successful.", txType)
						time.Sleep(1 * time.Second)
						ticker.Stop()
						fmt.Printf("\rReturning to main menu...\n")
					}
					return apiData // Exit trade loop regardless of success or failure
				} else if input == "r" {
					// User is manually refreshing, so the offer isn't "expired" in a way that requires a warning.
					offerExpired = false
					ticker.Stop()
					break EventLoop // Break inner loop to get a new price.
				} else if input == "n" || input == "" { // Explicitly cancel on 'n' or Enter
					// Print cancel message with a newline.
					fmt.Printf("\n\n%s cancelled.\n", txType)
					time.Sleep(1 * time.Second)
					ticker.Stop()
					return apiData
				} else {
					// Ignore all other keys that aren't 'y', 'r', 'n', Enter, or Esc
					continue EventLoop
				}
			}
		}
	}
}

func redrawTradeScreen(txType string, offerExpired bool, apiData *ApiDataResponse, tradeAmount float64, displayState string) {
	clearScreen()
	color.Yellow("*** %s Bitcoin ***", txType)
	if offerExpired {
		color.Yellow("\nOffer expired. A new price has been fetched.")
		// The offerExpired flag is now managed by the caller loop, so no need to reset it here.
	}

	// Determine message and color based on the new state
	var timeLeftMessage string
	var timeLeftColor *color.Color
	switch displayState {
	case "OneMinute":
		timeLeftMessage = "You have 1 minute to accept this offer."
		timeLeftColor = color.New(color.FgYellow)
	case "ThirtySeconds":
		timeLeftMessage = "You have 30 seconds to accept this offer."
		timeLeftColor = color.New(color.FgRed)
	case "Expired":
		timeLeftMessage = "Offer expired, please refresh for new offer."
		timeLeftColor = color.New(color.FgCyan)
	default: // "Initial" or fallback
		timeLeftMessage = "You have 2 minutes to accept this offer."
		timeLeftColor = color.New(color.FgWhite)
	}

	var usdAmount, btcAmount float64
	if txType == "Buy" {
		usdAmount = tradeAmount
		btcAmount = math.Floor((usdAmount/apiData.Rate)*1e8) / 1e8
	} else { // Sell
		btcAmount = tradeAmount
		usdAmount = math.Floor((btcAmount*apiData.Rate)*100) / 100
	}

	priceColor := color.New(color.FgWhite)
	if apiData.Rate > apiData.Rate24hAgo {
		priceColor = color.New(color.FgGreen)
	} else if apiData.Rate < apiData.Rate24hAgo {
		priceColor = color.New(color.FgRed)
	}

	fmt.Println()
	timeLeftColor.Println(timeLeftMessage)
	priceColor.Printf("Market Rate: $%s\n", formatFloat(apiData.Rate, 2))

	var confirmPrompt string
	if txType == "Buy" {
		confirmPrompt = fmt.Sprintf("Purchase %.8f BTC for $%s? ", btcAmount, formatFloat(usdAmount, 2))
	} else {
		confirmPrompt = fmt.Sprintf("Sell %.8f BTC for $%s? ", btcAmount, formatFloat(usdAmount, 2))
	}

	fmt.Print(confirmPrompt)
	if displayState == "Expired" {
		color.New(color.FgWhite).Print("[")
		color.New(color.FgCyan).Print("r")
		color.New(color.FgWhite).Println("]")
	} else {
		color.New(color.FgWhite).Print("[")
		color.New(color.FgGreen).Print("y")
		color.New(color.FgWhite).Print("/")
		color.New(color.FgCyan).Print("r")
		color.New(color.FgWhite).Print("/")
		color.New(color.FgRed).Print("n")
		color.New(color.FgWhite).Println("]")
	}
}

func parseTradeAmount(input string, maxAmount float64, txType string) (float64, bool) {
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, ",", "") // Allow commas

	// Percentage
	if strings.HasSuffix(input, "p") {
		percentString := strings.TrimSuffix(input, "p")
		expression, err := govaluate.NewEvaluableExpression(percentString)
		if err != nil {
			return 0, false // Invalid expression
		}
		result, err := expression.Evaluate(nil)
		if err != nil {
			return 0, false // Evaluation failed
		}
		percentVal, ok := result.(float64)
		if !ok {
			return 0, false // Result is not a float
		}

		if percentVal <= 0 || percentVal > 100 {
			return 0, false
		}
		calculatedAmount := (maxAmount * percentVal) / 100
		if txType == "Sell" {
			return math.Floor(calculatedAmount*1e8) / 1e8, true // Truncate for BTC
		}
		return math.Floor(calculatedAmount*100) / 100, true // Truncate for USD
	}

	// Satoshis
	if strings.HasSuffix(input, "s") {
		if txType == "Buy" {
			return 0, false
		}
		satoshiString := strings.TrimSuffix(input, "s")
		satoshiVal, err := strconv.ParseFloat(satoshiString, 64)
		if err != nil {
			return 0, false
		}
		return satoshiVal / 1e8, true
	}

	// Plain number
	amount, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, false
	}
	return amount, true
}

// --- Utility Functions ---

func getParentProcessName() (string, error) {
	ppid := os.Getppid()
	// On Windows, double-clicking from explorer.exe might make the parent process disappear
	// by the time we check. A PPID of 1 is not meaningful on Windows as it is on Linux.
	if ppid <= 1 {
		return "explorer", nil // Assume it was explorer if the parent is gone or system
	}
	parentProcess, err := process.NewProcess(int32(ppid))
	if err != nil {
		return "", err
	}
	return parentProcess.Name()
}

func formatFloat(num float64, decimals int) string {
	// Use a robust method to format numbers with commas.
	// 1. Format to a string with the specified number of decimals.
	str := fmt.Sprintf("%."+strconv.Itoa(decimals)+"f", math.Abs(num))
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = "." + parts[1]
	}

	// 2. Add commas to the integer part.
	n := len(integerPart)
	if n <= 3 {
		if num < 0 {
			return "-" + integerPart + decimalPart
		}
		return integerPart + decimalPart
	}

	// Calculate the number of commas needed.
	commas := (n - 1) / 3
	// Create a new byte slice to hold the result.
	result := make([]byte, n+commas)
	// Iterate from right to left, inserting commas.
	for i, j, k := n-1, len(result)-1, 0; i >= 0; i, j, k = i-1, j-1, k+1 {
		if k > 0 && k%3 == 0 {
			result[j] = ','
			j--
		}
		result[j] = integerPart[i]
	}

	if num < 0 {
		return "-" + string(result) + decimalPart
	}
	return string(result) + decimalPart
}

func formatProfitLoss(value float64, formatSuffix string) string {
	if value < 0 {
		return fmt.Sprintf("(%.2f%s)", math.Abs(value), formatSuffix)
	}
	return fmt.Sprintf("+%.2f%s", value, formatSuffix)
}

func formatDuration(first, last time.Time) string {
	if first.IsZero() || last.IsZero() || last.Before(first) {
		return ""
	}
	d := last.Sub(first)
	if d < 60*time.Minute {
		return fmt.Sprintf("%dM", int(d.Round(time.Minute).Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dH", int(d.Round(time.Hour).Hours()))
	}
	return fmt.Sprintf("%dD", int(d.Round(time.Hour).Hours()/24))
}

// formatCadence formats a cadence duration for display: M+S under 1h, H+M under 48h, D above.
func formatCadence(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Hour {
		m := int(d.Truncate(time.Minute).Minutes())
		s := int((d - time.Duration(m)*time.Minute).Truncate(time.Second).Seconds())
		return fmt.Sprintf("%dM%dS", m, s)
	}
	if d < 48*time.Hour {
		h := int(d.Truncate(time.Hour).Hours())
		m := int((d - time.Duration(h)*time.Hour).Truncate(time.Minute).Minutes())
		return fmt.Sprintf("%dH%dM", h, m)
	}
	days := int(d.Truncate(24*time.Hour).Hours() / 24)
	return fmt.Sprintf("%dD", days)
}

func writeAlignedLineCadence(label string, ledgerCadence, sessionCadence string, ledgerIsSlower bool, startColumn int) {
	valueStartColumn := 22
	if startColumn > 0 {
		valueStartColumn = startColumn
	}
	padding := valueStartColumn - len(label)
	if padding < 0 {
		padding = 0
	}
	fmt.Print(label)
	fmt.Print(strings.Repeat(" ", padding))
	if sessionCadence == "" {
		color.New(color.FgWhite).Println(ledgerCadence)
		return
	}
	if ledgerIsSlower {
		color.New(color.FgRed).Print(ledgerCadence)
		color.New(color.FgWhite).Print(" [")
		color.New(color.FgGreen).Print(sessionCadence)
		color.New(color.FgWhite).Println("]")
	} else {
		color.New(color.FgGreen).Print(ledgerCadence)
		color.New(color.FgWhite).Print(" [")
		color.New(color.FgRed).Print(sessionCadence)
		color.New(color.FgWhite).Println("]")
	}
}
