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
    Date: 2025-08-01
    Version: 1.3
#>

[CmdletBinding(DefaultParameterSetName='Monitor')]
param (
    [Parameter(ParameterSetName='Monitor')]
    [switch]$go,

    [Parameter(ParameterSetName='Monitor')]
    [switch]$golong,

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

function Invoke-SoundToggle {
    param([ref]$SoundEnabled)
    $SoundEnabled.Value = -not $SoundEnabled.Value
    if ($SoundEnabled.Value) { # Sound was just turned ON
        [System.Console]::Beep(1200, 350) # High tone
    }
    else { # Sound was just turned OFF
        [System.Console]::Beep(400, 350) # Low tone
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


# Initial Price Fetch for monitor modes
if ($go.IsPresent -or $golong.IsPresent) {
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

if ($go.IsPresent -or $golong.IsPresent) {
    # --- Mode Configuration ---
    $modeSettings = @{
        'go'     = @{ duration = 300;   interval = 5;  spinner = @('|', '/', '-', '\') }
        'golong' = @{ duration = 86400; interval = 20; spinner = @('*') }
    }
    $currentMode = if ($golong.IsPresent) { 'golong' } else { 'go' }

    # --- Initial State Setup ---
    $monitorStartPrice = $currentBtcPrice
    $previousIntervalPrice = $currentBtcPrice
    $previousPriceColor = "White"
    $monitorStartTime = Get-Date
    $soundEnabled = $false

    # These will be updated by the loop based on the current mode
    $monitorDurationSeconds = 0
    $waitIntervalSeconds = 0
    $spinner = @()
    $spinnerIndex = 0
 
    # Trap the Ctrl+C event (PipelineStoppedException) for a clean exit.
    # This has a better chance of running before the shell writes "^C" than a finally block.
    trap [System.Management.Automation.PipelineStoppedException] {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor()
        try {
            # Attempt to perfectly clear the line (works in standard PowerShell)
            $clearLine = "`r" + (' ' * ([System.Console]::WindowWidth - 1)) + "`r"
            [System.Console]::Write($clearLine)
        }
        catch {
            # Fallback for ps2exe where the above fails. Just add a newline.
            [System.Console]::Write("`n")
        }
        exit
    }

    try {
        [System.Console]::CursorVisible = $false
        while ($true) { # Loop indefinitely until duration is met or Ctrl+C
            # Set parameters based on the current mode
            $monitorDurationSeconds = $modeSettings[$currentMode].duration
            $waitIntervalSeconds = $modeSettings[$currentMode].interval
            $spinner = $modeSettings[$currentMode].spinner

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
            # Exit if duration is met for the current mode
            if (((Get-Date) - $monitorStartTime).TotalSeconds -ge $monitorDurationSeconds) { break }

            # Wait for interval, check for keys, and animate the spinner
            $waitStart = Get-Date
            $refreshed = $false
            $modeSwitched = $false
            $isFirstTick = $true # Flag for the first 500ms flash
            while (((Get-Date) - $waitStart).TotalSeconds -lt $waitIntervalSeconds) {
                # Animate spinner and draw the line
                $spinnerChar = $spinner[$spinnerIndex]
                $restOfLine = " Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString"

                # For 'go' mode, the spinner index animates. For 'golong', it's always 0.
                if ($currentMode -eq 'go') {
                    $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
                }

                # Use a try/finally block to ensure console colors are always reset
                $oldFgColor = [System.Console]::ForegroundColor
                $oldBgColor = [System.Console]::BackgroundColor
                try {
                    if ($flashNeeded -and $isFirstTick) {
                        # Inverted flash colors for the first 500ms (spinner and text)
                        [System.Console]::ForegroundColor = "Black"
                        [System.Console]::BackgroundColor = $priceColor
                        $fullLine = ($spinnerChar + $restOfLine).PadRight([System.Console]::WindowWidth)
                        [System.Console]::Write("$fullLine`r")
                    }
                    elseif ($currentMode -eq 'golong') {
                        # For -golong, the spinner is always white unless fetching (handled later)
                        [System.Console]::ForegroundColor = "White"
                        [System.Console]::Write($spinnerChar)
                        [System.Console]::ForegroundColor = $priceColor
                        [System.Console]::Write($restOfLine)
                        $lineLength = $spinnerChar.Length + $restOfLine.Length
                        $padding = [System.Console]::WindowWidth - $lineLength
                        if ($padding -gt 0) {
                            [System.Console]::Write(' ' * $padding)
                        }
                        [System.Console]::Write("`r")
                    }
                    else { # Normal 'go' mode drawing
                        # Normal colors: Spinner is always white during the 5-second wait.
                        $spinnerColor = "White"
                        [System.Console]::ForegroundColor = $spinnerColor
                        [System.Console]::Write($spinnerChar)

                        [System.Console]::ForegroundColor = $priceColor
                        [System.Console]::Write($restOfLine)

                        # Pad the rest of the line to clear previous, longer lines
                        $lineLength = $spinnerChar.Length + $restOfLine.Length
                        $padding = [System.Console]::WindowWidth - $lineLength
                        if ($padding -gt 0) {
                            [System.Console]::Write(' ' * $padding)
                        }
                        [System.Console]::Write("`r")
                    }
                }
                finally {
                    [System.Console]::ForegroundColor = $oldFgColor
                    [System.Console]::BackgroundColor = $oldBgColor
                }
                $isFirstTick = $false # The flash has occurred (or not), subsequent ticks are normal

                # Check for refresh or mode toggle key
                if ([System.Console]::KeyAvailable) {
                    $keyInfo = [System.Console]::ReadKey($true)
                    if ($keyInfo.KeyChar -eq 'r') {
                        # Reset the comparison point to the current price without a new API call.
                        $monitorStartPrice = $currentBtcPrice
                        $monitorStartTime = Get-Date
                        $refreshed = $true
                        break # Exit wait loop
                    }
                    if ($keyInfo.KeyChar -eq 'm') {
                        # Toggle mode
                        $currentMode = if ($currentMode -eq 'go') { 'golong' } else { 'go' }
                        # Reset timer for the new mode's duration
                        $monitorStartTime = Get-Date
                        $monitorStartPrice = $currentBtcPrice # Also reset start price
                        $modeSwitched = $true
                        # Reset spinner index to avoid out-of-bounds on array change
                        $spinnerIndex = 0
                        break # Exit wait loop to apply new settings
                    }
                    if ($keyInfo.KeyChar -eq 's') {
                        Invoke-SoundToggle -SoundEnabled ([ref]$soundEnabled)
                    }
                }
                Start-Sleep -Milliseconds 500
            }
            if ($refreshed -or $modeSwitched) { continue } # Continue main monitoring loop to redraw immediately

            # Update state for the next interval
            $previousIntervalPrice = $currentBtcPrice
            $previousPriceColor = $priceColor

            # Redraw the line with a cyan spinner to indicate an active API fetch is about to happen.
            # This provides a seamless visual cue without clearing the screen.
            if ($currentMode -eq 'go') {
                $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
            } # For golong, index is always 0
            $spinnerChar = $spinner[$spinnerIndex]
            $restOfLine = " Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString"

            $oldFg = [System.Console]::ForegroundColor
            $oldBg = [System.Console]::BackgroundColor
            try {
                # Draw the line with a Cyan spinner to indicate fetching
                [System.Console]::ForegroundColor = "Cyan"
                [System.Console]::Write($spinnerChar)

                [System.Console]::ForegroundColor = $priceColor
                [System.Console]::Write($restOfLine)

                # Pad the rest of the line to clear previous, longer lines
                $lineLength = $spinnerChar.Length + $restOfLine.Length
                $padding = [System.Console]::WindowWidth - $lineLength
                if ($padding -gt 0) {
                    [System.Console]::Write(' ' * $padding)
                }
                [System.Console]::Write("`r")
            }
            finally {
                [System.Console]::ForegroundColor = $oldFg
                [System.Console]::BackgroundColor = $oldBg
            }

            # Fetch the next price for the next iteration
            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) {
                if ($soundEnabled) {
                    if ($newPrice -gt $currentBtcPrice) { [System.Console]::Beep(1200, 150) } # High tone
                    elseif ($newPrice -lt $currentBtcPrice) { [System.Console]::Beep(400, 150) } # Low tone
                }
                $currentBtcPrice = $newPrice
            }
        }
    }
    finally {
        [System.Console]::CursorVisible = $true
        [System.Console]::ResetColor() # Ensure background is reset on exit
        # On exit, we try to perform a "perfect clear" of the line. This works reliably
        # in a standard PowerShell terminal.
        try {
            # Move cursor to the start, overwrite the line with spaces, then move back.
            $clearLine = "`r" + (' ' * ([System.Console]::WindowWidth - 1)) + "`r"
            [System.Console]::Write($clearLine)
        }
        catch {
            # In the ps2exe host, the 'try' block can fail. This fallback simply
            # writes a newline. It's less tidy (leaves the last price line visible)
            # but reliably prevents the prompt from overwriting the output.
            [System.Console]::Write("`n")
        }
    }
    exit
}
else {
    # Interactive Mode (original behavior)
    $soundEnabled = $false
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

        # Fetch a fresh price to set an accurate baseline for this monitoring session.
        Write-Host "`nStarting monitoring..." -ForegroundColor Cyan
        $newPrice = Get-BtcPrice -ApiKey $apiKey
        if ($null -ne $newPrice) { $currentBtcPrice = $newPrice }

        # Set the baseline for this monitoring session
        $monitorStartPrice = $currentBtcPrice
        $monitorStartTime = Get-Date

        # Reset the flash-tracking state for this new session. This ensures the first
        # display is always neutral (white) and becomes the new comparison point.
        $previousIntervalPrice = $currentBtcPrice
        $previousPriceColor = "White"
        $monitorDurationSeconds = 300 # 5 minutes

        # Monitoring State Loop (do-while style)
        do {
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
                Write-Host -NoNewline "Pause[" -ForegroundColor White; Write-Host -NoNewline "Space" -ForegroundColor Cyan; Write-Host -NoNewline "], Reset[" -ForegroundColor White; Write-Host -NoNewline "R" -ForegroundColor Cyan; Write-Host -NoNewline "], Sound[" -ForegroundColor White; Write-Host -NoNewline "S" -ForegroundColor Cyan; Write-Host -NoNewline "], Exit[" -ForegroundColor White; Write-Host -NoNewline "Ctrl+C" -ForegroundColor Cyan; Write-Host "]" -ForegroundColor White;
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
                    if ($keyInfo.KeyChar -eq 's') {
                        Invoke-SoundToggle -SoundEnabled ([ref]$soundEnabled)
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

            # Fetch the next price for the next iteration
            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) {
                if ($soundEnabled) {
                    if ($newPrice -gt $currentBtcPrice) { [System.Console]::Beep(1200, 150) } # High tone
                    elseif ($newPrice -lt $currentBtcPrice) { [System.Console]::Beep(400, 150) } # Low tone
                }
                $currentBtcPrice = $newPrice
            }
        } while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds)
    }
}