#!/bin/bash
set -e

# establish locations of SSL certs
: "${CERT_DOMAIN?Need CERT_DOMAIN in .env}"
live=/etc/letsencrypt/live/${CERT_DOMAIN}
mkdir -p "$live" /etc/nginx/certs

# If fullchain.pem is missing, generate a 1-day self-signed pair
if [ ! -f "$live/fullchain.pem" ] || [ ! -f "$live/privkey.pem" ]; then
  echo "Generating temporary self-signed certificate for $CERT_DOMAIN …"
  openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
          -keyout  "$live/privkey.pem" \
          -out     "$live/fullchain.pem" \
          -subj    "/CN=$CERT_DOMAIN"
fi

# Create the stable symlinks that nginx.conf expects
ln -sf "$live/fullchain.pem" /etc/nginx/certs/fullchain.pem
ln -sf "$live/privkey.pem"   /etc/nginx/certs/privkey.pem

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
