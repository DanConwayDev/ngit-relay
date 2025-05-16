# ngit-relay

**Nostr-permissioned Git server** with a bundled Nostr relay and a Blossom server. A complete hosting solution for a Nostr Git repository.

Easily self-host or deploy on a VPS using Docker. See [DEPLOYMENT.md](DEPLOYMENT.md) for more details.

## Why

Git was originally designed to work with an open communications protocol: email. GitHub created a more popular collaboration experience within their own platform.

**Protocols > Platforms.** With Nostr, we have the opportunity to bring software collaboration back to the realm of open communications protocols.

**Git Via Nostr** aims to connect Git and Nostr as simply as possible.

## What

`ngit-relay` is a wrapper that includes:
- **git-http-backend**: The reference implementation of the Git server.
- **A Nostr relay (using Khatru)**: For storing events related to Git repositories it has accepted.
- **A Blossom server (using Khatru)**: For storing images, videos, and files referenced in events on the relay.

This setup provides everything you need to store all data related to Git Via Nostr.

Only data related to Nostr Git repositories is stored. Here’s how it works:

- Git repositories are automatically provisioned when the relay receives a Nostr Git repository or state announcement.
- The `git-http-backend` uses a pre-receive Git hook that only accepts pushes matching the latest maintainer Nostr Git repository state announcement event on the relay.
- The relay only accepts Git announcement events and events that reference or are referenced by other events on the relay.
- The Blossom server stores blobs for 24 hours and then deletes them if they are not referenced in an accepted event (deletion not implemented yet).

A whitelist could be easily added for new repositories, but it's nice to give back and host other people's FOSS projects.

Currently, this only supports public repositories.

## How To Use It

1. **Self-host or deploy** it to a VPS (see [DEPLOYMENT.md](DEPLOYMENT.md)) or use a public instance, e.g., `gitnostr.com`.

2. **Create a local Git repository**:
   ```bash
   git init
   echo project > README.md
   git add . && git commit -m "initial commit"
   ```

3. **Install ngit**: Download and install the Nostr Git plugin called ngit from [gitworkshop.dev/ngit](https://gitworkshop.dev/ngit).

4. Run `ngit init` and follow the instructions. When prompted for relays, include one or more ngit-relay instances (e.g., `wss://gitnostr.com` or your self-hosted version). When prompted for Git server(s), include one or more ngit-relays in the format `http://domain.com/<npub123>/<identifier>.git` (e.g., `https://gitnostr.com/npub15qydau2hjma6ngxkl2cyar74wzyjshvl65za5k5rl69264ar2exs5cyejr/ngit-relay.git`). If you include other (non-ngit-relay) Git servers, you will need to provision the repositories manually and ensure you are set up to authenticate (e.g., over SSH). This will send a Git announcement Nostr event to the Nostr relay on the ngit-relay instance, provisioning the Git repository to receive your push.

5. Set the remote origin to the repository's Nostr address:
   ```bash
   git remote set-url origin nostr://npub123/my-repo
   ```

6. Run `git push`. As you are pushing to a `nostr://` remote, the Nostr Git plugin ngit will:
   - Issue a Git state Nostr event with your new state to your repository relays (including the ngit-relay instance(s)).
   - Push the Git state to each Git server listed in the announcement event. Instead of user authentication over SSH, the Git servers within the ngit-relay instances will not require user authentication but will check that the pushed ref matches the state event it just received via a pre-receive Git hook and proceed accordingly.
   - You can list any other Git servers you might use, such as GitHub or Codeberg.

See the [Git Workshop Quick Start Guide](https://gitworkshop.dev/quick-start) for more information.

## FAQ

**Why support pushing to multiple Git servers?**

[TODO]

**Why not provide a web frontend for viewing code, etc.?**

[TODO]
