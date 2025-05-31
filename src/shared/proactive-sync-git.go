package shared

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip34"
)

// align state with nostr state using other git servers listed in announcement event
func ProactiveSyncGit(pubkey string, identifier string, git_data_path string) error {
	ctx := context.Background()

	events, err := FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		return err
	}

	maintainers := GetMaintainers(events, pubkey, identifier)
	if len(maintainers) == 0 {
		return fmt.Errorf("repo announcement event from pubkey not on internal relay")
	}

	state, err := GetStateFromMaintainers(events, maintainers)
	if err != nil {
		return err
	}

	gitServers := GetGitServersFromMaintainers(events, maintainers)
	if len(gitServers) == 0 {
		return fmt.Errorf("repo announcement event(s) doesnt list any git servers")
	}

	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		return err
	}
	repo_path := git_data_path + "/" + npub + "/" + identifier + ".git"

	return ProactiveSyncGitFromStateAndServers(state, gitServers, repo_path)

}

func GetGitServersFromMaintainers(events []nostr.Event, maintainers []string) []string {
	gitServers := []string{}

	for _, event := range events {
		repos := nip34.ParseRepository(event).Clone

		for _, repo := range repos {
			// Remove trailing slash if it exists
			if len(repo) > 0 && repo[len(repo)-1] == '/' {
				repo = repo[:len(repo)-1]
			}

			// Add the server to our list if it's not already there
			found := false
			for _, server := range gitServers {
				if server == repo {
					found = true
					break
				}
			}

			if !found {
				gitServers = append(gitServers, repo)
			}
		}
	}

	return gitServers
}

func ProactiveSyncGitFromStateAndServers(state *nip34.RepositoryState, gitServers []string, repo_path string) error {
	// Use cmd and the installed git client
	var gitErrors []string
	missingRefs := make(map[string]bool)

	// Check repo_path is a bare git repo
	cmd := exec.Command("git", "-C", repo_path, "rev-parse", "--is-bare-repository")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error checking if repository is bare: %w", err)
	}
	if strings.TrimSpace(string(output)) != "true" {
		return fmt.Errorf("repository at %s is not a bare repository", repo_path)
	}

	// Get local refs
	cmd = exec.Command("git", "-C", repo_path, "show-ref", "--heads", "--tags")
	output, err = cmd.Output()
	localRefs := make(map[string]string)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				hash := parts[0]
				ref := parts[1]
				localRefs[ref] = hash
			}
		}
	}

	// Build the state refs map
	stateRefs := make(map[string]string)

	// Add branches to stateRefs
	for branch, hash := range state.Branches {
		ref := "refs/heads/" + branch
		stateRefs[ref] = hash
	}

	// Add tags to stateRefs
	for tag, hash := range state.Tags {
		ref := "refs/tags/" + tag
		stateRefs[ref] = hash
	}

	// Delete any refs that exist locally but aren't in state
	for ref := range localRefs {
		if _, exists := stateRefs[ref]; !exists {
			cmd = exec.Command("git", "-C", repo_path, "update-ref", "-d", ref)
			if err := cmd.Run(); err != nil {
				gitErrors = append(gitErrors, fmt.Sprintf("failed to delete ref %s: %v", ref, err))
			}
		}
	}

	// Identify refs in nip34state that don't have local refs that match
	for ref, hash := range stateRefs {
		if localHash, exists := localRefs[ref]; !exists || localHash != hash {
			missingRefs[ref] = true
		}
	}

	// If all refs match, return early
	if len(missingRefs) == 0 {
		return nil
	}

	// Try each git server
	for _, server := range gitServers {
		remoteName := "origin_" + strconv.FormatInt(time.Now().UnixNano(), 10)

		// Add remote
		cmd = exec.Command("git", "-C", repo_path, "remote", "add", remoteName, server)
		if err := cmd.Run(); err != nil {
			gitErrors = append(gitErrors, fmt.Sprintf("failed to add remote %s: %v", server, err))
			continue
		}

		// Fetch refs
		cmd = exec.Command("git", "-C", repo_path, "fetch", remoteName)
		if err := cmd.Run(); err != nil {
			gitErrors = append(gitErrors, fmt.Sprintf("failed to fetch from %s: %v", server, err))
			// Clean up remote
			exec.Command("git", "-C", repo_path, "remote", "remove", remoteName).Run()
			continue
		}

		// Try to update each missing ref
		for ref := range missingRefs {
			hash := stateRefs[ref]

			// Check if the hash exists in the remote
			cmd = exec.Command("git", "-C", repo_path, "cat-file", "-e", hash)
			if err := cmd.Run(); err != nil {
				continue // Hash doesn't exist in this remote
			}

			// Update the ref
			cmd = exec.Command("git", "-C", repo_path, "update-ref", ref, hash)
			if err := cmd.Run(); err != nil {
				continue
			}

			// Ref is now synced, remove from missing
			delete(missingRefs, ref)
		}

		// Clean up remote
		exec.Command("git", "-C", repo_path, "remote", "remove", remoteName).Run()

		// If all refs are synced, we're done
		if len(missingRefs) == 0 {
			break
		}
	}

	// Return error if there are still missing refs
	if len(missingRefs) > 0 {
		missingRefsList := make([]string, 0, len(missingRefs))
		for ref := range missingRefs {
			missingRefsList = append(missingRefsList, ref)
		}

		return fmt.Errorf("failed to sync repository: git errors: %v, missing refs: %v",
			strings.Join(gitErrors, "; "),
			strings.Join(missingRefsList, ", "))
	}

	return nil
}
