#!/bin/bash
set -e

# Make sure permissions are set correctly
mkdir -p /srv/repos /khatru-data
chown -R nginx:nginx /srv/repos /khatru-data
chmod -R 777 /srv/repos /khatru-data


# Install nostr-auth pre-receive hook for all existing repos
for repo in /srv/repos/*.git; do
  if [ -d "$repo/hooks" ]; then
    rm "$repo/hooks/pre-receive"
    cp /usr/local/bin/pre-receive "$repo/hooks/pre-receive"
    chmod +x "$repo/hooks/pre-receive"
  fi
done

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
