
# AI Assistant Instructions for ECHONET Lite Project

## Project Overview
プロジェクトの概要については [cline_docs/productContext.md](../../cline_docs/productContext.md) を参照してください。

## Key Features
主要機能については [cline_docs/productContext.md](../../cline_docs/productContext.md) を参照してください。

## Development Environment
開発環境の詳細については [cline_docs/techContext.md](../../cline_docs/techContext.md) を参照してください。
追加情報:
- Project Structure: (Project Root)/README.md を参照し、変更があったら随時更新すること。

## Build Commands
ビルドコマンドについては [cline_docs/techContext.md](../../cline_docs/techContext.md) も参照してください。
- Build: `go build`
- Run: `./echonet-list [-debug]`
- Run tests: `go test ./...`
- Run specific test: `go test ./echonet_lite -run TestOperationStatus_Encode`
- Format code: `go fmt ./...`
- Check code: `go vet ./...`

## Code Style Guidelines
コードの構成とスタイルについては [cline_docs/systemPatterns.md](../../cline_docs/systemPatterns.md) も参照してください。
- **File Organization**: Package main in root, echonet_lite package for protocol implementation
- **Imports**: Group standard library, then external packages, then local packages
- **Naming**: CamelCase for exported entities, camelCase for unexported entities
- **Comments**: Comments for exported functions start with function name
- **Types**: Define custom types for specialized uses (e.g., `OperationStatus`)
- **Error Handling**: Return errors for recoverable failures, use proper error types
- **Testing**: Table-driven tests with descriptive names for each test case
- **Formatting**: Standard Go formatting with proper indentation
- **Hexadecimal**: Use `0x` prefix for clarity (e.g., `0x80` not just `80`)

## Development Workflow
開発ワークフローについては [cline_docs/techContext.md](../../cline_docs/techContext.md) も参照してください。
1. Make changes to Go files
2. Always run `go fmt ./...` after making changes
3. After significant changes, run `go vet ./...` to check for potential issues
4. Run tests to ensure functionality: `go test ./...`
5. Build and run the application to verify changes: `go build && ./echonet-list`

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

## Common Tasks
コードの構成と一般的なタスクについては [cline_docs/systemPatterns.md](../../cline_docs/systemPatterns.md) も参照してください。
- **Adding support for a new device type**: Create a new file in the `echonet_lite/` package
- **Modifying command behavior**: Update the relevant functions in `Command.go`
- **Debugging communication issues**: Run with `-debug` flag and check logs

## Supported Device Types
サポートされているデバイスタイプについては [cline_docs/productContext.md](../../cline_docs/productContext.md) を参照してください。

## AI Assistant Guidelines
When assisting with this project:
1. Refer to this instruction file for context about the project structure and conventions
2. Follow Go best practices and the project's code style guidelines
3. Consider the ECHONET Lite protocol specifications when suggesting code changes
4. Test suggestions with the provided test commands when possible
5. Prioritize maintaining compatibility with existing device implementations
6. When suggesting new features, ensure they align with the project's architecture
7. If a user encounters the "address already in use" error, instruct them to terminate any existing instances of the application before restarting

## References
- [ECHONET Lite Specification](https://echonet.jp/spec_v114_lite/)
- [ECHONET Lite Object Specification](https://echonet.jp/spec_object_rr2/)
