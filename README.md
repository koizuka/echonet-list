# ECHONET Lite Device Discovery and Control Tool

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Features

- Automatic discovery of ECHONET Lite devices on the local network
- List all discovered devices with their properties
- Get property values from specific devices
- Set property values on specific devices
- Persistent storage of discovered devices in a JSON file
- Support for various device types (air conditioners, lighting, floor heating, etc.)

## Installation

### Prerequisites

- Go 1.20 or later

### Building from Source

1. Clone the repository
2. Build the application:

```bash
go build
```

## Usage

Run the application:

```bash
./echonet-list [options]
```

### Command Line Options

- `-debug`: Enable debug mode to display detailed communication logs (packet contents, hex dumps, etc.)
- `-log`: Specify a log file name

Example with debug mode:

```bash
./echonet-list -debug
```

### Commands

The application provides a command-line interface with the following commands:

#### Discover Devices

```bash
> discover
```

This command broadcasts a discovery message to find all ECHONET Lite devices on the network.

#### List Devices

```bash
> devices
> list
```

Lists all discovered devices. You can filter the results:

```bash
> devices [ipAddress] [classCode[:instanceCode]] [-all|-props] [EPC1 EPC2 ...]
```

Options:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)
- `-all`: Show all properties
- `-props`: Show only known properties
- `EPC`: Show only specific properties (2 hexadecimal digits, e.g., 80)

#### Get Property Values

```bash
> get [ipAddress] classCode[:instanceCode] epc1 [epc2...]
```

Gets property values from a specific device:

- `ipAddress`: Target device IP address (optional if only one device matches the class code)
- `classCode`: Class code (4 hexadecimal digits, required)
- `instanceCode`: Instance code (1-255, defaults to 1 if omitted)
- `epc`: Property code to get (2 hexadecimal digits, e.g., 80)

#### Set Property Values

```bash
> set [ipAddress] classCode[:instanceCode] property1 [property2...]
```

Sets property values on a specific device:

- `ipAddress`: Target device IP address (optional if only one device matches the class code)
- `classCode`: Class code (4 hexadecimal digits, required)
- `instanceCode`: Instance code (1-255, defaults to 1 if omitted)
- `property`: Property to set, in one of these formats:
  - `EPC:EDT` (e.g., 80:30)
  - `EPC` (e.g., 80) - displays available aliases for this EPC
  - Alias name (e.g., `on`) - automatically expanded to the corresponding EPC:EDT
  - Examples:
    - `on` (equivalent to setting operation status to ON)
    - `off` (equivalent to setting operation status to OFF)
    - `80:on` (equivalent to setting operation status to ON)
    - `b0:auto` (equivalent to setting air conditioner to auto mode)

#### Update Device Properties

```bash
> update [ipAddress] [classCode[:instanceCode]]
```

Updates all properties of devices that match the specified criteria:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)

This command retrieves all properties listed in the device's GetPropertyMap and updates the local cache. It can be used to refresh the property values of one or multiple devices.

#### Device Aliases

```bash
> alias
> alias <aliasName>
> alias <aliasName> [ipAddress] classCode[:instanceCode]
> alias -delete <aliasName>
```

Manages device aliases for easier reference:

- No arguments: Lists all registered aliases
- `<aliasName>`: Shows information about the specified alias
- `<aliasName> [ipAddress] classCode[:instanceCode]`: Creates or updates an alias for a device
- `-delete <aliasName>`: Deletes the specified alias

Examples:
```bash
> alias ac 192.168.0.3 0130:1  # Create alias 'ac' for air conditioner at 192.168.0.3
> alias ac 0130                # Create alias 'ac' for the only air conditioner (if only one exists)
> alias ac                     # Show information about alias 'ac'
> alias -delete ac             # Delete alias 'ac'
> alias                        # List all aliases
```

Using aliases with other commands:
```bash
> get ac 80                    # Get operation status of device with alias 'ac'
> set ac on                    # Turn on device with alias 'ac'
```

#### Debug Mode

```bash
> debug [on|off]
```

Displays or changes the debug mode:

- No arguments: Display current debug mode
- `on`: Enable debug mode
- `off`: Disable debug mode

#### Help

```bash
> help
```

Displays help information about available commands.

#### Quit

```bash
> quit
```

Exits the application.

## Supported Device Types

The application supports various ECHONET Lite device types, including:

- Home Air Conditioner (0x0130)
- Ventilation Fan (0x0133)
- Floor Heating (0x027b)
- Single-Function Lighting (0x0291)
- Lighting System (0x02a3)
- Refrigerator (0x03b7)
- Switch (0x05fd)
- Portable Terminal (0x05fe)
- Controller (0x05ff)
- Node Profile (0x0ef0)

## Project Structure

- `main.go`: Entry point and main application logic
- `Command.go`: Command parsing and execution
- `CommandProcessor.go`: Command processing and execution
- `echonet_lite/`: Package containing ECHONET Lite protocol implementation
  - `DeviceAliases.go`: Device alias management
  - `Devices.go`: Device implementation and management
  - `ECHONETLiteHandler.go`: ECHONET Lite communication handler
  - `echonet_lite.go`: Core ECHONET Lite message handling
  - `EOJ.go`: ECHONET Object implementation
  - `IPAndEOJ.go`: IP address and EOJ handling
  - `logger.go`: Logging utilities
  - `network.go`: Network communication utilities
  - `Property.go`: Property handling
  - `Session.go`: Session management for ECHONET Lite communication
  - `UDPConnection.go`: UDP communication handling
  - Device-specific implementations:
    - `NodeProfileObject.go`: Node profile implementation
    - `ProfileSuperClass.go`: Base class for profiles
    - `HomeAirConditioner.go`
    - `FloorHeating.go`
    - `SingleFunctionLighting.go`

## Example Use Cases

### Discovering and Controlling an Air Conditioner

1. Start the application
2. Discover devices: `discover`
3. List all devices: `devices`
4. Get the operation status of an air conditioner: `get 0130 80`
5. Turn on the air conditioner: `set 0130 on`
6. Set the temperature to 25°C: `set 0130 b3:19` (25°C in hexadecimal is 0x19)

### Controlling Lights

1. Discover devices: `discover`
2. List all lighting devices: `devices 0291`
3. Turn on a light: `set 0291 on`
4. Turn off a light: `set 0291 off`

### Updating Device Properties

1. Discover devices: `discover`
2. Update all properties of all air conditioners: `update 0130`
3. Update all properties of a specific device: `update 192.168.0.5 0130:1`
4. Check the updated properties: `devices 0130 -all`

## Troubleshooting

### Common Errors

#### Port Already in Use

If you encounter the error message:
```
listen udp :3610: bind: address already in use
```
This indicates that another instance of the application is already running and using UDP port 3610. 

**Resolution:**
1. Find and terminate the other running instance of the application
   - On Linux/macOS: `ps aux | grep echonet-list` to find the process, then `kill <PID>` to terminate it
   - On Windows: Use Task Manager to end the process
2. After stopping the other instance, try running the application again

## References

- [ECHONET Lite Specification](https://echonet.jp/spec_v114_lite/)
- [ECHONET Lite Object Specification](https://echonet.jp/spec_object_rr2/)
