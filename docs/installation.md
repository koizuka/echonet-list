# Installation Guide

This guide covers the prerequisites, build process, and initial setup for the ECHONET Lite Device Discovery and Control Tool.

## Prerequisites

- Go 1.23 or later
- Node.js 18+ and npm (for Web UI development)

## Building from Source

### 1. Clone the Repository

```bash
git clone https://github.com/koizuka/echonet-list.git
cd echonet-list
```

### 2. Build the Go Application

```bash
go build
```

This creates the `echonet-list` executable in the current directory.

### 3. Build the Web UI

```bash
cd web
npm install
npm run build
```

The Web UI will be built to `web/bundle/` directory, which is served by the Go HTTP server.

## Setting up TLS Certificates (Recommended)

For secure Web UI access, you'll need TLS certificates. For development, we recommend using mkcert.

### Quick Setup with mkcert

1. **Install mkcert:**

```bash
# macOS
brew install mkcert

# Linux
sudo apt install libnss3-tools
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo cp mkcert-v*-linux-amd64 /usr/local/bin/mkcert
```

2. **Install the local CA:**

```bash
mkcert -install
```

3. **Generate certificates:**

```bash
mkdir -p certs
mkcert -cert-file certs/localhost+2.pem -key-file certs/localhost+2-key.pem localhost 127.0.0.1 ::1
```

4. **Run with TLS enabled:**

```bash
./echonet-list -websocket -http-enabled -ws-tls -ws-cert-file=certs/localhost+2.pem -ws-key-file=certs/localhost+2-key.pem
```

5. **Access the Web UI** at `https://localhost:8080`

For detailed certificate setup instructions and troubleshooting, see the [mkcert Setup Guide](mkcert_setup_guide.md).

## Alternative Build Method

For convenience, you can use the provided build script:

```bash
./script/build.sh
```

This script builds both the Go application and Web UI in one step.

## Platform-Specific Notes

### macOS

- Ensure you have Xcode Command Line Tools installed
- If using Homebrew, you can install Go with: `brew install go`

### Linux

- Most distributions have Go in their package managers
- Ubuntu/Debian: `sudo apt install golang`
- Fedora: `sudo dnf install golang`

### Windows

- Use the official Go installer from <https://golang.org>
- Consider using WSL2 for a better development experience

## Next Steps

After successful installation:

1. Create a configuration file: `cp config.toml.sample config.toml`
2. See [Configuration Guide](configuration.md) for configuration options
3. Check [Quick Start Guide](quick-start.md) for basic usage
4. For daemon mode setup, see [Daemon Setup Guide](daemon-setup.md)

## Troubleshooting

If you encounter issues during installation, see the [Troubleshooting Guide](troubleshooting.md).
