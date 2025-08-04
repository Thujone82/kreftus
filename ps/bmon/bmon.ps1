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
    Date: 2025-08-02@1058
    Version: 1.4
#>

[CmdletBinding(DefaultParameterSetName='Monitor')]
param (
    [Parameter(ParameterSetName='Monitor')]
    [switch]$go,

    [Parameter(ParameterSetName='Monitor')]
    [switch]$golong,

    [Parameter(ParameterSetName='Monitor')]
    [Alias('history')]
    [switch]$h,

    [Parameter(ParameterSetName='Monitor')]
    [Alias('sound')]
    [switch]$s,

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
    param ([string]$ApiKey)

    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
    
    $maxAttempts = 3
    $baseDelaySeconds = 2

    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        try {
            $response = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -ErrorAction Stop
            
            $price = $response.rate -as [double] 
            
            if ($null -eq $price) {
                # Use Write-Host instead of Write-Warning to maintain display consistency
                $warningMsg = "API returned a non-numeric rate: '$($response.rate)'"
                Write-Host -ForegroundColor Yellow "`r$warningMsg$(' ' * ([System.Console]::WindowWidth - $warningMsg.Length - 1))`r" -NoNewline
            }
            return $price
        }
        catch {
            if ($attempt -ge $maxAttempts) {
                # Use Write-Host instead of Write-Warning to maintain display consistency
                $warningMsg = "API call failed after $maxAttempts attempts: $($_.Exception.Message)"
                Write-Host -ForegroundColor Yellow "`r$warningMsg$(' ' * ([System.Console]::WindowWidth - $warningMsg.Length - 1))`r" -NoNewline
                return $null
            }
            
            # Exponential backoff with jitter
            $jitterMs = Get-Random -Minimum 0 -Maximum 1000
            $backoffSeconds = [math]::Pow(2, $attempt - 1) * $baseDelaySeconds
            $sleepDurationMs = [int]($backoffSeconds * 1000) + $jitterMs
            
            # Use Write-Host instead of Write-Warning to maintain display consistency
            $warningMsg = "API call failed. Retrying in $([math]::Round($sleepDurationMs/1000, 1)) seconds... (Retry $attempt of $($maxAttempts - 1))"
            Write-Host -ForegroundColor Yellow "`r$warningMsg$(' ' * ([System.Console]::WindowWidth - $warningMsg.Length - 1))`r" -NoNewline
            Start-Sleep -Milliseconds $sleepDurationMs
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
    if ($History.Count -lt 2) { return "‖      ‖".PadRight(10) }

    $sparkChars = [char[]](' ', '▂', '▃', '▄', '▅', '▆', '▇', '█')
    $minPrice = ($History | Measure-Object -Minimum).Minimum
    $maxPrice = ($History | Measure-Object -Maximum).Maximum
    $priceRange = $maxPrice - $minPrice

    if ($priceRange -eq 0) { return "‖$([string]$sparkChars[0] * 8)‖".PadRight(10) }

    $sparkline = ""
    foreach ($price in $History) {
        $normalized = ($price - $minPrice) / $priceRange
        $charIndex = [math]::Floor($normalized * ($sparkChars.Length - 1))
        $sparkline += $sparkChars[$charIndex]
    }
    # Pad to 8 characters on the left if needed (to make new updates appear from the right)
    if ($sparkline.Length -lt 8) {
        $sparkline = $sparkline.PadLeft(8)
    }
    # Truncate to 8 characters if needed (keep the most recent data on the right)
    if ($sparkline.Length -gt 8) {
        $sparkline = $sparkline.Substring($sparkline.Length - 8)
    }
    return "‖$sparkline‖".PadRight(10)
}

# --- Main Script ---

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


# --- Conversion Logic Branch ---
if ($PSCmdlet.ParameterSetName -ne 'Monitor') {
    $price = Get-BtcPrice -ApiKey $apiKey
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
            if ($price -eq 0) { Write-Error "Bitcoin price is zero, cannot divide."; exit 1 }
            $btcValue = $UsdToBitcoin / $price
            Write-Host ("B{0}" -f $btcValue.ToString("F8"))
        }
        "USDToSats" {
            if ($price -eq 0) { Write-Error "Bitcoin price is zero, cannot divide."; exit 1 }
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
if ($go.IsPresent -or $golong.IsPresent) {
    Clear-Host
    Write-Host -NoNewline "Fetching initial price...`r" -ForegroundColor Cyan
} else {
    Write-Host "Fetching initial price..." -ForegroundColor Cyan
}
$currentBtcPrice = Get-BtcPrice -ApiKey $apiKey
if ($null -eq $currentBtcPrice) {
    Write-Host "Failed to fetch initial price. Check API key or network." -ForegroundColor Red
    exit
}

# --- Main Logic Branch ---

if ($go.IsPresent -or $golong.IsPresent) {
    # --- Mode Configuration ---
    $modeSettings = @{
        'go'     = @{ duration = 900;   interval = 5;  spinner = @('⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏') }
        'golong' = @{ duration = 86400; interval = 20; spinner = @('*') }
    }
    $currentMode = if ($golong.IsPresent) { 'golong' } else { 'go' }

    # --- Initial State Setup ---
    $monitorStartPrice = $currentBtcPrice
    $previousIntervalPrice = $currentBtcPrice
    $previousPriceColor = "White"
    $monitorStartTime = Get-Date
    $soundEnabled = $s.IsPresent
    $sparklineEnabled = $h.IsPresent
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
            $clearLine = "`r" + (' ' * ([System.Console]::WindowWidth - 1)) + "`r"
            Write-Host -NoNewline $clearLine
        }
        catch {
            Write-Host "`n"
        }
        exit
    }

    try {
        [System.Console]::CursorVisible = $false
        while ($true) {
            $monitorDurationSeconds = $modeSettings[$currentMode].duration
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
            if (((Get-Date) - $monitorStartTime).TotalSeconds -ge $monitorDurationSeconds) { break }

            $waitStart = Get-Date
            $refreshed = $false
            $modeSwitched = $false
            $isFirstTick = $true
            while (((Get-Date) - $waitStart).TotalSeconds -lt $waitIntervalSeconds) {
                $spinnerChar = $spinner[$spinnerIndex]
                $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { "" }
                $restOfLine = " Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString$sparklineString"

                if ($currentMode -eq 'go') {
                    $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
                }

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
                    if ($keyInfo.KeyChar -eq 'r') {
                        $monitorStartPrice = $currentBtcPrice
                        $monitorStartTime = Get-Date
                        $refreshed = $true
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

            if ($currentMode -eq 'go') {
                $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
            }
            $spinnerChar = $spinner[$spinnerIndex]
            $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { "" }
            $restOfLine = " Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString$sparklineString"
            
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
                if ($priceHistory.Count -gt 8) {
                    $priceHistory.RemoveAt(0)
                }
            }
        }
    }
    finally {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor()
        try {
            $clearLine = "`r" + (' ' * ([System.Console]::WindowWidth - 1)) + "`r"
            Write-Host -NoNewline $clearLine
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
        Write-Host -NoNewline "Start[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;

        while ($true) {
            if ([System.Console]::KeyAvailable) {
                if (([System.Console]::ReadKey($true)).Key -eq 'Spacebar') { break }
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
            param([boolean]$InvertColors)

            Clear-Host
            Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow

            $sparklineString = if ($sparklineEnabled) { " $(Get-Sparkline -History $priceHistory)" } else { "" }
            $priceLine = "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString$sparklineString"

            $fgColor = if ($InvertColors) { "Black" } else { $priceColor }
            $bgColor = if ($InvertColors) { $priceColor } else { [System.Console]::BackgroundColor }

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
            
            if ($flashNeeded) { & $drawScreen -InvertColors $true; Start-Sleep -Milliseconds 500 }
            & $drawScreen -InvertColors $false

            $waitStart = Get-Date
            $paused = $false
            $refreshed = $false
            while (((Get-Date) - $waitStart).TotalSeconds -lt 5) {
                if ([System.Console]::KeyAvailable) {
                    $keyInfo = [System.Console]::ReadKey($true)
                    if ($keyInfo.Key -eq 'Spacebar') {
                        $paused = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 'r') {
                        $refreshed = $true
                        break
                    }
                    if ($keyInfo.KeyChar -eq 's') {
                        Invoke-SoundToggle -SoundEnabled ([ref]$soundEnabled)
                    }
                    if ($keyInfo.KeyChar -eq 'h') {
                        $sparklineEnabled = -not $sparklineEnabled
                        & $drawScreen -InvertColors $false
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
                if ($priceHistory.Count -gt 8) {
                    $priceHistory.RemoveAt(0)
                }
            }
        } while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds)
    }
}
