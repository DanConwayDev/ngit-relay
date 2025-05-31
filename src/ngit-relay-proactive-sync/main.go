package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ngit-relay/shared"

	"go.uber.org/zap"
)

func main() {
	shared.Init("ngit-relay-proactive-sync", true, false)
	logger := shared.L()

	git_data_path := flag.String("git-data-dir", "", "Directory for repositories data")
	sync_interval := flag.Int("sync-interval", 15, "minutes between each proactive sync")
	flag.Parse()
	if *git_data_path == "" {
		flag.Usage()
		logger.Fatal("relay-data-dir, git-data-dir and blossom_data_path are required CLI arguments.")
	}

	// Wait 20s for warmup then run SyncRepos
	logger.Info("Waiting 20 seconds for warmup before starting sync", zap.Int("sync_interval", *sync_interval))
	time.Sleep(20 * time.Second)

	// Run SyncRepos every sync_interval minutes
	// If SyncRepos takes longer than sync_interval, run it again when it finishes
	for {
		startTime := time.Now()
		logger.Info("starting sync")
		SyncRepos(*git_data_path, logger)

		// Calculate time elapsed and sleep for the remainder of the interval
		elapsed := time.Since(startTime)
		interval := time.Duration(*sync_interval) * time.Minute

		if elapsed < interval {
			sleepTime := interval - elapsed
			logger.Info("Sync completed, waiting for next sync",
				zap.Duration("sleep_time", sleepTime),
				zap.Duration("elapsed", elapsed))
			time.Sleep(sleepTime)
		} else {
			// If sync took longer than interval, run again immediately
			logger.Info("Sync took longer than interval, running again immediately",
				zap.Duration("elapsed", elapsed),
				zap.Duration("interval", interval))
		}
	}
}

func SyncRepos(git_data_path string, logger *zap.Logger) {
	// git_data_path has a structure of [git_data_path]/npub123/repo.git

	// First, list all npub directories
	npubDirs, err := os.ReadDir(git_data_path)
	if err != nil {
		logger.Error("Failed to read git data directory", zap.String("path", git_data_path), zap.Error(err))
		return
	}

	// For each npub directory
	for _, npubDir := range npubDirs {
		if !npubDir.IsDir() || !strings.HasPrefix(npubDir.Name(), "npub") {
			continue // Skip if not a directory or not an npub directory
		}

		npubPath := filepath.Join(git_data_path, npubDir.Name())

		// List all repo.git directories for this npub
		repoDirs, err := os.ReadDir(npubPath)
		if err != nil {
			logger.Error("Failed to read npub directory", zap.String("path", npubPath), zap.Error(err))
			continue
		}

		// For each repo.git directory
		for _, repoDir := range repoDirs {
			if !repoDir.IsDir() || !strings.HasSuffix(repoDir.Name(), ".git") {
				continue // Skip if not a directory or not a git repository
			}

			repoPath := filepath.Join(npubPath, repoDir.Name())
			logger.Debug("Syncing repository", zap.String("repo_path", repoPath))

			err := SyncRepo(git_data_path, repoPath)
			if err != nil {
				logger.Error("Failed to sync repository",
					zap.String("repo_path", repoPath),
					zap.Error(err))
			}
		}
	}
}

func SyncRepo(git_data_path string, repoPath string) error {
	pubkey, _, identifier, err := shared.GetPubKeyAndIdentifierFromPath(repoPath)
	if err != nil {
		return err
	}
	return shared.ProactiveSyncGit(pubkey, identifier, git_data_path)
}
