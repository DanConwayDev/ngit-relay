package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip34"
)

func main() {

	pubkey, identifier := GetPubKeyAndIdentifier()

	ctx := context.Background()

	state, err := GetState(ctx, pubkey, identifier)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting nostr repository state event:", err)
		os.Exit(1)
	}

	// Create a scanner to read from standard input
	scanner := bufio.NewScanner(os.Stdin)

	// Read each line from stdin - each one represents a ref to push
	for scanner.Scan() {
		line := scanner.Text()
		// Split the line into oldrev, newrev, and refname
		parts := strings.Fields(line)
		if len(parts) != 3 {
			fmt.Fprintln(os.Stderr, "Invalid input format:", line)
			os.Exit(1)
		}

		oldRev := parts[0]
		newRev := parts[1]
		refName := parts[2]

		// reject branches with pr/ prefix
		if strings.HasPrefix(refName, "refs/heads/pr/") {
			fmt.Fprintln(os.Stderr, "Pushes 'pr/*' branches should be sent over nostr, not through the git server")
			os.Exit(1)
		}

		// reject branches / tags that don't match state event
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

func GetPubKeyAndIdentifier() (string, string) {
	// Get current path
	path, err := GetCurrentPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting repo path from within go binary:", err)
		os.Exit(1)
	}

	// Get the parent directory
	parentDir := filepath.Dir(path)

	// Get the grandparent directory
	grandParentDir := filepath.Dir(parentDir)

	// Get the base names of the parent and grandparent directories
	parentDirName := filepath.Base(parentDir)
	grandParentDirName := filepath.Base(grandParentDir)

	// Remove the .git postfix from the parent directory name if it exists
	identifier := strings.TrimSuffix(parentDirName, ".git")
	npub := grandParentDirName

	// decode the npub
	decoded_type, value, err := nip19.Decode(npub)
	pubkey, ok := value.(string)
	if err != nil || decoded_type != "npub" || !ok {
		fmt.Fprintln(os.Stderr, "invalid npub in directory name")
		return "", identifier // Return empty pubkey if type assertion fails
	}
	return pubkey, identifier
}

// GetCurrentPath returns the directory path of the current executable.
// If the executable was called through a symlink, it returns the directory
// containing the symlink. Otherwise, it returns the directory containing
// the actual binary.
func GetCurrentPath() (string, error) {
	// Determine how the program was invoked
	invoked, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}

	// Resolve any symlinks on the invoked path
	resolved, err := filepath.EvalSymlinks(invoked)
	if err != nil {
		return "", err
	}

	// If invoked differs from resolved, it's a symlink invocation
	if invoked != resolved {
		return filepath.Dir(invoked), nil
	}

	// Otherwise, return directory of the resolved path
	return filepath.Dir(resolved), nil
}

func AddReadOnlyOption(opt dgbadger.Options) dgbadger.Options {
	opt.ReadOnly = true
	return opt
}

func GetState(ctx context.Context, pubkey string, identifier string) (*nip34.RepositoryState, error) {
	db := badger.BadgerBackend{Path: "/khatru-data"} // TODO: path shouldn't be hard-coded but we can't pass it in as an argument
	if err := db.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	} // TODO: will this interfere with the instance opened by the relay? - can it be opened read-only?

	// TODO: We should get a list of all the maintainers in the announcement, recursively, then get the state events for all of them and choose the one with the biggest created_at
	filter := nostr.Filter{
		Kinds:   []int{30618},
		Authors: []string{pubkey},
		Tags: nostr.TagMap{
			"d": []string{identifier},
		},
		Limit: 1, // We only need to know if at least one exists
	}

	res, _ := db.QueryEvents(ctx, filter)
	for event := range res {
		state := nip34.ParseRepositoryState(*event)
		return &state, nil
	}
	return nil, fmt.Errorf("no valid state event found")
}

func MatchesStateEvent(ref string, to string, oldRev string, state *nip34.RepositoryState) (bool, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		for branchName, commitId := range state.Branches {
			if ref[11:] == branchName {
				if to == commitId {
					return true, nil
				} else {
					return false, fmt.Errorf("ref at different tip in our latest nostr state event")
				}
			}
			break
		}
	} else if strings.HasPrefix(ref, "refs/tags/") {
		for tagName, commitId := range state.Tags {
			if ref[10:] == tagName {
				if to == commitId {
					return true, nil
				} else {
					return false, fmt.Errorf("ref at different tip in our latest nostr state event")
				}
			}
			break
		}
	}
	return false, fmt.Errorf("ref not found in our latest nostr state event")
}
