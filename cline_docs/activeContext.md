# Active Context

## Current Task
The implementation of aliases.json saving functionality has been completed.

## Recent Changes
- Added documentation for the alias command to the PrintUsage function in Command.go
- Added a "Device Aliases" section to the README.md file
- Implemented the alias command to allow users to create, view, delete, and list aliases for devices
- Implemented SaveToFile functionality in ECHONETLiteHandler.go to persist aliases to disk in aliases.json

## Next Steps
1. ✅ Update PrintUsage in Command.go to include alias command documentation (COMPLETED)
2. ✅ Update README.md to include alias command documentation (COMPLETED)
3. ✅ Implement SaveToFile functionality for the alias command to persist aliases to disk (COMPLETED)
   - The DeviceAliases.go file has SaveToFile and LoadFromFile methods implemented
   - ECHONETLiteHandler.go calls SaveToFile after alias operations to persist aliases to disk
