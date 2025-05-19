package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func GetPubKeyAndIdentifier() (string, string) {
	// Get current path
	path, err := GetCurrentPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting repo path from within go binary:", err)
		os.Exit(1)
	}

	// Get the parent and grandparent directories
	parentDir := filepath.Dir(path)
	grandParentDir := filepath.Dir(parentDir)

	// Get the base names of the parent and grandparent directories
	parentDirName := filepath.Base(parentDir)
	grandParentDirName := filepath.Base(grandParentDir)

	// Remove the .git postfix from the parent directory name if it exists
	identifier := strings.TrimSuffix(parentDirName, ".git")
	npub := grandParentDirName

	// Decode the npub
	decodedType, value, err := nip19.Decode(npub)
	pubkey, ok := value.(string)
	if err != nil || decodedType != "npub" || !ok {
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

func GetState(ctx context.Context, pubkey string, identifier string) (*nip34.RepositoryState, error) {
	filter := nostr.Filter{
		Kinds:   []int{nostr.KindRepositoryAnnouncement, nostr.KindRepositoryState},
		Authors: []string{pubkey},
		Tags: nostr.TagMap{
			"d": []string{identifier},
		},
	}

	relay, err := nostr.RelayConnect(ctx, "ws://localhost:3334")
	if err != nil {
		return nil, fmt.Errorf("could not connect to internal relay to find state event")
	}
	sub, err := relay.Subscribe(ctx, []nostr.Filter{filter})
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to internal relay to find state event")
	}

	// Read events from the channel into a slice
	var events []nostr.Event
	for eventPtr := range sub.Events {
		if eventPtr != nil {
			events = append(events, *eventPtr) // Dereference the pointer and append to the slice
		}
	}

	maintainers := getMaintainers(events, pubkey, identifier)
	state, err := getStateFromMaintainers(events, maintainers)
	if err != nil {
		return nil, fmt.Errorf("no valid state event found")
	}
	return state, nil
}

// currently doesnt need to take identifer but this makes it reusable
func getMaintainers(events []nostr.Event, pubkey string, identifier string, checked ...map[string]bool) []string {
	// Initialize the checked map if not provided
	var checkedMap map[string]bool
	if len(checked) > 0 {
		checkedMap = checked[0]
	} else {
		checkedMap = make(map[string]bool)
	}

	var maintainers []string

	// Check if this pubkey has already been processed
	if checkedMap[pubkey] {
		return maintainers // Return empty if already checked
	}
	checkedMap[pubkey] = true // Mark this pubkey as checked

	// Find the announcement event
	event := findAnnouncementEventByPubKeyIdentifier(events, pubkey, identifier)
	if event == nil {
		return maintainers // Return empty if no event found
	}

	// Parse the repository to get maintainers
	repo := nip34.ParseRepository(*event)
	maintainers = append(maintainers, repo.Maintainers...)

	// Recursively find maintainers for each maintainer
	for _, maintainerPubKey := range repo.Maintainers {
		subMaintainers := getMaintainers(events, maintainerPubKey, repo.ID, checkedMap)
		maintainers = append(maintainers, subMaintainers...)
	}

	return maintainers
}

func findAnnouncementEventByPubKeyIdentifier(events []nostr.Event, pubkey string, identifier string) *nostr.Event {
	for _, event := range events {
		// Check if the PubKey matches
		if event.PubKey == pubkey {
			repo := nip34.ParseRepository(event) // Assuming this function returns a struct with an ID field
			if repo.ID == identifier {
				return &event // Return a pointer to the matching event
			}
		}
	}
	return nil // Return nil if no matching event is found
}

func getStateFromMaintainers(events []nostr.Event, maintainers []string) (*nip34.RepositoryState, error) {
	var latestEvent *nostr.Event
	var latestTimestamp nostr.Timestamp

	// Create a map for quick lookup of maintainers
	maintainerMap := make(map[string]bool)
	for _, maintainer := range maintainers {
		maintainerMap[maintainer] = true
	}

	// Iterate through events to find the latest valid event
	for _, event := range events {
		// Check if the event matches the criteria
		if event.Kind == nostr.KindRepositoryState && maintainerMap[event.PubKey] {
			// Check if this event is the latest one
			if event.CreatedAt > latestTimestamp {
				latestTimestamp = event.CreatedAt
				latestEvent = &event
			}
		}
	}

	// If a valid event was found, parse and return its state
	if latestEvent != nil {
		state := nip34.ParseRepositoryState(*latestEvent)
		return &state, nil
	}

	return nil, fmt.Errorf("no valid event found")
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
