@echo off
setlocal EnableExtensions EnableDelayedExpansion

rem Build stamp - change when testing so you know this file is the one running
set "RC_BUILD=20260529-debug3"

set "Command="
set "Period=5"
set "ClearMode=0"
set "SilentMode=0"
set "DebugMode=0"
set "Limit=0"
set "RunCount=0"
set "DebugLog=%TEMP%\rc-r-debug.log"

if "%~1"=="" goto interactive

set "Command=%~1"
if not "%~2"=="" (
    set "Period=%~2"
    for /f "delims=" %%t in ("!Period!") do set "Period=%%t"
)
shift
shift
goto parseFlags

:interactive
set /p "Command=Command: "
set /p "inputPeriod=Period (5, 15s, 5m, 1h) [default: 5]: "
if not "!inputPeriod!"=="" (
    for /f "delims=" %%t in ("!inputPeriod!") do set "Period=%%t"
)
set /p "inputFlags=Flags [-c -q -limit N -debug] (optional): "
if not "!inputFlags!"=="" call :parseFlagArgs !inputFlags!
goto argsDone

:parseFlags
if "%~1"=="" goto argsDone
if /I "%~1"=="-c" set "ClearMode=1" & shift & goto parseFlags
if /I "%~1"=="-q" set "SilentMode=1" & shift & goto parseFlags
if /I "%~1"=="-quiet" set "SilentMode=1" & shift & goto parseFlags
if /I "%~1"=="-debug" set "DebugMode=1" & shift & goto parseFlags
if /I "%~1"=="-limit" goto parseFlagsLimit
echo Warning: Ignoring unknown option "%~1"
shift
goto parseFlags
:parseFlagsLimit
shift
if "%~1"=="" goto parseFlagsLimitMissing
set "Limit=%~1"
shift
goto parseFlags
:parseFlagsLimitMissing
echo Warning: -limit missing value, ignoring
goto argsDone

:parseFlagArgs
if "%~1"=="" exit /b 0
if /I "%~1"=="-c" set "ClearMode=1" & shift & goto parseFlagArgs
if /I "%~1"=="-q" set "SilentMode=1" & shift & goto parseFlagArgs
if /I "%~1"=="-quiet" set "SilentMode=1" & shift & goto parseFlagArgs
if /I "%~1"=="-debug" set "DebugMode=1" & shift & goto parseFlagArgs
if /I "%~1"=="-limit" goto parseFlagArgsLimit
echo Warning: Ignoring unknown option "%~1"
shift
goto parseFlagArgs
:parseFlagArgsLimit
shift
if "%~1"=="" exit /b 0
set "Limit=%~1"
shift
goto parseFlagArgs

:argsDone
if "!DebugMode!"=="1" echo RC simplified !RC_BUILD! - debug on, log: !DebugLog!
call :debugLog "=== argsDone build=!RC_BUILD! ==="
call :debugLog "Command=[!Command!] Period=[!Period!] Limit=!Limit! Clear=!ClearMode! Silent=!SilentMode!"

if "!Command!"=="" goto errNoCommand

call :resolvePeriodSeconds "!Period!"
set "rpCode=!errorlevel!"
call :debugLog "resolvePeriodSeconds code=!rpCode! raw=[!rpRaw!] suffix=[!rpSuffix!] number=[!rpNumber!] mult=!rpMult! sec=!PeriodSeconds!"
if !rpCode! neq 0 goto useDefaultPeriod
goto periodReady

:useDefaultPeriod
echo Warning: Invalid period "!Period!" - using default 5 minutes
set "Period=5"
call :resolvePeriodSeconds "5"
set "rpCode=!errorlevel!"
call :debugLog "fallback resolve code=!rpCode! sec=!PeriodSeconds!"
if !rpCode! neq 0 goto errBadPeriod

:periodReady
set /A LimitCheck=!Limit! 2>nul
if errorlevel 1 goto warnBadLimit
goto limitOk
:warnBadLimit
echo Warning: -limit value "!Limit!" is invalid - using unlimited
set "Limit=0"
:limitOk
if !Limit! lss 0 (
    echo Warning: -limit must be 0 or greater - using unlimited
    set "Limit=0"
)

if "!SilentMode!"=="1" goto loop

call :describePeriod "!Period!"
echo Running "!Command!" every !PeriodDisplay!
if !Limit! gtr 0 echo Limit: !Limit! runs
echo Press Ctrl+C to stop
if "!DebugMode!"=="1" echo Debug log: !DebugLog!

:loop
if !Limit! gtr 0 if !RunCount! geq !Limit! goto finished

if "!ClearMode!"=="1" cls
call :debugLog "run !RunCount! cmd=[!Command!]"
cmd /d /c "!Command!"
set "LastError=!errorlevel!"
set /A RunCount+=1

if "!SilentMode!"=="1" goto loopWait
if not "!LastError!"=="0" echo Warning: Command failed with errorlevel !LastError!
call :describePeriod "!Period!"
echo Waiting !PeriodDisplay! - run !RunCount!!LimitSuffix!
echo Press Ctrl+C to stop

:loopWait
timeout /t !PeriodSeconds! /nobreak >nul
goto loop

:finished
if not "!SilentMode!"=="1" echo Limit reached - !Limit! runs - exiting
call :debugLog "finished after !RunCount! runs"
exit /b 0

:errNoCommand
echo Error: Command is required
exit /b 1

:errBadPeriod
echo Error: Could not parse period (even default 5). See debug log: !DebugLog!
exit /b 1

:debugLog
if not "!DebugMode!"=="1" exit /b 0
echo [DEBUG] %*
>>"!DebugLog!" echo [%date% %time%] %*
exit /b 0

:resolvePeriodSeconds
set "rpRaw=%~1"
if "!rpRaw!"=="" set "rpRaw=5"
call :normalizePeriodRaw
set "rpSuffix=!rpRaw:~-1!"
set "rpNumber=!rpRaw!"
set "rpMult=60"
if /I "!rpSuffix!"=="s" set "rpNumber=!rpRaw:~0,-1!"& set "rpMult=1"& goto rp_validate
if /I "!rpSuffix!"=="m" set "rpNumber=!rpRaw:~0,-1!"& set "rpMult=60"& goto rp_validate
if /I "!rpSuffix!"=="h" set "rpNumber=!rpRaw:~0,-1!"& set "rpMult=3600"& goto rp_validate
:rp_validate
set /A rpTest=!rpNumber! 2>nul
if errorlevel 1 exit /b 1
if !rpTest! lss 1 exit /b 1
set /A PeriodSeconds=!rpTest!*!rpMult!
exit /b 0

rem If last character is not s/m/h, strip one char (handles set /p CR and stray punctuation)
:normalizePeriodRaw
set "last=!rpRaw:~-1!"
if /I "!last!"=="s" exit /b 0
if /I "!last!"=="m" exit /b 0
if /I "!last!"=="h" exit /b 0
if "!rpRaw:~0,-1!"=="" exit /b 0
set "rpRaw=!rpRaw:~0,-1!"
goto normalizePeriodRaw

:describePeriod
set "displayRaw=%~1"
if "!displayRaw!"=="" set "displayRaw=5"
for /f "delims=" %%t in ("!displayRaw!") do set "displayRaw=%%t"
set "displaySuffix=!displayRaw:~-1!"
set "displayNumber=!displayRaw!"
set "displayUnit=minutes"
if /I "!displaySuffix!"=="s" set "displayNumber=!displayRaw:~0,-1!"& set "displayUnit=seconds"& goto dp_done
if /I "!displaySuffix!"=="m" set "displayNumber=!displayRaw:~0,-1!"& set "displayUnit=minutes"& goto dp_done
if /I "!displaySuffix!"=="h" set "displayNumber=!displayRaw:~0,-1!"& set "displayUnit=hours"& goto dp_done
:dp_done
set "PeriodDisplay=!displayNumber! !displayUnit!"
set "LimitSuffix="
if !Limit! gtr 0 set "LimitSuffix=/!Limit!"
exit /b 0
