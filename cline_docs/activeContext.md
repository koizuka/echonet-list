# Active Context

## Current Task
Documentation for the alias command has been added to PrintUsage in Command.go and README.md, but the alias command implementation is not yet complete.

## Recent Changes
- Added documentation for the alias command to the PrintUsage function in Command.go
- Added a "Device Aliases" section to the README.md file
- The alias command allows users to create, view, delete, and list aliases for devices

## Next Steps
1. ✅ Update PrintUsage in Command.go to include alias command documentation (COMPLETED)
2. ✅ Update README.md to include alias command documentation (COMPLETED)
3. ⬜ Implement SaveToFile functionality for the alias command to persist aliases to disk
   - The DeviceAliases.go file has SaveToFile and LoadFromFile methods, but they need to be called in the appropriate places
   - Likely need to update ECHONETLiteHandler.go to call SaveToFile after alias operations
