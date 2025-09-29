# AGENTS.md - ngit-relay

## 1) Quick project summary

ngit-relay is a Grasp implementation that bridges Git and Nostr protocols using NIP-34 (Nostr Git Repository Protocol). It's a self-hostable Git server with an embedded Nostr relay, enabling Git repositories to be managed through Nostr events. Built with Go 1.24.2, it uses khatru for the Nostr relay, BadgerDB for storage, and includes Git hooks for repository management.

## 2) Where to find context

- **Main source code**: [`src/ngit-relay-khatru/`](src/ngit-relay-khatru/) - Core Nostr relay and Git server
- **Git hooks**: [`src/ngit-relay-pre-receive/`](src/ngit-relay-pre-receive/), [`src/ngit-relay-post-receive/`](src/ngit-relay-post-receive/) - Repository hooks
- **Sync service**: [`src/ngit-relay-proactive-sync/`](src/ngit-relay-proactive-sync/) - Data synchronization
- **Shared utilities**: [`src/shared/`](src/shared/) - Common code and logging
- **Configuration**: [`docker-compose.yml`](docker-compose.yml), [`.env.example`](.env.example)
- **Documentation**: [`README.md`](README.md), [`DEPLOYMENT.md`](DEPLOYMENT.md)

## 3) Setup commands

```bash
# Development environment
nix develop

# Build Docker images
make build

# Start services
make up

# Stop services
make down

# Clean up
make clean

# Full upgrade
make upgrade
```

## 4) Typical agent workflow

1. **Development**: Use `nix develop` for reproducible Go environment
2. **Build**: Run `make build` to create Docker images
3. **Deploy**: Use `make up` to start all services
4. **Test**: Verify Git operations work through Nostr events
5. **Monitor**: Check logs via supervisord and nginx
6. **Update**: Use `make upgrade` for updates

## 5) Code style & conventions

- **Go modules**: Each component has its own go.mod in respective directories
- **Logging**: Use shared logger in [`src/shared/logger.go`](src/shared/logger.go)
- **Git hooks**: Follow standard pre-receive/post-receive patterns
- **Nostr events**: Implement NIP-34 Git Repository Protocol
- **Error handling**: Consistent error patterns across all components

## 6) Testing & CI

- **Manual testing**: Verify Git operations through Nostr interface
- **Integration**: Test end-to-end Git ↔ Nostr workflows
- **Docker**: All services containerized for consistent testing
- **Environment**: Use `.env.example` for test configuration

## 7) Security & data notes

- **Authentication**: Git operations require Nostr event signatures
- **Data storage**: BadgerDB for persistent storage
- **Network**: nginx reverse proxy for external access
- **SSL**: Optional SSL proxy available via [`docker-compose-ssl-proxy.yml`](docker-compose-ssl-proxy.yml)
- **Environment**: Secure sensitive data in `.env` files (not committed)

## 8) Large-change guidance

- **Modular design**: Each component (khatru, hooks, sync) is independent
- **Database**: BadgerDB allows for safe concurrent access
- **Git integration**: Changes to Git hooks require careful testing
- **Nostr protocol**: Follow NIP specifications for compatibility
- **Docker**: Update Docker images and compose files together

## 9) Localized AGENTS.md rules

- **Approval required**: Any changes to NIP-34 implementation require human review
- **Breaking changes**: Update version numbers and documentation for API changes
- **Dependencies**: Keep Go dependencies updated via `go mod tidy`
- **Docker**: Test all changes with `make up` before committing

## 10) Example commands & templates

```bash
# Quick development cycle
nix develop && make build && make up

# Check service status
docker compose ps

# View logs
docker compose logs -f ngit-relay-khatru

# Test Git operations
git clone http://localhost:8080/your-repo.git
```

**Environment variables to configure**:

- `NGIT_DOMAIN` - Domain name for the service
- `NGIT_RELAY_NAME` - Display name for the Nostr relay
- `NGIT_OWNER_NPUB` - Owner's Nostr public key
- `NGIT_PROACTIVE_SYNC_*` - Sync configuration
- `NGIT_BLOSSOM_*` - File upload limits
- `NGIT_LOG_*` - Logging configuration
