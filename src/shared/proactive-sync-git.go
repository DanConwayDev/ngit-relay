package shared

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		return err
	}

	gitServers := GetGitServersFromMaintainers(events, maintainers)
	if len(gitServers) == 0 {
		return fmt.Errorf("repo announcement event(s) doesnt list any git servers")
	}
	domain := getEnv("NGIT_DOMAIN", "")
	if domain != "" {
		// Create the currentGitRepoSubString to match
		currentGitRepoSubString := fmt.Sprintf("%s/%s/%s.git", domain, npub, identifier)

		// Create a new slice to hold the filtered results
		var filteredGitServers []string

		// Iterate over gitServers and filter out the unwanted ones
		for _, server := range gitServers {
			if !strings.Contains(server, currentGitRepoSubString) {
				filteredGitServers = append(filteredGitServers, server)
			}
		}

		// Update gitServers to the filtered list
		gitServers = filteredGitServers
	}
	// if len(gitServers) == 0 its still work proceeding to clean up state (delete branches)

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

	// check we can open the directory
	if _, err := os.Stat(repo_path); err != nil {
		return fmt.Errorf("cannot open repo directory: %w", err)
	}

	// Configure Git to accept the repository directory as safe
	configCmd := exec.Command("git", "config", "--global", "--add", "safe.directory", repo_path)
	if err := configCmd.Run(); err != nil {
		// Ignore this error as not all binaries have permission to do this (eg pre-receive git hook)
		// return fmt.Errorf("failed to set safe.directory: %w", err)
	}

	// Check repo_path is a git repository
	cmd := exec.Command("git", "-C", repo_path, "rev-parse", "--git-dir")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("not a git repository at %s: %w\nOutput: %s", repo_path, err, string(output))
	}

	// Get local refs
	cmd = exec.Command("git", "-C", repo_path, "show-ref", "--heads", "--tags")
	output, err := cmd.Output()
	localRefs := make(map[string]string)
	// It's okay if this fails with exit status 1 (no refs found)
	if err == nil || err.(*exec.ExitError).ExitCode() == 1 {
		if output != nil {
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
	} else {
		return fmt.Errorf("error getting local refs: %w", err)
	}

	// Build the state refs map (excluding HEAD)
	stateRefs := make(map[string]string)

	// Add branches to stateRefs
	for branch, hash := range state.Branches {
		ref := "refs/heads/" + branch
		stateRefs[ref] = hash
	}

	// Add tags to stateRefs
	for tag, hash := range state.Tags {
		// tag^{} is just a dereferenced version of the tag. we should ignore these
		if !strings.HasSuffix(tag, "^{}") {
			ref := "refs/tags/" + tag
			stateRefs[ref] = hash
		}
	}

	// Delete any refs that exist locally but aren't in state
	for ref := range localRefs {
		if _, exists := stateRefs[ref]; !exists {
			cmd = exec.Command("git", "-C", repo_path, "update-ref", "-d", ref)
			if output, err := cmd.CombinedOutput(); err != nil {
				gitErrors = append(gitErrors, fmt.Sprintf("failed to delete ref %s: %v, output: %s", ref, err, string(output)))
			}
		}
	}

	// Identify refs in nip34state that don't have local refs that match
	for ref, hash := range stateRefs {
		if localHash, exists := localRefs[ref]; !exists || localHash != hash {
			missingRefs[ref] = true
		}
	}

	// Try each git server
	for _, server := range gitServers {
		if len(missingRefs) == 0 {
			continue // no need to fetch any refs
		}
		if server == "" {
			continue // Skip empty server URLs
		}

		remoteName := fmt.Sprintf("origin_%d", time.Now().UnixNano())

		// Add remote
		cmd = exec.Command("git", "-C", repo_path, "remote", "add", remoteName, server)
		if output, err := cmd.CombinedOutput(); err != nil {
			gitErrors = append(gitErrors, fmt.Sprintf("failed to add remote %s: %v, output: %s", server, err, string(output)))
			continue
		}

		// Fetch refs with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		// Fetch all refs including orphaned tag commits
		fetchCmd := exec.CommandContext(ctx, "git", "-C", repo_path, "fetch", remoteName, "--tags", "--force")
		output, err := fetchCmd.CombinedOutput()
		cancel() // Always cancel the context

		if err != nil {
			gitErrors = append(gitErrors, fmt.Sprintf("failed to fetch from %s: %v, output: %s", server, err, string(output)))
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
			if output, err := cmd.CombinedOutput(); err != nil {
				gitErrors = append(gitErrors, fmt.Sprintf("failed to update ref %s to %s: %v, output: %s", ref, hash, err, string(output)))
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

	// Handle HEAD after all other refs are synced
	if state.HEAD != "" {
		targetRef := "refs/heads/" + state.HEAD
		if strings.HasPrefix(state.HEAD, "refs/") {
			targetRef = state.HEAD
		}

		// Check if target ref exists and update HEAD
		if _, exists := stateRefs[targetRef]; exists {
			cmd = exec.Command("git", "-C", repo_path, "symbolic-ref", "HEAD", targetRef)
			if output, err := cmd.CombinedOutput(); err != nil {
				gitErrors = append(gitErrors, fmt.Sprintf("failed to update HEAD to %s: %v, output: %s", targetRef, err, string(output)))
			}
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

	// Return git errors if any occurred, even if sync was successful
	if len(gitErrors) > 0 {
		return fmt.Errorf("repository synced with warnings: %v", strings.Join(gitErrors, "; "))
	}

	return nil
}

// updateState updates the git repository state based on the latest nostr state events.
// It fetches relevant events, identifies maintainers, determines the current state,
// and then calls ProactiveSyncGitFromStateAndServers to synchronize the local git repository.
func UpdateState(ctx context.Context, event *nostr.Event, pubkey string, npub string, identifier string, git_data_path string) error {

	events, err := FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		return err
	}
	events = append(events, *event)

	maintainers := GetMaintainers(events, pubkey, identifier)
	if len(maintainers) == 0 {
		return fmt.Errorf("repo announcement event from pubkey not on internal relay")
	}

	state, err := GetStateFromMaintainers(events, maintainers)
	if err != nil {
		return err
	}

	repo_path := git_data_path + "/" + npub + "/" + identifier + ".git"

	return ProactiveSyncGitFromStateAndServers(state, []string{}, repo_path)
}
