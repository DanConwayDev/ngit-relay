package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"ngit-relay/shared"
)

func main() {
	shared.Init("ngit-relay-post-receive", false, false)
	logger := shared.L()

	pubkey, npub, identifier, err := shared.GetPubKeyAndIdentifierFromPath()
	if err != nil {
		logger.Fatal("cannot extract repo pubkey and identifier from path", zap.Error(err))
	}

	logger = logger.With(zap.String("identifier", identifier), zap.String("npub", npub))
	ctx := context.Background()

	repo_path, err := os.Getwd()
	if err != nil {
		logger.Fatal("cannot get current working directory", zap.Error(err))
	}

	events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
	if err != nil {
		logger.Fatal("cannot FetchAnnouncementAndStateEventsFromRelay", zap.Error(err))
	}

	maintainers := shared.GetMaintainers(events, pubkey, identifier)
	if len(maintainers) == 0 {
		logger.Fatal("cannot GetMaintainers", zap.Error(err))
	}

	state, err := shared.GetStateFromMaintainers(events, maintainers)
	if err != nil {
		logger.Fatal("cannot GetStateFromMaintainers", zap.Error(err))
	}

	err = shared.ProactiveSyncGitFromStateAndServers(state, []string{}, repo_path)
	if err != nil {
		logger.Debug("ProactiveSyncGitFromStateAndServers not successful", zap.Error(err))
	}

	logger.Debug("post-receive hook completed successfully.")
	// If no issues, exit with success
	os.Exit(0)
}
