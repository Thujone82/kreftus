<#
.SYNOPSIS
    Runs a specified command repeatedly at a given interval.

.DESCRIPTION
    This script, rc.ps1 (Run Continuously), executes a given PowerShell command string in a loop.
    After each execution, it waits for a specified number of minutes before running the command again.
    The script will continue to run until manually stopped with Ctrl+C.

    If no parameters are provided, the script will interactively prompt the user for the command and the time period.

.PARAMETER Command
    The PowerShell command to execute on each iteration. This should be a string.
    If the command contains spaces, it should be enclosed in quotes.

.PARAMETER Period
    The time to wait between command executions, in minutes.
    The default value is 5 minutes.

.EXAMPLE
    .\rc.ps1 "Get-Process -Name 'chrome' | Stop-Process -Force" 1
    
    This command will attempt to stop all 'chrome' processes every 1 minute.

.EXAMPLE
    .\rc.ps1 "gw Portland" 10

    Runs the gw.ps1 script with its own parameter every 10 minutes.

.NOTES
    To stop the script, press Ctrl+C in the terminal window where it is running.
#>
param(
    [Parameter(Position=0)]
    [string]$Command,

    [Parameter(Position=1)]
    [int]$Period = 5
)

if (-not $Command) {
    $Command = Read-Host "Command"
    $inputPeriod = Read-Host "Period (minutes) [default: 5]"
    if ($inputPeriod -and [int]::TryParse($inputPeriod, [ref]$null)) {
        $Period = [int]$inputPeriod
    } else {
        $Period = 5
    }
}

Write-Host "Running `"$Command`" every $Period minute(s). Press Ctrl+C to stop.`n"

while ($true) {
    try {
        Invoke-Expression $Command
    }
    catch {
        Write-Warning "Command failed: $_"
    }

    # sleep for Period minutes (convert to seconds)
    Write-Host "Waiting $Period minute(s). Press Ctrl+C to stop.`n"
    Start-Sleep -Seconds ($Period * 60)
}
