<#
    This build script uses PS2EXE to conver vBTC PowerShell script into an executable.
#>
Write-Host "vbtc.ps1 -> vbtc.exe..." -ForegroundColor Cyan
Invoke-ps2exe "vbtc.ps1" "vbtc.exe" `
    -iconFile "../../btc/icons/bitcoin_small.ico" `
    -title "vBTC - Virtual Bitcoin Trading Simulator" `
    -description "An interactive PowerShell-based Bitcoin trading application." `
    -product "vBTC" `
    -company "kreft.us" `
    -copyright "Copyright (c) 2025" `
    -version "1.5.0.0"
Write-Host "bmon.ps1 -> bmon.exe..." -ForegroundColor Cyan
Invoke-ps2exe "bmon.ps1" "bmon.exe" `
    -iconFile "../../btc/icons/bitcoin_small.ico" `
    -title "Bitcoin Monitor" `
    -description "Lightweight BTC Price Monitor." `
    -product "Bmon" `
    -company "kreft.us" `
    -copyright "Copyright (c) 2025" `
    -version "1.0.0.0"
Write-Host "Build complete!" -ForegroundColor Green