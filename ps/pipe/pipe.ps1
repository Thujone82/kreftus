# Portland Big Pipe Report Script
# Parses 15-minute interval data and displays statistics with sparklines

param(
    [Alias("l")]
    [switch]$level,
    [Alias("b")]
    [switch]$banner,
    [switch]$sma,
    [Alias("cap")]
    [switch]$capacity,
    [Alias("lf")]
    [switch]$lastfull,
    [switch]$hl12,
    [switch]$hl24,
    [switch]$hl72,
    [switch]$s12,
    [switch]$s24,
    [switch]$s72
)

# Clear screen only if no specific output is requested
if (-not ($level -or $banner -or $sma -or $capacity -or $lastfull -or $hl12 -or $hl24 -or $hl72 -or $s12 -or $s24 -or $s72)) {
Clear-Host
}

# City of Portland Big Pipe endpoints (same backend; chart JSON is the stable series)
$chartUrl = "https://www.portlandoregon.gov/bes/bigpipe/chart.cfm"
$dataUrl = "https://www.portlandoregon.gov/bes/bigpipe/data.cfm"
$gaugeUrl = "https://www.portlandoregon.gov/bes/bigpipe/gauge.cfm"

# --- Helper Function to Get Pacific Timezone ---
function Get-PacificTimeZone {
    try {
        return [System.TimeZoneInfo]::FindSystemTimeZoneById("Pacific Standard Time")
    }
    catch {
        return [System.TimeZoneInfo]::FindSystemTimeZoneById("America/Los_Angeles")
    }
}

# --- Helper Function to Convert Unix Milliseconds to Pacific DateTime ---
function ConvertFrom-UnixMillisecondsToPacific {
    param ([Int64]$UnixMilliseconds)

    $utc = [DateTimeOffset]::FromUnixTimeMilliseconds($UnixMilliseconds).UtcDateTime
    return [TimeZoneInfo]::ConvertTimeFromUtc($utc, (Get-PacificTimeZone))
}

# --- Helper Function to Sort and Return Data Points ---
function ConvertTo-SortedPipeData {
    param ([array]$DataPoints)

    if ($null -eq $DataPoints -or $DataPoints.Count -eq 0) {
        return @()
    }
    return @($DataPoints | Sort-Object -Property DateTime)
}

# --- Parse 72-hour series from chart.cfm (ZingChart sensorData JSON) ---
function Get-PipeDataFromChart {
    param ([string]$HtmlContent)

    $dataPoints = @()
    $jsonMatch = [regex]::Match($HtmlContent, 'sensorData\s*=\s*(\{.*?\});', [System.Text.RegularExpressions.RegexOptions]::Singleline)
    if (-not $jsonMatch.Success) {
        return @()
    }

    try {
        $sensorData = $jsonMatch.Groups[1].Value | ConvertFrom-Json
    }
    catch {
        return @()
    }

    if ($null -eq $sensorData.DATA) {
        return @()
    }

    foreach ($row in @($sensorData.DATA)) {
        if ($null -eq $row) {
            continue
        }

        try {
            $unixMs = [int64][double]$row[0]
            $percentage = [double]$row[1]
            $dataPoints += [PSCustomObject]@{
                DateTime = ConvertFrom-UnixMillisecondsToPacific -UnixMilliseconds $unixMs
                Percentage = $percentage
            }
        }
        catch {
            continue
        }
    }

    return ConvertTo-SortedPipeData -DataPoints $dataPoints
}

# --- Parse 72-hour series from data.cfm HTML table ---
function Get-PipeDataFromTable {
    param ([string]$HtmlContent)

    $dataPoints = @()

    # <td><time>12/8/25 2:30 PM</time></td><td>37%</td>
    # also plain cells if <time> is dropped: <td>12/8/25 2:30 PM</td><td>37%</td>
    $pattern = "<td[^>]*>\s*(?:<time[^>]*>)?(\d{1,2}/\d{1,2}/\d{2,4})\s+(\d{1,2}:\d{2}\s+(?:AM|PM))(?:</time>)?\s*</td>\s*<td[^>]*>\s*(\d{1,3})%?\s*</td>"
    $regexMatches = [regex]::Matches($HtmlContent, $pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)

    foreach ($match in $regexMatches) {
        $dateTimeStr = "$($match.Groups[1].Value) $($match.Groups[2].Value)"
        $percentStr = $match.Groups[3].Value

        try {
            $dataPoints += [PSCustomObject]@{
                DateTime = [DateTime]::Parse($dateTimeStr)
                Percentage = [double]$percentStr
            }
        }
        catch {
            continue
        }
    }

    return ConvertTo-SortedPipeData -DataPoints $dataPoints
}

# --- Parse last published reading from gauge.cfm ---
function Get-PipeDataFromGauge {
    param ([string]$HtmlContent)

    $percentMatch = [regex]::Match($HtmlContent, 'percent__number">\s*(\d{1,3}(?:\.\d+)?)\s*<', [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
    if (-not $percentMatch.Success) {
        return @()
    }

    $timeMatch = [regex]::Match($HtmlContent, 'class="timestamp">\s*(.*?)\s*</p>', [System.Text.RegularExpressions.RegexOptions]::IgnoreCase -bor [System.Text.RegularExpressions.RegexOptions]::Singleline)
    $dateTime = $null
    if ($timeMatch.Success) {
        $timeStr = ($timeMatch.Groups[1].Value -replace '<br\s*/?>', ' ' -replace '\s+', ' ').Trim()
        try {
            $dateTime = [DateTime]::Parse($timeStr)
        }
        catch {
            $dateTime = $null
        }
    }

    if ($null -eq $dateTime) {
        $dateTime = Get-PacificTime
    }

    return @(
        [PSCustomObject]@{
            DateTime = $dateTime
            Percentage = [double]$percentMatch.Groups[1].Value
        }
    )
}

# --- Helper Function to Get Color Based on Percentage ---
function Get-PercentageColor {
    param ([double]$Percentage)
    
    if ($Percentage -le 5) {
        return [System.ConsoleColor]::White
    } elseif ($Percentage -le 20) {
        return [System.ConsoleColor]::Green
    } elseif ($Percentage -le 50) {
        return [System.ConsoleColor]::Cyan
    } elseif ($Percentage -le 80) {
        return [System.ConsoleColor]::Yellow
    } elseif ($Percentage -le 95) {
        return [System.ConsoleColor]::Red
    } else {
        return [System.ConsoleColor]::Magenta
    }
}

# --- Helper Function to Generate Sparkline from Time-Binned Data ---
function Get-Sparkline {
    param (
        [array]$DataPoints,  # Array of objects with DateTime and Percentage properties
        [TimeSpan]$BinSize,  # Size of each time bin (e.g., 30 minutes, 1 hour, 3 hours)
        [DateTime]$EndTime   # End time for binning (typically current time or most recent data point)
    )
    
    if ($null -eq $DataPoints -or $DataPoints.Count -eq 0) {
        return @{
            Sparkline = (" " * 24)
            BinnedValues = @()
        }
    }
    
    # Create 24 time bins going backwards from end time
    $binnedValues = @()
    $binStart = $EndTime
    
    for ($binIndex = 23; $binIndex -ge 0; $binIndex--) {
        $binEnd = $binStart
        $binStart = $binEnd.Subtract($BinSize)
        
        # Find all data points that fall within this time bin
        $pointsInBin = $DataPoints | Where-Object {
            $_.DateTime -ge $binStart -and $_.DateTime -lt $binEnd
        }
        
        if ($pointsInBin.Count -gt 0) {
            # Average the percentage values in this bin
            $avg = ($pointsInBin | Measure-Object -Property Percentage -Average).Average
            $binnedValues = @($avg) + $binnedValues
        } else {
            # No data in this bin (missing samples) - use null to indicate gap
            $binnedValues = @($null) + $binnedValues
        }
    }
    
    # Interpolate missing data (null values) to fill gaps
    for ($i = 0; $i -lt $binnedValues.Count; $i++) {
        if ($null -eq $binnedValues[$i]) {
            # Find previous non-null value
            $prevValue = $null
            $prevIndex = $i - 1
            while ($prevIndex -ge 0 -and $null -eq $prevValue) {
                $prevValue = $binnedValues[$prevIndex]
                $prevIndex--
            }
            
            # Find next non-null value
            $nextValue = $null
            $nextIndex = $i + 1
            while ($nextIndex -lt $binnedValues.Count -and $null -eq $nextValue) {
                $nextValue = $binnedValues[$nextIndex]
                $nextIndex++
            }
            
            # Interpolate based on available values
            if ($null -ne $prevValue -and $null -ne $nextValue) {
                # Linear interpolation between previous and next
                $distance = $nextIndex - $prevIndex - 1
                $position = $i - $prevIndex - 1
                $weight = if ($distance -gt 0) { $position / $distance } else { 0.5 }
                $binnedValues[$i] = $prevValue + ($nextValue - $prevValue) * $weight
            } elseif ($null -ne $prevValue) {
                # Only previous value available - use it (extend forward)
                $binnedValues[$i] = $prevValue
            } elseif ($null -ne $nextValue) {
                # Only next value available - use it (extend backward)
                $binnedValues[$i] = $nextValue
            }
            # If both are null, leave as null (will be handled below)
        }
    }
    
    # Filter out null values for min/max calculation
    $validValues = $binnedValues | Where-Object { $null -ne $_ }
    
    if ($validValues.Count -eq 0) {
        return @{
            Sparkline = (" " * 24)
            BinnedValues = @()
        }
    }
    
    $sparkChars = [char[]]([char]0x2581, [char]0x2582, [char]0x2583, [char]0x2584, [char]0x2585, [char]0x2586, [char]0x2587, [char]0x2588)
    $minValue = ($validValues | Measure-Object -Minimum).Minimum
    $maxValue = ($validValues | Measure-Object -Maximum).Maximum
    $valueRange = $maxValue - $minValue
    
    if ($valueRange -eq 0 -or $valueRange -lt 0.00000001) {
        # All values are the same - use lowest character for all bins
        $sparkline = ""
        foreach ($value in $binnedValues) {
            if ($null -eq $value) {
                $sparkline += " "
            } else {
                $sparkline += $sparkChars[0]
            }
        }
        return @{
            Sparkline = $sparkline
            BinnedValues = $binnedValues
        }
    }
    
    $sparkline = ""
    foreach ($value in $binnedValues) {
        if ($null -eq $value) {
            # Still null after interpolation attempt - use space (shouldn't happen, but safe fallback)
            $sparkline += " "
        } else {
        $normalized = ($value - $minValue) / $valueRange
        $charIndex = [math]::Floor($normalized * ($sparkChars.Length - 1))
        $sparkline += $sparkChars[$charIndex]
    }
    }
    
    return @{
        Sparkline = $sparkline
        BinnedValues = $binnedValues
    }
}

# --- Helper Function to Get Current Time in Pacific Timezone ---
function Get-PacificTime {
    try {
        return [TimeZoneInfo]::ConvertTimeFromUtc([DateTime]::UtcNow, (Get-PacificTimeZone))
    }
    catch {
        return Get-Date
    }
}
    
# --- Helper Function to Format Duration ---
function Format-Duration {
    param ([TimeSpan]$Duration)
    
    if ($Duration.TotalDays -ge 1) {
        $days = [math]::Floor($Duration.TotalDays)
        $hours = $Duration.Hours
        if ($hours -gt 0) {
            return "${days}d ${hours}h"
        } else {
            return "${days}d"
        }
    } elseif ($Duration.TotalHours -ge 1) {
        $hours = [math]::Floor($Duration.TotalHours)
        $minutes = $Duration.Minutes
        if ($minutes -gt 0) {
            return "${hours}h ${minutes}m"
        } else {
            return "${hours}h"
        }
    } elseif ($Duration.TotalMinutes -ge 1) {
        $minutes = [math]::Floor($Duration.TotalMinutes)
        $seconds = $Duration.Seconds
        if ($seconds -gt 0) {
            return "${minutes}m ${seconds}s"
        } else {
            return "${minutes}m"
        }
    } else {
        return "$($Duration.Seconds)s"
    }
}

# --- Helper Function to Display Colored Sparkline ---
function Write-ColoredSparkline {
    param (
        [string]$Label,
        [string]$Sparkline,
        [array]$BinnedValues  # Array that may contain null values for missing data
    )
    
    Write-Host -NoNewline -ForegroundColor White "$Label "
    
    # Find where the actual data starts (skip leading spaces)
    $dataStartIndex = 0
    for ($i = 0; $i -lt $Sparkline.Length; $i++) {
        if ($Sparkline[$i] -ne ' ') {
            $dataStartIndex = $i
            break
        }
    }
    
    for ($i = 0; $i -lt $Sparkline.Length; $i++) {
        $char = $Sparkline[$i]
        if ($char -eq ' ') {
            Write-Host -NoNewline $char
            continue
        }
        
        # Map glyph position to binned value index
        # The first non-space character corresponds to the first binned value
        $valueIndex = $i - $dataStartIndex
        if ($valueIndex -ge 0 -and $valueIndex -lt $BinnedValues.Count) {
            $percentage = $BinnedValues[$valueIndex]
            # Handle null values (missing data) - use white color
            if ($null -ne $percentage) {
                $color = Get-PercentageColor -Percentage $percentage
            } else {
                $color = [System.ConsoleColor]::White
            }
        } else {
            # Fallback for edge cases
            $color = [System.ConsoleColor]::White
        }
        
        Write-Host -NoNewline -ForegroundColor $color $char
    }
    
    Write-Host ""
}

try {
    $allData = @()
    $dataSource = $null
    $fetchErrors = @()

    try {
        $chartResponse = Invoke-WebRequest -Uri $chartUrl -UseBasicParsing
        $allData = @(Get-PipeDataFromChart -HtmlContent $chartResponse.Content)
        if ($allData.Count -gt 0) {
            $dataSource = "chart"
        }
    }
    catch {
        $fetchErrors += $_.Exception.Message
    }

    if ($allData.Count -eq 0) {
        try {
            $tableResponse = Invoke-WebRequest -Uri $dataUrl -UseBasicParsing
            $allData = @(Get-PipeDataFromTable -HtmlContent $tableResponse.Content)
            if ($allData.Count -gt 0) {
                $dataSource = "table"
            }
        }
        catch {
            $fetchErrors += $_.Exception.Message
        }
    }

    if ($allData.Count -eq 0) {
        try {
            $gaugeResponse = Invoke-WebRequest -Uri $gaugeUrl -UseBasicParsing
            $allData = @(Get-PipeDataFromGauge -HtmlContent $gaugeResponse.Content)
            if ($allData.Count -gt 0) {
                $dataSource = "gauge"
            }
        }
        catch {
            $fetchErrors += $_.Exception.Message
        }
    }

    if ($allData.Count -eq 0) {
        if ($fetchErrors.Count -gt 0) {
            Write-Error "Failed to reach the Big Pipe data source. Error: $($fetchErrors[0])"
        }
        else {
            Write-Error "The City of Portland Big Pipe feed currently has no samples (72-hour chart and data table are empty)."
        }
        exit 1
    }
    
    # Calculate statistics
    $currentLevel = $allData[-1].Percentage
    $currentTime = $allData[-1].DateTime
    
    # Calculate 100% duration if current level is 100%
    # Find the first 100% reading timestamp and calculate duration to current actual time
    # Note: This handles missing samples correctly by using actual timestamps rather than
    # counting samples, so gaps in the 15-minute interval data don't affect the calculation
    $duration100Percent = $null
    if ($currentLevel -eq 100) {
        # Find the first 100% reading by going backwards from the most recent
        # This finds the earliest consecutive 100% reading in the data
        $first100PercentTime = $null
        for ($i = $allData.Count - 1; $i -ge 0; $i--) {
            if ($allData[$i].Percentage -eq 100) {
                $first100PercentTime = $allData[$i].DateTime
            } else {
                # Found first non-100% reading, stop searching backwards
                break
            }
        }
        # Calculate duration from first 100% reading to current actual time in Pacific timezone
        # This accounts for missing samples by using real timestamps, not sample counts
        if ($null -ne $first100PercentTime) {
            $actualCurrentTimePacific = Get-PacificTime
            $duration100Percent = $actualCurrentTimePacific - $first100PercentTime
        }
    }
    
    # Calculate last 100% reading if current level is not 100%
    # Find the most recent 100% reading and calculate time since then
    $lastFullTime = $null
    $timeSinceLastFull = $null
    if ($currentLevel -lt 100) {
        # Find the most recent 100% reading by going backwards from the most recent
        for ($i = $allData.Count - 1; $i -ge 0; $i--) {
            if ($allData[$i].Percentage -eq 100) {
                $lastFullTime = $allData[$i].DateTime
                # Calculate time since last full in Pacific timezone
                $actualCurrentTimePacific = Get-PacificTime
                $timeSinceLastFull = $actualCurrentTimePacific - $lastFullTime
                break
            }
        }
    }
    
    # 12H SMA: Simple moving average of last 12 hours
    $twelveHoursAgo = $currentTime.AddHours(-12)
    $last12Hours = $allData | Where-Object { $_.DateTime -ge $twelveHoursAgo }
    $sma12H = if ($last12Hours.Count -gt 0) {
        ($last12Hours | Measure-Object -Property Percentage -Average).Average
    } else {
        $currentLevel
    }
    
    # 24H SMA: Simple moving average of last 24 hours
    $twentyFourHoursAgo = $currentTime.AddHours(-24)
    $last24Hours = $allData | Where-Object { $_.DateTime -ge $twentyFourHoursAgo }
    $sma24H = if ($last24Hours.Count -gt 0) {
        ($last24Hours | Measure-Object -Property Percentage -Average).Average
    } else {
        $currentLevel
    }
    
    # 72H SMA: Simple moving average of last 72 hours
    $seventyTwoHoursAgo = $currentTime.AddHours(-72)
    $last72Hours = $allData | Where-Object { $_.DateTime -ge $seventyTwoHoursAgo }
    $sma72H = if ($last72Hours.Count -gt 0) {
        ($last72Hours | Measure-Object -Property Percentage -Average).Average
    } else {
        $currentLevel
    }
    
    # 12H High and Low: Maximum and minimum of last 12 hours
    $high12H = if ($last12Hours.Count -gt 0) {
        ($last12Hours | Measure-Object -Property Percentage -Maximum).Maximum
    } else {
        $currentLevel
    }
    $low12H = if ($last12Hours.Count -gt 0) {
        ($last12Hours | Measure-Object -Property Percentage -Minimum).Minimum
    } else {
        $currentLevel
    }
    
    # 24H High and Low: Maximum and minimum of last 24 hours
    $high24H = if ($last24Hours.Count -gt 0) {
        ($last24Hours | Measure-Object -Property Percentage -Maximum).Maximum
    } else {
        $currentLevel
    }
    $low24H = if ($last24Hours.Count -gt 0) {
        ($last24Hours | Measure-Object -Property Percentage -Minimum).Minimum
    } else {
        $currentLevel
    }
    
    # 72H High and Low: Maximum and minimum of last 72 hours
    $high72H = if ($last72Hours.Count -gt 0) {
        ($last72Hours | Measure-Object -Property Percentage -Maximum).Maximum
    } else {
        $currentLevel
    }
    $low72H = if ($last72Hours.Count -gt 0) {
        ($last72Hours | Measure-Object -Property Percentage -Minimum).Minimum
    } else {
        $currentLevel
    }
    
    # Prepare data for sparklines using time-based binning (handles missing samples correctly)
    # Use current Pacific time as the end time for accurate binning
    $endTimePacific = Get-PacificTime
    
    # Filter data to relevant time ranges and generate sparklines
    # 12H: 30-minute bins (24 bins * 30 min = 12 hours)
    $twelveHoursAgoPacific = $endTimePacific.AddHours(-12)
    $data12H = $allData | Where-Object { $_.DateTime -ge $twelveHoursAgoPacific }
    $sparkline12HData = Get-Sparkline -DataPoints $data12H -BinSize (New-TimeSpan -Minutes 30) -EndTime $endTimePacific
    
    # 24H: 1-hour bins (24 bins * 1 hour = 24 hours)
    $twentyFourHoursAgoPacific = $endTimePacific.AddHours(-24)
    $data24H = $allData | Where-Object { $_.DateTime -ge $twentyFourHoursAgoPacific }
    $sparkline24HData = Get-Sparkline -DataPoints $data24H -BinSize (New-TimeSpan -Hours 1) -EndTime $endTimePacific
    
    # 72H: 3-hour bins (24 bins * 3 hours = 72 hours)
    $seventyTwoHoursAgoPacific = $endTimePacific.AddHours(-72)
    $data72H = $allData | Where-Object { $_.DateTime -ge $seventyTwoHoursAgoPacific }
    $sparkline72HData = Get-Sparkline -DataPoints $data72H -BinSize (New-TimeSpan -Hours 3) -EndTime $endTimePacific
    
    # Determine if we should show full output or specific lines
    $showFullOutput = -not ($level -or $banner -or $sma -or $capacity -or $lastfull -or $hl12 -or $hl24 -or $hl72 -or $s12 -or $s24 -or $s72)
    
    # Display output
    if ($banner -or $showFullOutput) {
        Write-Host "*** Portland Big Pipe Report ***" -ForegroundColor Green
        if ($showFullOutput) {
    Write-Host ""
        }
    }
    
    # Calculate padding to align values at column 16 (1-indexed)
    $targetColumn = 16
    
    # Current Level
    if ($level -or $showFullOutput) {
        $currentLevelFormatted = "$([math]::Round($currentLevel, 1))%"
        $currentLevelColor = Get-PercentageColor -Percentage $currentLevel
        $labelCurrent = "Current Level:"
        $paddingCurrent = " " * ($targetColumn - $labelCurrent.Length)
        Write-Host -NoNewline -ForegroundColor White $labelCurrent
        Write-Host -NoNewline $paddingCurrent
        Write-Host -ForegroundColor $currentLevelColor $currentLevelFormatted

        $dataAge = (Get-PacificTime) - $currentTime
        if ($dataSource -eq "gauge" -or $dataAge.TotalMinutes -gt 90) {
            $labelAsOf = "As of:"
            $paddingAsOf = " " * ($targetColumn - $labelAsOf.Length)
            Write-Host -NoNewline -ForegroundColor White $labelAsOf
            Write-Host -NoNewline $paddingAsOf
            Write-Host -ForegroundColor Yellow ($currentTime.ToString("M/d/yy h:mm tt"))
        }
        if ($dataSource -eq "gauge" -and $showFullOutput) {
            Write-Host -ForegroundColor Yellow "72-hour history is currently empty on the city feed."
        }
        
        # 100% Duration (only shown when current level is 100%)
        if (($capacity -or $showFullOutput) -and $currentLevel -eq 100 -and $null -ne $duration100Percent) {
            $durationFormatted = Format-Duration -Duration $duration100Percent
            $durationColor = [System.ConsoleColor]::Magenta
            $labelDuration = "100% Duration:"
            $paddingDuration = " " * ($targetColumn - $labelDuration.Length)
            Write-Host -NoNewline -ForegroundColor White $labelDuration
            Write-Host -NoNewline $paddingDuration
            Write-Host -ForegroundColor $durationColor $durationFormatted
        }
        
        # Last Full (only shown when current level is not 100% but there was a 100% reading)
        if (($lastfull -or $showFullOutput) -and $currentLevel -lt 100 -and $null -ne $timeSinceLastFull) {
            $lastFullFormatted = Format-Duration -Duration $timeSinceLastFull
            $lastFullColor = [System.ConsoleColor]::Magenta
            $labelLastFull = "Last Full:"
            $paddingLastFull = " " * ($targetColumn - $labelLastFull.Length)
            Write-Host -NoNewline -ForegroundColor White $labelLastFull
            Write-Host -NoNewline $paddingLastFull
            Write-Host -ForegroundColor $lastFullColor $lastFullFormatted
        }
    }
    
    # 12/24/72H SMA
    if ($sma -or $showFullOutput) {
        $sma12HFormatted = "$([math]::Round($sma12H, 1))%"
        $sma24HFormatted = "$([math]::Round($sma24H, 1))%"
        $sma72HFormatted = "$([math]::Round($sma72H, 1))%"
        $sma12HColor = Get-PercentageColor -Percentage $sma12H
        $sma24HColor = Get-PercentageColor -Percentage $sma24H
        $sma72HColor = Get-PercentageColor -Percentage $sma72H
        $labelSMA = "12/24/72H SMA:"
        $paddingSMA = " " * ($targetColumn - $labelSMA.Length)
        Write-Host -NoNewline -ForegroundColor White $labelSMA
        Write-Host -NoNewline $paddingSMA
        Write-Host -NoNewline -ForegroundColor $sma12HColor $sma12HFormatted
        Write-Host -NoNewline -ForegroundColor White "/"
        Write-Host -NoNewline -ForegroundColor $sma24HColor $sma24HFormatted
        Write-Host -NoNewline -ForegroundColor White "/"
        Write-Host -ForegroundColor $sma72HColor $sma72HFormatted
    }
    
    # 12H High/Low
    if ($hl12 -or $showFullOutput) {
        $high12HFormatted = "$([math]::Round($high12H, 1))%"
        $low12HFormatted = "$([math]::Round($low12H, 1))%"
        $high12HColor = Get-PercentageColor -Percentage $high12H
        $low12HColor = Get-PercentageColor -Percentage $low12H
        $label12H = "12H High/Low:"
        $padding12H = " " * ($targetColumn - $label12H.Length)
        Write-Host -NoNewline -ForegroundColor White $label12H
        Write-Host -NoNewline $padding12H
        Write-Host -NoNewline -ForegroundColor $high12HColor $high12HFormatted
        Write-Host -NoNewline -ForegroundColor White "/"
        Write-Host -ForegroundColor $low12HColor $low12HFormatted
    }
    
    # 24H High/Low
    if ($hl24 -or $showFullOutput) {
        $high24HFormatted = "$([math]::Round($high24H, 1))%"
        $low24HFormatted = "$([math]::Round($low24H, 1))%"
        $high24HColor = Get-PercentageColor -Percentage $high24H
        $low24HColor = Get-PercentageColor -Percentage $low24H
        $label24H = "24H High/Low:"
        $padding24H = " " * ($targetColumn - $label24H.Length)
        Write-Host -NoNewline -ForegroundColor White $label24H
        Write-Host -NoNewline $padding24H
        Write-Host -NoNewline -ForegroundColor $high24HColor $high24HFormatted
        Write-Host -NoNewline -ForegroundColor White "/"
        Write-Host -ForegroundColor $low24HColor $low24HFormatted
    }
    
    # 72H High/Low
    if ($hl72 -or $showFullOutput) {
        $high72HFormatted = "$([math]::Round($high72H, 1))%"
        $low72HFormatted = "$([math]::Round($low72H, 1))%"
        $high72HColor = Get-PercentageColor -Percentage $high72H
        $low72HColor = Get-PercentageColor -Percentage $low72H
        $label72H = "72H High/Low:"
        $padding72H = " " * ($targetColumn - $label72H.Length)
        Write-Host -NoNewline -ForegroundColor White $label72H
        Write-Host -NoNewline $padding72H
        Write-Host -NoNewline -ForegroundColor $high72HColor $high72HFormatted
        Write-Host -NoNewline -ForegroundColor White "/"
        Write-Host -ForegroundColor $low72HColor $low72HFormatted
    }
    
    if ($showFullOutput) {
    Write-Host ""
    }
    
    # Sparklines
    if ($s12 -or $showFullOutput) {
        Write-ColoredSparkline -Label "12H:" -Sparkline $sparkline12HData.Sparkline -BinnedValues $sparkline12HData.BinnedValues
    }
    if ($s24 -or $showFullOutput) {
        Write-ColoredSparkline -Label "24H:" -Sparkline $sparkline24HData.Sparkline -BinnedValues $sparkline24HData.BinnedValues
    }
    if ($s72 -or $showFullOutput) {
        Write-ColoredSparkline -Label "72H:" -Sparkline $sparkline72HData.Sparkline -BinnedValues $sparkline72HData.BinnedValues
    }
}
catch {
    Write-Error "Failed to reach the Big Pipe data source. Error: $_"
    exit 1
}
