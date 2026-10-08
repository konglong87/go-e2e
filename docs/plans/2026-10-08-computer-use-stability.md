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
| P0-2 | Fresh-run evidence isolation; scripts/computer-use-120s-acceptance.py and tests | Reject mixed time/session/build evidence and existing output dir; preserve launch receipt on observation failure | implemented; tests below; commit/push pending |
| P0-3 | Diagnose real launch/observe; bounded unique-window stabilization and capture snapshot consistency | Fresh build identity + repeated cold observe-only, screenshots and Stop | depends P0-1/2 |
| P0-4 | Dispatch-stage recovery contract across Swift/Go/controller/session | Native partial/complete/before-input faults, Go contract tests, unknown replay prevention, Pause/Stop/permission races | pending |
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
