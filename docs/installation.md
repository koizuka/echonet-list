# Deployment Guide (maintained)

This document replaces the old installation/daemon/how-to notes. Follow it for a fresh deployment that you can keep updated with `script/auto-update.sh`. Other docs in this folder are still useful but may lag behind current best practices; treat this guide as the canonical reference for provisioning and operating the server.

## Overview of the recommended flow

1. Prepare a Linux host with `git`, Go 1.23+, Node.js 18+, and systemd.
2. Clone this repository and build the release artifacts.
3. Generate TLS material with `mkcert` so the Web UI/WebSocket endpoint can stay on HTTPS/WSS.
4. Install the service via `script/install-systemd.sh`.
5. Distribute the mkcert CA to every browser/OS that should trust the UI.
6. Keep the instance up to date with `script/auto-update.sh` (plus an optional systemd timer or cron entry).

The sections below describe each step in more detail.

## 0. Requirements

- Ubuntu/Debian/Raspberry Pi OS (systemd based) with outbound HTTPS access.
- Packages: `git`, `build-essential`, `golang`, `nodejs`, `npm`, `mkcert`, `libnss3-tools`.
- A non-root user with sudo for build steps and root access for installation.
- Optional but recommended: `mkcert` installed on your laptop as well to ease CA distribution.

```bash
sudo apt update
sudo apt install -y git build-essential golang nodejs npm mkcert libnss3-tools
```

## 1. Clone the repository

```bash
cd /opt
sudo git clone https://github.com/koizuka/echonet-list.git
sudo chown -R "$USER":"$USER" echonet-list
cd echonet-list
```

If you maintain a fork, replace the clone URL accordingly. Keep the working copy clean—`script/auto-update.sh` assumes it can run `git pull` without local changes.

## 2. Build the server and Web UI

```bash
./script/build.sh server   # Go binary
./script/build.sh web      # Vite/React bundle
```

The default `./script/build.sh` with no arguments builds both parts. Confirm that `./echonet-list` exists and `web/bundle/` contains static assets before moving on.

## 3. Prepare TLS with mkcert

The systemd install script copies whatever lives in `./certs/` into `/etc/echonet-list/certs`, so generate the files now.

```bash
mkcert -install                       # one-time per host
mkdir -p certs
mkcert \
  -cert-file certs/localhost+2.pem \
  -key-file certs/localhost+2-key.pem \
  "$(hostname)" "$(hostname -f)" localhost 127.0.0.1 ::1
```

Update `config.toml` (or keep the provided `systemd/config.toml.systemd`) so the WebSocket listener points to the filenames above. You can regenerate the certificates later with the same command; just rerun `sudo ./script/update.sh` afterward so the files are copied into `/etc/echonet-list/certs`.

TLS is required even on a trusted LAN because modern browsers (especially on
mobile) block non-secure WebSocket connections from secure pages. Keep TLS
enabled and distribute the mkcert CA to every client device so HTTPS/WSS works
without warnings.

### FAQ: TLS on a LAN

**Why is TLS required on a local network?**  
Modern browsers enforce secure WebSocket rules: a page loaded over HTTPS cannot
connect to `ws://`. Mobile browsers are especially strict, so HTTPS/WSS is the
reliable default.

**What does mkcert do here?**  
It creates a local CA and a server certificate trusted by devices where you
install the CA. That avoids browser warnings while keeping traffic encrypted.

## 4. Install as a systemd service

1. Review configuration: copy `config.toml.sample` to `config.toml` if you need to tweak the defaults before installing, or edit `/etc/echonet-list/config.toml` after install.
2. Run the installer:

   ```bash
   sudo ./script/install-systemd.sh
   ```

   The script:
   - creates the `echonet` system user/group,
   - copies `echonet-list` into `/usr/local/bin`,
   - copies `web/bundle/` into `/usr/local/share/echonet-list/web`,
   - copies certificates into `/etc/echonet-list/certs`,
   - seeds `/var/lib/echonet-list` with `devices.json`, `groups.json`, etc.,
   - registers and starts `echonet-list.service`.

3. Verify the service:

   ```bash
   sudo systemctl status echonet-list
   sudo journalctl -u echonet-list -n 100
   ```

The Web UI is available at `https://<host>:8080` once the service is up and the certificate is trusted.

## 5. Trust the mkcert CA on every client

`mkcert` creates a host-local certificate authority. The UI will only work over HTTPS/WSS once each client trusts that CA.

1. Locate the CA file:

   ```bash
   mkcert -CAROOT
   # -> copy rootCA.pem from this directory
   ```

2. Install the CA:
   - **macOS**: double-click `rootCA.pem`, add it to the System keychain, and set "Always Trust".
   - **iOS/iPadOS**: AirDrop or email the file, install the profile, then enable it under `Settings > General > About > Certificate Trust Settings`.
   - **Android**: copy the file, rename to `rootCA.cer` if required, install under `Settings > Security > Encryption & credentials > Install from storage`, choose "VPN and apps".
   - **Windows**: run `certutil -addstore -f "ROOT" rootCA.pem` from an elevated prompt.
   - **Firefox** (all platforms): `Settings > Privacy & Security > Certificates > View Certificates > Authorities > Import`.

3. Repeat for every device that should open the UI.

If you regenerate the CA (`mkcert -uninstall` / `mkcert -install`), you must redistribute the new CA file to all clients.

## 6. Keeping the instance updated

`script/auto-update.sh` wraps the typical flow (`git pull` → decide what changed → `./script/build.sh` → `sudo ./script/update.sh`). Run it from the repository root.

```bash
./script/auto-update.sh        # production run
./script/auto-update.sh --dry-run
```

To automate updates, create a dedicated systemd timer (example):

```bash
sudo tee /etc/systemd/system/echonet-auto-update.service >/dev/null <<EOF
[Unit]
Description=Update echonet-list working copy
After=network-online.target

[Service]
Type=oneshot
User=$USER
WorkingDirectory=/opt/echonet-list
ExecStart=/opt/echonet-list/script/auto-update.sh
EOF

sudo tee /etc/systemd/system/echonet-auto-update.timer >/dev/null <<'EOF'
[Unit]
Description=Run echonet-list auto update hourly

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now echonet-auto-update.timer
```

The timer runs as the repository owner (not root) and invokes `sudo ./script/update.sh` internally whenever a rebuild is required.

## 7. Manual maintenance commands

- Rebuild only what changed:

  ```bash
  ./script/build.sh server
  ./script/build.sh web
  ```

- Deploy those bits without a git pull:

  ```bash
  sudo ./script/update.sh          # binary + web bundle + service reload
  sudo ./script/update.sh --web-only
  ```

- Check logs and status:

  ```bash
  sudo systemctl status echonet-list
  sudo journalctl -u echonet-list -f
  tail -f /var/log/echonet-list.log
  ```

- Rotate TLS certificates:

  ```bash
  mkcert -install       # only if you need a new CA
  mkcert -cert-file certs/localhost+2.pem -key-file certs/localhost+2-key.pem <hosts...>
  sudo ./script/update.sh
  sudo systemctl restart echonet-list
  ```

## 8. Related references

- [websocket_client_protocol.md](websocket_client_protocol.md) — actively updated API contract for custom clients.
- [../script/README.md](../script/README.md) — what each deployment script does and how to run it.
- Legacy docs (quick start, daemon, troubleshooting, etc.) remain in this folder for historical reasons but are no longer reviewed; rely on this guide instead for deployment questions.
