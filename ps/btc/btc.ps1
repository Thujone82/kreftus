<#
.SYNOPSIS
    Retrieves and displays Bitcoin (BTC) data from LiveCoinWatch API. Supports .ini configuration,
    profit/loss calculation, CSV logging, first-run setup, config update, and accurate 24h price
    difference calculation. Uses -Verbose for detailed operational messages. Clears console on start.

.PARAMETER UserBTCAmount
    Optional. Amount of Bitcoin owned. Overrides 'MyBTC' from btc.ini.
.PARAMETER UserTotalCost
    Optional. Total cost for the 'UserBTCAmount' of Bitcoin. Overrides 'MyCOST' from btc.ini.
    Required for Profit/Loss calculation if UserBTCAmount is also provided.
.PARAMETER LogToFile
    Optional. Path to the CSV log file. Overrides 'LogPath' from btc.ini.
.PARAMETER Update
    Optional. Switch to interactively update MyBTC, MyCOST, and LogPath in btc.ini.
    Aliases: -u, -config, -c
.PARAMETER Verbose
    Optional. Common parameter to display detailed operational messages.

.DESCRIPTION
    Clears the console. If btc.ini or ApiKey is missing, prompts for first-run setup.
    If -Update switch is used, prompts to update portfolio/log settings in btc.ini.
    Then fetches detailed Bitcoin data via LiveCoinWatch API.
    Core financial data is always displayed. Use -Verbose to see step-by-step messages.
    Features:
    - Reads/Writes configuration to 'btc.ini' (ApiKey, LogPath, MyBTC, MyCOST).
    - Command-line parameters override .ini settings for the current run.
    - Bitcoin price line: Current price (color-coded), 24h price difference (e.g., [+$100.50])
      calculated from actual historical price. Color is purple if current price is within 5%
      of All-Time High (ATH), otherwise green/red based on the calculated 24h dollar difference.
    - My BTC Value line (Optional): Current value (color-coded based on the calculated 24h dollar
      difference of BTC), 24h value difference calculated from actual historical price.
    - Profit/Loss line (Optional): Displays profit/loss in USD and percentage if MyBTC and MyCOST are provided.
    - 24H Volume line: Current volume (color-coded based on API's 24h price delta for BTC).
    - Market Cap line: Current market cap (color-coded based on API's 24h price delta for BTC).
    - Logs data to a CSV file if LogPath is configured and not empty.
    Monetary values align at column 16. Appended details are in default console color.

.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-10-31
    Version: 2.2 
    Added [CmdletBinding()] for robust -Verbose handling.
    btc.ini will be created/updated in the same directory as the script.
    Uses a second API call to /coins/single/history for more accurate 24h price difference and additional metrics.
    Color coding for BTC price and My BTC value now directly reflects the sign of the calculated 24h dollar difference.
    Added 1H SMA, 24H Ago, 24H High/Low, Volatility metrics, and History sparkline visualization.
#>

[CmdletBinding()] # Makes this an advanced script, ensuring -Verbose works as expected
param (
    [double]$UserBTCAmount,
    [double]$UserTotalCost,
    [string]$LogToFile,
    [Alias('u','config','c')]
    [switch]$Update
)

# --- Clear Console (Must be AFTER param block) ---
Clear-Host

# --- Helper Function to Parse INI File ---
function Get-IniConfiguration {
    param ([string]$FilePath)
    $ini = @{ "Settings" = @{ "ApiKey" = ""; "LogPath" = "" }; "Portfolio" = @{ "MyBTC" = "0.0"; "MyCOST" = "0.0" } }
    if (Test-Path $FilePath) {
        Write-Verbose "Reading configuration from $FilePath"
        $fileContent = Get-Content $FilePath -ErrorAction SilentlyContinue; $currentSection = ""
        foreach ($line in $fileContent) {
            $trimmedLine = $line.Trim()
            if ($trimmedLine -match "^\[(.+)\]$") { $currentSection = $matches[1].Trim(); if (-not $ini.ContainsKey($currentSection)) { $ini[$currentSection] = @{} } }
            elseif ($trimmedLine -match "^([^#;].*?)=(.*)$" -and $currentSection) { $key = $matches[1].Trim(); $value = $matches[2].Trim(); $ini[$currentSection][$key] = $value }
        }
    } else {
        Write-Verbose "Configuration file not found at $FilePath."
    }
    return $ini
}

# --- Helper Function to Write INI File ---
function Set-IniConfiguration {
    param ([string]$FilePath, [hashtable]$Configuration)
    $iniContent = @(); foreach ($sectionKey in $Configuration.Keys | Sort-Object) {
        $iniContent += "[$sectionKey]"; $section = $Configuration[$sectionKey]
        foreach ($key in $section.Keys | Sort-Object) { $iniContent += "$key=$($section[$key])" }
        $iniContent += ""
    }
    try { Set-Content -Path $FilePath -Value $iniContent -ErrorAction Stop; Write-Verbose "Configuration saved to: $FilePath" } 
    catch { Write-Error "Failed to save configuration to $FilePath. Error: $($_.Exception.Message)" }
}

# --- Helper Function to Write Colored and Aligned Output ---
function Write-ColoredLine {
    param (
        [string]$Label, [string]$Prefix = "$", [double]$Value, [string]$FormatString,
        [double]$ChangeIndicator, [int]$ValueStartColumn = 16, 
        [AllowNull()][double]$PriceChangeAmountToDisplay = $null,
        [string]$ChangeDisplayPrefixBracket = "[", [string]$ChangeDisplaySuffixBracket = "]",
        [AllowNull()][double]$PercentageChangeToDisplay = $null, [string]$PercentageLabel = "%",
        [AllowNull()][string]$ExplicitColorName = $null
    )
    Write-Host -NoNewline -ForegroundColor White $Label; $paddingRequired = ($ValueStartColumn - 1) - $Label.Length
    if ($paddingRequired -lt 0) { $paddingRequired = 0 }; $paddingSpaces = " " * $paddingRequired
    Write-Host -NoNewline $paddingSpaces; $formattedValueString = $Value.ToString($FormatString)
    $displayString = "$($Prefix)$($formattedValueString)"
    if (-not [string]::IsNullOrEmpty($ExplicitColorName)) {
        try { $colorToUse = [System.Enum]::Parse([System.ConsoleColor], $ExplicitColorName, $true); Write-Host -NoNewline -ForegroundColor $colorToUse $displayString } catch { Write-Host -NoNewline $displayString }
    } elseif ($ChangeIndicator -gt 0) { Write-Host -NoNewline -ForegroundColor ([System.ConsoleColor]::Green) $displayString }
    elseif ($ChangeIndicator -lt 0) { Write-Host -NoNewline -ForegroundColor ([System.ConsoleColor]::Red) $displayString }
    else { Write-Host -NoNewline $displayString } 
    if ($PSBoundParameters.ContainsKey('PriceChangeAmountToDisplay') -and $null -ne $PriceChangeAmountToDisplay) {
        $changeAmount = $PriceChangeAmountToDisplay; $formattedChangeAmount = "{0:N2}" -f [Math]::Abs($changeAmount); $sign = if ($changeAmount -gt 0) {"+"} elseif ($changeAmount -lt 0) {"-"} else {""}
        $changeValueDisplayString = "$($ChangeDisplayPrefixBracket)$($sign)$($Prefix)$($formattedChangeAmount)$($ChangeDisplaySuffixBracket)"; Write-Host -NoNewline " $($changeValueDisplayString)"
    }
    if ($PSBoundParameters.ContainsKey('PercentageChangeToDisplay') -and $null -ne $PercentageChangeToDisplay) {
        $percValue = $PercentageChangeToDisplay; $formattedNumericPartOfPercentage = "{0:N2}" -f $percValue
        if ($formattedNumericPartOfPercentage -ne "0.00" -or $PercentageLabel -ne "%") {
            $signedFormattedPercentage = $formattedNumericPartOfPercentage; if ($percValue -gt 0 -and $percValue -ne 0) { $signedFormattedPercentage = "+$formattedNumericPartOfPercentage" }
            Write-Host -NoNewline " ($($signedFormattedPercentage)$($PercentageLabel))"
        }
    }
    Write-Host ""
}

# --- Helper Function to Get Historical Data ---
function Get-HistoricalData {
    param ([string]$ApiKey, [string]$Currency, [string]$CoinCode)
    Write-Verbose "Getting historical API data..."
    if ([string]::IsNullOrEmpty($ApiKey)) {
        Write-Warning "API Key is not configured."
        return $null
    }
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $endTimestampMs = [int64](([datetime]::UtcNow) - (Get-Date "1970-01-01")).TotalMilliseconds
    $startTimestampMs = $endTimestampMs - (24 * 60 * 60 * 1000) # Full 24 hours

    try {
        $historicalBody = @{ currency = $Currency; code = $CoinCode; start = $startTimestampMs; end = $endTimestampMs; meta = $false } | ConvertTo-Json
        Write-Verbose "Fetching historical price for 24h (start: $startTimestampMs, end: $endTimestampMs)..."
        $historicalResponse = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single/history" -Method Post -Headers $headers -Body $historicalBody -ErrorAction Stop
        Write-Verbose "Historical API call completed. Response has history? $($null -ne $historicalResponse.history), Count: $(if($null -ne $historicalResponse.history){$historicalResponse.history.Count}else{'N/A'})"
        if ($null -ne $historicalResponse.history -and $historicalResponse.history.Count -gt 0) {
            Write-Verbose "Processing $($historicalResponse.history.Count) historical data points..."
            # Overall 24h stats
            $lowPoint24h = $historicalResponse.history | Sort-Object -Property rate | Select-Object -First 1
            $highPoint24h = $historicalResponse.history | Sort-Object -Property rate -Descending | Select-Object -First 1

            $targetTimestamp24hAgoMs = [int64]((([datetime]::UtcNow).AddHours(-24)) - (Get-Date "1970-01-01")).TotalMilliseconds
            $closestDataPoint = $historicalResponse.history | Sort-Object { [Math]::Abs($_.date - $targetTimestamp24hAgoMs) } | Select-Object -First 1

            $volatility24h = 0
            if ($lowPoint24h.rate -gt 0) {
                $volatility24h = (($highPoint24h.rate - $lowPoint24h.rate) / $lowPoint24h.rate) * 100
            }

            # 12-hour volatility stats
            $midpointTimestampMs = [int64]((([datetime]::UtcNow).AddHours(-12)) - (Get-Date "1970-01-01")).TotalMilliseconds
            $recentHistory = $historicalResponse.history | Where-Object { $_.date -ge $midpointTimestampMs }
            $oldHistory = $historicalResponse.history | Where-Object { $_.date -lt $midpointTimestampMs }

            $volatility12h = 0
            if ($recentHistory -and $recentHistory.Count -gt 0) {
                $recentStats = $recentHistory | Measure-Object -Property rate -Minimum -Maximum
                if ($recentStats.Minimum -gt 0) { $volatility12h = (($recentStats.Maximum - $recentStats.Minimum) / $recentStats.Minimum) * 100 }
            }
            $volatility12h_old = 0
            if ($oldHistory -and $oldHistory.Count -gt 0) {
                $oldStats = $oldHistory | Measure-Object -Property rate -Minimum -Maximum
                if ($oldStats.Minimum -gt 0) { $volatility12h_old = (($oldStats.Maximum - $oldStats.Minimum) / $oldStats.Minimum) * 100 }
            }

            # 1H SMA Calculation
            $sma1h = 0
            $smaPoints = 12 # ~1 hour of data (12 * 5 mins)
            $sortedHistory = $historicalResponse.history | Sort-Object -Property date
            if ($sortedHistory.Count -gt 0) {
                $smaHistory = $sortedHistory | Select-Object -Last $smaPoints
                if ($smaHistory.Count -gt 0) {
                    $sma1h = ($smaHistory | Measure-Object -Property rate -Average).Average
                }
            }

            $result = [PSCustomObject]@{
                High              = $highPoint24h.rate
                Low               = $lowPoint24h.rate
                Ago               = $closestDataPoint.rate
                HighTime          = ([datetime]'1970-01-01').AddMilliseconds($highPoint24h.date)
                LowTime           = ([datetime]'1970-01-01').AddMilliseconds($lowPoint24h.date)
                Volatility        = $volatility24h
                Volatility12h     = $volatility12h
                Volatility12h_old = $volatility12h_old
                Sma1h             = $sma1h
                History           = $historicalResponse.history
            }
            return $result
        }
        else {
            Write-Warning "No historical data returned."
            Write-Verbose "historicalResponse.history is null or empty"
        }
    }
    catch {
        if ($_.Exception -is [System.Net.WebException]) {
            $errorCode = $null
            if ($_.Exception.Response) {
                $errorCode = [int]$_.Exception.Response.StatusCode
            }
            return [PSCustomObject]@{ IsNetworkError = $true; ErrorCode = $errorCode }
        }
        Write-Warning "Failed to fetch historical price: $($_.Exception.Message)"
    }
    return $null
}

# --- Helper Function to Generate Sparkline (24 characters for 24 hourly samples) ---
function Get-Sparkline {
    param ([System.Collections.Generic.List[double]]$History)
    
    if ($null -eq $History -or $History.Count -lt 2) { 
        return (" " * 24)
    }

    $sparkChars = [char[]]('▁', '▂', '▃', '▄', '▅', '▆', '▇', '█')
    $minPrice = ($History | Measure-Object -Minimum).Minimum
    $maxPrice = ($History | Measure-Object -Maximum).Maximum
    $priceRange = $maxPrice - $minPrice

    if ($priceRange -eq 0 -or $priceRange -lt 0.00000001) { 
        return ([string]$sparkChars[0] * 24)
    }

    $sparkline = ""
    foreach ($price in $History) {
        $normalized = ($price - $minPrice) / $priceRange
        $charIndex = [math]::Floor($normalized * ($sparkChars.Length - 1))
        $sparkline += $sparkChars[$charIndex]
    }
    
    # Truncate to 24 if too long (keep rightmost 24)
    if ($sparkline.Length -gt 24) {
        $sparkline = $sparkline.Substring($sparkline.Length - 24)
    }
    
    # Pad to 24 if too short (pad left with spaces)
    if ($sparkline.Length -lt 24) {
        $sparkline = $sparkline.PadLeft(24, ' ')
    }
    
    # Final check - must be exactly 24
    if ($sparkline.Length -ne 24) {
        $sparkline = $sparkline.Substring(0, [Math]::Min(24, $sparkline.Length)).PadRight(24, ' ')
    }
    
    return $sparkline
}

# --- Configuration File Path ---
# Determine script directory - prioritize most reliable methods
if ($PSCommandPath) {
    # PowerShell 3.0+ - most reliable, always absolute path
    $scriptPath = [System.IO.Path]::GetDirectoryName((Resolve-Path $PSCommandPath).Path)
    Write-Verbose "Using PSCommandPath: $scriptPath (resolved from: $PSCommandPath)"
} elseif (-not [string]::IsNullOrEmpty($PSScriptRoot)) {
    # Fallback to PSScriptRoot if available
    $scriptPath = $PSScriptRoot
    Write-Verbose "Using PSScriptRoot: $scriptPath"
} elseif ($MyInvocation.MyCommand.Path) {
    # Older PowerShell versions
    $resolvedPath = Resolve-Path $MyInvocation.MyCommand.Path -ErrorAction SilentlyContinue
    if ($resolvedPath) {
        $scriptPath = [System.IO.Path]::GetDirectoryName($resolvedPath.Path)
    } else {
        $scriptPath = [System.IO.Path]::GetDirectoryName((Resolve-Path (Join-Path $PWD.Path $MyInvocation.MyCommand.Path) -ErrorAction SilentlyContinue).Path)
        if ([string]::IsNullOrEmpty($scriptPath)) {
            $scriptPath = Split-Path -Parent -Path $MyInvocation.MyCommand.Path
        }
    }
    Write-Verbose "Using MyInvocation.MyCommand.Path: $scriptPath (resolved from: $($MyInvocation.MyCommand.Path))"
} else {
    # Last resort - use current working directory
    $scriptPath = $PWD.Path
    Write-Verbose "Using current working directory: $scriptPath"
}
$iniFilePath = Join-Path -Path $scriptPath -ChildPath "btc.ini"
Write-Verbose "INI configuration file path set to: $iniFilePath"

# --- Initial Load of Configuration ---
Write-Verbose "Attempting to load configuration..."
$config = Get-IniConfiguration -FilePath $iniFilePath
if ($config.Settings.ApiKey) { Write-Verbose "Initial configuration loaded."}

# --- Handle -Update Parameter ---
if ($Update.IsPresent) {
    Write-Host "--- Update Configuration ---" -ForegroundColor Yellow 
    $currentMyBtc = $config.Portfolio.MyBTC; $newMyBtcInput = Read-Host "MyBTC Amount (current: $currentMyBtc, press Enter to keep)"
    if (-not [string]::IsNullOrEmpty($newMyBtcInput)) { try { $parsedVal = [double]::Parse($newMyBtcInput, [System.Globalization.CultureInfo]::InvariantCulture); if ($parsedVal -ge 0) { $config.Portfolio.MyBTC = $parsedVal.ToString("F8", [System.Globalization.CultureInfo]::InvariantCulture) } else { Write-Warning "BTC Amount must be non-negative." } } catch { Write-Warning "Invalid MyBTC input." } }
    $currentMyCost = $config.Portfolio.MyCOST; $newMyCostInput = Read-Host "Total Cost (USD) for MyBTC (current: $currentMyCost, press Enter to keep)"
    if (-not [string]::IsNullOrEmpty($newMyCostInput)) { try { $parsedVal = [double]::Parse($newMyCostInput, [System.Globalization.CultureInfo]::InvariantCulture); if ($parsedVal -ge 0) { $config.Portfolio.MyCOST = $parsedVal.ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture) } else { Write-Warning "Total Cost must be non-negative." } } catch { Write-Warning "Invalid MyCOST input." } }
    $currentLogPath = $config.Settings.LogPath; $newLogPathInput = Read-Host "Log File Path (current: '$currentLogPath', press Enter to keep)"
    if (-not [string]::IsNullOrEmpty($newLogPathInput)) { $config.Settings.LogPath = $newLogPathInput }
    else {
        $disableLogging = Read-Host "Disable Logging? [N/y]"
        if ($disableLogging -match "^[Yy]([Ee][Ss])?$") { $config.Settings.LogPath = "" }
    }
    Set-IniConfiguration -FilePath $iniFilePath -Configuration $config 
    Write-Host "Configuration updated." -ForegroundColor Green 
	exit 0
}

# --- First-Run Setup (if API Key is missing) ---
if ([string]::IsNullOrEmpty($config.Settings.ApiKey)) {
    Write-Host "--- First Time Setup / API Key Missing ---" -ForegroundColor Yellow 
    $newApiKey = ""; while ([string]::IsNullOrEmpty($newApiKey)) { $newApiKey = Read-Host "Please enter your LiveCoinWatch API Key (required)"; if ([string]::IsNullOrEmpty($newApiKey)) { Write-Warning "API Key cannot be empty." } }
    $config.Settings.ApiKey = $newApiKey; $newMyBtcInput = Read-Host "Enter your BTC Amount (e.g., 0.5, Enter to skip)"; $validBtcEntered = $false
    if (-not [string]::IsNullOrEmpty($newMyBtcInput)) { try { $parsedMyBtc = [double]::Parse($newMyBtcInput, [System.Globalization.CultureInfo]::InvariantCulture); if ($parsedMyBtc -ge 0) { $config.Portfolio.MyBTC = $parsedMyBtc.ToString("F8", [System.Globalization.CultureInfo]::InvariantCulture); $validBtcEntered = $true } else { Write-Warning "BTC Amount must be non-negative." } } catch { Write-Warning "Invalid BTC Amount." } }
    if ($validBtcEntered) { $newMyCostInput = Read-Host "Enter Total USD Cost for this BTC (e.g., 5000.00, Enter to skip)"; if (-not [string]::IsNullOrEmpty($newMyCostInput)) { try { $parsedMyCost = [double]::Parse($newMyCostInput, [System.Globalization.CultureInfo]::InvariantCulture); if ($parsedMyCost -ge 0) { $config.Portfolio.MyCOST = $parsedMyCost.ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture) } else { Write-Warning "MyCOST must be non-negative." } } catch { Write-Warning "Invalid MyCOST." } } else { $config.Portfolio.MyCOST = "0.0" } } else { $config.Portfolio.MyBTC = "0.0"; $config.Portfolio.MyCOST = "0.0" }
    $newLogPathInput = Read-Host "Enter Log File Path (e.g., btc_log.csv, Enter for blank/disabled)"; $config.Settings.LogPath = $newLogPathInput
    Set-IniConfiguration -FilePath $iniFilePath -Configuration $config 
}

# --- Final Configuration Values for Current Run ---
$apiKey = $config.Settings.ApiKey
if ([string]::IsNullOrEmpty($apiKey)) { Write-Error "API Key is missing. Exiting."; exit 1 } else { Write-Verbose "API Key successfully retrieved for use."}

$effectiveLogPath = $null
if (-not [string]::IsNullOrEmpty($LogToFile)) { 
    $effectiveLogPath = $LogToFile
    Write-Verbose "Log path explicitly set from command line: $effectiveLogPath" 
} elseif ($config.Settings.ContainsKey('LogPath')) { 
    if (-not [string]::IsNullOrEmpty($config.Settings.LogPath)) { 
        $effectiveLogPath = Join-Path -Path $scriptPath -ChildPath $config.Settings.LogPath
        Write-Verbose "Log path set from btc.ini: $effectiveLogPath" 
    } else { 
        Write-Verbose "Logging disabled: LogPath is present but empty in btc.ini."
    } 
} else { 
    $effectiveLogPath = Join-Path -Path $scriptPath -ChildPath "btc_log.csv" 
    Write-Verbose "Log path key not found in btc.ini, defaulted to: $effectiveLogPath" 
}

$mybtc = $null
if ($PSBoundParameters.ContainsKey('UserBTCAmount')) { 
    if ($UserBTCAmount -is [double] -and $UserBTCAmount -ge 0) { $mybtc = $UserBTCAmount; Write-Verbose "MyBTC amount overridden by command line: $mybtc" } 
    else { Write-Warning "Invalid -UserBTCAmount provided. Ignoring command line value." } 
} elseif ($null -ne $config.Portfolio.MyBTC -and $null -ne ($config.Portfolio.MyBTC -as [double]) -and ($config.Portfolio.MyBTC -as [double]) -ge 0) { 
    $mybtc = $config.Portfolio.MyBTC -as [double]
    Write-Verbose "MyBTC amount loaded from btc.ini: $mybtc"
} else {
    Write-Verbose "MyBTC amount not set from command line or btc.ini (or value is invalid/zero)."
}

$myCOST = $null
if ($PSBoundParameters.ContainsKey('UserTotalCost')) { 
    if ($UserTotalCost -is [double] -and $UserTotalCost -ge 0) { $myCOST = $UserTotalCost; Write-Verbose "MyCOST overridden by command line: $myCOST" } 
    else { Write-Warning "Invalid -UserTotalCost provided. Ignoring command line value." } 
} elseif ($null -ne $config.Portfolio.MyCOST -and $null -ne ($config.Portfolio.MyCOST -as [double]) -and ($config.Portfolio.MyCOST -as [double]) -ge 0) { 
    $myCOST = $config.Portfolio.MyCOST -as [double]
    Write-Verbose "MyCOST loaded from btc.ini: $myCOST"
} else {
    Write-Verbose "MyCOST not set from command line or btc.ini (or value is invalid/zero)."
}

if ($null -ne $myCOST -and ($null -eq $mybtc -or $mybtc -eq 0)) { Write-Warning "MyCOST set but MyBTC is zero/not set. P/L skipped."; $myCOST = $null }

if ($null -ne $mybtc -and $mybtc -gt 0) { 
    Write-Verbose "Tracking value for $mybtc BTC."
    if ($null -ne $myCOST) {
        Write-Verbose "Total cost basis: $($myCOST.ToString("C2"))"
    }
}

# --- Other Script Constants ---
$coinCode = "BTC"; $currency = "USD"; $apiBaseUrl = "https://api.livecoinwatch.com"; $athProximityPercent = 0.05 

# --- Construct API Request Headers ---
$headers = @{ "Content-Type" = "application/json"; "x-api-key" = $apiKey }

# --- Make API Call for Current Data ---
Write-Verbose "Fetching current Bitcoin data from LiveCoinWatch..."
try {
    $currentDataBody = @{ currency = $currency; code = $coinCode; meta = $true } | ConvertTo-Json
    $currentResponse = Invoke-RestMethod -Uri "$($apiBaseUrl)/coins/single" -Method Post -Headers $headers -Body $currentDataBody -ErrorAction Stop
    Write-Verbose "Current Bitcoin data API call successful."

    if ($null -ne $currentResponse -and $currentResponse.PSObject.Properties.Name -contains "rate" -and $currentResponse.PSObject.Properties.Name -contains "allTimeHighUSD") {
        Write-Verbose "Successfully parsed current Bitcoin data."
        $bitcoinPrice = $currentResponse.rate
        $volume24h = $currentResponse.volume
        $marketCap = $currentResponse.cap
        $priceChange24hPercent_BTC_API = $currentResponse.delta.day 
        $allTimeHighUSD = $currentResponse.allTimeHighUSD
        $liquidity = $currentResponse.liquidity; $totalSupply = $currentResponse.totalSupply
        $deltaHourPct = $currentResponse.delta.hour; $deltaWeekPct = $currentResponse.delta.week
        $deltaMonthPct = $currentResponse.delta.month; $deltaYearPct = $currentResponse.delta.year

        # --- Make API Call for Historical Data (Full 24 hours) ---
        $actualBitcoinPrice24hAgo = $null; $priceDifference24h = $null
        $historicalStats = Get-HistoricalData -ApiKey $apiKey -Currency $currency -CoinCode $coinCode
        
        if ($null -ne $historicalStats -and $historicalStats.PSObject.Properties.Name -contains 'IsNetworkError' -and $historicalStats.IsNetworkError) {
            Write-Warning "Historical data fetch failed with network error. Using fallback calculation."
            Write-Verbose "Network error detected, ErrorCode: $($historicalStats.ErrorCode)"
            if ($null -ne $priceChange24hPercent_BTC_API) {
                $priceChangeDecimal = $priceChange24hPercent_BTC_API / 100
                if ((1 + $priceChangeDecimal) -ne 0) { $estimatedPrice24hAgo = $bitcoinPrice / (1 + $priceChangeDecimal); $priceDifference24h = $bitcoinPrice - $estimatedPrice24hAgo }
            }
        } elseif ($null -ne $historicalStats) {
            Write-Verbose "Historical data fetch successful. Extracting 24h ago price..."
            $actualBitcoinPrice24hAgo = $historicalStats.Ago
            if ($null -ne $actualBitcoinPrice24hAgo) {
                $priceDifference24h = $bitcoinPrice - $actualBitcoinPrice24hAgo
                Write-Verbose "Historical price found: $($actualBitcoinPrice24hAgo.ToString("C2")), difference: $($priceDifference24h.ToString("C2"))"
            } else {
                Write-Verbose "HistoricalStats.Ago is null"
            }
        } else {
            Write-Warning "Could not fetch historical data. Using estimated 24h difference based on API delta."
            Write-Verbose "historicalStats is null, using fallback calculation"
            if ($null -ne $priceChange24hPercent_BTC_API) {
                $priceChangeDecimal = $priceChange24hPercent_BTC_API / 100
                if ((1 + $priceChangeDecimal) -ne 0) { $estimatedPrice24hAgo = $bitcoinPrice / (1 + $priceChangeDecimal); $priceDifference24h = $bitcoinPrice - $estimatedPrice24hAgo }
            }
        }
        
        $bitcoinLineColorNameToUse = $null
        if ($bitcoinPrice -ge ($allTimeHighUSD * (1 - $athProximityPercent))) { $bitcoinLineColorNameToUse = "Magenta" }

        # --- Display Data (Main output, should not be verbose) ---
        $btcLineChangeIndicator = if ($null -ne $priceDifference24h) { $priceDifference24h } else { $priceChange24hPercent_BTC_API }
        Write-ColoredLine -Label "Bitcoin ($($currency)): " -Prefix "$" -Value $bitcoinPrice -FormatString "N2" -ChangeIndicator $btcLineChangeIndicator -PriceChangeAmountToDisplay $priceDifference24h -ExplicitColorName $bitcoinLineColorNameToUse
        $mybtcValue = $null; $mybtcValueDifference24h = $null; $profitLossUSD = $null; $profitLossPercent = $null
        if ($null -ne $mybtc -and $mybtc -gt 0) {
            $mybtcValue = $mybtc * $bitcoinPrice
            if ($null -ne $priceDifference24h) { $mybtcValueDifference24h = $mybtc * $priceDifference24h }
            $myBtcLineChangeIndicator = if ($null -ne $priceDifference24h) { $priceDifference24h } else { $priceChange24hPercent_BTC_API }
            Write-ColoredLine -Label "My BTC: " -Prefix "$" -Value $mybtcValue -FormatString "N2" -ChangeIndicator $myBtcLineChangeIndicator -PriceChangeAmountToDisplay $mybtcValueDifference24h
            if ($null -ne $myCOST -and $myCOST -gt 0) {
                $profitLossUSD = $mybtcValue - $myCOST
                if ($myCOST -ne 0) { $profitLossPercent = ($profitLossUSD / $myCOST) * 100 }
                $plColorIndicator = if ($profitLossUSD -eq 0) { 0 } elseif ($profitLossUSD -gt 0) { 1 } else { -1 }
                Write-ColoredLine -Label "Profit/Loss: " -Prefix "$" -Value $profitLossUSD -FormatString "N2" -ChangeIndicator $plColorIndicator -PercentageChangeToDisplay $profitLossPercent -PercentageLabel "%"
            }
        }
        
        # --- Display Additional Market Metrics ---
        $hasNetworkError = $false
        if ($null -ne $historicalStats -and $historicalStats.PSObject.Properties.Name -contains 'IsNetworkError') {
            $hasNetworkError = $historicalStats.IsNetworkError
        }
        if ($null -ne $historicalStats -and -not $hasNetworkError) {
            # 1H SMA
            if ($historicalStats.PSObject.Properties.Name -contains 'Sma1h' -and $historicalStats.Sma1h -gt 0) {
                $smaColor = "White"
                if ($bitcoinPrice -gt $historicalStats.Sma1h) {
                    $smaColor = "Green"
                } elseif ($bitcoinPrice -lt $historicalStats.Sma1h) {
                    $smaColor = "Red"
                }
                Write-Host -NoNewline -ForegroundColor White "1H SMA: "
                $paddingRequired = 15 - "1H SMA: ".Length
                if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                Write-Host -ForegroundColor $smaColor ("{0:C2}" -f $historicalStats.Sma1h)
            }
            
            # 24H Ago
            if ($historicalStats.PSObject.Properties.Name -contains 'Ago' -and $null -ne $historicalStats.Ago) {
                $rate24hAgo = $historicalStats.Ago
                $percentChange24h = if ($rate24hAgo -ne 0) { (($bitcoinPrice - $rate24hAgo) / $rate24hAgo) * 100 } else { 0 }
                $roundedCurrent = [math]::Round([decimal]$bitcoinPrice, 2)
                $rounded24hAgo = [math]::Round([decimal]$rate24hAgo, 2)
                $priceColor24h = if ($roundedCurrent -gt $rounded24hAgo) { "Green" } elseif ($roundedCurrent -lt $rounded24hAgo) { "Red" } else { "White" }
                $agoDisplay = "{0:C2} [{1}{2}%]" -f $rate24hAgo, $(if($roundedCurrent -gt $rounded24hAgo){"+"}), ("{0:N2}" -f $percentChange24h)
                Write-Host -NoNewline -ForegroundColor White "24H Ago: "
                $paddingRequired = 15 - "24H Ago: ".Length
                if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                Write-Host -ForegroundColor $priceColor24h $agoDisplay
            }
            
            # 24H High
            if ($historicalStats.PSObject.Properties.Name -contains 'High' -and $null -ne $historicalStats.High) {
                $highDisplay = "{0:C2}" -f $historicalStats.High
                if ($historicalStats.PSObject.Properties.Name -contains 'HighTime' -and $historicalStats.HighTime) {
                    $highDisplay += " (at $($historicalStats.HighTime.ToLocalTime().ToString("HH:mm")))"
                }
                Write-Host -NoNewline -ForegroundColor White "24H High: "
                $paddingRequired = 15 - "24H High: ".Length
                if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                Write-Host $highDisplay
            }
            
            # 24H Low
            if ($historicalStats.PSObject.Properties.Name -contains 'Low' -and $null -ne $historicalStats.Low) {
                $lowDisplay = "{0:C2}" -f $historicalStats.Low
                if ($historicalStats.PSObject.Properties.Name -contains 'LowTime' -and $historicalStats.LowTime) {
                    $lowDisplay += " (at $($historicalStats.LowTime.ToLocalTime().ToString("HH:mm")))"
                }
                Write-Host -NoNewline -ForegroundColor White "24H Low: "
                $paddingRequired = 15 - "24H Low: ".Length
                if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                Write-Host $lowDisplay
            }
            
            # Volatility
            if ($historicalStats.PSObject.Properties.Name -contains 'Volatility' -and $historicalStats.Volatility -gt 0) {
                $volatilityColor = "White"
                if ($historicalStats.PSObject.Properties.Name -contains 'Volatility12h' -and $historicalStats.PSObject.Properties.Name -contains 'Volatility12h_old') {
                    if ($historicalStats.Volatility12h -gt $historicalStats.Volatility12h_old) {
                        $volatilityColor = "Green"
                    } elseif ($historicalStats.Volatility12h -lt $historicalStats.Volatility12h_old) {
                        $volatilityColor = "Red"
                    }
                }
                $volatilityDisplay = "{0:N2}%" -f $historicalStats.Volatility
                Write-Host -NoNewline -ForegroundColor White "Volatility: "
                $paddingRequired = 15 - "Volatility: ".Length
                if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                Write-Host -ForegroundColor $volatilityColor $volatilityDisplay
            }
        }
        
        $apiVolumeChange24hPercent = $currentResponse.delta.volumeDay
        $apiCapChange24hPercent = $currentResponse.delta.capDay
        Write-ColoredLine -Label "24H Volume: " -Prefix "$" -Value $volume24h -FormatString "N0" -ChangeIndicator $priceChange24hPercent_BTC_API -PercentageChangeToDisplay $apiVolumeChange24hPercent
        Write-ColoredLine -Label "Cap: " -Prefix "$" -Value $marketCap -FormatString "N0" -ChangeIndicator $priceChange24hPercent_BTC_API -PercentageChangeToDisplay $apiCapChange24hPercent
        
        # --- History Sparkline ---
        $hasNetworkErrorForSparkline = $false
        if ($null -ne $historicalStats -and $historicalStats.PSObject.Properties.Name -contains 'IsNetworkError') {
            $hasNetworkErrorForSparkline = $historicalStats.IsNetworkError
        }
        if ($null -ne $historicalStats -and -not $hasNetworkErrorForSparkline -and $historicalStats.PSObject.Properties.Name -contains 'History') {
            $historyData = $historicalStats.History
            if ($historyData -and $historyData.Count -gt 0) {
                # Sample hourly data points (24 points for 24h)
                $hourlySamples = [System.Collections.Generic.List[double]]::new()
                $sortedHistory = $historyData | Sort-Object -Property date
                $nowMs = [int64](([datetime]::UtcNow) - (Get-Date "1970-01-01")).TotalMilliseconds
                
                # Get 24 hourly samples (one per hour, going back 24 hours)
                for ($i = 23; $i -ge 0; $i--) {
                    $targetHourMs = $nowMs - ($i * 60 * 60 * 1000)
                    $closestPoint = $sortedHistory | Sort-Object { [Math]::Abs($_.date - $targetHourMs) } | Select-Object -First 1
                    if ($closestPoint) {
                        $hourlySamples.Add($closestPoint.rate)
                    }
                }
                
                # If we don't have enough hourly samples, use all available data points
                if ($hourlySamples.Count -lt 2) {
                    foreach ($point in $sortedHistory) {
                        $hourlySamples.Add($point.rate)
                    }
                }
                
                if ($hourlySamples.Count -ge 2) {
                    $sparkline = Get-Sparkline -History $hourlySamples
                    Write-Host -NoNewline -ForegroundColor White "History: "
                    $paddingRequired = 15 - "History: ".Length
                    if ($paddingRequired -gt 0) { Write-Host -NoNewline (" " * $paddingRequired) }
                    
                    # Color-code sparkline characters based on comparison with previous character
                    $sparkChars = [char[]]('▁', '▂', '▃', '▄', '▅', '▆', '▇', '█')
                    $prevIndex = $null
                    for ($i = 0; $i -lt $sparkline.Length; $i++) {
                        $char = $sparkline[$i]
                        if ($char -eq ' ') {
                            Write-Host -NoNewline $char
                            continue
                        }
                        $currentIndex = [Array]::IndexOf($sparkChars, $char)
                        if ($null -eq $prevIndex) {
                            $color = [System.ConsoleColor]::White
                        } elseif ($prevIndex -gt $currentIndex) {
                            $color = [System.ConsoleColor]::Red  # Previous was taller (price down)
                        } elseif ($prevIndex -lt $currentIndex) {
                            $color = [System.ConsoleColor]::Green  # Previous was shorter (price up)
                        } else {
                            $color = [System.ConsoleColor]::White  # Same height
                        }
                        Write-Host -NoNewline -ForegroundColor $color $char
                        $prevIndex = $currentIndex
                    }
                    Write-Host ""
                }
            }
        }

        # --- Logging ---
        if (-not [string]::IsNullOrEmpty($effectiveLogPath)) {
            Write-Verbose "Preparing to log data to: $effectiveLogPath"
            $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            $logColumnsOrder = @("Timestamp", "MyBTC", "MyCOST_USD", "ProfitLoss_USD", "ProfitLoss_Percent", "Rate_USD", "Volume24h_USD", "Cap_USD", "Liquidity_USD", "DeltaHour_Pct", "DeltaDay_Pct", "DeltaWeek_Pct", "DeltaMonth_Pct", "DeltaYear_Pct", "TotalSupply", "AllTimeHigh_USD")
            $logData = @{ Timestamp = $timestamp; MyBTC = if ($null -ne $mybtc) { $mybtc } else { "" }; MyCOST_USD = if ($null -ne $myCOST) { $myCOST } else { "" }; ProfitLoss_USD = if ($null -ne $profitLossUSD) { $profitLossUSD } else { "" }; ProfitLoss_Percent = if ($null -ne $profitLossPercent) { $profitLossPercent } else { "" }; Rate_USD = $bitcoinPrice; Volume24h_USD = $volume24h; Cap_USD = $marketCap; Liquidity_USD = if ($null -ne $liquidity) { $liquidity } else { "" }; DeltaHour_Pct = if ($null -ne $deltaHourPct) { $deltaHourPct } else { "" }; DeltaDay_Pct = if ($null -ne $priceChange24hPercent_BTC_API) { $priceChange24hPercent_BTC_API } else { "" }; DeltaWeek_Pct = if ($null -ne $deltaWeekPct) { $deltaWeekPct } else { "" }; DeltaMonth_Pct = if ($null -ne $deltaMonthPct) { $deltaMonthPct } else { "" }; DeltaYear_Pct = if ($null -ne $deltaYearPct) { $deltaYearPct } else { "" }; TotalSupply = if ($null -ne $totalSupply) { $totalSupply } else { "" }; AllTimeHigh_USD = if ($null -ne $allTimeHighUSD) { $allTimeHighUSD } else { "" } }
            $logHeader = '"Timestamp","MyBTC","MyCOST_USD","ProfitLoss_USD","ProfitLoss_Percent","Rate_USD","Volume24h_USD","Cap_USD","Liquidity_USD","DeltaHour_Pct","DeltaDay_Pct","DeltaWeek_Pct","DeltaMonth_Pct","DeltaYear_Pct","TotalSupply","AllTimeHigh_USD"'
            $logLineValues = foreach ($colName in $logColumnsOrder) { $value = $logData[$colName]; if ($null -eq $value) { $value = "" }; '"{0}"' -f ($value -replace '"', '""') }
            $logLine = $logLineValues -join ','
            if (-not (Test-Path $effectiveLogPath)) {
                try { New-Item -Path $effectiveLogPath -ItemType File -Force -ErrorAction Stop | Out-Null; Add-Content -Path $effectiveLogPath -Value $logHeader -ErrorAction Stop; Write-Verbose "Log file created: $effectiveLogPath" } 
                catch { Write-Warning "Could not create log file header at $effectiveLogPath. Error: $($_.Exception.Message)" }
            }
            try { Add-Content -Path $effectiveLogPath -Value $logLine -ErrorAction Stop; Write-Verbose "Data logged successfully." }
            catch { Write-Warning "Could not write to log file $effectiveLogPath. Error: $($_.Exception.Message)" }
        } else { Write-Verbose "Logging is disabled (no effective log path)." }
    } else { Write-Error "Failed to retrieve complete Bitcoin data from API."; Write-Host "Response:"; Write-Host ($currentResponse | ConvertTo-Json -Depth 5) }
}
catch { Write-Error "An error occurred: $($_.Exception.Message)"; if ($_.Exception.Response) { try { $es = $_.Exception.Response.GetResponseStream; $sr = New-Object System.IO.StreamReader($es); $eb = $sr.ReadToEnd(); $sr.Close(); $es.Close(); Write-Error "API Error: $eb" } catch { Write-Warning "Could not read API error." } } }
# --- End of Script ---
