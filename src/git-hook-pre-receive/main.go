package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Create a scanner to read from standard input
	scanner := bufio.NewScanner(os.Stdin)

	// Read each line from stdin
	for scanner.Scan() {
		line := scanner.Text()
		// Split the line into oldrev, newrev, and refname
		parts := strings.Fields(line)
		if len(parts) != 3 {
			fmt.Fprintln(os.Stderr, "Invalid input format:", line)
			os.Exit(1)
		}

		// oldRev := parts[0]
		// newRev := parts[1]
		refName := parts[2]

		// Check if the refname matches the pattern
		if strings.HasPrefix(refName, "refs/heads/pr/") {
			fmt.Fprintln(os.Stderr, "Pushes 'pr/*' branches should be sent over nostr, not through the git server")
			os.Exit(1)
		}
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	// If no issues, exit with success
	os.Exit(0)
}
