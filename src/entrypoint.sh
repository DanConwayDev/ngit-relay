#!/bin/bash
set -e

# Make sure permissions are set correctly
mkdir -p /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db /var/log/ngit-relay
chown -R nginx:nginx /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db /var/log/ngit-relay
chmod -R 777 /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db /var/log/ngit-relay

# Start supervisord
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
