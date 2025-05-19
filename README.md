# ngit-relay

**Nostr-permissioned Git server** bundled with a Nostr relay and a Blossom server. A complete hosting solution for a Nostr Git repository.

Easily self-host or deploy on a VPS using Docker. See [DEPLOYMENT.md](DEPLOYMENT.md) for more details.

## Why

Git was originally designed to work with an open communications protocol: email. GitHub created a more popular collaboration experience within their own platform.

**Protocols > Platforms.** With Nostr, we have the opportunity to bring software collaboration back to the realm of open communications protocols.

**Git Via Nostr** aims to connect Git and Nostr as simply as possible with [NIP-34](https://nips.nostr.com/34).

## What

`ngit-relay` is a wrapper that includes:

- **git-http-backend**: The reference implementation of the Git server.
  - **pre-receive Git hook**: To apply Nostr-based write permissions.
- **A Nostr relay (using Khatru)**: For storing events related to Git repositories it has accepted.
  - **event receive hook**: To create new Git repositories when [Git repository announcements](https://nips.nostr.com/34#repository-announcements) are received.
- **A Blossom server (using Khatru)**: For storing images, videos, and files referenced in events on the relay.

This setup provides everything you need to store all data related to Git Via Nostr.

Only data related to Nostr Git repositories is stored. Here’s how it works:

- Git repositories are automatically provisioned when the relay receives a Nostr [Git repository announcement](https://nips.nostr.com/34#repository-announcements) that lists the ngit-relay instance under 'clones' and 'relays'.
- The `git-http-backend` uses a pre-receive Git hook that only accepts pushes matching the latest maintainer Nostr Git repository [state announcement](https://nips.nostr.com/34#repository-state-announcements) event on the relay.
- The relay only accepts [Git repository announcements](https://nips.nostr.com/34#repository-announcements) events that list the ngit-relay instance and events that reference or are referenced by other events on the relay.
- The Blossom server stores blobs for 24 hours and then deletes them if they are not referenced in an accepted event (deletion not implemented yet).

A whitelist could be easily added for new repositories, but it's nice to give back and host other people's FOSS projects if they explicitly choose to use your instance. It might help limit centralization on a few public instances if everyone does this.

## How To Use It

1. **Self-host or deploy** it to a VPS (see [DEPLOYMENT.md](DEPLOYMENT.md)) or use a public instance, e.g., `gitnostr.com`.

2. **Create a local Git repository**:

   ```bash
   git init
   echo project > README.md
   git add . && git commit -m "initial commit"
   ```

3. **Install ngit**: Download and install the Nostr Git plugin called ngit from [gitworkshop.dev/ngit](https://gitworkshop.dev/ngit).

4. Run `ngit init` and follow the instructions. When prompted for relays, include one or more ngit-relay instances (e.g., `wss://gitnostr.com` or your self-hosted version) - see FAQ 'Why Use Multiple ngit Instances'. When prompted for Git server(s), include one or more ngit-relays in the format `http://domain.com/<npub123>/<identifier>.git` (e.g., `https://gitnostr.com/npub15qydau2hjma6ngxkl2cyar74wzyjshvl65za5k5rl69264ar2exs5cyejr/ngit-relay.git`). This will send a Git Announcement Nostr event to the Nostr relay on the ngit-relay instance, provisioning the Git repository to receive your push.

   Note: If you include other (non-ngit-relay) Git servers such as GitHub, you will need to provision the repositories manually through their custom interface and ensure you are set up to authenticate (e.g., over SSH).

5. Set the remote origin to the repository's Nostr address:

   ```bash
   git remote set-url origin nostr://npub123/my-repo
   ```

6. Run `git push`. As you are pushing to a `nostr://` remote, the Nostr Git plugin ngit will:
   - Issue a Git state Nostr event with your new state to your repository relays (including the ngit-relay instance(s)).
   - Push the Git state to each Git server listed in the [announcement](https://nips.nostr.com/34#repository-announcements) event. Instead of user authentication over SSH, the Git servers within the ngit-relay instances will not require user authentication but will check that the pushed ref matches the state event it just received via a pre-receive Git hook and proceed accordingly.
   - You can list any other Git servers you might use, such as GitHub or Codeberg.

See the [Git Workshop Quick Start Guide](https://gitworkshop.dev/quick-start) for more information.

## Out of Scope

**A User Interface, e.g., cgit.** Users only interact with Nostr relays over the Nostr protocol through Nostr clients. This comes with lots of benefits, e.g., changing relays easily. It should be the same for Git servers. Quickly browsing repository code on the web or sharing a permalink to some line of a commit is really useful. Someone should write a cgit-like web client for this that uses the Git protocol. It shouldn't be baked into Git servers. Protocols > platforms.

**SSH / non-Nostr-based permissions.** See 'When Private Repos?' in the FAQ section.

## Features / Roadmap

- [x] Standard smart HTTP Git server
- [x] Pre-receive Git hook to apply Nostr write permissions via NIP-34
- [ ] Multi-maintainer support
- [x] Nostr relay
- [x] Relay event hook to provision new Git repositories when [Git repository announcements](https://nips.nostr.com/34#repository-announcements) are received
- [x] Blossom server
- [ ] Blossom server only retains blobs referenced in Nostr events on the relay
- [ ] Sync - proactively fetch Nostr events related to stored repositories to back them up.
- [ ] Blossom sync - proactively fetch Blossom blobs referenced in accepted events to ensure they don't get lost.
- [ ] Spam prevention (e.g., via Vercel or a similar service run locally)
- [ ] Announcements - make it easy for users to find available ngit-relay instances to use via announcements on Nostr, including terms of service and pricing if appropriate.
- [ ] Whitelist - for other specific repositories
- [ ] Archive relay - create and serve a backup of repositories that don't list your ngit-relay instance.

## FAQ

### When Private Repos?

Semi-private instances could be easily added with auth-to-read (via [NIP-42](https://nips.nostr.com/42)) and a whitelist. Users would need to use Nostr clients that support protected events ([NIP-70](https://nips.nostr.com/70)) to prevent discussions spilling over onto public relays.

This would be perfect for early-stage FOSS projects that aren't ready to go public yet but are not ideal for proprietary software development.

To prevent attackers from guessing the Git repository endpoint (Git server with npub identifier), the instance could be behind a firewall.

A more robust but involved and complex option would be to design and implement a Nostr-based HTTP authentication scheme to [NIP-42](https://nips.nostr.com/42).

## Why Use Multiple ngit-relay Instances?

Users have high expectations of Git servers. GitHub is performant, reliable, and has near 100% uptime. A self-hosted Git server can't achieve that level of performance.

We can add resilience, reliability, and uptime by treating Git servers like Nostr relays if we pull and push from multiple instances simultaneously. This way, users can achieve comparable reliability and uptime, even if some of the instances they use are occasionally down or misbehaving.

This approach prevents us from centralizing around a few performant Git servers and also allows maintainers to change Git servers without requiring their users to make any configuration changes.

To make this vision a reality, we need to encourage contributors and users to clone using the `nostr://` URL with the ngit plugin installed, rather than the `https://` URL of a single ngit-relay instance.
