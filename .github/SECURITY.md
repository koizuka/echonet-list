# Security Policy

## Reporting a Vulnerability

Please **do not** open a public issue for security problems.

Report vulnerabilities privately through GitHub's
[Private Vulnerability Reporting](https://docs.github.com/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability):

1. Open the **Security** tab of this repository.
2. Click **Report a vulnerability**.
3. Describe the issue, affected versions, and reproduction steps.

We aim to acknowledge reports within a few days. This is a volunteer-maintained
project, so please allow reasonable time for a fix before any public disclosure.

## Supported Versions

Only the latest commit on the default branch (`main`) is supported. Fixes are
delivered as new commits/releases rather than backported to older versions.

## Threat Model

`echonet-list` is an ECHONET Lite controller intended to run inside a **trusted
home LAN**. By design it has **no built-in authentication** and assumes every
device on the local network is trusted. Exposing the controller (its console,
WebSocket endpoint, or HTTP server) to untrusted networks or the public internet
is outside the intended threat model and is strongly discouraged.

In-scope examples worth reporting:

- Remote code execution or memory-safety issues in the Go server.
- Cross-site scripting or similar flaws in the Web UI.
- Vulnerabilities in third-party dependencies that affect this project.
- Supply-chain weaknesses in the build or CI pipeline.

## Supply-Chain Hardening

For the project's supply-chain security measures and a reusable checklist, see
[`docs/supply-chain-hardening.md`](../docs/supply-chain-hardening.md).
