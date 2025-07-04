<#
.SYNOPSIS
    An interactive PowerShell-based Bitcoin trading application. Users can buy and sell Bitcoin using a simulated portfolio,
    track their trades in a ledger, and view real-time market data from the LiveCoinWatch API.

.DESCRIPTION
    This script provides a command-line interface for a simulated Bitcoin trading experience. On first run, it guides 
    the user through setting up their LiveCoinWatch API key and initializes their portfolio with a starting capital.

    The main screen displays:
    - Real-time Bitcoin market data (Price, 24h Change, Volume).
    - The user's personal portfolio (Cash, BTC holdings, and total value).

    Users can issue commands to interact with the application:
    - buy [amount]: Purchase a specific USD amount of Bitcoin.
    - sell [amount]: Sell a specific amount of Bitcoin (e.g., .5) or satoshis (e.g., 50000s).
    - ledger: View a history of all transactions.
    - refresh: Manually update the market data.
    - config: Access the configuration menu to update the API key or reset the portfolio.
    - help: Display the help screen with a list of commands.
    - exit: Display a final portfolio summary and exit the application.

    Tip: Commands may be shortened and still accepted, e.g., "b" for "buy", "s" for "sell".
    Tip: You can input percentage instead of absolute with suffix(e.g., '10p' for 10% of your 
         USD/BTC balance). Math is also accepted, e.g., '100/3p' for 1/3 of your balance.

    Requirements:
    - PowerShell
    - An internet connection
    - A free API key from https://www.livecoinwatch.com/tools/api

.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-07-03
    Version: 1.0
#>

[CmdletBinding()]
param ()

# --- Script Setup and Configuration ---
if ($PSScriptRoot) {
    $scriptPath = $PSScriptRoot
}
elseif ($MyInvocation.MyCommand.Path) {
    $scriptPath = Split-Path -Path $MyInvocation.MyCommand.Path -Parent
}
else {
    $scriptPath = Get-Location
}
$iniFilePath = Join-Path -Path $scriptPath -ChildPath "vbtc.ini"
$ledgerFilePath = Join-Path -Path $scriptPath -ChildPath "ledger.csv"
$startingCapital = 1000.00
$sessionStartTime = (Get-Date).ToUniversalTime()

# --- Helper Functions ---

function Get-IniConfiguration {
    param ([string]$FilePath)
    Write-Verbose "Reading INI file from $FilePath"
    $ini = @{ "Settings" = @{ "ApiKey" = "" }; "Portfolio" = @{ "PlayerUSD" = $startingCapital.ToString("F2"); "PlayerBTC" = "0.0"; "PlayerInvested" = "0.0" } }
    if (Test-Path $FilePath) {
        $fileContent = Get-Content $FilePath -ErrorAction SilentlyContinue
        $currentSection = ""
        foreach ($line in $fileContent) {
            $trimmedLine = $line.Trim()
            if ($trimmedLine -match "^\[(.+)\]$") {
                $currentSection = $matches[1].Trim()
                if (-not $ini.ContainsKey($currentSection)) { $ini[$currentSection] = @{} }
            }
            elseif ($trimmedLine -match "^([^#;].*?)=(.*)$" -and $currentSection) {
                $key = $matches[1].Trim()
                $value = $matches[2].Trim()
                $ini[$currentSection][$key] = $value
                Write-Verbose "Loaded from INI: [$currentSection] $key = $value"
            }
        }
    }
    return $ini
}

function Set-IniConfiguration {
    param ([string]$FilePath, [hashtable]$Configuration)
    $iniContent = @()
    foreach ($sectionKey in $Configuration.Keys | Sort-Object) {
        $iniContent += "[$sectionKey]"
        $section = $Configuration[$sectionKey]
        foreach ($key in $section.Keys | Sort-Object) {
            $iniContent += "$key=$($section[$key])"
        }
        $iniContent += ""
    }
    try {
        Set-Content -Path $FilePath -Value $iniContent -ErrorAction Stop
        Write-Verbose "Configuration saved to: $FilePath"
    }
    catch {
        Write-Error "Failed to save configuration to $FilePath. Error: $($_.Exception.Message)"
    }
}

function Get-HistoricalData {
    param ([hashtable]$Config)
    Write-Verbose "Getting historical API data..."
    $apiKey = $Config.Settings.ApiKey
    if ([string]::IsNullOrEmpty($apiKey)) {
        Write-Warning "API Key is not configured."
        return $null
    }
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $apiKey }
    $targetTimestamp24hAgoMs = [int64]((([datetime]::UtcNow).AddHours(-24)) - (Get-Date "1970-01-01")).TotalMilliseconds
    $historicalWindowMinutes = 5
    $startTimestampMs = $targetTimestamp24hAgoMs - ($historicalWindowMinutes * 60 * 1000)
    $endTimestampMs = $targetTimestamp24hAgoMs + ($historicalWindowMinutes * 60 * 1000)

    try {
        $historicalBody = @{ currency = "USD"; code = "BTC"; start = $startTimestampMs; end = $endTimestampMs; meta = $false } | ConvertTo-Json
        Write-Verbose "Fetching historical price for 24h ago..."
        $historicalResponse = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single/history" -Method Post -Headers $headers -Body $historicalBody -ErrorAction Stop
        if ($null -ne $historicalResponse.history -and $historicalResponse.history.Count -gt 0) {
            $closestDataPoint = $historicalResponse.history | Sort-Object { [Math]::Abs($_.date - $targetTimestamp24hAgoMs) } | Select-Object -First 1
            if ($null -ne $closestDataPoint) {
                return $closestDataPoint.rate
            }
        } else { Write-Warning "No historical data returned." }
    } catch { Write-Warning "Failed to fetch historical price: $($_.Exception.Message)" }
    return $null
}

function Update-ApiData {
    param ([hashtable]$Config)
    Show-LoadingScreen
    $apiData = Get-ApiData -Config $Config
    if ($apiData) {
        $historicalRate = Get-HistoricalData -Config $Config
        if ($historicalRate) {
            $apiData | Add-Member -MemberType NoteProperty -Name "rate24hAgo" -Value $historicalRate -Force
        }
    }
    return $apiData
}

function Get-ApiData {
    param ([hashtable]$Config)
    Write-Verbose "Getting API data..."
    $apiKey = $Config.Settings.ApiKey
    if ([string]::IsNullOrEmpty($apiKey)) {
        Write-Warning "API Key is not configured."
        return $null
    }
    Write-Verbose "API Key found."
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $apiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $true } | ConvertTo-Json
    try {
        Write-Verbose "Fetching main API data..."
        $currentResponse = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -ErrorAction Stop
        $currentResponse | Add-Member -MemberType NoteProperty -Name "fetchTime" -Value (Get-Date).ToUniversalTime()
        Write-Verbose "Main API call successful."
        return $currentResponse
    }
    catch {
        Write-Error "API call failed: $($_.Exception.Message)"
        if ($_.Exception.Response) {
            $errorResponse = $_.Exception.Response.GetResponseStream()
            $reader = New-Object System.IO.StreamReader($errorResponse)
            $errorText = $reader.ReadToEnd()
            Write-Error "API Response: $errorText"
        }
        return $null
    }
}

function Test-ApiKey {
    param ([string]$ApiKey)
    if ([string]::IsNullOrEmpty($ApiKey)) { return $false }
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $true } | ConvertTo-Json
    try {
        Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}


function Get-PortfolioValue {
    param (
        [double]$PlayerUSD,
        [double]$PlayerBTC,
        [object]$ApiData
    )

    if ($ApiData -and $ApiData.PSObject.Properties['rate'] -and $ApiData.rate) {
        try {
            $btcValue = [double]$PlayerBTC * [double]$ApiData.rate
            $totalValue = $btcValue + [double]$PlayerUSD
            return [math]::Round($totalValue, 2)
        } catch {
            # If any conversion or calculation fails, fall back to just the USD value.
            return [double]$PlayerUSD
        }
    } else {
        return [double]$PlayerUSD
    }
}

# --- UI Functions ---

function Write-AlignedLine {
    param (
        [string]$Label,
        [string]$Value,
        [System.ConsoleColor]$ValueColor = "White",
        [int]$ValueStartColumn = 22
    )
    Write-Host -NoNewline $Label
    $paddingRequired = $ValueStartColumn - $Label.Length
    if ($paddingRequired -gt 0) {
        Write-Host (" " * $paddingRequired) -NoNewline
    }
    Write-Host $Value -ForegroundColor $ValueColor
}

function Show-LoadingScreen {
    Clear-Host
    Write-Host "Loading Data..." -ForegroundColor "Yellow"
}

function Show-MainScreen {
    param ($ApiData, [hashtable]$Portfolio, [double]$SessionStartValue)
    if (-not $VerbosePreference) { Clear-Host }

    # --- 1. Data Calculation ---
    $playerBTC = 0.0
    $playerUSD = 0.0
    $playerInvested = 0.0
    $null = [double]::TryParse($Portfolio.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerBTC)
    $null = [double]::TryParse($Portfolio.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerUSD)
    $null = [double]::TryParse($Portfolio.Portfolio.PlayerInvested, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerInvested)

    $marketDataAvailable = $ApiData -and $ApiData.PSObject.Properties['rate'] -and $ApiData.PSObject.Properties['volume']
    
    if ($marketDataAvailable) {
        $currentBTC = $ApiData.rate
        $rate24hAgo = if ($ApiData.PSObject.Properties['rate24hAgo']) { $ApiData.rate24hAgo } elseif ($ApiData.PSObject.Properties['delta'] -and $ApiData.delta.PSObject.Properties['day']) { $currentBTC / (1 + ($ApiData.delta.day / 100)) } else { $currentBTC }
        $percentChange = if ($rate24hAgo -ne 0) { (($currentBTC - $rate24hAgo) / $rate24hAgo) * 100 } else { 0 }
        $priceDiff = $currentBTC - $rate24hAgo
        $priceColor = if ($priceDiff -gt 0) { "Green" } elseif ($priceDiff -lt 0) { "Red" } else { "White" }
        
        # Display Values
        $btcDisplay = "{0:C2}" -f $currentBTC
        $agoDisplay = "{0:C2} [{1}{2}%]" -f $rate24hAgo, $(if($priceDiff -gt 0){"+"}), ("{0:N2}" -f $percentChange)
        $volDisplay = "$($ApiData.volume.ToString("C0"))"
        $timeDisplay = $ApiData.fetchTime.ToLocalTime().ToString("MMddyy@HHmmss")

    } else {
        # Default/Warning Values
        $priceColor = "White"
        $btcDisplay = "N/A"
        $agoDisplay = "N/A"
        $volDisplay = "N/A"
        $timeDisplay = "N/A"
    }

    # --- 2. Screen Rendering ---
    Write-Host "*** Bitcoin Market ***" -ForegroundColor Yellow
    if (-not $marketDataAvailable) { Write-Warning "Could not retrieve market data. Please check your API key in the Config menu." }
    Write-AlignedLine -Label "Bitcoin (USD):" -Value $btcDisplay -ValueColor $priceColor
    Write-AlignedLine -Label "24H Ago:" -Value $agoDisplay -ValueColor $priceColor
    Write-AlignedLine -Label "24H Volume:" -Value $volDisplay
    Write-AlignedLine -Label "Time:" -Value $timeDisplay -ValueColor "Cyan"

    Write-Host ""
    Write-Host "*** Portfolio ***" -ForegroundColor Yellow
    
    $portfolioValue = Get-PortfolioValue -PlayerUSD $playerUSD -PlayerBTC $playerBTC -ApiData $ApiData
    if (-not ($portfolioValue -is [double])) {
        $portfolioValue = "N/A"
    }
    
    $portfolioColor = "White"
    if ($portfolioValue -is [double]) {
        if ($portfolioValue -gt $startingCapital) {
            $portfolioColor = "Green"
        } elseif ($portfolioValue -lt $startingCapital) {
            $portfolioColor = "Red"
        }
    }

    if ($playerBTC -gt 0) {
        $btcLabel = "Bitcoin:"
        $btcAmountDisplay = $playerBTC.ToString("F8")
        $btcValueDisplay = ""
        if ($ApiData -and $ApiData.PSObject.Properties['rate']) {
            $btcValue = $playerBTC * $ApiData.rate
            $btcValueDisplay = " ($($btcValue.ToString("C2")))"
        }
        
        Write-Host -NoNewline $btcLabel
        $paddingRequired = 22 - $btcLabel.Length
        if ($paddingRequired -gt 0) {
            Write-Host (" " * $paddingRequired) -NoNewline
        }
        Write-Host -NoNewline $btcAmountDisplay
        Write-Host $btcValueDisplay

        $investedLabel = "Invested:"
        $investedDisplay = $playerInvested.ToString("C2")
        $investedChangeDisplay = ""
        $changeColor = "White"
        if ($playerInvested -gt 0 -and $ApiData -and $ApiData.PSObject.Properties['rate']) {
            $btcValue = $playerBTC * $ApiData.rate
            $percentChange = (($btcValue - $playerInvested) / $playerInvested) * 100
            $changeColor = if ($percentChange -gt 0) { "Green" } elseif ($percentChange -lt 0) { "Red" } else { "White" }
            $investedChangeDisplay = " [{0}{1}%]" -f $(if($percentChange -gt 0){"+"}), ("{0:N2}" -f $percentChange)
        }

        Write-Host -NoNewline $investedLabel
        $paddingRequired = 22 - $investedLabel.Length
        if ($paddingRequired -gt 0) {
            Write-Host (" " * $paddingRequired) -NoNewline
        }
        Write-Host -NoNewline $investedDisplay
        Write-Host $investedChangeDisplay -ForegroundColor $changeColor
    }

    if ($playerUSD -gt 0) {
        Write-AlignedLine -Label "Cash:" -Value $playerUSD.ToString("C2")
    }
    
    $displayValue = if ($portfolioValue -is [double]) { $portfolioValue.ToString("C2") } else { $portfolioValue }
    Write-AlignedLine -Label "Value (USD):" -Value $displayValue -ValueColor $portfolioColor

    if ($SessionStartValue -gt 0 -and $portfolioValue -is [double]) {
        $sessionChange = $portfolioValue - $SessionStartValue
        $sessionPercent = ($sessionChange / $SessionStartValue) * 100
        $sessionColor = if ($sessionChange -gt 0) { "Green" } elseif ($sessionChange -lt 0) { "Red" } else { "White" }
        $sessionDisplay = "{0}{1:C2} [{2}{3:N2}%]" -f $(if($sessionChange -gt 0){"+"}), $sessionChange, $(if($sessionPercent -gt 0){"+"}), $sessionPercent
        Write-AlignedLine -Label "Session P/L:" -Value $sessionDisplay -ValueColor $sessionColor
    }
    
    Write-Host ""
    Write-Host -NoNewline "Commands: " -ForegroundColor Yellow
    Write-Host -NoNewline "Buy " -ForegroundColor Green
    Write-Host -NoNewline "Sell " -ForegroundColor Red
    Write-Host -NoNewline "Ledger " -ForegroundColor Yellow
    Write-Host -NoNewline "Refresh " -ForegroundColor Cyan
    Write-Host -NoNewline "Config " -ForegroundColor DarkYellow
    Write-Host -NoNewline "Help " -ForegroundColor Blue
    Write-Host "Exit" -ForegroundColor Yellow
}

function Show-FirstRunSetup {
    param ([ref]$Config)
    Clear-Host
    Write-Host "*** First Time Setup ***" -ForegroundColor Yellow
    $validKey = $false
    while (-not $validKey) {
        $apiKey = Read-Host "Please enter your LiveCoinWatch API Key"
        if (Test-ApiKey -ApiKey $apiKey) {
            $validKey = $true
            $Config.Value.Settings.ApiKey = $apiKey
            Set-IniConfiguration -FilePath $iniFilePath -Configuration $Config.Value
            Write-Host "API Key saved. Welcome!" -ForegroundColor Green
            Read-Host "Press Enter to start."
        } else {
            Write-Warning "Invalid API Key. Please try again."
        }
    }
}

function Show-ConfigScreen {
    param ([ref]$Config)
    while ($true) {
        Clear-Host
        Write-Host "*** Configuration ***" -ForegroundColor Yellow
        Write-Host "1. Update API Key"
        Write-Host "2. Reset Portfolio"
        Write-Host "3. Return to Main Screen"
        $choice = Read-Host "Enter your choice"

        switch ($choice) {
            "1" {
                $newApiKey = Read-Host "Enter your new LiveCoinWatch API Key"
                if (Test-ApiKey -ApiKey $newApiKey) {
                    $Config.Value.Settings.ApiKey = $newApiKey
                    Set-IniConfiguration -FilePath $iniFilePath -Configuration $Config.Value
                    Write-Host "API Key updated successfully." -ForegroundColor Green
                } else {
                    Write-Warning "The new API Key is invalid. It has not been saved."
                }
                Read-Host "Press Enter to continue."
            }
            "2" {
                $confirmReset = Read-Host "Are you sure you want to reset your portfolio? This cannot be undone. Type 'YES' to confirm"
                if ($confirmReset -eq "YES") {
                    $Config.Value.Portfolio.PlayerUSD = $startingCapital.ToString("F2")
                    $Config.Value.Portfolio.PlayerBTC = "0.0"
                    $Config.Value.Portfolio.PlayerInvested = "0.0"
                    if (Test-Path $ledgerFilePath) { Remove-Item $ledgerFilePath }
                    Set-IniConfiguration -FilePath $iniFilePath -Configuration $Config.Value
                    Write-Host "Portfolio has been reset." -ForegroundColor Green
                } else {
                    Write-Host "Portfolio reset cancelled."
                }
                Read-Host "Press Enter to continue."
            }
            "3" { return }
            default { Write-Warning "Invalid choice. Please try again."; Read-Host "Press Enter to continue." }
        }
    }
}

function Add-LedgerEntry {
    param ([string]$Type, [double]$UsdAmount, [double]$BtcAmount, [double]$BtcPrice, [double]$UserBtcAfter)
    
    try {
        if (-not (Test-Path $ledgerFilePath)) {
            Set-Content -Path $ledgerFilePath -Value '"TX","USD","BTC","BTC(USD)","User BTC","Time"' -ErrorAction Stop
        }
        $timestamp = (Get-Date).ToUniversalTime().ToString("MMddyy@HHmmss")
        $logEntry = """$Type"",""$UsdAmount"",""$BtcAmount"",""$BtcPrice"",""$UserBtcAfter"",""$timestamp"""
        Add-Content -Path $ledgerFilePath -Value $logEntry -ErrorAction Stop
    }
    catch {
        Write-Warning "Transaction complete, but failed to write to ledger.csv. Please ensure the file is not open in another program."
        # This is a non-terminating error, so we just warn the user.
    }
}

function Get-LedgerTotals {
    param($LedgerData)

    $totalBuyUSD = 0.0
    $totalSellUSD = 0.0
    $totalBuyBTC = 0.0
    $totalSellBTC = 0.0

    if ($null -ne $LedgerData) {
        $totalBuyUSD = ($LedgerData | Where-Object { $_.TX -eq "Buy" } | ForEach-Object { [double]$_.USD } | Measure-Object -Sum).Sum
        $totalSellUSD = ($LedgerData | Where-Object { $_.TX -eq "Sell" } | ForEach-Object { [double]$_.USD } | Measure-Object -Sum).Sum
        $totalBuyBTC = ($LedgerData | Where-Object { $_.TX -eq "Buy" } | ForEach-Object { [double]$_.BTC } | Measure-Object -Sum).Sum
        $totalSellBTC = ($LedgerData | Where-Object { $_.TX -eq "Sell" } | ForEach-Object { [double]$_.BTC } | Measure-Object -Sum).Sum
    }

    return [PSCustomObject]@{
        TotalBuyUSD  = $totalBuyUSD
        TotalSellUSD = $totalSellUSD
        TotalBuyBTC  = $totalBuyBTC
        TotalSellBTC = $totalSellBTC
    }
}

function Get-LedgerSummary {
    if (-not (Test-Path $ledgerFilePath)) { return $null }
    $ledgerData = Import-Csv -Path $ledgerFilePath
    if ($ledgerData.Count -eq 0) { return $null }
    return Get-LedgerTotals -LedgerData $ledgerData
}

function Get-SessionSummary {
    param ([datetime]$SessionStartTime)
    if (-not (Test-Path $ledgerFilePath)) {
        return $null
    }
    $sessionTransactions = @(Import-Csv -Path $ledgerFilePath) | Where-Object {
        try {
            return ([datetime]::ParseExact($_.Time, "MMddyy@HHmmss", $null) -ge $SessionStartTime)
        } catch {
            # This will prevent a crash if a line in the ledger is corrupted
            Write-Warning "Could not parse timestamp '$($_.Time)' in ledger.csv"
            return $false
        }
    }

    if ($null -eq $sessionTransactions -or $sessionTransactions.Count -eq 0) {
        return $null
    }

    return Get-LedgerTotals -LedgerData $sessionTransactions
}

function Invoke-Trade {
    param ([ref]$Config, [string]$Type, [string]$AmountString = $null)
    
    $playerUSD = 0.0
    $playerBTC = 0.0
    $playerInvested = 0.0
    $null = [double]::TryParse($Config.Value.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerUSD)
    $null = [double]::TryParse($Config.Value.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerBTC)
    $null = [double]::TryParse($Config.Value.Portfolio.PlayerInvested, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerInvested)

    $maxAmount = if ($Type -eq "Buy") { $playerUSD } else { $playerBTC }
    $prompt = if ($Type -eq "Buy") {
        "Amount in USD: [Max $($maxAmount.ToString("C2"))]"
    } else {
        "Amount in BTC: [Max $($maxAmount.ToString("F8"))] (or use 's' for satoshis)"
    }

    $userInput = $AmountString
    $tradeAmount = 0.0

    # --- Input Loop for Trade Amount ---
    while ($true) {
        Clear-Host
        Write-Host "*** $Type Bitcoin ***" -ForegroundColor Yellow
        if ([string]::IsNullOrEmpty($userInput)) {
            $userInput = Read-Host $prompt
            if ([string]::IsNullOrEmpty($userInput)) { return } # User cancelled
        }

        $parsedAmount = 0.0
        $parseSuccess = $false
        if ($userInput.Trim().EndsWith("s")) {
            if ($Type -eq "Buy") {
                Write-Warning "Satoshi notation ('s') is only valid for selling."
                $userInput = $null; Read-Host "Press Enter to continue."; continue
            }
            $satoshiString = $userInput.Trim().TrimEnd("s")
            $satoshiValue = 0.0
            if ([double]::TryParse($satoshiString, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$satoshiValue)) {
                $parsedAmount = $satoshiValue / 100000000
                $parseSuccess = $true
            }
        } elseif ($userInput.Trim().EndsWith("p")) {
            $percentString = $userInput.Trim().TrimEnd("p")
            try {
                $percentValue = Invoke-Expression $percentString
                if ($percentValue -gt 0 -and $percentValue -le 100) {
                    # Perform calculation and truncate to prevent floating point errors from exceeding the balance.
                    $calculatedAmount = ($maxAmount * $percentValue) / 100
                    if ($Type -eq "Sell") {
                        # Truncate BTC to 8 decimal places (satoshi)
                        $parsedAmount = [math]::Floor($calculatedAmount * 100000000) / 100000000
                    } else { # Buy
                        # Truncate USD to 2 decimal places (cent)
                        $parsedAmount = [math]::Floor($calculatedAmount * 100) / 100
                    }
                    $parseSuccess = $true
                } else {
                    Write-Warning "Percentage must be between 0 and 100."
                    $userInput = $null; Read-Host "Press Enter to continue."; continue
                }
            } catch {
                Write-Warning "Invalid percentage expression."
                $userInput = $null; Read-Host "Press Enter to continue."; continue
            }
        } else {
            if ([double]::TryParse($userInput, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$parsedAmount)) {
                $parseSuccess = $true
            }
        }

        if (-not $parseSuccess) {
            Write-Warning "Invalid number format."
            $userInput = $null; Read-Host "Press Enter to continue."; continue
        }
        
        $currentTradeAmount = if ($Type -eq "Buy") { [math]::Round($parsedAmount, 2) } else { [math]::Round($parsedAmount, 8) }

        if ($currentTradeAmount -le 0) {
            Write-Warning "Please enter a positive number."
            $userInput = $null; Read-Host "Press Enter to continue."; continue
        } elseif ($currentTradeAmount -gt $maxAmount) {
            Write-Warning "Amount exceeds your balance."
            $userInput = $null; Read-Host "Press Enter to continue."; continue
        }
        
        $tradeAmount = $currentTradeAmount
        break # Exit loop once valid trade amount is entered
    }

    # --- Confirmation Loop with Timeout ---
    $offerExpired = $false
    while ($true) {
        Show-LoadingScreen
        $apiData = Get-ApiData -Config $Config.Value
        if (-not $apiData) { Read-Host "Error fetching price. Press Enter to continue."; return }
        $currentBTC = $apiData.rate
        
        Clear-Host
        Write-Host "*** $Type Bitcoin ***" -ForegroundColor Yellow
        if ($offerExpired) {
            Write-Host "`nOffer expired. Fetching a new price..." -ForegroundColor Yellow
        }

        $usdAmount = if ($Type -eq "Buy") { $tradeAmount } else { [math]::Floor(($tradeAmount * $currentBTC) * 100) / 100 }
        $btcAmount = if ($Type -eq "Sell") { $tradeAmount } else { [math]::Floor(($tradeAmount / $currentBTC) * 100000000) / 100000000 }

        $rate24hAgo = if ($apiData.PSObject.Properties['rate24hAgo']) { $apiData.rate24hAgo } elseif ($apiData.PSObject.Properties['delta'] -and $apiData.delta.PSObject.Properties['day']) { $currentBTC / (1 + ($apiData.delta.day / 100)) } else { $currentBTC }
        $priceDiff = $currentBTC - $rate24hAgo
        $priceColor = if ($priceDiff -gt 0) { "Green" } elseif ($priceDiff -lt 0) { "Red" } else { "White" }

        $confirmPrompt = if ($Type -eq "Buy") {
            "Purchase $($btcAmount.ToString("F8")) BTC for $($usdAmount.ToString("C2"))? "
        } else {
            "Sell $($btcAmount.ToString("F8")) BTC for $($usdAmount.ToString("C2"))? "
        }
        
        Write-Host "`nYou have 2 minutes to accept this offer." -ForegroundColor Yellow
        Write-Host -NoNewline "Market Rate: "
        Write-Host ("{0:C2}" -f $currentBTC) -ForegroundColor $priceColor
        
        Write-Host -NoNewline $confirmPrompt
        Write-Host "[" -NoNewline
        Write-Host "y" -ForegroundColor Green -NoNewline
        Write-Host "/" -NoNewline
        Write-Host "r" -ForegroundColor Cyan -NoNewline
        Write-Host "/" -NoNewline
        Write-Host "n" -ForegroundColor Red -NoNewline
        Write-Host "]"
        
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $tradeinput = Read-Host

        if ($stopwatch.Elapsed.TotalMinutes -ge 2) {
            $offerExpired = $true
            continue
        }

        if ($tradeinput.ToLower() -eq 'y') {
            $newUserBtc = 0.0
            $newInvested = 0.0
            if ($Type -eq "Buy") {
                $Config.Value.Portfolio.PlayerUSD = ($playerUSD - $usdAmount).ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
                $newUserBtc = $playerBTC + $btcAmount
                $newInvested = $playerInvested + $usdAmount
                $Config.Value.Portfolio.PlayerBTC = $newUserBtc.ToString("F8", [System.Globalization.CultureInfo]::InvariantCulture)
                $Config.Value.Portfolio.PlayerInvested = $newInvested.ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
            } else { # Sell
                $newUserBtc = $playerBTC - $btcAmount
                if ($newUserBtc -le 0.000000005) { # Use a small tolerance for floating point comparison
                    $newInvested = 0
                    $newUserBtc = 0 # Ensure it's exactly zero if we're clearing it
                } else {
                    # Reduce invested capital proportionally to the amount of BTC sold.
                    # This preserves the average cost basis of the remaining holdings.
                    if ($playerBTC -gt 0) {
                        $newInvested = $playerInvested * ($newUserBtc / $playerBTC)
                    }
                    else {
                        $newInvested = 0
                    }
                }
                $Config.Value.Portfolio.PlayerBTC = $newUserBtc.ToString("F8", [System.Globalization.CultureInfo]::InvariantCulture)
                $Config.Value.Portfolio.PlayerUSD = ($playerUSD + $usdAmount).ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
                $Config.Value.Portfolio.PlayerInvested = $newInvested.ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
            }
            Set-IniConfiguration -FilePath $iniFilePath -Configuration $Config.Value
            Add-LedgerEntry -Type $Type -UsdAmount $usdAmount -BtcAmount $btcAmount -BtcPrice $currentBTC -UserBtcAfter $newUserBtc
            Write-Host "`n$Type successful."
            Start-Sleep -Seconds 1
            return 
        } 
        elseif ($tradeinput.ToLower() -eq 'r') {
            $offerExpired = $true 
            continue 
        }
        
        Write-Host "`n$Type cancelled."
        Start-Sleep -Seconds 1
        return
    }
}

function Show-HelpScreen {
    Clear-Host
    Write-Host "*** Help ***" -ForegroundColor Yellow
    Write-AlignedLine -Label "buy [amount]" -Value "Purchase a specific USD amount of Bitcoin."
    Write-AlignedLine -Label "sell [amount]" -Value "Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s)."
    Write-AlignedLine -Label "ledger" -Value "View a history of all your transactions."
    Write-AlignedLine -Label "refresh" -Value "Manually update the market data."
    Write-AlignedLine -Label "config" -Value "Access the configuration menu."
    Write-AlignedLine -Label "help" -Value "Show this help screen."
    Write-AlignedLine -Label "exit" -Value "Exit the application."
    Write-Host ""
    Read-Host "Press Enter to return to the Main Screen."
}

function Show-LedgerScreen {
    Clear-Host
    Write-Host "*** Ledger ***" -ForegroundColor Yellow
    
    if (-not (Test-Path $ledgerFilePath)) {
        Write-Host "You have not made any transactions yet."
    } else {
        $ledgerData = Import-Csv -Path $ledgerFilePath
        if ($ledgerData.Count -eq 0) {
            Write-Host "You have not made any transactions yet."
        } else {
            # 1. Parse and create display objects
            $displayData = $ledgerData | ForEach-Object {
                [PSCustomObject]@{
                    TX         = $_.TX
                    USD        = [double]::Parse($_.USD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture)
                    BTC        = [double]::Parse($_.BTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture)
                    'BTC(USD)' = [double]::Parse($_.'BTC(USD)', [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture)
                    'User BTC' = [double]::Parse($_.'User BTC', [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture)
                    Time       = $_.Time
                    DateTime   = [datetime]::ParseExact($_.Time, "MMddyy@HHmmss", $null)
                }
            }

            # 2. Calculate column widths
            $widths = @{}
            $columns = $displayData[0].psobject.Properties.Name
            foreach ($col in $columns) {
                if ($col -eq 'DateTime') { continue }
                $headerLength = $col.Length
                $maxLength = ($displayData | ForEach-Object {
                    $value = $_.($col)
                    $stringValue = ""
                    if ($col -in @('USD', 'BTC(USD)')) {
                        $stringValue = $value.ToString("C2")
                    }
                    elseif ($col -in @('BTC', 'User BTC')) {
                        $stringValue = $value.ToString("F8")
                    }
                    else {
                        $stringValue = $value.ToString()
                    }
                    $stringValue.Length
                } | Measure-Object -Maximum).Maximum
                $widths[$col] = [math]::Max($headerLength, $maxLength)
            }

            # 3. Create header and format string
            $headerString = ""
            $separator = ""
            $formatString = ""
            $padding = 2 # Spaces between columns
            $valueIndex = 0
            foreach ($col in $columns) {
                if ($col -eq 'DateTime') { continue }
                $width = $widths[$col]
                $headerString += "$($col.PadRight($width))" + (" " * $padding)
                $separator += ("-" * $width) + (" " * $padding)
                $formatString += "{" + $valueIndex + ",-$width}" + (" " * $padding)
                $valueIndex++
            }
            
            # 4. Print header
            Write-Host $headerString
            Write-Host $separator

            # 5. Print data rows
            $sessionStarted = $false
            foreach ($row in $displayData | Sort-Object DateTime) {
                if (-not $sessionStarted -and $row.DateTime -ge $sessionStartTime) {
                    $totalWidth = $separator.Length
                    $sessionText = "*** Current Session Start ***"
                    $paddingLength = [math]::Max(0, [math]::Floor(($totalWidth - $sessionText.Length) / 2))
                    $centeredText = (" " * $paddingLength) + $sessionText
                    Write-Host $centeredText -ForegroundColor White
                    $sessionStarted = $true
                }
                $rowColor = if ($row.TX -eq "Buy") { "Green" } else { "Red" }
                
                $values = $columns | Where-Object { $_ -ne 'DateTime' } | ForEach-Object {
                    $value = $row.$_
                    if ($_ -in @('USD', 'BTC(USD)')) {
                        $value.ToString("C2")
                    }
                    elseif ($_ -in @('BTC', 'User BTC')) {
                        $value.ToString("F8")
                    }
                    else {
                        $value.ToString()
                    }
                }
                Write-Host ($formatString -f $values) -ForegroundColor $rowColor
            }

            $summary = Get-LedgerSummary
            if ($summary) {
                Write-Host ""
                Write-Host "*** Ledger Summary ***" -ForegroundColor Yellow
                Write-AlignedLine -Label "Total Bought (USD):" -Value ("{0:C2}" -f $summary.TotalBuyUSD) -ValueColor "Green"
                Write-AlignedLine -Label "Total Sold (USD):" -Value ("{0:C2}" -f $summary.TotalSellUSD) -ValueColor "Red"
                Write-AlignedLine -Label "Total Bought (BTC):" -Value $summary.TotalBuyBTC.ToString("F8") -ValueColor "Green"
                Write-AlignedLine -Label "Total Sold (BTC):" -Value $summary.TotalSellBTC.ToString("F8") -ValueColor "Red"
            }
        }
    }
    
    Read-Host "Press Enter to return to Main screen"
}


# --- Main Game Loop ---

$config = Get-IniConfiguration -FilePath $iniFilePath
if ([string]::IsNullOrEmpty($config.Settings.ApiKey)) {
    Show-FirstRunSetup -Config ([ref]$config)
    $config = Get-IniConfiguration -FilePath $iniFilePath
}

$apiData = Update-ApiData -Config $config
$initialPlayerUSD = 0.0
$initialPlayerBTC = 0.0
$null = [double]::TryParse($config.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$initialPlayerUSD)
$null = [double]::TryParse($config.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$initialPlayerBTC)
$sessionStartPortfolioValue = Get-PortfolioValue -PlayerUSD $initialPlayerUSD -PlayerBTC $initialPlayerBTC -ApiData $apiData

$commands = @("buy", "sell", "ledger", "refresh", "config", "help", "exit")

while ($true) {
    Show-MainScreen -ApiData $apiData -Portfolio $config -SessionStartValue $sessionStartPortfolioValue

    $userInput = (Read-Host "Enter command").Trim()
    if ([string]::IsNullOrEmpty($userInput)) { continue }

    $parts = $userInput.Split(" ", 2)
    $commandInput = $parts[0].ToLower()
    $amount = if ($parts.Count -gt 1) { $parts[1] } else { $null }

    $matchedCommands = @($commands | Where-Object { $_.StartsWith($commandInput) })

    if ($matchedCommands.Count -eq 1) {
        $command = $matchedCommands[0]
        switch ($command) {
            "buy" {
                Invoke-Trade -Config ([ref]$config) -Type "Buy" -AmountString $amount
                $apiData = Get-ApiData -Config $config
            }
            "sell" {
                Invoke-Trade -Config ([ref]$config) -Type "Sell" -AmountString $amount
                $apiData = Get-ApiData -Config $config
            }
            "ledger" { Show-LedgerScreen }
            "refresh" { $apiData = Update-ApiData -Config $config }
            "config" {
                Show-ConfigScreen -Config ([ref]$config)
            }
            "help" { Show-HelpScreen }
            "exit" {
                Clear-Host
                Write-Host "*** Portfolio Summary ***" -ForegroundColor Yellow

                $playerBTC = 0.0
                $playerUSD = 0.0
                $null = [double]::TryParse($config.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerBTC)
                $null = [double]::TryParse($config.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerUSD)
                $finalValue = Get-PortfolioValue -PlayerUSD $playerUSD -PlayerBTC $playerBTC -ApiData $apiData

                $profit = $finalValue - $startingCapital
                $profitColor = if ($profit -gt 0) { "Green" } elseif ($profit -lt 0) { "Red" } else { "White" }
            
                Write-AlignedLine -Label "Portfolio Value:" -Value ("{0:C2}" -f $finalValue) -ValueColor $profitColor
                Write-AlignedLine -Label "Total Profit/Loss:" -Value ("{0:C2}" -f $profit) -ValueColor $profitColor

                if ($sessionStartPortfolioValue -gt 0 -and $finalValue -is [double]) {
                    $sessionChange = $finalValue - $sessionStartPortfolioValue
                    $sessionPercent = ($sessionChange / $sessionStartPortfolioValue) * 100
                    $sessionColor = if ($sessionChange -gt 0) { "Green" } elseif ($sessionChange -lt 0) { "Red" } else { "White" }
                    $sessionDisplay = "{0}{1:C2} [{2}{3:N2}%]" -f $(if($sessionChange -gt 0){"+"}), $sessionChange, $(if($sessionPercent -gt 0){"+"}), $sessionPercent
                    Write-AlignedLine -Label "Session P/L:" -Value $sessionDisplay -ValueColor $sessionColor
                }

                $summary = Get-SessionSummary -SessionStartTime $sessionStartTime
                if ($summary) {
                    Write-Host ""
                    Write-Host "*** Session Summary ***" -ForegroundColor Yellow
                    if ($summary.TotalBuyUSD -gt 0) {
                        Write-AlignedLine -Label "Total Bought (USD):" -Value ("{0:C2}" -f $summary.TotalBuyUSD) -ValueColor "Green"
                        Write-AlignedLine -Label "Total Bought (BTC):" -Value $summary.TotalBuyBTC.ToString("F8") -ValueColor "Green"
                    }
                    if ($summary.TotalSellUSD -gt 0) {
                        Write-AlignedLine -Label "Total Sold (USD):" -Value ("{0:C2}" -f $summary.TotalSellUSD) -ValueColor "Red"
                        Write-AlignedLine -Label "Total Sold (BTC):" -Value $summary.TotalSellBTC.ToString("F8") -ValueColor "Red"
                    }
                }
            
                Write-Host ""
                $parentProcess = (Get-CimInstance -ClassName Win32_Process -Filter "ProcessId = $PID").ParentProcessId
                $parentProcessName = (Get-Process -Id $parentProcess).ProcessName
                if ($parentProcessName -notin @("cmd", "powershell", "explorer")) {
                    Read-Host "Press Enter to exit"
                }
                exit
            }
        }
    } elseif ($matchedCommands.Count -gt 1) {
        Write-Warning "Ambiguous command. Did you mean: $($matchedCommands -join ', ')? "
        Read-Host "Press Enter to continue."
    } else {
        Write-Warning "Invalid command. Type 'help' for a list of commands."
        Read-Host "Press Enter to continue."
    }
}