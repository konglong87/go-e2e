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


### Current desktop controls / exact cold-start evidence guard

- e5b389b committed/pushed; full web 79 files / 793 tests and typecheck passed.
  Current attested desktop build used for actual CUA clicks on Pause, Resume and
  Stop. Exact host session snapshots confirm paused → needs_observation → stopped.
  Evidence: `continuation-ui-e5b389b/02-paused.png`, `03-resumed.png`,
  `04-stopped.png` and corresponding same-session JSON. This is the current Wails
  in-window UI, not disabled standalone NSPanel acceptance. A stale CUA index
  briefly opened the app menu; cancelled, freshly reindexed and never counted
  as a Pause pass. Successful controls followed with actual screenshots.
- Cold-start auditing found fixture executable is `Electron`, not its display
  name `WorkBuddy`. Earlier local pgrep-by-display-name preflights cannot prove
  cold state. Withdraw those cold labels (rendered captures/binding remain real);
  do not combine them with a later run to claim closure.
- Intended files: acceptance Python wrapper/tests and this ledger. Preflight now
  reads public Info.plist metadata, verifies the registered bundle, matches the
  exact primary executable path in `ps pid,comm` only, and rejects a warm target
  before model-session creation. No environment/auth/config reads. Python 24
  tests pass. Wrapper prompt also requires replacing any pre-existing draft
  before the one fresh type action; old draft is never proof of new input.
- One rapid relaunch after fixture Quit returned launch_timeout and no image;
  preserved in `continuation-cold-e5b389b-02/`. Exact process exit must be confirmed
  before rerun; no timeout extension or blind input. Fresh independent batches
  and real image inspection still required.
- Next highest-priority gates: commit/push + rebuild; two cold observe-only
  native batches with exact process preflight; requested gpt-6-sol/high real
  launch/type/send once/reply/Stop <=120 s. Deferred gates unchanged.


### Exact cold-start gates passed / model-wrapper discovery follow-up

- Current 6e1365a attested app: two exact cold native batches passed, starting
  with no primary executable and launching distinct PID/window pairs
  (48855/27610 and 49436/27668). Each saved three same-session, same-target PNGs
  and Stop. Launch 4.385/3.306 s. Actual ready images inspected; all six assets
  retained under `continuation-cold-6e1365a-{01,02}/`. Separate project-ComputerUse
  cleanup sessions Quit the fixture and confirm exact process termination.
- 120 s model attempt stopped before any model call: wrapper process discovery
  incorrectly required the complete host command line to end in its executable;
  the native acceptance socket arguments made the valid host invisible.
- Intended files: wrapper, regression tests, ledger. Match the executable token,
  not the full command suffix; preserve existing workspace/server-parent gates.
  Synthetic-token regression fails before fix; Python 25 tests pass after fix.
  No auth token is emitted or persisted; no credential/config files accessed.
- Next: commit/push, rebuild the current attested app, then actual requested-model
  120 s loop (fresh directory). Native/cold successes above remain run-specific,
  never merged into the model action chain. Hardware/signing deferrals unchanged.


### RFC3339Nano / complete sanitized receipt follow-up

- Actual requested-provider/model/high run on attested 8c0aae9 returned completed
  in 93.887 s, with observe → input-focus click → select-all hotkey → type once
  → send click once → wait → Stop. Wrapper rejected evidence because this
  Mac's older Python cannot parse legal Go RFC3339Nano fractions of 4/5 digits
  (`.39121Z`, `.8689Z`). It was a parsing false rejection, not old timestamps.
- Preserve original failure/timing files. A read-only post-run image audit
  downloaded the original six asset IDs into `continuation-model-8c0aae9-01/
  post-run-image-audit/`. The wait screenshot visibly shows this run's newly
  created task (19 vs initial 18), `1+1=2` and real response `2 ✅`. This audit
  does not rewrite the failed batch as passed or replace a fresh acceptance run.
- Intended files: Python wrapper/tests, computeruse/result_metadata.go and its
  tests, this ledger. Normalize valid Nano fractions to Python microseconds
  for comparison only; retain original timestamps and all isolation gates.
  Preserve exact cold preflight instead of overwriting it with first-observe
  metadata. Reuse ResultMetadata before text truncation to retain a sanitized
  full action receipt (IDs, dispatch, verification, duration, completion and
  window identity); exclude private errors, summaries, fingerprint/media URIs
  and titles. Python retains these structured receipts and launch evidence.
- Atomic timing comes from actual LaunchReceipt duration/completion; binding
  remains included in launch and is not invented as an independently measured
  operation. Separate first-capture readiness timing and stage notes are saved.
- Timestamp regression fails before fix; Python 28 tests (including mixed native
  session/stale completion rejection), focused Go, full `go test ./...`, tagged
  desktop, ComputerUse/macOS race and native 267 assertions pass. Fixture Quit
  and exact process exit confirmed through project ComputerUse. Commit/push,
  rebuild and fresh requested-model 120 s loop remain the next gates.


### Legal maximum wait exposes shared-deadline defect

- 6dcd58c current-build model run -02 completed in 74.401 s, but failed acceptance:
  legal `wait(duration_ms=10000)` consumed the entire 10 s Controller/RPC deadline,
  leaving no capture/receipt time. RPC timed out, helper became unavailable and
  subsequent Stop had control_failed. Preserve failed batch/screenshots; no replay.
- Receipt metadata follow-up: empty success error codes must remain empty, not
  normalize to action_failed. Regression and focused Go gate pass before commit.
- Next architecture slice: retain 10 s operation-duration validation but separate
  a fixed 2 s evidence grace in Controller + macOS execute RPC timeout. Reuse a
  platform-neutral timeout policy, apply to wait/drag only, honor shorter caller
  deadlines/cancellation/Stop. This is <=12 s bounded per operation, not any change
  to the business run budget or the 120 s acceptance deadline. Tests must inspect
  both Controller and native envelope deadlines without sleeping ten seconds.
- New run -01 started its wrapper before the asynchronous local service child
  existed; no model call occurred. -02 started only after read-only child readiness
  was confirmed. Startup failures remain separate and are not mixed into evidence.


### Bounded operation/evidence deadline separation

- Implemented a shared platform-neutral ActionExecutionTimeout policy: legal
  wait/drag duration up to 10 s plus fixed 2 s evidence grace, only when longer
  than the baseline. Invalid durations and other kinds never expand deadlines.
  Earlier caller deadlines, cancellation, user Pause/Stop and total run budgets
  remain authoritative. macOS execution envelopes and broker leases use this
  same policy; ordinary observation/control RPC deadlines are unchanged.
- Intended/completed files: computeruse/model.go, controller.go,
  computeruse_test.go; macos/backend.go, backend_test.go; this ledger. Both the
  Controller deadline test and native-envelope fixture failed before fix at
  9.999 s / closed response; pass with <=12 s now. Earlier caller deadline and
  kind/invalid-duration boundary regressions pass without long sleeps.
- Focused six packages, full `go test ./...`, ComputerUse/macOS race, tagged
  desktop tests pass. No native input was executed by these fake tests.
- Next: commit/push; fresh tagged app; real 10000 ms wait + after-image + Stop
  on Calculator (no model), then isolated cold requested-model 120 s closure.
  Unknown input recovery/replay rules unchanged. Deferred gates unchanged.
