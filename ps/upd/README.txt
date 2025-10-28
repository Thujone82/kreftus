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
    - Job name validation (no spaces or special characters)
    - Smart transfer size formatting (KB/MB/GB based on file size)

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
        Name:       Display name for the job (validated for safety)
        Remote:     Source location (file path or URL)
        Local:      Destination directory path
        
    Job Name Validation:
        - Only letters, numbers, hyphens (-), and underscores (_) allowed
        - No spaces or special characters
        - Case-insensitive matching for command-line selection
        - Examples: "my-job", "job_1", "BackupScript", "update-v2"
        
    Example Job:
        Name:       BMON
        Remote:     https://kreft.us/ps/bmon/bmon.exe
        Local:      C:\Tools\
        
    Result: Downloads bmon.exe to C:\Tools\bmon.exe

DYNAMIC DATE PLACEHOLDERS:
    Both Remote and Local paths support dynamic date/time placeholders using [dateformat] notation.
    Placeholders are replaced with current date/time when jobs execute.
    
    Format Specifiers:
        yyyy    - 4-digit year (2025)
        yy      - 2-digit year (25)
        MM      - 2-digit month (01-12)
        M       - 1-2 digit month (1-12)
        dd      - 2-digit day (01-31)
        d       - 1-2 digit day (1-31)
        HH      - 2-digit hour 24h (00-23)
        hh      - 2-digit hour 12h (01-12)
        mm      - 2-digit minute (00-59)
        ss      - 2-digit second (00-59)
        tt      - AM/PM designator
    
    Examples:
        Weather Radar:     http://radar.com?time=[MMddHH]-[mm].png
        Log Files:         C:\logs\app-[yyyy-MM-dd].log
        Timestamped API:   https://api.com/data?date=[yyyyMMdd]&time=[HHmmss]
        Backup Files:      backup-[yyyy-MM-dd_HH-mm-ss].zip
        Monthly Folders:   Local: C:\downloads\[yyyy-MM]\ (creates monthly folders)
        Daily Archives:    Local: C:\backups\[yyyy-MM-dd]\ (creates daily folders)
    
    Multiple placeholders in a single path are supported.
    All placeholders use the same timestamp (when job execution starts).

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

TRANSFER SIZE FORMATTING:
    The tool automatically formats transfer sizes with appropriate units:
    - Files < 1 MB: Displayed in KB (e.g., "5.39 KB")
    - Files 1 MB to < 1 GB: Displayed in MB (e.g., "1.5 MB")
    - Files >= 1 GB: Displayed in GB (e.g., "2.5 GB")
    
    Transfer speed is also intelligently formatted:
    - Speeds < 1 MB/s: Displayed in KB/s (e.g., "5.39 KB/s")
    - Speeds >= 1 MB/s: Displayed in MB/s (e.g., "1.2 MB/s")

NOTES:
    - Local directories are created automatically if they don't exist
    - Downloaded files retain their original filename
    - Copied files retain their original filename
    - Job selection persists within a session
    - Multiple jobs can be selected and executed together

VERSION HISTORY:
    v1.2 - Improved transfer size display
        - Smart transfer size formatting (KB/MB/GB based on file size)
        - Better visibility for small file transfers
        - Consistent unit formatting for both size and speed
        
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

