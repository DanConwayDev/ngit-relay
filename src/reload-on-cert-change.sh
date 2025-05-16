#!/bin/sh
: "${CERT_DOMAIN?Need CERT_DOMAIN env var}"
LIVE_DIR="/etc/letsencrypt/live/${CERT_DOMAIN}"
echo "Watching ${LIVE_DIR} for certificate updates ..."
while inotifywait -q -e close_write,move,create,delete "${LIVE_DIR}"; do
  echo "Certificates changed – reloading nginx"
  ln -sf "${LIVE_DIR}/fullchain.pem" /etc/nginx/certs/fullchain.pem
  ln -sf "${LIVE_DIR}/privkey.pem" /etc/nginx/certs/privkey.pem
  nginx -s reload
done