@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "Command="
set "Period=5"
set "ClearMode=0"
set "SilentMode=0"
set "Limit=0"
set "RunCount=0"

if "%~1"=="" goto interactive

set "Command=%~1"
if not "%~2"=="" set "Period=%~2"
shift
shift

:parseFlags
if "%~1"=="" goto argsDone
if /I "%~1"=="-c" (
    set "ClearMode=1"
    shift
    goto parseFlags
)
if /I "%~1"=="-s" (
    set "SilentMode=1"
    shift
    goto parseFlags
)
if /I "%~1"=="-limit" (
    shift
    if "%~1"=="" (
        echo Warning: -limit missing value, ignoring.
        goto argsDone
    )
    set "Limit=%~1"
    shift
    goto parseFlags
)
echo Warning: Ignoring unknown option "%~1".
shift
goto parseFlags

:interactive
set /p "Command=Command: "
set /p "inputPeriod=Period (5, 15s, 5m, 1h) [default: 5]: "
if not "!inputPeriod!"=="" set "Period=!inputPeriod!"
set /p "inputFlags=Flags [-c -s -limit N] (optional): "
if not "!inputFlags!"=="" call :parseFlags !inputFlags!

:argsDone
if "%Command%"=="" (
    echo Error: Command is required.
    exit /b 1
)

call :resolvePeriodSeconds "%Period%"
if errorlevel 1 (
    echo Warning: Invalid period "%Period%". Using default 5 minutes.
    set "Period=5"
    call :resolvePeriodSeconds "5"
)

set /A LimitCheck=%Limit% 2>nul
if errorlevel 1 (
    echo Warning: -limit value "%Limit%" is invalid. Using 0 (unlimited).
    set "Limit=0"
)
if %Limit% lss 0 (
    echo Warning: -limit must be 0 or greater. Using 0 (unlimited).
    set "Limit=0"
)

if not "%SilentMode%"=="1" (
    call :describePeriod "%Period%"
    if %Limit% gtr 0 (
        echo Running "%Command%" every !PeriodDisplay!. Limit: %Limit% run^(s^). Press Ctrl+C to stop.
    ) else (
        echo Running "%Command%" every !PeriodDisplay!. Press Ctrl+C to stop.
    )
)

:loop
if %Limit% gtr 0 if %RunCount% geq %Limit% goto finished

if "%ClearMode%"=="1" cls
cmd /d /c "%Command%"
set "LastError=%errorlevel%"
set /A RunCount+=1

if not "%SilentMode%"=="1" (
    if not "%LastError%"=="0" echo Warning: Command failed with errorlevel %LastError%.
    call :describePeriod "%Period%"
    echo Waiting !PeriodDisplay! ^(run %RunCount%%LimitSuffix%^). Press Ctrl+C to stop.
)

timeout /t %PeriodSeconds% /nobreak >nul
goto loop

:finished
if not "%SilentMode%"=="1" echo Limit reached ^(%Limit% run^(s^)^). Exiting.
exit /b 0

:resolvePeriodSeconds
set "raw=%~1"
if "%raw%"=="" set "raw=5"
set "suffix=%raw:~-1%"
set "number=%raw%"
set "multiplier=60"

if /I "%suffix%"=="s" (
    set "number=%raw:~0,-1%"
    set "multiplier=1"
) else if /I "%suffix%"=="m" (
    set "number=%raw:~0,-1%"
    set "multiplier=60"
) else if /I "%suffix%"=="h" (
    set "number=%raw:~0,-1%"
    set "multiplier=3600"
)

set /A testNumber=%number% 2>nul
if errorlevel 1 exit /b 1
if %testNumber% lss 1 exit /b 1

set /A PeriodSeconds=%testNumber%*%multiplier%
exit /b 0

:describePeriod
set "displayRaw=%~1"
if "%displayRaw%"=="" set "displayRaw=5"
set "displaySuffix=%displayRaw:~-1%"
set "displayNumber=%displayRaw%"
set "displayUnit=minute(s)"

if /I "%displaySuffix%"=="s" (
    set "displayNumber=%displayRaw:~0,-1%"
    set "displayUnit=second(s)"
) else if /I "%displaySuffix%"=="m" (
    set "displayNumber=%displayRaw:~0,-1%"
    set "displayUnit=minute(s)"
) else if /I "%displaySuffix%"=="h" (
    set "displayNumber=%displayRaw:~0,-1%"
    set "displayUnit=hour(s)"
)

set "PeriodDisplay=%displayNumber% %displayUnit%"
set "LimitSuffix="
if %Limit% gtr 0 set "LimitSuffix=/%Limit%"
exit /b 0
