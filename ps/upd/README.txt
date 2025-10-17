================================================================================
                              UPDate Tool v1.0
                    PowerShell File Update Utility
================================================================================

DESCRIPTION:
    UPDate is an interactive command-line tool for managing and executing 
    file update jobs. It allows you to configure update sources (local paths 
    or URLs) and destination directories, then execute updates with a simple 
    interactive interface.

FEATURES:
    - Interactive CLI with color-coded output
    - Multiple job selection and execution
    - Support for local file copying and URL downloads
    - JSON-based configuration storage
    - Command-line arguments for automation
    - Verbose mode for debugging

USAGE:
    Interactive Mode:
        .\upd.ps1
        
    Pre-select Job:
        .\upd.ps1 -jobname
        
    Auto-execute Job:
        .\upd.ps1 -jobname -a
        .\upd.ps1 -jobname -auto
        
    Note: Auto mode (-a) now completes without requiring key press and leaves output visible
        
    Verbose Mode:
        .\upd.ps1 -Verbose
        .\upd.ps1 -jobname -a -Verbose

INTERACTIVE COMMANDS:

    Main Update Screen:
        [1-N]       Toggle job selection (numbers correspond to listed jobs)
        A           Select all jobs
        J           Open Jobs management screen
        E/Enter     Execute selected jobs (with confirmation)
        Esc         Exit application
        
    Jobs Management Screen:
        [1-N]       Edit existing job
        N           Create new job
        D/Esc/Enter Return to main update screen
        
    Job Editing:
        Remove Job? [y/N]  Remove job from configuration (yellow prompt)
        Any other input   Continue with normal editing

JOB CONFIGURATION:
    Each job consists of:
        Name:       Display name for the job
        Remote:     Source location (file path or URL)
        Local:      Destination directory path
        
    Example Job:
        Name:       BMON
        Remote:     https://kreft.us/ps/bmon/bmon.exe
        Local:      C:\Tools\
        
    Result: Downloads bmon.exe to C:\Tools\bmon.exe

COMMAND LINE ARGUMENTS:
    -jobname        Pre-select a job by name (case-insensitive)
    -a, -auto       Auto-execute pre-selected jobs without confirmation
    -Verbose        Enable verbose debug output

EXAMPLES:
    1. Interactive mode with job management:
        .\upd.ps1
        
    2. Pre-select "bmon" job for manual execution:
        .\upd.ps1 -bmon
        
    3. Automatically execute "bmon" and "larry" jobs:
        .\upd.ps1 -bmon -larry -a
        
    4. Debug mode with verbose output:
        .\upd.ps1 -bmon -a -Verbose

CONFIGURATION FILE:
    Location:   Same directory as upd.ps1
    Filename:   upd.json
    Format:     JSON
    
    Example:
    {
        "Jobs": [
            {
                "Name": "BMON",
                "Remote": "https://example.com/bmon.exe",
                "Local": "C:\\Tools\\"
            },
            {
                "Name": "Larry",
                "Remote": "C:\\Source\\larry.exe",
                "Local": "C:\\Tools\\"
            }
        ],
        "Version": "1.0"
    }

COLOR CODING:
    Cyan:       Headers and hotkey indicators
    White:      Normal text and unselected jobs
    Green:      Selected jobs and success messages
    Red:        Error messages
    Yellow:     Prompts and warnings

REQUIREMENTS:
    - PowerShell 5.1 or later
    - Windows operating system
    - Write permissions for destination directories

TROUBLESHOOTING:
    Q: Jobs aren't saving
    A: Ensure you have write permissions in the script directory
    
    Q: URL downloads failing
    A: Check your internet connection and firewall settings
    
    Q: Can't select multiple jobs
    A: Use number keys (1-9) to toggle each job individually
    
    Q: Command line arguments not working
    A: Ensure job names match exactly (case-insensitive)
       Use -Verbose flag to see what's being processed

NOTES:
    - Local directories are created automatically if they don't exist
    - Downloaded files retain their original filename
    - Copied files retain their original filename
    - Job selection persists within a session
    - Multiple jobs can be selected and executed together

VERSION HISTORY:
    v1.1 - Enhanced navigation and job management
        - Added Esc key navigation (exit from main, return from jobs)
        - Added job removal functionality in edit mode
        - Improved auto mode (-a) behavior (no key press required, output remains visible)
        - Enhanced output display with Source/Destination information
        
    v1.0 - Initial release
        - Interactive job management
        - Multiple job selection
        - URL and local file support
        - Command-line automation
        - JSON configuration storage

AUTHOR:
    Created for the Kreftus project

LICENSE:
    See LICENSE file in project root

================================================================================

