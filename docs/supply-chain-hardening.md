# Supply-Chain Hardening Checklist

A reusable checklist of supply-chain security measures, with notes on how this
repository (`echonet-list`) currently satisfies each item. It can be copied to
other Go / TypeScript projects as a starting point.

Legend: `[x]` done in this repo · `[ ]` not applicable or left to repo settings.

## GitHub Actions

- [x] **Pin actions to full commit SHAs**, not tags or branches. A tag can be
  moved to point at malicious code; a 40-character SHA cannot. Keep a `# vX.Y.Z`
  comment next to each SHA for readability.
- [x] **Least-privilege `GITHUB_TOKEN`.** Every workflow declares a restrictive
  top-level `permissions:` block (`contents: read`, or `{}`). Jobs that need more
  (for example, posting a PR comment) elevate with their own job-level block.
- [x] **Avoid `pull_request_target`.** It runs with a read/write token in the
  context of the base repo while checking out untrusted PR code. This repo uses
  only `pull_request` and `push`.
- [x] **Keep actions updated.** Dependabot watches the `github-actions`
  ecosystem so pinned SHAs receive update PRs.
- [ ] Consider a runner-hardening action (e.g. `step-security/harden-runner`) and
  OpenSSF Scorecard if the project's risk profile warrants it.

## npm / Node.js

- [x] **Commit the lockfile** (`web/package-lock.json`) and never delete it.
- [x] **Use `npm ci`**, not `npm install`, in CI and build scripts. `npm ci`
  installs exactly what the lockfile specifies and fails on any drift.
- [x] **No lifecycle scripts** (`postinstall`, etc.) in `web/package.json` that
  run arbitrary code on install.
- [x] **Single source of truth for the Node version.** `web/.nvmrc` pins the
  version; `package.json` `engines.node` records the minimum; CI's `setup-node`
  reads `node-version-file: web/.nvmrc`.
- [x] **Audit dependencies in CI.** The `npm-audit` job runs
  `npm audit --package-lock-only` on pull requests (report-only) and posts a
  sticky comment when high/critical advisories are found.
- [x] **Dependabot with a cooldown.** A `cooldown` delay avoids pulling a brand-new
  (potentially compromised) release immediately after publication.

## Go

- [x] **Commit `go.sum`.** It records cryptographic checksums for every module;
  `go` verifies downloads against it.
- [x] **Scan with `govulncheck`.** The `go-vuln` CI job runs the official Go
  vulnerability scanner (report-only) and reports reachable vulnerabilities via a
  sticky PR comment. The `govulncheck` tool version is pinned.
- [x] **Keep modules updated.** Dependabot watches the `gomod` ecosystem.
- [ ] Optionally set `GOFLAGS=-mod=readonly` / `GONOSUMCHECK` policy if the build
  environment needs stricter controls.

## Repository configuration

- [x] **`SECURITY.md`** documents private vulnerability reporting.
- [x] **`CODEOWNERS`** designates review responsibility.
- [ ] **Branch protection** (configured in GitHub settings, not in the repo):
  require PR review, require status checks (`CI Summary`), and optionally require
  Code Owner review.
- [ ] **Enable GitHub Private Vulnerability Reporting** and the Dependabot /
  secret-scanning alerts in repository settings.

## Verifying locally

```sh
# npm
cd web && npm audit --package-lock-only

# Go (requires network access to vuln.go.dev)
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

Both scans are **report-only** in CI: they surface findings without blocking
merges. Triage findings and let Dependabot deliver the fixes.
