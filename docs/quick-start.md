# Quick Start Guide

Get up and running with the ECHONET Lite controller in just a few minutes.
For Raspberry Pi / long-running installs, use
[docs/installation.md](installation.md) for the maintained systemd workflow.
Quick Start is intended for short-lived local evaluation only.

## 1. Build

```bash
# Clone and build
git clone https://github.com/koizuka/echonet-list.git
cd echonet-list
./script/build.sh
```

## 2. TLS Setup (Required for browsers)

TLS is required even on a trusted LAN because modern browsers (especially on
mobile) block non-secure WebSocket connections from secure pages.

```bash
# Install mkcert (macOS)
brew install mkcert
mkcert -install

# Generate certificates
mkdir -p certs
mkcert -cert-file certs/localhost+2.pem -key-file certs/localhost+2-key.pem localhost 127.0.0.1 ::1

# Run with TLS
./echonet-list -websocket -http-enabled -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem
```

Open your browser to `https://localhost:8080`.

## 3. Basic Usage

### Web UI

The Web UI automatically discovers and displays ECHONET Lite devices on your network:

- **Tabs**: Devices are organized by location and device groups
- **Device Cards**: Click to expand and see all properties
- **Property Editing**: Click on editable properties to change values
- **Real-time Updates**: Changes are reflected immediately

### Console UI

For direct terminal control:

```bash
./echonet-list
```

Common commands:

- `discover`: Discover ECHONET Lite devices on the network
- `devices` or `list`: List all discovered devices
- `get [device] [epc]`: Get property value from a device
- `set [device] [property]`: Set property value on a device
- `alias`: Manage device aliases for easier access
- `group`: Manage device groups
- `help`: Show available commands
- `quit`: Exit the program

## 4. Common Scenarios

### Home Automation Setup

```bash
# Create configuration
cp config.toml.sample config.toml

# Edit config.toml to enable WebSocket/HTTP and TLS, then run:
./echonet-list -config config.toml
```

### Development Mode

```bash
# Terminal 1: Run server
./echonet-list -websocket -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem

# Terminal 2: Run web dev server
cd web && npm run dev
```

### Background Service (Linux/macOS)

```bash
# Use the maintained systemd installer
./script/build.sh
sudo ./script/install-systemd.sh
```

## 5. Supported Devices

The following ECHONET Lite device types are supported:

### Climate Control

- **Home Air Conditioner (0x0130)**: Temperature, operation modes (cool/heat/dry/fan/auto), air flow control, humidity monitoring
- **Floor Heating (0x027B)**: Temperature levels, timer scheduling, room/floor/water temperature monitoring

### Lighting

- **Single Function Lighting (0x0291)**: On/off control, brightness adjustment (0-100%)
- **Lighting System (0x02A3)**: Advanced control with scene management (up to 253 scenes)

### Kitchen Appliances

- **Refrigerator (0x03B7)**: Door status monitoring, open door alerts

### System Components

- **Controller (0x05FF)**: Device management and network control
- **Node Profile (0x0EF0)**: Required system component for all ECHONET Lite nodes

## 6. Troubleshooting

### No devices found?

- Check that your computer and devices are on the same network
- Ensure UDP port 3610 is not blocked by firewall
- Try running with debug mode: `./echonet-list -debug`

### Certificate errors?

- Make sure mkcert CA is installed: `mkcert -install`
- Regenerate certificates if needed
- Check [mkcert Setup Guide](mkcert_setup_guide.md)

### Can't access Web UI?

- Verify the server is running with `-http-enabled`
- Check if port 8080 is available
- Try a different port: `-http-port 8081`

## Next Steps

- [Console UI Usage Guide](console_ui_usage.md) - Master the terminal interface
- [Configuration Guide](configuration.md) - Customize your setup
- [Server Modes Guide](server-modes.md) - Understand different operation modes
- [Web UI Implementation Guide](web_ui_implementation_guide.md) - Learn about the web interface
- [Daemon Setup Guide](daemon-setup.md) - Run as a system service
