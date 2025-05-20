package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"ngit-relay/shared"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	pubkey, identifier := shared.GetPubKeyAndIdentifierFromPath()
	ctx := context.Background()

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:cannot fetch state events:", err)
		os.Exit(1)
	}

	// ensure our git repo is as up-to-date as possible with our nostr state
	CheckStateMatchesNostrAndTryToRepair(events, pubkey, identifier)

	// Update other git repos which use this pubkey as a maintainer
	// Identify the git repos
	pubkeys := shared.GetOtherPubkeysThatUseThisPubkeyAsRepoMaintainer(events, pubkey)

	// these probably should now inherit our maintainers state but we will check
	for _, otherPubkey := range pubkeys {
		state, err := shared.GetState(events, otherPubkey, identifier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting nostr repository state event for following maintainer %s: %v\n", otherPubkey, err)
			continue
		}
		if state.Event.PubKey != pubkey {
			// skipping - another maintainer probably published a state after the original pubkey but before the git push to the server was performed.
			continue

		}
		if err := MirrorRepo(pubkey, otherPubkey, identifier); err != nil {
			fmt.Fprintf(os.Stderr, "error updating git repository for following maintainer %s: %v\n", otherPubkey, err)
			continue
		}
		fmt.Printf("updating git repository for following maintainer: %s\n", otherPubkey)
	}

	os.Exit(0)
}

func CheckStateMatchesNostrAndTryToRepair(events []nostr.Event, pubkey string, identifier string) error {
	// state, err := shared.GetState(events, pubkey, identifier)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error getting nostr repository state event for pubkey %s: %v\n", pubkey, err)
	// 	os.Exit(1)
	// }

	// 1. apply fixes possible within local repo:
	// get repo state with `git show-ref --heads --tags`
	// check for items present on nostr but not in local or at a different tip
	// check the oids exist locally and if so set the refs with the tips and keep a record of oids we need to fetch
	// delete all refs not on nostr (doing this too early might loose the oids)
	// check HEAD `git symbolic-ref HEAD` matches nostr and if not fix. do we need to check for other symbolic refs?

	// 2. get fixes from other local repos:
	// foeach each pubkey in otherPubkeySources add the remote doesnt already exist with name npub
	// check the the presence of missing oids
	// fetch those oids found from the remote and update the refs
	// break when all oids have been found

	// 3. If we are still missing some we could check other sources with ngit. `git pull nostr-remote --mirror` and ignore pr/* but maybe we should leave that to a sync function
	return nil
}

// MirrorRepo mirrors a repository from one pubkey to another.
func MirrorRepo(fromPubkey string, toPubkey string, identifier string) error {
	gitPath, err := shared.GetGitPath()
	if err != nil {
		return fmt.Errorf("failed to get Git path: %w", err)
	}

	fromNpub, err := nip19.EncodePublicKey(fromPubkey)
	if err != nil {
		return fmt.Errorf("failed to encode from public key: %w", err)
	}

	toNpub, err := nip19.EncodePublicKey(toPubkey)
	if err != nil {
		return fmt.Errorf("failed to encode to public key: %w", err)
	}

	fromPath := shared.ConstructRepoPath(gitPath, fromNpub, identifier)
	toPath := shared.ConstructRepoPath(gitPath, toNpub, identifier)

	if err := AddRemote(toPath, fromPath, fromNpub); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Perform git pull --mirror from the remote
	cmd := exec.Command("git", "-C", toPath, "pull", "--mirror", fromNpub)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull from remote '%s': %w", fromNpub, err)
	}

	return nil
}

func AddRemote(repoDirPath string, remoteDirPath string, name string) error {
	// Change to the repository directory
	if err := exec.Command("git", "-C", repoDirPath, "remote", "get-url", name).Run(); err == nil {
		return nil
	}

	// Add the remote
	cmd := exec.Command("git", "-C", repoDirPath, "remote", "add", name, remoteDirPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}
	return nil
}
