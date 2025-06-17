# Contributing to ECHONET Lite Device Discovery and Control Tool

Thank you for your interest in contributing to this project! We welcome contributions of all kinds, including bug reports, feature requests, documentation improvements, and code contributions.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Respect differing viewpoints and experiences

## How to Contribute

### Reporting Issues

1. Check if the issue already exists in the [issue tracker](https://github.com/your-repo/echonet-list/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Your environment (OS, Go version, etc.)

### Suggesting Features

1. Open an issue with the "feature request" label
2. Describe the feature and its benefits
3. Include use cases and examples

### Contributing Code

#### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/echonet-list.git
   cd echonet-list
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/original-repo/echonet-list.git
   ```
4. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

#### Development Setup

1. Ensure you have Go 1.23+ installed
2. For Web UI development, install Node.js 18+ and npm
3. Install dependencies:
   ```bash
   # Go dependencies
   go mod download
   
   # Web UI dependencies
   cd web
   npm install
   ```

#### Coding Standards

**Go Code:**
- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for common issues
- Follow Go idioms and best practices
- Add tests for new functionality

**TypeScript/React Code:**
- Run `npm run lint` in the `web/` directory
- Follow the existing code style (function components, TypeScript types)
- Add unit tests using Vitest
- Use shadcn/ui components consistently

**General:**
- Write clear, self-documenting code
- Add comments for complex logic
- Update documentation as needed
- Keep commits focused and atomic

#### Testing

Before submitting:

1. Run Go tests:
   ```bash
   go test ./...
   ```

2. Run Web UI tests:
   ```bash
   cd web
   npm run test
   npm run lint
   ```

3. Test the integrated build:
   ```bash
   # Build everything
   go build
   cd web && npm run build && cd ..
   
   # Run with Web UI
   ./echonet-list -websocket -http-enabled
   ```

#### Submitting Pull Requests

1. Sync with upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. Create a Pull Request:
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changes you made and why
   - Include screenshots for UI changes
   - Ensure all checks pass

### Documentation

- Update README.md for new features
- Add/update documentation in the `docs/` directory
- Include code examples where appropriate
- Keep language clear and concise

## Development Workflow

1. **Discuss First**: For major changes, open an issue to discuss before starting work
2. **Test-Driven**: Write tests first when adding new functionality
3. **Small PRs**: Keep pull requests focused and manageable
4. **Review Ready**: Ensure your PR is ready for review (tests pass, code formatted, etc.)

## Release Process

Maintainers handle releases by:
1. Updating version numbers
2. Creating release notes
3. Tagging the release
4. Building and publishing binaries

## Getting Help

- Check existing documentation in `docs/`
- Look through existing issues and PRs
- Ask questions in issue discussions
- Refer to CLAUDE.md for project-specific guidelines

## Recognition

Contributors will be recognized in:
- The project's contributor list
- Release notes for significant contributions
- Special mentions for exceptional help

Thank you for contributing to the ECHONET Lite Device Discovery and Control Tool!