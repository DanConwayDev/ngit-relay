# Makefile for building and deploying the application

# Variables
BRANCH_NAME = master
# Prompt for SSL proxy inclusion
default_to_ssl := $(shell [ -f amce.json ] && echo "yes" || echo "no")
options := $(shell if [ "$(default_to_ssl)" = "yes" ]; then echo "Y/n"; else echo "y/N"; fi)
include_ssl_proxy := $(shell read -p "Do you want to include the SSL proxy? ($(options)) " confirm; \
	if [ "$confirm" = "y" ] || { [ -z "$confirm" ] && [ "$(default_to_ssl)" = "yes" ]; }; then \
		echo "-f docker-compose-ssl-proxy.yml"; \
	else \
		echo ""; \
	fi)

# Set COMPOSE_FILES based on user input
COMPOSE_FILES = -f docker-compose.yml $(include_ssl_proxy)

# Get the current commit ID
COMMIT_ID = $(shell git rev-parse --short HEAD)

# Default target
all: upgrade clean

# Build the Docker images with the commit ID
build:
	@echo "Building Docker images with commit ID: $(COMMIT_ID)"
	docker compose $(COMPOSE_FILES) build --build-arg VCS_REF=$(COMMIT_ID)

# Bring up the services --force-recreate ensure ssl-proxy gets updated
up:
	docker compose $(COMPOSE_FILES) up -d --force-recreate

# Stop and remove the services
down:
	docker compose $(COMPOSE_FILES) down

# Clean up unused images and containers
clean:
	docker system prune -f

# Upgrade target
upgrade: check-repo check-updates pull-images build up

# Check if the current directory is a Git repository
check-repo:
	@if [ ! -d ".git" ]; then \
		echo "Not a git repository. Exiting."; \
		exit 1; \
	fi

# Check for updates
check-updates:
	@echo "Checking for updates..."
	@git fetch origin
	@LOCAL=$(shell git rev-parse $(BRANCH_NAME)); \
	REMOTE=$(shell git rev-parse origin/$(BRANCH_NAME)); \
	if [ "$LOCAL" != "$REMOTE" ]; then \
		echo "Local branch '$(BRANCH_NAME)' is not up to date with 'origin/$(BRANCH_NAME)'. Pulling latest changes..."; \
		git pull origin $(BRANCH_NAME); \
	else \
		echo "Branch '$(BRANCH_NAME)' is up-to-date."; \
	fi

# Pull new images (only during upgrade)
pull-images:
	@echo "Pulling new images..."
	@if ! docker compose $(COMPOSE_FILES) pull; then \
		echo "Failed to pull images. Please check the logs."; \
		exit 1; \
	fi

