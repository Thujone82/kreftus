<#
.SYNOPSIS
    Runs a specified command repeatedly at a given interval.

.DESCRIPTION
    This script, rc.ps1 (Run Continuously), executes a given PowerShell command string in a loop.
    After each execution, it waits for a specified number of minutes before running the command again.
    The script will continue to run until manually stopped with Ctrl+C.

    By default, the wait period starts after the command finishes. For more accurate scheduling,
    use the -Precision switch. This will account for the command's execution time to ensure
    each run starts at a consistent interval.

    If no parameters are provided, the script will interactively prompt the user for the command and the time period.

.PARAMETER Command
    The PowerShell command to execute on each iteration. This should be a string.
    If the command contains spaces, it should be enclosed in quotes.

.PARAMETER Period
    The time to wait between command executions, in minutes.
    The default value is 5 minutes.

.PARAMETER Precision
    A switch to enable "Precision Mode". When enabled, the script accounts for the
    command's execution time to ensure that each new execution starts at a precise
    interval from the start of the previous one. This prevents timing drift.
    Alias: -p

.EXAMPLE
    .\rc.ps1 "Get-Process -Name 'chrome' | Stop-Process -Force" 1
    
    This command will attempt to stop all 'chrome' processes every 1 minute.

.EXAMPLE
    .\rc.ps1 "gw Portland" 10

    Runs the gw.ps1 script with its own parameter every 10 minutes.

.EXAMPLE
    .\rc.ps1 ".\my-data-logger.ps1" 10 -Precision

    Runs 'my-data-logger.ps1' on a fixed 10-minute schedule based on the script's start time.
    If a run starts at 10:00:00 and takes 20 seconds, the next run will be scheduled for exactly 10:10:00.
    If a run takes 11 minutes, it will finish late, and the script will immediately start the next run to get back on schedule.

.NOTES
    To stop the script, press Ctrl+C in the terminal window where it is running.
#>
param(
    [Parameter(Position=0)]
    [string]$Command,

    [Parameter(Position=1)]
    [int]$Period = 5,

    [Parameter(Mandatory=$false, HelpMessage="Enables precision mode to account for command execution time.")]
    [Alias('p')]
    [switch]$Precision
)

if (-not $Command) {
    Write-Host "*** Run Continuously v1 ***" -ForegroundColor Yellow
    $Command = Read-Host "Command"
    $inputPeriod = Read-Host "Period (minutes) [default: 5]"
    if ($inputPeriod -and [int]::TryParse($inputPeriod, [ref]$null)) {
        $Period = [int]$inputPeriod
    } else {
        $Period = 5
    }
    $inputPrecision = Read-Host "Enable Precision Mode? (y/n) [default: n]"
    if ($inputPrecision.ToLower() -eq 'y') {
        $Precision = $true
    }
}

Write-Host "Running `"$Command`" every $Period minute(s). Press Ctrl+C to stop.`n"
$scriptStartTime = Get-Date
if ($Precision.IsPresent) {
    Write-Host "Precision mode is enabled. Aligning to grid starting at $($scriptStartTime.ToString('HH:mm:ss'))." -ForegroundColor Cyan
}

while ($true) {
    $loopStartTime = Get-Date
    try {
        Write-Host "($(Get-Date -Format 'HH:mm:ss')) Executing command..."
        Invoke-Expression $Command
    }
    catch {
        Write-Warning "Command failed: $_"
    }

    if ($Precision.IsPresent) {
        $currentTime = Get-Date
        $commandDuration = $currentTime - $loopStartTime

        # Calculate the next scheduled run time based on the script's start time (grid alignment)
        $totalElapsedMinutes = ($currentTime - $scriptStartTime).TotalMinutes
        $intervalsCompleted = [math]::Floor($totalElapsedMinutes / $Period)
        $nextTargetTime = $scriptStartTime.AddMinutes(($intervalsCompleted + 1) * $Period)

        $sleepTimeSpan = $nextTargetTime - $currentTime

        if ($sleepTimeSpan.TotalSeconds -gt 0) {
            Write-Host "Command took $($commandDuration.TotalSeconds.ToString('F2'))s. Waiting for $([math]::Round($sleepTimeSpan.TotalSeconds, 0))s. Next run at $($nextTargetTime.ToString('HH:mm:ss')).`nPress Ctrl+C to stop."
            Start-Sleep -Seconds $sleepTimeSpan.TotalSeconds
        } else {
            Write-Warning "Command execution time ($($commandDuration.TotalSeconds.ToString('F2'))s) overran its schedule. Running next iteration immediately."
        }
    } else {
        # Original behavior
        Write-Host "Waiting $Period minute(s). Press Ctrl+C to stop.`n"
        Start-Sleep -Seconds ($Period * 60)
    }
}
