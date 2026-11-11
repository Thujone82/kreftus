package main

import (
	"log"
)

func main() {
	if err := RunGUI(); err != nil {
		log.Fatalf("Skirmish DICE error: %v", err)
	}
}
