#!/bin/bash
set -e

# Setup SSL smbolic links for nginx to use

: "${CERT_DOMAIN?Need CERT_DOMAIN in .env}"

mkdir -p /etc/nginx/certs
ln -sf "/etc/letsencrypt/live/${CERT_DOMAIN}/fullchain.pem" /etc/nginx/certs/fullchain.pem
ln -sf "/etc/letsencrypt/live/${CERT_DOMAIN}/privkey.pem"   /etc/nginx/certs/privkey.pem

# Make sure permissions are set correctly
mkdir -p /srv/repos /srv/blossom /khatru-data 
chown -R nginx:nginx /srv/repos /srv/blossom /khatru-data
chmod -R 777 /srv/repos /srv/blossom /khatru-data


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
