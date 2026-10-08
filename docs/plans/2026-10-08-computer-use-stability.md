# Computer Use stability and real desktop closure — 2026-10-08

## Architecture and safety contract

Reuse the host target registry, Controller/Runner, macOS Backend, native bridge,
SafetyState and existing acceptance wrapper. Keep stable window identity
(window ID + owner PID + bundle ID) separate from capture geometry, render
readiness and input authority. Diagnostics are metadata-only NDJSON, never
screenshots, window titles, input text, keys, headers or raw private errors.

An input outcome is based on dispatch stage, never only on an error code:
pre-dispatch rejection; fully acknowledged input but missing evidence;
partial/uncertain input; broken transport; user Pause/Stop; permission revocation.
Unknown input must not automatically replay even under a new action ID.
Explicit Pause/Stop and permissions always win. No app-specific core logic.

## Change ledger / execution queue

Baseline: main, HEAD/origin/main bb13e6e; only pre-existing generated
scripts/__pycache__/ is dirty. Never include it. No branch/worktree, no credential
inspection. Existing /tmp/tool-observe.log and /tmp/go-mismatch.log retained.

| Priority | Slice / intended files | Gate | Status |
|---|---|---|---|
| P0-1 | Append-only Go/native diagnostics; computerdiag, tool, backend, controller, native Diagnostics/Engine/Platform/Safety, helper build/tests | Focused Go + native regression; commit/push before real run | implemented; focused Go and 257 native assertions pass; committed/pushed d16fad7 |
| P0-2 | Fresh-run evidence isolation; scripts/computer-use-120s-acceptance.py and tests | Reject mixed time/session/build evidence and existing output dir; preserve launch receipt on observation failure | committed/pushed 3dcec8d + 6cf1d91; follow-up evidence in ee92517 |
| P0-3 | Diagnose real launch/observe; bounded unique-window stabilization and capture snapshot consistency | Fresh build identity + repeated cold observe-only, screenshots and Stop | implementation committed/pushed 222de75 + 950818d; real gate blocked by Mac lock |
| P0-4 | Dispatch-stage recovery contract across Swift/Go/controller/session | Native partial/complete/before-input faults, Go contract tests, unknown replay prevention, Pause/Stop/permission races | committed/pushed c6c2d48; automated safety gates pass |
| P0-5 | Actual autonomous model closure | Actual provider jiuan-responses-gpt-5.6sol, gpt-6-sol, high; launch→observe ready→new task→type 1+1=2→send once→reply screenshot→Stop, <=120 s | depends above |
| P1 | Current queue native Resume/Stop + broader regression, non-WorkBuddy fixture | Native clicks/screenshots; focused then broader Go/race | pending |
| Deferred | Second physical display; formal signing/distribution; blocked optional release artifacts | User explicitly deferred; not represented as passed | unchanged |

## Verification / delivery

For each coherent source slice: focused tests, git diff --check, status/stat,
commit konglong <konglong@com>, push origin/main. Desktop build exclusively via
bash scripts/build-desktop-v2.sh. GUI target operations through project
ComputerUse only. Use unique dated evidence directories, bind every receipt and
image to current session/run/build, inspect actual images, preserve failed
screenshots. Readiness is not proved by non-white pixels alone. At most 12 s
passive blank-screen waiting in the complete 120 s loop, no blind input.

Keep requested diagnostic logs and validation evidence locally and out of Git;
clean only task-created unrequested temporaries. Final gate: diff --check,
status, log -1, HEAD == origin/main. Report tests, screenshots, commit/push,
actual loop result and exact residual risk; no false closure.

### Slice 1 verification

Append-only metadata diagnostics implemented; no recovery or identity gate
semantics changed. Focused six Go packages plus computerdiag pass; native
fake-platform suite 257 assertions passes. Diagnostics now identify helper
command/request IDs and native identity/geometry gates. New native log:
`/tmp/swift-mismatch.log`. Real current-build acceptance still pending.

### Slice 2 verification

Acceptance now uses unique default directories and rejects nonempty directories;
action timestamps must belong to the current run. Structured launch identity is
read even when tool output is truncated. Launch receipt is retained if the bound
observe fails. Build entry adds a signed-resource source/byte attestation;
acceptance CLI rejects stale/dirty/tampered builds. Added observe-only gate.
Tests: Python 20 tests, ComputerUse tool tests; shell syntax check. Native test
script executable-bit change from slice 1 is restored to the repository pattern.
Full model closure remains pending; no old mixed batch is counted as a pass.

Build-attestation follow-up: the outer codesign seal changes the main executable
signature bytes, so embedding its full SHA in its own signed resources is
circular. Use the immutable Go build ID in the resource manifest, helper SHA,
and strict outer/nested codesign verification; measure final executable SHA in
the run evidence. Initial gate correctly rejected this inconsistency before
any target input. Follow-up tested; latest build rerun required.

### Fresh desktop diagnosis / slice 3

Current attested 6cf1d91 cold run saved under
`desktop-v2/build/validation/20261008/stability-observe-6cf1d91-01`.
Correct actual route: jiuan-responses-gpt-5.6sol / gpt-6-sol / high, image/png.
Native launch discovered two same-app windows and selected a frontmost blank
full-screen window; the body screenshot is visibly white, not a pass. Later
wait failed before backend dispatch. Persisted server events discarded structured
identity and truncated the nested launch window. Fixes: all supplied model
session IDs are rebound to trusted current session; shared ResultMetadata is
used in both runtime and server event paths, with titles/private fields excluded.
Focused/full computeruse/tool/agentruntime/server tests and Python 20 tests pass.
This slice is separate from pending native readiness and dispatch recovery.

### Slice 4 — native readiness / snapshot consistency

Generic content-body variation sampling excludes titlebar/shadows and normalizes
pixel format/alpha. A ready candidate must keep the same stable identity/frame
for 100 ms; ambiguous startup inventory waits within the existing launch budget
rather than failing immediately. Capture returns the actual geometry/window
snapshot, which is saved for next-input authorization; motion after observation
still rejects input. No identity loosening. Python blank guard ignores chrome
and preserves failed/white images as invalid evidence instead of dropping them.
Native 267 assertions and Python 20 tests pass. Actual latest-build cold/native
and model acceptance pending; older white image remains failed evidence.

### Slice 5 — dispatch-stage recovery

Native/Go receipts now carry dispatch_state independently of screenshot
verification. Fully acknowledged input with a failed after-image stays executed
with verification unknown; only observation is refreshed, never the input.
Partial/unknown input is terminal even if focus_changed or the helper is alive;
Session latches uncertainty so new action IDs and Resume cannot replay it.
Pause/Stop revocation and permission-required explicit recovery remain gates.
Public screenshot_failed/input_uncertain codes are retained; read-only capture
retries are bounded and generation/identity/permission checked.
Verification: focused Go tests, broader query/agentruntime/server packages,
race computeruse/macos, native 266 assertions (after-capture expectations now
correctly distinguish acknowledged input from partial input). Actual native
and production-model closure remain pending; no simulated pass is substituted.

### Scripted acceptance support

The opt-in computeracceptance build now exposes registered target launch through
the same active Controller (forged session/unregistered target are rejected),
so native smoke/cold-repeat checks do not spend model turns or use external
GUI automation. This adapter remains disabled in ordinary builds. Build evidence
also attests the embedded model-service bytes, not only desktop/helper.
Tagged desktop tests and Python 20 tests pass. Next highest-priority gate:
latest tagged .app, real cold target capture and a non-WorkBuddy fixture, then
actual requested-provider/model 120-second closure. Deferred gates unchanged.

Launch selection follow-up: readiness now filters the inventory before selection;
a unique rendered main window may be raised during launch when a blank splash
is frontmost. Exact frontmost identity must then stabilize before binding; no
activation is added to input/evidence checks. Multiple ready ambiguous windows
still fail closed within the existing budget. Native fake regression added.

## Current phase closure / remaining queue — 2026-10-08

Code slices committed/pushed through 950818d; final malformed-dispatch/privacy
and acceptance-gate follow-up tested below. Full `go test ./...` passes. Earlier
full-suite runs exposed a stale CLI Stop test expecting the removed observe_failed
label (updated to the stable public error contract), and one unrelated default
agent-eval timing failure (focused repeated run passes; final all-package run
passes). ComputerUse focused/race suites, tagged desktop tests, Python 22 tests,
and native 267 assertions pass. Latest tagged desktop build and deep/strict
local signature verification pass. No release-signing claim.

Concrete external blocker: native Computer Use returned “The Mac is locked and
automatic unlock could not unlock it” when exiting the old app for cold-start
acceptance. User was asked to unlock manually. Do not bypass the lock, type an
unlock credential, or claim actual target/control/model acceptance.

Remaining in descending priority:
1. **Native scripted acceptance (blocked on unlock):** cold-start the current
   attested app; repeated target observe/launch/bind/content screenshots + Stop;
   also Calculator fixture and actual native Pause/Resume/Stop clicks/screenshots.
2. **Autonomous model acceptance (depends on #1):** actual requested provider,
   gpt-6-sol/high, image/png; launch→ready→type 1+1=2→send exactly once→reply
   screenshot→Stop within 120 seconds. New output directory each run. No pass yet.
3. **Regression-driven implementation:** act only on fresh current-build evidence
   if the native/model gates expose another failure; preserve failing images,
   known/unknown dispatch semantics and receipt identities.
4. **Deferred unchanged:** second physical display and formal signing/distribution;
   optional release artifacts remain externally blocked.

Existing real white screenshot in stability-observe-6cf1d91-01 remains failed
historical evidence, not a passing screenshot. All requested /tmp diagnostic
logs retained. Pre-existing scripts/__pycache__/ left untouched and uncommitted.


### Resumed cold-start investigation — 2026-10-08

- Resumed target thread `01a1194a-9317-75b3-b88b-3823b44a8577`; its last user
  message was “已解锁，继续”. Pulled main (d789c13); only pre-existing generated
  `scripts/__pycache__/` is dirty and remains excluded.
- Current attested tagged app built and signed. Desktop is unlocked and host TCC
  reports capture/input approved. Cold native launch returned the real rendered
  WorkBuddy window in 4.59 s; first bound observe failed. Fresh diagnostics prove
  launch used session `host`, observe used the actual controller session, and
  Swift rejected `stale_observation`. This is not a capture/permission guess.
- Minimal architecture correction: BackendLauncher receives the authorized
  session explicitly from Controller; never infer it from a prior observation
  or fall back to `host`. Native cross-session checks remain unchanged.
- Intended/current-task files: computeruse/backend.go, controller.go,
  targets_test.go; macos/backend.go, backend_test.go; this ledger. New cold-start
  regression failed before the fix. Missing session is rejected before IPC;
  forged Controller session never reaches the launcher. Focused six-package
  tests, ComputerUse/macOS race tests, full `go test ./...`, tagged desktop
  tests and Python 22 tests all pass. Commit/push precede rebuild and fresh
  native/model acceptance.
- Evidence (local ignored):
  `desktop-v2/build/validation/20261008/continuation-d789c13/` (build identity,
  initial state, build log) and `continuation-native-01/` (launch, failure
  diagnostics, paused snapshot, Stop). No failed observation counted as pass.
- Remaining order unchanged: repeated cold target captures + non-WorkBuddy
  fixture + native controls; then requested-model 120 s loop. Old lock blocker
  is cleared. Second physical display/formal signing remain deferred.


### Resumed desktop control discovery slice

- e45454d is committed/pushed. Current rebuilt native host cold-starts WorkBuddy
  and captures rendered content successfully. `continuation-native-e45454d-02/`
  holds three same-session/window/PID/bundle captures (0.138/0.128/0.251 s), a
  3.287 s launch and Stop. Calculator launch (1.075 s) and two 396x700 captures
  are in `continuation-calculator-e45454d-01/`. Actual PNGs inspected. The -01
  WorkBuddy capture succeeded, but the local validation client used the wrong
  metadata path and aborted; retained as harness failure, not product pass.
- Current architecture already intentionally disabled the independent NSPanel
  in c77d5d7 (“keep computer use overlay inside desktop window”). Do not undo
  this product choice to satisfy historical NSPanel queue text. Current desktop
  control acceptance targets actual clicks/screenshots inside the Wails window;
  disabled NSPanel acceptance remains unclaimed.
- Fresh native screenshot showed stale “stopped” while host's exact-session
  snapshot was ready. Root cause: useComputerSession stopped discovery forever
  as soon as any session ID existed, even after terminal Stop/failure.
- Intended files: useComputerSession.ts and its tests, this ledger. Restart
  read-only discovery only when idle/terminal; never resurrect same terminal ID,
  resume Pause/Stop, Observe as a fallback, or retain old images/receipts/approval
  when binding a replacement. Generation checks reject races with local starts.
- Regression tests fail before fix; focused UI tests and typecheck pass. Full
  web suite passes: 79 files / 793 tests. Commit/push, fresh build and native
  UI click acceptance next. Model
  120 s acceptance follows native controls and a repeated cold observation.
