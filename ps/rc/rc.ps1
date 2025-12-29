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
    The time to wait between command executions. Accepts suffixes: 's' for seconds, 'm' for minutes (optional), 
    'h' for hours. Integers without suffix default to minutes. Examples: 5, 15s, 5m, 1h.
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

.PARAMETER Skip
    The number of initial executions to skip before starting to run the command.
    For example, -Skip 2 will skip the first and second executions, then start
    executing from the third iteration onwards. If -Skip 0 is specified, it
    defaults to 1 (skips the first execution). If -Skip is not specified at all,
    no executions are skipped (default is 0).

.PARAMETER Limit
    The maximum number of executions to perform. Skipped executions do not count toward this limit.
    If -Limit is not specified or set to 0, there is no limit (default is 0).

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

.EXAMPLE
    .\rc.ps1 "Get-Process" 5 -Skip 2

    Runs 'Get-Process' every 5 minutes, but skips the first 2 executions. Execution will begin on the 3rd iteration.

.EXAMPLE
    .\rc.ps1 "Get-Date" 1 -Skip 0

    Runs 'Get-Date' every minute, but skips the first execution. Since -Skip 0 was specified, it defaults to 1.
    Execution will begin on the 2nd iteration. To skip more executions, use -Skip 2, -Skip 3, etc.

.EXAMPLE
    .\rc.ps1 "Get-Process" 15s -Limit 5

    Runs 'Get-Process' every 15 seconds, but only executes 5 times total, then exits.

.EXAMPLE
    .\rc.ps1 "Get-Date" 1h -Skip 1 -Limit 3

    Runs 'Get-Date' every hour, skips the first execution, then executes 3 times before exiting.

.NOTES
    To stop the script, press Ctrl+C in the terminal window where it is running.
#>
param(
    [Parameter(Position=0)]
    [string]$Command,

    [Parameter(Position=1)]
    [string]$Period = "5",

    [Parameter(Mandatory=$false, HelpMessage="Enables precision mode to account for command execution time.")]
    [Alias('p')]
    [switch]$Precision,

    [Parameter(Mandatory=$false, HelpMessage="Enables silent mode to suppress status output messages.")]
    [Alias('s')]
    [switch]$Silent,

    [Parameter(Mandatory=$false, HelpMessage="Clears the screen before executing the command in each iteration.")]
    [Alias('cl', 'c')]
    [switch]$Clear,

    [Parameter(Mandatory=$false, HelpMessage="Number of initial executions to skip before starting to run the command. If -Skip is used without a value (or with value 0), it defaults to 1.")]
    [int]$Skip = 0,

    [Parameter(Mandatory=$false, HelpMessage="Maximum number of executions to perform. Skipped executions do not count. 0 = no limit.")]
    [int]$Limit = 0
)

# Function to parse period string with suffixes (s, m, h) and convert to minutes
function Convert-Period {
    param([string]$PeriodStr)
    
    $PeriodStr = $PeriodStr.Trim()
    if ([string]::IsNullOrWhiteSpace($PeriodStr)) {
        return @{ Minutes = 5; Display = "5 minutes" }
    }
    
    # Check for suffix
    $suffix = $PeriodStr.Substring($PeriodStr.Length - 1).ToLower()
    $numberStr = $PeriodStr.Substring(0, $PeriodStr.Length - 1)
    
    if ($suffix -eq 's' -or $suffix -eq 'm' -or $suffix -eq 'h') {
        # Has suffix, extract number
        if ([double]::TryParse($numberStr, [ref]$null)) {
            $number = [double]$numberStr
        } else {
            # Invalid format, default to 5 minutes
            return @{ Minutes = 5; Display = "5 minutes" }
        }
    } else {
        # No suffix, treat entire string as number (minutes)
        if ([double]::TryParse($PeriodStr, [ref]$null)) {
            $number = [double]$PeriodStr
            $suffix = 'm'
        } else {
            # Invalid format, default to 5 minutes
            return @{ Minutes = 5; Display = "5 minutes" }
        }
    }
    
    # Convert to minutes based on suffix
    $minutes = switch ($suffix) {
        's' { $number / 60.0 }
        'm' { $number }
        'h' { $number * 60.0 }
        default { 5 }
    }
    
    # Generate display string
    $display = switch ($suffix) {
        's' { 
            if ($number -eq 1) { "1 second" } 
            else { "$number seconds" }
        }
        'm' { 
            if ($number -eq 1) { "1 minute" } 
            else { "$number minutes" }
        }
        'h' { 
            if ($number -eq 1) { "1 hour" } 
            else { "$number hours" }
        }
    }
    
    return @{ Minutes = $minutes; Display = $display }
}

# Parse period string
$periodInfo = Convert-Period $Period
$PeriodMinutes = $periodInfo.Minutes
$PeriodDisplay = $periodInfo.Display

# If -Skip parameter was explicitly provided but value is 0, default to 1
# This allows -Skip to default to skipping 1 execution when used without a value
if ($PSBoundParameters.ContainsKey('Skip') -and $Skip -eq 0) {
    $Skip = 1
}

if (-not $Command) {
    Write-Host "*** Run Continuously v1 ***" -ForegroundColor Yellow
    $Command = Read-Host "Command"
    $inputPeriod = Read-Host "Period (e.g., 5, 15s, 5m, 1h) [default: 5]"
    if ($inputPeriod) {
        $periodInfo = Convert-Period $inputPeriod
        $PeriodMinutes = $periodInfo.Minutes
        $PeriodDisplay = $periodInfo.Display
        $Period = $inputPeriod
    } else {
        $PeriodMinutes = 5
        $PeriodDisplay = "5 minutes"
        $Period = "5"
    }
    $inputPrecision = Read-Host "Enable Precision Mode? (y/n) [default: n]"
    if ($inputPrecision.ToLower() -eq 'y') {
        $Precision = $true
    }
    $inputClear = Read-Host "Enable Clear Mode? (y/n) [default: n]"
    if ($inputClear.ToLower() -eq 'y') {
        $Clear = $true
    }
    $inputLimit = Read-Host "Limit executions? (enter number, or 0 for no limit) [default: 0]"
    if ($inputLimit -and [int]::TryParse($inputLimit, [ref]$null)) {
        $Limit = [int]$inputLimit
    }
}

# Clear screen if requested - must be done before any output
if ($Clear.IsPresent) {
    Clear-Host
}

if (-not $Silent.IsPresent) {
    Write-Host "Running `"$Command`" every $PeriodDisplay. Press Ctrl+C to stop.`n"
    if ($Skip -gt 0) {
        Write-Host "Skipping the first $Skip execution(s)." -ForegroundColor Yellow
    }
    if ($Limit -gt 0) {
        Write-Host "Limited to $Limit execution(s)." -ForegroundColor Cyan
    }
}
$scriptStartTime = Get-Date
if ($Precision.IsPresent -and -not $Silent.IsPresent) {
    Write-Host "Precision mode is enabled. Aligning to grid starting at $($scriptStartTime.ToString('HH:mm:ss'))." -ForegroundColor Cyan
}

# Initialize execution counter to track loop iterations
$executionCount = 0
$actualExecutionCount = 0
while ($true) {
    $executionCount++
    $loopStartTime = Get-Date
    
    # Skip execution if we haven't reached the skip threshold yet
    # User feedback is provided unless Silent mode is enabled
    if ($executionCount -le $Skip) {
        if (-not $Silent.IsPresent) {
            Write-Host "($(Get-Date -Format 'HH:mm:ss')) Skipping execution $executionCount of $Skip..." -ForegroundColor Yellow
        }
    } else {
        # Execute the command once we've passed the skip threshold
        $actualExecutionCount++
        try {
            if ($Clear.IsPresent) {
                Clear-Host
            }
            if (-not $Silent.IsPresent) {
                Write-Host "($(Get-Date -Format 'HH:mm:ss')) Executing command..."
            }
            Invoke-Expression $Command
        }
        catch {
            Write-Warning "Command failed: $_"
        }
        
        # Check if limit reached
        if ($Limit -gt 0 -and $actualExecutionCount -ge $Limit) {
            if (-not $Silent.IsPresent) {
                Write-Host "`nReached execution limit of $Limit. Exiting." -ForegroundColor Green
            }
            break
        }
    }

    if ($Precision.IsPresent) {
        $currentTime = Get-Date
        $commandDuration = $currentTime - $loopStartTime

        # Calculate the next scheduled run time based on the script's start time (grid alignment)
        $totalElapsedMinutes = ($currentTime - $scriptStartTime).TotalMinutes
        $intervalsCompleted = [math]::Floor($totalElapsedMinutes / $PeriodMinutes)
        $nextTargetTime = $scriptStartTime.AddMinutes(($intervalsCompleted + 1) * $PeriodMinutes)

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
        # Standard mode: wait for the specified period after command execution
        # Note: This wait period also applies during skipped executions to maintain timing
        if (-not $Silent.IsPresent) {
            Write-Host "Waiting $PeriodDisplay. Press Ctrl+C to stop.`n"
        }
        Start-Sleep -Seconds ($PeriodMinutes * 60)
    }
}
