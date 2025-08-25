package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Isame Load Balancer Control Tool v0.1.0")
		fmt.Println("Usage: isame-ctl [command]")
		fmt.Println("Commands:")
		fmt.Println("  version  - Show version information")
		fmt.Println("  help     - Show this help message")
		return
	}

	command := os.Args[1]
	switch command {
	case "version":
		fmt.Println("isame-ctl version 0.1.0")
		fmt.Println("Phase 0 - Bootstrap implementation")
	case "help":
		fmt.Println("Isame Load Balancer Control Tool")
		fmt.Println("This is a basic CLI tool for Phase 0.")
		fmt.Println("More functionality will be added in later phases.")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Use 'isame-ctl help' for available commands.")
		os.Exit(1)
	}
}
