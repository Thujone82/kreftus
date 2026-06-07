package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

const replaceMarker = "^*"

// parsePeriod parses a period string with optional suffix (s, m, h) and returns
// the duration and a human-readable display string.
// Examples: "5" -> 5 minutes, "15s" -> 15 seconds, "1h" -> 1 hour
func parsePeriod(periodStr string) (time.Duration, string, error) {
	periodStr = strings.TrimSpace(periodStr)
	if periodStr == "" {
		return 5 * time.Minute, "5 minutes", nil
	}

	// Check for suffix
	if len(periodStr) > 0 {
		lastChar := strings.ToLower(periodStr[len(periodStr)-1:])
		if lastChar == "s" || lastChar == "m" || lastChar == "h" {
			// Has suffix, extract number
			numberStr := periodStr[:len(periodStr)-1]
			number, err := strconv.ParseFloat(numberStr, 64)
			if err != nil {
				return 5 * time.Minute, "5 minutes", err
			}

			switch lastChar {
			case "s":
				duration := time.Duration(number * float64(time.Second))
				display := fmt.Sprintf("%.0f second", number)
				if number != 1 {
					display += "s"
				}
				return duration, display, nil
			case "m":
				duration := time.Duration(number * float64(time.Minute))
				display := fmt.Sprintf("%.0f minute", number)
				if number != 1 {
					display += "s"
				}
				return duration, display, nil
			case "h":
				duration := time.Duration(number * float64(time.Hour))
				display := fmt.Sprintf("%.0f hour", number)
				if number != 1 {
					display += "s"
				}
				return duration, display, nil
			}
		}
	}

	// No suffix, treat as minutes
	number, err := strconv.ParseFloat(periodStr, 64)
	if err != nil {
		return 5 * time.Minute, "5 minutes", err
	}
	duration := time.Duration(number * float64(time.Minute))
	display := fmt.Sprintf("%.0f minute", number)
	if number != 1 {
		display += "s"
	}
	return duration, display, nil
}

func formatCompactDuration(d time.Duration, showFractionUnderMinute bool) string {
	totalSec := int(d.Seconds())
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60

	if h >= 1 {
		return fmt.Sprintf("%02d:%02d:%02ds", h, m, s)
	}
	if m >= 1 {
		return fmt.Sprintf("%02d:%02ds", m, s)
	}
	if showFractionUnderMinute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%ds", int(math.Round(d.Seconds())))
}

func formatDateAwareTimestamp(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04:05")
	}
	return t.Format("010206@15:04:05")
}

func formatSuccessRuntime(d time.Duration) string {
	totalSec := int(d.Seconds())
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	cs := d.Milliseconds() / 10
	if cs >= 100 {
		cs = 99
	}
	return fmt.Sprintf("%02d:%02d:%02d.%02d", h, m, s, cs)
}

type expectState struct {
	threshold              time.Duration
	display                string
	successCount           int
	actualCount            int
	totalSuccessfulRuntime time.Duration
	lastSuccessfulRuntime  time.Duration
	lastSuccessfulComplete time.Time
	hasLastSuccess         bool
}

func formatCompactPeriodLabel(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSec := int(math.Round(d.Seconds()))
	if totalSec >= 3600 {
		h := totalSec / 3600
		rem := totalSec % 3600
		if rem == 0 {
			return fmt.Sprintf("%dH", h)
		}
		m := rem / 60
		if m > 0 {
			return fmt.Sprintf("%dH%dM", h, m)
		}
		return fmt.Sprintf("%dH", h)
	}
	if totalSec >= 60 {
		return fmt.Sprintf("%dM", totalSec/60)
	}
	if totalSec <= 0 {
		return "0S"
	}
	return fmt.Sprintf("%dS", totalSec)
}

func formatExpectConfigDetails(
	expect *expectState,
	successLimitActive int,
	successTimeThreshold time.Duration,
	failLimitActive int,
	failTimeThreshold time.Duration,
	failedExecutionCount int,
	failedRetryTime time.Duration,
) string {
	var parts []string
	if expect != nil {
		parts = append(parts, fmt.Sprintf("Expect: %s", formatCompactPeriodLabel(expect.threshold)))
	}
	if successLimitActive > 0 {
		if expect != nil && expect.successCount > 0 {
			parts = append(parts, fmt.Sprintf("Success: %d/%d", expect.successCount, successLimitActive))
		} else {
			parts = append(parts, fmt.Sprintf("Success: %d", successLimitActive))
		}
	}
	if successTimeThreshold > 0 {
		totalLabel := formatCompactPeriodLabel(successTimeThreshold)
		if expect != nil && expect.totalSuccessfulRuntime > 0 {
			remaining := successTimeThreshold - expect.totalSuccessfulRuntime
			if remaining < 0 {
				remaining = 0
			}
			parts = append(parts, fmt.Sprintf("SuccessTime: %s / %s", formatCompactPeriodLabel(remaining), totalLabel))
		} else {
			parts = append(parts, fmt.Sprintf("SuccessTime: %s", totalLabel))
		}
	}
	if failLimitActive > 0 {
		if failedExecutionCount > 0 {
			parts = append(parts, fmt.Sprintf("Fail: %d/%d", failedExecutionCount, failLimitActive))
		} else {
			parts = append(parts, fmt.Sprintf("Fail: %d", failLimitActive))
		}
	}
	if failTimeThreshold > 0 {
		totalLabel := formatCompactPeriodLabel(failTimeThreshold)
		if failedRetryTime > 0 {
			remaining := failTimeThreshold - failedRetryTime
			if remaining < 0 {
				remaining = 0
			}
			parts = append(parts, fmt.Sprintf("FailTime: %s / %s", formatCompactPeriodLabel(remaining), totalLabel))
		} else {
			parts = append(parts, fmt.Sprintf("FailTime: %s", totalLabel))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func printExpectSummary(expect *expectState, executionCount, skip int, silent bool) {
	if expect == nil {
		return
	}
	if executionCount <= skip {
		return
	}
	if silent {
		return
	}

	lastSuccessDisplay := "N/A"
	if expect.hasLastSuccess {
		lastSuccessDisplay = formatDateAwareTimestamp(expect.lastSuccessfulComplete)
	}
	totalSuccessDisplay := formatSuccessRuntime(expect.totalSuccessfulRuntime)
	lastSuccessRuntimeDisplay := "N/A"
	if expect.hasLastSuccess {
		lastSuccessRuntimeDisplay = formatSuccessRuntime(expect.lastSuccessfulRuntime)
	}
	fmt.Printf("Last Success: %s (%d/%d)\n", lastSuccessDisplay, expect.successCount, expect.actualCount)
	fmt.Printf("Total Runtime: %s (%s)\n", totalSuccessDisplay, lastSuccessRuntimeDisplay)
}

func applyReplace(commandStr, replaceValue string, replaceSet, silent bool) string {
	if !replaceSet {
		return commandStr
	}
	if !strings.Contains(commandStr, replaceMarker) && !silent {
		color.Yellow("WARNING: -replace was specified but command does not contain the %s marker.", replaceMarker)
	}
	return strings.ReplaceAll(commandStr, replaceMarker, replaceValue)
}

// clearScreen clears the terminal screen using platform-specific commands or ANSI escape sequences.
func clearScreen() {
	if runtime.GOOS == "windows" {
		// On Windows, use cls command
		cmd := exec.Command("cmd", "/C", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		// On Unix-like systems, use ANSI escape sequence
		fmt.Print("\033[2J\033[H")
	}
}

// executeCommand runs the given command string in the appropriate shell for the OS.
// It pipes the command's stdout and stderr to the application's stdout and stderr.
func executeCommand(command string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		// For Linux, macOS, etc.
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		color.Yellow("Command failed: %v", err)
	}
}

func printUsage() {
	color.Yellow("SYNOPSIS")
	fmt.Println("    Runs a specified command repeatedly at a given interval.")
	fmt.Println()

	color.Yellow("DESCRIPTION")
	fmt.Println("    rc (Run Continuously) executes a given command string in a loop.")
	fmt.Println("    After each execution, it waits for a specified number of minutes")
	fmt.Println("    before running the command again.")
	fmt.Println()
	fmt.Println("    Use the -p or -precision flag to account for the command's")
	fmt.Println("    execution time to ensure each run starts at a consistent interval.")
	fmt.Println()
	fmt.Println("    If no parameters are provided, the script will interactively prompt")
	fmt.Println("    the user for the command and the time period.")
	fmt.Println()

	color.Yellow("USAGE")
	fmt.Println("    rc \"<command>\" [period] [-p] [-q] [-c] [-skip <number>] [-limit <number>]")
	fmt.Println("       [-e <period>] [-r <string>] [-f <number>] [-ft <period>] [-s <number>] [-st <period>]")
	fmt.Println()

	color.Yellow("PARAMETERS")
	color.Cyan("  <command>")
	fmt.Println("    The command to execute, enclosed in quotes if it contains spaces.")
	fmt.Println()
	color.Cyan("  [period]")
	fmt.Println("    Optional. The time to wait between executions. Accepts suffixes: 's' for seconds,")
	fmt.Println("    'm' for minutes (optional), 'h' for hours. Integers without suffix default to minutes.")
	fmt.Println("    Examples: 5, 15s, 5m, 1h. Defaults to 5.")
	fmt.Println()
	color.Cyan("  -p, -precision")
	fmt.Println("    Optional. Enables precision mode to prevent timing drift.")
	fmt.Println()
	color.Cyan("  -q, -quiet, -silent")
	fmt.Println("    Optional. Enables silent mode to suppress status output messages.")
	fmt.Println()
	color.Cyan("  -c, -clear")
	fmt.Println("    Optional. Clears the screen before executing the command in each iteration.")
	fmt.Println()
	color.Cyan("  -skip <number>")
	fmt.Println("    Optional. The number of initial executions to skip before starting to run the command.")
	fmt.Println("    If -skip 0 is specified, it defaults to 1 (skips the first execution).")
	fmt.Println("    If -skip is not specified at all, no executions are skipped (default is 0).")
	fmt.Println()
	color.Cyan("  -limit <number>")
	fmt.Println("    Optional. The maximum number of executions to perform. Skipped executions do not count.")
	fmt.Println("    If -limit is not specified or set to 0, there is no limit (default is 0).")
	fmt.Println()
	color.Cyan("  -e, -expect <period>")
	fmt.Println("    Optional. Minimum expected command runtime (period format). Runs below threshold are failures.")
	fmt.Println("    Prints success summary after each run in standard and precision modes.")
	fmt.Println()
	color.Cyan("  -r, -replace <string>")
	fmt.Println("    Optional. Replaces every literal ^* marker in the command with this value.")
	fmt.Println("    Emits a soft warning if -replace is set but the command has no ^* marker.")
	fmt.Println()
	color.Cyan("  -f, -fail <number>")
	fmt.Println("    Optional. Exit after this many failed runs (duration below -expect). Requires -expect. 0 = unlimited.")
	fmt.Println()
	color.Cyan("  -ft, -failtime <period>")
	fmt.Println("    Optional. Exit when failed runs times retry interval reaches this cap. Period format. Requires -expect.")
	fmt.Println()
	color.Cyan("  -s, -success <number>")
	fmt.Println("    Optional. Exit after this many successful runs (duration at or above -expect). Requires -expect. 0 = unlimited.")
	fmt.Println()
	color.Cyan("  -st, -successtime <period>")
	fmt.Println("    Optional. Exit when accumulated successful run time reaches this cap. Period format. Requires -expect.")
	fmt.Println()

	color.Yellow("EXAMPLES")
	color.Green("    rc \"go run main.go\" 1")
	fmt.Println("    Runs 'go run main.go' every 1 minute.")
	fmt.Println()
	color.Green("    rc \"gw Portland\" 10 -p")
	fmt.Println("    Runs the 'gw' command on a fixed 10-minute schedule.")
	fmt.Println()
	color.Green("    rc \"date\" 1 -q")
	fmt.Println("    Runs 'date' every minute in silent mode, suppressing status messages.")
	fmt.Println()
	color.Green("    rc \"my-script.sh\" 5 -p -q")
	fmt.Println("    Runs 'my-script.sh' every 5 minutes with precision timing and silent output.")
	fmt.Println()
	color.Green("    rc \"date\" 1 -c")
	fmt.Println("    Runs 'date' every minute with the screen cleared before each execution.")
	fmt.Println()
	color.Green("    rc \"Get-Process\" 5 -skip 2")
	fmt.Println("    Runs 'Get-Process' every 5 minutes, but skips the first 2 executions.")
	fmt.Println("    Execution will begin on the 3rd iteration.")
	fmt.Println()
	color.Green("    rc \"date\" 1 -skip 0")
	fmt.Println("    Runs 'date' every minute, but skips the first execution.")
	fmt.Println("    Since -skip 0 was specified, it defaults to 1.")
	fmt.Println()
	color.Green("    rc \"Get-Process\" 15s -limit 5")
	fmt.Println("    Runs 'Get-Process' every 15 seconds, but only executes 5 times total, then exits.")
	fmt.Println()
	color.Green("    rc \"date\" 1h -skip 1 -limit 3")
	fmt.Println("    Runs 'date' every hour, skips the first execution, then executes 3 times before exiting.")
	fmt.Println()
	color.Green(`    rc "gf -x ^*" 5 -r pdx`)
	fmt.Println("    Runs 'gf -x pdx' every 5 minutes by substituting pdx for the ^* marker.")
	fmt.Println()
	color.Green("    rc \"date\" 5s -e 1s")
	fmt.Println("    Runs every 5 seconds and tracks successful runs where duration is at least 1 second.")
	fmt.Println()
	color.Green("    rc \"date\" 5m -e 30s -fail 3")
	fmt.Println("    Exits after 3 runs that finish faster than the 30 second expected minimum.")
	fmt.Println()
	color.Green("    rc \"date\" 5m -e 30s -success 5")
	fmt.Println("    Exits after 5 runs that meet the 30 second expected minimum.")
	fmt.Println()
}

func warnDuplicateFlag(seen map[string]bool, label string) bool {
	if seen[label] {
		color.Yellow("WARNING: Flag -%s specified more than once; using the first value.", label)
		return true
	}
	seen[label] = true
	return false
}

func main() {
	// Manual argument parsing is used to allow flags to be placed anywhere in the command.
	// The standard `flag` package stops parsing at the first non-flag argument.
	var commandStr string
	periodStr := "5" // Default period as string
	var precision bool
	var silent bool
	var clear bool
	skip := 0 // Default skip count
	limit := 0 // Default limit (0 = no limit)
	var expectStr string
	var expectSet bool
	var replaceValue string
	var replaceSet bool
	var failLimit int
	var failSet bool
	var failTimeStr string
	var failTimeSet bool
	var successLimit int
	var successSet bool
	var successTimeStr string
	var successTimeSet bool
	var nonFlagArgs []string
	skipFlagFound := false

	seenFlags := make(map[string]bool)

	args := os.Args[1:]
	skipValue := func(i int) int {
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return 2
		}
		return 1
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-p", "-precision":
			if warnDuplicateFlag(seenFlags, "precision") {
				continue
			}
			precision = true
		case "-q", "-quiet", "-silent", "-Silent":
			if warnDuplicateFlag(seenFlags, "silent") {
				continue
			}
			silent = true
		case "-c", "-clear":
			if warnDuplicateFlag(seenFlags, "clear") {
				continue
			}
			clear = true
		case "-skip", "-Skip":
			if warnDuplicateFlag(seenFlags, "skip") {
				i += skipValue(i)
				continue
			}
			skipFlagFound = true
			if i+1 < len(args) {
				if s, err := strconv.Atoi(args[i+1]); err == nil {
					skip = s
					i++
				}
			}
		case "-limit", "-Limit":
			if warnDuplicateFlag(seenFlags, "limit") {
				i += skipValue(i)
				continue
			}
			if i+1 < len(args) {
				if l, err := strconv.Atoi(args[i+1]); err == nil && l >= 0 {
					limit = l
					i++
				}
			}
		case "-e", "-expect", "-Expect":
			if warnDuplicateFlag(seenFlags, "expect") {
				i += skipValue(i)
				continue
			}
			expectSet = true
			if i+1 < len(args) {
				expectStr = args[i+1]
				i++
			}
		case "-r", "-replace", "-Replace":
			if warnDuplicateFlag(seenFlags, "replace") {
				i += skipValue(i)
				continue
			}
			replaceSet = true
			if i+1 < len(args) {
				replaceValue = args[i+1]
				i++
			}
		case "-f", "-fail", "-Fail":
			if warnDuplicateFlag(seenFlags, "fail") {
				i += skipValue(i)
				continue
			}
			failSet = true
			if i+1 < len(args) {
				if f, err := strconv.Atoi(args[i+1]); err == nil {
					failLimit = f
					i++
				}
			}
		case "-ft", "-failtime", "-FailTime":
			if warnDuplicateFlag(seenFlags, "failtime") {
				i += skipValue(i)
				continue
			}
			failTimeSet = true
			if i+1 < len(args) {
				failTimeStr = args[i+1]
				i++
			}
		case "-s", "-success", "-Success":
			if warnDuplicateFlag(seenFlags, "success") {
				i += skipValue(i)
				continue
			}
			successSet = true
			if i+1 < len(args) {
				if s, err := strconv.Atoi(args[i+1]); err == nil {
					successLimit = s
					i++
				}
			}
		case "-st", "-successtime", "-SuccessTime":
			if warnDuplicateFlag(seenFlags, "successtime") {
				i += skipValue(i)
				continue
			}
			successTimeSet = true
			if i+1 < len(args) {
				successTimeStr = args[i+1]
				i++
			}
		case "-h", "-help":
			if warnDuplicateFlag(seenFlags, "help") {
				continue
			}
			printUsage()
			os.Exit(0)
		default:
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	// If -skip was used but value is 0, default to 1
	if skipFlagFound && skip == 0 {
		skip = 1
	}

	// Process the remaining non-flag arguments for the command and period.
	if len(nonFlagArgs) > 0 {
		commandStr = nonFlagArgs[0]
	}
	if len(nonFlagArgs) > 1 {
		// Try to parse period from remaining arguments (could be number or number with suffix)
		for _, arg := range nonFlagArgs[1:] {
			// Try parsing as period string (supports suffixes)
			_, _, err := parsePeriod(arg)
			if err == nil {
				periodStr = arg
				break // Use the first valid period found
			}
		}
	}

	// --- Interactive Mode ---
	// If no command is provided via arguments, prompt the user for input.
	if commandStr == "" {
		color.Yellow("*** Run Continuously v1 ***")

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Command: ")
		cmdInput, _ := reader.ReadString('\n')
		commandStr = strings.TrimSpace(cmdInput)

		fmt.Print("Period (e.g., 5, 15s, 5m, 1h) [default: 5]: ")
		periodInput, _ := reader.ReadString('\n')
		periodInput = strings.TrimSpace(periodInput)
		if periodInput != "" {
			_, _, err := parsePeriod(periodInput)
			if err == nil {
				periodStr = periodInput
			}
		}

		fmt.Print("Enable Precision Mode? (y/n) [default: n]: ")
		precisionInput, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(precisionInput)) == "y" {
			precision = true
		}

		fmt.Print("Enable Clear Mode? (y/n) [default: n]: ")
		clearInput, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(clearInput)) == "y" {
			clear = true
		}

		fmt.Print("Skip initial executions? (enter number, or 0 for default skip 1) [default: 0]: ")
		skipInput, _ := reader.ReadString('\n')
		skipInput = strings.TrimSpace(skipInput)
		if skipInput != "" {
			if s, err := strconv.Atoi(skipInput); err == nil && s >= 0 {
				skip = s
				if skip == 0 {
					skip = 1 // Default to 1 if 0 is explicitly entered in interactive mode
				}
			}
		}
		// If empty, skip remains 0 (no skipping)

		fmt.Print("Limit executions? (enter number, or 0 for no limit) [default: 0]: ")
		limitInput, _ := reader.ReadString('\n')
		limitInput = strings.TrimSpace(limitInput)
		if limitInput != "" {
			if l, err := strconv.Atoi(limitInput); err == nil && l >= 0 {
				limit = l
			}
		}
	}

	// Exit if no command was provided either by argument or interactively.
	if commandStr == "" {
		fmt.Println("No command provided. Exiting.")
		return
	}

	// Parse period string to get duration and display string
	periodDuration, periodDisplay, err := parsePeriod(periodStr)
	if err != nil {
		periodDuration = 5 * time.Minute
		periodDisplay = "5 minutes"
	}

	var expect *expectState
	if expectSet {
		expectDuration, expectDisplay, parseErr := parsePeriod(expectStr)
		if parseErr != nil {
			expectDuration = time.Minute
			expectDisplay = "1 minute"
		}
		expect = &expectState{
			threshold: expectDuration,
			display:   expectDisplay,
		}
	}

	commandStr = applyReplace(commandStr, replaceValue, replaceSet, silent)

	failLimitActive := 0
	var failTimeThreshold time.Duration
	var failTimeDisplay string
	failLimitRequested := failSet && failLimit > 0
	failTimeRequested := failTimeSet
	successLimitActive := 0
	var successTimeThreshold time.Duration
	var successTimeDisplay string
	successLimitRequested := successSet && successLimit > 0
	successTimeRequested := successTimeSet

	if failLimitRequested || failTimeRequested || successLimitRequested || successTimeRequested {
		if expect == nil {
			if !silent {
				color.Yellow("WARNING: -fail, -failtime, -success, and -successtime require -expect (-e) and were ignored.")
			}
		} else {
			if failLimitRequested {
				failLimitActive = failLimit
			}
			if failTimeRequested {
				ftDuration, ftDisplay, parseErr := parsePeriod(failTimeStr)
				if parseErr == nil && ftDuration > 0 {
					failTimeThreshold = ftDuration
					failTimeDisplay = ftDisplay
				}
			}
			if successLimitRequested {
				successLimitActive = successLimit
			}
			if successTimeRequested {
				stDuration, stDisplay, parseErr := parsePeriod(successTimeStr)
				if parseErr == nil && stDuration > 0 {
					successTimeThreshold = stDuration
					successTimeDisplay = stDisplay
				}
			}
		}
	}

	failedExecutionCount := 0
	var failedRetryTime time.Duration
	expectConfigDetails := formatExpectConfigDetails(expect, successLimitActive, successTimeThreshold, failLimitActive, failTimeThreshold, 0, 0)

	// --- Initial Output ---
	if clear {
		clearScreen()
	}
	if !silent {
		fmt.Printf("Running \"%s\" every %s. Press Ctrl+C to stop.\n\n", commandStr, periodDisplay)
		if expectConfigDetails != "" {
			color.Magenta(expectConfigDetails)
		}
		if skip > 0 {
			color.Yellow("Skipping the first %d execution(s).", skip)
		}
		if limit > 0 {
			color.Cyan("Limited to %d execution(s).", limit)
		}
	}
	var scriptStartTime time.Time
	if precision {
		scriptStartTime = time.Now()
		if !silent {
			color.Cyan("Precision mode is enabled. Aligning to grid starting at %s.", scriptStartTime.Format("15:04:05"))
		}
	}

	// --- Main Execution Loop ---
	executionCount := 0
	actualExecutionCount := 0
	var pendingExitMsg string
	var pendingExitGreen bool
	for {
		executionCount++
		loopStartTime := time.Now()
		var commandDuration time.Duration
		var hasCommandDuration bool

		if executionCount <= skip {
			if !silent {
				color.Yellow("(%s) Skipping execution %d of %d...", loopStartTime.Format("15:04:05"), executionCount, skip)
			}
		} else {
			actualExecutionCount++
			if clear {
				clearScreen()
			}
			if !silent {
				executeMessage := fmt.Sprintf("(%s) Executing command...", loopStartTime.Format("15:04:05"))
				executeConfigDetails := formatExpectConfigDetails(expect, successLimitActive, successTimeThreshold, failLimitActive, failTimeThreshold, failedExecutionCount, failedRetryTime)
				if executeConfigDetails != "" {
					executeMessage += fmt.Sprintf(" [%s]", executeConfigDetails)
				}
				color.White(executeMessage)
			}
			executeCommand(commandStr)
			commandEndTime := time.Now()
			commandDuration = commandEndTime.Sub(loopStartTime)
			hasCommandDuration = true

			if expect != nil && commandDuration >= expect.threshold {
				expect.successCount++
				expect.totalSuccessfulRuntime += commandDuration
				expect.lastSuccessfulRuntime = commandDuration
				expect.lastSuccessfulComplete = commandEndTime
				expect.hasLastSuccess = true
			} else if expect != nil {
				failedExecutionCount++
				failedRetryTime += periodDuration
			}
			if expect != nil {
				expect.actualCount = actualExecutionCount
			}

			if limit > 0 && actualExecutionCount >= limit {
				pendingExitMsg = fmt.Sprintf("Reached execution limit of %d. Exiting.", limit)
				pendingExitGreen = true
			} else if failLimitActive > 0 && failedExecutionCount >= failLimitActive {
				pendingExitMsg = fmt.Sprintf("Reached failure limit of %d. Exiting.", failLimitActive)
				pendingExitGreen = false
			} else if failTimeThreshold > 0 && failedRetryTime >= failTimeThreshold {
				pendingExitMsg = fmt.Sprintf("Reached failure time limit of %s. Exiting.", failTimeDisplay)
				pendingExitGreen = false
			} else if successLimitActive > 0 && expect != nil && expect.successCount >= successLimitActive {
				pendingExitMsg = fmt.Sprintf("Reached success limit of %d. Exiting.", successLimitActive)
				pendingExitGreen = true
			} else if successTimeThreshold > 0 && expect != nil && expect.totalSuccessfulRuntime >= successTimeThreshold {
				pendingExitMsg = fmt.Sprintf("Reached success time limit of %s. Exiting.", successTimeDisplay)
				pendingExitGreen = true
			}
		}

		if pendingExitMsg != "" {
			break
		}

		if precision {
			currentTime := time.Now()
			if !hasCommandDuration {
				commandDuration = currentTime.Sub(loopStartTime)
			}

			totalElapsed := currentTime.Sub(scriptStartTime)
			periodMinutes := periodDuration.Minutes()
			intervalsCompleted := math.Floor(totalElapsed.Minutes() / periodMinutes)
			nextTargetTime := scriptStartTime.Add(time.Duration(intervalsCompleted+1) * periodDuration)
			sleepDuration := nextTargetTime.Sub(currentTime)

			if sleepDuration.Seconds() > 0 {
				if !silent {
					runtimeDisplay := formatCompactDuration(commandDuration, true)
					waitingDisplay := formatCompactDuration(sleepDuration, false)
					nextRunDisplay := formatDateAwareTimestamp(nextTargetTime)
					color.White("Runtime: %s Waiting: %s Next Run: %s", runtimeDisplay, waitingDisplay, nextRunDisplay)
					printExpectSummary(expect, executionCount, skip, silent)
					color.White("Press Ctrl+C to stop.")
				}
				time.Sleep(sleepDuration)
			} else if !silent {
				color.Yellow("WARNING: Command execution time (%.2fs) overran its schedule. Running next iteration immediately.\n", commandDuration.Seconds())
			}
		} else {
			if !silent {
				if expect != nil && executionCount > skip {
					fmt.Printf("Waiting %s.\n", periodDisplay)
					printExpectSummary(expect, executionCount, skip, silent)
					color.White("Press Ctrl+C to stop.\n")
				} else {
					color.White("Waiting %s. Press Ctrl+C to stop.\n", periodDisplay)
				}
			}
			time.Sleep(periodDuration)
		}
	}

	if pendingExitMsg != "" && !silent {
		printExpectSummary(expect, executionCount, skip, silent)
		if pendingExitGreen {
			color.Green("\n%s", pendingExitMsg)
		} else {
			color.Red("\n%s", pendingExitMsg)
		}
	}
}
