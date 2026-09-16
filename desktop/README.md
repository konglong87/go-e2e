# Desktop Technical Validation

This directory contains the Phase 1 Wails v2 desktop shell.

## Development

Build the desktop validation package:

```bash
scripts/build-desktop.sh
```

For development, build the WebUI and local server binary:

```bash
npm --prefix web run build
go build -o desktop/golang-cc ./cmd/golang-cc
```

`scripts/build-desktop.sh` copies `web/dist` into the Wails
`frontend/dist` directory before building. The server binary is resolved from
beside the desktop executable, or from `GOLANG_CC_SERVER_BINARY`; the build
script places it beside the macOS app executable after Wails packaging.

## Phase 1 boundary

The current validation shell launches the local Go server as a child process.
The first launch asks for a workspace and stores it in the per-user desktop
config directory with restrictive permissions. SQLite, keychain storage,
provider onboarding, signing, and auto-update are planned for later phases.
