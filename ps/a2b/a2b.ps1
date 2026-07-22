<#
.SYNOPSIS
    Converts a string between ASCII and its binary representation.

.DESCRIPTION
    This script can be run in three modes:
    1. Direct Mode: Provide a string as an argument for a one-off conversion.
    2. Pipeline Mode: Pipe strings into the script (one conversion per input object).
    3. Interactive Mode: Run with no arguments to enter a loop and convert multiple strings.

    By default the script auto-detects whether the input is ASCII or binary.
    - ASCII to Binary: 'Hi' -> '01001000 01101001'
    - Binary to ASCII: '01001000 01101001' -> 'Hi'

    Strings that contain only 0, 1, and whitespace are treated as binary in auto mode.
    Use -ToBinary to force ASCII-to-binary for those inputs (e.g. "01").

    Binary input may contain spaces (ignored). Binary output is formatted in 8-bit chunks.
    Invalid input (non-ASCII characters, binary length not a multiple of 8, empty after
    cleaning) prints a short red message. Direct and pipeline modes exit with code 1 on failure.

.PARAMETER InputString
    The string to convert. If omitted (and nothing is piped), the script enters interactive mode.

.PARAMETER ToBinary
    Force ASCII-to-binary conversion, even when the string looks like binary.

.PARAMETER ToAscii
    Force binary-to-ASCII conversion.

.PARAMETER Help
    Show this help text and exit.

.EXAMPLE
    .\a2b.ps1 "Hello World"
    Converts the ASCII string "Hello World" to its binary representation.

.EXAMPLE
    .\a2b.ps1 "01001000 01101001"
    Converts the binary string "01001000 01101001" to "Hi".

.EXAMPLE
    .\a2b.ps1 "01" -ToBinary
    Forces ASCII-to-binary for the characters '0' and '1'.

.EXAMPLE
    "Hi" | .\a2b.ps1
    Converts piped input from ASCII to binary.

.EXAMPLE
    .\a2b.ps1
    Starts interactive conversion mode.

.EXAMPLE
    .\a2b.ps1 -Help
    Shows help for the script.
#>
[CmdletBinding(DefaultParameterSetName = 'Auto')]
param (
    [Parameter(Position = 0, ValueFromPipeline, ParameterSetName = 'Auto')]
    [Parameter(Position = 0, ValueFromPipeline, ParameterSetName = 'ToBinary')]
    [Parameter(Position = 0, ValueFromPipeline, ParameterSetName = 'ToAscii')]
    [string]$InputString,

    [Parameter(ParameterSetName = 'ToBinary')]
    [switch]$ToBinary,

    [Parameter(ParameterSetName = 'ToAscii')]
    [switch]$ToAscii,

    [Parameter(ParameterSetName = 'Help')]
    [Alias('h')]
    [switch]$Help
)

begin {
    Set-StrictMode -Version Latest

    $script:HadInput = $false
    $script:HadFailure = $false
    $script:ShowHelpOnly = $false

    if ($Help) {
        Get-Help -Name $PSCommandPath -Detailed
        $script:ShowHelpOnly = $true
        return
    }

    function ConvertTo-Binary {
        param (
            [Parameter(Mandatory)]
            [AllowEmptyString()]
            [string]$AsciiString
        )

        if ([string]::IsNullOrWhiteSpace($AsciiString)) {
            throw 'Unexpected Input: empty or whitespace-only string.'
        }

        if ($AsciiString -match '[^\x00-\x7F]') {
            throw 'Unexpected Input: non-ASCII characters are not supported.'
        }

        $bytes = [System.Text.Encoding]::ASCII.GetBytes($AsciiString)
        $binaryChunks = foreach ($byte in $bytes) {
            [System.Convert]::ToString($byte, 2).PadLeft(8, '0')
        }
        return ($binaryChunks -join ' ')
    }

    function ConvertTo-Ascii {
        param (
            [Parameter(Mandatory)]
            [AllowEmptyString()]
            [string]$BinaryString
        )

        $cleanedBinary = $BinaryString -replace '\s'
        if ([string]::IsNullOrEmpty($cleanedBinary)) {
            throw 'Unexpected Input: empty or whitespace-only binary string.'
        }

        if ($cleanedBinary -notmatch '^[01]+$') {
            throw 'Unexpected Input: binary strings may contain only 0, 1, and whitespace.'
        }

        if ($cleanedBinary.Length % 8 -ne 0) {
            throw 'Unexpected Input: binary length must be a multiple of 8 after removing spaces.'
        }

        $bytes = for ($i = 0; $i -lt $cleanedBinary.Length; $i += 8) {
            [System.Convert]::ToByte($cleanedBinary.Substring($i, 8), 2)
        }

        return [System.Text.Encoding]::ASCII.GetString($bytes)
    }

    function Invoke-Conversion {
        param (
            [Parameter(Mandatory)]
            [AllowEmptyString()]
            [string]$StringToConvert,

            [ValidateSet('Auto', 'ToBinary', 'ToAscii')]
            [string]$Mode = 'Auto'
        )

        switch ($Mode) {
            'ToBinary' { return ConvertTo-Binary -AsciiString $StringToConvert }
            'ToAscii'  { return ConvertTo-Ascii -BinaryString $StringToConvert }
            default {
                if ($StringToConvert -match '^[01\s]+$') {
                    return ConvertTo-Ascii -BinaryString $StringToConvert
                }
                return ConvertTo-Binary -AsciiString $StringToConvert
            }
        }
    }

    function Get-ConversionMode {
        if ($ToBinary) { return 'ToBinary' }
        if ($ToAscii) { return 'ToAscii' }
        return 'Auto'
    }

    function Write-ConversionResult {
        param (
            [Parameter(Mandatory)]
            [AllowEmptyString()]
            [string]$StringToConvert,

            [Parameter(Mandatory)]
            [ValidateSet('Auto', 'ToBinary', 'ToAscii')]
            [string]$Mode,

            [switch]$AsHostOutput
        )

        try {
            $result = Invoke-Conversion -StringToConvert $StringToConvert -Mode $Mode
            if ($AsHostOutput) {
                Write-Host $result
            }
            else {
                Write-Output $result
            }
        }
        catch {
            $script:HadFailure = $true
            Write-Host $_.Exception.Message -ForegroundColor Red
        }
    }

    $script:ConversionMode = Get-ConversionMode
}

process {
    if ($script:ShowHelpOnly) {
        return
    }

    if ($PSBoundParameters.ContainsKey('InputString')) {
        $script:HadInput = $true
        Write-ConversionResult -StringToConvert $InputString -Mode $script:ConversionMode
    }
}

end {
    if ($script:ShowHelpOnly) {
        return
    }

    if ($script:HadInput) {
        if ($script:HadFailure) {
            exit 1
        }
        return
    }

    # Interactive mode
    Write-Host 'ASCII 2 BINARY' -ForegroundColor Yellow
    Write-Host 'Enter a string to convert. Empty line, q, quit, or exit to leave.' -ForegroundColor DarkGray

    while ($true) {
        $userInput = Read-Host 'Convert string'
        if ([string]::IsNullOrEmpty($userInput)) {
            break
        }
        if ($userInput -match '^(?i)(q|quit|exit)$') {
            break
        }
        Write-ConversionResult -StringToConvert $userInput -Mode $script:ConversionMode -AsHostOutput
    }
}
