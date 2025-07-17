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
	fmt.Println("    rc \"<command>\" [period] [-p]")
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

	color.Yellow("EXAMPLES")
	color.Green("    rc \"go run main.go\" 1")
	fmt.Println("    Runs 'go run main.go' every 1 minute.")
	fmt.Println()
	color.Green("    rc \"gw Portland\" 10 -p")
	fmt.Println("    Runs the 'gw' command on a fixed 10-minute schedule.")
	fmt.Println()
}

func main() {
	// Manual argument parsing is used to allow flags to be placed anywhere in the command.
	// The standard `flag` package stops parsing at the first non-flag argument.
	var commandStr string
	period := 5 // Default period in minutes
	var precision bool
	var nonFlagArgs []string

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-p", "-precision":
			precision = true
		case "-h", "-help":
			printUsage()
			os.Exit(0)
		default:
			nonFlagArgs = append(nonFlagArgs, arg)
		}
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
	}

	// Exit if no command was provided either by argument or interactively.
	if commandStr == "" {
		fmt.Println("No command provided. Exiting.")
		return
	}

	// --- Initial Output ---
	fmt.Printf("Running \"%s\" every %d minute(s). Press Ctrl+C to stop.\n\n", commandStr, period)
	var scriptStartTime time.Time
	if precision {
		scriptStartTime = time.Now()
		color.Cyan("Precision mode is enabled. Aligning to grid starting at %s.", scriptStartTime.Format("15:04:05"))
	}

	// --- Main Execution Loop ---
	periodDuration := time.Duration(period) * time.Minute
	for {
		loopStartTime := time.Now()
		color.White("(%s) Executing command...", loopStartTime.Format("15:04:05"))
		executeCommand(commandStr)

		if !precision {
			// Standard mode: Wait for the full period after the command finishes.
			color.White("Waiting %d minute(s). Press Ctrl+C to stop.\n", period)
			fmt.Println() // Extra newline to match PS script's `n
			time.Sleep(periodDuration)
		} else {
			// Precision mode: Account for execution time to maintain a fixed grid.
			currentTime := time.Now()
			commandDuration := currentTime.Sub(loopStartTime)

			totalElapsed := currentTime.Sub(scriptStartTime)
			intervalsCompleted := math.Floor(totalElapsed.Minutes() / float64(period))
			nextTargetTime := scriptStartTime.Add(time.Duration(intervalsCompleted+1) * periodDuration)
			sleepDuration := nextTargetTime.Sub(currentTime)

			if sleepDuration.Seconds() > 0 {
				color.White("Command took %.2fs. Waiting for %.0fs. Next run at %s.\nPress Ctrl+C to stop.\n", commandDuration.Seconds(), math.Round(sleepDuration.Seconds()), nextTargetTime.Format("15:04:05"))
				time.Sleep(sleepDuration)
			} else {
				color.Yellow("WARNING: Command execution time (%.2fs) overran its schedule. Running next iteration immediately.\n", commandDuration.Seconds())
			}
		}
	}
}
