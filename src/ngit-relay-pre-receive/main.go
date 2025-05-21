package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip34"
	"go.uber.org/zap"

	"ngit-relay/shared"
)

func main() {
	shared.Init("ngit-relay-pre-receive")
	logger := shared.L()
	// Defers are not guaranteed to run on os.Exit(), but useful for normal completion or panic.
	defer logger.Sync()

	pubkey, identifier := shared.GetPubKeyAndIdentifierFromPath()
	if pubkey == "" {
		// GetPubKeyAndIdentifierFromPath logs its own error and returns "" on failure
		// shared.L() would not be initialized yet if GetPubKeyAndIdentifierFromPath itself fails
		// to call Init. However, GetPubKeyAndIdentifierFromPath uses fmt.Fprintln for its errors.
		// For now, assume shared.Init is called inside GetPubKeyAndIdentifierFromPath or similar early.
		// A more robust solution would be to have GetPubKeyAndIdentifierFromPath take a logger.
		os.Exit(1)
	}
	logger = logger.With(zap.String("repoIdentifier", identifier), zap.String("repoNpub", pubkey))

	ctx := context.Background()

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		logger.Error("Cannot fetch state events from relay", zap.Error(err))
		os.Exit(1)
	}

	state, err := shared.GetState(events, pubkey, identifier)
	if err != nil {
		logger.Error("Error getting nostr repository state event", zap.Error(err))
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
			logger.Error("Invalid input format from git hook", zap.String("line", line))
			os.Exit(1)
		}

		oldRev := parts[0]
		newRev := parts[1]
		refName := parts[2]
		refLogger := logger.With(
			zap.String("refName", refName),
			zap.String("oldRev", oldRev),
			zap.String("newRev", newRev),
		)

		// Reject branches with pr/ prefix
		if strings.HasPrefix(refName, "refs/heads/pr/") {
			refLogger.Warn("Rejecting push of 'pr/*' branch via git server")
			fmt.Fprintln(os.Stderr, "Pushes 'pr/*' branches should be sent over nostr, not through the git server")
			os.Exit(1)
		}

		// Reject branches/tags that don't match state event
		if state == nil {
			refLogger.Warn("Rejecting push, no nostr state event found for this repository")
			// TODO: If state doesnt match or exist we could then look up the relays in all of the announcements and look for newer state annoucnments
			fmt.Fprintln(os.Stderr, "could not find a nostr state event for this repository")
			os.Exit(1)
		} else {
			matches, matchErr := MatchesStateEvent(refName, newRev, oldRev, state)
			if !matches {
				refLogger.Warn("Rejecting push, ref does not match nostr state event", zap.Error(matchErr))
				fmt.Fprintln(os.Stderr, matchErr) // Send the specific error message to git client
				os.Exit(1)
			}
			refLogger.Info("Allowing push, ref matches nostr state event")
		}
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		logger.Error("Error reading input from git hook stdin", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Pre-receive hook completed successfully, all refs accepted.")
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
