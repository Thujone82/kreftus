<#
.SYNOPSIS
    Converts a string between ASCII and its binary representation.

.DESCRIPTION
    This script can be run in two modes:
    1. Direct Mode: Provide a string as an argument to the script for a one-off conversion.
    2. Interactive Mode: Run the script without arguments to enter a loop where you can convert multiple strings.

    The script automatically detects whether the input is ASCII or binary.
    - ASCII to Binary: 'Hi' -> '01001000 01101001'
    - Binary to ASCII: '01001000 01101001' -> 'Hi'

    Binary input can contain spaces, which will be ignored. Binary output is always formatted in 8-bit chunks.
    If binary input is provided that does not have a length that is a multiple of 8 (after removing spaces),
    it will be considered unexpected input.

.PARAMETER InputString
    The string to be converted. If omitted, the script enters interactive mode.

.EXAMPLE
    .\a2b.ps1 "Hello World"
    Converts the ASCII string "Hello World" to its binary representation.

.EXAMPLE
    .\a2b.ps1 "01001000 01101001"
    Converts the binary string "01001000 01101001" to "Hi".

.EXAMPLE
    .\a2b.ps1
    Starts the interactive conversion mode.
#>
[CmdletBinding()]
param (
    [string]$InputString
)

function Convert-ToBinary {
    param (
        [Parameter(Mandatory)]
        [string]$AsciiString
    )
    # Ensure the input string contains only valid ASCII characters (0-127).
    # Otherwise, it's not a true ASCII string and should be considered unexpected.
    if ($AsciiString -match '[^\x00-\x7F]') {
        return "Unexpected Input"
    }

    $bytes = [System.Text.Encoding]::ASCII.GetBytes($AsciiString)
    $binaryChunks = foreach ($byte in $bytes) {
        [System.Convert]::ToString($byte, 2).PadLeft(8, '0')
    }
    return $binaryChunks -join ' '
}

function Convert-ToAscii {
    param (
        [Parameter(Mandatory)]
        [string]$BinaryString
    )
    try {
        # Remove all spaces and validate
        $cleanedBinary = $BinaryString -replace '\s'
        if ($cleanedBinary.Length % 8 -ne 0) {
            return "Unexpected Input"
        }

        # The -split operator with a capture group '(.{8})' splits the string by 8-character chunks
        # and includes the delimiters (the chunks themselves) in the result.
        # The Where-Object then filters out the empty strings that result from the split.
        $binaryChunks = $cleanedBinary -split '(.{8})' | Where-Object { $_ }

        $bytes = foreach ($chunk in $binaryChunks) {
            [System.Convert]::ToByte($chunk, 2)
        }

        return [System.Text.Encoding]::ASCII.GetString($bytes)
    }
    catch {
        return "Unexpected Input"
    }
}

function Invoke-Conversion {
    param (
        [Parameter(Mandatory)]
        [string]$StringToConvert
    )

    # Check if the string consists only of 0, 1, and spaces.
    if ($StringToConvert -match '^[01\s]+$') {
        # It looks like binary, try to convert to ASCII.
        # The conversion function will handle validation (e.g., length % 8).
        return Convert-ToAscii -BinaryString $StringToConvert
    }
    else {
        # It's not purely binary characters, so treat as ASCII.
        return Convert-ToBinary -AsciiString $StringToConvert
    }
}

# --- Main Script Logic ---

if (-not [string]::IsNullOrEmpty($InputString)) {
    # Direct mode: an input string was provided as a parameter
    Invoke-Conversion -StringToConvert $InputString
}
else {
    # Interactive mode
    Write-Host "ASCII 2 BINARY" -ForegroundColor Yellow
    while ($true) {
        $userInput = Read-Host "Convert string:"
        if ([string]::IsNullOrEmpty($userInput)) {
            # Exit on empty input
            break
        }
        $result = Invoke-Conversion -StringToConvert $userInput
        Write-Host $result
    }
}