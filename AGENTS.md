# Repository Guidelines

## Project Structure & Module Organization

Go services live in `server/`, `echonet_lite/`, and `protocol/`; `main.go` exposes CLI and daemon flags. Device metadata (`devices.json`, `groups.json`) stay at the root—copy `config.toml.sample` into `config/` when overriding defaults. The React/Vite UI sits in `web/` (start with `src/App.tsx` and `src/components/PropertyEditor.tsx`). Console tools live in `console/`, docs in `docs/`, integration assets in `integration/tests`, and `script/` is for target deployments.

## Build, Test, and Development Commands

Local loops rely on Go and npm directly: run `go build` or `go run ./main.go -debug` from the repo root, adding `-daemon -websocket` when validating background mode. Before committing, mirror `.github/workflows/ci.yml`: `gofmt -w .` (or format touched files), `go vet ./...`, and `go test -race -coverprofile=coverage.out ./...`. Inside `web/`, install once with `npm install`, develop via `npm run dev`, and set `VITE_WS_URL=wss://<host>/ws` to hit remote servers. Run `npm run lint`, `npm run typecheck`, `npm run test`, and `npm run build` ahead of pushes. Deployment scripts in `script/` (e.g., `build.sh`, `update.sh`, `install-systemd.sh`) assume the target host and should not drive local automation.

## Coding Style & Naming Conventions

Always format Go sources with `gofmt`; exports stay PascalCase, package helpers lowerCamelCase, and WebSocket logic follows the existing handler/transport split. Keep log strings concise and in English for downstream translation tables. React/TypeScript components use PascalCase filenames under `web/src`, hooks begin with `use`, and Vitest specs mirror targets via `.test.tsx`. Reuse shadcn/ui primitives and Tailwind utilities already adopted in the project.

## Web UI Behavior & Constraints

Device navigation already relies on tab groupings with per-tab/device status indicators, so retain the existing UX patterns when adding screens. Device cards fetch their always-visible data from `DEVICE_PRIMARY_PROPERTIES` and conditionally hide compact-view rows through `PROPERTY_VISIBILITY_CONDITIONS`; prefer alias strings (e.g., `'auto'`, `'cooling'`) over numeric EPC values when authoring those conditions, and remember they do not affect the expanded view. Property editing flows go through `PropertyEditor`: alias-backed fields render selects, raw values expose an edit button, and editing is restricted to what `Set Property Map (0x9E)` allows—keep `formatPropertyValue` in sync with any new formats or units you introduce.

Device history is rendered via `DeviceHistoryDialog` using the `useDeviceHistory` hook, so keep the existing data flow intact: the hook issues a `get_device_history` request with `settableOnly` baked into the payload—do not try to merge settable/non-settable streams client-side. Rows are grouped by second-level timestamps, origin, and settable flag; online/offline events always form their own rows, and columns must stay aligned with DeviceCard ordering (primary EPCs first, then insertion-order secondary EPCs). The table uses sticky timestamp/origin columns plus z-index layering so horizontal + vertical scrolling stays readable; preserve the `bg-*` priority (online green > offline red > settable blue) and compact row compression introduced in commits #367–#370. Hex data viewing is opt-in per cell via the binary button that calls `edtToHexString`, and it should remain a non-blocking drawer under the table—reuse `shouldShowHexViewer` to decide when to show that control.

## Internationalization

Property tables and descriptions are localized—`protocol/protocol.go`, `echonet_lite/PropertyDesc.go`, and `server/websocket_server_handlers_properties.go` already honor the `lang` parameter documented in `docs/internationalization.md`. Continue using English alias keys for protocol payloads (`set_properties`, etc.) while rendering translated labels from `aliasTranslations` in the UI, and add new strings in both languages when expanding device support.

## Testing Guidelines

Run `go test ./...` frequently and promote new coverage with focused unit cases. Device flows live in `integration/tests`; enable them with `go test -tags=integration ./integration/tests/...` after rebuilding artefacts. In the UI, colocate Vitest suites with components, execute `npm run test -- --run`, and inspect WebSocket payloads with `DEVICE_PRIMARY_PROPERTIES` when debugging property rendering.

## Runtime & Debug Tips

Daemon deployments inherit PID/log defaults from `config/config.go`; rotate logs with `SIGHUP` and inspect `journalctl -u echonet-list` on systemd hosts. Web bundles land in `web/bundle/`, letting the Go server serve HTTP and WebSocket from one origin. The console UI exposes `debugoffline <device> [on|off]` to simulate availability during UI tuning.

## Commit & Pull Request Guidelines

Commits follow Conventional Commits (`feat:`, `fix:`, `ci:`) with merged PR numbers appended, e.g. `feat: add multicast probe retry (#251)`. Keep commits focused and squash noisy dependency bumps. PRs should link issues, explain behavior changes, share UI screenshots when relevant, and confirm Go/web checks before requesting review.
