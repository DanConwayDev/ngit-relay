## Deploy to VPS

To deploy this ngit-relay to a fresh VPS over SSL, follow these steps:

### Prerequisites

1. **VPS Setup**: Ensure you have a fresh VPS with a Linux distribution (e.g., Ubuntu).
2. **Domain Name**: Have a domain name pointing to your VPS IP address.
3. **Docker and Docker Compose**: Install Docker and Docker Compose on your VPS.

### Step 1: Install Docker and Docker Compose

Run the following commands to install Docker and Docker Compose:

```bash
# add Docker’s repository
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg |
  sudo tee /etc/apt/keyrings/docker.asc >/dev/null
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" |
  sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io \
                    docker-buildx-plugin docker-compose-plugin

# Start Docker and enable it to run on boot
sudo systemctl start docker
sudo systemctl enable docker
```

Note: dont use `sudo apt install -y docker.io docker-compose` as `docker-compose` package is not maintained so it will cause issues.


#### Step 2: Clone the ngit-relay Repository

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

### Step 3: Configure Environment Settings

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

### Step 4: Deploy


1. **Start Your ngit-relay**:


Use Docker Compose to build and start your ngit-relay from the cloned repo directory (~/ngit-relay):

```bash
sudo docker compose up --build -d
```
This command will run your ngit-relay in detached mode.

2. **(Optional) Start SSL Provisioning**:
if you want ngit-relay to handle provisioning and renewing SSL certs with letsencypt run:

```bash
sudo docker compose -f docker-compose.certbot.yml up -d
```

More testing required here. If a letsencrypt cert isn't active within 30s check the logs. You may need to open a shell within the certbot docker image `sudo docker exec -it ngint-relay-certbot-1 sh`, remove the dir `rm -fr /etc/letsencrypt` and run `certbot certbot certonly --webroot -w /var/www/certbot --domains domain.com` manually after which it should check for renewals every 12hrs. 

3. **Check the Status of Your Containers**:

You can check if your containers are running correctly with:

```bash
sudo docker compose ps
```

5. **Test Your ngit-relay instance**:

Open a web browser and navigate to `https://domain.com` to see a landing page running with a valid SSL.

check the relay is working:
```bash
nak req -l 1 wss://domain.com
```


6. **Monitor Logs**:

If you encounter any issues, you can check the logs of your using:

```bash
sudo docker compose logs
```

### Step 6: Upgrade

YOLO upgrade from `~/ngit-relay` with 
```bash
sudo ./yolo-upgrade.sh
```



### Conclusion

ngit-relay is now deployed on a VPS with SSL enabled. Make sure to monitor your ngit-relay and logs for any issues, and keep your server and dependencies updated.
```