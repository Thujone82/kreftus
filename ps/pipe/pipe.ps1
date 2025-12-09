# Portland Big Pipe Report Script
# Parses 15-minute interval data and displays statistics with sparklines

param(
    [Alias("l")]
    [switch]$level,
    [Alias("b")]
    [switch]$banner,
    [switch]$sma,
    [switch]$s12,
    [switch]$s24,
    [switch]$s72
)

# Clear screen only if no specific output is requested
if (-not ($level -or $banner -or $sma -or $s12 -or $s24 -or $s72)) {
    Clear-Host
}

# URL for the raw data table
$url = "https://www.portlandoregon.gov/bes/bigpipe/data.cfm"

# --- Helper Function to Parse HTML and Extract All Data Points ---
function Get-PipeData {
    param ([string]$HtmlContent)
    
    $dataPoints = @()
    
    # Pattern to match HTML table structure: <time>date time</time> followed by <td>percentage</td>
    # This handles the actual HTML structure: <td><time>12/8/25 2:30 PM</time></td><td>37%</td>
    $pattern = "<time[^>]*>(\d{1,2}/\d{1,2}/\d{2,4})\s+(\d{1,2}:\d{2}\s+(?:AM|PM))</time></td>\s*<td>(\d{1,3}%)</td>"
    
    $regexMatches = [regex]::Matches($HtmlContent, $pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
    
    foreach ($match in $regexMatches) {
        $dateStr = $match.Groups[1].Value
        $timeStr = $match.Groups[2].Value
        $percentStr = $match.Groups[3].Value -replace '%', ''
        
        try {
            # Parse date and time
            $dateTimeStr = "$dateStr $timeStr"
            $dateTime = [DateTime]::Parse($dateTimeStr)
            
            # Parse percentage
            $percentage = [double]$percentStr
            
            $dataPoints += [PSCustomObject]@{
                DateTime = $dateTime
                Percentage = $percentage
            }
        }
        catch {
            # Skip malformed entries
            continue
        }
    }
    
    # Sort chronologically (oldest to newest)
    $dataPoints = $dataPoints | Sort-Object -Property DateTime
    
    return $dataPoints
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

# --- Helper Function to Generate Sparkline from Binned Data ---
function Get-Sparkline {
    param (
        [double[]]$Values,
        [int]$SamplesPerGlyph
    )
    
    if ($null -eq $Values -or $Values.Count -lt $SamplesPerGlyph) {
        return @{
            Sparkline = (" " * 24)
            BinnedValues = @()
        }
    }
    
    # Bin the data: average every N consecutive samples
    $binnedValues = @()
    for ($i = 0; $i -lt $Values.Count; $i += $SamplesPerGlyph) {
        $bin = $Values[$i..([Math]::Min($i + $SamplesPerGlyph - 1, $Values.Count - 1))]
        $avg = ($bin | Measure-Object -Average).Average
        $binnedValues += $avg
    }
    
    # Take the last 24 bins (most recent)
    if ($binnedValues.Count -gt 24) {
        $binnedValues = $binnedValues[-24..-1]
    }
    
    if ($binnedValues.Count -eq 0) {
        return @{
            Sparkline = (" " * 24)
            BinnedValues = @()
        }
    }
    
    $sparkChars = [char[]]([char]0x2581, [char]0x2582, [char]0x2583, [char]0x2584, [char]0x2585, [char]0x2586, [char]0x2587, [char]0x2588)
    $minValue = ($binnedValues | Measure-Object -Minimum).Minimum
    $maxValue = ($binnedValues | Measure-Object -Maximum).Maximum
    $valueRange = $maxValue - $minValue
    
    if ($valueRange -eq 0 -or $valueRange -lt 0.00000001) {
        return @{
            Sparkline = ([string]$sparkChars[0] * 24)
            BinnedValues = $binnedValues
        }
    }
    
    $sparkline = ""
    foreach ($value in $binnedValues) {
        $normalized = ($value - $minValue) / $valueRange
        $charIndex = [math]::Floor($normalized * ($sparkChars.Length - 1))
        $sparkline += $sparkChars[$charIndex]
    }
    
    # Pad to 24 if too short (pad left with spaces)
    # Note: binnedValues array stays as-is, only sparkline gets padded
    if ($sparkline.Length -lt 24) {
        $sparkline = $sparkline.PadLeft(24, ' ')
    }
    
    # Truncate to 24 if too long (keep rightmost 24)
    if ($sparkline.Length -gt 24) {
        $sparkline = $sparkline.Substring($sparkline.Length - 24)
        $binnedValues = $binnedValues[-24..-1]
    }
    
    return @{
        Sparkline = $sparkline
        BinnedValues = $binnedValues
    }
}

# --- Helper Function to Display Colored Sparkline ---
function Write-ColoredSparkline {
    param (
        [string]$Label,
        [string]$Sparkline,
        [double[]]$BinnedValues
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
            $color = Get-PercentageColor -Percentage $percentage
        } else {
            # Fallback for edge cases
            $color = [System.ConsoleColor]::White
        }
        
        Write-Host -NoNewline -ForegroundColor $color $char
    }
    
    Write-Host ""
}

try {
    # Fetch the webpage content
    $response = Invoke-WebRequest -Uri $url -UseBasicParsing
    
    # Parse all data points from HTML
    $allData = Get-PipeData -HtmlContent $response.Content
    
    if ($allData.Count -eq 0) {
        Write-Error "Could not parse any data points from the page."
        exit 1
    }
    
    # Calculate statistics
    $currentLevel = $allData[-1].Percentage
    $currentTime = $allData[-1].DateTime
    
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
    
    # Prepare data for sparklines (get percentage values in chronological order)
    $percentageValues = $allData | ForEach-Object { $_.Percentage }
    
    # Generate sparklines
    # 12H: 2 samples per glyph (30 min bins) - need last 48 samples (24 glyphs * 2)
    $last48Samples = if ($percentageValues.Count -ge 48) {
        $percentageValues[-48..-1]
    } else {
        $percentageValues
    }
    $sparkline12HData = Get-Sparkline -Values $last48Samples -SamplesPerGlyph 2
    
    # 24H: 4 samples per glyph (1 hour bins) - need last 96 samples (24 glyphs * 4)
    $last96Samples = if ($percentageValues.Count -ge 96) {
        $percentageValues[-96..-1]
    } else {
        $percentageValues
    }
    $sparkline24HData = Get-Sparkline -Values $last96Samples -SamplesPerGlyph 4
    
    # 72H: 12 samples per glyph (3 hour bins) - need last 288 samples (24 glyphs * 12)
    $last288Samples = if ($percentageValues.Count -ge 288) {
        $percentageValues[-288..-1]
    } else {
        $percentageValues
    }
    $sparkline72HData = Get-Sparkline -Values $last288Samples -SamplesPerGlyph 12
    
    # Determine if we should show full output or specific lines
    $showFullOutput = -not ($level -or $banner -or $sma -or $s12 -or $s24 -or $s72)
    
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
    if ($showFullOutput) {
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
        
        # 24H High/Low
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
        
        # 72H High/Low
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
