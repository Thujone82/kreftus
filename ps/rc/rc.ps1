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

.PARAMETER Silent
    A switch to enable "Silent Mode". When enabled, the script suppresses status
    output messages such as execution timing and wait periods, while still
    displaying the actual command output and any errors.
    Alias: -s

.PARAMETER Clear
    A switch to enable "Clear Mode". When enabled, the script clears the screen
    before executing the command in each iteration, providing a clean output for
    each run.
    Alias: -c

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

.EXAMPLE
    .\rc.ps1 "Get-Date" 1 -Silent

    Runs 'Get-Date' every minute in silent mode, suppressing status messages while still showing the date output.

.EXAMPLE
    .\rc.ps1 "Get-Date" 1 -Clear

    Runs 'Get-Date' every minute with the screen cleared before each execution, providing a clean output display.

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
    [switch]$Precision,

    [Parameter(Mandatory=$false, HelpMessage="Enables silent mode to suppress status output messages.")]
    [Alias('s')]
    [switch]$Silent,

    [Parameter(Mandatory=$false, HelpMessage="Clears the screen before executing the command in each iteration.")]
    [Alias('cl', 'c')]
    [switch]$Clear
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
    $inputClear = Read-Host "Enable Clear Mode? (y/n) [default: n]"
    if ($inputClear.ToLower() -eq 'y') {
        $Clear = $true
    }
}

# Clear screen if requested - must be done before any output
# Check parameter in multiple ways to ensure it's recognized
if ($PSBoundParameters.ContainsKey('Clear') -or $Clear.IsPresent -or $Clear) {
    Clear-Host
}

if (-not $Silent.IsPresent) {
    Write-Host "Running `"$Command`" every $Period minute(s). Press Ctrl+C to stop.`n"
}
$scriptStartTime = Get-Date
if ($Precision.IsPresent -and -not $Silent.IsPresent) {
    Write-Host "Precision mode is enabled. Aligning to grid starting at $($scriptStartTime.ToString('HH:mm:ss'))." -ForegroundColor Cyan
}

while ($true) {
    $loopStartTime = Get-Date
    try {
        if ($Clear -or $Clear.IsPresent) {
            try {
                [Console]::Clear()
            } catch {
                try {
                    Clear-Host
                } catch {
                    # Fallback: output ANSI escape sequence
                    Write-Host "`e[2J`e[H" -NoNewline
                }
            }
        }
        if (-not $Silent.IsPresent) {
            Write-Host "($(Get-Date -Format 'HH:mm:ss')) Executing command..."
        }
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
            if (-not $Silent.IsPresent) {
                Write-Host "Command took $($commandDuration.TotalSeconds.ToString('F2'))s. Waiting for $([math]::Round($sleepTimeSpan.TotalSeconds, 0))s. Next run at $($nextTargetTime.ToString('HH:mm:ss')).`nPress Ctrl+C to stop."
            }
            Start-Sleep -Seconds $sleepTimeSpan.TotalSeconds
        } else {
            if (-not $Silent.IsPresent) {
                Write-Warning "Command execution time ($($commandDuration.TotalSeconds.ToString('F2'))s) overran its schedule. Running next iteration immediately."
            }
        }
    } else {
        # Original behavior
        if (-not $Silent.IsPresent) {
            Write-Host "Waiting $Period minute(s). Press Ctrl+C to stop.`n"
        }
        Start-Sleep -Seconds ($Period * 60)
    }
}
