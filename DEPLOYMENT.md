## Deploy to VPS

To deploy this application to a fresh VPS over SSL, follow these steps:

### Prerequisites

1. **VPS Setup**: Ensure you have a fresh VPS with a Linux distribution (e.g., Ubuntu).
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

To ensure your SSL certificates are renewed automatically, you can set up a cron job. Open the crontab editor:

```bash
sudo crontab -e
```

Add the following line to the crontab to run the renewal command twice a day:

```bash
0 0,12 * * * certbot renew --quiet
```

This command will check for certificate renewals at midnight and noon every day.

you can test it out with:

```bash
sudo certbot renew --dry-run
```

#### Step 3: Clone the ngit-relay Repository

1. **Navigate to Your Home Directory**:
Open your terminal and run the following command to ensure you are in your home directory:
```bash
cd ~
```

2. **Download ngit**:

get the ngit download link to the latest version from [https://gitworkshop.dev/ngit](https://gitworkshop.dev/ngit) for the ubuntu version you are running (24.04?).

Use `wget` to download the ngit package from the link:
```bash
wget https://github.com/DanConwayDev/ngit-cli/releases/download/v1.6.3/ngit-v1.6.3-ubuntu-24.04.tar.gz
```

3. **Unzip the Downloaded File**:
Extract the contents of the downloaded tar.gz file:
```bash
tar -xzf ngit-v1.6.3-ubuntu-24.04.tar.gz
```

Be sure to use the correct filename.

4. **Copy ngit Binaries to the Path**:
Assuming the binaries are located in the extracted folder, copy `ngit` and `git-remote-nostr` to a directory that is included in your system's PATH. For example, you can copy them to `/usr/local/bin`:
```bash
sudo cp ngit /usr/local/bin/
sudo cp git-remote-nostr /usr/local/bin/
```

5. **Verify Installation**
To ensure that ngit and git-remote-nostr are correctly installed, you can check their versions:
```bash
ngit --version
git-remote-nostr --version
```

6. **Clone the ngit-relay Repository**:
Use the following command to clone the ngit-relay repository:
```bash
git clone nostr://npub15qydau2hjma6ngxkl2cyar74wzyjshvl65za5k5rl69264ar2exs5cyejr/relay.damus.io/ngit-relay
```

7. **Navigate to the ngit-relay Directory**:
Change into the newly cloned directory:
```bash
cd ngit-relay
```

### Step 4: Configure Environment Settings

To configure the server, follow these steps to copy the `.env.example` file to `.env` and adjust the necessary settings.

1. **Copy the `.env.example` File**

Open your terminal and navigate to your project directory. Then, run the following command to copy the `.env.example` file to `.env`:

```bash
cp .env.example .env
```

2. **Edit the `.env` File**

Open the newly created `.env` file in a text editor of your choice. For example, you can use `nano`:

```bash
nano .env
```

3. **Adjust the Settings**

4. **Save and Exit**

After making the necessary changes, save the file and exit the text editor. If you are using `nano`, you can do this by pressing `CTRL + X`, then `Y` to confirm, and `Enter` to save.

### Step 5: Deploy


1. **Start Your Application**:


Use Docker Compose to build and start your application from the cloned repo directory (~/ngit-relay):

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

### Conclusion

ngit-relay is now deployed on a VPS with SSL enabled. Make sure to monitor your ngit-relay and logs for any issues, and keep your server and dependencies updated.
```