# Computer Use functional parity — implementation ledger

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
| Exact same-app window selection | Current MacDesktop activates the application, not an exact AX window; multi-window ambiguity/raise still needs implementation and native acceptance |
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
