#!/bin/bash
set -e

# Setup SSL smbolic links for nginx to use

# Check if SSL_CERT_PATH or SSL_KEY_PATH is empty or set to the default values
if [[ -z "$SSL_CERT_PATH" || "$SSL_CERT_PATH" == "/etc/letsencrypt/live/domain.com/fullchain.pem" ]]; then
    echo "Error: SSL_CERT_PATH is not set or is set to the default value."
    echo "Please set SSL_CERT_PATH to a valid certificate path in your .env file."
    exit 1
fi
if [[ -z "$SSL_KEY_PATH" || "$SSL_KEY_PATH" == "/etc/letsencrypt/live/domain.com/privkey.pem" ]]; then
    echo "Error: SSL_KEY_PATH is not set or is set to the default value."
    echo "Please set SSL_KEY_PATH to a valid key path in your .env file."
    exit 1
fi
# Create the directory for SSL certificates if it doesn't exist
mkdir -p /etc/nginx/certs
# Create symbolic links for Nginx to use, if they don't already exist
if [ ! -L /etc/nginx/certs/cert.pem ]; then
    ln -sf "$SSL_KEY_PATH" /etc/nginx/certs/cert.pem
else
    echo "Symbolic link for cert.pem already exists."
fi
if [ ! -L /etc/nginx/certs/privkey.pem ]; then
    ln -sf "$SSL_KEY_PATH" /etc/nginx/certs/privkey.pem
else
    echo "Symbolic link for privkey.pem already exists."
fi

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
