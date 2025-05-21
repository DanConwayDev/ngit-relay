package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"ngit-relay/shared"

	"go.uber.org/zap"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
	shared.Init("ngit-relay-post-receive")
	logger := shared.L()
	defer logger.Sync() // Ensure all logs are flushed on exit

	pubkey, identifier := shared.GetPubKeyAndIdentifierFromPath()
	if pubkey == "" {
		// Assuming GetPubKeyAndIdentifierFromPath now calls shared.L itself or we improve error handling
		os.Exit(1)
	}
	logger = logger.With(zap.String("repoIdentifier", identifier), zap.String("repoNpub", pubkey))

	ctx := context.Background()

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		logger.Error("Cannot fetch state events from relay", zap.Error(err))
		os.Exit(1)
	}

	// ensure our git repo is as up-to-date as possible with our nostr state
	// TODO: CheckStateMatchesNostrAndTryToRepair needs to be implemented and use the logger
	logger.Debug("Calling CheckStateMatchesNostrAndTryToRepair (currently a stub)")
	CheckStateMatchesNostrAndTryToRepair(events, pubkey, identifier)
	logger.Debug("Finished CheckStateMatchesNostrAndTryToRepair")

	// Update other git repos which use this pubkey as a maintainer
	// Identify the git repos
	otherRepoPubkeys := shared.GetOtherPubkeysThatUseThisPubkeyAsRepoMaintainer(events, pubkey)
	logger.Info("Found other repositories maintained by this pubkey", zap.Strings("otherRepoPubkeys", otherRepoPubkeys))

	// these probably should now inherit our maintainers state but we will check
	for _, otherRepoPubkey := range otherRepoPubkeys {
		otherLogger := logger.With(zap.String("otherRepoPubkey", otherRepoPubkey))
		state, err := shared.GetState(events, otherRepoPubkey, identifier)
		if err != nil {
			otherLogger.Error("Error getting nostr repository state event for following maintainer", zap.Error(err))
			continue
		}
		if state.Event.PubKey != pubkey {
			otherLogger.Info("Skipping mirroring to other repo; its state event pubkey does not match current repo's pubkey",
				zap.String("stateEventPubkey", state.Event.PubKey),
				zap.String("currentRepoPubkey", pubkey),
			)
			continue
		}
		otherLogger.Info("Attempting to mirror repository to following maintainer's repo")
		if err := MirrorRepo(logger, pubkey, otherRepoPubkey, identifier); err != nil {
			otherLogger.Error("Error updating git repository for following maintainer", zap.Error(err))
			continue
		}
		otherLogger.Info("Successfully updated git repository for following maintainer")
	}

	logger.Info("Post-receive hook completed successfully.")
	os.Exit(0)
}

func CheckStateMatchesNostrAndTryToRepair(events []nostr.Event, pubkey string, identifier string) error {
	// TODO: This function needs to be implemented.
	// It should use the logger passed to it or acquire one via shared.L().
	// For example:
	// logger := shared.L().With(zap.String("fn", "CheckStateMatchesNostrAndTryToRepair"))
	// logger.Info("Checking nostr state against local repo...")
	//
	// state, err := shared.GetState(events, pubkey, identifier)
	// if err != nil {
	// 	logger.Error("Error getting nostr repository state event for pubkey", zap.String("pubkeyForState", pubkey), zap.Error(err))
	// 	os.Exit(1) // Or return error
	// }
	// logger.Debug("State fetched", zap.Any("state", state))

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
func MirrorRepo(logger *zap.Logger, fromPubkey string, toPubkey string, identifier string) error {
	taskLogger := logger.With(
		zap.String("fromPubkey", fromPubkey),
		zap.String("toPubkey", toPubkey),
		zap.String("identifier", identifier),
	)
	taskLogger.Info("Starting repository mirror operation")

	gitPath, err := shared.GetGitPath()
	if err != nil {
		taskLogger.Error("Failed to get Git path", zap.Error(err))
		return fmt.Errorf("failed to get Git path: %w", err)
	}
	taskLogger.Debug("Got git path", zap.String("gitPath", gitPath))

	fromNpub, err := nip19.EncodePublicKey(fromPubkey)
	if err != nil {
		taskLogger.Error("Failed to encode 'from' public key to npub", zap.Error(err))
		return fmt.Errorf("failed to encode from public key: %w", err)
	}

	toNpub, err := nip19.EncodePublicKey(toPubkey)
	if err != nil {
		taskLogger.Error("Failed to encode 'to' public key to npub", zap.Error(err))
		return fmt.Errorf("failed to encode to public key: %w", err)
	}
	taskLogger.Debug("Encoded npubs", zap.String("fromNpub", fromNpub), zap.String("toNpub", toNpub))

	fromPath := shared.ConstructRepoPath(gitPath, fromNpub, identifier)
	toPath := shared.ConstructRepoPath(gitPath, toNpub, identifier)
	taskLogger.Debug("Constructed repo paths", zap.String("fromPath", fromPath), zap.String("toPath", toPath))

	if err := AddRemote(taskLogger, toPath, fromPath, fromNpub); err != nil {
		// AddRemote logs its own errors
		return fmt.Errorf("failed to add remote: %w", err)
	}

	taskLogger.Info("Performing git pull --mirror", zap.String("remoteName", fromNpub), zap.String("targetRepoPath", toPath))
	// Perform git pull --mirror from the remote
	cmd := exec.Command("git", "-C", toPath, "pull", "--mirror", fromNpub)
	output, err := cmd.CombinedOutput() // Capture output for logging
	if err != nil {
		taskLogger.Error("Failed to pull from remote",
			zap.String("remoteName", fromNpub),
			zap.ByteString("output", output),
			zap.Error(err),
		)
		return fmt.Errorf("failed to pull from remote '%s': %w. Output: %s", fromNpub, err, string(output))
	}
	taskLogger.Info("Successfully pulled from remote", zap.String("remoteName", fromNpub), zap.ByteString("output", output))

	return nil
}

func AddRemote(logger *zap.Logger, repoDirPath string, remoteDirPath string, name string) error {
	taskLogger := logger.With(
		zap.String("repoDirPath", repoDirPath),
		zap.String("remoteDirPath", remoteDirPath),
		zap.String("remoteName", name),
	)
	taskLogger.Debug("Checking if remote exists")

	// Check if remote already exists
	if err := exec.Command("git", "-C", repoDirPath, "remote", "get-url", name).Run(); err == nil {
		taskLogger.Info("Remote already exists, skipping add.", zap.String("remoteName", name))
		return nil
	}

	taskLogger.Info("Adding remote", zap.String("remoteName", name), zap.String("remotePath", remoteDirPath))
	// Add the remote
	cmd := exec.Command("git", "-C", repoDirPath, "remote", "add", name, remoteDirPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		taskLogger.Error("Failed to add remote", zap.ByteString("output", output), zap.Error(err))
		return fmt.Errorf("failed to add remote '%s': %w. Output: %s", name, err, string(output))
	}
	taskLogger.Info("Successfully added remote", zap.ByteString("output", output))
	return nil
}
