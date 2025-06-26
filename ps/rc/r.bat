@echo off
setlocal enabledelayedexpansion

rem — Get parameters
set "Command=%~1"
set "Period=%~2"

rem — If no command was passed in, prompt for both
if "%Command%"=="" (
    set /p "Command=Command: "
    set /p "inputPeriod=Period (minutes) [default: 5]: "

    rem — if the user actually typed something
    if not "!inputPeriod!"=="" (
        rem — try parsing as an integer
        set /A tempPeriod=!inputPeriod! 2>nul
        if errorlevel 1 (
            rem — not a valid integer
            set "Period=5"
        ) else (
            rem — use the parsed value
            set "Period=!tempPeriod!"
        )
    ) else (
        rem — no input, default
        set "Period=5"
    )
) else (
    rem — command-line supplied but no period ? default
    if "%Period%"=="" set "Period=5"
)
echo Running "%Command%" every %Period% minute(s). Press Ctrl+C to stop.

:loop
    rem — Execute the command
    call %Command%

    rem — Check for error
    if errorlevel 1 echo Warning: Command failed with errorlevel %errorlevel%.

    echo Running "Waiting %Period% minute(s). Press Ctrl+C to stop.

    rem — Sleep for Period*60 seconds
    set /A seconds=%Period%*60
    timeout /t %seconds% /nobreak >nul

goto loop
