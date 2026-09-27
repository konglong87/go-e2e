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

## Production bridge preparation — 2026-09-26
- Resume ledger: the only dirty files were task-owned `desktop-v2/computer_agent_service*.go` and `internal/computerbridge/**`; no pre-existing user changes were included. Pulled origin/main with fast-forward-only before continuing.
- Added bounded authenticated Unix HTTP client/handler, strict wire validation, image validation, and ambiguous-execution receipts without retry. There is deliberately no Start/approval operation.
- Added host adapter to the existing Controller; every operation checks the full tenant/user/conversation binding. Synthetic preview ownership (conversation=0) is inaccessible. UI/controller Stop revokes lookup.
- Added real Unix-socket client -> handler -> host adapter -> Controller integration test with a fake backend; this validates wire/authority/stop/replay boundaries, NOT native GUI input.
- Verification passed: focused Go tests; race tests for desktop-v2, computerbridge, computeruse, computerbackend/... and tools/computeruse; go vet for computerbridge and desktop-v2. macOS linker emitted LC_DYSYMTAB warnings but tests exited successfully.
- Delivery: this slice is committed/pushed separately as transport preparation. No runtime listener, trusted active-conversation UI approval, or model injection is wired yet. No new desktop screenshot acceptance is claimed for this slice. Existing native evidence does not validate this new model path.
- Remaining gates unchanged: production model routing, real focus-perturbation regression, real permission revocation, clean-install/signed-upgrade/notarization. Keep `go-e2e-desktop`; it is the Wails executable, not a confirmed obsolete entry.

## Next architecture slice: trusted conversation binding
- Preserve one shared Controller as the authority; the model bridge can discover/use, never approve/create. Add an authenticated desktop-only owner-resolution endpoint: resolve tenant/user from server context and the numeric conversation ID through existing SessionControl.Get ownership checks. Never trust tenant/user IDs supplied by JavaScript.
- The Wails approval API accepts only a conversation ref. Resolve it through the desktop's own loopback server using its private token. A running controller cannot be reassigned to a different owner; the user must Stop before granting another conversation. Local preview remains conversation=0 and undiscoverable to model callers.
- Independently prepare the private Unix listener lifecycle (fresh memory-only token, restrictive socket/directory, cancellation/cleanup). Do not expose the host handler on HTTP/TCP or mint authority through it.
- Validate forged/missing refs, cross-owner access, old session reuse, failed lookup, stop/rebind and real Unix cleanup before commit/push. UI binding and model-provider image capability gating follow this boundary, not a hardcoded opt-in.

## Integration slice results and change ledger
| Slice | Files / scope | Verification | Delivery |
|---|---|---|---|
| Private listener and launch pipe | internal/computerbridge/listener*, launch* | real Unix client/handler, lifecycle/race, child FD handoff, token not in argv/env/disk | d5573dd pushed |
| Legitimate app activation | native/macos/Engine.swift, tests/main.swift | red/green regression; 151 fake-platform safety + 436 non-posting event assertions | c21e162 pushed |
| Trusted owner + normal query composition | desktop-v2/computer*, main.go, local_service.go; internal/server, cli, config; generated API docs/types | shared Controller/race; exact image-route gates including all fallbacks; mocked-provider Query/tool/image loop; real SQLite/JSONL owner resolution; desktop build and native UI approval/capture/pause/resume/stop | verified slice; commit/push with this ledger |
| Explicit UI binding | web/src/v2/computer/** and WebUIV2App.tsx | 737 frontend tests, both TS projects; native screenshots 01/02/04/05 in bridge evidence directory | verified with composition slice |

Real desktop testing found two identity-context failures before passing. The
final fix resolves the configured desktop actor with TenantService.ResolveContextOnce
and carries the bound context through SessionControl.Get. A real store integration
test covers missing and forged browser identity headers; numeric scope alone is
not sufficient for TenantManagedStore.

Known non-green test: the full internal/server suite encountered
TestSessionControlStopCancelsInflightRunAndReplaysOnSQLite with `database is locked`.
A five-repeat targeted rerun reproduced contention. Do not describe the full
server suite as green or assume this is release-safe. Focused owner/route tests pass.

Native evidence is under desktop-v2/build/validation/20260926/bridge/. Approval UI
was driven by Codex CUA; the displayed desktop images were captured by go-e2e's
own native helper. This does NOT prove real-provider autonomous model planning.
The empty test conversation was stopped and archived via the scoped session API;
a follow-up owner lookup returned 404. The old isolated fixture process was stopped.
No system permissions or user conversations were altered. Existing screenshots
and acceptance evidence remain private/ignored; no credentials or DBs are staged.

## Real-provider acceptance and SQLite follow-up (2026-09-27)
- Resume status was clean at 8479011; origin/main was pulled fast-forward-only.
- Local critical path: probe actual configured routes with generated non-private
  digit images, no fallback, before any desktop screenshot is sent. Use a separate
  owner-only home directory for isolated model settings, workspace and DB; do not
  alter the user's normal settings. Native UI still owns conversation approval.
- Parallel slice: SQLite audit read-to-write upgrade contention. Files are
  internal/storage/mysql/sqlite.go, sqlite_transaction_test.go and
  internal/server/session_control_stop_sqlite_test.go. Driver-level IMMEDIATE
  transactions reserve the writer before reads; no blanket retries or timeout
  increases. A deterministic competing-connection test failed before the fix and
  passes afterward in both rollback and WAL journal modes. Detached test workers
  are drained before DB cleanup.
- Verification: both SQLite Stop cases repeated 50 times, affected storage/mysql,
  sessioncontrol and server suites (ordinary + race), repeated focused race
  tests all passed. This fixes the reproduced lock-upgrade case; long-held
  external writers may still legitimately exhaust the busy timeout.

### First actual model run: new gates found
- Two standard-font random-digit probes passed on the configured Responses route;
  the default glm route reported no image, and an earlier seven-segment probe had
  one wrong digit. These are capability observations, not general vision accuracy.
- A normal, non-acceptance-tag desktop build ran in a separate private home
  configuration/workspace/DB. UI approval bound a real created conversation;
  its permission default was deny with only ComputerUse allowed. WorkBuddy was
  installed but had no running app processes before the model run.
- Real provider -> normal Query -> ComputerUse observe -> real helper succeeded.
  The observation expired after 10 seconds (capture RPC timeout). The model's
  hotkey arrived ~15 seconds later and was rejected with no dispatch. The next
  provider request failed with unexpected end of JSON input; no Stop tool call
  occurred. The test operator then stopped the host session. No blind input retry.
- Critical-path fix: decouple the helper's bounded 30-second observation lifetime
  from the capture RPC's independent deadline. Return exact expiry metadata and
  validate it in Go. Native injected-clock tests cover fresh observations after
  capture-RPC expiration and rejected observations beyond their own expiry.
- Parallel fix: bind runtime cleanup to the exact acquired host session and
  revoke on query termination even when the model/provider fails to call Stop.
  No re-lookup that might stop a later grant; no use of cancelled request context
  for cleanup; preserve original run errors.
- Runtime cleanup implementation verified: captures the originally acquired
  service/owner/session ID, uses an independent bounded five-second context,
  runs on construction failure and normal/error/cancelled/max-turn teardown,
  and never re-lookups or stops a replacement grant. Host Stop is idempotent for
  that exact revoked owner/session while all observation/lookup authority stays
  revoked. Focused and complete CLI/desktop/bridge/domain/backend/tool race
  suites passed. This cannot guarantee delivery to an unreachable host; cleanup
  errors remain visible and do not overwrite the original query result/error.

### Second actual model run and history-schema regression
- The normal Query's hotkey executed and opened Spotlight with the independent
  30-second snapshot lifetime. Subsequent observe/type/stop requests had invalid
  parameters. WorkBuddy was not launched. The model's completed status is NOT
  acceptance success; its final answer explicitly said the task was unfinished.
- Runtime teardown revoked the host grant despite the malformed Stop call:
  a later native UI observe request was rejected as unauthorized. The UI still
  displayed cached ready/progress state, exposing a separate status-sync gap.
- Model history currently presents redacted audit JSON as ComputerUse function
  arguments even though that JSON is intentionally not the executable schema.
  A regression test fails on this malformed historical example. Project the
  ComputerUse call/result pair to plain historical text only at the model-input
  boundary; preserve all privacy redaction, real execution input, images and
  unrelated tool call/result protocols. No raw input is restored to history.
- Red/green privacy/history tests and complete query, CLI, ComputerUse-tool and
  provider-adapter race suites passed. Actual-provider behavior of this history
  change still requires another independent run; earlier failures remain saved.

### Canonical live context vs audit context (refinement)
- A third Responses-route run had valid calls but stopped after its immediate
  after-image did not yet show the launcher; no WorkBuddy launch was proven.
- An independent Chat-Completions route passed the generated image probe, but its
  desktop run repeatedly observed without input. Provider input grew from ~16.5k
  to ~46.4k tokens across six observations; the test was explicitly stopped.
- Refine the previous history projection: the current query must retain the
  canonical tool call/result exchange in short-lived memory for its model.
  Only restored transcript/audit calls become plain historical notes. The source
  requirement forbids raw sensitive input in transcripts, not faithful ephemeral
  model protocol within the active run.
- All tool-parameter/image persistence boundaries remain redacted. Compaction
  input and even full prompt dumps receive sanitized copies without transient
  screenshot bytes; execution/model memory is not mutated by those copies.
- Retain only the latest two computer screenshots in live request history for
  visual comparison; preserve receipt metadata and non-computer/user images.
  This bounds request growth, not a claim of universal model visual accuracy.

### Native UI state synchronization
- Added a read-only Wails snapshot method and one-second, non-overlapping polling
  of the exact active session. Polling never Observe/captures, reads image bytes,
  or supersedes a model observation. Stale responses after controls/client/session
  changes or unmount cannot restore old state; Stop/Pause intent is preserved.
- Verification: 749 frontend tests, TypeScript, focused desktop race tests and
  normal desktop build. Live model Stop updated the floating panel to stopped
  without a manual refresh; screenshot 04-auto-synced-stopped.png is retained.
- This polls last_receipt; it is status/progress feedback, not a replacement for
  the complete receipt audit when multiple actions occur between poll intervals.

## Resume ledger — 2026-09-27, after 55e9339

- Initial worktree clean; `git pull --ff-only origin main` reported up to date.
- Intended tracked slice: this ledger and acceptance report. No source fixes
  proposed before coordinate root cause is established.
- Revalidated attempts 6/7: cold launch proven; explicit navigation failed.
- Read-only provider coordinate probes: generated targets exact; static WorkBuddy
  localization plausible. No native input issued by probes; no blanket scale fix.
- Release/identity sidecar: single active display, no window targeting/drag;
  `go-e2e-desktop` remains the Wails executable; strict ad-hoc signature passes,
  clean-install/signed-upgrade/notarization remain unverified.
- Ongoing live test uses an isolated approved conversation, normal production
  bridge and only ComputerUse for input. Its terminal outcome and independent
  screenshots must be checked separately before any success claim.
- Verification for this documentation slice: local JSON evidence inspected,
  source mapping/identity audited, `git diff --check`; private screenshots,
  generated probes, and isolated settings remain ignored and uncommitted.

### Verified slice: screenshot protocol and live navigation

- Source: `internal/tools/computeruse/tool.go`, `tool_test.go`.
- Red/green caption regression; affected tool/query/CLI/domain/backend/bridge/
  desktop/provider race suites all passed. Normal app rebuilt and signature checked.
- Source committed/pushed as `f8332a2`. Receipt images now explicitly remain
  non-actionable evidence; domain/backend semantics are unchanged.
- Attempt 10: passive native observer independently captured Assistant selected,
  then New Task selected/empty editor after two model clicks. Real target-app
  input came exclusively from go-e2e. Private artifacts remain ignored.
- Next: same-build cold-launch plus navigation in one run, followed by remaining
  fixture/safety/distribution gates. Overall stable-release gate stays open.

### Next coherent slice: successful input returns a real fresh observation

Attempt 11's cold launch stopped after typing into Spotlight: the model again
skipped Observe and its next key action was rejected. Caption guidance alone is
therefore not sufficient protocol robustness. Preserve that failed run.

Architecture: keep Controller/backend/helper unchanged. At the agent-tool boundary,
a successful executed action with validated after-image is followed by the existing
owner-bound `Service.Observe`. Return its real observation ID, expiry, metadata and
image beside the original receipt. This composes existing read-only capture, never
retries input, never promotes an evidence-image ID into authority, and reuses all
TCC/focus/expiry/Stop gates. Any action error/unknown/rejection, missing after-image,
cancellation, or capture failure must fail closed while preserving the receipt.

Write scope: tool implementation and tests, runtime guidance, this plan and report.
Verification: exact operation-order/unit privacy/error tests, affected race suites,
normal Wails build and fresh autonomous cold-launch/native screenshot evidence.

### Delivery and evidence checkpoint after 801423c

- Auto-observe slice committed/pushed as `801423c`; red/green operation-order and
  fail-closed tests plus all eight affected race suites passed. Normal `.app`
  rebuilt and strict ad-hoc signature checked.
- Attempt 12 terminated with the authoritative two-minute agent-stream idle
  timeout after initial Observe, before any input. This is not a native pass.
- Task and native panel both stopped; isolated app/server/helper exited. Exact
  task-created settings/credential-copy root removed; requested ignored evidence
  retained. User global configuration untouched.
- Next locally actionable work remains the latest-build native fixture/safety
  rerun, then a new isolated model run for cold-start/navigation. Distribution
  and real permission-revocation gates still need their own evidence/consent.
