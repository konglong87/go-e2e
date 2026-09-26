# Computer Use: host-owned real-input acceptance

## Goal and boundaries
Validate input from **go-e2e's running Wails host -> approved Controller -> macOS backend -> bundled Swift helper**, not Codex CUA-generated input. Open WorkBuddy using that chain and click its New Task entry without submitting messages/tasks to a provider. Use a local fixture for text, mouse, keys and assertions. System grants, sensitive actions and credentials remain human-controlled.

## Architecture
Reuse Controller, observation binding, generation/deadline checks and receipts. An acceptance-only build tag enables a Unix-domain HTTP control adapter; it is absent from normal builds, only starts with an explicit CLI flag, resides in a private owner-only directory, uses bounded strict JSON and does not expose TCP. It cannot mint model authority or replace production tenant authorization. Execution must use the same computerManager as the native UI, with explicit start approval. A separate loopback fixture records trusted browser events; screenshots/receipts remain local ignored validation artifacts.

The normal model-to-desktop service integration must be separately checked: acceptance adapter success does NOT establish production autonomous agent routing. No unverified signing/TCC changes. Keep the currently authorized application identity during input acceptance. Do not delete executable or privacy entries until identity ownership is known.

## Change ledger (initial worktree clean; main == origin/main 962c3d9)
| Slice | Intended files | Verification | Delivery |
|---|---|---|---|
| Host acceptance adapter | desktop-v2/main.go, computer.go, computer_acceptance*.go, scripts/computer-acceptance.py | default/tagged/race Go tests; real .app build; WorkBuddy native hotkey/type/click + PNG evidence; cancellation/drain regressions | verified; committing slice |
| Isolated event fixture | internal/computeracceptance/** | HTTP/race tests; trusted DOM events observed from native input; fixture-final.png | verified in e02bad7 |
| Real native acceptance | scripts/computer-acceptance*, ignored build/validation/20260926 | WorkBuddy open/new task; 11 isolated input cases; safety results; native screenshots | verified on 2026-09-26; see report |
| Failure and scope report | docs/reports/computer-use-acceptance-20260926.md | stale/focus/crash/pause/stop; permission revocation and unsupported scope recorded | verified; remaining gates explicit |
| Release audit | docs/reports/computer-use-acceptance-20260926.md | signing, install/upgrade, agent routing and `go-e2e-desktop` usage audit | verified as blocked with evidence |

## Plan and gates
1. Add opt-in host adapter and tests without changing native safety behavior. Review, test, commit, push slice.
2. Fixture server and runner save before/after screenshots and redacted receipts; no private screenshots/logs committed. Run actual clicks/double/right/move/scroll/type/keys. Assert trusted events and target effects, not only outcome=executed.
3. Using the same host pipeline, open WorkBuddy via native keyboard navigation and click visible New Task. Inspect actual screenshot; never send content or start paid/cloud work.
4. Test stale observation rejection, unexpected focus change, helper crash, pause/stop concurrent with bounded wait, no subsequent input. Separate simulated permission revocation from actual OS revocation, which needs consent.
5. Review identity/install/upgrade and capability limits. Clean generated temporary files. Final report must distinguish supported/unimplemented, fake/real, passed/failed/not-run. Stable release remains blocked by any unverified mandatory gate.

## Non-goals for the first slice
Do not silently add window targeting, drag, multi-display, an unsecured web input endpoint, or broad TCC resets. These require independent design and validation. No new branch/worktree. Use konglong <konglong@com>; push every verified slice to origin/main.

## Observations from first native run
- Wails host launch using the explicit Unix socket flag reported input/capture ready after this local rebuild (not a signed-upgrade or clean-install certification).
- Native hotkey command+space, type WorkBuddy, Return opened installed WorkBuddy. Native coordinate clicks navigated to Assistant then New Task; final screenshot has an empty focused task composer. No task/message submitted.
- Output `executed` means dispatch acknowledged, not UI verified: immediately returned screenshots can precede animation/navigation. A subsequent observation confirmed page state.
- Browser startup produced one observe failure/session pause; reason not yet isolated. Explicit resume and a fresh observation succeeded. Do not label that case passed until characterized.
- Default builds do not compile the acceptance listener. This does not connect the production model runtime to the desktop controller.
