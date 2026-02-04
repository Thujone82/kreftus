<#
.SYNOPSIS
    Virtual Bitcoin Trader (vBTC) - Interactive Bitcoin trading simulation

.DESCRIPTION
    Use -help or -? to display the help screen; use -config to open configuration (e.g. to fix API key) and exit.

.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-07-28
    Version: 1.5
#>

[CmdletBinding()]
param (
    [switch]$Help,
    [Alias("Config")]
    [switch]$OpenConfig   # -OpenConfig / -Config: open config menu and exit (avoids colliding with $config hashtable)
)

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
$script:LastGoodApiData = $null

# Check for help parameters (before any other processing)
if ($Help.IsPresent) {
    # Define the help function inline to avoid dependency issues
    function Show-HelpScreen {
        Clear-Host
        Write-Host "Virtual Bitcoin Trader (vBTC) - Version 1.5" -ForegroundColor Yellow
        Write-Host "===============================================================" -ForegroundColor DarkGray
        Write-Host ""
        
        Write-Host "COMMANDS:" -ForegroundColor Cyan
        Write-Host "    buy [amount]     " -NoNewline -ForegroundColor White; Write-Host "Purchase a specific USD amount of Bitcoin" -ForegroundColor Gray
        Write-Host "    sell [amount]    " -NoNewline -ForegroundColor White; Write-Host "Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s)" -ForegroundColor Gray
        Write-Host "    ledger           " -NoNewline -ForegroundColor White; Write-Host "View a history of all your transactions" -ForegroundColor Gray
        Write-Host "    refresh          " -NoNewline -ForegroundColor White; Write-Host "Manually update the market data" -ForegroundColor Gray
        Write-Host "    config           " -NoNewline -ForegroundColor White; Write-Host "Access the configuration menu" -ForegroundColor Gray
        Write-Host "    help             " -NoNewline -ForegroundColor White; Write-Host "Show this help screen" -ForegroundColor Gray
        Write-Host "    exit             " -NoNewline -ForegroundColor White; Write-Host "Exit the application" -ForegroundColor Gray
        Write-Host ""
        
        Write-Host "TIPS:" -ForegroundColor Green
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Commands may be shortened (e.g. 'b 10' to buy $10 of BTC)" -ForegroundColor Gray
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Use 'p' for percentage trades (e.g., '50p' for 50%, '100/3p' for 33.3%)" -ForegroundColor Gray
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Volatility shows the price swing (High vs Low) over the last 24 hours" -ForegroundColor Gray
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "1H SMA is the average price over the last hour. Green = price is above average" -ForegroundColor Gray
        Write-Host ""
        
        Write-Host "REQUIREMENTS:" -ForegroundColor Blue
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "PowerShell" -ForegroundColor Gray
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "An internet connection" -ForegroundColor Gray
        Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "A free API key from https://www.livecoinwatch.com/tools/api" -ForegroundColor Gray
        Write-Host ""
        Write-Host "COMMAND LINE:" -ForegroundColor Cyan
        Write-Host "    -help, -?     " -NoNewline -ForegroundColor White; Write-Host "Show this help and exit" -ForegroundColor Gray
        Write-Host "    -config      " -NoNewline -ForegroundColor White; Write-Host "Open configuration (e.g. to fix API key) and exit" -ForegroundColor Gray
        Write-Host ""
        Write-Host "===============================================================" -ForegroundColor DarkGray
        Write-Host ""
    }
    Show-HelpScreen
    exit 0
}

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
        return $true
    }
    catch {
        Write-Error "Failed to save configuration to $FilePath. Error: $($_.Exception.Message)"
        return $false
    }
}

function Format-ProfitLoss {
    param(
        [double]$Value,
        [string]$FormatString, # e.g., "C2" for currency, "N2" for number
        [string]$Suffix = ""
    )
    if ($Value -lt 0) {
        return "({0}{1})" -f (([Math]::Abs($Value)).ToString($FormatString)), $Suffix
    } else {
        return "+{0}{1}" -f ($Value.ToString($FormatString)), $Suffix
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
    $endTimestampMs = [int64](([datetime]::UtcNow) - (Get-Date "1970-01-01")).TotalMilliseconds
    $startTimestampMs = $endTimestampMs - (24 * 60 * 60 * 1000) # Full 24 hours

    try {
        $historicalBody = @{ currency = "USD"; code = "BTC"; start = $startTimestampMs; end = $endTimestampMs; meta = $false } | ConvertTo-Json
        Write-Verbose "Fetching historical price for 24h ago..."
        $historicalResponse = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single/history" -Method Post -Headers $headers -Body $historicalBody -ErrorAction Stop
        if ($null -ne $historicalResponse.history -and $historicalResponse.history.Count -gt 0) {
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

            return [PSCustomObject]@{
                High              = $highPoint24h.rate
                Low               = $lowPoint24h.rate
                Ago      = $closestDataPoint.rate
                HighTime          = ([datetime]'1970-01-01').AddMilliseconds($highPoint24h.date)
                LowTime           = ([datetime]'1970-01-01').AddMilliseconds($lowPoint24h.date)
                Volatility        = $volatility24h
                Volatility12h     = $volatility12h
                Volatility12h_old = $volatility12h_old
                Sma1h             = $sma1h
            }
        }
        else {
            Write-Warning "No historical data returned."
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

function Copy-HistoricalData {
    param ([PSCustomObject]$Source, [PSCustomObject]$Destination)
    if (-not $Source -or -not $Destination) { return }
    
    $propertiesToCopy = @(
        "rate24hHigh",
        "rate24hLow",
        "rate24hHighTime",
        "rate24hLowTime",
        "rate24hAgo",
        "volatility24h",
        "volatility12h",
        "volatility12h_old",
        "sma1h",
        "HistoricalDataFetchTime"
    )

    foreach ($prop in $propertiesToCopy) {
        if ($Source.PSObject.Properties[$prop]) {
            $Destination | Add-Member -MemberType NoteProperty -Name $prop -Value $Source.$prop -Force
        }
    }
}

function Update-ApiData {
    param ([hashtable]$Config, [PSCustomObject]$OldApiData, [switch]$SkipHistorical)
    Show-LoadingScreen
    $newData = Get-ApiData -Config $Config
    $is403 = $newData -and (($newData.PSObject.Properties['IsForbidden'] -and $newData.IsForbidden) -or ($newData.PSObject.Properties['ErrorCode'] -and $newData.ErrorCode -eq 403))
    if ($is403 -and -not $OldApiData) {
        Write-Host "403 Encountered: Ensure API Key Configured and Enabled" -ForegroundColor Red
        Write-Host "Execute 'vbtc -config' to configure" -ForegroundColor Yellow
        exit 1
    }
    if ($is403 -and $OldApiData) {
        Write-Warning "403 on API request. Continuing with cached data."
        return $OldApiData
    }
    if ($newData -and $newData.PSObject.Properties['IsNetworkError']) {
        return $newData # Propagate the network error object up
    }
    if (-not $newData) {
        Write-Warning "Failed to fetch current price data. Returning last known data."
        return $OldApiData
    }

    if (-not $SkipHistorical.IsPresent) {
        $isStale = $false
        if (-not $OldApiData) {
            $isStale = $true
        } else {
            # Check if cache is expired by time
            if ((-not $OldApiData.PSObject.Properties['HistoricalDataFetchTime']) -or (((Get-Date).ToUniversalTime() - $OldApiData.HistoricalDataFetchTime).TotalMinutes -gt 15)) {
                $isStale = $true
            } elseif ($OldApiData.PSObject.Properties['rate24hHigh'] -and $OldApiData.PSObject.Properties['rate24hLow'] -and (($newData.rate -gt $OldApiData.rate24hHigh) -or ($newData.rate -lt $OldApiData.rate24hLow))) {
                # Also mark as stale if the current price breaks the known 24h high/low.
                $isStale = $true
            }
        }

        if ($isStale) {
            Write-Host "Fetching updated historical data..." -ForegroundColor Yellow
            Start-Sleep -Seconds 1 # Let user see the message

            $historicalStats = Get-HistoricalData -Config $Config
            if ($historicalStats -and $historicalStats.PSObject.Properties['IsNetworkError']) {
                # Historical data failed with a network error.
                # We have current data, but we should still show the error message.
                # Let's add the error flag to the newData object.
                $newData | Add-Member -MemberType NoteProperty -Name "IsNetworkError" -Value $true -Force
                $newData | Add-Member -MemberType NoteProperty -Name "ErrorCode" -Value $historicalStats.ErrorCode -Force
                Write-Warning "Could not fetch historical data due to a network error. Using fallbacks."
                if ($OldApiData) {
                    Copy-HistoricalData -Source $OldApiData -Destination $newData
                }
                # The fallback logic below will handle the rest
            }
            elseif ($historicalStats) {
                $newData | Add-Member -MemberType NoteProperty -Name "rate24hHigh" -Value $historicalStats.High -Force
                $newData | Add-Member -MemberType NoteProperty -Name "rate24hLow" -Value $historicalStats.Low -Force
                $newData | Add-Member -MemberType NoteProperty -Name "rate24hHighTime" -Value $historicalStats.HighTime -Force
                $newData | Add-Member -MemberType NoteProperty -Name "rate24hLowTime" -Value $historicalStats.LowTime -Force
                $newData | Add-Member -MemberType NoteProperty -Name "rate24hAgo" -Value $historicalStats.Ago -Force
                $newData | Add-Member -MemberType NoteProperty -Name "volatility24h" -Value $historicalStats.Volatility -Force
                $newData | Add-Member -MemberType NoteProperty -Name "volatility12h" -Value $historicalStats.Volatility12h -Force
                $newData | Add-Member -MemberType NoteProperty -Name "volatility12h_old" -Value $historicalStats.Volatility12h_old -Force
                $newData | Add-Member -MemberType NoteProperty -Name "sma1h" -Value $historicalStats.Sma1h -Force
                $newData | Add-Member -MemberType NoteProperty -Name "HistoricalDataFetchTime" -Value (Get-Date).ToUniversalTime() -Force
            } else {
                Write-Warning "Could not fetch historical data. Using fallbacks."
                if ($OldApiData) {
                    # If historical fetch fails, try to reuse the old historical data.
                    Copy-HistoricalData -Source $OldApiData -Destination $newData
                }
                else {
                    # No old data to fall back on, use the delta from the current price data.
                    $newData | Add-Member -MemberType NoteProperty -Name "rate24hHigh" -Value $newData.rate -Force
                    $newData | Add-Member -MemberType NoteProperty -Name "rate24hLow" -Value $newData.rate -Force
                    $newData | Add-Member -MemberType NoteProperty -Name "volatility24h" -Value 0 -Force
                    $newData | Add-Member -MemberType NoteProperty -Name "volatility12h" -Value 0 -Force
                    $newData | Add-Member -MemberType NoteProperty -Name "volatility12h_old" -Value 0 -Force
                    $newData | Add-Member -MemberType NoteProperty -Name "sma1h" -Value 0 -Force
                    $fallbackRate = if ($newData.PSObject.Properties['delta'] -and $newData.delta.PSObject.Properties['day'] -and $newData.delta.day -ne 0) {
                        $newData.rate / (1 + ($newData.delta.day / 100))
                    } else {
                        $newData.rate
                    }
                    $newData | Add-Member -MemberType NoteProperty -Name "rate24hAgo" -Value $fallbackRate -Force
                    }
            }
        } else {
            # Historical data is fresh, just copy it over.
            Copy-HistoricalData -Source $OldApiData -Destination $newData
        }
    } else { # SkipHistorical is true
        # If skipping historical, just copy the old data over.
        Copy-HistoricalData -Source $OldApiData -Destination $newData
    }

    # Update last-known good dataset when we have a valid rate
    if ($newData -and $newData.PSObject.Properties['rate'] -and $newData.rate) {
        $script:LastGoodApiData = $newData
    }
    return $newData
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
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
    try {
        Write-Verbose "Fetching main API data..."
        $currentResponse = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -ErrorAction Stop
        $currentResponse | Add-Member -MemberType NoteProperty -Name "fetchTime" -Value (Get-Date).ToUniversalTime()
        Write-Verbose "Main API call successful."
        return $currentResponse
    }
    catch {
        $errorCode = $null
        if ($_.Exception -is [System.Net.WebException]) {
            if ($_.Exception.Response) {
                $errorCode = [int]$_.Exception.Response.StatusCode
            }
            # 403 on launch: exit gracefully with user message (handled in Update-ApiData)
            if ($errorCode -eq 403) {
                return [PSCustomObject]@{ IsForbidden = $true; ErrorCode = 403 }
            }
            return [PSCustomObject]@{ IsNetworkError = $true; ErrorCode = $errorCode }
        }
        # PowerShell 7 / HttpClient: Response may be HttpResponseMessage (no GetResponseStream)
        if ($_.Exception.Response) {
            try {
                $errorCode = [int]$_.Exception.Response.StatusCode
            } catch {
                $errorCode = $null
            }
            if ($errorCode -eq 403) {
                return [PSCustomObject]@{ IsForbidden = $true; ErrorCode = 403 }
            }
        }
        # For other errors (e.g. bad API key), log and optionally read body
        Write-Error "API call failed: $($_.Exception.Message)"
        if ($_.Exception.Response) {
            $response = $_.Exception.Response
            try {
                if (Get-Member -InputObject $response -Name 'GetResponseStream' -MemberType Method -ErrorAction SilentlyContinue) {
                    $errorStream = $response.GetResponseStream()
                    $reader = New-Object System.IO.StreamReader($errorStream)
                    $errorText = $reader.ReadToEnd()
                    $reader.Close()
                    Write-Error "API Response: $errorText"
                } elseif (Get-Member -InputObject $response -Name 'Content' -MemberType Property -ErrorAction SilentlyContinue) {
                    $errorText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
                    if ($errorText) { Write-Error "API Response: $errorText" }
                }
            } catch {
                # Ignore errors reading response body
            }
        }
        return $null
    }
}

function Get-BestApiData {
    param([object]$Preferred)
    if ($Preferred -and $Preferred.PSObject.Properties['rate'] -and $Preferred.rate) { return $Preferred }
    if ($script:LastGoodApiData) { return $script:LastGoodApiData }
    return $Preferred
}

function Test-ApiKey {
    param ([string]$ApiKey)
    if ([string]::IsNullOrEmpty($ApiKey)) { return $false }
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
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
    param ($ApiData, [hashtable]$Portfolio, [double]$SessionStartValue, [decimal]$InitialSessionBtcPrice)
    if (-not $VerbosePreference) { Clear-Host }
    
    $isNetworkError = $ApiData -and $ApiData.PSObject.Properties['IsNetworkError'] -and $ApiData.IsNetworkError
    if ($isNetworkError) {
        $errorMessage = "API Provider Problem"
        if ($ApiData.PSObject.Properties['ErrorCode'] -and $ApiData.ErrorCode) {
            $errorMessage += " ($($ApiData.ErrorCode))"
        }
        $errorMessage += " - Try again later"
        Write-Host $errorMessage -ForegroundColor Red
    }
    
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
        $percentChange24h = if ($rate24hAgo -ne 0) { (($currentBTC - $rate24hAgo) / $rate24hAgo) * 100 } else { 0 }

        # Round values to 2 decimal places for color comparison to ensure equality works as expected.
        $roundedCurrent = [math]::Round([decimal]$currentBTC, 2)
        $rounded24hAgo = [math]::Round([decimal]$rate24hAgo, 2)
        $priceColor24h = if ($roundedCurrent -gt $rounded24hAgo) { "Green" } elseif ($roundedCurrent -lt $rounded24hAgo) { "Red" } else { "White" }

        # Color for the main price line is based on session start price
        $roundedInitial = [math]::Round($InitialSessionBtcPrice, 2) # Already a decimal from param
        $priceColorSession = if ($roundedCurrent -gt $roundedInitial) { "Green" } elseif ($roundedCurrent -lt $roundedInitial) { "Red" } else { "White" }
        
        # Display Values
        $btcDisplay = "{0:C2}" -f $currentBTC
        $agoDisplay = "{0:C2} [{1}{2}%]" -f $rate24hAgo, $(if($roundedCurrent -gt $rounded24hAgo){"+"}), ("{0:N2}" -f $percentChange24h)
        
        $highDisplay = "N/A"
        if ($apiData.PSObject.Properties['rate24hHigh']) {
            $highDisplay = "{0:C2}" -f $apiData.rate24hHigh
            if ($apiData.PSObject.Properties['rate24hHighTime']) { $highDisplay += " (at $($apiData.rate24hHighTime.ToLocalTime().ToString("HH:mm")))" }
        }
        $lowDisplay = "N/A"
        if ($apiData.PSObject.Properties['rate24hLow']) {
            $lowDisplay = "{0:C2}" -f $apiData.rate24hLow
            if ($apiData.PSObject.Properties['rate24hLowTime']) { $lowDisplay += " (at $($apiData.rate24hLowTime.ToLocalTime().ToString("HH:mm")))" }
        }

        $volDisplay = "$($ApiData.volume.ToString("C0"))"
        # Time: shows when the (historical) API data was fetched, not when the main modal was loaded.
        $timeDisplay = if ($ApiData.PSObject.Properties['HistoricalDataFetchTime'] -and $ApiData.HistoricalDataFetchTime) {
            $ApiData.HistoricalDataFetchTime.ToLocalTime().ToString("MMddyy@HHmmss")
        } else {
            $ApiData.fetchTime.ToLocalTime().ToString("MMddyy@HHmmss")
        }

    } else {
        # Default/Warning Values
        $priceColorSession = "White"
        $priceColor24h = "White"
        $btcDisplay = "N/A"
        $agoDisplay = "N/A"
        $highDisplay = "N/A"
        $lowDisplay = "N/A"
        $volDisplay = "N/A"
        $timeDisplay = "N/A"
    }

    # --- 2. Screen Rendering ---
    Write-Host "*** Bitcoin Market ***" -ForegroundColor Yellow
    if (-not $marketDataAvailable -and -not $isNetworkError) {
        Write-Warning "Could not retrieve market data. Please check your API key in the Config menu."
    }
    Write-AlignedLine -Label "Bitcoin (USD):" -Value $btcDisplay -ValueColor $priceColorSession
    if ($ApiData -and $ApiData.PSObject.Properties['sma1h'] -and $ApiData.sma1h -gt 0) {
        $smaColor = "White"
        if ($ApiData.rate -gt $ApiData.sma1h) {
            $smaColor = "Green"
        } elseif ($ApiData.rate -lt $ApiData.sma1h) {
            $smaColor = "Red"
        }
        $smaDisplay = "{0:C2}" -f $ApiData.sma1h
        Write-AlignedLine -Label "1H SMA:" -Value $smaDisplay -ValueColor $smaColor
    }
    Write-AlignedLine -Label "24H Ago:" -Value $agoDisplay -ValueColor $priceColor24h
    Write-AlignedLine -Label "24H High:" -Value $highDisplay
    Write-AlignedLine -Label "24H Low:" -Value $lowDisplay
    if ($ApiData -and $ApiData.PSObject.Properties['volatility24h'] -and $ApiData.volatility24h -gt 0) {
        $volatilityColor = "White"
        if ($ApiData.PSObject.Properties['volatility12h'] -and $ApiData.PSObject.Properties['volatility12h_old']) {
            if ($ApiData.volatility12h -gt $ApiData.volatility12h_old) {
                $volatilityColor = "Green"
            } elseif ($ApiData.volatility12h -lt $ApiData.volatility12h_old) {
                $volatilityColor = "Red"
            }
        }
        $volatilityDisplay = "{0:N2}%" -f $ApiData.volatility24h
        Write-AlignedLine -Label "Volatility:" -Value $volatilityDisplay -ValueColor $volatilityColor
    }
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
        $sessionDisplay = "{0} [{1}{2:N2}%]" -f (Format-ProfitLoss -Value $sessionChange -FormatString "C2"), $(if($sessionPercent -ge 0){"+"}), $sessionPercent
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
    Write-Host "Get Free Key: https://www.livecoinwatch.com/tools/api" -ForegroundColor Green
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
        Write-Host "3. Archive Ledger"
        Write-Host "4. Merge Archived Ledgers"
        Write-Host "5. Return to Main Screen"
        Write-Host ""
        Write-Host "Enter your choice (Number 1-5): " -NoNewline
        
        # Wait for user input with Esc key support
        $choice = $null
        while ($true) {
            if ([System.Console]::KeyAvailable) {
                $key = [System.Console]::ReadKey($true)
                
                # Handle Esc key to return
                if ($key.Key -eq 'Escape') {
                    return
                }
                
                # Handle Enter key (empty input = return)
                if ($key.Key -eq 'Enter') {
                    $choice = ""
                    break
                }
                
                # Handle numeric keys 1-5
                if ($key.KeyChar -ge '1' -and $key.KeyChar -le '5') {
                    $choice = $key.KeyChar.ToString()
                    Write-Host $choice
                    break
                }
            }
            Start-Sleep -Milliseconds 50
        }

        switch ($choice) {
            "1" {
                # Read current key from file so we show what's actually in vbtc.ini (avoids stale in-memory config)
                $fileConfig = Get-IniConfiguration -FilePath $iniFilePath
                $currentKey = $fileConfig.Settings.ApiKey
                if ([string]::IsNullOrEmpty($currentKey)) { $currentKey = "(not set)" }
                Write-Host "Current API Key: $currentKey" -ForegroundColor Cyan
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
                $confirmReset = Read-Host "Are you sure you want to reset your portfolio? This cannot be undone. Type 'YES' to confirm" -ForegroundColor Red
                if ($confirmReset -ceq "YES") { # -ceq is case-sensitive equals
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
            "3" {
                Invoke-LedgerArchive -LedgerFilePath $ledgerFilePath
            }
            "4" {
                Invoke-LedgerMerge -ScriptPath $scriptPath
            }
            { $_ -eq '5' -or [string]::IsNullOrEmpty($_) } {
                return # Return on '5' or if input is empty
            }
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
    $buyTransactions = 0
    $sellTransactions = 0
    $totalWeightedBuyPrice = 0.0
    $totalWeightedSellPrice = 0.0
    $minUSD = [double]::MaxValue
    $maxUSD = [double]::MinValue
    $firstDt = $null
    $lastDt = $null

    if ($null -ne $LedgerData) {
        foreach ($row in $LedgerData) {
            try {
                $rowDt = [datetime]::ParseExact($row.Time, "MMddyy@HHmmss", $null)
                if ($null -eq $firstDt -or $rowDt -lt $firstDt) { $firstDt = $rowDt }
                if ($null -eq $lastDt -or $rowDt -gt $lastDt) { $lastDt = $rowDt }
            } catch {
                # Skip rows with unparseable timestamp
            }
            $rowUSD = [double]$row.USD
            # Tx Range = min/max Bitcoin price (USD per BTC) at time of any transaction, not total tx value
            $rowBtcPrice = [double]$row.'BTC(USD)'
            if ($rowBtcPrice -gt 0) {
                if ($rowBtcPrice -lt $minUSD) { $minUSD = $rowBtcPrice }
                if ($rowBtcPrice -gt $maxUSD) { $maxUSD = $rowBtcPrice }
            }
            if ($row.TX -eq "Buy") {
                $totalBuyUSD += $rowUSD
                $totalBuyBTC += [double]$row.BTC
                $buyTransactions++
                $totalWeightedBuyPrice += [double]$row.'BTC(USD)' * [double]$row.BTC
            } elseif ($row.TX -eq "Sell") {
                $totalSellUSD += $rowUSD
                $totalSellBTC += [double]$row.BTC
                $sellTransactions++
                $totalWeightedSellPrice += [double]$row.'BTC(USD)' * [double]$row.BTC
            }
        }
    }
    if ($minUSD -gt $maxUSD) { $minUSD = 0; $maxUSD = 0 }

    # Calculate average prices (weighted by BTC amount)
    $avgBuyPrice = 0.0
    $avgSalePrice = 0.0
    if ($totalBuyBTC -gt 0) {
        $avgBuyPrice = $totalWeightedBuyPrice / $totalBuyBTC
    }
    if ($totalSellBTC -gt 0) {
        $avgSalePrice = $totalWeightedSellPrice / $totalSellBTC
    }

    return [PSCustomObject]@{
        TotalBuyUSD      = $totalBuyUSD
        TotalSellUSD     = $totalSellUSD
        TotalBuyBTC      = $totalBuyBTC
        TotalSellBTC     = $totalSellBTC
        AvgBuyPrice      = $avgBuyPrice
        AvgSalePrice     = $avgSalePrice
        BuyTransactions  = $buyTransactions
        SellTransactions = $sellTransactions
        MinUSD           = $minUSD
        MaxUSD           = $maxUSD
        FirstDateTime    = $firstDt
        LastDateTime     = $lastDt
    }
}

function Format-Duration {
    param($Start, $End)
    if ($null -eq $Start -or $null -eq $End -or $End -lt $Start) {
        return ""
    }
    $duration = $End - $Start
    if ($duration.TotalMinutes -lt 60) {
        return "{0}M" -f [int][math]::Round($duration.TotalMinutes)
    }
    if ($duration.TotalHours -lt 24) {
        return "{0}H" -f [int][math]::Round($duration.TotalHours)
    }
    return "{0}D" -f [int][math]::Round($duration.TotalDays)
}

function Format-Cadence {
    param([TimeSpan]$Duration)
    if ($null -eq $Duration -or $Duration.TotalSeconds -le 0) { return "" }
    if ($Duration.TotalHours -lt 1) {
        $m = [int][math]::Floor($Duration.TotalMinutes)
        $s = [int][math]::Floor($Duration.TotalSeconds) % 60
        return "{0}M{1}S" -f $m, $s
    }
    if ($Duration.TotalHours -lt 48) {
        $h = [int][math]::Floor($Duration.TotalHours)
        $m = [int][math]::Floor($Duration.TotalMinutes) % 60
        return "{0}H{1}M" -f $h, $m
    }
    $d = [int][math]::Floor($Duration.TotalDays)
    return "{0}D" -f $d
}

function Get-AllLedgerData {
    $allEntries = @()
    $processedTimestamps = @{}

    # 1. Read current ledger
    if (Test-Path $ledgerFilePath) {
        $currentData = Import-Csv -Path $ledgerFilePath -ErrorAction SilentlyContinue
        if ($currentData) {
            foreach ($entry in $currentData) {
                if (-not $processedTimestamps.ContainsKey($entry.Time)) {
                    $processedTimestamps[$entry.Time] = $true
                    $allEntries += $entry
                }
            }
        }
    }

    # 2. Add merged ledger if present (historical bulk)
    $mergedLedgerPath = Join-Path -Path $scriptPath -ChildPath "vBTC - Ledger_Merged.csv"
    if (Test-Path $mergedLedgerPath) {
        $mergedData = Import-Csv -Path $mergedLedgerPath -ErrorAction SilentlyContinue
        if ($mergedData) {
            foreach ($entry in $mergedData) {
                if (-not $processedTimestamps.ContainsKey($entry.Time)) {
                    $processedTimestamps[$entry.Time] = $true
                    $allEntries += $entry
                }
            }
        }
    }

    # 3. Always add all unmerged archives (more recent; may be multiple) — dedup by Time
    # Match both legacy (MMddyy.csv) and new (MMddyy@HHmmss.csv) so multiple archives per day are included
    $archivePattern = "vBTC - Ledger_*.csv"
    $archiveFiles = Get-ChildItem -Path $scriptPath -Filter $archivePattern -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match "^vBTC - Ledger_(\d{6})(@(\d{6}))?\.csv$" } |
        Sort-Object {
            $base = $_.BaseName
            $suffix = $base -replace '^vBTC - Ledger_', ''
            if ($suffix -match '^(\d{6})@(\d{6})$') {
                [datetime]::ParseExact($suffix, 'MMddyy@HHmmss', $null)
            } else {
                [datetime]::ParseExact($suffix, 'MMddyy', $null)
            }
        }

    foreach ($archiveFile in $archiveFiles) {
        $archiveData = Import-Csv -Path $archiveFile.FullName -ErrorAction SilentlyContinue
        if ($archiveData) {
            foreach ($entry in $archiveData) {
                if (-not $processedTimestamps.ContainsKey($entry.Time)) {
                    $processedTimestamps[$entry.Time] = $true
                    $allEntries += $entry
                }
            }
        }
    }

    # 4. Sort all entries by DateTime chronologically
    return $allEntries | Sort-Object { [datetime]::ParseExact($_.Time, "MMddyy@HHmmss", $null) }
}

function Get-LedgerSummary {
    if (-not (Test-Path $ledgerFilePath)) { return $null }
    $ledgerData = Import-Csv -Path $ledgerFilePath
    if ($ledgerData.Count -eq 0) { return $null }
    return Get-LedgerTotals -LedgerData $ledgerData
}

function Get-SessionSummary {
    param ([datetime]$SessionStartTime)
    # Use all ledger data (current + archives) so session stats stay correct if user archived during session.
    $allEntries = Get-AllLedgerData
    if ($null -eq $allEntries -or $allEntries.Count -eq 0) {
        return $null
    }
    # Ledger timestamps are written in UTC (whole seconds only); truncate session start to seconds so trades in the same second are included.
    $sessionStartTruncated = [DateTime]::new($SessionStartTime.Year, $SessionStartTime.Month, $SessionStartTime.Day, $SessionStartTime.Hour, $SessionStartTime.Minute, $SessionStartTime.Second, $SessionStartTime.Kind)
    $sessionTransactions = @($allEntries) | Where-Object {
        try {
            $parsed = [datetime]::ParseExact($_.Time, "MMddyy@HHmmss", [System.Globalization.CultureInfo]::InvariantCulture)
            $parsedUtc = [DateTime]::SpecifyKind($parsed, [DateTimeKind]::Utc)
            return ($parsedUtc -ge $sessionStartTruncated)
        } catch {
            Write-Warning "Could not parse timestamp '$($_.Time)' in ledger."
            return $false
        }
    }

    if ($null -eq $sessionTransactions -or $sessionTransactions.Count -eq 0) {
        return $null
    }

    return Get-LedgerTotals -LedgerData $sessionTransactions
}

function Invoke-LedgerArchive {
    param ([string]$LedgerFilePath)

    if (-not (Test-Path $LedgerFilePath)) {
        Write-Warning "Ledger file not found. No action taken."
        Read-Host "Press Enter to continue."
        return
    }

    $linesToKeep = -1
    while ($linesToKeep -lt 0) {
        $promptinput = Read-Host "Keep X Recent Lines? [0]"
        if ([string]::IsNullOrEmpty($promptinput)) {
            $linesToKeep = 0
            break
        }
        if ([int]::TryParse($promptinput, [ref]$linesToKeep) -and $linesToKeep -ge 0) {
            break
        } else {
            Write-Warning "Invalid input. Please enter a non-negative integer."
            $linesToKeep = -1 # Reset for loop continuation
        }
    }

    $archiveFileName = "vBTC - Ledger_$(Get-Date -Format 'MMddyy@HHmmss').csv"
    $archivePath = Join-Path -Path $scriptPath -ChildPath $archiveFileName
    try {
        Copy-Item -Path $LedgerFilePath -Destination $archivePath -Force -ErrorAction Stop
        Write-Host "Ledger successfully backed up to '$archiveFileName'." -ForegroundColor Green
    } catch {
        Write-Error "Failed to create ledger archive. Error: $($_.Exception.Message)"
        Read-Host "Press Enter to continue."
        return
    }

    try {
        $header = Get-Content $LedgerFilePath -TotalCount 1
        $recordsToKeep = if ($linesToKeep -gt 0) { Get-Content $LedgerFilePath | Select-Object -Skip 1 | Select-Object -Last $linesToKeep } else { @() }
        
        Set-Content -Path $LedgerFilePath -Value $header -ErrorAction Stop
        if ($recordsToKeep.Count -gt 0) { Add-Content -Path $LedgerFilePath -Value $recordsToKeep -ErrorAction Stop }

        Write-Host "Original ledger has been purged, keeping the last $linesToKeep transaction(s)." -ForegroundColor Green
    } catch {
        Write-Error "Failed to purge the ledger file. Please check file permissions. The archive was still created. Error: $($_.Exception.Message)"
    }
    Read-Host "Press Enter to continue."
}

function Invoke-LedgerMerge {
    param ([string]$ScriptPath)

    Clear-Host
    Write-Host "*** Merge Archived Ledgers ***" -ForegroundColor Yellow

    $mergedLedgerPath = Join-Path -Path $ScriptPath -ChildPath "vBTC - Ledger_Merged.csv"
    $archivePattern = "vBTC - Ledger_*.csv"

    # 1. Find and sort archives (legacy MMddyy.csv and new MMddyy@HHmmss.csv)
    Write-Host "Searching for archives..."
    $archiveFiles = Get-ChildItem -Path $ScriptPath -Filter $archivePattern |
        Where-Object { $_.Name -match "^vBTC - Ledger_(\d{6})(@(\d{6}))?\.csv$" } |
        Sort-Object {
            $base = $_.BaseName
            $suffix = $base -replace '^vBTC - Ledger_', ''
            if ($suffix -match '^(\d{6})@(\d{6})$') {
                [datetime]::ParseExact($suffix, 'MMddyy@HHmmss', $null)
            } else {
                [datetime]::ParseExact($suffix, 'MMddyy', $null)
            }
        }

    if ($archiveFiles.Count -eq 0) {
        Write-Warning "No ledger archives found to merge."
        Read-Host "Press Enter to continue."
        return
    }
    Write-Host "Found $($archiveFiles.Count) archive(s) to analyze." -ForegroundColor Green

    # 2. Load existing data for deduplication
    $existingMergedData = @()
    $existingTimestamps = [System.Collections.Generic.HashSet[string]]::new()
    if (Test-Path $mergedLedgerPath) {
        try {
            $existingMergedData = Import-Csv -Path $mergedLedgerPath -ErrorAction Stop
            foreach ($row in $existingMergedData) {
                [void]$existingTimestamps.Add($row.Time)
            }
            Write-Host "Existing merged ledger contains $($existingMergedData.Count) transactions."
        }
        catch {
            Write-Warning "Could not read existing merged ledger at '$mergedLedgerPath'. It may be corrupt. Please check the file."
            Read-Host "Press Enter to continue."
            return
        }
    } else {
        Write-Host "No existing 'vBTC - Ledger_Merged.csv' found. A new one would be created."
    }

    # 3. Process archives and collect new unique transactions
    $newUniqueTransactions = [System.Collections.Generic.List[PSObject]]::new()
    $totalScannedTxCount = 0
    $processedTimestamps = [System.Collections.Generic.HashSet[string]]::new($existingTimestamps) # Clone the existing set

    Write-Host "Analyzing archives for new transactions..."
    foreach ($archiveFile in $archiveFiles) {
        try {
            $archiveData = Import-Csv -Path $archiveFile.FullName -ErrorAction Stop
            $totalScannedTxCount += $archiveData.Count
            foreach ($row in $archiveData) {
                if ($processedTimestamps.Add($row.Time)) {
                    # .Add() returns $true if the item was added (i.e., it was new)
                    $newUniqueTransactions.Add($row)
                }
            }
        } catch {
            Write-Warning "Could not read or parse '$($archiveFile.Name)'. Skipping this file."
        }
    }

    # 4. Display summary report
    Write-Host ""
    Write-Host "*** Merge Summary ***" -ForegroundColor Yellow
    Write-AlignedLine -Label "Archives Found:" -Value $archiveFiles.Count
    Write-AlignedLine -Label "Transactions in Archives:" -Value $totalScannedTxCount
    Write-AlignedLine -Label "Existing Merged TXs:" -Value $existingMergedData.Count
    Write-AlignedLine -Label "New Unique TXs to Add:" -Value $newUniqueTransactions.Count -ValueColor "Green"
    $expectedTotal = $existingMergedData.Count + $newUniqueTransactions.Count
    Write-AlignedLine -Label "New Total TXs:" -Value $expectedTotal

    Write-Host ""

    if ($newUniqueTransactions.Count -eq 0) {
        Write-Host "No new transactions to merge." -ForegroundColor Cyan
        Read-Host "Press Enter to continue."
        return
    }

    # 5. Get confirmation
    $mergedFileName = Split-Path -Path $mergedLedgerPath -Leaf
    $confirm = Read-Host "Proceed with merge? This will create/update '$mergedFileName'. (y/n)"
    if ($confirm.ToLower() -ne 'y') {
        Write-Host "Merge cancelled." -ForegroundColor Yellow
        Read-Host "Press Enter to continue."
        return
    }

    # 6. Perform safe write and verification
    Write-Host "Merging transactions..."
    $finalLedgerData = $existingMergedData + $newUniqueTransactions

    # Sort the final list by date to ensure chronological order
    $sortedFinalData = $finalLedgerData | Sort-Object { [datetime]::ParseExact($_.Time, "MMddyy@HHmmss", $null) }

    $tempFilePath = $null
    try {
        # Using a temporary file for a safer write operation
        $tempFilePath = [System.IO.Path]::GetTempFileName()
        $sortedFinalData | Export-Csv -Path $tempFilePath -NoTypeInformation -Force -ErrorAction Stop

        # Verify temp file before replacing the original
        $verificationData = Import-Csv -Path $tempFilePath -ErrorAction Stop
        if ($verificationData.Count -ne $expectedTotal) {
            throw "Verification failed. Expected $($expectedTotal) rows, but found $($verificationData.Count) in the temporary file."
        }

        # If verification passes, move the temp file to the final destination
        Move-Item -Path $tempFilePath -Destination $mergedLedgerPath -Force -ErrorAction Stop
        $tempFilePath = $null # Prevent deletion in finally block

        Write-Host "Verification successful. Cleaning up source archives..." -ForegroundColor Cyan
        foreach ($archiveFile in $archiveFiles) {
            try {
                Remove-Item -Path $archiveFile.FullName -Force -ErrorAction Stop
                Write-Verbose "Deleted $($archiveFile.Name)"
            }
            catch {
                Write-Warning "Could not delete archive file: $($archiveFile.FullName). Please remove it manually."
            }
        }

        Write-Host "Merge successful. '$mergedFileName' has been updated." -ForegroundColor Green
        Write-Host "$($archiveFiles.Count) source archive(s) have been deleted." -ForegroundColor Green
    }
    catch {
        Write-Error "An error occurred during the merge process: $($_.Exception.Message)"
        Write-Error "The merge has been aborted, and the original merged file (if any) is unchanged."
        Write-Error "No source archives were deleted."
    }
    finally {
        if ($null -ne $tempFilePath -and (Test-Path $tempFilePath)) {
            Remove-Item $tempFilePath -Force
        }
    }

    Read-Host "Press Enter to continue."
}

function Invoke-Trade {
    param ([ref]$Config, [string]$Type, [string]$AmountString = $null, [PSCustomObject]$CurrentApiData)

    # For the most accurate UI prompt, we should read the latest config from disk here too.
    # This prevents showing the user a stale "Max" amount if another client has made a trade.
    $promptConfig = Get-IniConfiguration -FilePath $iniFilePath
    if (-not $promptConfig) {
        # If the read fails, fall back to the in-memory config for the prompt.
        # The critical "read-before-write" is still performed later.
        Write-Warning "Could not read latest portfolio for prompt, using cached value."
        $promptConfig = $Config.Value
    }

    $playerUSD = 0.0
    $playerBTC = 0.0
    $null = [double]::TryParse($promptConfig.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerUSD)
    $null = [double]::TryParse($promptConfig.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerBTC)

    $maxAmount = if ($Type -eq "Buy") { $playerUSD } else { $playerBTC }
    $prompt = if ($Type -eq "Buy") {
        "Amount in USD [Max $($maxAmount.ToString("C2"))]:"
    } else {
        "Amount in BTC [Max $($maxAmount.ToString("F8"))] (or use 's' for satoshis):"
    }

    $userInput = $AmountString
    $tradeAmount = 0.0

    # --- Input Loop for Trade Amount ---
    while ($true) {
        Clear-Host
        Write-Host "*** $Type Bitcoin ***" -ForegroundColor Yellow
        if ([string]::IsNullOrEmpty($userInput)) { # Prompt for input if not provided as an argument
            $userInput = Read-Host $prompt
            if ([string]::IsNullOrEmpty($userInput)) { return (Get-BestApiData -Preferred $CurrentApiData) } # User cancelled: prefer last good snapshot
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
    $offerExpiredMessageNeeded = $false
    $tradeApiData = $CurrentApiData # Use a local variable for the loop

    :OuterTradeLoop while ($true) {
        # Update the local tradeApiData object, skipping the historical call for speed.
        $tradeApiData = Update-ApiData -Config $Config.Value -OldApiData $tradeApiData -SkipHistorical
        if ($tradeApiData -and $tradeApiData.PSObject.Properties['IsNetworkError'] -and $tradeApiData.IsNetworkError) {
            $errorMessage = "`nAPI Provider Problem"
            if ($tradeApiData.PSObject.Properties['ErrorCode'] -and $tradeApiData.ErrorCode) {
                $errorMessage += " ($($tradeApiData.ErrorCode))"
            }
            $errorMessage += " - Try again later"
            Write-Host $errorMessage -ForegroundColor Red
            Read-Host "Press Enter to return to the main menu."
            return (Get-BestApiData -Preferred $CurrentApiData)
        }
        if (-not $tradeApiData) { Read-Host "Error fetching price. Press Enter to continue."; return (Get-BestApiData -Preferred $CurrentApiData) } # return old/best on failure
        $offerTimestamp = Get-Date # Record the time the offer is presented.
        $timeoutSeconds = 120
        $tradeinput = $null # To store the user's choice

        $displayState = $null # Tracks what is currently on screen to prevent flicker

        # This loop will run until user provides input or it times out.
        :InnerInputLoop while ($true) {
            # Calculate remaining time and determine message/color
            $elapsedSeconds = ((Get-Date) - $offerTimestamp).TotalSeconds
            $secondsRemaining = $timeoutSeconds - $elapsedSeconds

            $requiredState = "Initial"
            if ($secondsRemaining -le 0) {
                $requiredState = "Expired"
            } elseif ($secondsRemaining -le 30) {
                $requiredState = "ThirtySeconds"
            } elseif ($secondsRemaining -le 60) {
                $requiredState = "OneMinute"
            }

            # --- Redraw logic ---
            # Only redraw the screen if the state has changed.
            if ($requiredState -ne $displayState) {
                $displayState = $requiredState # Update the current state
                $isExpiredOnScreen = ($displayState -eq "Expired")

                # Determine message and color based on the new state
                switch ($displayState) {
                    "Initial"       { $timeLeftMessage = "You have 2 minutes to accept this offer."; $timeLeftColor = "White" }
                    "OneMinute"     { $timeLeftMessage = "You have 1 minute to accept this offer."; $timeLeftColor = "Yellow" }
                    "ThirtySeconds" { $timeLeftMessage = "You have 30 seconds to accept this offer."; $timeLeftColor = "Red" }
                    "Expired"       { $timeLeftMessage = "Offer expired, please refresh for new offer."; $timeLeftColor = "Cyan" }
                }

                Clear-Host
                Write-Host "*** $Type Bitcoin ***" -ForegroundColor Yellow
                if ($offerExpiredMessageNeeded) {
                    Write-Host "`nOffer expired. A new price has been fetched." -ForegroundColor Yellow
                    $offerExpiredMessageNeeded = $false # Reset the flag after showing the message
                }

                # Display the offer details (same as before)
                $usdAmount = if ($Type -eq "Buy") { $tradeAmount } else { [math]::Floor(($tradeAmount * $tradeApiData.rate) * 100) / 100 }
                $btcAmount = if ($Type -eq "Sell") { $tradeAmount } else { [math]::Floor(($tradeAmount / $tradeApiData.rate) * 100000000) / 100000000 }
                $rate24hAgo = if ($tradeApiData.PSObject.Properties['rate24hAgo']) { $tradeApiData.rate24hAgo } else { $tradeApiData.rate }
                $priceDiff = $tradeApiData.rate - $rate24hAgo
                $priceColor = if ($priceDiff -gt 0) { "Green" } elseif ($priceDiff -lt 0) { "Red" } else { "White" }
                $confirmPrompt = if ($Type -eq "Buy") { "Purchase $($btcAmount.ToString("F8")) BTC for $($usdAmount.ToString("C2"))? " } else { "Sell $($btcAmount.ToString("F8")) BTC for $($usdAmount.ToString("C2"))? " }

                Write-Host "`n$timeLeftMessage" -ForegroundColor $timeLeftColor
                Write-Host -NoNewline "Market Rate: "
                Write-Host ("{0:C2}" -f $tradeApiData.rate) -ForegroundColor $priceColor
                Write-Host -NoNewline $confirmPrompt
                if ($isExpiredOnScreen) {
                    Write-Host "[" -NoNewline; Write-Host "r" -ForegroundColor Cyan -NoNewline; Write-Host "]"
                } else {
                    Write-Host "[" -NoNewline; Write-Host "y" -ForegroundColor Green -NoNewline; Write-Host "/" -NoNewline; Write-Host "r" -ForegroundColor Cyan -NoNewline; Write-Host "/" -NoNewline; Write-Host "n" -ForegroundColor Red -NoNewline; Write-Host "]"
                }
            }

            # --- Input/Timeout Check ---
            $loopStart = Get-Date
            while (((Get-Date) - $loopStart).TotalMilliseconds -lt 250) { # Check for input every 250ms to reduce flicker
                if ([System.Console]::KeyAvailable) {
                    $key = [System.Console]::ReadKey($true)

                    # Handle Enter key press separately
                    if ($key.Key -eq 'Enter') {
                        $isExpiredOnScreen = ($displayState -eq "Expired")
                        if ($isExpiredOnScreen) {
                            # On expired screen, Enter returns to main menu immediately.
                            return $tradeApiData
                        }
                        # On an active offer, Enter should cancel. We set a non-y/r value to fall through.
                        $tradeinput = "n"
                    }
                    # Handle Esc key as an alias for 'n' (cancel)
                    elseif ($key.Key -eq 'Escape') {
                        $tradeinput = "n"
                    }
                    # Handle arrow keys as aliases
                    elseif ($key.Key -eq 'LeftArrow') {
                        $tradeinput = "n"  # Left arrow = Esc = cancel
                    }
                    elseif ($key.Key -eq 'UpArrow') {
                        $tradeinput = "y"  # Up arrow = Y = accept
                    }
                    elseif ($key.Key -eq 'RightArrow') {
                        $tradeinput = "r"  # Right arrow = R = refresh
                    }
                    elseif ($key.Key -eq 'DownArrow') {
                        $tradeinput = "n"  # Down arrow = N = cancel
                    }
                    else {
                        $tradeinput = $key.KeyChar.ToString()
                    }
                    break # Break the inner 250ms loop
                }
                Start-Sleep -Milliseconds 50
            }

            if ($null -ne $tradeinput) {
                $isExpiredOnScreen = ($displayState -eq "Expired")
                if ($isExpiredOnScreen) {
                    # If the offer is expired on-screen, allow 'r' to refresh or Esc/Enter to cancel
                    if ($tradeinput.ToLower() -eq 'r') {
                        # Allow refresh - will fall through to handler below
                    } elseif ($tradeinput -eq "n" -or $tradeinput -eq "") {
                        # Esc was converted to 'n', or Enter was pressed - cancel and return
                        return $tradeApiData
                    } else {
                        $tradeinput = $null # Ignore invalid input (like 'y')
                        continue InnerInputLoop # Redraw the screen
                    }
                } else {
                    # On active offer screen, only allow 'y', 'r', 'n', Enter, or Esc
                    $validInput = $tradeinput.ToLower() -in @('y', 'r', 'n')
                    if (-not $validInput) {
                        $tradeinput = $null # Ignore invalid input
                        continue InnerInputLoop # Redraw the screen
                    }
                }
                # If we reach here, the input is valid for the current state, so break the loop to process it.
                break InnerInputLoop
            }

        } # End of InnerInputLoop

        if ($null -eq $tradeinput) {
            # Defensive: if somehow we fall through with no input, keep current data
            return (Get-BestApiData -Preferred $tradeApiData)
        }
        
        if ($tradeinput.ToLower() -eq 'y') {
            # Check if the offer has expired *at the moment of acceptance*.
            if (((Get-Date) - $offerTimestamp).TotalMinutes -ge 2) {
                $offerExpiredMessageNeeded = $true
                continue OuterTradeLoop # The offer is stale, loop to get a new price.
            }

            # --- RACE CONDITION FIX: Read-before-write ---
            # Reload config from disk to get the absolute latest portfolio state before committing the trade.
            $tradeConfig = Get-IniConfiguration -FilePath $iniFilePath
            if (-not $tradeConfig) {
                Write-Warning "`nCritical Error: Could not read portfolio file to finalize trade."
                Write-Warning "Your trade has been CANCELLED to prevent data loss."
                Read-Host "Press Enter to continue."
                return $tradeApiData # Cancel the trade
            }

            # Get the most up-to-date portfolio values
            $currentPlayerUSD = 0.0; $currentPlayerBTC = 0.0; $currentPlayerInvested = 0.0
            $null = [double]::TryParse($tradeConfig.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$currentPlayerUSD)
            $null = [double]::TryParse($tradeConfig.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$currentPlayerBTC)
            $null = [double]::TryParse($tradeConfig.Portfolio.PlayerInvested, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$currentPlayerInvested)

            # Re-verify if the trade is still possible with the latest balance
            if ($Type -eq "Buy" -and $usdAmount -gt $currentPlayerUSD) {
                Write-Warning "`nTrade cancelled. Your USD balance has changed since the trade was initiated."
                Write-Warning "Your current balance is $($currentPlayerUSD.ToString("C2")), but the trade required $($usdAmount.ToString("C2"))."
                Read-Host "Press Enter to continue."
            return (Get-BestApiData -Preferred $tradeApiData)
            }
            if ($Type -eq "Sell" -and $btcAmount -gt $currentPlayerBTC) {
                Write-Warning "`nTrade cancelled. Your BTC balance has changed since the trade was initiated."
                Write-Warning "Your current balance is $($currentPlayerBTC.ToString("F8")) BTC, but the trade required $($btcAmount.ToString("F8")) BTC."
                Read-Host "Press Enter to continue."
                return $tradeApiData
            }

            # --- Perform trade with fresh data ---
            $newUserBtc = 0.0; $newInvested = 0.0
            if ($Type -eq "Buy") {
                $tradeConfig.Portfolio.PlayerUSD = ($currentPlayerUSD - $usdAmount).ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
                $newUserBtc = $currentPlayerBTC + $btcAmount
                $newInvested = $currentPlayerInvested + $usdAmount
            } else { # Sell
                $newUserBtc = $currentPlayerBTC - $btcAmount
                if ($newUserBtc -le 0.000000005) { $newUserBtc = 0; $newInvested = 0 }
                else {
                    if ($currentPlayerBTC -gt 0) { $newInvested = $currentPlayerInvested * ($newUserBtc / $currentPlayerBTC) }
                    else { $newInvested = 0 }
                }
                $tradeConfig.Portfolio.PlayerUSD = ($currentPlayerUSD + $usdAmount).ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)
            }
            $tradeConfig.Portfolio.PlayerBTC = $newUserBtc.ToString("F8", [System.Globalization.CultureInfo]::InvariantCulture)
            $tradeConfig.Portfolio.PlayerInvested = $newInvested.ToString("F2", [System.Globalization.CultureInfo]::InvariantCulture)

            $saveSuccess = Set-IniConfiguration -FilePath $iniFilePath -Configuration $tradeConfig
            if ($saveSuccess) {
                $Config.Value = $tradeConfig # Update the in-memory config to reflect the successful trade
                Add-LedgerEntry -Type $Type -UsdAmount $usdAmount -BtcAmount $btcAmount -BtcPrice $tradeApiData.rate -UserBtcAfter $newUserBtc
                Write-Host "`n$Type successful."
                Start-Sleep -Seconds 1
            } else {
                Write-Warning "`nTrade failed: Could not save portfolio update. Please check file permissions for vbtc.ini."
                Read-Host "Press Enter to continue."
            }
            return $tradeApiData # Exit trade loop, returning the latest data
        } 
        elseif ($tradeinput.ToLower() -eq 'r') {
            # User is manually refreshing, so the offer isn't "expired".
            # The loop will naturally get a new price by continuing the outer loop.
            continue OuterTradeLoop
        }
        
        # For any other outcome ('n' cancel or Enter-cancel), return the latest tradeApiData so
        # the caller retains a valid (possibly cached) snapshot without forcing a full refresh.
        Write-Host "`n$Type cancelled."
        Start-Sleep -Seconds 1
        return (Get-BestApiData -Preferred $tradeApiData)
    }
}

function Show-HelpScreen {
    Clear-Host
    Write-Host "Virtual Bitcoin Trader (vBTC) - Version 1.5" -ForegroundColor Yellow
    Write-Host "===============================================================" -ForegroundColor DarkGray
    Write-Host ""
    
    Write-Host "COMMANDS:" -ForegroundColor Cyan
    Write-Host "    buy [amount]     " -NoNewline -ForegroundColor White; Write-Host "Purchase a specific USD amount of Bitcoin" -ForegroundColor Gray
    Write-Host "    sell [amount]    " -NoNewline -ForegroundColor White; Write-Host "Sell a specific amount of BTC (e.g., 0.5) or satoshis (e.g., 50000s)" -ForegroundColor Gray
    Write-Host "    ledger           " -NoNewline -ForegroundColor White; Write-Host "View a history of all your transactions" -ForegroundColor Gray
    Write-Host "    refresh          " -NoNewline -ForegroundColor White; Write-Host "Manually update the market data" -ForegroundColor Gray
    Write-Host "    config           " -NoNewline -ForegroundColor White; Write-Host "Access the configuration menu" -ForegroundColor Gray
    Write-Host "    help             " -NoNewline -ForegroundColor White; Write-Host "Show this help screen" -ForegroundColor Gray
    Write-Host "    exit             " -NoNewline -ForegroundColor White; Write-Host "Exit the application" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "TIPS:" -ForegroundColor Green
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Commands may be shortened (e.g. 'b 10' to buy $10 of BTC)" -ForegroundColor Gray
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Use 'p' for percentage trades (e.g., '50p' for 50%, '100/3p' for 33.3%)" -ForegroundColor Gray
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "Volatility shows the price swing (High vs Low) over the last 24 hours" -ForegroundColor Gray
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "1H SMA is the average price over the last hour. Green = price is above average" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "REQUIREMENTS:" -ForegroundColor Blue
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "PowerShell" -ForegroundColor Gray
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "An internet connection" -ForegroundColor Gray
    Write-Host "    * " -NoNewline -ForegroundColor Yellow; Write-Host "A free API key from https://www.livecoinwatch.com/tools/api" -ForegroundColor Gray
    Write-Host ""
    Write-Host "COMMAND LINE:" -ForegroundColor Cyan
    Write-Host "    -help, -?     " -NoNewline -ForegroundColor White; Write-Host "Show help and exit" -ForegroundColor Gray
    Write-Host "    -config      " -NoNewline -ForegroundColor White; Write-Host "Open configuration (e.g. to fix API key) and exit" -ForegroundColor Gray
    Write-Host ""
    Write-Host "===============================================================" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "Press Enter or Esc to return to the Main Screen."
    
    # Wait for user input with Esc key support
    while ($true) {
        if ([System.Console]::KeyAvailable) {
            $key = [System.Console]::ReadKey($true)
            
            # Handle Enter key to return to main screen
            if ($key.Key -eq 'Enter') {
                return
            }
            
            # Handle Esc key to return to main screen
            if ($key.Key -eq 'Escape') {
                return
            }
        }
        Start-Sleep -Milliseconds 50
    }
}

function Show-LedgerScreen {
    Clear-Host
    Write-Host "*** Ledger ***" -ForegroundColor Yellow

    # All data (current + archives) for summary; current log only for table
    $allLedgerData = Get-AllLedgerData
    $hasAnyData = $allLedgerData -and $allLedgerData.Count -gt 0
    $currentLedgerData = $null
    if (Test-Path $ledgerFilePath) {
        $currentLedgerData = Import-Csv -Path $ledgerFilePath -ErrorAction SilentlyContinue
    }
    $currentHasRows = $currentLedgerData -and $currentLedgerData.Count -gt 0

    if (-not $hasAnyData) {
        Write-Host "You have not made any transactions yet."
    } else {
        if ($currentHasRows) {
            # 1. Parse and create display objects
            $displayData = $currentLedgerData | ForEach-Object {
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
        } else {
            Write-Host "Log Empty"
        }

        # Ledger Summary from all data (current + archives) when any data exists
        $summary = Get-LedgerTotals -LedgerData $allLedgerData
        $sessionSummary = Get-SessionSummary -SessionStartTime $sessionStartTime
        if ($summary) {
                Write-Host ""
                Write-Host "*** Ledger Summary ***" -ForegroundColor Yellow
                
                # Portfolio Summary Section
                $playerUSD = 0.0
                $playerBTC = 0.0
                $playerInvested = 0.0
                $null = [double]::TryParse($config.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerUSD)
                $null = [double]::TryParse($config.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerBTC)
                $null = [double]::TryParse($config.Portfolio.PlayerInvested, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$playerInvested)
                $portfolioValue = Get-PortfolioValue -PlayerUSD $playerUSD -PlayerBTC $playerBTC -ApiData $apiData

                $portfolioColor = "White"
                if ($portfolioValue -is [double]) {
                    if ($portfolioValue -gt $startingCapital) { $portfolioColor = "Green" }
                    elseif ($portfolioValue -lt $startingCapital) { $portfolioColor = "Red" }
                }

                # Portfolio Value with session delta in [] (green if up, red if down)
                Write-Host -NoNewline "Portfolio Value:"
                $pvPadding = 22 - "Portfolio Value:".Length
                if ($pvPadding -gt 0) { Write-Host (" " * $pvPadding) -NoNewline }
                Write-Host -NoNewline ("{0:C2}" -f $portfolioValue) -ForegroundColor $portfolioColor
                if ($script:sessionStartPortfolioValue -gt 0 -and $portfolioValue -is [double]) {
                    $sessionPortfolioDelta = $portfolioValue - $script:sessionStartPortfolioValue
                    $deltaStr = " [{0:+#0.00;-#0.00;0.00}]" -f $sessionPortfolioDelta
                    $deltaColor = if ($sessionPortfolioDelta -gt 0) { "Green" } elseif ($sessionPortfolioDelta -lt 0) { "Red" } else { "White" }
                    Write-Host $deltaStr -ForegroundColor $deltaColor
                } else {
                    Write-Host ""
                }

                # Trading Statistics Section (all-time with session in [])
                if ($summary.TotalBuyUSD -gt 0) {
                    $usdVal = "{0:C2}" -f $summary.TotalBuyUSD
                    if ($sessionSummary) { $usdVal += " [{0:C2}]" -f $sessionSummary.TotalBuyUSD }
                    Write-AlignedLine -Label "Total Bought (USD):" -Value $usdVal -ValueColor "Green"
                    $btcVal = $summary.TotalBuyBTC.ToString("F8")
                    if ($sessionSummary) { $btcVal += " [{0}]" -f $sessionSummary.TotalBuyBTC.ToString("F8") }
                    Write-AlignedLine -Label "Total Bought (BTC):" -Value $btcVal -ValueColor "Green"
                }

                # Display additional statistics
                $totalTransactions = $summary.BuyTransactions + $summary.SellTransactions
                if ($totalTransactions -gt 0) {
                    $txVal = $totalTransactions.ToString()
                    if ($sessionSummary) {
                        $sessionTx = $sessionSummary.BuyTransactions + $sessionSummary.SellTransactions
                        $txVal += " [$sessionTx]"
                    }
                    Write-AlignedLine -Label "Transaction Count:" -Value $txVal -ValueColor "White"
                }

                if ($summary.AvgBuyPrice -gt 0) {
                    $avgVal = "{0:C2}" -f $summary.AvgBuyPrice
                    if ($sessionSummary) {
                        if ($sessionSummary.AvgBuyPrice -gt 0) { $avgVal += " [{0:C2}]" -f $sessionSummary.AvgBuyPrice }
                        else { $avgVal += " [$0.00]" }
                    }
                    Write-AlignedLine -Label "Average Purchase:" -Value $avgVal -ValueColor "Green"
                }

                if ($summary.AvgSalePrice -gt 0) {
                    $saleVal = "{0:C2}" -f $summary.AvgSalePrice
                    if ($sessionSummary) {
                        if ($sessionSummary.AvgSalePrice -gt 0) { $saleVal += " [{0:C2}]" -f $sessionSummary.AvgSalePrice }
                        else { $saleVal += " [$0.00]" }
                    }
                    Write-AlignedLine -Label "Average Sale:" -Value $saleVal -ValueColor "Red"
                }
                if ($totalTransactions -gt 0 -and $summary.MaxUSD -ge $summary.MinUSD) {
                    Write-AlignedLine -Label "Tx Range:" -Value ("{0:C2} - {1:C2}" -f $summary.MinUSD, $summary.MaxUSD) -ValueColor "White"
                    if ($sessionSummary -and $sessionSummary.MaxUSD -ge $sessionSummary.MinUSD) {
                        Write-AlignedLine -Label "Session Tx Range:" -Value ("{0:C2} - {1:C2}" -f $sessionSummary.MinUSD, $sessionSummary.MaxUSD) -ValueColor "White"
                    }
                }
                $totalLen = Format-Duration -Start $summary.FirstDateTime -End $summary.LastDateTime
                if ($totalLen -ne "") {
                    $timeVal = $totalLen
                    if ($sessionSummary -and $null -ne $sessionSummary.FirstDateTime -and $null -ne $sessionSummary.LastDateTime) {
                        $sessionLen = Format-Duration -Start $sessionSummary.FirstDateTime -End $sessionSummary.LastDateTime
                        if ($sessionLen -ne "") { $timeVal += " [$sessionLen]" }
                    }
                    Write-AlignedLine -Label "Time:" -Value $timeVal -ValueColor "White"
                }
                # Cadence: time per trade (ledger and session); slower = red, quicker = green
                if ($totalTransactions -gt 0 -and $null -ne $summary.FirstDateTime -and $null -ne $summary.LastDateTime -and $summary.LastDateTime -gt $summary.FirstDateTime) {
                    $ledgerSpan = $summary.LastDateTime - $summary.FirstDateTime
                    $ledgerCadenceDur = [TimeSpan]::FromSeconds($ledgerSpan.TotalSeconds / $totalTransactions)
                    $ledgerCadenceStr = Format-Cadence -Duration $ledgerCadenceDur
                    if ($ledgerCadenceStr -ne "") {
                        $sessionCadenceStr = ""
                        $sessionCadenceDur = [TimeSpan]::Zero
                        if ($sessionSummary) {
                            $sessionTx = $sessionSummary.BuyTransactions + $sessionSummary.SellTransactions
                            if ($sessionTx -gt 0) {
                                # Session cadence: session start to now (not first trade to last trade)
                                $sessionSpan = (Get-Date).ToUniversalTime() - $sessionStartTime
                                $sessionCadenceDur = [TimeSpan]::FromSeconds($sessionSpan.TotalSeconds / $sessionTx)
                                $sessionCadenceStr = Format-Cadence -Duration $sessionCadenceDur
                            }
                        }
                        $label = "Cadence:"
                        $paddingLen = [math]::Max(0, 22 - $label.Length)
                        Write-Host $label -NoNewline
                        Write-Host (" " * $paddingLen) -NoNewline
                        if ($sessionCadenceStr -eq "") {
                            Write-Host $ledgerCadenceStr -ForegroundColor White
                        } else {
                            $ledgerIsSlower = $ledgerCadenceDur -ge $sessionCadenceDur
                            if ($ledgerIsSlower) {
                                Write-Host $ledgerCadenceStr -NoNewline -ForegroundColor Red
                                Write-Host " [" -NoNewline -ForegroundColor White
                                Write-Host $sessionCadenceStr -NoNewline -ForegroundColor Green
                                Write-Host "]" -ForegroundColor White
                            } else {
                                Write-Host $ledgerCadenceStr -NoNewline -ForegroundColor Green
                                Write-Host " [" -NoNewline -ForegroundColor White
                                Write-Host $sessionCadenceStr -NoNewline -ForegroundColor Red
                                Write-Host "]" -ForegroundColor White
                            }
                        }
                    }
                }
            }
        }

    Write-Host ""
    Write-Host "Press Enter to return to Main screen, or R to refresh"
    
    # Wait for user input
    while ($true) {
        if ([System.Console]::KeyAvailable) {
            $key = [System.Console]::ReadKey($true)
            
            # Handle Enter key to return to main screen
            if ($key.Key -eq 'Enter') {
                return
            }
            
            # Handle Esc key to return to main screen
            if ($key.Key -eq 'Escape') {
                return
            }
            
            # Handle Right Arrow or 'R' or 'r' for refresh
            if ($key.Key -eq 'RightArrow' -or $key.KeyChar -eq 'R' -or $key.KeyChar -eq 'r') {
                # Reload config from disk (for portfolio values)
                $script:config = Get-IniConfiguration -FilePath $iniFilePath
                if (-not $script:config) {
                    Write-Warning "Error reloading portfolio from vbtc.ini."
                    Write-Host "Press Enter to continue."
                    while ($true) {
                        if ([System.Console]::KeyAvailable) {
                            $key2 = [System.Console]::ReadKey($true)
                            if ($key2.Key -eq 'Enter') {
                                break
                            }
                        }
                        Start-Sleep -Milliseconds 50
                    }
                    return
                }
                
                # Stale check: refresh API data if >15 minutes old so Portfolio Value on summary is current.
                $isStale = -not $script:apiData -or (-not $script:apiData.PSObject.Properties['HistoricalDataFetchTime']) -or (((Get-Date).ToUniversalTime() - $script:apiData.HistoricalDataFetchTime).TotalMinutes -gt 15)
                if ($isStale) { $script:apiData = Update-ApiData -Config $script:config -OldApiData $script:apiData }
                
                # Recursively call Show-LedgerScreen to redraw with fresh ledger data
                Show-LedgerScreen
                return
            }
        }
        Start-Sleep -Milliseconds 50
    }
}


# --- Main Game Loop ---

$config = Get-IniConfiguration -FilePath $iniFilePath
if ($OpenConfig.IsPresent) {
    if (-not $config) {
        $config = @{
            Settings   = @{ ApiKey = "" }
            Portfolio  = @{ PlayerUSD = $startingCapital.ToString("F2"); PlayerBTC = "0.0"; PlayerInvested = "0.0" }
        }
    }
    Show-ConfigScreen -Config ([ref]$config)
    exit 0
}
# Try API first; on 403 with no cache we show message and exit (no first-time setup)
$apiData = Update-ApiData -Config $config -OldApiData $apiData
if ($null -eq $apiData -and [string]::IsNullOrEmpty($config.Settings.ApiKey)) {
    Show-FirstRunSetup -Config ([ref]$config)
    $config = Get-IniConfiguration -FilePath $iniFilePath
    $apiData = Update-ApiData -Config $config -OldApiData $apiData
}

# Only after the data is fully fetched, set the session-start values.
# This prevents the race condition and ensures the initial price is correct.
$initialSessionBtcPrice = if ($apiData -and $apiData.rate) { [decimal]$apiData.rate } else { 0 }

$initialPlayerUSD = 0.0
$initialPlayerBTC = 0.0
$null = [double]::TryParse($config.Portfolio.PlayerUSD, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$initialPlayerUSD)
$null = [double]::TryParse($config.Portfolio.PlayerBTC, [System.Globalization.NumberStyles]::Any, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$initialPlayerBTC)
$sessionStartPortfolioValue = Get-PortfolioValue -PlayerUSD $initialPlayerUSD -PlayerBTC $initialPlayerBTC -ApiData $apiData

# Session = from when the main screen is first shown (after setup); used for Ledger session stats.
$sessionStartTime = (Get-Date).ToUniversalTime()

$commands = @("buy", "sell", "ledger", "refresh", "config", "help", "exit")

while ($true) {
    Show-MainScreen -ApiData $apiData -Portfolio $config -SessionStartValue $sessionStartPortfolioValue -InitialSessionBtcPrice $initialSessionBtcPrice

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
                $returned = Invoke-Trade -Config ([ref]$config) -Type "Buy" -AmountString $amount -CurrentApiData $apiData
                $apiData = Get-BestApiData -Preferred $returned
                # After returning from trade, always reload config to ensure the main screen is perfectly in sync.
                $config = Get-IniConfiguration -FilePath $iniFilePath
                # Stale check: refresh API data before showing main screen if >15 minutes old.
                $isStale = -not $apiData -or (-not $apiData.PSObject.Properties['HistoricalDataFetchTime']) -or (((Get-Date).ToUniversalTime() - $apiData.HistoricalDataFetchTime).TotalMinutes -gt 15)
                if ($isStale) { $apiData = Update-ApiData -Config $config -OldApiData $apiData }
            }
            "sell" {
                $returned = Invoke-Trade -Config ([ref]$config) -Type "Sell" -AmountString $amount -CurrentApiData $apiData
                $apiData = Get-BestApiData -Preferred $returned
                # After returning from trade, always reload config to ensure the main screen is perfectly in sync.
                $config = Get-IniConfiguration -FilePath $iniFilePath
                # Stale check: refresh API data before showing main screen if >15 minutes old.
                $isStale = -not $apiData -or (-not $apiData.PSObject.Properties['HistoricalDataFetchTime']) -or (((Get-Date).ToUniversalTime() - $apiData.HistoricalDataFetchTime).TotalMinutes -gt 15)
                if ($isStale) { $apiData = Update-ApiData -Config $config -OldApiData $apiData }
            }
            "ledger" { Show-LedgerScreen }
            "refresh" {
                $config = Get-IniConfiguration -FilePath $iniFilePath
                $apiData = Update-ApiData -Config $config -OldApiData $apiData
            }
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
                $profitColor = if ($profit -gt 0) { "Green" }
                elseif ($profit -lt 0) { "Red" }
                else { "White" }
            
                Write-AlignedLine -Label "Portfolio Value:" -Value ("{0:C2}" -f $finalValue) -ValueColor $profitColor

                # --- Session Summary ---
                Write-Host ""
                Write-Host "*** Session Summary ***" -ForegroundColor Yellow
                $sessionValueStartColumn = 22 # Use a consistent start column for this block to align values

                $summary = Get-SessionSummary -SessionStartTime $sessionStartTime
                if ($summary) {
                    $totalTransactions = $summary.BuyTransactions + $summary.SellTransactions
                    Write-AlignedLine -Label "Transactions:" -Value ($totalTransactions.ToString()) -ValueColor "White" -ValueStartColumn $sessionValueStartColumn
                }

                $finalBtcPrice = if ($apiData -and $apiData.rate) { [decimal]$apiData.rate } else { $initialSessionBtcPrice }
                $roundedInitial = [math]::Round($initialSessionBtcPrice, 2)
                $roundedFinal = [math]::Round($finalBtcPrice, 2)
                $sessionPriceColor = if ($roundedFinal -gt $roundedInitial) { "Green" }
                elseif ($roundedFinal -lt $roundedInitial) { "Red" }
                else { "White" }

                Write-AlignedLine -Label "Start BTC(USD):" -Value ("{0:C2}" -f $initialSessionBtcPrice) -ValueStartColumn $sessionValueStartColumn
                Write-AlignedLine -Label "End BTC(USD):" -Value ("{0:C2}" -f $finalBtcPrice) -ValueColor $sessionPriceColor -ValueStartColumn $sessionValueStartColumn

                if ($sessionStartPortfolioValue -gt 0 -and $finalValue -is [double]) {
                    $sessionChange = $finalValue - $sessionStartPortfolioValue
                    $sessionPercent = ($sessionChange / $sessionStartPortfolioValue) * 100
                    $sessionColor = if ($sessionChange -gt 0) { "Green" }
                    elseif ($sessionChange -lt 0) { "Red" }
                    else { "White" }
                    $sessionDisplay = "{0} [{1}{2:N2}%]" -f (Format-ProfitLoss -Value $sessionChange -FormatString "C2"), $(if($sessionPercent -ge 0){"+"}), $sessionPercent
                    Write-AlignedLine -Label "P/L:" -Value $sessionDisplay -ValueColor $sessionColor -ValueStartColumn $sessionValueStartColumn
                }

                if ($summary) {
                    if ($summary.TotalBuyUSD -gt 0) {
                        Write-AlignedLine -Label "Total Bought (USD):" -Value ("{0:C2}" -f $summary.TotalBuyUSD) -ValueColor "Green" -ValueStartColumn $sessionValueStartColumn
                        Write-AlignedLine -Label "Total Bought (BTC):" -Value $summary.TotalBuyBTC.ToString("F8") -ValueColor "Green" -ValueStartColumn $sessionValueStartColumn
                    }
                    if ($summary.TotalSellUSD -gt 0) {
                        Write-AlignedLine -Label "Total Sold (USD):" -Value ("{0:C2}" -f $summary.TotalSellUSD) -ValueColor "Red" -ValueStartColumn $sessionValueStartColumn
                        Write-AlignedLine -Label "Total Sold (BTC):" -Value $summary.TotalSellBTC.ToString("F8") -ValueColor "Red" -ValueStartColumn $sessionValueStartColumn
                    }
                    if ($summary.AvgBuyPrice -gt 0) {
                        Write-AlignedLine -Label "Average Purchase:" -Value ("{0:C2}" -f $summary.AvgBuyPrice) -ValueColor "Green" -ValueStartColumn $sessionValueStartColumn
                    }
                    if ($summary.AvgSalePrice -gt 0) {
                        Write-AlignedLine -Label "Average Sale:" -Value ("{0:C2}" -f $summary.AvgSalePrice) -ValueColor "Red" -ValueStartColumn $sessionValueStartColumn
                    }
                    if ($summary.MaxUSD -ge $summary.MinUSD) {
                        Write-AlignedLine -Label "Session Tx Range:" -Value ("{0:C2} - {1:C2}" -f $summary.MinUSD, $summary.MaxUSD) -ValueColor "White" -ValueStartColumn $sessionValueStartColumn
                    }
                    if ($null -ne $summary.FirstDateTime -and $null -ne $summary.LastDateTime) {
                        $sessionLen = Format-Duration -Start $summary.FirstDateTime -End $summary.LastDateTime
                        if ($sessionLen -ne "") {
                            Write-AlignedLine -Label "Time:" -Value $sessionLen -ValueColor "White" -ValueStartColumn $sessionValueStartColumn
                        }
                    }
                    $sessionTx = $summary.BuyTransactions + $summary.SellTransactions
                    if ($sessionTx -gt 0) {
                        $sessionSpan = (Get-Date).ToUniversalTime() - $sessionStartTime
                        $sessionCadenceDur = [TimeSpan]::FromSeconds($sessionSpan.TotalSeconds / $sessionTx)
                        $sessionCadenceStr = Format-Cadence -Duration $sessionCadenceDur
                        if ($sessionCadenceStr -ne "") {
                            Write-AlignedLine -Label "Cadence:" -Value $sessionCadenceStr -ValueColor "White" -ValueStartColumn $sessionValueStartColumn
                        }
                    }
                }

                # --- Comprehensive Ledger Summary ---
                Write-Host ""
                Write-Host "*** Trading History Summary ***" -ForegroundColor Yellow
                $allLedgerData = Get-AllLedgerData
                if ($allLedgerData -and $allLedgerData.Count -gt 0) {
                    $allTimeSummary = Get-LedgerTotals -LedgerData $allLedgerData
                    $ledgerValueStartColumn = 22 # Use consistent column alignment

                    # Display transaction counts
                    $totalTransactions = $allTimeSummary.BuyTransactions + $allTimeSummary.SellTransactions
                    if ($totalTransactions -gt 0) {
                        Write-AlignedLine -Label "Total Transactions:" -Value $totalTransactions.ToString() -ValueColor "White" -ValueStartColumn $ledgerValueStartColumn
                    }

                    # Display totals
                    if ($allTimeSummary.TotalBuyUSD -gt 0) {
                        Write-AlignedLine -Label "Total Bought (USD):" -Value ("{0:C2}" -f $allTimeSummary.TotalBuyUSD) -ValueColor "Green" -ValueStartColumn $ledgerValueStartColumn
                        Write-AlignedLine -Label "Total Bought (BTC):" -Value $allTimeSummary.TotalBuyBTC.ToString("F8") -ValueColor "Green" -ValueStartColumn $ledgerValueStartColumn
                    }
                    if ($allTimeSummary.TotalSellUSD -gt 0) {
                        Write-AlignedLine -Label "Total Sold (USD):" -Value ("{0:C2}" -f $allTimeSummary.TotalSellUSD) -ValueColor "Red" -ValueStartColumn $ledgerValueStartColumn
                        Write-AlignedLine -Label "Total Sold (BTC):" -Value $allTimeSummary.TotalSellBTC.ToString("F8") -ValueColor "Red" -ValueStartColumn $ledgerValueStartColumn
                    }

                    # Display average prices
                    if ($allTimeSummary.AvgBuyPrice -gt 0) {
                        Write-AlignedLine -Label "Average Purchase:" -Value ("{0:C2}" -f $allTimeSummary.AvgBuyPrice) -ValueColor "Green" -ValueStartColumn $ledgerValueStartColumn
                    }
                    if ($allTimeSummary.AvgSalePrice -gt 0) {
                        Write-AlignedLine -Label "Average Sale:" -Value ("{0:C2}" -f $allTimeSummary.AvgSalePrice) -ValueColor "Red" -ValueStartColumn $ledgerValueStartColumn
                    }
                    $exitTxCount = $allTimeSummary.BuyTransactions + $allTimeSummary.SellTransactions
                    if ($exitTxCount -gt 0 -and $allTimeSummary.MaxUSD -ge $allTimeSummary.MinUSD) {
                        Write-AlignedLine -Label "Tx Range:" -Value ("{0:C2} - {1:C2}" -f $allTimeSummary.MinUSD, $allTimeSummary.MaxUSD) -ValueColor "White" -ValueStartColumn $ledgerValueStartColumn
                    }
                    $exitTimeLen = Format-Duration -Start $allTimeSummary.FirstDateTime -End $allTimeSummary.LastDateTime
                    if ($exitTimeLen -ne "") {
                        Write-AlignedLine -Label "Time:" -Value $exitTimeLen -ValueColor "White" -ValueStartColumn $ledgerValueStartColumn
                    }

                    # Net BTC Position
                    $netBTC = $allTimeSummary.TotalBuyBTC - $allTimeSummary.TotalSellBTC
                    $netBTCColor = "White"
                    if ($netBTC -gt 0) { $netBTCColor = "Green" }
                    elseif ($netBTC -lt 0) { $netBTCColor = "Red" }
                    Write-AlignedLine -Label "Net BTC Position:" -Value $netBTC.ToString("F8") -ValueColor $netBTCColor -ValueStartColumn $ledgerValueStartColumn

                    # Net Profit/Loss USD
                    $netProfitLoss = $allTimeSummary.TotalSellUSD - $allTimeSummary.TotalBuyUSD
                    $netPLColor = "White"
                    if ($netProfitLoss -gt 0) { $netPLColor = "Green" }
                    elseif ($netProfitLoss -lt 0) { $netPLColor = "Red" }
                    Write-AlignedLine -Label "Net Trading P/L (USD):" -Value ("{0:C2}" -f $netProfitLoss) -ValueColor $netPLColor -ValueStartColumn $ledgerValueStartColumn
                } else {
                    Write-Host "No trading history found." -ForegroundColor Cyan
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