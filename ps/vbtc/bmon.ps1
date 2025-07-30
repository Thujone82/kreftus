<#
.SYNOPSIS
    A lightweight, real-time Bitcoin price monitor.

.DESCRIPTION
    This script provides a simple command-line interface to monitor the real-time price of Bitcoin.
    It reads the API key from the vBTC configuration file (vbtc.ini).
 
    It can be run in two modes:
    - Interactive Mode: The main screen displays the current price. Press the space bar to start/pause monitoring.
    - Continuous Mode (`-go`): Monitors the price immediately for 5 minutes and then exits.
 
    In both modes, the price line will flash with an inverted color for 500ms to draw attention to significant price movements. This flash occurs when the price color changes (e.g., from neutral to green) or when the price continues to move in an already established direction (e.g., ticking up again while a green).
 
    This script requires the vbtc.ini file from the vBTC application to be configured with a
    valid LiveCoinWatch API key.

.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-07-28
    Version: 1.1
#>

[CmdletBinding(DefaultParameterSetName='Monitor')]
param (
    [Parameter(ParameterSetName='Monitor')]
    [switch]$go,

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

function Get-BtcPrice {
    param ([string]$ApiKey)
    
    $headers = @{ "Content-Type" = "application/json"; "x-api-key" = $ApiKey }
    $body = @{ currency = "USD"; code = "BTC"; meta = $false } | ConvertTo-Json
    try {
        $response = Invoke-RestMethod -Uri "https://api.livecoinwatch.com/coins/single" -Method Post -Headers $headers -Body $body -ErrorAction Stop
        return $response.rate
    }
    catch {
        Write-Warning "API call failed: $($_.Exception.Message)"
        return $null
    }
}

# --- Main Script ---

$config = Get-IniConfiguration -FilePath $iniFilePath
$apiKey = $config.Settings.ApiKey

if ([string]::IsNullOrEmpty($apiKey)) {
    Write-Host "Configure vBTC first..." -ForegroundColor Red
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


# Initial Price Fetch
if ($go.IsPresent) {
    Clear-Host
    # For -go mode, write on one line so it can be overwritten by the first price update.
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

if ($go.IsPresent) {
    # "Go" Mode - Monitor immediately and exit after 5 minutes
    $monitorStartPrice = $currentBtcPrice
    $previousIntervalPrice = $currentBtcPrice
    $previousPriceColor = "White"
    $monitorStartTime = Get-Date
    $monitorDurationSeconds = 300 # 5 minutes
    $spinner = @('|', '/', '-', '\')
    $spinnerIndex = 0
 
    try {
        [System.Console]::CursorVisible = $false
        while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds) {
            # Calculate change and determine current color
            $priceChange = $currentBtcPrice - $monitorStartPrice
            $priceColor = "White"
            $changeString = ""
            if ($priceChange -ge 0.01) {
                $priceColor = "Green"
                $changeString = " [+$($priceChange.ToString("C2"))]"
            } elseif ($priceChange -le -0.01) {
                $priceColor = "Red"
                $changeString = " [$($priceChange.ToString("C2"))]" # Negative sign is included by ToString("C2")
            }

            # Determine if a flash is needed based on the plan
            $flashNeeded = $false
            # Condition 1: Color has changed from its previous state (and isn't just neutral)
            if ($priceColor -ne "White" -and $priceColor -ne $previousPriceColor) {
                $flashNeeded = $true
            }
            # Condition 2: Price continues to move in the same direction
            elseif (($priceColor -eq "Green" -and $currentBtcPrice -gt $previousIntervalPrice) -or
                    ($priceColor -eq "Red" -and $currentBtcPrice -lt $previousIntervalPrice)) {
                $flashNeeded = $true
            }

            # Wait for 5 seconds, check for 'r' key, and animate the spinner
            $waitStart = Get-Date
            $refreshed = $false
            $isFirstTick = $true # Flag for the first 500ms flash
            while (((Get-Date) - $waitStart).TotalSeconds -lt 5) {
                # Animate spinner and draw the line
                $spinnerChar = $spinner[$spinnerIndex]
                $line = "$spinnerChar Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString"
                $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
                $paddedLine = $line.PadRight([System.Console]::WindowWidth)

                # Use a try/finally block to ensure console colors are always reset
                $oldFgColor = [System.Console]::ForegroundColor
                $oldBgColor = [System.Console]::BackgroundColor
                try {
                    if ($flashNeeded -and $isFirstTick) {
                        # Inverted flash colors for the first 500ms
                        [System.Console]::ForegroundColor = "Black"
                        [System.Console]::BackgroundColor = $priceColor
                    } else {
                        # Normal colors for the rest of the interval
                        [System.Console]::ForegroundColor = $priceColor
                        [System.Console]::BackgroundColor = "Black" # Assuming default is black
                    }
                    [System.Console]::Write("$paddedLine`r")
                }
                finally {
                    [System.Console]::ForegroundColor = $oldFgColor
                    [System.Console]::BackgroundColor = $oldBgColor
                }
                $isFirstTick = $false # The flash has occurred (or not), subsequent ticks are normal

                # Check for refresh key
                if ([System.Console]::KeyAvailable -and (([System.Console]::ReadKey($true)).KeyChar -eq 'r')) {
                    # Reset the comparison point to the current price without a new API call.
                    $monitorStartPrice = $currentBtcPrice
                    $monitorStartTime = Get-Date
                    $refreshed = $true
                    break # Exit wait loop
                }
                Start-Sleep -Milliseconds 500
            }
            if ($refreshed) { continue } # Continue main monitoring loop to redraw immediately

            # Update state for the next interval
            $previousIntervalPrice = $currentBtcPrice
            $previousPriceColor = $priceColor

            # Fetch the next price for the next iteration
            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) { $currentBtcPrice = $newPrice }
        }
    }
    finally {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor() # Ensure background is reset on exit
        try {
            # Ideal cleanup for a standard console. This may fail in the PS2EXE host.
            # Move to the beginning of the current line.
            [System.Console]::SetCursorPosition(0, [System.Console]::CursorTop)
            # Overwrite the entire line with spaces to clear any leftover characters.
            [System.Console]::Write(' ' * ([System.Console]::WindowWidth - 1))
            # Move the cursor back to the beginning of the now-blank line, ready for the shell prompt.
            [System.Console]::SetCursorPosition(0, [System.Console]::CursorTop)
        } catch {
            # Fallback for non-interactive/compiled environments where the above might fail.
            # A direct, unbuffered write of a newline is the most reliable way
            # to ensure the next shell prompt appears on a fresh line.
            [System.Console]::Write("`n")
        }
    }
    exit
}
else {
    # Interactive Mode (original behavior)
    # State tracking for the flash feature
    $previousIntervalPrice = $currentBtcPrice
    $previousPriceColor = "White"

    while ($true) {
        # Paused State Display
        Clear-Host
        Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow
        Write-Host "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))" -ForegroundColor White
        Write-Host -NoNewline "Start[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;

        # Wait for user to start monitoring
        while ($true) {
            if ([System.Console]::KeyAvailable) {
                if (([System.Console]::ReadKey($true)).Key -eq 'Spacebar') { break }
            }
            Start-Sleep -Milliseconds 100
        }

        # Mark the price when monitoring starts

        # Monitoring State Loop
        $monitorStartPrice = $currentBtcPrice # Reset baseline for this monitoring session
        $monitorStartTime = Get-Date
        $monitorDurationSeconds = 300 # 5 minutes

        while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds) {
            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) { $currentBtcPrice = $newPrice }

            # Calculate change and determine color
            $priceChange = $currentBtcPrice - $monitorStartPrice
            $priceColor = "White"
            $changeString = ""
            if ($priceChange -ge 0.01) {
                $priceColor = "Green"
                $changeString = " [+$($priceChange.ToString("C2"))]"
            } elseif ($priceChange -le -0.01) {
                $priceColor = "Red"
                $changeString = " [$($priceChange.ToString("C2"))]" # Negative sign is included by ToString("C2")
            }

            # Determine if a flash is needed based on the plan
            $flashNeeded = $false
            # Condition 1: Color has changed from its previous state (and isn't just neutral)
            if ($priceColor -ne "White" -and $priceColor -ne $previousPriceColor) {
                $flashNeeded = $true
            }
            # Condition 2: Price continues to move in the same direction
            elseif (($priceColor -eq "Green" -and $currentBtcPrice -gt $previousIntervalPrice) -or
                    ($priceColor -eq "Red" -and $currentBtcPrice -lt $previousIntervalPrice)) {
                $flashNeeded = $true
            }

            # --- Screen Drawing with Flash Logic ---
            # Reusable block to draw the screen content
            $drawScreen = {
                param([boolean]$InvertColors)

                Clear-Host
                Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow

                $fgColor = if ($InvertColors) { "Black" } else { $priceColor }
                $bgColor = if ($InvertColors) { $priceColor } else { [System.Console]::BackgroundColor }

                $oldFg = [System.Console]::ForegroundColor
                $oldBg = [System.Console]::BackgroundColor
                try {
                    [System.Console]::ForegroundColor = $fgColor
                    [System.Console]::BackgroundColor = $bgColor
                    Write-Host "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString"
                }
                finally {
                    [System.Console]::ForegroundColor = $oldFg
                    [System.Console]::BackgroundColor = $oldBg
                }
                Write-Host -NoNewline "Pause[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Reset[" -ForegroundColor White; Write-Host -NoNewline "R" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;
            }

            if ($flashNeeded) { & $drawScreen -InvertColors $true; Start-Sleep -Milliseconds 500 }
            & $drawScreen -InvertColors $false

            # Wait 5 seconds, but allow pausing by checking for key presses
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
                }
                Start-Sleep -Milliseconds 100
            }
            if ($paused) { break } # Exit monitoring loop if user paused
            if ($refreshed) {
                # Reset the comparison point to the current price without a new API call.
                # The main loop will fetch a new price on its next iteration.
                $monitorStartPrice = $currentBtcPrice
                $monitorStartTime = Get-Date # Reset timer
                continue # Redraw screen immediately with new baseline
            }

            # Update state for the next interval
            $previousIntervalPrice = $currentBtcPrice
            $previousPriceColor = $priceColor
        }
    }
}