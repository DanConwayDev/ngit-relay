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

# Final stage with nginx, fcgiwrap, git backend, and nostr relay
FROM alpine:latest
# Copy nostr relay binaries
COPY --from=builder /relay-app/ngit-relay-khatru /usr/local/bin/ngit-relay-khatru
COPY --from=builder /pre-receive-app/ngit-relay-pre-receive /usr/local/bin/ngit-relay-pre-receive

# Install necessary packages
RUN apk add --no-cache \
    nginx \
    fcgiwrap \
    git \
    git-daemon \
    spawn-fcgi \
    supervisor \
    bash

# Copy configuration, hooks, and entrypoint
RUN mkdir -p /srv/ngit-relay/static && \
    chmod -R 777 /srv/ngit-relay/static
COPY static/ /srv/ngit-relay/static/
RUN mkdir -p /var/log/supervisor /run/nginx /var/log/nginx && \
    chown -R nginx:nginx /run/nginx /var/log/nginx
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
RUN chmod +x /usr/local/bin/ngit-relay-pre-receive

# Expose HTTP and nostr ports
EXPOSE 8081

# Start all services via entrypoint
ENTRYPOINT ["/entrypoint.sh"]
