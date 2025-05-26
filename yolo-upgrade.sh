#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Navigate to the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" || exit

# Define the branch name
BRANCH_NAME="master"

# Check if the script is being called for the second time
if [ "$1" != "--upgraded" ]; then
    # Check if the current directory is a Git repository
    if [ ! -d ".git" ]; then
        echo "Not a git repository. Exiting."
        exit 1
    fi

    # Fetch the latest changes from the remote repository
    git fetch origin

    # Check if origin is ahead of main
    LOCAL=$(git rev-parse "$BRANCH_NAME")
    REMOTE=$(git rev-parse "origin/$BRANCH_NAME")
    
    if [ "$LOCAL" != "$REMOTE" ]; then
        echo "The local branch '$BRANCH_NAME' is not up to date with 'origin/$BRANCH_NAME'."
        echo "Local commit: $LOCAL"
        echo "Remote commit: $REMOTE"
        
        # Optionally, you can check if the remote is ahead
        if git rev-list --count "$LOCAL..$REMOTE" > /dev/null; then
            echo "'origin/$BRANCH_NAME' is ahead of '$BRANCH_NAME'."
            # Pull the latest changes from the repository
            git pull origin "$BRANCH_NAME"
            echo "Successfully pulled the latest changes."
        else
            echo "'$BRANCH_NAME' is ahead of 'origin/$BRANCH_NAME'. No need to pull."
        fi
    else
        echo "up-to-date"
    fi

    # Run the script again with the --upgraded flag
    ./yolo-upgrade.sh --upgraded
    exit 0
else
    # This block runs only if the script is called with the --upgraded flag
    echo "Running docker compose to pull new images, rebuild and deploy the upgrade..."
    if ! sudo docker compose -f docker-compose.yml -f docker-compose-ssl-proxy.yml up -d --force-recreate --pull; then
        echo "Failed to run docker compose up --build -d. Please check the logs."
        exit 1
    fi
fi
