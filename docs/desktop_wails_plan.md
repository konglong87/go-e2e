# Desktop Wails Plan

> 2026-09-16 状态更新：本文保留分阶段设计，后续状态以文末 Current Progress
> 和 [go-e2e 构建/使用说明](../desktop-v2/README.md) 为准。SQLite、模型设置
> 与 onboarding 已实现，不应再按初期 Phase 1 待办理解。

## Goal

Package `go-e2e` as a desktop application for macOS and Windows so a
non-technical user can select a workspace, configure a provider, and use the
existing Agent WebUI without opening a terminal.

## Architecture

```text
Wails desktop shell
  -> manages a local golang-cc server process
  -> exposes the server URL to the WebUI
  -> owns desktop-only lifecycle and platform integration

React/Vite WebUI
  -> existing chat, session, permission, file, trace, and settings flows

golang-cc server
  -> existing Gin API and Agent runtime
  -> localhost-only binding with an app-generated auth token
```

The desktop shell must not duplicate Agent business logic. Desktop-only
capabilities belong behind a small Wails service boundary; business requests
continue to use the existing HTTP/SSE API so Web, CLI, Desktop, and future
clients keep the same protocol.

## Desktop variants

The repository now keeps two installable Wails applications:

| Application | Directory | UI entry | Output |
| --- | --- | --- | --- |
| Legacy desktop | `desktop/` | Existing WebUI 1/default dashboard | `golang-cc-desktop` |
| WebUI 2 desktop | `desktop-v2/` | `/` desktop entry and `/webui/v2` session workbench | `go-e2e.app` / `go-e2e-setup.exe` |

The two variants share the same `web` source tree, Go server binary, API/SSE
contract, and workspace selection behavior. The v2 build sets a build-only
`VITE_DESKTOP_UI_VERSION=2` flag, so the legacy desktop and browser routes keep
their existing default entry behavior. Variant configuration files are
separate (`config.json` and `config-v2.json`) so installing both does not
overwrite desktop preferences.

Build the variants independently:

```bash
scripts/build-desktop.sh
scripts/build-desktop-v2.sh
```

## Scope and Milestones

### Phase 1: Wails technical validation

- Add a Wails v2 desktop entrypoint.
- Reuse the existing WebUI build.
- Bundle the WebUI build into the desktop binary resources.
- Start the local Go server automatically.
- Open the WebUI in a native window.
- Verify development and production builds on macOS and Windows.

Phase 1 deliberately does not include SQLite, onboarding, keychain storage,
auto-update, signing, or production packaging.

### Phase 2: Local desktop runtime

- Replace the temporary child-process bridge with a reusable in-process Go
  runtime composition.
- Add random-port discovery and readiness checks.
- Add graceful shutdown and crash recovery.
- Add desktop-local data directory selection.

### Phase 3: SQLite and first-run experience

- Add SQLite storage behind the existing tenant service contracts.
- Run SQLite migrations at startup.
- Add workspace and provider setup wizard.
- Store secrets using macOS Keychain and Windows Credential Manager.
- Keep MySQL as the server/team deployment option.

### Phase 4: Release hardening

- Add macOS and Windows installers.
- Add signing and macOS notarization.
- Add version migration and update checks.
- Add cross-platform smoke tests and failure diagnostics.

## Storage Direction

The first desktop release will not require MySQL. SQLite will implement the
existing tenant repository/service contracts so the HTTP handlers and WebUI
remain unchanged. The current code exposes some MySQL model types through
those contracts; this is acceptable for the first adapter, but common models
and contracts should later move out of `internal/storage/mysql` so SQLite and
MySQL become symmetric implementations.

## Phase 1 Acceptance

- `web` builds successfully.
- The desktop package can be compiled on macOS.
- The desktop package has a Windows build target.
- The packaged app contains the WebUI assets.
- Launching the desktop app starts a localhost-only Go server.
- The native window loads the WebUI and reaches `/health`.
- Closing the window stops the child server process.
- Existing Go and WebUI tests remain unchanged and pass where the local
  toolchain allows them to run.

## Current Progress

- Phase 1 complete: Wails shell, embedded WebUI, localhost server process, and
  macOS production build.
- WebUI 2 desktop variant complete: independent Wails shell, v2 build flag,
  separate output name/configuration, and macOS production build.
- Desktop server ports now default to an available localhost port; the
  `GOLANG_CC_DESKTOP_SERVER_PORT` override remains available for diagnostics.
- Local runtime: first-run workspace selection, durable desktop config,
  random loopback port, per-process auth token, readiness checks and bounded
  child-process shutdown are implemented. In-process composition and automatic
  crash recovery remain pending.
- SQLite with startup migrations, model settings/connection validation and
  onboarding are implemented. Legacy and v2 use separate SQLite files.
- macOS app build and Windows NSIS build workflow exist. The latest Windows
  workflow did not start because of GitHub billing limits; installer execution
  is not claimed as verified.
- Keychain/Credential Manager, signing/notarization and update checks remain
  pending. Runtime settings currently manage provider credentials.
- Desktop-v2 accepts the Wails root path as its session index without changing
  web/legacy routes or masking invalid deep links. See
  [entry acceptance](manual_testing/desktop_v2_entry.md) for current evidence.

## Known Temporary Trade-off

Phase 1 uses a local `golang-cc server` child process because the existing CLI
owns server composition and the server package does not yet expose a
listener-oriented runtime lifecycle. Phase 2 will extract that composition
behind a reusable runtime API; the child process is not the final architecture.

## Topology Impact

- `Topology impact: updated`
- `Blast radius: B2_MODE`
- `Topology reason: a new Desktop entrypoint consumes the existing WebUI and
  server output boundary; Agent runtime behavior remains unchanged, while
  process lifecycle and packaged asset delivery become a new mode-specific
  boundary.
