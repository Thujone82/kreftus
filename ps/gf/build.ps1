<#
    This build script uses PS2EXE to convert gf PowerShell script into an executable.
#>
Write-Host "gf.ps1 -> gf.exe..." -ForegroundColor Cyan
Invoke-ps2exe "gf.ps1" "gf.exe" `
    -iconFile "gf.ico" `
    -title "GetForecast" `
    -description "Terminal Weather Application" `
    -product "GF" `
    -version "2.7.0.0" `
    -company "kreft.us" `
    -copyright "Copyright (c) 2025" 
Write-Host "Build complete!" -ForegroundColor Green