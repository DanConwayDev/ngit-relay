package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func EventRecieveHook(git_data_path string) func(ctx context.Context, event *nostr.Event) {
	return func(ctx context.Context, event *nostr.Event) {
		if event.Kind == nostr.KindRepositoryAnnouncement {
			npub, _ := nip19.EncodePublicKey(event.PubKey)
			identifier := event.Tags.Find("d")[1]
			repo_path := git_data_path + "/" + npub + "/" + identifier + ".git"

			if _, err := os.Stat(repo_path); err == nil {
				// repo directory exists
			} else if !os.IsNotExist(err) {
				fmt.Println("Error checking repo path:", err)
			} else {
				// repo doesn't exist
				fmt.Println("Creating empty git repo for ", npub+"/"+identifier)

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
		}
	}
}
