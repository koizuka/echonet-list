# Repository Guidelines

## Project Structure & Module Organization

Go services live in `server/`, `echonet_lite/`, and `protocol/`; `main.go` exposes CLI and daemon flags. Device metadata (`devices.json`, `groups.json`) stay at the root—copy `config.toml.sample` into `config/` when overriding defaults. The React/Vite UI sits in `web/` (start with `src/App.tsx` and `src/components/PropertyEditor.tsx`). Console tools live in `console/`, docs in `docs/`, integration assets in `integration/tests`, and `script/` is for target deployments.

## Build, Test, and Development Commands

Local loops rely on Go and npm directly: run `go build` or `go run ./main.go -debug` from the repo root, adding `-daemon -websocket` when validating background mode. Before committing, mirror `.github/workflows/ci.yml`: `gofmt -w .` (or format touched files), `go vet ./...`, and `go test -race -coverprofile=coverage.out ./...`. Inside `web/`, install once with `npm install`, develop via `npm run dev`, and set `VITE_WS_URL=wss://<host>/ws` to hit remote servers. Run `npm run lint`, `npm run typecheck`, `npm run test`, and `npm run build` ahead of pushes. Deployment scripts in `script/` (e.g., `build.sh`, `update.sh`, `install-systemd.sh`) assume the target host and should not drive local automation.

## Coding Style & Naming Conventions

Always format Go sources with `gofmt`; exports stay PascalCase, package helpers lowerCamelCase, and WebSocket logic follows the existing handler/transport split. Keep log strings concise and in English for downstream translation tables. React/TypeScript components use PascalCase filenames under `web/src`, hooks begin with `use`, and Vitest specs mirror targets via `.test.tsx`. Reuse shadcn/ui primitives and Tailwind utilities already adopted in the project.

## Testing Guidelines

Run `go test ./...` frequently and promote new coverage with focused unit cases. Device flows live in `integration/tests`; enable them with `go test -tags=integration ./integration/tests/...` after rebuilding artefacts. In the UI, colocate Vitest suites with components, execute `npm run test -- --run`, and inspect WebSocket payloads with `DEVICE_PRIMARY_PROPERTIES` when debugging property rendering.

## Runtime & Debug Tips

Daemon deployments inherit PID/log defaults from `config/config.go`; rotate logs with `SIGHUP` and inspect `journalctl -u echonet-list` on systemd hosts. Web bundles land in `web/bundle/`, letting the Go server serve HTTP and WebSocket from one origin. The console UI exposes `debugoffline <device> [on|off]` to simulate availability during UI tuning.

## Commit & Pull Request Guidelines

Commits follow Conventional Commits (`feat:`, `fix:`, `ci:`) with merged PR numbers appended, e.g. `feat: add multicast probe retry (#251)`. Keep commits focused and squash noisy dependency bumps. PRs should link issues, explain behavior changes, share UI screenshots when relevant, and confirm Go/web checks before requesting review.
