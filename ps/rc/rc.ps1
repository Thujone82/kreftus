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
