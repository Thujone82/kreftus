NAME
    btc.ps1 - Bitcoin Price Checker & Portfolio Tracker (BTCv2.2)

SYNOPSIS
    .\btc.ps1 [-UserBTCAmount <double>] [-UserTotalCost <double>] 
              [-LogToFile <string>] [-Update] [-Verbose]

DESCRIPTION
    The btc.ps1 script retrieves and displays real-time and historical Bitcoin (BTC) 
    data from the LiveCoinWatch API. It supports portfolio tracking (amount of BTC 
    owned and total cost), profit/loss calculation, and detailed CSV logging of 
    fetched data.

    The script features an interactive first-run setup to configure essential 
    settings, such as the API key, if a configuration file (config.ini) is 
    missing or incomplete. It also provides an interactive update mechanism via 
    the -Update switch to modify stored portfolio and logging preferences.

    Key display features include:
    - Current Bitcoin price, color-coded based on 24-hour performance or 
      proximity to its All-Time High (ATH).
    - Calculated 24-hour price difference based on actual historical data.
    - Optional display of the user's BTC holdings value and its 24-hour change.
    - Optional profit/loss calculation in USD and percentage.
    - 1H SMA (Simple Moving Average) - average price over the last hour, 
      color-coded (Green if current > SMA, Red if current < SMA).
    - 24H Ago price with percentage change, color-coded based on comparison.
    - 24H High and Low prices with timestamps.
    - Volatility percentage showing price swing over 24 hours, with velocity 
      metric in brackets (e.g. "3.99% [15]"), color-coded based on recent vs. 
      older volatility trends.
    - Current 24-hour trading volume and market capitalization for Bitcoin.
    - History sparkline visualization showing 24-hour price trend.
    - Verbose mode for detailed operational messages.

    The script clears the console upon execution for a clean output.

PARAMETERS
    -UserBTCAmount <double>
        Optional. Specifies the amount of Bitcoin owned by the user (e.g., 0.5). 
        This value overrides the 'MyBTC' setting in config.ini for the current 
        script run.

    -UserTotalCost <double>
        Optional. Specifies the total USD cost for the amount of Bitcoin defined 
        by -UserBTCAmount or 'MyBTC' in config.ini. This value overrides the 
        'MyCOST' setting in config.ini for the current script run. It is 
        required for the Profit/Loss calculation.

    -LogToFile <string>
        Optional. Specifies the full or relative path to the CSV log file. 
        This value overrides the 'LogPath' setting in config.ini for the 
        current script run.

    -Update [<SwitchParameter>]
        Optional. If this switch is present, the script will enter an 
        interactive mode to update the 'MyBTC', 'MyCOST', and 'LogPath' 
        settings stored in the config.ini file. After updating, the script 
        will proceed with its normal operation using the new settings.

    -Verbose [<SwitchParameter>]
        Optional. A common PowerShell parameter. If present, the script will 
        display detailed operational messages, such as configuration loading steps, 
        API call statuses, and logging activities.

CONFIGURATION FILE (config.ini)
    The script uses a configuration file named 'config.ini', expected to be in 
    the same directory as btc.ps1. If the file or essential settings (like 
    ApiKey) are missing, the script will initiate a first-run setup.

    The config.ini structure is as follows:

    [Settings]
    ApiKey=YOUR_API_KEY_HERE
    LogPath=btc_log.csv

    [Portfolio]
    MyBTC=0.0
    MyCOST=0.0

    Sections and Keys:
    - [Settings]
        - ApiKey: (Required) Your personal API key for LiveCoinWatch.com.
        - LogPath: (Optional) The path for the CSV log file. Can be an absolute 
          path or relative to the script's directory. If left empty, default 
          logging is disabled (unless overridden by -LogToFile). If the key is 
          entirely absent, it defaults to 'btc_log.csv' in the script's directory.

    - [Portfolio]
        - MyBTC: (Optional) The default amount of Bitcoin you own. Can be 
          overridden by -UserBTCAmount.
        - MyCOST: (Optional) The default total USD cost for your MyBTC amount. 
          Can be overridden by -UserTotalCost.

FIRST-RUN SETUP
    If the config.ini file is not found, or if the ApiKey is missing or empty 
    within it, the script will guide the user through a first-time setup process:
    1. Prompts for the LiveCoinWatch API Key (mandatory).
    2. Prompts for the user's BTC amount (optional).
    3. If a BTC amount is entered, prompts for the total cost of that BTC (optional).
    4. Prompts for a Log File Path (optional; leaving blank disables default logging).
    The script then creates/updates config.ini with these details.

UPDATE FUNCTIONALITY (-Update)
    When run with the -Update switch, the script allows interactive modification 
    of the following settings in config.ini:
    - MyBTC Amount
    - Total Cost (MyCOST)
    - Log File Path
    The script displays the current values and prompts the user to enter new ones 
    or press Enter to keep the existing settings.

OUTPUT
    The script clears the console and then displays the following information:
    - Bitcoin (USD): Current price, color-coded (Magenta if near ATH, otherwise 
      Green if up in 24h, Red if down), and the 24-hour absolute price 
      difference (e.g., [+$100.50]).
    - My BTC: (If MyBTC amount is configured/provided) Current total value of 
      your BTC holdings, color-coded based on Bitcoin's 24h performance, and 
      the 24-hour absolute change in your holdings' value.
    - Profit/Loss: (If MyBTC and MyCOST are configured/provided) Your total 
      unrealized profit or loss in USD, color-coded (Green for profit, Red for 
      loss), and the profit/loss percentage.
    - 1H SMA: Simple Moving Average over the last hour, color-coded (Green if 
      current price is above average, Red if below, White if equal).
    - 24H Ago: Price from 24 hours ago with percentage change in brackets, 
      color-coded based on comparison with current price.
    - 24H High: Highest price in the last 24 hours with timestamp (HH:mm format).
    - 24H Low: Lowest price in the last 24 hours with timestamp (HH:mm format).
    - Volatility: Percentage showing the price swing (High vs Low) over the last 
      24 hours, plus velocity in brackets (e.g. "3.99% [15]"). The percentage 
      is color-coded (Green if recent volatility > older volatility, Red if less, 
      White if equal). The velocity bracket is colored separately: Green when 
      last hour average delta > 24h average hourly delta, Red otherwise; 
      White when multiplier data is missing.
    - 24H Volume: Bitcoin's 24-hour trading volume in USD, color-coded based 
      on Bitcoin's 24h price performance. May also show % change if API provides it.
    - Cap: Bitcoin's current market capitalization in USD, color-coded based 
      on Bitcoin's 24h price performance. May also show % change if API provides it.
    - History: Visual sparkline showing 24-hour price trend using Unicode 
      block characters (▁▂▃▄▅▆▇█).

    Verbose Output (-Verbose):
    Includes messages about configuration loading, API call progress, historical 
    price fetching details (including TotalChange for velocity), velocity 
    calculation details (TotalChange, 1HourDeltaTotal, 24H High/Low, range, Volatility, hourlyAvg, multiplier, result), 
    logging status, and portfolio value tracking.

    Velocity (in Volatility bracket):
    The bracketed integer after Volatility (e.g. "3.99% [15]") is the velocity. 
    Formula: Velocity = (TotalChange / (24H High - 24H Low)) * Volatility * 
    (1HourDeltaTotal / (24DeltaTotal/24)). TotalChange (24DeltaTotal) is the 
    sum of absolute price deltas between consecutive 24h historical data points. 
    1HourDeltaTotal is the same sum for the last hour only. The multiplier 
    compares last-hour activity to the 24h average: above-average hourly 
    activity increases velocity, below-average decreases it. Volatility is used 
    as a whole number (e.g. 3.99% means multiply by 3.99). The result is rounded 
    to the nearest integer. If the 24h range is zero or data is missing, the 
    bracket is omitted. Velocity color: Green when 1HourDeltaTotal > (24DeltaTotal/24) 
    (last hour above average), Red otherwise; White when multiplier data is missing. 
    Use -Verbose to see the calculation details.

LOGGING
    If a valid log path is determined (via -LogToFile parameter or config.ini), 
    the script appends a new entry to a CSV file with each run. 
    The log includes:
    - Timestamp (YYYY-MM-DD HH:MM:SS)
    - MyBTC (if configured)
    - MyCOST_USD (if configured)
    - ProfitLoss_USD (if calculated)
    - ProfitLoss_Percent (if calculated)
    - Rate_USD (Current Bitcoin price)
    - Volume24h_USD
    - Cap_USD
    - Liquidity_USD
    - DeltaHour_Pct, DeltaDay_Pct, DeltaWeek_Pct, DeltaMonth_Pct, DeltaYear_Pct
    - TotalSupply
    - AllTimeHigh_USD
    If the log file does not exist, it is created with a header row.

EXAMPLES
    .\btc.ps1
        Runs the script with settings from config.ini.

    .\btc.ps1 -UserBTCAmount 0.5 -UserTotalCost 25000
        Runs the script, overriding portfolio settings from config.ini for this run.

    .\btc.ps1 -Update
        Prompts the user to update portfolio and log path settings in config.ini.

    .\btc.ps1 -LogToFile "D:\CryptoLogs\bitcoin_tracker.csv"
        Runs the script and logs data to the specified custom path.

    .\btc.ps1 -Verbose
        Runs the script and displays detailed operational messages.

NOTES
    - The script requires an active internet connection.
    - An API key from LiveCoinWatch.com is mandatory and should be stored in 
      config.ini or entered during first-run setup.
    - The 24-hour price difference and additional metrics (1H SMA, High/Low, 
      Volatility) are calculated by fetching full 24-hour historical data via 
      the API. If this call fails, the script falls back to an estimated 
      difference based on the API's 24-hour percentage delta.
    - The History sparkline is generated from hourly samples of the 24-hour 
      historical data, providing a visual representation of price trends.
    - Color coding for the main Bitcoin price and "My BTC" value reflects the 
      sign of the calculated 24-hour dollar difference, with ATH proximity 
      (purple) taking precedence for the Bitcoin price line.
    - The script uses [CmdletBinding()] for robust handling of common parameters 
      like -Verbose.

AUTHOR
    Kreft&Gemini[Gemini 2.5 Pro (preview)]

VERSION
    2.2



