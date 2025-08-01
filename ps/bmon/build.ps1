<#
    This build script uses PS2EXE to conver bmon PowerShell script into an executable.
#>
Write-Host "bmon.ps1 -> bmon.exe..." -ForegroundColor Cyan
Invoke-ps2exe "bmon.ps1" "bmon.exe" `
    -iconFile "../../btc/icons/bitcoin_small.ico" `
    -title "Bitcoin Monitor" `
    -description "Lightweight BTC Price Monitor" `
    -product "Bmon" `
    -company "kreft.us" `
    -copyright "Copyright (c) 2025" `
    -version "1.4.0.0"
Write-Host "Build complete!" -ForegroundColor Green