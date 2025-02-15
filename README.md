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

- Go 1.16 or later

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
  - `on` (equivalent to setting operation status to ON)
  - `off` (equivalent to setting operation status to OFF)

#### Update Device Properties

```bash
> update [ipAddress] [classCode[:instanceCode]]
```

Updates all properties of devices that match the specified criteria:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)

This command retrieves all properties listed in the device's GetPropertyMap and updates the local cache. It can be used to refresh the property values of one or multiple devices.

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
- `Devices.go`: Device management and storage
- `Session.go`: Session management for ECHONET Lite communication
- `UDPConnection.go`: UDP communication handling
- `echonet_lite/`: Package containing ECHONET Lite protocol implementation
  - `echonet_lite.go`: Core ECHONET Lite message handling
  - `EOJ.go`: ECHONET Object implementation
  - `Property.go`: Property handling
  - `ProfileSuperClass.go`: Base class for profiles
  - Device-specific implementations:
    - `HomeAirConditioner.go`
    - `FloorHeating.go`
    - `SingleFunctionLighting.go`
    - etc.

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

## References

- [ECHONET Lite Specification](https://echonet.jp/spec_v114_lite/)
- [ECHONET Lite Object Specification](https://echonet.jp/spec_object_rr2/)
