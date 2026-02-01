<#
.SYNOPSIS
    A lightweight, real-time Bitcoin price monitor.

.DESCRIPTION
    This script provides a simple command-line interface to monitor the real-time price of Bitcoin.
    If an API key is not found in a local bmon.ini, it will check for the vbtc.ini file from the vBTC application. If no key is found in either location, it will guide the user through a one-time setup to create bmon.ini.
 
    It can be run in several modes:
    - Interactive Mode: The main screen displays the current price. Press the space bar to start/pause monitoring.
    - Go Mode (`-go`): Monitors the price immediately for 15 minutes and then exits.
    - Long Go Mode (`-golong`): Monitors for 24 hours with a longer update interval.

    Sound (`-s`) and the history sparkline (`-h`) can be enabled from the command line for any monitoring mode.
 
    In monitoring modes, the price line will flash with an inverted color for 500ms to draw attention to significant price movements. This flash occurs when the price color changes (e.g., from neutral to green) or when the price continues to move in an already established direction (e.g., ticking up again while a green).
 
.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-08-07@1430
    Version: 1.5
#>

[CmdletBinding(DefaultParameterSetName='Monitor')]
param (
    [Parameter(ParameterSetName='Monitor')]
    [Alias('g')]
    [switch]$go,

    [Parameter(ParameterSetName='Monitor')]
    [Alias('gl')]
    [switch]$golong,

    [Parameter(ParameterSetName='Monitor')]
    [Alias('history')]
    [switch]$h,

    [Parameter(ParameterSetName='Monitor')]
    [Alias('sound')]
    [switch]$s,

    [Parameter(ParameterSetName='Monitor')]
    [switch]$k,

    [Parameter(ParameterSetName='Monitor')]
    [switch]$Help,

    [Parameter(ParameterSetName='Monitor')]
    [switch]$config,

    [Parameter(ParameterSetName='BitcoinToUsd')]
    [Alias('bu')]
    [ValidateRange(0, [double]::MaxValue)]
    [double]$BitcoinToUsd,

    [Parameter(ParameterSetName='UsdToBitcoin')]
    [Alias('ub')]
    [ValidateRange(0, [double]::MaxValue)]
    [double]$UsdToBitcoin,

    [Parameter(ParameterSetName='USDToSats')]
    [Alias('us')]
    [ValidateRange(0, [double]::MaxValue)]
    [double]$USDToSats,

    [Parameter(ParameterSetName='SatsToUSD')]
    [Alias('su')]
    [ValidateRange(0, [double]::MaxValue)]
    [double]$SatsToUSD
)

# --- Script Setup and Configuration ---
# Set output encoding to UTF-8 to properly display special characters.
[System.Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# Show help if requested
if ($Help.IsPresent) {
    Write-Host "Bitcoin Monitor (bmon) - Version 1.5" -ForegroundColor Yellow
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor DarkGray
    Write-Host ""
    
    Write-Host "USAGE:" -ForegroundColor Cyan
    Write-Host "    .\bmon.ps1              " -NoNewline -ForegroundColor White; Write-Host "# Interactive mode" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -go          " -NoNewline -ForegroundColor White; Write-Host "# Monitor for 15 minutes" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -g           " -NoNewline -ForegroundColor White; Write-Host "# Monitor for 15 minutes (alias)" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -golong      " -NoNewline -ForegroundColor White; Write-Host "# Monitor for 24 hours" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -gl          " -NoNewline -ForegroundColor White; Write-Host "# Monitor for 24 hours (alias)" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -s           " -NoNewline -ForegroundColor White; Write-Host "# Enable sound alerts" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -h           " -NoNewline -ForegroundColor White; Write-Host "# Enable history sparkline" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -config      " -NoNewline -ForegroundColor White; Write-Host "# Open configuration menu" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -bu 0.5      " -NoNewline -ForegroundColor White; Write-Host "# 0.5 BTC to USD" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -ub 50000    " -NoNewline -ForegroundColor White; Write-Host "# `$50,000 to BTC" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -us 100      " -NoNewline -ForegroundColor White; Write-Host "# `$100 to satoshis" -ForegroundColor Gray
    Write-Host "    .\bmon.ps1 -su 1000000  " -NoNewline -ForegroundColor White; Write-Host "# 1M satoshis to USD" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "MONITORING MODES:" -ForegroundColor Green
    Write-Host "    Interactive: " -NoNewline -ForegroundColor White; Write-Host "Press Space to start/pause, R to reset, Ctrl+C to exit" -ForegroundColor Gray
    Write-Host "    Go Mode: " -NoNewline -ForegroundColor White; Write-Host "15-minute monitoring with 5-second updates" -ForegroundColor Gray
    Write-Host "    Long Go Mode: " -NoNewline -ForegroundColor White; Write-Host "24-hour monitoring with 20-second updates" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "CONTROLS (during monitoring):" -ForegroundColor Magenta
    Write-Host "    R - " -NoNewline -ForegroundColor White; Write-Host "Reset baseline price and timer" -ForegroundColor Gray
    Write-Host "    E - " -NoNewline -ForegroundColor White; Write-Host "Extend session timer (keep baseline)" -ForegroundColor Gray
    Write-Host "    M - " -NoNewline -ForegroundColor White; Write-Host "Switch between go/golong modes" -ForegroundColor Gray
    Write-Host "    S - " -NoNewline -ForegroundColor White; Write-Host "Toggle sound alerts" -ForegroundColor Gray
    Write-Host "    H - " -NoNewline -ForegroundColor White; Write-Host "Toggle history sparkline" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "FEATURES:" -ForegroundColor Blue
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Real-time Bitcoin price monitoring" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Price change indicators (green/red)" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Visual price flash alerts" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Sound alerts for price movements" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Historical price sparkline" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "BTC/USD conversion tools" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Satoshi conversion tools" -ForegroundColor Gray
    Write-Host "    • " -NoNewline -ForegroundColor Yellow; Write-Host "Automatic API key management" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "API KEY:" -ForegroundColor Red
    Write-Host "    Get a free API key from: " -ForegroundColor White
    Write-Host "    " -NoNewline -ForegroundColor White; Write-Host "https://www.livecoinwatch.com/tools/api" -ForegroundColor Cyan
    Write-Host "    The script will guide you through setup on first run." -ForegroundColor Gray
    Write-Host ""
    Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor DarkGray
    exit 0
}

if ($PSScriptRoot) {
    $scriptPath = $PSScriptRoot
}
elseif ($MyInvocation.MyCommand.Path) {
    $scriptPath = Split-Path -Path $MyInvocation.MyCommand.Path -Parent
}
else {
    $scriptPath = Get-Location
}
$bmonIniFilePath = Join-Path -Path $scriptPath -ChildPath "bmon.ini"
$vbtcIniFilePath = Join-Path -Path $scriptPath -ChildPath "vbtc.ini"


# --- Helper Functions ---

function Get-IniConfiguration {
    param ([string]$FilePath)
    $ini = @{ "Settings" = @{ "ApiKey" = "" } }
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
                # We only care about the ApiKey in the Settings section for this script
                if ($currentSection -eq "Settings" -and $key -eq "ApiKey") {
                    $ini[$currentSection][$key] = $value
                }
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
        return $true
    }
    catch {
        Write-Error "Failed to save configuration to $FilePath. Error: $($_.Exception.Message)"
        return $false
    }
}

# Tracks whether the previous operation printed a one-line warning/retry message
$script:WarningLineShown = $false

function Write-RetryIndicator {
    param(
        [int]$Attempt,
        [switch]$Final,
        [switch]$Fetching
    )
    # Color logic: Cyan when fetching (like spinner), Red on final failure, Yellow during wait
    $fg = if ($Fetching) { 'Cyan' } elseif ($Final) { 'Red' } else { 'Yellow' }
    $digit = [string]$Attempt
    try {
        # Move to column 0, write the colored digit only, then return to column 0.
        # Do NOT pad the rest of the line; leave existing content intact.
        Write-Host -NoNewline "`r"
        Write-Host -NoNewline -ForegroundColor $fg $digit
        Write-Host -NoNewline "`r"
    } catch {
        Write-Host -NoNewline "`r$digit"
    }
}

function Test-ApiKey {
    param ([string]$ApiKey)
    if ([string]::IsNullOrEmpty($ApiKey)) { return $false }
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
    try {
        Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -TimeoutSec 10 -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

function Invoke-Onboarding {
    param([string]$BmonIniPath)
    Clear-Host
    Write-Host "*** bmon First Time Setup ***" -ForegroundColor Yellow
    Write-Host "A LiveCoinWatch API key is required to monitor prices." -ForegroundColor White
    Write-Host "Get a free key at: https://www.livecoinwatch.com/tools/api" -ForegroundColor Green
    $validKey = $false
    $newApiKey = $null
    while (-not $validKey) {
        $userInput = Read-Host "Please enter your LiveCoinWatch API Key (or press Enter to exit)"
        if ([string]::IsNullOrEmpty($userInput)) {
            return $null # User cancelled
        }
        if (Test-ApiKey -ApiKey $userInput) {
            $validKey = $true
            $newApiKey = $userInput
            $configToSave = @{ "Settings" = @{ "ApiKey" = $newApiKey } }
            if (Set-IniConfiguration -FilePath $BmonIniPath -Configuration $configToSave) {
                Write-Host "API Key is valid and has been saved to bmon.ini." -ForegroundColor Green
                Read-Host "Press Enter to start monitoring."
            } else {
                Write-Error "API Key was valid, but failed to save to bmon.ini. Please check file permissions."
                Read-Host "Press Enter to exit."
                return $null
            }
        } else {
            Write-Warning "Invalid API Key. Please try again."
        }
    }
    return $newApiKey
}

function Get-BtcPrice {
    param ([string]$ApiKey, [switch]$IsInitialFetch)

    if ([string]::IsNullOrEmpty($ApiKey)) {
        Write-Error "API Key is null or empty"
        return $null
    }

    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
    
    $maxAttempts = 5
    $baseDelaySeconds = 2

    $retried = $false
    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        try {
            $response = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -TimeoutSec 10 -ErrorAction Stop
            
            # Clear any prior retry indicator on success
            if ($retried -or $script:WarningLineShown) { $script:WarningLineShown = $false }

            $price = $response.rate -as [double]
            
            if ($null -eq $price -or $price -le 0) {
                # Treat invalid price as transient; show a retry digit '1'
                Write-RetryIndicator -Attempt 1
                $script:WarningLineShown = $true
                return $null
            }
            return $price
        }
        catch {
            $retried = $true
            if ($attempt -ge $maxAttempts) {
                # Final failure indicator: red '5'
                Write-RetryIndicator -Attempt $maxAttempts -Final
                $script:WarningLineShown = $true
                return $null
            }
            
            # Show timeout message for initial fetch on first retry
            if ($IsInitialFetch -and $attempt -eq 1) {
                Write-ClearLine
                Write-Host -NoNewline "  Timeout, retrying...`r" -ForegroundColor Yellow
            }
            
            # Exponential backoff with jitter
            $jitterMs = Get-Random -Minimum 0 -Maximum 1000
            $backoffSeconds = [math]::Pow(2, $attempt - 1) * $baseDelaySeconds
            $sleepDurationMs = [int]($backoffSeconds * 1000) + $jitterMs
            
            # Show attempt number in yellow at spinner position
            Write-RetryIndicator -Attempt $attempt
            $script:WarningLineShown = $true
            Start-Sleep -Milliseconds $sleepDurationMs
            
            # Change to cyan before retry attempt (like spinner does before fetch)
            Write-RetryIndicator -Attempt $attempt -Fetching
        }
    }
    return $null
}

function Invoke-SoundToggle {
    param([ref]$SoundEnabled)
    $SoundEnabled.Value = -not $SoundEnabled.Value
    if ($SoundEnabled.Value) {
        [System.Console]::Beep(1200, 350)
    }
    else {
        [System.Console]::Beep(400, 350)
    }
}

function Get-Sparkline {
    param ([System.Collections.Generic.List[double]]$History)
    if ($null -eq $History -or $History.Count -lt 2) { return "              ".PadRight(14) }

    # Use Unicode code points for compatibility with older PowerShell and encoding
    $sparkChars = [char[]]([char]0x2581, [char]0x2582, [char]0x2583, [char]0x2584, [char]0x2585, [char]0x2586, [char]0x2587, [char]0x2588)
    $minPrice = ($History | Measure-Object -Minimum).Minimum
    $maxPrice = ($History | Measure-Object -Maximum).Maximum
    $priceRange = $maxPrice - $minPrice

    if ($priceRange -eq 0 -or $priceRange -lt 0.00000001) { return ([string]$sparkChars[0] * 14).PadRight(14) }

    $sparkline = ""
    foreach ($price in $History) {
        $normalized = ($price - $minPrice) / $priceRange
        $charIndex = [math]::Floor($normalized * ($sparkChars.Length - 1))
        $sparkline += $sparkChars[$charIndex]
    }
    # Pad to 14 characters on the left if needed (to make new updates appear from the right)
    if ($sparkline.Length -lt 14) {
        $sparkline = $sparkline.PadLeft(14)
    }
    # Truncate to 14 characters if needed (keep the most recent data on the right)
    if ($sparkline.Length -gt 14) {
        $sparkline = $sparkline.Substring($sparkline.Length - 14)
    }
    return $sparkline.PadRight(14)
}

function Get-ConsoleWidth {
    try {
        return [System.Console]::WindowWidth
    }
    catch {
        return 80  # Default fallback width
    }
}

function Write-ClearLine {
    param([string]$Message = "")
    # Prefer VT sequence to clear the entire line, fallback to space padding
    try {
        $esc = [char]27
        # ESC[2K clears the entire current line; CR returns cursor to column 0
        Write-Host -NoNewline ("${esc}[2K`r")
        if ($Message) { Write-Host -NoNewline $Message }
    }
    catch {
        try {
            $consoleWidth = Get-ConsoleWidth
            $clearLine = "`r" + (' ' * [math]::Max(0, $consoleWidth)) + "`r"
            Write-Host -NoNewline $clearLine
            if ($Message) { Write-Host -NoNewline $Message }
        }
        catch { Write-Host "" }
    }
}

# Optional: full-screen clear via VT, used if a warning wrapped onto more than one line
function Write-ClearScreen {
    try {
        $esc = [char]27
        Write-Host -NoNewline ("${esc}[2J${esc}[H")
    } catch { Clear-Host }
}

# --- Main Script ---

# -config: show current settings (if any) and open configuration menu, then exit
if ($config.IsPresent) {
    $bmonCfg = Get-IniConfiguration -FilePath $bmonIniFilePath
    $vbtcCfg = Get-IniConfiguration -FilePath $vbtcIniFilePath
    $currentKey = $bmonCfg.Settings.ApiKey
    $configPath = $bmonIniFilePath
    if ([string]::IsNullOrEmpty($currentKey)) {
        $currentKey = $vbtcCfg.Settings.ApiKey
        $configPath = $vbtcIniFilePath
    }
    Clear-Host
    Write-Host "*** bmon Configuration ***" -ForegroundColor Yellow
    Write-Host ""
    if (-not [string]::IsNullOrEmpty($currentKey)) {
        $masked = if ($currentKey.Length -le 8) { "****" } else { $currentKey.Substring(0, 4) + "***" + $currentKey.Substring($currentKey.Length - 4) }
        Write-Host "Config file: " -NoNewline -ForegroundColor White
        Write-Host $configPath -ForegroundColor Cyan
        Write-Host "ApiKey: " -NoNewline -ForegroundColor White
        Write-Host $masked -ForegroundColor Gray
        Write-Host ""
    }
    Write-Host "Get a free API key from: https://www.livecoinwatch.com/tools/api" -ForegroundColor Green
    Write-Host ""
    $userInput = Read-Host "Enter new LiveCoinWatch API Key (or press Enter to keep current and exit)"
    if ([string]::IsNullOrEmpty($userInput)) {
        if (-not [string]::IsNullOrEmpty($currentKey)) {
            Write-Host "No changes made." -ForegroundColor Gray
        }
        exit 0
    }
    if (Test-ApiKey -ApiKey $userInput) {
        $configToSave = @{ "Settings" = @{ "ApiKey" = $userInput } }
        if (Set-IniConfiguration -FilePath $bmonIniFilePath -Configuration $configToSave) {
            Write-Host "API Key saved to bmon.ini." -ForegroundColor Green
        } else {
            Write-Error "API Key was valid, but failed to save to bmon.ini."
            exit 1
        }
    } else {
        Write-Host "Invalid API Key. No changes saved." -ForegroundColor Red
        exit 1
    }
    exit 0
}

# Try to get API key from bmon.ini first
$bmonConfig = Get-IniConfiguration -FilePath $bmonIniFilePath
$apiKey = $bmonConfig.Settings.ApiKey

# If not found, try vbtc.ini
if ([string]::IsNullOrEmpty($apiKey)) {
    $vbtcConfig = Get-IniConfiguration -FilePath $vbtcIniFilePath
    $apiKey = $vbtcConfig.Settings.ApiKey
}

# If still not found, start onboarding
if ([string]::IsNullOrEmpty($apiKey)) {
    $apiKey = Invoke-Onboarding -BmonIniPath $bmonIniFilePath
}

# Final check, if onboarding was cancelled or failed
if ([string]::IsNullOrEmpty($apiKey)) {
    Write-Host "API Key not found or configured. Exiting." -ForegroundColor Red
    exit
}

# Handle undocumented -k switch: acts as shortcut for -go with -h enabled

# --- Conversion Logic Branch ---
if ($PSCmdlet.ParameterSetName -ne 'Monitor') {
    $price = Get-BtcPrice -ApiKey $apiKey -IsInitialFetch
    if ($null -eq $price) {
        Write-Error "Could not retrieve Bitcoin price. Cannot perform conversion."
        exit 1
    }

    switch ($PSCmdlet.ParameterSetName) {
        "BitcoinToUsd" {
            $usdValue = $BitcoinToUsd * $price
            Write-Host ('${0}' -f $usdValue.ToString("N2"))
        }
        "UsdToBitcoin" {
            if ($price -le 0.00000001) { Write-Error "Bitcoin price is too low or zero, cannot divide."; exit 1 }
            $btcValue = $UsdToBitcoin / $price
            Write-Host ("B{0}" -f $btcValue.ToString("F8"))
        }
        "USDToSats" {
            if ($price -le 0.00000001) { Write-Error "Bitcoin price is too low or zero, cannot divide."; exit 1 }
            $satoshiValue = ($USDToSats / $price) * 100000000
            Write-Host ("{0}s" -f [math]::Round($satoshiValue).ToString("N0"))
        }
        "SatsToUSD" {
            $usdValue = ($SatsToUSD / 100000000) * $price
            Write-Host ('${0}' -f $usdValue.ToString("N2"))
        }
    }
    exit 0
}


# Initial Price Fetch for monitor modes
if ($go.IsPresent -or $golong.IsPresent -or $k.IsPresent) {
    Clear-Host
    Write-Host -NoNewline "Fetching initial price...`r" -ForegroundColor Cyan
} else {
    Write-Host "Fetching initial price..." -ForegroundColor Cyan
}
$currentBtcPrice = Get-BtcPrice -ApiKey $apiKey -IsInitialFetch
if ($null -eq $currentBtcPrice) {
    Write-Host "Failed to fetch initial price. Check API key or network." -ForegroundColor Red
    exit
}

# --- Main Logic Branch ---

if ($go.IsPresent -or $golong.IsPresent -or $k.IsPresent) {
    # --- Mode Configuration ---
    # Spinner chars via Unicode code points for older PowerShell/encoding compatibility
    $modeSettings = @{
        'go'     = @{ duration = 900;   interval = 5;  spinner = @([char]0x280B, [char]0x2819, [char]0x2839, [char]0x2838, [char]0x283C, [char]0x2834, [char]0x2826, [char]0x2827, [char]0x2807, [char]0x280F) }
        'golong' = @{ duration = 86400; interval = 20; spinner = @([char]0x259A, [char]0x259A, [char]0x259A, [char]0x259A, [char]0x259A, [char]0x259A, [char]0x259E, [char]0x259E, [char]0x259E, [char]0x259E, [char]0x259E, [char]0x259E) }
        'k'      = @{ duration = 1800;   interval = 4;  spinner = @([char]0x258F, [char]0x258E, [char]0x258D, [char]0x258C, [char]0x258B, [char]0x258A, [char]0x2589, [char]0x2588, [char]0x2589, [char]0x258A, [char]0x258B, [char]0x258C, [char]0x258D, [char]0x258E) }
    }
    $currentMode = if ($k.IsPresent) { 'k' } elseif ($golong.IsPresent) { 'golong' } else { 'go' }

    # --- Initial State Setup ---
    $monitorStartPrice = $currentBtcPrice
    $previousIntervalPrice = $currentBtcPrice
    $previousPriceColor = "White"
    $monitorStartTime = Get-Date
    $soundEnabled = $s.IsPresent
    $sparklineEnabled = $h.IsPresent -or $k.IsPresent
    $priceHistory = [System.Collections.Generic.List[double]]::new()
    $priceHistory.Add($currentBtcPrice)

    $monitorDurationSeconds = 0
    $waitIntervalSeconds = 0
    $spinner = @()
    $spinnerIndex = 0
 
    trap [System.Management.Automation.PipelineStoppedException] {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor()
        try {
            Write-ClearLine
        }
        catch {
            Write-Host "`n"
        }
        exit
    }

    try {
        [System.Console]::CursorVisible = $false
        while ($true) {
            # Set mode-specific variables and immediately check for termination.
            $monitorDurationSeconds = $modeSettings[$currentMode].duration
            if (((Get-Date) - $monitorStartTime).TotalSeconds -ge $monitorDurationSeconds) { break }

            $waitIntervalSeconds = $modeSettings[$currentMode].interval
            $spinner = $modeSettings[$currentMode].spinner

            $priceChange = $currentBtcPrice - $monitorStartPrice
            $priceColor = "White"
            $changeString = ""
            if ($priceChange -ge 0.01) {
                $priceColor = "Green"
                $changeString = " [+$($priceChange.ToString("C2"))]"
            } elseif ($priceChange -le -0.01) {
                $priceColor = "Red"
                $changeString = " [$($priceChange.ToString("C2"))]"
            }

            $flashNeeded = $false
            if ($priceColor -ne "White" -and $priceColor -ne $previousPriceColor) {
                $flashNeeded = $true
            }
            elseif (($priceColor -eq "Green" -and $currentBtcPrice -gt $previousIntervalPrice) -or
                    ($priceColor -eq "Red" -and $currentBtcPrice -lt $previousIntervalPrice)) {
                $flashNeeded = $true
            }

            $waitStart = Get-Date
            $refreshed = $false
            $modeSwitched = $false
            $isFirstTick = $true
            while (((Get-Date) - $waitStart).TotalSeconds -lt $waitIntervalSeconds) {
                $spinnerChar = $spinner[$spinnerIndex]
                $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { " Bitcoin (USD):" }
                $restOfLine = "$sparklineString $($currentBtcPrice.ToString("C2"))$changeString"

                $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length

                $fullLine = "$spinnerChar$restOfLine"
                $paddedLine = $fullLine.PadRight([System.Console]::WindowWidth)

                if ($flashNeeded -and $isFirstTick) {
                    Write-Host -NoNewline -BackgroundColor $priceColor -ForegroundColor Black "$paddedLine`r"
                }
                else {
                    Write-Host -NoNewline -ForegroundColor White $spinnerChar
                    Write-Host -NoNewline -ForegroundColor $priceColor $restOfLine
                    $paddingSize = [System.Console]::WindowWidth - $fullLine.Length
                    if ($paddingSize -gt 0) {
                        Write-Host -NoNewline (' ' * $paddingSize)
                    }
                    Write-Host -NoNewline "`r"
                }
                
                $isFirstTick = $false

                if ([System.Console]::KeyAvailable) {
                    $keyInfo = [System.Console]::ReadKey($true)
                    # Arrow key aliases
                    if ($keyInfo.Key -eq 'LeftArrow') {
                        # Left arrow is alias for E (extend session timer)
                        # Extend session timer without changing comparison baseline
                        $monitorStartTime = Get-Date
                        # do not mark refreshed; keep baseline
                        
                        # Visual feedback: flash the screen using existing mechanism
                        $spinnerChar = $spinner[$spinnerIndex]
                        $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { " Bitcoin (USD):" }
                        $restOfLine = "$sparklineString $($currentBtcPrice.ToString("C2"))$changeString"
                        $fullLine = "$spinnerChar$restOfLine"
                        $paddedLine = $fullLine.PadRight([System.Console]::WindowWidth)
                        
                        # Flash with inverted colors
                        Write-Host -NoNewline -BackgroundColor $priceColor -ForegroundColor Black "$paddedLine`r"
                        Start-Sleep -Milliseconds 300
                        
                        # Audio feedback: brief beep if sound is enabled
                        if ($soundEnabled) {
                            [System.Console]::Beep(800, 200)
                        }
                    }
                    if ($keyInfo.Key -eq 'UpArrow') {
                        # Up arrow is alias for K
                        $currentMode = 'k'
                        $sparklineEnabled = $true
                        $monitorStartTime = Get-Date
                        $monitorStartPrice = $currentBtcPrice
                        $modeSwitched = $true
                        $spinnerIndex = 0
                        break
                    }
                    if ($keyInfo.Key -eq 'RightArrow') {
                        # Right arrow is alias for R
                        $monitorStartPrice = $currentBtcPrice
                        $monitorStartTime = Get-Date
                        $refreshed = $true
                        break
                    }
                    if ($keyInfo.Key -eq 'DownArrow') {
                        # Down arrow is alias for M
                        $currentMode = if ($currentMode -eq 'go') { 'golong' } else { 'go' }
                        $monitorStartTime = Get-Date
                        $monitorStartPrice = $currentBtcPrice
                        $modeSwitched = $true
                        $spinnerIndex = 0
                        break
                    }
                    if ($keyInfo.Key -eq 'Escape') {
                        [System.Console]::CursorVisible = $true
                        [System.Console]::ResetColor()
                        try { Write-ClearLine } catch { Write-Host "`n" }
                        exit
                    }
                    if ($keyInfo.KeyChar -eq 'r') {
                        $monitorStartPrice = $currentBtcPrice
                        $monitorStartTime = Get-Date
                        $refreshed = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'e' -or $keyInfo.KeyChar -eq 'E') {
                        # Extend session timer without changing comparison baseline
                        $monitorStartTime = Get-Date
                        # do not mark refreshed; keep baseline
                        
                        # Visual feedback: flash the screen using existing mechanism
                        $spinnerChar = $spinner[$spinnerIndex]
                        $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { " Bitcoin (USD):" }
                        $restOfLine = "$sparklineString $($currentBtcPrice.ToString("C2"))$changeString"
                        $fullLine = "$spinnerChar$restOfLine"
                        $paddedLine = $fullLine.PadRight([System.Console]::WindowWidth)
                        
                        # Flash with inverted colors
                        Write-Host -NoNewline -BackgroundColor $priceColor -ForegroundColor Black "$paddedLine`r"
                        Start-Sleep -Milliseconds 300
                        
                        # Audio feedback: brief beep if sound is enabled
                        if ($soundEnabled) {
                            [System.Console]::Beep(800, 200)
                        }
                    }
                    if ($keyInfo.KeyChar -eq 'k' -or $keyInfo.KeyChar -eq 'K') {
                        $currentMode = 'k'
                        $sparklineEnabled = $true
                        $monitorStartTime = Get-Date
                        $monitorStartPrice = $currentBtcPrice
                        $modeSwitched = $true
                        $spinnerIndex = 0
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'm') {
                        $currentMode = if ($currentMode -eq 'go') { 'golong' } else { 'go' }
                        $monitorStartTime = Get-Date
                        $monitorStartPrice = $currentBtcPrice
                        $modeSwitched = $true
                        $spinnerIndex = 0
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'i' -or $keyInfo.KeyChar -eq 'I') {
                        $paused = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 's') {
                        Invoke-SoundToggle -SoundEnabled ([ref]$soundEnabled)
                    }
                    if ($keyInfo.KeyChar -eq 'h') {
                        $sparklineEnabled = -not $sparklineEnabled
                    }
                }
                Start-Sleep -Milliseconds 500
            }
            if ($refreshed -or $modeSwitched) { continue }

            $previousIntervalPrice = $currentBtcPrice
            $previousPriceColor = $priceColor

            $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
            $spinnerChar = $spinner[$spinnerIndex]
            $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { " Bitcoin (USD):" }
            $restOfLine = "$sparklineString $($currentBtcPrice.ToString("C2"))$changeString"
            
            Write-Host -NoNewline -ForegroundColor Cyan $spinnerChar
            Write-Host -NoNewline -ForegroundColor $priceColor $restOfLine
            $fullLine = "$spinnerChar$restOfLine"
            $paddingSize = [System.Console]::WindowWidth - $fullLine.Length
            if ($paddingSize -gt 0) {
                Write-Host -NoNewline (' ' * $paddingSize)
            }
            Write-Host -NoNewline "`r"

            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) {
                if ($soundEnabled) {
                    if ($newPrice -ge ($currentBtcPrice + 0.01)) { [System.Console]::Beep(1200, 150) }
                    elseif ($newPrice -le ($currentBtcPrice - 0.01)) { [System.Console]::Beep(400, 150) }
                }
                $currentBtcPrice = $newPrice
                $priceHistory.Add($currentBtcPrice)
                if ($priceHistory.Count -gt 14) {
                    $priceHistory.RemoveAt(0)
                }
            }
        }
    }
    finally {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor()
        try {
            Write-ClearLine
        }
        catch {
            Write-Host "`n"
        }
    }
    exit
}
else {
    # Interactive Mode (original behavior)
    $soundEnabled = $s.IsPresent
    $sparklineEnabled = $h.IsPresent
    while ($true) {
        Clear-Host
        Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow
        Write-Host "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))" -ForegroundColor White
        Write-Host -NoNewline "Start[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Go Mode[" -ForegroundColor White; Write-Host -NoNewline "G" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;

        while ($true) {
            if ([System.Console]::KeyAvailable) {
                $key = [System.Console]::ReadKey($true)
                if ($key.Key -eq 'Escape') {
                    [System.Console]::CursorVisible = $true
                    [System.Console]::ResetColor()
                    exit
                }
                if ($key.Key -eq 'Spacebar') { break }
                if ($key.KeyChar -eq 'g' -or $key.KeyChar -eq 'G') {
                    $argsList = @()
                    if ($s.IsPresent) { $argsList += '-s' }
                    if ($h.IsPresent) { $argsList += '-h' }
                    & $PSCommandPath -go @argsList
                    return
                }
            }
            Start-Sleep -Milliseconds 100
        }

        Write-Host "`nStarting monitoring..." -ForegroundColor Cyan
        $newPrice = Get-BtcPrice -ApiKey $apiKey
        if ($null -ne $newPrice) { $currentBtcPrice = $newPrice }

        $monitorStartPrice = $currentBtcPrice
        $monitorStartTime = Get-Date
        $priceHistory = [System.Collections.Generic.List[double]]::new()
        $priceHistory.Add($currentBtcPrice)

        $previousIntervalPrice = $currentBtcPrice
        $previousPriceColor = "White"
        $monitorDurationSeconds = 300

        $drawScreen = {
            param([boolean]$InvertColors, [string]$ChangeString, [string]$PriceColor)

            Clear-Host
            Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow

            $sparklineString = if ($sparklineEnabled) { "$(Get-Sparkline -History $priceHistory)" } else { "Bitcoin (USD):" }
            $priceLine = "$sparklineString $($currentBtcPrice.ToString("C2"))$ChangeString"

            $fgColor = if ($InvertColors) { "Black" } else { $PriceColor }
            $bgColor = if ($InvertColors) { $PriceColor } else { [System.Console]::BackgroundColor }

            $oldFg = [System.Console]::ForegroundColor
            $oldBg = [System.Console]::BackgroundColor
            try {
                [System.Console]::ForegroundColor = $fgColor
                [System.Console]::BackgroundColor = $bgColor
                Write-Host $priceLine
            }
            finally {
                [System.Console]::ForegroundColor = $oldFg
                [System.Console]::BackgroundColor = $oldBg
            }
            Write-Host -NoNewline "Pause[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Reset[" -ForegroundColor White; Write-Host -NoNewline "R" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;
        }

        do {
            $priceChange = $currentBtcPrice - $monitorStartPrice
            $priceColor = "White"
            $changeString = ""
            if ($priceChange -ge 0.01) {
                $priceColor = "Green"
                $changeString = " [+$($priceChange.ToString("C2"))]"
            } elseif ($priceChange -le -0.01) {
                $priceColor = "Red"
                $changeString = " [$($priceChange.ToString("C2"))]"
            }
 
            $flashNeeded = $false
            if ($priceColor -ne "White" -and $priceColor -ne $previousPriceColor) {
                $flashNeeded = $true
            }
            elseif (($priceColor -eq "Green" -and $currentBtcPrice -gt $previousIntervalPrice) -or
                    ($priceColor -eq "Red" -and $currentBtcPrice -lt $previousIntervalPrice)) {
                $flashNeeded = $true
            }
            
            if ($flashNeeded) { & $drawScreen -InvertColors $true -ChangeString $changeString -PriceColor $priceColor; Start-Sleep -Milliseconds 500 }
            & $drawScreen -InvertColors $false -ChangeString $changeString -PriceColor $priceColor

            $waitStart = Get-Date
            $paused = $false
            $refreshed = $false
            while (((Get-Date) - $waitStart).TotalSeconds -lt 5) {
                if ([System.Console]::KeyAvailable) {
                    $keyInfo = [System.Console]::ReadKey($true)
                    # Arrow key aliases
                    if ($keyInfo.Key -eq 'LeftArrow') {
                        # Left arrow is alias for E (extend session timer)
                        # Extend interactive session timer without changing baseline
                        $monitorStartTime = Get-Date
                        
                        # Visual feedback: flash the screen
                        & $drawScreen -InvertColors $true -ChangeString $changeString -PriceColor $priceColor
                        Start-Sleep -Milliseconds 300
                        & $drawScreen -InvertColors $false -ChangeString $changeString -PriceColor $priceColor
                        
                        # Audio feedback: brief beep if sound is enabled
                        if ($soundEnabled) {
                            [System.Console]::Beep(800, 200)
                        }
                    }
                    if ($keyInfo.Key -eq 'RightArrow') {
                        # Right arrow is alias for R
                        $refreshed = $true
                        break
                    }
                    if ($keyInfo.Key -eq 'Escape') {
                        [System.Console]::CursorVisible = $true
                        [System.Console]::ResetColor()
                        try { Write-ClearLine } catch { Write-Host "`n" }
                        exit
                    }
                    if ($keyInfo.Key -eq 'Spacebar') {
                        $paused = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'r') {
                        $refreshed = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'e' -or $keyInfo.KeyChar -eq 'E') {
                        # Extend interactive session timer without changing baseline
                        $monitorStartTime = Get-Date
                        
                        # Visual feedback: flash the screen
                        & $drawScreen -InvertColors $true -ChangeString $changeString -PriceColor $priceColor
                        Start-Sleep -Milliseconds 300
                        & $drawScreen -InvertColors $false -ChangeString $changeString -PriceColor $priceColor
                        
                        # Audio feedback: brief beep if sound is enabled
                        if ($soundEnabled) {
                            [System.Console]::Beep(800, 200)
                        }
                    }
                    if ($keyInfo.KeyChar -eq 's') {
                        Invoke-SoundToggle -SoundEnabled ([ref]$soundEnabled)
                    }
                    if ($keyInfo.KeyChar -eq 'h') {
                        $sparklineEnabled = -not $sparklineEnabled
                        & $drawScreen -InvertColors $false -ChangeString $changeString -PriceColor $priceColor
                    }
                }
                Start-Sleep -Milliseconds 100
            }
            if ($paused) { break }
            if ($refreshed) {
                $monitorStartPrice = $currentBtcPrice
                $monitorStartTime = Get-Date
                continue
            }

            $previousIntervalPrice = $currentBtcPrice
            $previousPriceColor = $priceColor

            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) {
                if ($soundEnabled) {
                    if ($newPrice -ge ($currentBtcPrice + 0.01)) { [System.Console]::Beep(1200, 150) }
                    elseif ($newPrice -le ($currentBtcPrice - 0.01)) { [System.Console]::Beep(400, 150) }
                }
                $currentBtcPrice = $newPrice
                $priceHistory.Add($currentBtcPrice)
                if ($priceHistory.Count -gt 14) {
                    $priceHistory.RemoveAt(0)
                }
            }
        } while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds)
    }
}
