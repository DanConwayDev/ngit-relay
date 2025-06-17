# Deployment

Below is a copy/paste beginner's guide to deployment onto a VPS. But first, here are some hints for system administrators who don't want to use Docker:

1. When `ngit-relay-khatru` creates a new repository, it adds a symbolic link to `/usr/local/bin/ngit-relay-pre-receive` to install the Git hook. The binary must be available from that location.
2. `ngit-relay-pre-receive` uses `ws://localhost:3334` as its source for Git announcement/state events, so `ngit-relay-khatru` or another relay must be exposed there.
3. the EVs in .env should be set for `ngit-relay-khatru`

## Deploy to VPS

Beginner instructions to deploy ngit-relay over HTTPS/WSS on a fresh Ubuntu VPS:

### Prerequisites

1. **VPS Setup**: Purchase a fresh VPS with a Linux distribution (these instructions assume Ubuntu) eg. from [lnvps.net](https://lnvps.net).
2. **Domain Name**: Have a domain name pointing to your VPS IP address.

## 1. Prepare the VPS

1. Log in as root  
   `ssh root@VPS_IP`

2. Install Docker Engine + Compose (official packages)

```bash
# add Docker’s repository
sudo apt install -y ca-certificates curl gnupg make
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

3. Open HTTPS ports and enable the firewall

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable
```

## 2. Clone the ngit-relay Repository

1. **Navigate to Your Home Directory**:
   Open your terminal and run the following command to ensure you are in your home directory:

```bash
cd ~
```

2. **Install ngit**:

```bash
sudo curl -Ls https://ngit.dev/install.sh | sudo bash
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

The only thing you **need** to change is`NGIT_DOMAIN=example.com` to your domain name if you want to support SSL.

3. **Adjust the Settings**

4. **Save and Exit**

After making the necessary changes, save the file and exit the text editor. If you are using `nano`, you can do this by pressing `CTRL + X`, then `Y` to confirm, and `Enter` to save.

## 4. Deploy

1. Create the certificate store once

   ```bash
   touch acme.json && chmod 600 acme.json
   ```

2. Launch the stack with HTTPS

   ```bash
   sudo make
   ```

## 5. Test

1. Visit `https://example.com` (replace with your domain) — you should see the landing page with a valid certificate.

2. Verify relay traffic - if you have `nak` installed

   ```bash
   nak req -l 1 wss://example.com
   ```

3. Verify blossom server

```bash
npx blossom-audit audit https://example.com bitcoin --sec nsec1l9z9xem0s96paxa3kvk6vadgev0vm0rq6hl8e5wgmj6h0ustsh2qz3sntn
```

4. Follow the README.md instructions to create and push to a new repository pointing to this ngit-relay instance

### 5. Upgrade

YOLO upgrade from `~/ngit-relay` with

```bash
sudo ./yolo-upgrade.sh
```

### Conclusion

You now have ngit-relay running on a fresh VPS with automatic HTTPS.  
Monitor logs occasionally and keep both the system and containers up-to-date.
