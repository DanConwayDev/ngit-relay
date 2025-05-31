package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip34"
)

func FetchAnnouncementAndStateEventsFromRelay(ctx context.Context, identifier string) ([]nostr.Event, error) {
	// Create the filter for repository announcements and states
	identifierAnnFilter := nostr.Filter{
		Kinds: []int{nostr.KindRepositoryAnnouncement, nostr.KindRepositoryState},
		Tags: nostr.TagMap{
			"d": []string{identifier},
		},
	}

	relay, err := nostr.RelayConnect(ctx, "ws://localhost:3334")
	if err != nil {
		return nil, fmt.Errorf("could not connect to internal relay to find state event")
	}

	var events []nostr.Event

	// Use a context with cancel to handle cleanup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sub, err := relay.Subscribe(ctx, []nostr.Filter{identifierAnnFilter})
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to internal relay to find state event")
	}

	go func() {
		<-sub.EndOfStoredEvents
		cancel()
	}()

	for ev := range sub.Events {
		if ev != nil {
			events = append(events, *ev)
		}
	}
	return events, nil
}

func GetState(events []nostr.Event, pubkey string, identifier string) (*nip34.RepositoryState, error) {
	maintainers := GetMaintainers(events, pubkey, identifier)
	if len(maintainers) == 0 {
		return nil, fmt.Errorf("repo announcement event from pubkey not on internal relay")
	}
	return GetStateFromMaintainers(events, maintainers)
}

// GetMaintainers recursively finds maintainers for a given repository identifier
// and a trusted maintainer. checked is used internally to manage the recursion.
// if alice lists bob but bob doesnt list alice:
//   - getMaintainers(alice) ~> [alice, bob]
//   - getMaintainers(bob) ~> [bob]
func GetMaintainers(events []nostr.Event, pubkey string, identifier string, checked ...map[string]bool) []string {
	// Initialize the checked map if not provided
	var checkedMap map[string]bool
	if len(checked) > 0 {
		checkedMap = checked[0]
	} else {
		checkedMap = make(map[string]bool)
	}

	// Check if this pubkey has already been processed
	if checkedMap[pubkey] {
		return []string{} // Return empty if already checked
	}
	checkedMap[pubkey] = true // Mark this pubkey as checked

	// Find the announcement event
	event := FindAnnouncementEventByPubKeyIdentifier(events, pubkey, identifier)
	if event == nil {
		return []string{} // Return empty if no event found
	}

	// Parse the repository to get maintainers
	repo := nip34.ParseRepository(*event)

	// Include the current pubkey in the checked map
	checkedMap[pubkey] = true

	// Recursively find maintainers for each maintainer
	for _, maintainerPubKey := range repo.Maintainers {
		// Call recursively and ignore the returned output
		GetMaintainers(events, maintainerPubKey, repo.ID, checkedMap)
	}

	// Collect all unique maintainers from the checked map
	var maintainers []string
	for key := range checkedMap {
		maintainers = append(maintainers, key)
	}

	return maintainers
}

// FindAnnouncementEventByPubKeyIdentifier searches a list of events for a NIP-34
// repository announcement (kind 30317) that matches a specific pubkey and identifier (d tag).
func FindAnnouncementEventByPubKeyIdentifier(events []nostr.Event, pubkey string, identifier string) *nostr.Event {
	for _, event := range events {
		// Check if the PubKey matches and it's a repository announcement
		if event.Kind == nostr.KindRepositoryAnnouncement && event.PubKey == pubkey {
			repo := nip34.ParseRepository(event) // Assuming this function returns a struct with an ID field
			if repo.ID == identifier {
				return &event // Return a pointer to the matching event
			}
		}
	}
	return nil // Return nil if no matching event is found
}

// GetStateFromMaintainers finds the latest NIP-34 repository state event (kind 30318)
// from a list of events, authored by one of the provided maintainers.
func GetStateFromMaintainers(events []nostr.Event, maintainers []string) (*nip34.RepositoryState, error) {
	var latestEvent *nostr.Event
	var latestTimestamp nostr.Timestamp

	// Create a map for quick lookup of maintainers
	maintainerMap := make(map[string]bool)
	for _, maintainer := range maintainers {
		maintainerMap[maintainer] = true
	}

	// Iterate through events to find the latest valid event
	for i := range events {
		event := events[i] // Use a copy or index to avoid issues with pointer if events slice is modified elsewhere
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

	return nil, fmt.Errorf("no valid NIP-34 state event found from maintainers")
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

func GetPubKeyAndIdentifierFromPath() (string, string, string, error) {
	// Get current path
	path, err := GetCurrentPath()
	if err != nil {
		return "", "", "", err
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
	pubkey, err := GetPubkeyFromNpub(npub)
	if err != nil {
		return "", npub, identifier, fmt.Errorf("could not get pubkey from npub: %s", npub)
	}
	return pubkey, npub, identifier, nil
}

func GetPubkeyFromNpub(npub string) (string, error) {
	decodedType, value, err := nip19.Decode(npub)
	if err != nil {
		return "", err
	}
	pubkey, ok := value.(string)
	if decodedType != "npub" || !ok {
		return "", fmt.Errorf("invalid npub: %s", npub)
	}
	return pubkey, nil
}
