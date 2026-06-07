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
    Aliases: -q, -quiet

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

.PARAMETER Expect
    Minimum expected command runtime using Period format (s/m/h, or minutes without suffix).
    Runs that complete faster than this threshold are treated as failures.
    Success summary is printed after each run in both default and precision scheduling modes.
    Alias: -e

.PARAMETER Replace
    Replaces every literal ^* marker in -Command with this string before execution.
    Alias: -r

.PARAMETER Fail
    Maximum number of failed runs (below -Expect threshold) before exiting. Requires -Expect.
    Alias: -f

.PARAMETER FailTime
    Maximum cumulative failure time (failed runs times retry interval) before exiting. Requires -Expect.
    Uses period format (s/m/h). Alias: -ft

.PARAMETER Success
    Number of successful runs (meeting -Expect threshold) before exiting. Requires -Expect.
    Alias: -s

.PARAMETER SuccessTime
    Maximum accumulated successful run time before exiting. Requires -Expect.
    Uses period format (s/m/h). Alias: -st

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
    .\rc.ps1 "Get-Date" 1 -Quiet

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

.EXAMPLE
    .\rc.ps1 "Invoke-WebRequest https://example.com" 5s -Expect 1s

    Runs every 5 seconds and tracks successful runs where command duration is at least 1 second.

.EXAMPLE
    .\rc.ps1 "gf -x ^*" 5 -r pdx

    Runs 'gf -x pdx' every 5 minutes by substituting pdx for the ^* marker in the command.

.EXAMPLE
    .\rc.ps1 "Get-Date" 5m -Expect 30s -Fail 3

    Exits after 3 runs that finish faster than the 30 second expected minimum.

.EXAMPLE
    .\rc.ps1 "Get-Date" 5m -Expect 30s -Success 5

    Exits after 5 runs that meet the 30 second expected minimum.

.PARAMETER Help
    Displays full command-line reference (arguments, period format, scheduling behavior) and exits.

.NOTES
    To stop the script, press Ctrl+C in the terminal window where it is running.
#>
param()

$ReplaceMarker = '^*'

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

function Parse-RcCliArgs {
    param(
        [string[]]$ArgumentList,
        [switch]$QuietWarnings
    )

    $switchMap = @{
        'p' = 'Precision'; 'precision' = 'Precision'
        'q' = 'Silent'; 'quiet' = 'Silent'; 'silent' = 'Silent'
        'c' = 'Clear'; 'cl' = 'Clear'; 'clear' = 'Clear'
        'h' = 'Help'; '?' = 'Help'; 'help' = 'Help'
    }
    $valueMap = @{
        'skip' = 'Skip'; 'limit' = 'Limit'
        'e' = 'Expect'; 'expect' = 'Expect'
        'r' = 'Replace'; 'replace' = 'Replace'
        'f' = 'Fail'; 'fail' = 'Fail'
        'ft' = 'FailTime'; 'failtime' = 'FailTime'
        's' = 'Success'; 'success' = 'Success'
        'st' = 'SuccessTime'; 'successtime' = 'SuccessTime'
    }

    $result = @{
        Help        = $false
        Command     = $null
        Period      = '5'
        Precision   = $false
        Silent      = $false
        Clear       = $false
        Skip        = 0
        Limit       = 0
        Expect      = $null
        Replace     = $null
        Fail        = 0
        FailTime    = $null
        Success     = 0
        SuccessTime = $null
        Bound       = @{}
        SkipFlagFound = $false
    }
    $seen = @{}

    $warnDuplicate = {
        param([string]$FlagLabel)
        if ($seen.ContainsKey($FlagLabel)) {
            if (-not $QuietWarnings) {
                Write-Warning "Flag -$FlagLabel specified more than once; using the first value."
            }
            return $true
        }
        $seen[$FlagLabel] = $true
        return $false
    }

    $nonFlags = [System.Collections.Generic.List[string]]::new()
    $i = 0
    while ($i -lt $ArgumentList.Count) {
        $arg = $ArgumentList[$i]
        if ($arg -match '^-{1,2}(.+)$') {
            $flagName = $Matches[1].ToLower()
            if ($switchMap.ContainsKey($flagName)) {
                $key = $switchMap[$flagName]
                if (& $warnDuplicate $flagName) {
                    $i++
                    continue
                }
                $result[$key] = $true
                $result.Bound[$key] = $true
                $i++
                continue
            }
            if ($valueMap.ContainsKey($flagName)) {
                $key = $valueMap[$flagName]
                if (& $warnDuplicate $flagName) {
                    if ($i + 1 -lt $ArgumentList.Count -and $ArgumentList[$i + 1] -notmatch '^-') {
                        $i += 2
                    } else {
                        $i++
                    }
                    continue
                }
                if ($i + 1 -lt $ArgumentList.Count -and $ArgumentList[$i + 1] -notmatch '^-') {
                    $value = $ArgumentList[$i + 1]
                    if ($key -eq 'Skip') {
                        if ([int]::TryParse($value, [ref]$null)) {
                            $result.Skip = [int]$value
                            $result.SkipFlagFound = $true
                        }
                    } elseif ($key -eq 'Limit') {
                        if ([int]::TryParse($value, [ref]$null)) {
                            $result.Limit = [int]$value
                        }
                    } elseif ($key -eq 'Fail') {
                        if ([int]::TryParse($value, [ref]$null)) {
                            $result.Fail = [int]$value
                        }
                    } elseif ($key -eq 'Success') {
                        if ([int]::TryParse($value, [ref]$null)) {
                            $result.Success = [int]$value
                        }
                    } else {
                        $result[$key] = $value
                    }
                    $result.Bound[$key] = $true
                    $i += 2
                    continue
                }
                $result.Bound[$key] = $true
                $i++
                continue
            }
            $nonFlags.Add($arg)
            $i++
            continue
        }
        $nonFlags.Add($arg)
        $i++
    }

    if ($nonFlags.Count -gt 0) {
        $result.Command = $nonFlags[0]
    }
    if ($nonFlags.Count -gt 1) {
        for ($j = 1; $j -lt $nonFlags.Count; $j++) {
            $candidate = $nonFlags[$j]
            $periodTest = Convert-Period $candidate
            if ($periodTest) {
                $result.Period = $candidate
                break
            }
        }
    }

    return $result
}

$cli = Parse-RcCliArgs -ArgumentList $args
$Help = [bool]$cli.Help
$Command = $cli.Command
$Period = $cli.Period
$Precision = [bool]$cli.Precision
$Silent = [bool]$cli.Silent
$Clear = [bool]$cli.Clear
$Skip = [int]$cli.Skip
$Limit = [int]$cli.Limit
$Expect = $cli.Expect
$Replace = $cli.Replace
$Fail = [int]$cli.Fail
$FailTime = $cli.FailTime
$Success = [int]$cli.Success
$SuccessTime = $cli.SuccessTime
$rcBound = $cli.Bound
$skipFlagFound = [bool]$cli.SkipFlagFound

if ($Help) {
    $banner = 'Run Continuously (rc.ps1) - CLI reference'
    Write-Host $banner -ForegroundColor Cyan
    Write-Host ('=' * $banner.Length) -ForegroundColor DarkGray
    Write-Host ''
    Write-Host @'
SYNOPSIS
  Runs a PowerShell command string on a repeating interval until you press Ctrl+C or -Limit is reached.

USAGE
  .\rc.ps1 [[-Command] string] [[-Period] string] [-Precision] [-Silent] [-Clear]
           [-Skip int] [-Limit int] [-Expect string] [-Replace string]
           [-Fail int] [-FailTime string] [-Success int] [-SuccessTime string] [-Help]

  With no -Command, the script prompts for command, period, precision, clear, and limit.

PARAMETERS
  -Command string     (positional 0)
      Expression evaluated each iteration (Invoke-Expression). Quote strings that contain spaces.
      Example: "Get-Date" or "Get-Process | Select-Object -First 5"

  -Period string      (positional 1, default: 5)
      Wait between iterations. See PERIOD FORMAT below. Invalid values fall back to 5 minutes.

  -Precision          Alias: -p
      Grid-aligned scheduling from script start time. Sleep fills the gap to the next interval boundary;
      if a run exceeds its slot, the next iteration starts immediately to recover.

  -Silent             Aliases: -q, -quiet
      Suppresses status lines (timestamps, waits, skip/limit banners). Command output and errors still show.

  -Clear              Aliases: -c, -cl
      Clears the host before each run (and once at startup if set).

  -Skip int           (default: 0 when omitted)
      Number of first loop iterations that skip invoking -Command but still advance the wait schedule.
      If you pass -Skip and the value is 0, it is treated as 1 (skip the first execution only).

  -Limit int          (default: 0)
      Maximum times -Command is actually executed. Skipped iterations do not count. 0 = unlimited.

  -Expect string      Alias: -e
      Minimum expected command runtime in Period format. A run counts as success only when
      command duration is greater than or equal to this threshold.
      Prints success summary after each run in standard and precision modes.

  -Replace string     Alias: -r
      Replaces every literal ^* marker in -Command with this value before execution.
      Emits a soft warning if -Replace is set but the command has no ^* marker.

  -Fail int           Alias: -f
      Exit after this many failed runs (duration below -Expect). Requires -Expect. 0 = unlimited.

  -FailTime string    Alias: -ft
      Exit when failed runs times retry interval reaches this cap. Period format. Requires -Expect.

  -Success int         Alias: -s
      Exit after this many successful runs (duration at or above -Expect). Requires -Expect. 0 = unlimited.

  -SuccessTime string  Alias: -st
      Exit when accumulated successful run time reaches this cap. Period format. Requires -Expect.

  -Help               Aliases: -h, -?
      Show this reference and exit.

PERIOD FORMAT (internal function: Convert-Period)
  Suffix   Meaning        Examples
  (none)   minutes        5  -> 5 minutes
  s        seconds        15s
  m        minutes        5m
  h        hours          1h
  Blank or non-numeric values use 5 minutes.

SCHEDULING
  Standard (default)     After each iteration (or skip wait), sleeps for the full period. Simple; can drift.
  -Precision             Aligns to multiples of the period from the time the script started.

STOPPING
  Ctrl+C in the same console window.

EXAMPLES
  .\rc.ps1 "Get-Date" 1
  .\rc.ps1 ".\my-script.ps1" 10 -Precision -Quiet
  .\rc.ps1 "Get-Process" 15s -Limit 5
  .\rc.ps1 "Invoke-WebRequest https://example.com" 5s -Expect 1s
  .\rc.ps1 "gf -x ^*" 5 -r pdx
  .\rc.ps1 "Get-Date" 5m -e 30s -fail 3
  .\rc.ps1 "Get-Date" 5m -e 30s -success 5
  .\rc.ps1 -Help

For comment-based help: Get-Help .\rc.ps1 -Full

'@
    exit 0
}

function Format-CompactDuration {
    param(
        [TimeSpan]$Span,
        [switch]$ShowFractionWhenUnderMinute
    )

    if ($Span.TotalHours -ge 1) {
        return ('{0:00}:{1:00}:{2:00}s' -f [int]$Span.TotalHours, $Span.Minutes, $Span.Seconds)
    }

    if ($Span.TotalMinutes -ge 1) {
        return ('{0:00}:{1:00}s' -f [int]$Span.TotalMinutes, $Span.Seconds)
    }

    if ($ShowFractionWhenUnderMinute) {
        return ('{0:N2}s' -f $Span.TotalSeconds)
    }

    return ('{0}s' -f [math]::Round($Span.TotalSeconds, 0))
}

function Format-DateAwareTimestamp {
    param([datetime]$Timestamp)

    if ($Timestamp.Date -eq (Get-Date).Date) {
        return $Timestamp.ToString('HH:mm:ss')
    }

    return $Timestamp.ToString('MMddyy@HH:mm:ss')
}

function Format-CompactPeriodLabel {
    param([TimeSpan]$Span)

    $totalSec = [int][math]::Round([math]::Max(0, $Span.TotalSeconds))
    if ($totalSec -ge 3600) {
        $h = [math]::Floor($totalSec / 3600)
        $rem = $totalSec % 3600
        if ($rem -eq 0) {
            return "${h}H"
        }
        $m = [math]::Floor($rem / 60)
        if ($m -gt 0) {
            return "${h}H${m}M"
        }
        return "${h}H"
    }
    if ($totalSec -ge 60) {
        return "$([math]::Floor($totalSec / 60))M"
    }
    if ($totalSec -le 0) {
        return '0S'
    }
    return "${totalSec}S"
}

function Format-ExpectConfigDetails {
    param(
        $ExpectThreshold,
        [int]$SuccessLimit,
        $SuccessTimeThreshold,
        [int]$SuccessfulExecutionCount = 0,
        [TimeSpan]$TotalSuccessfulRuntime = [TimeSpan]::Zero,
        [int]$FailLimit,
        $FailTimeThreshold,
        [int]$FailedExecutionCount = 0,
        [TimeSpan]$FailedRetryTime = [TimeSpan]::Zero
    )

    $parts = @()
    if ($ExpectThreshold) {
        $parts += "Expect: $(Format-CompactPeriodLabel $ExpectThreshold)"
    }
    if ($SuccessLimit -gt 0) {
        if ($SuccessfulExecutionCount -gt 0) {
            $parts += "Success: $SuccessfulExecutionCount/$SuccessLimit"
        } else {
            $parts += "Success: $SuccessLimit"
        }
    }
    if ($SuccessTimeThreshold) {
        $totalLabel = Format-CompactPeriodLabel $SuccessTimeThreshold
        if ($TotalSuccessfulRuntime.TotalSeconds -gt 0) {
            $remaining = $SuccessTimeThreshold - $TotalSuccessfulRuntime
            if ($remaining.TotalSeconds -lt 0) {
                $remaining = [TimeSpan]::Zero
            }
            $parts += "SuccessTime: $(Format-CompactPeriodLabel $remaining) / $totalLabel"
        } else {
            $parts += "SuccessTime: $totalLabel"
        }
    }
    if ($FailLimit -gt 0) {
        if ($FailedExecutionCount -gt 0) {
            $parts += "Fail: $FailedExecutionCount/$FailLimit"
        } else {
            $parts += "Fail: $FailLimit"
        }
    }
    if ($FailTimeThreshold) {
        $totalLabel = Format-CompactPeriodLabel $FailTimeThreshold
        if ($FailedRetryTime.TotalSeconds -gt 0) {
            $remaining = $FailTimeThreshold - $FailedRetryTime
            if ($remaining.TotalSeconds -lt 0) {
                $remaining = [TimeSpan]::Zero
            }
            $parts += "FailTime: $(Format-CompactPeriodLabel $remaining) / $totalLabel"
        } else {
            $parts += "FailTime: $totalLabel"
        }
    }

    if ($parts.Count -eq 0) {
        return $null
    }

    return $parts -join ' | '
}

function Write-ExpectSummaryIfNeeded {
    param(
        $ExpectThreshold,
        [int]$ExecutionCount,
        [int]$Skip,
        $LastSuccessfulCompletionTime,
        [int]$SuccessfulExecutionCount,
        [int]$ActualExecutionCount,
        [TimeSpan]$TotalSuccessfulRuntime,
        $LastSuccessfulRuntime
    )

    if (-not $ExpectThreshold) { return }
    if ($ExecutionCount -le $Skip) { return }

    $lastSuccessDisplay = if ($LastSuccessfulCompletionTime) { Format-DateAwareTimestamp $LastSuccessfulCompletionTime } else { 'N/A' }
    $totalSuccessDisplay = '{0:00}:{1:00}:{2:00}.{3:00}' -f [int]$TotalSuccessfulRuntime.TotalHours, $TotalSuccessfulRuntime.Minutes, $TotalSuccessfulRuntime.Seconds, [int]([math]::Floor($TotalSuccessfulRuntime.Milliseconds / 10))
    $lastSuccessRuntimeDisplay = if ($LastSuccessfulRuntime) {
        '{0:00}:{1:00}:{2:00}.{3:00}' -f [int]$LastSuccessfulRuntime.TotalHours, $LastSuccessfulRuntime.Minutes, $LastSuccessfulRuntime.Seconds, [int]([math]::Floor($LastSuccessfulRuntime.Milliseconds / 10))
    } else {
        'N/A'
    }
    Write-Host "Last Success: $lastSuccessDisplay ($SuccessfulExecutionCount/$ActualExecutionCount)"
    Write-Host "Total Runtime: $totalSuccessDisplay ($lastSuccessRuntimeDisplay)"
}

# Parse period string
$periodInfo = Convert-Period $Period
$PeriodMinutes = $periodInfo.Minutes
$PeriodDisplay = $periodInfo.Display
$expectThreshold = $null
$expectDisplay = $null

if ($rcBound.ContainsKey('Expect')) {
    $expectInfo = Convert-Period $Expect
    $expectThreshold = [TimeSpan]::FromMinutes($expectInfo.Minutes)
    $expectDisplay = $expectInfo.Display
}

$failLimit = 0
$failTimeThreshold = $null
$failTimeDisplay = $null
$failLimitRequested = $rcBound.ContainsKey('Fail') -and $Fail -gt 0
$failTimeRequested = $rcBound.ContainsKey('FailTime')
$successLimit = 0
$successTimeThreshold = $null
$successTimeDisplay = $null
$successLimitRequested = $rcBound.ContainsKey('Success') -and $Success -gt 0
$successTimeRequested = $rcBound.ContainsKey('SuccessTime')

if ($failLimitRequested -or $failTimeRequested -or $successLimitRequested -or $successTimeRequested) {
    if (-not $expectThreshold) {
        if (-not $Silent) {
            Write-Warning '-Fail, -FailTime, -Success, and -SuccessTime require -Expect (-e) and were ignored.'
        }
    } else {
        if ($failLimitRequested) {
            $failLimit = $Fail
        }
        if ($failTimeRequested) {
            $failTimeInfo = Convert-Period $FailTime
            $failTimeThreshold = [TimeSpan]::FromMinutes($failTimeInfo.Minutes)
            $failTimeDisplay = $failTimeInfo.Display
        }
        if ($successLimitRequested) {
            $successLimit = $Success
        }
        if ($successTimeRequested) {
            $successTimeInfo = Convert-Period $SuccessTime
            $successTimeThreshold = [TimeSpan]::FromMinutes($successTimeInfo.Minutes)
            $successTimeDisplay = $successTimeInfo.Display
        }
    }
}

$periodInterval = [TimeSpan]::FromMinutes($PeriodMinutes)

# If -Skip parameter was explicitly provided but value is 0, default to 1
# This allows -Skip to default to skipping 1 execution when used without a value
if ($skipFlagFound -and $Skip -eq 0) {
    $Skip = 1
}

if ($rcBound.ContainsKey('Replace')) {
    if (-not $Command.Contains($ReplaceMarker) -and -not $Silent) {
        Write-Warning "-Replace was specified but command does not contain the $ReplaceMarker marker."
    }
    $Command = $Command.Replace($ReplaceMarker, $Replace)
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
if ($Clear) {
    Clear-Host
}

$expectConfigDetails = Format-ExpectConfigDetails -ExpectThreshold $expectThreshold -SuccessLimit $successLimit -SuccessTimeThreshold $successTimeThreshold -FailLimit $failLimit -FailTimeThreshold $failTimeThreshold

if (-not $Silent) {
    Write-Host "Running `"$Command`" every $PeriodDisplay. Press Ctrl+C to stop.`n"
    if ($expectConfigDetails) {
        Write-Host $expectConfigDetails -ForegroundColor Magenta
    }
    if ($Skip -gt 0) {
        Write-Host "Skipping the first $Skip execution(s)." -ForegroundColor Yellow
    }
    if ($Limit -gt 0) {
        Write-Host "Limited to $Limit execution(s)." -ForegroundColor Cyan
    }
}
$scriptStartTime = Get-Date
if ($Precision -and -not $Silent) {
    Write-Host "Precision mode is enabled. Aligning to grid starting at $($scriptStartTime.ToString('HH:mm:ss'))." -ForegroundColor Cyan
}

# Initialize execution counter to track loop iterations
$executionCount = 0
$actualExecutionCount = 0
$successfulExecutionCount = 0
$failedExecutionCount = 0
$failedRetryTime = [TimeSpan]::Zero
$totalSuccessfulRuntime = [TimeSpan]::Zero
$lastSuccessfulRuntime = $null
$lastSuccessfulCompletionTime = $null
$pendingExitMessage = $null
$pendingExitColor = $null
while ($true) {
    $executionCount++
    $loopStartTime = Get-Date
    $commandDuration = $null
    $commandEndTime = $null
    
    # Skip execution if we haven't reached the skip threshold yet
    # User feedback is provided unless Silent mode is enabled
    if ($executionCount -le $Skip) {
        if (-not $Silent) {
            Write-Host "($(Get-Date -Format 'HH:mm:ss')) Skipping execution $executionCount of $Skip..." -ForegroundColor Yellow
        }
    } else {
        # Execute the command once we've passed the skip threshold
        $actualExecutionCount++
        try {
            if ($Clear) {
                Clear-Host
            }
            if (-not $Silent) {
                $executeMessage = "($(Get-Date -Format 'HH:mm:ss')) Executing command..."
                $executeConfigDetails = Format-ExpectConfigDetails -ExpectThreshold $expectThreshold -SuccessLimit $successLimit -SuccessTimeThreshold $successTimeThreshold -SuccessfulExecutionCount $successfulExecutionCount -TotalSuccessfulRuntime $totalSuccessfulRuntime -FailLimit $failLimit -FailTimeThreshold $failTimeThreshold -FailedExecutionCount $failedExecutionCount -FailedRetryTime $failedRetryTime
                if ($executeConfigDetails) {
                    $executeMessage += " [$executeConfigDetails]"
                }
                Write-Host $executeMessage
            }
            Invoke-Expression $Command
            $commandEndTime = Get-Date
            $commandDuration = $commandEndTime - $loopStartTime
        }
        catch {
            $commandEndTime = Get-Date
            $commandDuration = $commandEndTime - $loopStartTime
            Write-Warning "Command failed: $_"
        }

        if ($expectThreshold -and $commandDuration -ge $expectThreshold) {
            $successfulExecutionCount++
            $totalSuccessfulRuntime = $totalSuccessfulRuntime.Add($commandDuration)
            $lastSuccessfulRuntime = $commandDuration
            $lastSuccessfulCompletionTime = $commandEndTime
        } elseif ($expectThreshold) {
            $failedExecutionCount++
            $failedRetryTime = $failedRetryTime.Add($periodInterval)
        }

        # Check if limit reached
        if ($Limit -gt 0 -and $actualExecutionCount -ge $Limit) {
            $pendingExitMessage = "Reached execution limit of $Limit. Exiting."
            $pendingExitColor = 'Green'
        } elseif ($failLimit -gt 0 -and $failedExecutionCount -ge $failLimit) {
            $pendingExitMessage = "Reached failure limit of $failLimit. Exiting."
            $pendingExitColor = 'Red'
        } elseif ($failTimeThreshold -and $failedRetryTime -ge $failTimeThreshold) {
            $pendingExitMessage = "Reached failure time limit of $failTimeDisplay. Exiting."
            $pendingExitColor = 'Red'
        } elseif ($successLimit -gt 0 -and $successfulExecutionCount -ge $successLimit) {
            $pendingExitMessage = "Reached success limit of $successLimit. Exiting."
            $pendingExitColor = 'Green'
        } elseif ($successTimeThreshold -and $totalSuccessfulRuntime -ge $successTimeThreshold) {
            $pendingExitMessage = "Reached success time limit of $successTimeDisplay. Exiting."
            $pendingExitColor = 'Green'
        }
    }

    if ($pendingExitMessage) {
        break
    }

    if ($Precision) {
        $currentTime = Get-Date
        if (-not $commandDuration) {
            $commandDuration = $currentTime - $loopStartTime
        }

        # Calculate the next scheduled run time based on the script's start time (grid alignment)
        $totalElapsedMinutes = ($currentTime - $scriptStartTime).TotalMinutes
        $intervalsCompleted = [math]::Floor($totalElapsedMinutes / $PeriodMinutes)
        $nextTargetTime = $scriptStartTime.AddMinutes(($intervalsCompleted + 1) * $PeriodMinutes)

        $sleepTimeSpan = $nextTargetTime - $currentTime

        if ($sleepTimeSpan.TotalSeconds -gt 0) {
            if (-not $Silent) {
                $runtimeDisplay = Format-CompactDuration -Span $commandDuration -ShowFractionWhenUnderMinute
                $waitingDisplay = Format-CompactDuration -Span $sleepTimeSpan
                $nextRunDisplay = Format-DateAwareTimestamp $nextTargetTime
                Write-Host "Runtime: $runtimeDisplay Waiting: $waitingDisplay Next Run: $nextRunDisplay"
                Write-ExpectSummaryIfNeeded -ExpectThreshold $expectThreshold -ExecutionCount $executionCount -Skip $Skip -LastSuccessfulCompletionTime $lastSuccessfulCompletionTime -SuccessfulExecutionCount $successfulExecutionCount -ActualExecutionCount $actualExecutionCount -TotalSuccessfulRuntime $totalSuccessfulRuntime -LastSuccessfulRuntime $lastSuccessfulRuntime
                Write-Host "Press Ctrl+C to stop."
            }
            Start-Sleep -Seconds $sleepTimeSpan.TotalSeconds
        } else {
            if (-not $Silent) {
                Write-Warning "Command execution time ($($commandDuration.TotalSeconds.ToString('F2'))s) overran its schedule. Running next iteration immediately."
            }
        }
    } else {
        # Standard mode: wait for the specified period after command execution
        # Note: This wait period also applies during skipped executions to maintain timing
        if (-not $Silent) {
            if ($expectThreshold -and $executionCount -gt $Skip) {
                Write-Host "Waiting $PeriodDisplay."
                Write-ExpectSummaryIfNeeded -ExpectThreshold $expectThreshold -ExecutionCount $executionCount -Skip $Skip -LastSuccessfulCompletionTime $lastSuccessfulCompletionTime -SuccessfulExecutionCount $successfulExecutionCount -ActualExecutionCount $actualExecutionCount -TotalSuccessfulRuntime $totalSuccessfulRuntime -LastSuccessfulRuntime $lastSuccessfulRuntime
                Write-Host "Press Ctrl+C to stop.`n"
            } else {
                Write-Host "Waiting $PeriodDisplay. Press Ctrl+C to stop.`n"
            }
        }
        Start-Sleep -Seconds ($PeriodMinutes * 60)
    }
}

if ($pendingExitMessage -and -not $Silent) {
    Write-ExpectSummaryIfNeeded -ExpectThreshold $expectThreshold -ExecutionCount $executionCount -Skip $Skip -LastSuccessfulCompletionTime $lastSuccessfulCompletionTime -SuccessfulExecutionCount $successfulExecutionCount -ActualExecutionCount $actualExecutionCount -TotalSuccessfulRuntime $totalSuccessfulRuntime -LastSuccessfulRuntime $lastSuccessfulRuntime
    Write-Host "`n$pendingExitMessage" -ForegroundColor $pendingExitColor
}
