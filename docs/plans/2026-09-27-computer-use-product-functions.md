# Computer Use functional parity — implementation ledger

## Current ordered queue — 2026-09-28

This section is the current execution order. Older sections below are historical
phase evidence, not an assertion that their pending statuses remain current.
After **every phase**: verify its actual acceptance boundary, reorder all remaining
items here, commit/push the verified slice, then take the highest-priority
unblocked item. Record blockers instead of silently skipping requirements.

| Order | Remaining item | Next acceptance gate |
|---|---|---|
| P0-1 | Non-drag helper-crash input release | Implement host-owned bounded press/release batches for click, double-click pairs, key/hotkey and each Unicode scalar; preserve per-operation cancellation/focus checks. Verify deterministic helper death, balanced native events, no held keys/buttons, unknown receipts and no replay. Drag crash-release evidence does not cover these actions. |
| P0-2 | WorkBuddy autonomous model loop | Fresh built app and authenticated session-control run: automatic session start, launch WorkBuddy, visibly navigate, click New Task, inspect fresh screenshot, Stop. Require actual tool trace and visual evidence, with no parent GUI intervention during the model run. Tasks 84/85 did not complete this gate. |
| P0-3 | Host permission revoke/regrant lifecycle | Real host-bound revoke, dispatched Observe/Execute failures, recovery (restart if required), fresh capabilities and observation, no old input replay. A disabled UI or ineffective reset is insufficient. |
| P1-1 | Lost Ensure ACK recovery | Bind startup to an owner/query-attempt token and cancel/query that exact attempt; test permanent ACK loss without guessing another query's session ID. Same-query Stop fix alone does not close this boundary. |
| P1-2 | Auto-created session visibility in desktop UI | Read-only authoritative discovery/status and exact-session controls; verify a model-created session and its Stop state in the real desktop UI. |
| P1-3 | Multi-display software coverage without second hardware | Audit/add negative-origin, mixed-DPI, display routing and topology-change tests. Separate mock/virtual results from physical hardware acceptance. |
| P1-4 | Final functional regression and evidence audit | Rebuild desktop, run affected Go/race and native/UI suites, inspect screenshots, reconcile requirements with actual evidence, remove unrequested temporary outputs. Include development-build restart/upgrade behavior without claiming signed clean-install acceptance. |
| Deferred | Second physical display | No second physical monitor available; do not mark hardware acceptance passed. |
| Deferred LAST | Formal signing and distribution | Developer ID, notarization, signed fresh-install/upgrade and release delivery are outside this functional phase. |

## Phase update — P0-1 real host acceptance — 2026-09-28

The rebuilt tagged Wails app (`computeracceptance`) was exercised through the
actual host acceptance socket with a real Chrome window and the isolated trusted
event fixture. The run used the new host-owned batch path, not the old helper
local CGEvent path.

Real evidence under
`desktop-v2/build/validation/20260928/p0-1-real-acceptance/evidence-fix/`:

- trusted left click, double-click, right-click and move;
- trusted drag with source/drop endpoints;
- trusted scroll;
- focus input, Unicode text `go-e2e 中文🙂`, ⌘A selection, replacement,
  backspace and arrow-left;
- all receipts were `executed`; fixture event logs reported `isTrusted=true`,
  `client dropped/uncertain=0`, and final screenshot shows `验收AB` with the
  caret at the expected position;
- key-event inspection proves each ⌘ chord ends with `MetaLeft` keyup and
  `meta=false`; no incorrect `ControlLeft` release remains. This caught and
  fixed the host C batch modifier-index bug in `b6e1e83`.

This closes the real functional input matrix for click/type/key/hotkey on the
current single-screen build. It does **not** yet close the entire P0-1 safety
gate: real desktop helper-SIGKILL during a non-drag batch still needs a
purpose-built deterministic fault-injection run. The Go backend test already
covers deterministic `broker-batch-sigkill` with a host fake driver; that is
strong protocol evidence but not a substitute for the production host/helper
fault-injection evidence.

Priority reorder after this phase:

1. **P0-1 remaining safety gate:** production helper SIGKILL during a real
   non-drag batch, then confirm unknown receipt, no replay, no stuck modifier or
   mouse button, fresh session recovery, trusted fixture state and screenshots.
2. **P0-2:** fresh authenticated `gpt-6-sol` autonomous WorkBuddy loop using
   this rebuilt helper; require launch, visible navigation, New Task click,
   fresh after-observation and Stop with no parent GUI input during the run.
3. **P0-3:** real host-bound Screen Recording/Accessibility revoke and regrant.
4. **P1-1:** exact startup-attempt token for permanent Ensure ACK loss.
5. **P1-2:** authoritative model-created session discovery in desktop UI.
6. **P1-3:** virtual/mocked multi-display topology and mixed-DPI coverage.
7. **P1-4:** final functional regression/evidence audit.
8. Deferred: physical second display and formal signing/distribution.

## Phase update — P0-3 permission lifecycle audit — 2026-09-29

The restart-bound CLI revoke attempt was re-run against the current tagged host:

1. stopped the desktop host/helper;
2. ran `tccutil reset ScreenCapture com.wails.go-e2e` and
   `tccutil reset Accessibility com.wails.go-e2e` successfully;
3. launched a fresh host/helper with a new acceptance socket;
4. queried fresh capabilities.

The fresh host still reported `capture_readiness=ready`,
`input_readiness=ready`, and `permission_state=approved`. This is **negative
lifecycle evidence, not a pass**. It confirms that a successful `tccutil reset`
exit code is insufficient on this development machine and that no code or
report may claim permissions were revoked. The current host remains stopped
again after the audit.

Next gate requires an action-time user-authorized System Settings toggle for
both Screen Recording and Accessibility, followed by: fresh host launch,
failed capabilities/observe/execute evidence, regrant, host/helper restart if
required, fresh capabilities and observe, and proof that no old action replayed.

Priority reorder after this phase:

1. **P0-3 continuation:** user-authorized real TCC toggle/regrant lifecycle;
   do not substitute `tccutil` output or a disabled UI.
2. **P1-1:** exact startup-attempt token for permanent Ensure ACK loss.
3. **P1-2:** authoritative model-created session discovery in desktop UI.
4. **P1-3:** virtual/mocked multi-display topology and mixed-DPI coverage.
5. **P1-4:** final functional regression/evidence audit.
6. Deferred: physical second display and formal signing/distribution.

## Phase closure — P0-2 autonomous WorkBuddy model loop — 2026-09-29

P0-2 is now **closed and passed** on the rebuilt tagged Wails app using the
normal authenticated session-control API and `jiuan-responses-gpt-5.6sol` with
model `gpt-6-sol`. No parent GUI action was used during the successful model
run.

Evidence:

- Successful run: `run_id=87`, session ref
  `tenant:computer-live-5142b8da514744a784404c22f96756e9`.
- Sanitized summary:
  `desktop-v2/build/validation/20260929/model-autonomy-gpt6-retry/autonomy-summary.json`.
- Full private transcript copied to the ignored validation directory as
  `transcript.jsonl`; raw model evidence remains outside Git.
- The ComputerUse trace contains fresh screenshots/observations with active
  window `WorkBuddy`, bundle `com.workbuddy.workbuddy`, PID `23941`, including
  observation IDs 23, 27 and 31, followed by an explicit Stop acknowledgement.
- The model reported and the trace supports: Spotlight launch, WorkBuddy
  frontmost confirmation, 助理 navigation, 新建任务 navigation to an empty
  composer, no assistant run, no task content, no submit/send/login/settings or
  points changes.
- The first attempt in the same phase was correctly recorded as **failed**:
  it clicked Claude instead of WorkBuddy and stopped without continuing. The
  retry prompt tightened launch disambiguation and then passed; this is kept as
  negative evidence, not hidden.

Priority reorder after this phase:

1. **P0-3:** real host-bound Screen Recording/Accessibility revoke and regrant,
   including failed observe/execute, recovery, fresh capabilities/observation,
   no replay, helper/host restart boundaries.
2. **P1-1:** exact startup-attempt token for permanent Ensure ACK loss.
3. **P1-2:** authoritative model-created session discovery in desktop UI.
4. **P1-3:** virtual/mocked multi-display topology and mixed-DPI coverage.
5. **P1-4:** final functional regression/evidence audit.
6. Deferred: physical second display and formal signing/distribution.

## Phase closure — P0-1 non-drag host-owned input — 2026-09-28

P0-1 is now **closed for the single-screen functional scope**. In addition to
the real trusted-input matrix recorded above, a deterministic production
host/helper fault-injection run was completed with the tagged Wails app:

- The opt-in validation hold was propagated through the host's deliberate
  helper environment allowlist; normal environments remain unchanged.
- The helper was verified by exact host PID/path and exact helper PID/parent
  path. The execute future was still pending after the batch request had been
  sent and the helper environment contained the hold hook.
- The helper was SIGKILLed while a real Unicode batch was in flight. The host
  had already committed the balanced key sequence. The receipt was
  `outcome=unknown`, `verification=unknown`; the next observe was rejected and
  the session paused, with no action replay.
- The real fixture recorded the trusted `Z` keydown/beforeinput/input/keyup and
  final state `验收ABZ`, proving the host-owned batch completed physically even
  though the helper died before its ACK reached the caller. Evidence:
  `desktop-v2/build/validation/20260928/p0-1-real-acceptance/evidence-crash-type/crash-type-result.json`
  and `crash-type-fixture-after.json`.

P0-1 delivery commits:

- `9a9dabc` host-owned bounded batches;
- `b6e1e83` correct modifier-release mapping, caught by the real fixture;
- `6ec8d17` deterministic validation hold hook;
- `3450d7d` explicit helper environment propagation for that local validation.

The test-only hold is an acceptance instrument, not a production default, and
formal signing/distribution remains deferred.

Priority reorder after P0-1 closure:

1. **P0-2:** fresh authenticated `gpt-6-sol` autonomous WorkBuddy loop using
   the rebuilt helper; require automatic startup, WorkBuddy launch, visible
   navigation, New Task click, fresh after-observation and Stop, with no parent
   GUI input during the model run. Previous Tasks 84/85 remain non-passing.
2. **P0-3:** real host-bound Screen Recording/Accessibility revoke and regrant.
3. **P1-1:** exact startup-attempt token for permanent Ensure ACK loss.
4. **P1-2:** authoritative model-created session discovery in desktop UI.
5. **P1-3:** virtual/mocked multi-display topology and mixed-DPI coverage.
6. **P1-4:** final functional regression/evidence audit.
7. Deferred: physical second display and formal signing/distribution.

## Phase update — P0-1 bounded non-drag input ownership — 2026-09-28

Implementation and deterministic fault-injection tests are complete for the
host-owned bounded input path; this is **not yet a claim of full real-desktop
P0-1 acceptance**. The host now precomputes one balanced batch for each click
pair, each Unicode scalar, and each key/hotkey. The inherited helper channel
carries only an unguessable token and sequence; it carries no coordinates,
text, key code, or arbitrary event list. The host allocates and posts the full
balanced CGEvent sequence synchronously, so helper SIGKILL cannot leave a
partially posted click or key chord. Drag keeps its separate lease protocol.

Evidence completed:

- Go batch-plan tests cover observed negative/global geometry, left/right click,
  double-click pair count, Unicode scalar/supplementary scalar, key, hotkey,
  duplicate modifiers, bounds and invalid UTF-8.
- Go broker/backend tests cover normal click/double-click/right-click/type/key/
  hotkey batches and deterministic helper SIGKILL after the first batch. The
  normal path records the exact expected number of balanced host commits; the
  crash path returns `unknown` and never replays the already committed batch.
- `go test -race ./internal/computerbackend/macos` passed.
- Native helper build passed. `bash native/macos/tests/platform.sh` passed
  469 assertions, including batch framing, sequence, no event-data leakage,
  inactive rejection, lost-ACK uncertainty and no reuse. `bash
  native/macos/tests/run.sh` passed 240 fake-platform safety assertions.
- No real desktop input was posted by these tests. They prove host ownership,
  bounded protocol behavior and fault injection, not real fixture event logs or
  screenshots after the new path is bundled into the desktop app.

Priority reorder after this phase:

1. **P0-1 acceptance follow-up:** build the current desktop app and perform a
   real isolated fixture run for click, double-click, right-click, Unicode,
   key and hotkey, collecting trusted event logs, before/after screenshots,
   receipts, and a helper-crash run. Do not claim P0-1 complete until this
   evidence exists.
2. **P0-2:** fresh authenticated `gpt-6-sol` WorkBuddy autonomous loop after
   the new helper is bundled; require actual observe/click/new-task/after-image/
   Stop trace and no parent GUI intervention.
3. **P0-3:** real host-bound permission revoke/regrant lifecycle.
4. **P1-1:** exact startup-attempt token for permanent Ensure ACK loss.
5. **P1-2:** authoritative model-created session discovery in desktop UI.
6. **P1-3:** virtual/mocked multi-display topology and mixed-DPI coverage.
7. **P1-4:** final functional regression/evidence audit.
8. Deferred: physical second display and formal signing/distribution.

### Resume/change ledger

- Baseline: `d1ff501`, clean working tree, `git pull --ff-only origin main`
  confirmed up to date. No pre-existing dirty files.
- This documentation slice owns only `AGENTS.md` and this ledger. It persists
  the user's phase-priority rule; it does not implement the pending input broker.
- Non-drag batch delegation returned a design only, with no source changes.
  Protocol remains subject to review: do not expand the existing public hotkey
  limits or batch an entire text action merely to fit a proposed wire schema.
- Baseline tests passed: `go test ./internal/computerbackend/macos
  ./internal/computerbridge ./internal/computeruse`.
- No new native UI acceptance or screenshots in this documentation slice.
  Delivery is recorded by the Git commit containing this section and its push.


## Scope and delivery order

Functional work first; Developer ID, notarization, release distribution and
signed clean-install/upgrade acceptance are explicitly deferred until last.
No claim of complete Codex parity is made from a single successful task.

## Architecture

- Keep model tools → trusted conversation service → controller → macOS backend
  → native helper. No unauthenticated input endpoint or implicit model approval.
- Targets carry display/window identity, current geometry and observation expiry.
  All input coordinates refer to the exact observed image, not a UI preview.
- Multi-step input owns a bounded press/release lifetime. Cancellation cleanup
  must finish before Stop acknowledges; helper death requires an independent
  release guard (a defer alone cannot handle SIGKILL).
- Automatic session startup must consume user-granted scope; model output is
  never authority to approve itself. First OS consent remains user-controlled.
- Prefer the existing macOS/Go interfaces and Apple native APIs. Keep target
  discovery, coordinate conversion, input lifetime and session policy separate.

## Existing work and correction gate

Initial worktree was clean at 8825bc3. All subsequent source edits in this run
are task-owned. No pre-existing user source edits have been observed.

| Slice | State | Verification / delivery |
|---|---|---|
| Drag contract and initial helper path | pushed b141f86; NOT native accepted | Go suites and 161 fake Swift assertions; safety review found cleanup and capability gaps requiring correction before live input |
| Display inventory and selection foundation | pushed b369cb5; NOT multi-monitor accepted | Go focused tests and 163 fake Swift assertions; physical two-monitor matrix outstanding |
| Input safety correction + regression matrix | in progress | preallocate releases, explicit button binding, strict protocol fields, synchronous Stop cleanup, independent crash release |
| Window inventory/identity and target binding | pushed f6aec45; acceptance targeting plumbing pushed in this slice | native Window Server inventory, owner PID/bundle/title/frame, window activation, window capture geometry, active-window guard, Wails acceptance observe target plumbing, tool/DTO binding; real multi-window Wails acceptance outstanding |
| Window capture/focus and stale geometry | pending | same-app multiple windows, overlap, close/move/minimize and stale IDs |
| Multi-display UX and topology invalidation | foundation pushed b369cb5 | inventory and selected-display capture are implemented/tested; negative origins, mixed DPI, unplug, topology invalidation and real two-monitor acceptance remain |
| Authorized automatic session coordinator | pending | no UI manual start within scope; cross-conversation isolation, revoke and Stop precedence |
| Permission lifecycle recovery | pending | active-session OS revoke, helper/host restart requirement, recovery without replay |
| Native E2E completion | pending | Wails-host-owned actions, trusted fixture events and real before/after screenshots |
| Signing and distribution | deferred LAST | no tag/release or signing changes in functional work |

## Acceptance rule

A disabled Start button only proves UI gating, not that a dispatched request is
rejected. A standalone helper probe is not equivalent to a Wails-host TCC path.
Prior permission report overstates that full gate: rerun actual host observe and
execute rejection plus recovery. Similarly, executed receipts with
verification=not_checked are not proof of the intended effect; require fixture
state/events and screenshot inspection. Preserve requested evidence under
ignored desktop-v2/build/validation/YYYYMMDD; do not commit screenshots, logs,
credentials or user databases. Commit/push each tested coherent source slice.

## Current slice evidence boundary

The native tests now exercise a fake window target and selected secondary display
(165 assertions). This proves protocol/state transitions and coordinate binding,
not real Window Server enumeration, real target activation, or a physical
multi-monitor setup. The next gate is a Wails-host build and a real fixture with
WorkBuddy plus a second app, including screenshots and active-window metadata.

## Real Wails-host window-target acceptance — 2026-09-27

On the current ad-hoc build with the real Wails host acceptance socket:

- Capabilities enumerated 14 on-screen Window Server targets and one display.
- A visible `go-e2e` target was observed by `window_id`, producing a window-only
  screenshot (`2674x1588`) and an `executed` native move receipt whose active
  window remained the requested ID.
- WorkBuddy was then launched cold, its visible main window (`bundle_id`
  `com.workbuddy.workbuddy`, window ID `8792`) was selected from the host's
  capability inventory, and a window-only screenshot (`2704x1688`) was taken.
- A real window-targeted move and a harmless click in the empty WorkBuddy
  composer both returned `outcome=executed`; every receipt retained the same
  WorkBuddy active-window ID and fresh after-image.
- Evidence is retained under ignored
  `desktop-v2/build/validation/20260927/window-target/`, including
  `capabilities-after-workbuddy.json`, `workbuddy-window-result.json`,
  `workbuddy-window-click-result.json`, and their before/after PNGs.

This is a real single-display window-target pass, not proof of multi-monitor,
window-close/move invalidation, multiple same-app window disambiguation, or
model-driven automatic target selection. Those remain open acceptance gates.

## Real drag acceptance — 2026-09-27

- The isolated fixture now includes `drag-source` and `drag-drop-target`; the
  server validates both target IDs and the page only marks `dragCompleted` after
  trusted native `mousedown`/intermediate `mousemove`/`mouseup` events.
- Through the Wails host's real Computer Use acceptance socket, the current
  build ran all 12 fixture cases. The new drag case passed with an executed
  receipt, 29 trusted event records, source endpoint `(266,400)`, drop endpoint
  `(539,400)`, and `dragCompleted=true`.
- Evidence is retained under ignored
  `desktop-v2/build/validation/20260927/drag-fixture/`, especially
  `fixture-drag-before.png`, `fixture-drag-after.png`,
  `fixture-drag-receipt.json`, `fixture-results.json`, and `fixture-run.log`.
- This closes basic real drag execution for the single-display fixture. Pause,
  Stop, focus perturbation, helper crash, window-target drag, and multi-display
  drag remain separate gates.

## Display topology stale-observation guard — 2026-09-27

The native snapshot now records the sorted display-ID topology. Every input
revalidates that topology in addition to the selected display geometry, focus,
and target window. A display add/remove therefore rejects a stale action before
any input post. The fake native matrix now reports 167 assertions; physical
monitor hot-plug and mixed-DPI acceptance remain outstanding.

## Lazy managed-session orchestration — 2026-09-27

The ComputerUse tool may now omit `session_id` for its first `observe` call.
The trusted desktop service implements an optional coordinator that lazily
creates/approves a session for the current managed conversation, using the
already-established ComputerUse permission gate and the Wails host lifetime.
Local preview remains explicit and cannot enter this path because local owners
are rejected. Cross-conversation ownership checks remain enforced.

This removes the model-run dependency on a user clicking “授权本次会话” after
OS permissions are already granted. It does not automate first-time macOS TCC
consent, does not let model output mint permission, and does not yet provide a
runtime task-finalizer that automatically stops every session if the model
fails to call Stop.

## Drag interruption release guard — 2026-09-27

The native fake matrix now covers focus loss after drag mouse-down: the result
is `unknown`/`focus_changed` and the emergency mouse-up is posted before the
helper reports the failure. The matrix is now 170 assertions. Real pause/stop,
permission revoke, and helper-crash drag interruption remain to be run.

## Model-route gate for auto orchestration — 2026-09-27

A real latest-source Wails task was run without starting the Computer Use panel
manually. The configured `gpt-5.6-sol` route failed before any ComputerUse call
with provider `404 model is not found`. A retry with the UI-listed `gpt-5.5`
route failed with the same 404. Therefore the new lazy coordinator is covered
by focused unit tests but has not yet received a live model call in this
environment; no automatic-session pass is claimed until an available
vision-capable route is configured.

## Active-session permission lifecycle UI — 2026-09-27

Readiness refreshes are no longer suppressed merely because a session is active.
Focus/visibility/recheck refreshes can now surface a TCC revoke/restore while
preserving the session's safety controls. The permission guide is also rendered
when an active session reports `permission_state=required`, so the user gets a
recovery path instead of a stale “ready” surface. Native action gates remain
fail-closed; this slice does not silently replay an action after recovery.

Focused frontend verification: typecheck passed; 52 Computer Use hook/workspace
tests passed.

## Latest functional slices — 2026-09-28

### Lazy cold-start bridge wiring

The desktop bridge now exposes a host-owned `ensure` operation in addition to
lookup. The runtime no longer requires an existing approved session before it
registers Computer Use: the first model `observe` may omit `session_id`, the
trusted host coordinator creates/binds the session, and the runtime tracks the
created ID for bounded cleanup. Existing non-coordinating test adapters retain
the explicit lookup compatibility path.

Focused Go tests cover the wire command, owner authorization, lazy setup without
an existing-session lookup, and cleanup binding. The production host bridge is
still subject to provider/image-route gating; this change does not bypass that
operator assertion.

### Dynamic targets and stale-window diagnostics

Active sessions refresh dynamic display/window inventory on capabilities/start
refresh instead of retaining the window list from session creation. Native
rejections preserve allowlisted safety codes so stale display/window, focus, and
permission failures can be distinguished without exposing helper text. Stale
window IDs remain fail-closed and are never remapped by title alone.

### Helper crash recovery

When an active helper transport dies, the host retries capabilities once with a
fresh bundled helper. The session is paused and its previous observation is
invalidated before the replacement controller is exposed; recovery requires
explicit Resume plus a fresh Observe, and no action is replayed.

Real latest-source Wails acceptance evidence:

```text
desktop-v2/build/validation/20260928/real-latest/crash-before.png
desktop-v2/build/validation/20260928/real-latest/helper-before-kill.txt
desktop-v2/build/validation/20260928/real-latest/crash-capabilities-after.json
desktop-v2/build/validation/20260928/real-latest/crash-snapshot-after.json
desktop-v2/build/validation/20260928/real-latest/crash-recovered-observe.png
```

The helper PID changed after SIGKILL; the session reported `paused`, then
Resume + a new observation succeeded. This is helper recovery evidence, not a
claim of SIGKILL-time system mouse-up.

### Permission lifecycle boundary

A live `tccutil reset Accessibility/ScreenCapture` request was issued against
the current host bundle. macOS kept the running process's readiness approved;
this is recorded as a deferred live-revocation result rather than a pass. A
restart-bound TCC revoke/restore remains a separate acceptance gate and no
permission is claimed revoked when the native probe still reported approved.

### Effective image-route gate correction — 2026-09-28

The runtime image gate now checks the explicitly asserted effective primary
route instead of requiring every unrelated configured fallback to advertise
image input. This keeps the security boundary (no ComputerUse without an exact
operator assertion) while avoiding a text-only fallback disabling a selected
vision route before the query starts.

The local settings used for the latest provider probe now contain explicit
routes for `jiuan-responses-gpt-5.6sol` with `gpt-5.6-sol`/`gpt-6-sol` and the
primary aliases. A direct Responses request with a 1x1 PNG returned HTTP 200
from the configured Jiuan route. This proves provider image acceptance, not a
completed model-driven desktop run.

Latest model-driven reruns remain uncounted as ComputerUse passes: one deny-mode
run correctly failed closed because the tool was not in the allow list; the
allow-mode rerun was stopped before producing a ComputerUse tool trace. The
native host/bridge acceptance and helper recovery evidence remain independent
passes.

## Scope update and current safety slice — 2026-09-28

The user explicitly deferred physical second-monitor acceptance as well as
formal signing/distribution. Neither blocks this phase; neither is claimed
verified. Single-display window selection, drag cancellation/release, permission
recovery, and automatic session lifecycle remain required.

Worktree at `31840a0` was clean; no pre-existing user edits. Current slice:
- Native target resolution must be read-only during Execute and evidence capture.
  Only an explicit Observe may activate a requested target. Otherwise capture
  can steal focus back after an action/user focus change and conceal the change.
- Bind identity to Window Server ID, process and bundle, not a mutable title.
  Geometry/visibility must remain checked independently before input.
- Regression matrix: identity/geometry replacement, title-only navigation,
  external focus between checks/capture; count activation calls in fakes.
- Verification: native tests, Go controller/backend suites, rebuild Wails app,
  real window-target before/after screenshot and focus-rejection evidence.
- Delivery: pending tests, commit and push; no release tag or distribution work.

Native target-read-only slice results:
- Red/green regression: the new title-navigation test failed before the change;
  203 fake-platform native assertions pass after it. Execute calls activation
  zero times, including focus loss during geometry resolution and after input.
- Go backend/domain/desktop focused suites pass; tagged Wails `.app` built.
- Real host readiness reports approved/ready. Target Observe was attempted while
  the desktop switched Spaces/focus; capture failed and session paused. No input
  was dispatched. This is NOT a real window-target success; that acceptance is
  still pending a stable test window. The test session was explicitly stopped.

### Remaining functional gates (explicit user exclusions applied)

| Requirement | Current evidence / next required gate |
|---|---|
| Unified target/geometry binding | Native snapshot identity and Go metadata validation in progress; reject mismatched target/capture geometry, no cached bounds blending |
| Exact same-app window selection | Public AX exact-window activation and identical-title A/B fixture now pass; see exact-window slice below. Ambiguous matches intentionally fail closed |
| Single-screen drag | Basic trusted-event fixture passed earlier; real Pause/Stop/focus-loss while holding a button and independent SIGKILL release remain open |
| Automatic orchestration | Real run 77 (`live-workbuddy-dock-activation`) recorded WorkBuddy bundle identity plus three clicks and Stop. Final screenshot was inspected; a fresh independently captured intermediate Assistant screenshot is still needed for the strongest navigation evidence |
| Permission lifecycle | Readiness regression now pauses the domain session and restore does not auto-resume. Running-process TCC revocation evidence remains deferred/insufficient, not a pass |
| Target state changes | Native fakes cover stale identity/geometry and focus perturbation; current-source stable real target, close/minimize/move and same-app windows remain acceptance work |
| Physical second monitor | Deferred by user; not needed to finish this phase, not verified |
| Formal signing/distribution | Deferred by user; no release/tag/notarization work in this phase |

### Capture-bound metadata and detached snapshots

The helper now emits the geometry of this capture. The Go backend rejects
missing/malformed geometry, dimensions/scale/display mismatches, mismatched
requested/returned window IDs, missing target frames, and any target Frame that
differs from the capture Bounds. Whole-display capture clears the previous
TargetWindow instead of inheriting it. All nested frame pointers in capabilities,
observations and receipts are detached copies.

Verification: 207 fake-platform native assertions; seven focused Go packages
passed including CLI; five backend/domain/bridge/tool/desktop packages passed
with race detection. The expiry harness also passed 23 offline tests after
adapting its wait budget to the existing 120-second native TTL (actual expiry
still comes from the returned metadata; runtime TTL was not changed).

Latest tagged Wails host capture succeeded after a clean new process launch:
`target-safety/metadata-target-before.png` is a real window-only `2674x1588` PNG;
`metadata-result.json` binds window `10151` to global Bounds `(7,35,1337,794)`
and scale `2`. Evidence is local under
`desktop-v2/build/validation/20260928/target-safety/` (not committed). Screenshot
was visually inspected. The subsequent input scenario found no visible test
window and stopped without dispatch; it is not counted as an input pass. This
supersedes the earlier failed capture attempt, not the still-open input matrix.

Delivery: target validation committed/pushed as `4ea2b42`, expiry harness as
`24a0b69`; capture-bound metadata is the coherent source slice accompanying this
entry. Physical second-monitor and formal release gates remain explicitly out
of this phase; all other open functional gates above remain in scope.

### Exact-window activation slice

Baseline `bb7bdbf` clean; pulled before editing. Architecture: public macOS AX
window inventory + unique geometry/title match within the target PID, explicit
AXRaise, then Window Server ID verification. Ambiguous/missing matches fail
closed; title alone is never identity, and no private AX-to-CG ID API is used.
Activation stays exclusively in Observe; Execute remains read-only. Bound AX
messaging and activation settling; Pause/Stop must interrupt the settle loop.
Write scope: native selector/adapter, Engine activation gate, build/test source
lists; independent agent owns only the two-window AppKit fixture. Verification:
fake selectors and lifecycle cases, tagged Wails build, two identical-title
native windows with click counters and target screenshots. No physical second
monitor or formal release work. Commit/push pending focused/native verification.

Exact-window slice verified on 2026-09-28 (single physical display):
- Public AX adapter matches the selected PID plus geometry/title, rejects
  ambiguity/unreadable candidates, raises the specific AX window, and confirms
  the Window Server ID. Each AX object has a 100ms messaging timeout and checks
  cancellation between reads. Observe settling is bounded and interruptible;
  input/evidence validation never reactivates a window.
- 220 native fake assertions and 436 non-posting platform event assertions pass.
- Tagged Wails build ran the new native AppKit fixture with two identical-title
  windows: A (`10345`) -> B (`10346`) -> A. Counts were `(1,0)`, `(1,1)`, `(2,1)`.
  Each positive click was dispatched by the actual Wails Controller/helper.
- Switching observation to B rejects A's superseded observation. Independently,
  a tester-injected AXRaise to B after observing A caused the old A click to be
  rejected by the native backend; B remained frontmost and counts stayed `(2,1)`.
  The external AXRaise is fault injection, not evidence of model autonomy.
- Real A/B screenshot pixels and counters were inspected. Evidence retained in
  ignored `desktop-v2/build/validation/20260928/window-selection/`: `results.json`,
  `03-A-verified.png`, `02-B-verified.png`, `external-focus-result.json`, and
  `external-focus-B.png`. Stop confirmed in both scenarios.
- Reusable fixture builder: `scripts/build-computer-window-fixture.sh`; runner:
  `scripts/computer-acceptance-windows.py --fixture <absolute state.json>
  --socket <tagged host socket> --output <evidence directory>`.
  Open the fixture `.app` with `--args --output <absolute state.json>` before
  running. The runner scopes every target to the fixture PID and bundle ID.

This closes the previously missing exact-window activation + same-title native
selection gate, not arbitrary-app AX compatibility or all Computer Use gates.
Drag interruption/crash release, move/minimize/close target changes, and full
permission lifecycle remain work. No physical second-screen or release claim.

Delivery/readback: feature `102aeef` pushed to `origin/main`. Rebuilt that exact
source and cold-launched the tagged host; repeated A/B/A acceptance passed with
counts `(3,1) -> (3,2) -> (4,2)`, stale-target rejection and confirmed Stop.
Evidence: `window-selection/head-rerun/results.json` and corresponding verified
PNGs; final A image visually inspected. Six related Go packages pass. The
purpose-built fixture process was stopped and both generated fixture app copies
were removed; requested screenshots/JSON remain. Normal untagged desktop build
is restored after testing; no release artifacts/tags were published.

### Held-button interruption slice (baseline 87600b4, clean, pulled)

Review found Stop acknowledged before the helper's executor drained; the Go
controller then canceled the active RPC, potentially terminating the helper
before drag cleanup. Also emergency release used AppKit bottom-left cursor
coordinates as CG top-left coordinates and allocated its event during failure.
Design: preallocate/reuse the release event in CG coordinates before down;
keep native reader/control responsive but enqueue control acknowledgements after
executor cleanup; retain explicit paused state for Resume+fresh Observe, not
input replay. Close requests bounded Stop before abort. Verify normal drag and
Pause/Stop/focus interruption in isolated AppKit event fixture, counters and
screenshots. SIGKILL needs an independent release authority and stays an explicit
open gate; successful cooperative cleanup is not crash safety.

Held-button slice acceptance on 2026-09-28:
- Native fake safety matrix: 231 assertions; non-posting platform event matrix:
  455 assertions. Focused Go tests and race tests passed for controller, macOS
  backend/native transport, bridge, tool, and Wails host.
- Actual tagged Wails Controller/helper passed eight native fixture cases:
  normal left/right drag, Pause left/right, Stop left/right, and independent
  focus perturbation left/right. Each recorded exactly one down/up pair, up at
  the last posted CG point, fixture pressed=false, and global mouse mask=0.
  Normal cases completed the drop; interrupted receipts remained unknown.
  Pause resumed only with a fresh observation; all cases confirmed Stop.
- Real execution exposed two extra failures before the passing rerun: Go Abort
  prevented Pause recovery, and focus-error Abort prevented Stop confirmation.
  Locally requested, native-acknowledged interruption now keeps the helper for
  explicit Resume; focus failure quarantines it to Stop/Close only. Neither
  exception permits replay or changes an unknown receipt into success.
- Final evidence: ignored `desktop-v2/build/validation/20260928/drag-interruption/final/`
  (`results.json`, normal-left/right, pause, stop, focus before/after PNGs).
  Normal-right and Pause native screenshots were visually inspected: completed
  drop and released-interrupted states respectively. This is scripted native
  acceptance, not model-autonomous drag acceptance. No SIGKILL safety claim.

### Window lifecycle real acceptance slice (baseline 2c1895a)

Previous drag-release slice committed/pushed. Current task owns only the fixture,
a new lifecycle runner, and this ledger. Reuse the existing Window Server
identity/geometry guards and Wails acceptance driver; add an opt-in fixture-only
mutation channel for move/resize/minimize/restore/close, never cross-app input.
Verify stale screenshot clicks reject with unchanged native counters; missing
windows cannot be observed; recover with a fresh target image and real click.
Keep screenshot and event evidence, then commit/push after actual acceptance.

Lifecycle acceptance passed on 2026-09-28 with the actual tagged Wails host:
move, resize, minimize/restore, and close each rejected the old-image click with
unchanged native click counters. Minimized/closed targets could not be observed;
observe did not silently restore them. Recovery used explicit Resume after the
failed capture and fresh observations, then native A counts 1/2/3 and B count 1.
All four Stop acknowledgements passed. Swift6 warnings-as-errors fixture build
and Python syntax checks passed. Evidence: ignored
`desktop-v2/build/validation/20260928/window-lifecycle/final/results.json` plus
before/recovery PNGs. Visually inspected restored A (Clicks: 3) and surviving B
(Clicks: 1). Initial test runner omitted Resume after failed capture; corrected
the runner rather than weakening the existing paused-session safety boundary.
This proves single-display window lifecycle guards, not general AX compatibility.

### Independent helper-crash release (baseline e7f432b)

Architecture review rejected a helper-owned defer or best-effort armed message:
SIGKILL can destroy it, and cross-process armed/post messages cannot be atomic.
Use a narrow host-owned mouse broker for drag down/drag/up, with a preallocated
release event and one active action lease. The helper sends bounded requests on
inherited private pipes; no public socket, arbitrary-script endpoint, or local
down fallback. The live host serializes posting and consumes the release once
on normal up, control revocation, action completion, or helper death. Host TCC
is checked independently before drag; helper ACK loss remains OutcomeUnknown.
Scope is helper death with a live, still-authorized host, not simultaneous host
death or permission revocation during a press. Native control gates and target
checks stay in place; no automatic replay. Go broker/transport worker and local
Swift/client/acceptance work have disjoint write scopes. Verify deterministic
allocation/ACK/revoke/death races, existing eight real drag cases, then left/right
SIGKILL fixture cases with real native events, mask=0, screenshot and fresh-session
recovery. Do not mark accepted until that real Wails path passes.

Independent release acceptance on 2026-09-28:
- Real rebuilt/cold-launched tagged Wails host passed all ten cases: the previous
  normal/Pause/Stop/focus left/right matrix plus left/right helper SIGKILL.
  Crash was injected only after fixture down and nonzero system mask, after
  checking exact helper executable and host parent PID twice. Host stayed alive.
- Crash left/right each recorded exactly down, one dragged, one up. Up matched
  the last actual CG point `(182.18182373046875,466.5)`; global masks `1/2 -> 0`,
  view pressed=false, and no duplicate input. Receipts stayed unknown; old action
  rejected. New sessions with fresh screenshot executed a move and confirmed Stop.
  Dead-helper Stop correctly returned an explicit transport error, not a fake ACK.
- Evidence: ignored `desktop-v2/build/validation/20260928/crash-release/run-1/`
  `results.json`, `crash-left/right-held.png`, `crash-left/right-after.png`, and
  recovery evidence. Right-button Holding and Released/interrupted screenshots
  visually inspected. This is scripted native acceptance, not a model trace.
- Native fake matrix: 234 assertions; platform/client/real-pipe (non-posting)
  matrix: 465 assertions. Go worker passed broker/transport normal/race tests,
  CGO-disabled builds, vet, fake-child SIGKILL, lost ACKs, duplicate releases,
  permission denial, and concurrent terminal paths. Whole-related-package race
  rerun is the final delivery gate.
- A broad race run during desktop compilation exposed an old test assumption:
  its 100ms deadline could expire during 900KB JSON encoding, before dispatch,
  when the production contract correctly keeps the unused helper alive. The
  blocked-write test now waits for dispatch before cancellation, still requires
  MayHaveRun and reader/reaping completion; 20 race repetitions pass.
- Scope: independent drag release with live, authorized host. Simultaneous host
  death, permissions revoked during a held input, and non-drag key/modifier
  crash release are NOT covered by this pass. No second-display/release claim.

Final related-package `go test -race ... -count=1` rerun passed all seven packages
(controller, macOS backend, native transport, bridge, ComputerUse tool, Wails,
CLI). Existing Darwin LC_DYSYMTAB linker warnings remain non-fatal.

### Delivery checkpoint — 2026-09-28

Pushed to origin/main:
- `2c1895a`: cooperative held-button cleanup, Resume after Pause, control ACK barrier.
- `e7f432b`: native move/resize/minimize/restore/close stale-target acceptance.
- `5d53053`: host-owned drag broker and actual helper SIGKILL release acceptance.

Cleanup: stopped the acceptance session, verified global button mask zero,
stopped the fixture by verified PID/path, removed generated fixture bundles and
standalone test executables. Kept requested PNG/JSON evidence. Restored and
launched the normal untagged desktop `.app`; private acceptance sockets and
session files are removed. No formal release/signing work was performed.

Remaining functional acceptance priorities (NOT claimed complete):
1. A latest-source production model trace that autonomously starts ComputerUse,
   selects/opens the intended app/window, performs a bounded task, inspects fresh
   screenshots and stops, without a manually pre-started ComputerUse session.
   Scripted fixture passes do not replace this gate.
2. Host-bound permission lifecycle: demonstrably revoked capture/input, actual
   Observe/Execute failure, regrant/restart as needed, explicit safe recovery
   without replay. The earlier live reset that still reported ready is not a pass.
3. Audit non-drag key/modifier held-input lifetime during helper failure. The new
   independent owner covers drag only; do not generalize its guarantee to every
   input or simultaneous host death.

Physical second-display real-hardware acceptance and formal signed distribution
remain explicitly deferred by the user and are not blockers for this phase.

### Production model loop investigation (baseline 4343f52, clean/pulled)

Created a separate managed desktop session through the UI, selected the existing
jiuan gpt-5.6-sol route and retained allow mode. No ComputerUse session was
manually pre-started. The real model invoked observe with no session_id, received
observe_failed, then invoked Stop successfully. This establishes model/tool
connectivity and lazy start, not screenshot/input acceptance. Diagnose the
host-vs-bridge boundary before fixing; temporary safe error diagnostics are
limited to host Observe and the trusted runtime bridge wrapper. No screenshots,
credentials or provider responses will be committed. WorkBuddy was not operated.

The observation-only diagnostic rerun succeeded and the model called Stop;
the initial observe failure was not reproduced, so no speculative capture fix
or temporary logging is retained. A subsequent full model run (task 80) used
only ComputerUse: observe, Spotlight hotkey, type application name, Enter, wait,
Stop. WorkBuddy cold-launched at 18:16:05 while the preceding image still showed
the launch surface. The passive wait was rejected after focus changed.

Root cause confirmed with a failing native fake test: Engine applied input
preflight to an empty operation plan (wait), incorrectly requiring old focus
even though it posts no event. Scope of fix: skip input preflight only for empty
plans; retain observation identity/expiry/consumption, generation/Stop, geometry,
and stable capture checks. A companion test must still reject a focus change
*during* the screenshot; input focus guards are untouched. Intended source
changes now only Engine.swift + native tests + this ledger. Fresh desktop/model
reproduction and native tests remain required before marking this slice passed.

Passive wait slice verified on 2026-09-28:
- Regression failed before the one-line input-preflight condition and passes
  after it. Native matrix 240 assertions; platform/client matrix 465 assertions;
  all seven related Go packages passed. Desktop rebuilt/cold-launched.
- Through the actual tagged Wails Controller, captured a display, externally
  changed application focus to the isolated native fixture, then executed wait
  against the preceding observation. Outcome executed; after-image identified
  the fixture PID/window; native fixture counters and drag events stayed zero;
  Stop confirmed. Before focus belonged to the parent test controller's software
  cursor surface, not the fixture. No claim that this was model-driven input.
- Evidence: ignored `desktop-v2/build/validation/20260928/passive-wait/`
  `result.json`, `before-controlled-switch.png`, and
  `after-controlled-switch-after.png`; the final native image was inspected.

Model gate remains open: task 81 again failed its first observe and stopped.
Task 82's read-only model observe and Stop succeeded; its final response needed
provider stream retries and eventually completed (it was not cancelled).
Successful screenshots were near 5MiB, but repeated full-display probes did not
reproduce a budget error, including a maximized window. Do not claim a size or
permission root cause. Temporary host/CLI/native size diagnostics are removed.
Parent CUA's Software Cursor appeared as a Window Server active surface during
manual inspection; avoid parent UI intervention during the next autonomous run,
prefer the desktop's normal trusted session-control API to start it, and inspect
only task/transcript state until it finishes. No app-specific focus bypass.

### Vision-safe fallback selection (baseline aed0641, clean/pushed)

Production normal-build task 83 was submitted via the existing authenticated
session-control API, with no parent GUI calls during its run. First observe and
Spotlight hotkey succeeded. Primary gpt-5.6-sol then returned HTTP 200 SSE that
failed JSON decoding before any event; after its retry was exhausted the client
attempted glm-5.1 without an image-input assertion. The task eventually stopped
itself and reported incomplete, with no WorkBuddy click. No restart or duplicate
submission was issued. The same query still owned ComputerUse authority while
client-internal fallback changed routes: the existing primary-only registry gate
is insufficient to constrain that live fallback chain.

Design: keep primary registration gate; before constructing the client for an
enabled ComputerUse query, copy/filter fallbacks to exactly asserted provider +
effective model. Ordinary queries retain all existing fallbacks. Canonical
fallback naming is shared with the client; unnamed original positions must be
frozen in retained copies to avoid ordinal alias changes. Do not persist settings
or weaken image/owner/input gates. Independent worker owns only the new CLI tests;
main owns routing helper/wiring. Verify real local HTTP primary failure after an
observe screenshot, zero calls to unasserted fallback, approved fallback image
receipt, model overrides, immutable config, and ordinary-query compatibility.

Vision fallback verification:
- 17 pure route-table cases and five production newQuerySession/local-HTTP
  scenarios pass. Primary first invokes Observe; its next request demonstrably
  contains the returned PNG. HTTP overloaded errors and SSE errors exercise
  actual failover (ordinary non-retryable 400 would be a false-positive test).
- With the client-construction filter deliberately removed, the undeclared-route
  regression fails: one forbidden fallback request receives the image and the
  query incorrectly succeeds. Restoring the filter passes, including -race.
  Declared fallback positive controls still receive the actual image and finish;
  cleanup Stop occurs exactly once on both success and failure.
- Full config, anthropic transport, and CLI packages pass. The shared canonical
  naming helper preserves general client behavior; ordinary queries are unfiltered.
- A minimal gpt-6-sol probe through the actual production Responses client, with
  fallbacks disabled and a generated 1x1 image, succeeded in 2.81s and returned OK.
  An earlier bare urllib probe returned 403; that is not evidence that the model
  is unavailable, given the successful production-client counterexample. This
  proves connectivity/image acceptance only, not desktop task completion.

### Latest model/readback checkpoint

Normal source `8a7e7c2` cold-launched. Fresh managed session 60 / task 84 used
`gpt-6-sol` at low effort, launched through the normal authenticated API. No
manual ComputerUse start or parent GUI calls occurred during the run. Native
observe/hotkey/type/click succeeded and WorkBuddy cold-launched at 21:45:06 local.
Independent native screenshot `model-autonomy/gpt6-workbuddy-opened.png` (existing
window 10676) was inspected. This proves launch only: the run did NOT reach the
requested New Task click, and the empty landing page is not proof of that click.

The query failed on two HTTP 200 SSE responses with zero decoded events and
`unexpected end of JSON input` / `responses_sse_json_decode`. With the route guard
in place, no unasserted fallback was used. Raw frames are still needed to identify
the decoding cause; no empty-frame/truncation hypothesis is claimed proven.

Additional scope findings: the UI hook lacks read-only discovery of auto-created
session IDs; its idle panel is not authoritative Stop evidence. Audit whether
same-query omitted-session Observe can recreate a user-stopped grant. These
session orchestration items, real permission revoke/regrant, non-drag crash
release, and complete model navigation remain required. Investigation and
next-step boundaries are recorded in `memory/2026-09-28-computer-use-runtime-debug.md`.

Task-owned temporary fixture binaries, diagnostic source, test sockets and probe
source were removed. Normal desktop build is running; requested evidence remains
under ignored validation paths. No formal signing or release was attempted.

### SSE framing and same-query Stop slices (baseline f5ae71a, clean/pulled)

Main investigates Responses SSE framing using the pinned SDK v3.32.0, official
streaming documentation, and a gated synthetic live probe. Capture only field
counts/data lengths/classification, never frame contents, prompts, keys or images.
Do not filter frames until the real failure shape is known; corrupt nonempty JSON
must remain an error. Separate worker owns tracked per-query Ensure and CLI tests
to prevent omitted-session Observe from recreating a user-stopped grant. No new
branch/worktree. Each tested coherent slice will be reviewed and pushed separately.

SSE slice root cause and verification:
- With synthetic generated text plus a one-pixel image (no user data), the live
  gpt-6-sol endpoint sent HTTP 200 SSE whose first block was three raw bytes,
  one comment field, zero data bytes. Pinned SDK v3.32.0 dispatches every blank
  separator even when no data field exists, then JSON-decodes the empty payload.
  The probe failed with `unexpected end of JSON input` and zero semantic events.
- Local regression reproducing exactly `:\n\n` failed before the fix. A private
  Responses-only HTTP body adapter now drops *only blocks without data fields*;
  any explicit empty/whitespace/malformed data remains byte-for-byte and errors.
  It does not replace or globally register an SDK decoder, retry calls, rewrite
  JSON, normalize input, or alter non-SSE/error responses. Memory/aggregate frame
  size bounded to the SDK ceiling. Underlying close/cancel/partial errors persist.
- The adapter wraps outside the raw activity timeout guard: comment traffic
  keeps the network alive even if no semantic event is emitted. Tests cover
  partial reads, CRLF, multiline data, EOF, malformed/empty data, scoped responses,
  bounded comments/data, cancellation and idle activity.
- A second large synthetic live run with the fix saw a comment-only block
  followed by nine valid data blocks, and completed successfully in 13.45s.
  Sanitized evidence in ignored `sse-framing/probe-before.json` and
  `sse-framing/probe-after-large.json` contains counts/lengths only. Raw contents
  were never persisted. Temporary opt-in probe source is removed after use.
- Full race suites passed: anthropic client, CLI, bridge, controller, tool. An
  official documentation fetch was attempted but unavailable; the concrete
  claim rests on pinned SDK code and before/after live + local reproduction,
  not an unverified documentation assertion.

Same-query Stop precedence slice:
- Before-change regressions reproduced implicit re-Ensure after Stop, concurrent
  duplicate startup (24 calls), and retry after uncertain startup. Query-local
  service now consumes exactly one startup attempt, returns the same bound ID,
  coalesces concurrent callers without holding state locks over IPC, keeps errors
  sticky, and checks owner/context even on cache hits. Host Stop remains authority;
  subsequent observe addresses the stopped grant and is denied, not recreated.
- Cleanup closes the query to new startup. A late ACK containing the exact ID is
  cleaned up once; callbacks cannot rebind to another session. Independent queries
  still construct independent trackers and can start under their own lifecycle.
- New query tests passed under -race for 50 repetitions, existing ComputerUse and
  cleanup tests passed, and full five-package race run passed with the SSE slice.
- Explicit open boundary: if the host created a grant but its ACK is permanently
  lost, no exact session ID is available for Stop. No Lookup-based guessing is
  used. A future query/start-attempt binding plus cancellation protocol must close
  that gap; this commit does not claim lost-ACK authority revocation is solved.

### Phase closure — P1-1 exact startup-attempt recovery — 2026-09-29

The lost-Ensure-ACK boundary is now implemented for the production desktop
bridge. A query creates one opaque `cu-attempt-*` token and sends it with the
host Ensure request. The host records the `(tenant, user, conversation,
startup-attempt)` binding only after it has an approved live session. If the
Ensure response is lost, the query asks the host to resolve that exact token;
it never falls back to Lookup and cannot bind another conversation's session.
The bridge validates and echoes the token on both Ensure and exact resolve
responses, and legacy bridges retain their previous Ensure-only compatibility
path.

Verification:

- Runtime regression simulates a host that committed one grant and permanently
  drops the Ensure acknowledgement. Exact resolution recovers the same ID with
  one startup and zero Lookup calls.
- A real Unix-socket bridge integration sends the attempt token over the wire,
  resolves it, and confirms the existing owner/controller remains the only
  target. Focused `desktop-v2`, `internal/computerbridge`, and `internal/cli`
  tests pass (the broad CLI suite still has one pre-existing/flaky agent-eval
  failure unrelated to this slice).

Priority after this phase:

1. **P0-3:** real host-bound Screen Recording/Accessibility revoke and regrant;
   user has now confirmed action-time authorization, so perform the native
   settings transition and failure/recovery evidence next.
2. **P1-2:** authoritative model-created session discovery/status in the
   desktop UI, with exact-session controls and real UI evidence.
3. **P1-3:** software multi-display negative-origin, mixed-DPI, display-routing,
   and topology-change coverage; physical second-display acceptance remains
   deferred.
4. **P1-4:** final functional regression and evidence audit.
5. Deferred LAST: formal signing/distribution and second physical display.

### Phase closure — P1-2 authoritative model-session discovery — 2026-09-29

The desktop UI now exposes a read-only `GetActiveComputerSession` bridge backed
by the same host manager used by the model runtime. It does not initialize a
helper, capture a screenshot, refresh observation freshness, or call Lookup.
Snapshots carry the immutable owner and an explicit `owner_kind`; the frontend
polls the authoritative status only while it has no bound local session.
Model-created sessions become visible in the floating Computer Use workspace,
continue through the existing exact-session status polling, and expose only
Stop in the UI. Observe/Pause/Resume remain disabled for a model-managed grant
so the desktop surface cannot inject an unscoped action into the model loop.

Verification:

- Go bridge contract and read-only snapshot tests pass.
- Web Computer Use tests pass: 83 tests, including discovery of a managed
  session and exact-ID Stop; typecheck and production Vite build pass.
- The frontend retains compatibility with older hosts that do not expose the
  optional discovery method; no fallback to Lookup was added.

Priority after this phase:

1. **P0-3:** finish the real Screen Recording/Accessibility revoke and regrant
   run now that the user has confirmed it. macOS has currently opened an
   authentication sheet for the requested toggle; a Touch ID/password action
   is required at the OS boundary.
2. **P1-3:** software multi-display negative-origin, mixed-DPI, display-routing,
   and topology-change coverage; physical second-display acceptance remains
   deferred.
3. **P1-4:** final functional regression and evidence audit.
4. Deferred LAST: formal signing/distribution and second physical display.
