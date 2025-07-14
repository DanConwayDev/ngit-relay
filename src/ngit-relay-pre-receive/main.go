package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip34"
	"go.uber.org/zap"

	"ngit-relay/shared"
)

func main() {
	shared.Init("ngit-relay-pre-receive", false, false)
	logger := shared.L()

	pubkey, npub, identifier, err := shared.GetPubKeyAndIdentifierFromPath()

	if err != nil {
		logger.Fatal(LogStderr("cannot extract repo pubkey and identifier from path"), zap.Error(err))
	}
	logger = logger.With(zap.String("identifier", identifier), zap.String("npub", npub))
	ctx := context.Background()

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		logger.Fatal(LogStderr("cannot fetch state events from internal relay", err), zap.Error(err))
	}
	logger = logger.With(zap.Any("events", events))

	state, err := shared.GetState(events, pubkey, identifier)
	if err != nil {
		logger.Fatal(LogStderr("state event not on internal relay", err), zap.Error(err))
	}

	// Create a scanner to read from standard input
	scanner := bufio.NewScanner(os.Stdin)

	// Read each line from stdin - each one represents a ref to push
	for scanner.Scan() {
		line := scanner.Text()
		refLogger := logger.With(zap.String("line", line))
		// Split the line into oldRev, newRev, and refName
		parts := strings.Fields(line)
		if len(parts) != 3 {
			refLogger.Fatal(LogStderr("Invalid input format from git hook"))
		}

		oldRev := parts[0]
		newRev := parts[1]
		refName := parts[2]

		if strings.HasPrefix(refName, "refs/nostr/") {
			if nostr.IsValid32ByteHex(strings.Replace(refName, "refs/nostr/", "", 1)) {
				refLogger.Debug("Allowing push for PR ref", zap.Any("tags", state.Tags), zap.Any("branches", state.Branches))
				continue
			}
			logger.Fatal(LogStderr("refs/nostr/<event-id> must use a valid event id", err), zap.Error(err))
		}

		// Reject branches with pr/ prefix
		if strings.HasPrefix(refName, "refs/heads/pr/") {
			refLogger.Debug(LogStderr("'pr/*' branches should be sent over nostr, not through the git server"))
			os.Exit(1)
		}

		// Reject branches/tags that don't match state event
		matches, err := MatchesStateEvent(refName, newRev, oldRev, state)
		if !matches {
			refLogger.Debug(LogStderr(err.Error()), zap.Error(err))
			os.Exit(1)
		}
		refLogger.Debug("Allowing push for ref as it matches nostr state event", zap.Any("tags", state.Tags), zap.Any("branches", state.Branches))
	}

	// add any refs (tags, branches, etc) that required this git data
	git_data_path, err := os.Getwd()
	if err != nil {
		logger.Warn(LogStderr("cannot get current working directory for git_data_path"), zap.Error(err))
	}
	repo_path := filepath.Join(git_data_path, npub, identifier+".git")
	if err := shared.ProactiveSyncGitFromStateAndServers(state, []string{}, repo_path); err != nil {
		logger.Warn(LogStderr("Error syncing git state", err), zap.String("repo_path", repo_path), zap.Error(err))
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		logger.Fatal(LogStderr("Error reading input from git hook stdin", err), zap.Error(err))
	}

	logger.Debug("Pre-receive hook completed successfully, all refs accepted.")
	// If no issues, exit with success
	os.Exit(0)
}

// prints message and error to stderr - in a server-side git hook, this gets printed in git clients as "remote: ${msg}"
func LogStderr(msg string, err ...error) string {
	errMsg := ""
	if len(err) > 0 && err[0] != nil {
		msg += ": " + err[0].Error()
	}
	os.Stderr.WriteString("error: " + msg + errMsg + "\n")
	return msg
}

func MatchesStateEvent(ref string, to string, oldRev string, state *nip34.RepositoryState) (bool, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		for branchName, commitId := range state.Branches {
			if ref[11:] == branchName {
				if to == commitId {
					return true, nil
				}
				return false, fmt.Errorf("cannot push %s to %s as nostr state event is at %s", ref[11:], to[:7], commitId[:7])
			}
		}
	} else if strings.HasPrefix(ref, "refs/tags/") {
		for tagName, commitId := range state.Tags {
			if ref[10:] == tagName {
				if to == commitId {
					return true, nil
				}
				return false, fmt.Errorf("cannot push %s to %s as nostr state event is at %s", ref[10:], to[:7], commitId[:7])
			}
		}
	}
	return false, fmt.Errorf("%s not found in our latest nostr state event", ref)
}
