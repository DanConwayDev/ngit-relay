#!/bin/bash
set -e

# establish locations of SSL certs
: "${CERT_DOMAIN?Need CERT_DOMAIN in .env}"
live=/etc/letsencrypt/live/${CERT_DOMAIN}
mkdir -p "$live" /etc/nginx/certs

# Check if fullchain.pem and privkey.pem exist in the live location
if [ ! -f "$live/fullchain.pem" ] || [ ! -f "$live/privkey.pem" ]; then
  echo "Generating temporary self-signed certificate for $CERT_DOMAIN …"
  temp_certs=/tmp/certs  # Temporary location for self-signed certs
  mkdir -p "$live" /etc/nginx/certs "$temp_certs"  # Create necessary directories
  # Generate self-signed certificates in the temporary location
  openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
          -keyout  "$temp_certs/privkey.pem" \
          -out     "$temp_certs/fullchain.pem" \
          -subj    "/CN=$CERT_DOMAIN"
  # Create symbolic links to the temporary certificates
  ln -sf "$temp_certs/fullchain.pem" /etc/nginx/certs/fullchain.pem
  ln -sf "$temp_certs/privkey.pem"   /etc/nginx/certs/privkey.pem
else
  echo "Using existing certificates from live location."
  # Create symbolic links to the live certificates
  ln -sf "$live/fullchain.pem" /etc/nginx/certs/fullchain.pem
  ln -sf "$live/privkey.pem"   /etc/nginx/certs/privkey.pem
fi

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
