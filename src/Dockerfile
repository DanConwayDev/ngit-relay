# Use the official Golang image to build the application
FROM golang:1.24 AS builder
WORKDIR /app
COPY khatru-relay/. .
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o khatru-relay .

# Final stage with nginx, fcgiwrap, git backend, and nostr relay
FROM alpine:latest
# Copy nostr relay binary
COPY --from=builder /app/khatru-relay /usr/local/bin/khatru-relay

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
RUN mkdir -p /srv/repos /var/log/supervisor /run/nginx /var/log/nginx /khatru-data && \
    chown -R nginx:nginx /run/nginx /var/log/nginx && \
    chmod -R 777 /srv/repos /khatru-data

# Copy configuration, hooks, and entrypoint
COPY nginx.conf /etc/nginx/http.d/git.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
COPY git-http-backend/pre-receive /usr/local/bin/pre-receive
RUN chmod +x /usr/local/bin/pre-receive


# Expose HTTP and nostr ports
EXPOSE 8080 3334

# Start all services via entrypoint
ENTRYPOINT ["/entrypoint.sh"]
