<#
.SYNOPSIS
Calculates the nth Fibonacci number, using a local database for caching.

.DESCRIPTION
This script takes an integer 'n' as a command-line argument. It checks a local
'fibonacci.db' file for the requested Fibonacci number. If found, it returns
the cached value. If not, it computes the number iteratively, stores all newly
calculated values (from the last known to the requested 'n') in the database,
and then returns the result. The database file is created in the same directory
as the script if it doesn't exist.

.PARAMETER n
The position in the Fibonacci sequence (a non-negative integer).

.EXAMPLE
.\Get-Fibonacci.ps1 10
# Computes F(10), caches values up to F(10) if not already present, and outputs 55.

.EXAMPLE
.\Get-Fibonacci.ps1 5
# Computes F(5), caches values up to F(5) if not already present, and outputs 5.
# If run again with 5, it will retrieve F(5) directly from the cache.

.NOTES
Author: Your Name/AI
Date: 2023-10-27
Version: 1.0

The fibonacci.db file stores entries in the format 'index:value', one per line.
For example:
0:0
1:1
2:1
3:2
...
#>
param(
    [Parameter(Mandatory=$true, Position=0)]
    [int]$n
)

# --- Configuration ---
$scriptDir = $PSScriptRoot
$dbFile = Join-Path $scriptDir "fibonacci.db"
$fibCache = @{} # Hashtable to store computed Fibonacci numbers in memory
$newlyComputedEntries = @() # Array to hold new entries to be written to DB

# --- Input Validation ---
if ($n -lt 0) {
    Write-Error "Input must be a non-negative integer."
    exit 1
}

# --- Handle Base Cases Directly ---
# These are the simplest cases and can be returned immediately without DB lookup
if ($n -eq 0) {
    Write-Output (0).ToString("F0")
    exit 0
}
if ($n -eq 1) {
    Write-Output (1).ToString("F0")
    exit 0
}

# --- Database Loading ---
if (Test-Path $dbFile) {
    try {
        Get-Content $dbFile | ForEach-Object {
            $line = $_.Trim()
            # Parse lines in 'index:value' format, handling both integer and scientific notation values
            if ($line -match "^(\d+):([+-]?\d*\.?\d+E?[+-]?\d*)$") {
                $index = [int]$matches[1]
                # Parse value as double to handle scientific notation, then convert to appropriate integer type
                $value = [double]$matches[2]
                $fibCache[$index] = $value
            }
        }
    } catch {
        Write-Warning ("Could not read or parse '{0}'. Starting with an empty cache." -f $dbFile)
        $fibCache.Clear() # Clear cache if there was an error reading the file
    }
}

# --- Check Cache Before Computation ---
if ($fibCache.ContainsKey($n)) {
    Write-Output ($fibCache[$n]).ToString("F0")
    exit 0
}

# --- Compute Missing Fibonacci Numbers ---

# Ensure base cases 0 and 1 are in the cache if they are not already there.
# This is crucial for starting the iterative calculation.
if (-not $fibCache.ContainsKey(0)) {
    $fibCache[0] = 0
    $newlyComputedEntries += "0:0"
}
if (-not $fibCache.ContainsKey(1)) {
    $fibCache[1] = 1
    $newlyComputedEntries += "1:1"
}

# Determine the highest index already computed and stored in the cache.
# If cache is empty or only has base cases, this will be 1.
$currentMaxIndex = 0
if ($fibCache.Count -gt 0) {
    # Get the maximum key from the cache, ensuring it's sorted correctly.
    # If keys are not numeric or sorted, this might fail.
    $currentMaxIndex = ($fibCache.Keys | Sort-Object -Descending)[0]
}

# If the requested 'n' is greater than the highest index in our cache, we need to compute.
if ($n -gt $currentMaxIndex) {
    # Get the last two known Fibonacci numbers to continue the sequence.
    # We need fib(k-1) and fib(k) to compute fib(k+1).
    # Ensure we have valid previous values to start the loop.
    $prev = $fibCache[$currentMaxIndex - 1]
    $curr = $fibCache[$currentMaxIndex]

    # Iterate from the next index after the current maximum up to 'n'.
    for ($i = $currentMaxIndex + 1; $i -le $n; $i++) {
        $nextFib = $prev + $curr
        $fibCache[$i] = $nextFib
$newlyComputedEntries += "{0}:{1}" -f $i, ($nextFib.ToString("F0")) # Store for database update

        # Update prev and curr for the next iteration
        $prev = $curr
        $curr = $nextFib
    }
}

# Output the requested Fibonacci number
Write-Output ($fibCache[$n]).ToString("F0")

# --- Update Database ---
# If we computed any new Fibonacci numbers, append them to the database file.
if ($newlyComputedEntries.Count -gt 0) {
    # Create the database file if it doesn't exist.
    if (-not (Test-Path $dbFile)) {
        try {
            New-Item -Path $dbFile -ItemType File | Out-Null
        } catch {
            Write-Error ("Failed to create database file '{0}'. Cannot save computed values." -f $dbFile)
            # We still proceed to output the result, but it won't be persisted.
        }
    }

    # Append new entries to the database file.
    try {
        $newlyComputedEntries | ForEach-Object { Add-Content -Path $dbFile -Value $_ }
    } catch {
        Write-Warning ("Failed to write new entries to '{0}'." -f $dbFile)
    }
}
