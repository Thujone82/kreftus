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
	fmt.Println("    rc \"<command>\" [period] [-p] [-s] [-c] [-skip <number>]")
	fmt.Println()

	color.Yellow("PARAMETERS")
	color.Cyan("  <command>")
	fmt.Println("    The command to execute, enclosed in quotes if it contains spaces.")
	fmt.Println()
	color.Cyan("  [period]")
	fmt.Println("    Optional. The time in minutes to wait between executions. Defaults to 5.")
	fmt.Println()
	color.Cyan("  -p, -precision")
	fmt.Println("    Optional. Enables precision mode to prevent timing drift.")
	fmt.Println()
	color.Cyan("  -s, -silent")
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

	color.Yellow("EXAMPLES")
	color.Green("    rc \"go run main.go\" 1")
	fmt.Println("    Runs 'go run main.go' every 1 minute.")
	fmt.Println()
	color.Green("    rc \"gw Portland\" 10 -p")
	fmt.Println("    Runs the 'gw' command on a fixed 10-minute schedule.")
	fmt.Println()
	color.Green("    rc \"date\" 1 -s")
	fmt.Println("    Runs 'date' every minute in silent mode, suppressing status messages.")
	fmt.Println()
	color.Green("    rc \"my-script.sh\" 5 -p -s")
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
}

func main() {
	// Manual argument parsing is used to allow flags to be placed anywhere in the command.
	// The standard `flag` package stops parsing at the first non-flag argument.
	var commandStr string
	period := 5 // Default period in minutes
	var precision bool
	var silent bool
	var clear bool
	skip := 0 // Default skip count
	var nonFlagArgs []string
	skipFlagFound := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-p", "-precision":
			precision = true
		case "-s", "-silent":
			silent = true
		case "-c", "-clear":
			clear = true
		case "-skip", "-Skip":
			skipFlagFound = true
			// Check if there's a next argument and it's a number
			if i+1 < len(args) {
				if s, err := strconv.Atoi(args[i+1]); err == nil {
					skip = s
					i++ // Skip the next argument since we consumed it
				}
			}
		case "-h", "-help":
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
		for _, arg := range nonFlagArgs[1:] {
			if p, err := strconv.Atoi(arg); err == nil && p > 0 {
				period = p
				break // Use the first valid integer found
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

		fmt.Print("Period (minutes) [default: 5]: ")
		periodInput, _ := reader.ReadString('\n')
		periodInput = strings.TrimSpace(periodInput)
		if p, err := strconv.Atoi(periodInput); err == nil && p > 0 {
			period = p
		} else {
			period = 5 // Default if empty or invalid
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
	}

	// Exit if no command was provided either by argument or interactively.
	if commandStr == "" {
		fmt.Println("No command provided. Exiting.")
		return
	}

	// --- Initial Output ---
	if clear {
		clearScreen()
	}
	if !silent {
		fmt.Printf("Running \"%s\" every %d minute(s). Press Ctrl+C to stop.\n\n", commandStr, period)
		if skip > 0 {
			color.Yellow("Skipping the first %d execution(s).", skip)
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
	periodDuration := time.Duration(period) * time.Minute
	executionCount := 0
	for {
		executionCount++
		loopStartTime := time.Now()

		// Skip execution if we haven't reached the skip threshold yet
		if executionCount <= skip {
			if !silent {
				color.Yellow("(%s) Skipping execution %d of %d...", loopStartTime.Format("15:04:05"), executionCount, skip)
			}
		} else {
			// Execute the command once we've passed the skip threshold
			if clear {
				clearScreen()
			}
			if !silent {
				color.White("(%s) Executing command...", loopStartTime.Format("15:04:05"))
			}
			executeCommand(commandStr)
		}

		if !precision {
			// Standard mode: Wait for the full period after the command finishes.
			// Note: This wait period also applies during skipped executions to maintain timing
			if !silent {
				color.White("Waiting %d minute(s). Press Ctrl+C to stop.\n", period)
				fmt.Println() // Extra newline to match PS script's `n
			}
			time.Sleep(periodDuration)
		} else {
			// Precision mode: Account for execution time to maintain a fixed grid.
			currentTime := time.Now()
			var commandDuration time.Duration
			if executionCount > skip {
				commandDuration = currentTime.Sub(loopStartTime)
			} else {
				// During skipped executions, commandDuration is effectively 0
				commandDuration = 0
			}

			totalElapsed := currentTime.Sub(scriptStartTime)
			intervalsCompleted := math.Floor(totalElapsed.Minutes() / float64(period))
			nextTargetTime := scriptStartTime.Add(time.Duration(intervalsCompleted+1) * periodDuration)
			sleepDuration := nextTargetTime.Sub(currentTime)

			if sleepDuration.Seconds() > 0 {
				if !silent {
					color.White("Command took %.2fs. Waiting for %.0fs. Next run at %s.\nPress Ctrl+C to stop.\n", commandDuration.Seconds(), math.Round(sleepDuration.Seconds()), nextTargetTime.Format("15:04:05"))
				}
				time.Sleep(sleepDuration)
			} else {
				if !silent {
					color.Yellow("WARNING: Command execution time (%.2fs) overran its schedule. Running next iteration immediately.\n", commandDuration.Seconds())
				}
			}
		}
	}
}
