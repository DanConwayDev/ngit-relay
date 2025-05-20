#!/bin/bash
set -e

# Make sure permissions are set correctly
mkdir -p /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db 
chown -R nginx:nginx /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db
chmod -R 777 /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db

# Install /upgrade nostr-auth pre-receive hook for all existing repos
for repo in /srv/ngit-relay/repos/*.git; do
  if [ -d "$repo/hooks" ]; then
    ## TODO - this should be a symlink
    rm "$repo/hooks/pre-receive"
    cp /usr/local/bin/ngit-relay-pre-receive "$repo/hooks/pre-receive"
    chmod +x "$repo/hooks/pre-receive"
    rm "$repo/hooks/post-receive"
    cp /usr/local/bin/ngit-relay-post-receive "$repo/hooks/post-receive"
    chmod +x "$repo/hooks/post-receive"
  fi
done

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
