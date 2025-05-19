## Deploy to VPS

To deploy this ngit-relay to a fresh VPS over SSL, follow these steps:

### Prerequisites

1. **VPS Setup**: Ensure you have a fresh VPS with a Linux distribution (e.g., Ubuntu).
2. **Domain Name**: Have a domain name pointing to your VPS IP address.

## 1. Prepare the VPS

1. Log in as root  
   `ssh root@VPS_IP`

2. Install Docker Engine + Compose (official packages)

```bash
apt update && apt -y upgrade

apt -y install ca-certificates curl gnupg lsb-release
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg |
  tee /etc/apt/keyrings/docker.asc >/dev/null
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" |
  tee /etc/apt/sources.list.d/docker.list >/dev/null

apt update
apt -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
newgrp docker # avoid reboot
```

3. Open HTTPS ports and enable the firewall

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
```

## 2. Clone the ngit-relay Repository

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
cd ngit-relay
```

## 3. Configure Environment Settings

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

## 4. First start

1. Create the certificate store once

   ```bash
   touch acme.json && chmod 600 acme.json
   ```

2. Launch the stack with HTTPS

   ```bash
   docker compose \
     -f docker-compose.yml \
     -f docker-compose-ssl-proxy.yml \
     up -d
   ```

(Leave out the second `-f` to run **without** the proxy.)

3. Optional: watch Traefik fetch the Let’s Encrypt cert

   ```bash
   docker compose -f docker-compose.yml -f docker-compose-ssl-proxy.yml logs -f ssl-proxy
   ```

## 5. Test

1. Visit `https://example.com` (replace with your domain) — you should see the landing page with a valid certificate.

2. Verify relay traffic

   ```bash
   nak req -l 1 wss://example.com
   ```

````

### 5. Upgrade

YOLO upgrade from `~/ngit-relay` with

```bash
sudo ./yolo-upgrade.sh
````

### Conclusion

You now have ngit-relay running on a fresh VPS with automatic HTTPS.  
Monitor logs occasionally and keep both the system and containers up-to-date.
