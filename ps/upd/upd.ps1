# UPDate PowerShell Tool
# Supports dynamic date placeholders in Remote and Local paths using [dateformat] notation
# Example: http://radar.com?time=[MMddHH]-[mm].png becomes http://radar.com?time=112209-41.png
# Example: Local C:\downloads\[yyyy-MM]\ creates monthly folders like C:\downloads\2025-01\
param(
    [string[]]$JobNames = @(),
    [alias("a")][switch]$Auto,
    [switch]$Verbose
)

Write-Verbose "Script started with arguments: $($args -join ', ')"
Write-Verbose "Parameters processed - JobNames: $($JobNames -join ', '), Auto: $Auto"

# Enable verbose output if -Verbose flag is set
if ($Verbose -or $VerbosePreference -eq 'Continue') {
    $VerbosePreference = 'Continue'
}

# Process any job name arguments (exclude PowerShell built-in parameters)
$builtInParams = @('Debug', 'ErrorAction', 'WarningAction', 'InformationAction', 'ErrorVariable', 'WarningVariable', 'InformationVariable', 'OutVariable', 'OutBuffer', 'PipelineVariable', 'WhatIf', 'Confirm')

foreach ($arg in $args) {
    if ($arg -match '^-(.+)$') {
        $paramName = $matches[1]
        # Only process if it's not a built-in PowerShell parameter
        if ($paramName -notin $builtInParams) {
            Write-Verbose "Found job argument: -$paramName"
            $JobNames += $paramName
            Write-Verbose "JobNames now: $($JobNames -join ', ')"
        }
        else {
            Write-Verbose "Skipping built-in parameter: -$paramName"
        }
    }
}

# Global variables
$script:Jobs = @()
$script:ConfigPath = Join-Path $PSScriptRoot "upd.json"

# Initialize SelectedJobs only if it doesn't exist
if (-not (Get-Variable -Name "SelectedJobs" -Scope Script -ErrorAction SilentlyContinue)) {
    $script:SelectedJobs = @()
    Write-Verbose "Initialized SelectedJobs array"
}

# Color functions
function Write-ColorText {
    param(
        [string]$Text,
        [string]$Color = "White"
    )
    Write-Host $Text -ForegroundColor $Color
}

function Write-Cyan {
    param([string]$Text)
    Write-ColorText $Text "Cyan"
}

function Write-Green {
    param([string]$Text)
    Write-ColorText $Text "Green"
}

function Write-Red {
    param([string]$Text)
    Write-ColorText $Text "Red"
}

function Write-White {
    param([string]$Text)
    Write-ColorText $Text "White"
}

function Write-Yellow {
    param([string]$Text)
    Write-ColorText $Text "Yellow"
}

# Configuration management
function Import-Config {
    Write-Verbose "Loading configuration from: $script:ConfigPath"
    if (Test-Path $script:ConfigPath) {
        try {
            $configContent = Get-Content -Path $script:ConfigPath -Raw
            $configData = $configContent | ConvertFrom-Json
            $script:Jobs = @()
            
            # Convert PSCustomObjects back to hashtables
            foreach ($job in $configData.Jobs) {
                $script:Jobs += @{
                    Name = $job.Name
                    Remote = $job.Remote
                    Local = $job.Local
                }
            }
            
            Write-Verbose "Loaded $($script:Jobs.Count) jobs from configuration"
            if ($null -eq $script:Jobs) {
                $script:Jobs = @()
            }
        }
        catch {
            Write-Red "Error loading configuration: $($_.Exception.Message)"
            $script:Jobs = @()
        }
    }
    else {
        Write-Verbose "Configuration file not found, starting with empty jobs list"
        $script:Jobs = @()
    }
}

function Export-Config {
    try {
        # Convert hashtables to PSCustomObjects for JSON serialization
        $jobsForJson = @()
        foreach ($job in $script:Jobs) {
            $jobsForJson += [PSCustomObject]@{
                Name = $job.Name
                Remote = $job.Remote
                Local = $job.Local
            }
        }
        
        $configData = [PSCustomObject]@{
            Jobs = $jobsForJson
            Version = "1.0"
        }
        
        $configData | ConvertTo-Json -Depth 10 | Set-Content -Path $script:ConfigPath -Force
        return $true
    }
    catch {
        Write-Red "Error saving configuration: $($_.Exception.Message)"
        return $false
    }
}

function Initialize-Config {
    if (-not (Test-Path $script:ConfigPath)) {
        $script:Jobs = @()
        Export-Config
    }
}

# UI utilities
function Clear-Screen {
    Clear-Host
}

function Read-SingleKey {
    $key = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    return $key.Character.ToString().ToLower()
}

function Show-Header {
    param([string]$Title)
    Write-Cyan $Title
    Write-Host ""
}

# Job management functions
function Get-JobStatus {
    param([int]$Index)
    return $script:SelectedJobs -contains $Index
}

function Set-JobSelection {
    param([int]$Index)
    Write-Verbose "Toggling job selection for index: $Index"
    
    if ($script:SelectedJobs -contains $Index) {
        Write-Verbose "Removing index $Index from selection"
        $script:SelectedJobs = @($script:SelectedJobs | Where-Object { $_ -ne $Index })
    }
    else {
        Write-Verbose "Adding index $Index to selection"
        $script:SelectedJobs = @($script:SelectedJobs) + @($Index)
    }
    
    Write-Verbose "SelectedJobs now contains: $($script:SelectedJobs -join ', ')"
}

function Select-AllJobs {
    $script:SelectedJobs = 0..($script:Jobs.Count - 1)
}

function Clear-JobSelection {
    $script:SelectedJobs = @()
}

function Get-JobByName {
    param([string]$Name)
    Write-Verbose "Looking for job named '$Name'"
    Write-Verbose "Available jobs: $($script:Jobs.Name -join ', ')"
    for ($i = 0; $i -lt $script:Jobs.Count; $i++) {
        Write-Verbose "Comparing '$Name' with '$($script:Jobs[$i].Name)'"
        if ($script:Jobs[$i].Name -ieq $Name) {
            Write-Verbose "Found match at index $i"
            return $i
        }
    }
    Write-Verbose "No match found for job '$Name'"
    return -1
}

# Date placeholder expansion
function Expand-DatePlaceholders {
    param([string]$Path)
    
    $currentDate = Get-Date
    $result = $Path
    
    # Find all [dateformat] patterns
    $pattern = '\[([^\]]+)\]'
    $patternMatches = [regex]::Matches($Path, $pattern)
    
    foreach ($match in $patternMatches) {
        $formatString = $match.Groups[1].Value
        $dateValue = $currentDate.ToString($formatString)
        $result = $result.Replace($match.Value, $dateValue)
    }
    
    return $result
}

# File operations
function Invoke-UpdateJob {
    param([hashtable]$Job)
    
    try {
        $remote = Expand-DatePlaceholders -Path $Job.Remote
        $local = Expand-DatePlaceholders -Path $Job.Local
        $bytesTransferred = 0
        
        Write-Verbose "Updating job '$($Job.Name)' from '$remote' to '$local'"
        
        # Ensure local directory exists
        if (-not (Test-Path $local)) {
            Write-Verbose "Creating local directory: $local"
            New-Item -ItemType Directory -Path $local -Force | Out-Null
        }
        
        # Determine if remote is URL or local path
        if ($remote -match "^https?://") {
            # Download from URL
            Write-Verbose "Detected URL, downloading from: $remote"
            Write-White "Downloading from: $remote"
            $response = Invoke-WebRequest -Uri $remote -UseBasicParsing
            $fileName = Split-Path $remote -Leaf
            if ([string]::IsNullOrEmpty($fileName)) {
                $fileName = "downloaded_file"
            }
            $localPath = Join-Path $local $fileName
            Write-Verbose "Saving to: $localPath"
            [System.IO.File]::WriteAllBytes($localPath, $response.Content)
            $bytesTransferred = $response.Content.Length
            Write-Green "Downloaded: $localPath"
        }
        else {
            # Copy local file
            Write-Verbose "Detected local path, copying from: $remote"
            if (Test-Path $remote) {
                $fileName = Split-Path $remote -Leaf
                $localPath = Join-Path $local $fileName
                Write-Verbose "Copying to: $localPath"
                
                # Get file size before copying
                $fileInfo = Get-Item $remote
                $bytesTransferred = $fileInfo.Length
                
                Copy-Item -Path $remote -Destination $localPath -Force
                Write-White "Source: $remote"
                Write-White "Destination: $localPath"
            }
            else {
                Write-Red "Source file not found: $remote"
                return $false
            }
        }
        
        # Return success with bytes transferred information
        return @{
            Success = $true
            BytesTransferred = $bytesTransferred
        }
    }
    catch {
        Write-Red "Error updating job '$($Job.Name)': $($_.Exception.Message)"
        return $false
    }
}

function Start-SelectedJobs {
    if ($script:SelectedJobs.Count -eq 0) {
        Write-Red "No jobs selected for execution."
        return
    }
    
    Write-Verbose "Starting execution of $($script:SelectedJobs.Count) selected jobs"
    $successCount = 0
    $totalStartTime = Get-Date
    $totalBytesTransferred = 0
    
    foreach ($index in $script:SelectedJobs) {
        if ($index -lt $script:Jobs.Count) {
            Write-White "Executing job: $($script:Jobs[$index].Name)"
            Write-Verbose "Job details - Remote: $($script:Jobs[$index].Remote), Local: $($script:Jobs[$index].Local)"
            $jobResult = Invoke-UpdateJob -Job $script:Jobs[$index]
            if ($jobResult -is [hashtable] -and $jobResult.Success) {
                $successCount++
                $totalBytesTransferred += $jobResult.BytesTransferred
                Write-Verbose "Job '$($script:Jobs[$index].Name)' completed successfully"
            }
            elseif ($jobResult -eq $true) {
                $successCount++
                Write-Verbose "Job '$($script:Jobs[$index].Name)' completed successfully"
            }
            else {
                Write-Verbose "Job '$($script:Jobs[$index].Name)' failed"
            }
        }
    }
    
    $totalEndTime = Get-Date
    $totalDuration = $totalEndTime - $totalStartTime
    
    # Calculate speed and display informational line
    if ($totalBytesTransferred -gt 0 -and $totalDuration.TotalSeconds -gt 0) {
        $speedMBps = [math]::Round(($totalBytesTransferred / 1MB) / $totalDuration.TotalSeconds, 2)
        $speedKBps = [math]::Round(($totalBytesTransferred / 1KB) / $totalDuration.TotalSeconds, 2)
        
        # Format size with appropriate units
        $sizeStr = if ($totalBytesTransferred -ge 1GB) {
            $sizeGB = [math]::Round($totalBytesTransferred / 1GB, 2)
            "$sizeGB GB"
        } elseif ($totalBytesTransferred -ge 1MB) {
            $sizeMB = [math]::Round($totalBytesTransferred / 1MB, 2)
            "$sizeMB MB"
        } else {
            $sizeKB = [math]::Round($totalBytesTransferred / 1KB, 2)
            "$sizeKB KB"
        }
        
        $durationStr = if ($totalDuration.TotalSeconds -lt 1) { 
            "$([math]::Round($totalDuration.TotalMilliseconds, 0))ms" 
        } else { 
            "$([math]::Round($totalDuration.TotalSeconds, 2))s" 
        }
        
        $speedStr = if ($speedMBps -ge 1) { 
            "$speedMBps MB/s" 
        } else { 
            "$speedKBps KB/s" 
        }
        
        Write-Yellow "Transfer: $sizeStr in $durationStr ($speedStr)"
    }
    
    Write-Green "Completed: $successCount of $($script:SelectedJobs.Count) jobs successful"
}

# Function to validate job names (no spaces or special characters)
function Test-JobName {
    param([string]$Name)
    
    if ([string]::IsNullOrWhiteSpace($Name)) {
        return $false
    }
    
    # Check for spaces
    if ($Name -match '\s') {
        return $false
    }
    
    # Check for special characters (allow only alphanumeric, hyphens, and underscores)
    if ($Name -match '[^a-zA-Z0-9_-]') {
        return $false
    }
    
    return $true
}

# Job creation and editing
function New-Job {
    Write-White "Creating new job..."
    Write-Host ""
    
    do {
        $name = Read-Host "Name"
        if ([string]::IsNullOrWhiteSpace($name)) {
            Write-Red "Name cannot be empty."
            continue
        }
        if (-not (Test-JobName $name)) {
            Write-Red "Job name must contain only letters, numbers, hyphens, and underscores. No spaces or special characters allowed."
            continue
        }
        break
    } while ($true)
    
    $remote = Read-Host "Remote (file path or URL. Use [dateformat] for dynamic dates)"
    if ([string]::IsNullOrWhiteSpace($remote)) {
        Write-Red "Remote cannot be empty."
        return
    }
    
    $local = Read-Host "Local (directory path. Use [dateformat] for dynamic dates)"
    if ([string]::IsNullOrWhiteSpace($local)) {
        Write-Red "Local cannot be empty."
        return
    }
    
    # Validate local path
    try {
        if (-not (Test-Path $local)) {
            $create = Read-Host "Directory '$local' does not exist. Create it? (y/N)"
            if ($create -eq 'y' -or $create -eq 'Y') {
                New-Item -ItemType Directory -Path $local -Force | Out-Null
            }
            else {
                Write-Red "Job creation cancelled."
                return
            }
        }
    }
    catch {
        Write-Red "Invalid local path: $($_.Exception.Message)"
        return
    }
    
    # Add new job
    $newJob = @{
        Name = $name
        Remote = $remote
        Local = $local
    }
    
    $script:Jobs += $newJob
    if (Export-Config) {
        Write-Green "Job '$name' created successfully."
    }
    else {
        Write-Red "Failed to save job."
    }
}

function Edit-Job {
    param([int]$Index)
    
    if ($Index -ge $script:Jobs.Count) {
        Write-Red "Invalid job index."
        return
    }
    
    $job = $script:Jobs[$Index]
    Write-White "Editing job: $($job.Name)"
    Write-Host ""
    
    # Ask if user wants to remove the job
    Write-Host "Remove Job? [y/N]" -ForegroundColor Yellow -NoNewline
    $removeJob = Read-Host " "
    if ($removeJob -eq 'y' -or $removeJob -eq 'Y') {
        # Remove the job from the array
        $script:Jobs = $script:Jobs | Where-Object { $_ -ne $script:Jobs[$Index] }
        if (Export-Config) {
            Write-Green "Job '$($job.Name)' removed successfully."
        }
        else {
            Write-Red "Failed to make changes."
        }
        return
    }
    
    do {
        $name = Read-Host "Name [$($job.Name)]"
        if ([string]::IsNullOrWhiteSpace($name)) {
            $name = $job.Name
            break
        }
        if (-not (Test-JobName $name)) {
            Write-Red "Job name must contain only letters, numbers, hyphens, and underscores. No spaces or special characters allowed."
            continue
        }
        break
    } while ($true)
    
    $remote = Read-Host "Remote [$($job.Remote)] (Use [dateformat] for dates)"
    if ([string]::IsNullOrWhiteSpace($remote)) {
        $remote = $job.Remote
    }
    
    $local = Read-Host "Local [$($job.Local)] (Use [dateformat] for dates)"
    if ([string]::IsNullOrWhiteSpace($local)) {
        $local = $job.Local
    }
    
    # Update job
    $script:Jobs[$Index] = @{
        Name = $name
        Remote = $remote
        Local = $local
    }
    
    if (Export-Config) {
        Write-Green "Job updated successfully."
    }
    else {
        Write-Red "Failed to save changes."
    }
}

# Screen displays
function Show-UpdateScreen {
    while ($true) {
        Clear-Screen
        Show-Header "*** UPDate Tool ***"
        
        if ($script:Jobs.Count -eq 0) {
            Write-White "No jobs configured. Redirecting to Jobs management..."
            Start-Sleep -Seconds 2
            Show-JobsScreen
            return
        }
        
        # Display jobs
        Write-Verbose "Displaying jobs with SelectedJobs: [$($script:SelectedJobs -join ', ')]"
        for ($i = 0; $i -lt $script:Jobs.Count; $i++) {
            $job = $script:Jobs[$i]
            $isSelected = Get-JobStatus $i
            $color = if ($isSelected) { "Green" } else { "White" }
            Write-Verbose "Job $i ($($job.Name)) - Selected: $isSelected"
            Write-ColorText "$($i + 1)) $($job.Name)" $color
        }
        
        Write-Host ""
        
        # Command bar with proper highlighting
        $jobRange = if ($script:Jobs.Count -eq 1) { "1" } else { "1-$($script:Jobs.Count)" }
        Write-Host "Enable[" -ForegroundColor White -NoNewline
        Write-Host $jobRange -ForegroundColor Cyan -NoNewline
        Write-Host "] [" -ForegroundColor White -NoNewline
        Write-Host "A" -ForegroundColor Cyan -NoNewline
        Write-Host "]ll [" -ForegroundColor White -NoNewline
        Write-Host "J" -ForegroundColor Cyan -NoNewline
        Write-Host "]obs [" -ForegroundColor White -NoNewline
        Write-Host "E" -ForegroundColor Cyan -NoNewline
        Write-Host "/" -ForegroundColor White -NoNewline
        Write-Host "Enter" -ForegroundColor Cyan -NoNewline
        Write-Host "]xecute" -ForegroundColor White
        
        # Handle input
        $key = Read-SingleKey
        
        switch ($key) {
        { $_ -match '^[0-9]$' } {
            $num = [int]$_
            if ($num -ge 1 -and $num -le $script:Jobs.Count) {
                Set-JobSelection ($num - 1)
                continue
            }
        }
            'a' {
                Select-AllJobs
                continue
            }
            'j' {
                Show-JobsScreen
                return
            }
            'e' {
                if ($script:SelectedJobs.Count -gt 0) {
                    $confirm = Read-Host "Execute $($script:SelectedJobs.Count) selected job(s)? (y/N)"
                    if ($confirm -eq 'y' -or $confirm -eq 'Y') {
                        Start-SelectedJobs
                        Write-Host "Press any key to continue..." -ForegroundColor Yellow
                        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
                        continue
                    }
                    else {
                        continue
                    }
                }
                else {
                    Write-Red "No jobs selected."
                    Start-Sleep -Seconds 1
                    continue
                }
            }
            { $_ -eq [char]13 } {  # Enter key
                if ($script:SelectedJobs.Count -gt 0) {
                    $confirm = Read-Host "Execute $($script:SelectedJobs.Count) selected job(s)? (y/N)"
                    if ($confirm -eq 'y' -or $confirm -eq 'Y') {
                        Start-SelectedJobs
                        Write-Host "Press any key to continue..." -ForegroundColor Yellow
                        $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
                        continue
                    }
                    else {
                        continue
                    }
                }
                else {
                    Write-Red "No jobs selected."
                    Start-Sleep -Seconds 1
                    continue
                }
            }
            { $_ -eq [char]27 } {  # Esc key
                Write-White "Exiting..."
                return
            }
        }
    }
}

function Show-JobsScreen {
    Clear-Screen
    Show-Header "*** Jobs Management ***"
    
    # Display existing jobs
    for ($i = 0; $i -lt $script:Jobs.Count; $i++) {
        $job = $script:Jobs[$i]
        Write-White "$($i + 1)) $($job.Name)"
    }
    
    Write-Host ""
    
    # Command bar with proper highlighting
    $jobRange = if ($script:Jobs.Count -eq 0) { "" } elseif ($script:Jobs.Count -eq 1) { "1" } else { "1-$($script:Jobs.Count)" }
    if ($script:Jobs.Count -gt 0) {
        Write-Host "Edit[" -ForegroundColor White -NoNewline
        Write-Host $jobRange -ForegroundColor Cyan -NoNewline
        Write-Host "] [" -ForegroundColor White -NoNewline
        Write-Host "N" -ForegroundColor Cyan -NoNewline
        Write-Host "]ew [" -ForegroundColor White -NoNewline
        Write-Host "D" -ForegroundColor Cyan -NoNewline
        Write-Host "]one" -ForegroundColor White
    }
    else {
        Write-Host "[" -ForegroundColor White -NoNewline
        Write-Host "N" -ForegroundColor Cyan -NoNewline
        Write-Host "]ew [" -ForegroundColor White -NoNewline
        Write-Host "D" -ForegroundColor Cyan -NoNewline
        Write-Host "]one" -ForegroundColor White
    }
    
    # Handle input
    $key = Read-SingleKey
    
    switch ($key) {
        { $_ -match '^[0-9]$' } {
            $num = [int]$_
            if ($num -ge 1 -and $num -le $script:Jobs.Count) {
                Edit-Job ($num - 1)
                Write-Host "Press any key to continue..."
                $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
                Show-JobsScreen
            }
        }
        'n' {
            New-Job
            Write-Host "Press any key to continue..."
            $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
            Show-JobsScreen
        }
        'd' {
            Show-UpdateScreen
        }
        { $_ -eq [char]27 -or $_ -eq [char]13 } {  # Esc or Enter
            Show-UpdateScreen
        }
        default {
            Show-JobsScreen
        }
    }
}

# Command line argument processing
function Test-CommandLineArgs {
    Write-Verbose "JobNames count: $($JobNames.Count)"
    Write-Verbose "JobNames: $($JobNames -join ', ')"
    Write-Verbose "Auto flag: $Auto"
    
    if ($JobNames.Count -gt 0) {
        Write-Verbose "Processing command line job names: $($JobNames -join ', ')"
        foreach ($jobName in $JobNames) {
            Write-Verbose "Looking for job: '$jobName'"
            $index = Get-JobByName $jobName
            if ($index -ge 0) {
                $script:SelectedJobs += $index
                Write-Verbose "Pre-selected job: $jobName"
                Write-Verbose "SelectedJobs now contains: $($script:SelectedJobs -join ', ')"
            }
            else {
                Write-Red "Job not found: $jobName"
                Write-White "Available jobs: $($script:Jobs.Name -join ', ')"
            }
        }
    }
    
    if ($Auto) {
        Write-Verbose "Auto mode enabled"
        if ($script:SelectedJobs.Count -gt 0) {
            Write-White "Auto mode: Executing selected jobs..."
            Start-SelectedJobs
            return $true
        }
        else {
            Write-Red "Auto mode: No jobs selected."
            return $false
        }
    }
    
    return $false
}

# Main execution
function Main {
    Initialize-Config
    Import-Config
    
    Write-Verbose "Loaded $($script:Jobs.Count) jobs from configuration"
    
    # Process command line arguments
    if (Test-CommandLineArgs) {
        return
    }
    
    # Log pre-selected jobs if any
    if ($script:SelectedJobs.Count -gt 0) {
        $selectedJobNames = $script:SelectedJobs | ForEach-Object { $script:Jobs[$_].Name }
        Write-Verbose "Pre-selected jobs: $($selectedJobNames -join ', ')"
    }
    
    # Start interactive mode
    Show-UpdateScreen
}

# Run the main function
Main