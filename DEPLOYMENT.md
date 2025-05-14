    ## Deploy to VPS

    To deploy this application to a fresh VPS over SSL, follow these steps:

    ### Prerequisites

    1. **VPS Setup**: Ensure you have a fresh VPS with a Linux distribution (e.g., Ubuntu, CentOS).
    2. **Domain Name**: Have a domain name pointing to your VPS IP address.
    3. **Docker and Docker Compose**: Install Docker and Docker Compose on your VPS.

    ### Step 1: Install Docker and Docker Compose

    Run the following commands to install Docker and Docker Compose:

    ```bash
    # Update package index
    sudo apt update

    # Install Docker
    sudo apt install -y docker.io

    # Start Docker and enable it to run on boot
    sudo systemctl start docker
    sudo systemctl enable docker

    # Install Docker Compose
    sudo apt install -y docker-compose
    ```

    ### Step 2: Set Up SSL with Let's Encrypt

    1. **Install Certbot**:

    ```bash
    sudo apt install -y certbot
    ```

    2. **Obtain SSL Certificate**:

    Replace `yourdomain.com` with your actual domain name.

    ```bash
    sudo certbot certonly --standalone -d yourdomain.com
    ```

    3. **Set Up Automatic Renewal**:

    Certbot automatically sets up a cron job for renewal. You can check it with:

    ```bash
    sudo certbot renew --dry-run
    ```

    ### Step 3: Update Your Nginx Configuration

    Edit your `nginx.conf` file to include the following configuration:

    ```nginx
    server {
        listen 80;
        server_name _;  # Catch-all for any domain
        return 301 https://$host$request_uri;  # Redirect all HTTP requests to HTTPS
    }

    server {
        listen 443 ssl;
        server_name _;  # Catch-all for any domain

        ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;  # Path to your SSL certificate
        ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;  # Path to your SSL certificate key

        # Log errors to help with debugging
        error_log /var/log/nginx/error.log debug;
        access_log /var/log/nginx/access.log;

        # Serve Git repositories
        location ~ ^/npub1([a-z0-9]+)/([^/]+\.git)(/.*)?$ {
            # Enable CORS
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization';
            add_header 'Access-Control-Expose-Headers' '*';

            # Route to git-http-backend
            fastcgi_pass unix:/var/run/fcgiwrap.socket;
            include fastcgi_params;
            fastcgi_param SCRIPT_FILENAME /usr/libexec/git-core/git-http-backend;
            fastcgi_param GIT_HTTP_EXPORT_ALL "";
            fastcgi_param GIT_PROJECT_ROOT /srv/repos;
            fastcgi_param PATH_INFO /npub1$1/$2$3;
        }

        # Proxy to khatru-relay and blossom server
        location / {
            # Enable CORS
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization, X-Requested-With';
            add_header 'Access-Control-Expose-Headers' '*';
            add_header 'Access-Control-Max-Age' '86400';

            # Handle preflight requests
            if ($request_method = 'OPTIONS') {
                add_header 'Access-Control-Allow-Origin' '*';
                add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
                add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization, X-Requested-With';
                add_header 'Access-Control-Expose-Headers' '*';
                add_header 'Access-Control-Max-Age' '86400';
                add_header 'Content-Length' 0;
                return 204;  # No content response
            }

            # Proxy traffic to khatru
            proxy_pass http://localhost:3334;  # Adjust this if your khatru service runs on a different port
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # Handle WebSocket connections
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
    ```

    ### Step 4: Deploy Your Application

    1. **Create a Directory for Your Project**:

    ```bash
    mkdir ~/my-docker-app
    cd ~/my-docker-app
    ```

    2. **Copy Your Docker Compose Files**:

    Place your `docker-compose.yml`, `nginx.conf`, and any other necessary files into the `~/my-docker-app` directory.

    3. **Start Your Application**:

    Use Docker Compose to build and start your application:

    ```bash
    sudo docker-compose up -d
    ```

    This command will run your application in detached mode.

    4. **Check the Status of Your Containers**:

    You can check if your containers are running correctly with:

    ```bash
    sudo docker-compose ps
    ```

    5. **Test Your Application**:

    Open a web browser and navigate to `https://yourdomain.com` (replace `yourdomain.com` with your actual domain). You should see your application running over SSL.

    6. **Monitor Logs**:

    If you encounter any issues, you can check the logs of your application using:

    ```bash
    sudo docker-compose logs
    ```

    ### Step 5: Set Up Automatic Renewal for SSL Certificates

    To ensure your SSL certificates are renewed automatically, you can set up a cron job. Open the crontab editor:

    ```bash
    sudo crontab -e
    ```

    Add the following line to the crontab to run the renewal command twice a day:

    ```bash
    0 0,12 * * * certbot renew --quiet
    ```

    This command will check for certificate renewals at midnight and noon every day.

    ### Conclusion

    Your application is now deployed on a VPS with SSL enabled. Make sure to monitor your application and logs for any issues, and keep your server and dependencies updated.
    ```