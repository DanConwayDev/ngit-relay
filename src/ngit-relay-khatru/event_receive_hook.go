package main

import (
	"context"
	"errors"
	"fmt"
	"ngit-relay/internal/nip34util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func EventRecieveHook(relay *khatru.Relay, git_data_path string) func(ctx context.Context, event *nostr.Event) {
	return func(ctx context.Context, event *nostr.Event) {
		// TODO - what if we receieve a delete request for an announcement?
		if event.Kind == nostr.KindRepositoryAnnouncement {
			npub, _ := nip19.EncodePublicKey(event.PubKey)
			identifier := event.Tags.Find("d")[1]

			// TODO confrm that the current event also gets returned in this:
			announcementEvents, err := FetchAnnouncementAndStateEvents(ctx, relay, event.PubKey, identifier)
			if err != nil {
				fmt.Println("Error fetching announcement events:", err)
				return
			}

			maintainers := nip34util.GetMaintainers(announcementEvents, event.PubKey, identifier)

			// if git repo exists, no action needed.
			// Edge cases to consider:
			//   - the maintainer removed this ngit-relay instance? Do we want to delete it? say after 48hrs after looking for symblinks etc?
			//   - only some of the maintainers list the ngit-relay instance and therefore don't push to it / update the relay - the current behaviour is acceptable
			repo_exists, repo_is_symlink, symlink_target, err := CheckRepo(git_data_path, npub, identifier)
			if err != nil {
				fmt.Println("Error checking if repo exists:", err)
				return
			}

			if repo_exists && !repo_is_symlink {
				// TODO check if a git repo also exists for a maintainer (that might have been just added)
				// if yes - decide which one to delete and replace with a symbolink. (most recently published state event wins)
				return // repo already exists - no change needed
			}

			if repo_is_symlink {
				symlink_correct, err := SymlinkPointsToValidMaintainer(git_data_path, symlink_target, maintainers)
				if err != nil {
					fmt.Println("Error checking SymlinkPointsToValidMaintainer:", err)
					return
				}
				if symlink_correct {
					return // symblink points to a valid maintainer - no change needed
				}
				// symblink points to a now invalid maintainer - delete the syblink and proceed to create new symlink or git repo
				DeleteSymlink(git_data_path, npub, identifier)
				if err != nil {
					fmt.Println("Error failed to delete symblink:", err)
					return
				}
			}

			// symblink to another maintainers git repo if it exists
			for _, maintainer := range maintainers {
				repo_exists, _, _, err := CheckRepo(git_data_path, maintainer, identifier)
				if err != nil {
					fmt.Println("Error checking if maintainer repo exists:", err)
					return
				}
				if repo_exists {
					err := CreateSymLink(git_data_path, npub, maintainer, identifier)
					if err != nil {
						fmt.Println("Error creating smylink to existing maintainer git repo:", err)
						return
					}
					return // symblink created
				}
			}

			// create git repo
			CreateRepository(git_data_path, npub, identifier)
			return
		}
	}
}

func FetchAnnouncementAndStateEvents(ctx context.Context, relay *khatru.Relay, pubkey string, identifier string) ([]nostr.Event, error) {
	// Create the filter for repository announcements
	identifierAnnFilter := nostr.Filter{
		Kinds:   []int{nostr.KindRepositoryAnnouncement, nostr.KindRepositoryState},
		Authors: []string{pubkey},
		Tags: nostr.TagMap{
			"d": []string{identifier},
		},
	}

	var announcementEvents []nostr.Event

	// Iterate over each function in the relay.QueryEvents slice
	for _, queryFunc := range relay.QueryEvents {
		// Call the function to get the events channel
		eventsChan, err := queryFunc(ctx, identifierAnnFilter)
		if err != nil {
			return nil, err // Return the error if the query fails
		}

		// Collect events from the channel
		for eventPtr := range eventsChan {
			if eventPtr != nil {
				announcementEvents = append(announcementEvents, *eventPtr) // Dereference the pointer and append to the slice
			}
		}
	}

	// Return the collected announcement events
	return announcementEvents, nil
}

// CheckRepo checks if the repository exists and if it is a symlink.
func CheckRepo(git_data_path, npub, identifier string) (repo_exists, repo_is_symlink bool, symlink_target string, err error) {
	repoPath := ConstructRepoPath(git_data_path, npub, identifier)

	info, err := os.Stat(repoPath)
	if err == nil {
		// Repo directory exists
		repo_exists = true
		repo_is_symlink = info.Mode()&os.ModeSymlink != 0

		// If it's a symlink, get the target
		if repo_is_symlink {
			symlink_target, err = os.Readlink(repoPath)
			if err != nil {
				return repo_exists, repo_is_symlink, "", err // Return error if Readlink fails
			}
		}
		return repo_exists, repo_is_symlink, symlink_target, nil
	} else if os.IsNotExist(err) {
		// Repo directory does not exist
		repo_exists = false
		repo_is_symlink = false
		return repo_exists, repo_is_symlink, "", nil
	}
	// Return the error if it is not a "not exist" error
	return false, false, "", err
}

// CreateSymLink creates a symbolic link from the source repository to the destination repository.
func CreateSymLink(git_data_path, from_npub string, to_npub string, identifier string) error {
	// Construct the source and destination paths
	fromPath := ConstructRepoPath(git_data_path, from_npub, identifier)
	toPath := ConstructRepoPath(git_data_path, to_npub, identifier)

	// Remove the destination path if it already exists
	if _, err := os.Lstat(toPath); err == nil {
		if err := os.RemoveAll(toPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink or directory: %w", err)
		}
	}

	// Create the symbolic link
	if err := os.Symlink(fromPath, toPath); err != nil {
		return fmt.Errorf("failed to create symlink from %s to %s: %w", fromPath, toPath, err)
	}

	return nil
}

func SymlinkPointsToValidMaintainer(git_data_path string, symlink_target string, maintainers []string) (bool, error) {
	// Extract npub and identifier from the symlink target
	maintainer_npub, _, err := ExtractNpubIdentifierFromRepoPath(git_data_path, symlink_target)
	if err != nil {
		return false, fmt.Errorf("error extracting npub from symlink path %s: %v", symlink_target, err)
	}

	// Convert npub to public key
	maintainer_pubkey := nPubToPubkey(maintainer_npub)

	// Check if the public key is in the list of maintainers
	for _, item := range maintainers {
		if item == maintainer_pubkey {
			return true, nil // Found a valid maintainer
		}
	}

	return false, nil // Not found
}

// DeleteSymlink deletes the symbolic link for the given npub and identifier.
func DeleteSymlink(git_data_path, npub, identifier string) error {
	// Construct the path to the symlink
	symlinkPath := ConstructRepoPath(git_data_path, npub, identifier)

	// Attempt to remove the symlink
	err := os.Remove(symlinkPath)
	if err != nil {
		return fmt.Errorf("error deleting symlink %s: %v", symlinkPath, err)
	}

	return nil // Successfully deleted
}

// ConstructRepoPath constructs the repository path from the given parameters.
func ConstructRepoPath(git_data_path, npub, identifier string) string {
	return filepath.Join(git_data_path, npub, identifier+".git")
}

func ExtractNpubIdentifierFromRepoPath(git_data_path string, repo_path string) (npub string, identifier string, err error) {
	// Normalize the repo_path to ensure it uses the same separator as git_data_path
	repo_path = filepath.Clean(repo_path)

	// Get the absolute path of git_data_path
	gitDataPathAbs, err := filepath.Abs(git_data_path)
	if err != nil {
		return "", "", err
	}

	// Check if the repo_path starts with git_data_path
	if !strings.HasPrefix(repo_path, gitDataPathAbs) {
		return "", "", errors.New("repo_path does not start with git_data_path")
	}

	// Remove the git_data_path prefix from repo_path
	relativePath := strings.TrimPrefix(repo_path, gitDataPathAbs)

	// Split the relative path into components
	parts := strings.Split(strings.TrimPrefix(relativePath, string(filepath.Separator)), string(filepath.Separator))

	// Check if we have at least two parts (npub and identifier)
	if len(parts) < 2 {
		return "", "", errors.New("repo_path does not contain enough parts to extract npub and identifier")
	}

	// Extract npub and identifier
	npub = parts[0]
	identifierWithExt := parts[1]

	// Remove the ".git" extension from identifier
	identifier = strings.TrimSuffix(identifierWithExt, ".git")

	return npub, identifier, nil
}

func CreateRepository(git_data_path string, npub string, identifier string) {
	// repo doesn't exist
	fmt.Println("Creating empty git repo for ", npub+"/"+identifier)

	repo_path := ConstructRepoPath(git_data_path, npub, identifier)

	// create directory
	err := os.MkdirAll(repo_path, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}
	fmt.Println("Directory created successfully:", repo_path)

	// init git repo
	cmd := exec.Command("git", "init", "--bare", repo_path)
	cmd.Dir = git_data_path // Set the working directory for the command
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error initializing Git repository:", err)
		return
	}

	// allow unauthenticated push (we handle write permissions via pre-receive git hook)
	cmd = exec.Command("git", "config", "http.receivepack", "true")
	cmd.Dir = repo_path // Set the working directory for the command
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error configuring Git to enable push:", err)
		return
	}

	// allow uploadpack from tips - required for ngit remote helper to pull desired data
	// without it ngit won't be able to fetch, pull or clone.
	cmd = exec.Command("git", "config", "uploadpack.allowTipSHA1InWant", "true")
	cmd.Dir = repo_path // Set the working directory for the command
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error configuring Git to enable uploadpakc.allowTipSHA1InWant:", err)
		return
	}

	// allow allowUnreachable which enables ngit to download blobs not in the ancestory of tips.
	// this might be useful if we store blobs related to pr/* without storing the tips.
	// it is also might be helpful in other scenarios where the git server and nostr state
	// event is out of sync.
	cmd = exec.Command("git", "config", "uploadpack.allowUnreachable", "true")
	cmd.Dir = repo_path // Set the working directory for the command
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error configuring Git to enable uploadpakc.allowUnreachable:", err)
		return
	}

	// Create symlink to pre-receive hook
	err = os.Symlink("/usr/local/bin/ngit-relay-pre-receive", repo_path+"/hooks/pre-receive")
	if err != nil {
		fmt.Println("Error creating symlink:", err)
		return
	}

	// set permissions
	err = os.Chmod(repo_path, 0777)
	if err != nil {
		fmt.Println("Error changing permissions:", err)
		return
	}

	// ensure correct ownership for git-http-backend
	cmd = exec.Command("chown", "-R", "nginx:nginx", repo_path)
	_, err = cmd.Output()
	if err != nil {
		fmt.Println("Error changing ownership:", err)
		return
	}

	fmt.Println("Created empty git repo")
}
