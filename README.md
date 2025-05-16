# ngit-relay

Nostr-permissioned Git server with bundled a nostr relay and a blossom server. A complete hosting solution for a nostr git repository.

Easy to self-host or on a VPS through docker. See DEPLOYMENT.md.

## Why 

Git was originally designed to work with an open communications protocol; email. Github created a more popular collaboration experieince within their own platform.

Protocols > Platforms. With nostr we have the opportuntiy to bring software collaboration back to the realm of open communications protocols.

Git Via Nostr tries to connect git and nostr as simiply as possible.

## What

ngit-relay is a wrapper:
. git-http-backend - the git server reference implementation
. a nostr relay (using khatru) - for storing events related to git repositories it has accepted
. a blossom server (using khatru) - for storing images, videos and files referenced in events on the relay

This is everything you need to store all data related to Git Via Nostr.

Only data related to nostr git repositories get stored. Here is how:

 > git repositories get automically provisioned when the relay receives a nostr git repository or state annocunement.
 > the git-http-backend uses a pre-receive git hook that only accepts pushes that match the latest maintainer nostr git repository state announcement event on the relay.
 > the relay only accept git announcements events and events that reference, or are referenced by other events on the relay.
 > the blossom server stores blobs for 24h and then deletes them if they are not referenced in an accepted event (deletion not implemented yet).

A whitelist could be easily added for new repositories but its nice to give back and host other peoles FOSS projects.

Right now this only support public repositories.

## How To Use It

1. **Self-host or deploy** it to a VPs (See DEPLOYMENT.md) or use a public instance eg. gitnostr.com

2. **create local git repo**
```
git init
echo project > README.md
git add . && git commit -m "initial commit"
```
3. **install ngit** - download and install the nostr git plugin called ngit [gitworkshop.dev/ngit](https://gitworkshop.dev/ngit)

4. `ngit init` and follow the instructions.
when prompted for some relays include one or more ngit-relay instances eg. `wss://gitnostr.com` or your self-hosted version.
when prompted for git server(s) include one or more ngit-relays but use the format `http://domain.com/<npub123>/<identifier>.git` eg `https://gitnostr.com/npub15qydau2hjma6ngxkl2cyar74wzyjshvl65za5k5rl69264ar2exs5cyejr/ngit-relay.git`. If you include other (non ngit-relay) git servers, you will need provision the repositories manually and make sure you are setup to authenticate to the eg over ssh.
this will send an git announcement nostr event to the nostr relay on the ngit-relay instance which will provision git repository ready to receive your push.

5. set remote origin to the repositories nostr address `git remote set-url origin nostr://npub123/my-repo`

6. `git push` - as you are pushing to a `nostr://` remote, the nostr git plugin ngit will:
    - issue a git state nostr event with your new state to your repository relays (including the ngit-relay instance(s)); then
    - push the git state to each git server listed in the announcement event. Instead of doing user authentication over ssh, the git servers within the ngit-relay instances will do no user authentication but rather check that the pushed ref matches the state event it just received via a pre-receive git hook and proceed on that basis.
    - you can list any other git servers you might use eg github or codeberg.

See the gitworkshop.dev quick start guide for more info.

[gitworkshop.dev/quick-start](https://gitworkshop.dev/quick-start)

## FAQ

**Why support pushing to multiple git servers?**

[TODO]

**Why not provide a web-frontend for viewing code, etc**

[TODO]
