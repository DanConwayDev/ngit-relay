#!/bin/bash
set -e

# Make sure permissions are set correctly
mkdir -p /srv/repos /srv/blossom /relay-db 
chown -R nginx:nginx /srv/repos /srv/blossom /relay-db
chmod -R 777 /srv/repos /srv/blossom /relay-db

# Install /upgrade nostr-auth pre-receive hook for all existing repos
for repo in /srv/repos/*.git; do
  if [ -d "$repo/hooks" ]; then
    rm "$repo/hooks/pre-receive"
    cp /usr/local/bin/pre-receive "$repo/hooks/pre-receive"
    chmod +x "$repo/hooks/pre-receive"
  fi
done

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
