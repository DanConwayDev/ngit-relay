package main

import (
	"context"
	"ngit-relay/shared"
	"os"
	"os/exec"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
)

func EventReceiveHook(git_data_path string) func(ctx context.Context, event *nostr.Event) {
	return func(ctx context.Context, event *nostr.Event) {
		// process event in a routine so we don't delay notifiying the user that the event was saved
		go processEvent(context.Background(), event, git_data_path)
	}
}

func processEvent(ctx context.Context, event *nostr.Event, git_data_path string) {
	// create empty git repo for new announcement events
	if event.Kind == nostr.KindRepositoryAnnouncement {
		// If you are looking enforcement that announcement events list this ngit instance, look in policies
		logger := shared.L().With(zap.String("type", "RepoAnnEventReceiveHook"), zap.String("eventjson", event.String()))
		logger.Debug("repo annoucement received")

		npub, _ := nip19.EncodePublicKey(event.PubKey)
		d_tags := event.Tags.Find("d")
		if d_tags == nil {
			logger.Error("repo announcement missing d tag")
			return
		}
		identifier := event.Tags.Find("d")[1]
		repo_path := git_data_path + "/" + npub + "/" + identifier + ".git"

		logger = logger.With(zap.String("repo_path", repo_path))

		if _, err := os.Stat(repo_path); err == nil {
			logger.Debug("git repo dir already exists for annoucement")
		} else if !os.IsNotExist(err) {
			logger.Error("Error checking repo path:", zap.Error(err))
		} else {
			// repo doesn't exist
			logger.Debug("Creating empty git repo")

			// create directory
			err := os.MkdirAll(repo_path, os.ModePerm)
			if err != nil {
				logger.Error("Error creating directory:", zap.Error(err))
				return
			}

			// init git repo
			cmd := exec.Command("git", "init", "--bare", repo_path)
			cmd.Dir = git_data_path // Set the working directory for the command
			_, err = cmd.Output()
			if err != nil {
				logger.Error("Error initializing Git repository:", zap.Error(err))
				return
			}

			// allow unauthenticated push (we handle write permissions via pre-receive git hook)
			cmd = exec.Command("git", "config", "http.receivepack", "true")
			cmd.Dir = repo_path // Set the working directory for the command
			_, err = cmd.Output()
			if err != nil {
				logger.Error("Error configuring Git to enable push:", zap.Error(err))
				return
			}

			// allow uploadpack from tips - required for ngit remote helper to pull desired data
			// without it ngit won't be able to fetch, pull or clone.
			cmd = exec.Command("git", "config", "uploadpack.allowTipSHA1InWant", "true")
			cmd.Dir = repo_path // Set the working directory for the command
			_, err = cmd.Output()
			if err != nil {
				logger.Error("Error configuring Git to enable uploadpakc.allowTipSHA1InWant", zap.Error(err))
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
				logger.Error("Error configuring Git to enable uploadpakc.allowUnreachable", zap.Error(err))
				return
			}

			// Create symlink to pre-receive hook
			err = os.Symlink("/usr/local/bin/ngit-relay-pre-receive", repo_path+"/hooks/pre-receive")
			if err != nil {
				logger.Error("Error creating symlink", zap.Error(err))
				return
			}

			// set permissions
			err = os.Chmod(repo_path, 0777)
			if err != nil {
				logger.Error("Error changing permissions", zap.Error(err))
				return
			}

			// ensure correct ownership for git-http-backend
			cmd = exec.Command("chown", "-R", "nginx:nginx", repo_path)
			_, err = cmd.Output()
			if err != nil {
				logger.Error("Error changing ownership", zap.Error(err))
				return
			}

			logger.Info("Created empty git repo for " + npub + "/" + identifier + ".git")

			// sync git repository (useful if an existing repository just added this ngit instance)
			time.Sleep(1 * time.Second) // wait for state event to be processed, if sent with announcement
			err = shared.ProactiveSyncGit(event.PubKey, identifier, git_data_path)
			if err != nil {
				logger.Debug("ProactiveSyncGit on creation error, usually because 1. its a fresh repo or 2. our relay doesnt have the state event yet", zap.Error(err))
				return
			} else {
				logger.Debug("ProactiveSyncGit on creation completed without error")
			}
		}
	}
	// proactive sync git repos
	if event.Kind == nostr.KindRepositoryState && shared.GetEnvBool("NGIT_PROACTIVE_SYNC_GIT", true) {
		logger := shared.L().With(
			zap.String("type", "RepoAnnEventReceiveHook"),
			zap.String("eventjson", event.String()),
		)

		identifier := event.Tags.Find("d")[1]

		// Wait for 60 seconds before processing - this allows time for ngit-cli clients to push git data to git servers
		time.Sleep(60 * time.Second)

		events, err := shared.FetchAnnouncementAndStateEventsFromRelay(ctx, identifier)
		if err != nil {
			logger.Error("FetchAnnouncementAndStateEventsFromRelay failed during KindRepositoryState path", zap.Error(err))
			return
		}

		processed := make([]string, 0) // Initialize processed as a slice of strings
		for _, e := range events {
			if e.Kind == nostr.KindRepositoryAnnouncement && !contains(processed, e.PubKey) {
				if err := shared.ProactiveSyncGit(e.PubKey, identifier, git_data_path); err != nil {
					logger.Error("ProactiveSyncGit failed", zap.String("pubKey", e.PubKey), zap.Error(err))
					return
				}
				processed = append(processed, e.PubKey) // Add the processed PubKey to the list
			}
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
