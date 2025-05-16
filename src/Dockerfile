# Use the official Golang image to build the relay app
FROM golang:1.24 AS builder
WORKDIR /relay-app
COPY khatru-relay/. .
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o khatru-relay .

WORKDIR /pre-receive-app
COPY git-hook-pre-receive/. .
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o pre-receive .

# Final stage with nginx, fcgiwrap, git backend, and nostr relay
FROM alpine:latest
# Copy nostr relay binary
COPY --from=builder /relay-app/khatru-relay /usr/local/bin/khatru-relay
COPY --from=builder /pre-receive-app/pre-receive /usr/local/bin/pre-receive

# install dependancies
RUN apk add --no-cache \
    nginx \
    fcgiwrap \
    git \
    git-daemon \
    spawn-fcgi \
    supervisor \
    bash \
    # openssl for temp self-signed ssl cert
    openssl \
    inotify-tools

# Create directory structure and set permissions
RUN mkdir -p /srv/static /srv/repos /srv/blossom /var/log/supervisor /run/nginx /var/log/nginx /relay-db && \
    chown -R nginx:nginx /run/nginx /var/log/nginx && \
    chmod -R 777 /srv/static /srv/repos /srv/blossom /relay-db

# Copy configuration, hooks, and entrypoint
COPY static/ /srv/static/
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
COPY reload-on-cert-change.sh /usr/local/bin/reload-on-cert-change.sh
RUN chmod +x /entrypoint.sh
RUN chmod +x /usr/local/bin/pre-receive
RUN chmod +x /usr/local/bin/reload-on-cert-change.sh


# Expose HTTP and nostr ports
EXPOSE 80 443

# Start all services via entrypoint
ENTRYPOINT ["/entrypoint.sh"]
