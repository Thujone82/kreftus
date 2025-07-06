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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/Knetic/govaluate"
	"github.com/shirou/gopsutil/v3/process"
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
	FetchTime  time.Time
	Rate24hAgo float64
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
	TotalBuyUSD  float64
	TotalSellUSD float64
	TotalBuyBTC  float64
	TotalSellBTC float64
}

// --- Main Application ---
func main() {
	setup()
	mainLoop()
}

func setup() {
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
		showFirstRunSetup()
	}

	// Perform the initial data fetch to get a complete data object.
	apiData = updateApiData()

	// Only after the data is fully fetched, set the session-start values.
	// This prevents the race condition and ensures the initial price is correct.
	if apiData != nil {
		initialSessionBtcPrice = apiData.Rate
	}
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	sessionStartPortfolioValue = getPortfolioValue(playerUSD, playerBTC, apiData)
}

func mainLoop() {
	commands := map[string]string{
		"b": "buy", "buy": "buy",
		"s": "sell", "sell": "sell",
		"l": "ledger", "ledger": "ledger",
		"r": "refresh", "refresh": "refresh",
		"c": "config", "config": "config",
		"h": "help", "help": "help",
		"e": "exit", "exit": "exit",
	}

	reader := bufio.NewReader(os.Stdin)

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
				invokeTrade("Buy", amount)
				apiData = updateApiData()
			case "sell":
				invokeTrade("Sell", amount)
				apiData = updateApiData()
			case "ledger":
				showLedgerScreen()
			case "refresh":
				apiData = updateApiData()
			case "config":
				showConfigScreen()
			case "help":
				showHelpScreen()
			case "exit":
				showExitScreen()
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

	// Market Data
	color.New(color.FgYellow).Println("*** Bitcoin Market ***")
	if apiData == nil {
		color.Red("Could not retrieve market data. Please check your API key in the Config menu.")
	} else {
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
		writeAlignedLine("24H Ago:", fmt.Sprintf("$%s [%+.2f%%]", formatFloat(apiData.Rate24hAgo, 2), percentChange), priceColor24h)
		writeAlignedLine("24H Volume:", fmt.Sprintf("$%s", formatFloat(apiData.Volume, 0)), color.New(color.FgWhite))
		writeAlignedLine("Time:", apiData.FetchTime.Local().Format("010206@150405"), color.New(color.FgCyan))
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
		writeAlignedLine("Session P/L:", fmt.Sprintf("%+.2f [%+.2f%%]", sessionChange, sessionPercent), sessionColor)
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

func showFirstRunSetup() {
	clearScreen()
	color.Yellow("*** First Time Setup ***")
	reader := bufio.NewReader(os.Stdin)
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

func showConfigScreen() {
	reader := bufio.NewReader(os.Stdin)
	for {
		clearScreen()
		color.Yellow("*** Configuration ***")
		fmt.Println("1. Update API Key")
		fmt.Println("2. Reset Portfolio")
		fmt.Println("3. Archive Ledger")
		fmt.Println("4. Return to Main Screen")
		fmt.Print("Enter your choice: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
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
		case "2":
			fmt.Print("Are you sure you want to reset your portfolio? This cannot be undone. Type 'YES' to confirm: ")
			confirm, _ := reader.ReadString('\n')
			if strings.TrimSpace(confirm) == "YES" {
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
		case "3":
			invokeLedgerArchive()
		case "4":
			return
		default:
			color.Red("Invalid choice. Please try again.")
			fmt.Println("Press Enter to continue.")
			reader.ReadString('\n')
		}
	}
}

func showHelpScreen() {
	clearScreen()
	color.Yellow("*** Help ***")
	writeAlignedLine("buy [amount]", "Purchase a specific USD amount of Bitcoin.", color.New(color.FgWhite))
	writeAlignedLine("sell [amount]", "Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s).", color.New(color.FgWhite))
	writeAlignedLine("ledger", "View a history of all your transactions.", color.New(color.FgWhite))
	writeAlignedLine("refresh", "Manually update the market data.", color.New(color.FgWhite))
	writeAlignedLine("config", "Access the configuration menu.", color.New(color.FgWhite))
	writeAlignedLine("help", "Show this help screen.", color.New(color.FgWhite))
	writeAlignedLine("exit", "Exit the application.", color.New(color.FgWhite))
	fmt.Println()
	color.New(color.FgCyan).Println("Tip: Use 'p' for percentage trades (e.g., '50p' for 50% of your balance).")
	fmt.Println()
	fmt.Println("Press Enter to return to the Main Screen.")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func showLedgerScreen() {
	clearScreen()
	color.Yellow("*** Ledger ***")

	ledgerEntries, err := readAndParseLedger()
	if err != nil {
		color.Red("Error reading ledger file: %v", err)
		fmt.Println("\nPress Enter to return to Main screen")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	if len(ledgerEntries) == 0 {
		fmt.Println("You have not made any transactions yet.")
		fmt.Println("\nPress Enter to return to Main screen")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

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

	summary := getLedgerTotals(ledgerEntries)
	if summary != nil {
		fmt.Println()
		color.Yellow("*** Ledger Summary ***")
		summaryValueStartColumn := 22 // Align with portfolio summary
		if summary.TotalBuyUSD > 0 {
			writeAlignedLine("Total Bought (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalBuyUSD, 2)), color.New(color.FgGreen), summaryValueStartColumn)
			writeAlignedLine("Total Bought (BTC):", fmt.Sprintf("%.8f", summary.TotalBuyBTC), color.New(color.FgGreen), summaryValueStartColumn)
		}
		if summary.TotalSellUSD > 0 {
			writeAlignedLine("Total Sold (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalSellUSD, 2)), color.New(color.FgRed), summaryValueStartColumn)
			writeAlignedLine("Total Sold (BTC):", fmt.Sprintf("%.8f", summary.TotalSellBTC), color.New(color.FgRed), summaryValueStartColumn)
		}
	}

	fmt.Println("\nPress Enter to return to Main screen")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func showExitScreen() {
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
	writeAlignedLine("Total Profit/Loss:", fmt.Sprintf("%s%s", plusSign(profit), formatFloat(profit, 2)), profitColor)

	// --- Session Summary ---
	fmt.Println()
	color.Yellow("*** Session Summary ***")
	sessionValueStartColumn := 22 // Use a consistent start column for this block

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
		writeAlignedLine("P/L:", fmt.Sprintf("%s%.2f [%+.2f%%]", plusSign(sessionChange), sessionChange, sessionPercent), sessionColor, sessionValueStartColumn)
	}

	summary := getSessionSummary()
	if summary != nil {
		if summary.TotalBuyUSD > 0 {
			writeAlignedLine("Total Bought (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalBuyUSD, 2)), color.New(color.FgGreen), sessionValueStartColumn)
			writeAlignedLine("Total Bought (BTC):", fmt.Sprintf("%.8f", summary.TotalBuyBTC), color.New(color.FgGreen), sessionValueStartColumn)
		}
		if summary.TotalSellUSD > 0 {
			writeAlignedLine("Total Sold (USD):", fmt.Sprintf("$%s", formatFloat(summary.TotalSellUSD, 2)), color.New(color.FgRed), sessionValueStartColumn)
			writeAlignedLine("Total Sold (BTC):", fmt.Sprintf("%.8f", summary.TotalSellBTC), color.New(color.FgRed), sessionValueStartColumn)
		}
	}

	// Pause the screen if the application was likely run by double-clicking.
	// We do this by checking if the parent process is a known interactive shell.
	parentName, err := getParentProcessName()
	shouldPause := true // Default to pausing to be safe.
	if err == nil {
		parentName = strings.ToLower(strings.TrimSuffix(parentName, ".exe"))
		// List of shells where we should NOT pause.
		interactiveShells := []string{"cmd", "powershell", "pwsh", "wt", "alacritty", "explorer"}
		for _, shell := range interactiveShells {
			// The check for "explorer" is a special case for running from VS Code's integrated terminal,
			// which can sometimes have explorer.exe as a parent.
			if parentName == shell && shell != "explorer" {
				shouldPause = false
				break
			}
		}
	} else {
		fmt.Printf("\nCould not determine parent process: %v. Pausing by default.", err)
	}

	if shouldPause {
		fmt.Println("\nPress Enter to exit.")
		bufio.NewReader(os.Stdin).ReadString('\n')
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API for current price returned status code %d", resp.StatusCode)
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

func getHistoricalData(apiKey string) (*HistoryResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is empty")
	}
	end := time.Now().UTC().Add(-24 * time.Hour).UnixMilli()
	start := end - (5 * 60 * 1000) // 5 minute window

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
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API for historical price returned status code %d", resp.StatusCode)
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

func updateApiData() *ApiDataResponse {
	showLoadingScreen()
	apiKey := cfg.Section("Settings").Key("ApiKey").String()

	data, err := fetchCurrentPriceData(apiKey)
	if err != nil {
		fmt.Printf("Error fetching current price data: %v\n", err)
		return nil
	}

	// Get historical data for 24h ago
	history, err := getHistoricalData(apiKey)
	if err == nil && history != nil && len(history.History) > 0 {
		targetTs := time.Now().UTC().Add(-24 * time.Hour).UnixMilli()
		closest := history.History[0]
		minDiff := int64(math.Abs(float64(closest.Date - targetTs)))
		for _, p := range history.History {
			diff := int64(math.Abs(float64(p.Date - targetTs)))
			if diff < minDiff {
				minDiff = diff
				closest = p
			}
		}
		data.Rate24hAgo = closest.Rate
	} else {
		if err != nil {
			fmt.Printf("Warning: could not fetch historical data: %v. Using fallback.\n", err)
		}
		// Fallback to using the delta from the current price data if historical fails
		if data.Delta.Day != 0 {
			data.Rate24hAgo = data.Rate / (1 + (data.Delta.Day / 100))
		} else {
			data.Rate24hAgo = data.Rate // If no historical or delta, assume no change
		}
	}

	return data
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
	for _, entry := range entries {
		switch entry.TX {
		case "Buy":
			summary.TotalBuyUSD += entry.USD
			summary.TotalBuyBTC += entry.BTC
		case "Sell":
			summary.TotalSellUSD += entry.USD
			summary.TotalSellBTC += entry.BTC
		}
	}
	return summary
}

func getSessionSummary() *LedgerSummary {
	allEntries, err := readAndParseLedger()
	if err != nil || allEntries == nil {
		return nil
	}
	var sessionEntries []LedgerEntry
	for _, entry := range allEntries {
		if !entry.DateTime.IsZero() && !entry.DateTime.Before(sessionStartTime) {
			sessionEntries = append(sessionEntries, entry)
		}
	}
	if len(sessionEntries) == 0 {
		return nil
	}
	return getLedgerTotals(sessionEntries)
}

func invokeLedgerArchive() {
	// Check if ledger exists
	if _, err := os.Stat(ledgerFilePath); os.IsNotExist(err) {
		color.Yellow("Ledger file not found. No action taken.")
		fmt.Println("\nPress Enter to continue.")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	// Get user input
	reader := bufio.NewReader(os.Stdin)
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

	// Archive the file
	archiveFileName := fmt.Sprintf("vBTC - Ledger_%s.csv", time.Now().Format("010206"))
	sourceFile, err := os.Open(ledgerFilePath)
	if err != nil {
		color.Red("Error opening ledger file: %v", err)
		fmt.Println("Press Enter to continue.")
		reader.ReadString('\n')
		return
	}
	defer sourceFile.Close()

	destFile, err := os.Create(archiveFileName)
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

func addLedgerEntry(txType string, usdAmount, btcAmount, btcPrice, userBtcAfter float64) {
	file, err := os.OpenFile(ledgerFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		color.Red("Transaction complete, but failed to write to ledger.csv. Please ensure the file is not open in another program.")
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if file is new
	info, _ := file.Stat()
	if info.Size() == 0 {
		writer.Write([]string{"TX", "USD", "BTC", "BTC(USD)", "User BTC", "Time"})
	}

	record := []string{
		txType,
		fmt.Sprintf("%.2f", usdAmount),
		fmt.Sprintf("%.8f", btcAmount),
		fmt.Sprintf("%.2f", btcPrice),
		fmt.Sprintf("%.8f", userBtcAfter),
		time.Now().UTC().Format("010206@150405"),
	}
	writer.Write(record)
}

func invokeTrade(txType, amountString string) {
	playerUSD, _ := cfg.Section("Portfolio").Key("PlayerUSD").Float64()
	playerBTC, _ := cfg.Section("Portfolio").Key("PlayerBTC").Float64()
	playerInvested, _ := cfg.Section("Portfolio").Key("PlayerInvested").Float64()

	var maxAmount float64
	var prompt string
	if txType == "Buy" {
		maxAmount = playerUSD
		prompt = fmt.Sprintf("Amount in USD: [Max $%s]", formatFloat(maxAmount, 2))
	} else {
		maxAmount = playerBTC
		prompt = fmt.Sprintf("Amount in BTC: [Max %.8f] (or use 's' for satoshis)", maxAmount)
	}

	reader := bufio.NewReader(os.Stdin)
	var tradeAmount float64

	for {
		clearScreen()
		color.Yellow("*** %s Bitcoin ***", txType)

		userInput := amountString
		if userInput == "" {
			fmt.Print(prompt + ": ")
			userInput, _ = reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)
			if userInput == "" {
				return // Cancel
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
	for {
		currentApiData := updateApiData()
		if currentApiData == nil {
			color.Red("Error fetching price. Press Enter to continue.")
			reader.ReadString('\n')
			return
		}
		currentBTCPrice := currentApiData.Rate
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
			btcAmount = math.Floor((usdAmount/currentBTCPrice)*1e8) / 1e8
		} else { // Sell
			btcAmount = tradeAmount
			usdAmount = math.Floor((btcAmount*currentBTCPrice)*100) / 100
		}

		priceColor := color.New(color.FgWhite)
		if currentBTCPrice > currentApiData.Rate24hAgo {
			priceColor = color.New(color.FgGreen)
		} else if currentBTCPrice < currentApiData.Rate24hAgo {
			priceColor = color.New(color.FgRed)
		}

		fmt.Println("\nYou have 2 minutes to accept this offer.")
		priceColor.Printf("Market Rate: $%s\n", formatFloat(currentBTCPrice, 2))

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

		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "y" {
			// Check if the offer has expired *at the moment of acceptance*.
			if time.Since(offerTimestamp).Minutes() >= 2 {
				offerExpired = true
				continue // The offer is stale, loop to get a new price.
			}

			var newUserBtc, newInvested float64
			if txType == "Buy" {
				cfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", playerUSD-usdAmount))
				newUserBtc = playerBTC + btcAmount
				newInvested = playerInvested + usdAmount
			} else { // Sell
				newUserBtc = playerBTC - btcAmount
				if newUserBtc < 1e-9 { // Tolerance for float comparison
					newUserBtc = 0
					newInvested = 0
				} else if playerBTC > 0 {
					newInvested = playerInvested * (newUserBtc / playerBTC)
				}
				cfg.Section("Portfolio").Key("PlayerUSD").SetValue(fmt.Sprintf("%.2f", playerUSD+usdAmount))
			}
			cfg.Section("Portfolio").Key("PlayerBTC").SetValue(fmt.Sprintf("%.8f", newUserBtc))
			cfg.Section("Portfolio").Key("PlayerInvested").SetValue(fmt.Sprintf("%.2f", newInvested))
			cfg.SaveTo(iniFilePath)

			addLedgerEntry(txType, usdAmount, btcAmount, currentBTCPrice, newUserBtc)
			fmt.Printf("\n%s successful.\n", txType)
			time.Sleep(1 * time.Second)
			return
		} else if input == "r" {
			// User is manually refreshing, so the offer isn't "expired" in a way that requires a warning.
			offerExpired = false
			continue // Reload the loop
		} else {
			fmt.Printf("\n%s cancelled.\n", txType)
			time.Sleep(1 * time.Second)
			return
		}
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

func plusSign(num float64) string {
	if num > 0 {
		return "+"
	}
	return ""
}
