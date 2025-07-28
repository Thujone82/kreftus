<#
.SYNOPSIS
    A lightweight, real-time Bitcoin price monitor.

.DESCRIPTION
    This script provides a simple command-line interface to monitor the real-time price of Bitcoin.
    It reads the API key from the vBTC configuration file (vbtc.ini).

    The main screen displays the current Bitcoin price. The user can press the space bar to
    enter a monitoring mode where the price is updated every 5 seconds for up to 5 minutes.
    Pressing the space bar again will pause the monitoring.

    This script requires the vbtc.ini file from the vBTC application to be configured with a
    valid LiveCoinWatch API key.

.NOTES
    Author: Kreft&Gemini[Gemini 2.5 Pro (preview)]
    Date: 2025-07-06
    Version: 1.0
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
    $apiKey = ""
    if (Test-Path $FilePath) {
        # Efficiently find the ApiKey line without parsing the whole file
        $apiKeyLine = Get-Content $FilePath | Select-String -Pattern "^\s*ApiKey\s*="
        if ($apiKeyLine) {
            # Take the first match, split on '=', and get the value
            $apiKey = ($apiKeyLine[0].ToString().Split("=", 2)[1]).Trim()
        }
    }
    return @{ "Settings" = @{ "ApiKey" = $apiKey } }
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
            Write-Host ("₿{0}" -f $btcValue.ToString("F8"))
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
    $monitorStartTime = Get-Date
    $monitorDurationSeconds = 300 # 5 minutes
    $spinner = @('|', '/', '-', '\')
    $spinnerIndex = 0
    Clear-Host
 
    try {
        [System.Console]::CursorVisible = $false
        while (((Get-Date) - $monitorStartTime).TotalSeconds -lt $monitorDurationSeconds) {
            # Calculate change and determine color based on the current price
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

            # Wait for 5 seconds, check for 'r' key, and animate the spinner
            $waitStart = Get-Date
            $refreshed = $false
            while (((Get-Date) - $waitStart).TotalSeconds -lt 5) {
                # Animate spinner and draw the line
                $spinnerChar = $spinner[$spinnerIndex]
                $line = "$spinnerChar Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString"
                $spinnerIndex = ($spinnerIndex + 1) % $spinner.Length
                $paddedLine = $line.PadRight([System.Console]::WindowWidth - 1)
                Write-Host -NoNewline "$paddedLine`r" -ForegroundColor $priceColor

                # Check for refresh key
                if ([System.Console]::KeyAvailable -and (([System.Console]::ReadKey($true)).KeyChar -eq 'r')) {
                    $newPrice = Get-BtcPrice -ApiKey $apiKey
                    if ($null -ne $newPrice) {
                        $currentBtcPrice = $newPrice
                        $monitorStartPrice = $currentBtcPrice
                        $monitorStartTime = Get-Date
                    }
                    $refreshed = $true
                    break # Exit wait loop
                }
                Start-Sleep -Milliseconds 500
            }
            if ($refreshed) { continue } # Continue main monitoring loop to redraw immediately

            # Fetch the next price for the next iteration
            $newPrice = Get-BtcPrice -ApiKey $apiKey
            if ($null -ne $newPrice) { $currentBtcPrice = $newPrice }
        }
    }
    finally {
        [System.Console]::CursorVisible = $true
        # Clean up console after the loop finishes or is interrupted by Ctrl+C
        Write-Host ""
    }
    exit
}
else {
    # Interactive Mode (original behavior)
    while ($true) {
        # Paused State Display
        Clear-Host
        Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow
        Write-Host "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))" -ForegroundColor White
        Write-Host -NoNewline "Press " -ForegroundColor Cyan; Write-Host -NoNewline "Space Bar" -ForegroundColor Yellow; Write-Host " to Monitor..." -ForegroundColor Cyan
        Write-Host "Press Ctrl+C to Exit" -ForegroundColor White

        # Wait for user to start monitoring
        while ($true) {
            if ([System.Console]::KeyAvailable) {
                if (([System.Console]::ReadKey($true)).Key -eq 'Spacebar') { break }
            }
            Start-Sleep -Milliseconds 100
        }

        # Mark the price when monitoring starts
        $monitorStartPrice = $currentBtcPrice

        # Monitoring State Loop
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

            Clear-Host
            Write-Host "*** BTC Monitor ***" -ForegroundColor DarkYellow
            Write-Host "Bitcoin (USD): $($currentBtcPrice.ToString("C2"))$changeString" -ForegroundColor $priceColor
            Write-Host -NoNewline "Press " -ForegroundColor Cyan; Write-Host -NoNewline "Space Bar" -ForegroundColor Yellow; Write-Host -NoNewline " to Pause, " -ForegroundColor Cyan; Write-Host -NoNewline "'r'" -ForegroundColor Yellow; Write-Host " to Reset" -ForegroundColor Cyan
            Write-Host "Press Ctrl+C to Exit" -ForegroundColor White

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
                # Fetch a new price immediately to set the new baseline
                $newPrice = Get-BtcPrice -ApiKey $apiKey
                if ($null -ne $newPrice) {
                    $currentBtcPrice = $newPrice
                    $monitorStartPrice = $currentBtcPrice
                }
                $monitorStartTime = Get-Date # Reset timer
                continue # Redraw screen immediately with new baseline
            }
        }
    }
}