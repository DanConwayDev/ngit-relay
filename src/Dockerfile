FROM alpine:latest

# Install required packages
RUN apk add --no-cache \
    nginx \
    fcgiwrap \
    git \
    git-daemon \
    spawn-fcgi \
    supervisor \
    bash \
    curl

# Create directory structure
RUN mkdir -p /srv/repos /var/log/supervisor

# Configure nginx for git-http-backend
COPY nginx.conf /etc/nginx/http.d/git.conf

# Add pre-receive hook
COPY git-http-backend/pre-receive /usr/local/bin/pre-receive
RUN chmod +x /usr/local/bin/pre-receive

# Configure supervisord to manage processes
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Make sure all required directories exist with proper permissions
RUN mkdir -p /run/nginx /var/log/nginx && \
    chown -R nginx:nginx /var/lib/nginx /run/nginx /var/log/nginx && \
    mkdir -p /var/run && \
    chmod 777 /var/run

# Create modified entrypoint that uses the correct path for Alpine
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose HTTP port
EXPOSE 8080

# Start services
ENTRYPOINT ["/entrypoint.sh"]
