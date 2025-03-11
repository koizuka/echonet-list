# Progress

## What Works
- The ECHONET Lite application is functional and can discover and control devices
- The alias command functionality is partially implemented
  - Users can create, view, delete, and list aliases for devices in memory
  - Documentation for the alias command is complete in both PrintUsage and README.md
- The DeviceAliases.go file has SaveToFile and LoadFromFile methods implemented

## What's Left to Build
- Implement the persistence mechanism for device aliases
  - Call SaveToFile after alias operations to persist aliases to disk
  - Call LoadFromFile at startup to load saved aliases

## Progress Status
- **Alias Command Documentation in PrintUsage**: ✅ COMPLETED
- **Alias Command Documentation in README.md**: ✅ COMPLETED
- **Alias Command Persistence Implementation**: ⬜ PENDING
  - Need to implement calls to SaveToFile in the appropriate places
