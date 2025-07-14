package main

import (
	"bufio"
	"flag"
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

func main() {
	// Flag definitions to match rc.ps1
	period := flag.Int("period", 5, "The time to wait between command executions, in minutes.")
	var precision bool
	flag.BoolVar(&precision, "precision", false, "Enables precision mode to account for command execution time.")
	flag.BoolVar(&precision, "p", false, "Alias for -precision.")

	flag.Parse()

	commandStr := strings.Join(flag.Args(), " ")

	// --- Interactive Mode ---
	// If no command is provided via arguments, prompt the user for input.
	if commandStr == "" {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Command: ")
		cmdInput, _ := reader.ReadString('\n')
		commandStr = strings.TrimSpace(cmdInput)

		fmt.Print("Period (minutes) [default: 5]: ")
		periodInput, _ := reader.ReadString('\n')
		periodInput = strings.TrimSpace(periodInput)
		if p, err := strconv.Atoi(periodInput); err == nil && p > 0 {
			*period = p
		} else {
			*period = 5 // Default if empty or invalid
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
	fmt.Printf("Running \"%s\" every %d minute(s). Press Ctrl+C to stop.\n\n", commandStr, *period)
	var scriptStartTime time.Time
	if precision {
		scriptStartTime = time.Now()
		color.Cyan("Precision mode is enabled. Aligning to grid starting at %s.", scriptStartTime.Format("15:04:05"))
	}

	// --- Main Execution Loop ---
	periodDuration := time.Duration(*period) * time.Minute
	for {
		loopStartTime := time.Now()
		color.White("(%s) Executing command...", loopStartTime.Format("15:04:05"))
		executeCommand(commandStr)

		if !precision {
			// Standard mode: Wait for the full period after the command finishes.
			color.White("Waiting %d minute(s). Press Ctrl+C to stop.\n", *period)
			fmt.Println() // Extra newline to match PS script's `n
			time.Sleep(periodDuration)
		} else {
			// Precision mode: Account for execution time to maintain a fixed grid.
			currentTime := time.Now()
			commandDuration := currentTime.Sub(loopStartTime)

			totalElapsed := currentTime.Sub(scriptStartTime)
			intervalsCompleted := math.Floor(totalElapsed.Minutes() / float64(*period))
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