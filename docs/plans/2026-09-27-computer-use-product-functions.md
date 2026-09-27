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
