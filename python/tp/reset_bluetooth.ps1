<#
.SYNOPSIS
    Reset Windows Bluetooth / BLE scanning without rebooting.

.DESCRIPTION
    TemPy in-app radio toggle only flips the WinRT Bluetooth radio state.
    When bleak logs Watcher status STOPPED and 0 BLE devices are found,
    the adapter stack usually needs a deeper reset:

      1. Realtek Bluetooth Adapter (PnP disable/enable) - most important
      2. Microsoft Bluetooth LE Enumerator (PnP disable/enable)
      3. Restart bthserv (30s stop timeout, then force-kill pid if stuck)
      4. Optional: restart RmSvc (often blocked while in use)

    Run this script in an elevated (Administrator) PowerShell window.

.EXAMPLE
    .\reset_bluetooth.ps1
#>

$ErrorActionPreference = 'Continue'

function Test-Admin {
    $current = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($current)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Restart-BthSupportService {
    param(
        [int]$StopTimeoutSeconds = 30
    )
    $name = 'bthserv'
    $svc = Get-Service -Name $name -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "  skip $name (not installed)" -ForegroundColor DarkGray
        return $false
    }

    Write-Host "  restart $name ($($svc.DisplayName))" -ForegroundColor Cyan

    if ($svc.Status -in @('Running', 'StopPending', 'StartPending')) {
        Write-Host "  stop $name (wait up to ${StopTimeoutSeconds}s)" -ForegroundColor Cyan
        try {
            Stop-Service -Name $name -Force -ErrorAction SilentlyContinue
        }
        catch {
            Write-Host "  warn: Stop-Service $name - $($_.Exception.Message)" -ForegroundColor Yellow
        }

        $deadline = (Get-Date).AddSeconds($StopTimeoutSeconds)
        do {
            $status = (Get-Service -Name $name).Status
            if ($status -eq 'Stopped') {
                break
            }
            Start-Sleep -Seconds 1
        } while ((Get-Date) -lt $deadline)

        $status = (Get-Service -Name $name).Status
        if ($status -ne 'Stopped') {
            $processId = (Get-CimInstance Win32_Service -Filter "Name='$name'" -ErrorAction SilentlyContinue).ProcessId
            if ($processId -and [int]$processId -gt 0) {
                Write-Host "  force kill $name pid $processId (stop timed out)" -ForegroundColor Yellow
                & taskkill.exe /F /PID $processId 2>$null | Out-Null
                Start-Sleep -Seconds 2
            }
            else {
                Write-Host "  warn: $name still $status and no service pid to kill" -ForegroundColor Yellow
            }
        }
    }

    $svc = Get-Service -Name $name
    if ($svc.Status -ne 'Running') {
        Write-Host "  start $name" -ForegroundColor Cyan
        try {
            Start-Service -Name $name -ErrorAction Stop
        }
        catch {
            Write-Host "  error: could not start $name - $($_.Exception.Message)" -ForegroundColor Red
            return $false
        }
    }

    $finalStatus = (Get-Service -Name $name).Status
    Write-Host "  ok $name ($finalStatus)" -ForegroundColor DarkGray
    return ($finalStatus -eq 'Running')
}

function Restart-ServiceSafe {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    $svc = Get-Service -Name $Name -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "  skip $Name (not installed)" -ForegroundColor DarkGray
        return $false
    }
    Write-Host "  restart $Name ($($svc.DisplayName))" -ForegroundColor Cyan
    try {
        Restart-Service -Name $Name -Force -ErrorAction Stop
        Write-Host "  ok $Name" -ForegroundColor DarkGray
        return $true
    }
    catch {
        Write-Host "  warn: could not restart $Name ($($_.Exception.Message))" -ForegroundColor Yellow
        Write-Host "        continuing - PnP adapter reset usually matters more" -ForegroundColor DarkGray
        return $false
    }
}

function Reset-PnpDeviceSafe {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FriendlyName,
        [int]$SettleSeconds = 5
    )
    $dev = Get-PnpDevice | Where-Object { $_.FriendlyName -eq $FriendlyName } | Select-Object -First 1
    if (-not $dev) {
        Write-Host "  skip $FriendlyName (not found)" -ForegroundColor DarkGray
        return $false
    }
    try {
        Write-Host "  disable $FriendlyName" -ForegroundColor Cyan
        Disable-PnpDevice -InstanceId $dev.InstanceId -Confirm:$false -ErrorAction Stop
        Start-Sleep -Seconds $SettleSeconds
        Write-Host "  enable $FriendlyName" -ForegroundColor Cyan
        Enable-PnpDevice -InstanceId $dev.InstanceId -Confirm:$false -ErrorAction Stop
        Start-Sleep -Seconds $SettleSeconds
        Write-Host "  ok $FriendlyName" -ForegroundColor DarkGray
        return $true
    }
    catch {
        Write-Host "  error: $FriendlyName - $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

if (-not (Test-Admin)) {
    Write-Host "Run this script in an elevated PowerShell (Run as administrator)." -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "Resetting Bluetooth / BLE stack (no reboot)..." -ForegroundColor White
Write-Host ""

Write-Host "[1/4] Reset Realtek Bluetooth Adapter" -ForegroundColor Yellow
$adapterOk = Reset-PnpDeviceSafe -FriendlyName 'Realtek Bluetooth Adapter'

Write-Host ""
Write-Host "[2/4] Reset Microsoft Bluetooth LE Enumerator" -ForegroundColor Yellow
$leOk = Reset-PnpDeviceSafe -FriendlyName 'Microsoft Bluetooth LE Enumerator'

Write-Host ""
Write-Host "[3/4] Restart services (best effort)" -ForegroundColor Yellow
Restart-BthSupportService -StopTimeoutSeconds 30 | Out-Null
Restart-ServiceSafe -Name 'RmSvc' | Out-Null
Start-Sleep -Seconds 3

Write-Host ""
Write-Host "[4/4] Wait for stack to settle" -ForegroundColor Yellow
Start-Sleep -Seconds 8

Write-Host ""
if ($adapterOk -or $leOk) {
    Write-Host "Done. Adapter reset complete." -ForegroundColor Green
} else {
    Write-Host "Done, but adapter reset steps failed - check errors above." -ForegroundColor Yellow
}
Write-Host "Run TemPy when ready." -ForegroundColor DarkGray
