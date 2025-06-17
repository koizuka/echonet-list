# Console UI Usage Guide

This guide describes how to use the console UI of the ECHONET Lite Device Discovery and Control Tool.

## Starting the Console UI

Run the application without the `-daemon` flag to use the interactive console interface:

```bash
./echonet-list [options]
```

## Console Commands

The application provides a command-line interface with the following commands:

### Discover Devices

```bash
> discover
```

This command broadcasts a discovery message to find all ECHONET Lite devices on the network.

### List Devices

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

### Get Property Values

```bash
> get [ipAddress] classCode[:instanceCode] epc1 [epc2...] [-skip-validation]
```

Gets property values from a specific device:

- `ipAddress`: Target device IP address (optional if only one device matches the class code)
- `classCode`: Class code (4 hexadecimal digits, required)
- `instanceCode`: Instance code (1-255, defaults to 1 if omitted)
- `epc`: Property code to get (2 hexadecimal digits, e.g., 80)
- `-skip-validation`: Skip device existence validation (useful for testing timeout behavior)

### Set Property Values

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

### Update Device Properties

```bash
> update [ipAddress] [classCode[:instanceCode]]
```

Updates all properties of devices that match the specified criteria:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)

This command retrieves all properties listed in the device's GetPropertyMap and updates the local cache. It can be used to refresh the property values of one or multiple devices.

### Device Aliases

```bash
> alias
> alias <aliasName>
> alias <aliasName> [ipAddress] classCode[:instanceCode] [property1 property2...]
> alias -delete <aliasName>
```

Manages device aliases for easier reference:

- No arguments: Lists all registered aliases
- `<aliasName>`: Shows information about the specified alias
- `<aliasName> [ipAddress] classCode[:instanceCode] [property1 property2...]`: Creates or updates an alias for a device
- `-delete <aliasName>`: Deletes the specified alias

Examples:

```bash
> alias ac 192.168.0.3 0130:1           # Create alias 'ac' for air conditioner at 192.168.0.3
> alias ac 0130                          # Create alias 'ac' for the only air conditioner (if only one exists)
> alias aircon1 0130 living1             # Create alias 'aircon1' for air conditioner with installation location 'living1'
> alias aircon2 0130 on kitchen1         # Create alias 'aircon2' for powered-on air conditioner in the kitchen1
```

## Notes

- The console UI is not available when running in daemon mode (`-daemon` flag)
- Commands are case-insensitive
- Property codes (EPC) are specified in hexadecimal format
- Device class codes are 4-digit hexadecimal values as defined in the ECHONET Lite specification
