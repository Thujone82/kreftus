# a2b - ASCII to Binary Converter

## Description
a2b.ps1 is a simple and efficient PowerShell utility for converting strings between ASCII and their binary representation. The script automatically detects whether the input is an ASCII string or a binary string and performs the appropriate conversion.

It can be run in two modes:
1.  **Direct Mode:** Provide a string directly as a command-line argument for a quick, one-off conversion.
2.  **Interactive Mode:** Run the script without arguments to enter a console loop where you can convert multiple strings.

## Features
-   **Automatic Detection:** Intelligently determines if the input is ASCII or binary.
-   **Bidirectional Conversion:** Converts ASCII to binary and binary to ASCII.
-   **Formatted Output:** Binary output is neatly formatted into 8-bit chunks separated by spaces (e.g., 01001000 01101001).
-   **Flexible Input:** Tolerates and removes spaces from binary input strings for easier use.
-   **Input Validation:** Rejects non-ASCII characters and binary strings that are not a valid multiple of 8 bits, returning "Unexpected Input".
-   **Two Modes of Operation:** Supports both direct command-line execution and an interactive console mode.

## Requirements
-   PowerShell

## Usage

### Direct Mode
To convert a string from the command line, pass it as an argument to the script.

**Convert ASCII to Binary:**
```powershell
.\a2b.ps1 "Hello World"
```
Output:
`01001000 01100101 01101100 01101100 01101111 00100000 01010111 01101111 01110010 01101100 01100100`

**Convert Binary to ASCII:**
```powershell
.\a2b.ps1 "01001000 01101001"
```
Output:
`Hi`

### Interactive Mode
To start the interactive mode, simply run the script without any arguments.

```powershell
.\a2b.ps1
```

The script will display a banner and prompt you for input. To exit, just press Enter on an empty line.

## Files
-   `a2b.ps1`: The main script file.
-   `README.txt`: This file.