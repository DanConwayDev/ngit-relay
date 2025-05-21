# Use the official Golang image to build the relay app
FROM golang:1.24 AS builder
WORKDIR /relay-app
COPY ngit-relay-khatru/. .
COPY shared/ ../shared/
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o ngit-relay-khatru .

WORKDIR /pre-receive-app
COPY ngit-relay-pre-receive/. .
COPY shared/ ../shared/
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o ngit-relay-pre-receive .

WORKDIR /post-receive-app
COPY ngit-relay-post-receive/. .
COPY shared/ ../shared/
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o ngit-relay-post-receive .

# Final stage with nginx, fcgiwrap, git backend, and nostr relay
FROM alpine:latest
# Copy nostr relay binary
COPY --from=builder /relay-app/ngit-relay-khatru /usr/local/bin/ngit-relay-khatru
COPY --from=builder /pre-receive-app/ngit-relay-pre-receive /usr/local/bin/ngit-relay-pre-receive
COPY --from=builder /post-receive-app/ngit-relay-post-receive /usr/local/bin/ngit-relay-post-receive

# install dependancies
RUN apk add --no-cache \
    nginx \
    fcgiwrap \
    git \
    git-daemon \
    spawn-fcgi \
    supervisor \
    bash

# Create directory structure and set permissions
RUN mkdir -p /srv/ngit-relay/static /srv/ngit-relay/repos /srv/ngit-relay/blossom /var/log/supervisor /run/nginx /var/log/nginx /srv/ngit-relay/relay-db /var/log/ngit-relay && \
    chown -R nginx:nginx /run/nginx /var/log/nginx && \
    chmod -R 777 /srv/ngit-relay/static /srv/ngit-relay/repos /srv/ngit-relay/blossom /srv/ngit-relay/relay-db /var/log/ngit-relay

# Copy configuration, hooks, and entrypoint
COPY static/ /srv/ngit-relay/static/
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
RUN chmod +x /usr/local/bin/ngit-relay-pre-receive
RUN chmod +x /usr/local/bin/ngit-relay-post-receive

# Expose HTTP and nostr ports
EXPOSE 8081

# Start all services via entrypoint
ENTRYPOINT ["/entrypoint.sh"]
