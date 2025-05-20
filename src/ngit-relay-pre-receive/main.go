package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip34"

	"ngit-relay/shared"
)

func main() {
	pubkey, identifier := shared.GetPubKeyAndIdentifierFromPath()
	ctx := context.Background()

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, pubkey, identifier)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:cannot fetch state events:", err)
		os.Exit(1)
	}

	state, err := shared.GetState(events, pubkey, identifier)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting nostr repository state event:", err)
		os.Exit(1)
	}

	// Create a scanner to read from standard input
	scanner := bufio.NewScanner(os.Stdin)

	// Read each line from stdin - each one represents a ref to push
	for scanner.Scan() {
		line := scanner.Text()
		// Split the line into oldRev, newRev, and refName
		parts := strings.Fields(line)
		if len(parts) != 3 {
			fmt.Fprintln(os.Stderr, "Invalid input format:", line)
			os.Exit(1)
		}

		oldRev := parts[0]
		newRev := parts[1]
		refName := parts[2]

		// Reject branches with pr/ prefix
		if strings.HasPrefix(refName, "refs/heads/pr/") {
			fmt.Fprintln(os.Stderr, "Pushes 'pr/*' branches should be sent over nostr, not through the git server")
			os.Exit(1)
		}

		// Reject branches/tags that don't match state event
		if state == nil {
			// TODO: If state doesnt match or exist we could then look up the relays in all of the announcements and look for newer state annoucnments
			fmt.Fprintln(os.Stderr, "could not find a nostr state event for this repository")
			os.Exit(1)
		} else {
			matches, msg := MatchesStateEvent(refName, newRev, oldRev, state)
			if !matches {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(1)
			}
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

func MatchesStateEvent(ref string, to string, oldRev string, state *nip34.RepositoryState) (bool, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		for branchName, commitId := range state.Branches {
			if ref[11:] == branchName {
				if to == commitId {
					return true, nil
				}
				return false, fmt.Errorf("ref at different tip in our latest nostr state event")
			}
		}
	} else if strings.HasPrefix(ref, "refs/tags/") {
		for tagName, commitId := range state.Tags {
			if ref[10:] == tagName {
				if to == commitId {
					return true, nil
				}
				return false, fmt.Errorf("ref at different tip in our latest nostr state event")
			}
		}
	}
	return false, fmt.Errorf("ref not found in our latest nostr state event")
}
