# Computer Use 真实桌面验收报告（2026-09-26）

## 结论

**真实 native 输入链路通过；稳定版发布仍未通过。**

本报告验证的是：

```text
go-e2e Wails host
  -> approved computerManager / Controller
  -> macOS backend
  -> bundled Swift helper
  -> CGEvent / screenshot
```

它不是 Codex CUA 的点击结果，也不是生产模型自动规划链路的验收。

## 已通过

### WorkBuddy 真实应用用例

通过 go-e2e 自身控制器完成：

1. `Command+Space` 打开 Spotlight；
2. 输入 `WorkBuddy`；
3. `Return` 启动已安装的 WorkBuddy；
4. 点击助理页面；
5. 点击左侧“新建任务”；
6. 后续截图确认任务输入框处于新建任务页，未提交任何任务、消息、凭据或云端操作。

证据目录：`desktop-v2/build/validation/20260926/native-input/`

关键证据：

- `06-workbuddy-opened.png`
- `08-workbuddy-before-new-task.png`
- `10-workbuddy-new-task-verified.png`
- 相应 `*-receipt.json` 文件中包含 action outcome 与 before/after image refs。

### 隔离页面真实输入

隔离页面由 `internal/computeracceptance` 提供，事件通过真实浏览器页面上报，保留 `event.isTrusted`、事件目标、坐标、键盘修饰键和输入状态。

当前通过 11 项：

- click
- double click
- right click / context menu
- mouse move
- scroll
- text target focus
- 中文 + Emoji 输入
- Command+A 全选
- 替换选区
- Backspace
- ArrowLeft

最终截图：

- `desktop-v2/build/validation/20260926/native-input/fixture-final.png`

最终页面状态可见：

```text
acknowledged = 99
client dropped/uncertain = 0
inputValue = 验收AB
scrollTop = 420
```

### 安全与异常

通过真实运行的 go-e2e host/helper 验证：

- 过期 observation 拒绝；
- 被新 observation 替换的旧 observation 拒绝；
- pause during wait；
- pause 后 resume；
- stop during wait；
- helper 被精确定位并强制结束后，动作结果为 unknown，不伪报 executed；
- helper 崩溃后新 session 可恢复。

结果文件：

- `desktop-v2/build/validation/20260926/native-input/safety-results.json`

原生安全测试：

- `bash native/macos/tests/platform.sh`：436 assertions，未发送真实事件；
- `bash native/macos/tests/run.sh`：84 fake-platform assertions，未发送真实输入/截图。

### 修复并通过的控制语义

- native hotkey 现在发送完整的 modifier down -> key down/up -> modifier up 序列；
- pause 先发送 backend pause，再在有界时间内等待动作收敛；
- pause/resume 不再因为取消旧 RPC 而无条件杀掉 helper；
- stop 仍然优先、可打断阻塞动作；
- The post-action old-focus check introduced in `c6e6188` could reject legitimate app activation. `c21e162` replaces it with focus consistency during capture; receipts still do not prove visual intent was achieved.

提交：`c6e6188`。

## 未通过或未完成

### 1. Production model loop: real-provider acceptance still pending

当前真实验收使用的是显式 opt-in 的 Unix socket acceptance adapter。它调用与 Wails UI 共享的 controller，但**不能证明**：

```text
模型 observation -> 模型 ComputerUse tool -> desktop controller -> receipt -> 下一轮模型 observation
```

The initial audit found a separate local-server process without desktop-owned service injection. Subsequent slices add the private bridge, trusted conversation approval, and image-route-gated Query injection. Real-provider autonomous planning remains unverified.

### 2. 窗口能力范围

当前实现：

- 单活动显示器；多显示器会被拒绝；
- The post-action old-focus check introduced in `c6e6188` could reject legitimate app activation. `c21e162` replaces it with focus consistency during capture; receipts still do not prove visual intent was achieved.
- 非空 `window_id` 当前拒绝；
- 没有独立的窗口枚举、窗口 ID 绑定或按窗口定向操作；
- 没有拖拽 action。

### 3. 焦点变化真实干扰

原生 fake-platform 和代码路径有焦点变化保护；本次真实验收没有获得可靠的“外部应用切换后旧 observation 被拒绝”的桌面证据，因为 CUA 的外部 focus 操作不能稳定改变测试进程观察到的前台 PID。该项标为 **未完成真实验收**，不是通过。

### 4. 权限撤销

未自动撤销 Screen Recording / Accessibility 权限。撤销系统授权会改变用户系统设置，需要用户明确参与；当前仅验证了已授权状态下的 capture/input，以及 helper crash 后的恢复。

### 5. 新用户安装、升级、签名

未完成：

- clean user / clean macOS machine 首次授权；
- Developer ID signing；
- notarization / stapling / Gatekeeper；
- 两个真实签名版本之间的升级 TCC continuity；
- 无历史 ad-hoc 授权的首次安装。

当前 ad-hoc 开发构建的 `codesign --verify --deep --strict` 通过，不等于可信发行签名。

## `go-e2e-desktop` 权限项结论

不要删除。

当前运行中的 Wails 主程序是：

```text
go-e2e.app/Contents/MacOS/go-e2e-desktop
```

它是当前桌面进程，且主程序自身会请求 TCC；helper 也独立执行 screenshot/input。系统设置里的 `go-e2e-desktop` 条目不能仅凭显示名称认定为废弃记录。需要在正式签名、明确 bundle/path 映射和干净环境验证后，才可评估删除某一条旧权限记录。

## Git / release gate

已推送：

- `fbf62ec` — opt-in desktop-owned native input acceptance adapter
- `e02bad7` — trusted-event fixture and native input safety acceptance
- `c6e6188` — native modifier and controller safety fixes

Historical snapshot only: the original native-input slice was clean and pushed at c6e6188. This is not a claim about the current HEAD or worktree.

**最终发布判断：** 当前可以发布为“实验性/开发者预览验收结果”，不能按“Computer Use 完全通过、稳定版、Codex App 等价能力”发布。生产模型桥接、窗口定向/拖拽能力、clean install/upgrade/notarization 和真实焦点撤销验收仍是发布阻塞项。

## Follow-up: normal-query bridge and trusted desktop approval

The following implementation is now present:

```text
Native approval UI (captured conversation ref)
 -> own authenticated local server
 -> configured desktop identity + bound tenant context + SessionControl.Get
 -> approved immutable Controller owner
 -> private Unix listener / inherited memory-only launch pipe
 -> normal Query (exact declared image routes + approved-session Lookup)
 -> ComputerUse tool / native controller / screenshot result
```

Validation is deliberately separated:

- **Production construction with a mocked model:** actual newQuerySession tool
  registration, normal tool dispatch and image attachment were tested using a
  local fake provider and fake image backend. This validates runtime plumbing,
  not a model's ability to plan desktop actions.
- **Real desktop:** built the Wails `.app`, clicked the actual approval UI,
  resolved a newly created managed test conversation, received a real native
  screenshot, paused, resumed, refreshed to a new observation, then stopped.
  Codex CUA performed these UI acceptance clicks; go-e2e captured the desktop.
- **Identity regression:** fake owner tests missed a requirement in the real
  managed store. The added SQLite/JSONL composition test requires the complete
  bound tenant context, without trusting incoming tenant/user headers.
- **Image capability:** explicit operator declarations are required for the
  final selected model and every effective fallback route. No model/provider
  allowlist was added to the user's settings during this run. Undeclared routes
  do not receive ComputerUse. See internal/computerbridge/README.md for the format.

Evidence (private, ignored):

- `desktop-v2/build/validation/20260926/bridge/01-local-preview-approval.png`
- `desktop-v2/build/validation/20260926/bridge/02-bound-conversation-approval.png`
- `desktop-v2/build/validation/20260926/bridge/03-owner-resolution-failure-before-fix.png`
- `desktop-v2/build/validation/20260926/bridge/04-bound-conversation-screenshot.png`
- `desktop-v2/build/validation/20260926/bridge/05-bound-conversation-stopped.png`

Tests: frontend 737 tests and both TypeScript projects; core desktop/bridge/domain/
backend/tool race suites; CLI/config tests; focused owner/runtime/image-route race
suites; Go vet; tagged desktop build and ad-hoc deep signature verification.
Native safety now has 151 fake-platform assertions and 436 non-posting platform
assertions. The complete native input fixture was NOT rerun on this build.

**Non-green result:** full internal/server testing and a repeated targeted run
encountered SQLite `database is locked` in the existing session-stop test. Its
cause has not been established here; it remains a tracked risk, not a passed gate.

**Still not certified:** autonomous WorkBuddy planning by a real configured model,
real external-focus perturbation, actual OS permission revocation, clean-user
installation, signed upgrade/TCC continuity, Developer ID and notarization. The
single-display/no-window-target/no-drag capability boundaries are unchanged.
`go-e2e-desktop` remains the actual Wails executable and was not removed.

## SQLite contention follow-up (2026-09-27)

The previously recorded `database is locked` failure was reproduced and traced
to a deferred read transaction upgrading after a concurrent writer reservation.
The opener now uses the driver's `_txlock=immediate` connection option. A
controlled competing writer proves the regression in both rollback-journal and
WAL modes. There is no retry loop, busy-timeout increase or pool-size workaround.

The two SQLite Stop cases passed 50 repetitions each; complete storage/mysql,
sessioncontrol and server suites passed normally and with `-race`. The test also
drains detached finalization before removing its SQLite fixture. This supersedes
the earlier unresolved-lock status for this reproduced case, not all possible
SQLite failures. Legitimate external write contention still returns errors when
the existing timeout is exhausted.

## First real-provider attempt (2026-09-27): failed, not counted as passed

A normal desktop build (no acceptance adapter) ran with isolated settings,
workspace and SQLite/JSONL state in a private home directory. Only ComputerUse
was allowed; all other tools defaulted to deny. The verified Responses route
read two standard-font random-digit images correctly; the default route reported
no image, and an earlier seven-segment image was misread. This proves image
transport on that route, not universal visual accuracy.

The normal model Query called ComputerUse observe and received a real native
screenshot. Its next hotkey arrived about 15 seconds later, after the returned
observation's 10-second expiry, and was safely rejected before dispatch. This
exposed an implementation bug: snapshot lifetime used the capture RPC timeout,
although the domain freshness limit is 30 seconds. Snapshot expiry is now returned
by the helper independently of the RPC deadline, and Go rejects missing, expired
or overlong metadata. Injected-clock native tests prove both boundaries without
sleeping. The original 30-second freshness limit is not relaxed.

The next provider request failed with `unexpected end of JSON input`; the model
never issued Stop. The operator stopped the host session. This failure is retained
as evidence, not erased or retried blindly. Runtime teardown is being hardened to
revoke the originally acquired grant even when the provider/model does not Stop.
WorkBuddy had no running app processes before this attempt; this failed attempt
has not established its launch or New Task acceptance.

### Query teardown revocation

Normal Query cleanup now issues Stop for the exact acquired service/owner/session,
including malformed-provider-response, cancellation, construction-failure and
max-turn paths. Cleanup is bounded independently of the cancelled request and
never looks up a replacement grant. Host duplicate Stop is allowed for the exact
revoked binding, without restoring any lookup/observe privilege. Unit/integration
coverage includes a mocked reproduction of observe -> rejected hotkey -> malformed
third response. Full affected CLI/desktop/bridge/domain/backend/tool race suites
passed. An unreachable host can still prevent delivery; the original query error
is preserved and cleanup failure is logged rather than hidden.

## Second real-provider attempt (2026-09-27): partial input, still failed acceptance

The model's hotkey executed and opened Spotlight. The returned observation now
has the bounded 30-second lifetime, separate from RPC timeout. Later calls failed
strict input decoding; no WorkBuddy process was launched. The model returned a
completed conversation but explicitly reported the desktop task unfinished.
Runtime cleanup revoked the grant even though the model's Stop call was invalid;
a subsequent native UI screenshot request was denied. The UI retained a cached
ready state, so agent-to-UI status synchronization remains a known gap.

Investigation found that redacted audit JSON was sent back to the model as
ComputerUse function-call arguments despite not matching the tool schema. A
regression test failed before the correction. Model history now projects these
call/result pairs as ordinary historical text; strict execution decoding and
all input/image privacy boundaries remain intact. No raw input is restored to
history. Full query/CLI/ComputerUse/provider-adapter race suites passed. This
fix's live-provider effect remains to be verified; the second attempt is not
reclassified as successful.

## Live-history refinement and bounded frames

Subsequent real-provider attempts still did not complete WorkBuddy acceptance:
one stopped after an executed shortcut's immediate image did not confirm the
launcher; another repeatedly observed and grew provider input from about 16.5k
to 46.4k tokens. The observation-only run was explicitly stopped, not reported as
success.

The current live query now preserves canonical model-generated tool arguments
and matching results in ephemeral memory. This supersedes the earlier decision
to turn *all* active calls into historical text. Restored transcript/audit records
remain non-executable historical notes. Tool parameter recording, callbacks and
hooks stay redacted; compaction inputs and full prompt dumps now also exclude raw
ComputerUse inputs and transient screenshot bytes. Only the latest two desktop
frames are retained in model request history; unrelated/user images remain.

This is a protocol/privacy-boundary correction, not a relaxation of native
permission, owner binding, freshness, Stop, or strict action validation. The
actual autonomous WorkBuddy objective still requires a successful live run.

## Native UI synchronization

The floating panel now reads the shared Controller state through a read-only
Wails method. Its one-second non-overlapping polling does not capture or replace
screenshots and cannot overwrite pending user Stop/Pause with stale results.
The full frontend suite passed 749 tests, TypeScript passed, and the normal
`.app` was rebuilt. A live model Stop propagated to the panel automatically;
see `desktop-v2/build/validation/20260927/04-auto-synced-stopped.png`.
Last-receipt polling can miss intermediate rapid actions, so the panel is not
claimed to replace complete event/receipt evidence.

The fifth live test stopped on a real pre-existing macOS informational alert
saying the go-e2e application could no longer be opened. The active desktop
instance itself remained running. The alert was independently observed in
CoreServicesUIAgent and dismissed; the subsequent go-e2e native screenshot no
longer showed it. This environmental abort is retained, not counted as success.

## Autonomous cold launch and targeted navigation — 2026-09-27

Attempt 6 ran through the normal configured model Query, ComputerUse tool,
authenticated desktop bridge, shared Wails Controller and bundled Swift helper.
WorkBuddy had no running app processes before the attempt; it was running after
model-generated native input. Independent read-only native inspection confirmed
New Task selected and an empty composer. No tester-generated WorkBuddy click or
typing was used. Private evidence:

- `desktop-v2/build/validation/20260927/workbuddy-before.json`
- `desktop-v2/build/validation/20260927/workbuddy-after-model.json`
- `desktop-v2/build/validation/20260927/05-live-model-workbuddy-new-task.png`

This establishes autonomous launch, **not a proven click on New Task**: that page
can be the default landing state. Attempt 7 therefore required the distinct
Assistant -> New Task transition. It dispatched two clicks, then stopped and
reported opening the macOS window-control menu instead of Assistant. Independent
AX inspection never established Assistant selected. The failure remains recorded
in `attempt-7/` and `06-live-model-navigation-state.png`; it is not a pass.

### Coordinate investigation, not yet a root-cause claim

The current native mapping is screenshot pixels -> global display points, with
scale applied once in `DisplayGeometry.point`. The provider adapter sends the PNG
without a client-side resize. Read-only tests against the same configured Responses
route, with fallback disabled, localized three generated targets on a 2704x1756
image exactly, both with explicit coordinate instructions and with the existing
observation-style metadata. A separate recorded full-display WorkBuddy frame also
returned plausible in-item Assistant/New Task coordinates. These probes weaken a
*systematic* scale-factor explanation but do not prove reliable localization in
multi-turn live operation or explain attempt 7. No coordinate transform or safety
checks were changed based on this hypothesis. Probe evidence remains private under
`desktop-v2/build/validation/20260927/coordinate-probe/`.

### Release scope and identity recheck

- Current helper requires exactly one active display; nonempty window targeting
  is rejected. Focus binding is an application PID, not a window identity. Multiple
  windows of one application are not certified by that guard.
- Drag is not an advertised native action. A draggable go-e2e control panel does
  not establish native drag automation support.
- `go-e2e-desktop` is the current bundle's `CFBundleExecutable`; the adjacent
  `go-e2e` binary is its local server. Do not remove the desktop executable or
  its authorization as obsolete. Helper/display naming is not sufficient to map
  individual TCC entries conclusively.
- Read-only strict signature verification of the existing bundle passed, but it
  is ad-hoc signed with no TeamIdentifier. This is not Developer ID distribution.
- Installer scripts/CI launch smoke tests do not establish clean-user TCC behavior
  or in-place signed upgrade continuity. No clean-install, two-version signed
  upgrade, notarized-release, or actual permission-revocation pass is asserted.

Stable release remains **not accepted**. Prior native fixture results are retained
with their original build/date; they are not silently relabeled as current-build
passes. The next live attempt is independently tracked and cannot supersede the
failed navigation record without actual page-transition evidence.

## Fresh-observation protocol clarification and independent navigation pass

Attempt 8 executed a hotkey, skipped the mandatory subsequent Observe, and had
its click rejected before backend dispatch. `ComputerSession.BeginAction`
consumes each observation; a receipt after-image does not mint a new actionable
observation. Both image kinds previously had the same generic caption. The tool
now explicitly labels fresh observations versus post-action evidence, includes
the actual decoded image dimensions/ID, and explains full-image pixel coordinates
and the native scale conversion. This changes model guidance only: no automatic
retry, coordinate transform, permission, or freshness guard was relaxed.

Regression test `TestScreenshotContextDistinguishesObservationFromActionEvidence`
failed before and passed after. Full affected tool/query/CLI/domain/backend/bridge/
desktop/provider packages passed with the race detector. The normal untagged Wails
app was rebuilt and strict signature verification passed. Source commit: `f8332a2`.

Attempt 9 obeyed observe-between-inputs and reported successful navigation, but
independent screenshots were taken after the intermediate state had ended. It is
not used as independent evidence of the Assistant transition.

**Attempt 10 independently passed Assistant -> New Task navigation on this normal
build.** A passive observer reacted to the model's recorded tool-result events:

- First model click acknowledged at 02:24:47 UTC; native AX/screenshot captured at
  02:24:51 UTC showed Assistant selected and its existing conversation page.
- Second model click acknowledged at 02:25:25 UTC; native AX/screenshot captured
  at 02:25:28 UTC showed New Task selected and an empty composer.
- Observe followed each click; valid Stop and terminal completion were recorded.
- The tester made no WorkBuddy clicks/typing. The observer only read AX and
  captured screenshots; no task, message, assistant execution, update or setting
  change was submitted by the test.

Private evidence: `desktop-v2/build/validation/20260927/attempt-10/`, particularly
`native-evidence.json`, `event-10-click.png`, `event-19-click.png`, the terminal
screenshot, and the redacted conversation. The two transition images were also
visually inspected. They contain private existing application content and must
not be committed or included in public release assets.

This is a genuine successful navigation workflow, not a claim that all Computer
Use capabilities or release gates pass. Cold-launch evidence is still attempt 6
on its earlier build; a single-run current-build cold launch plus navigation and
remaining input/safety/distribution gates remain distinct work.

## Successful input now composes a fresh authorized observation

Attempt 11 was a genuine current-build cold-start test: WorkBuddy was installed
but no app process was running. The model opened Spotlight and dispatched text,
but skipped Observe before a key action; the key was rejected. WorkBuddy never
started. Thus the earlier caption improvement alone did not make the protocol
robust enough. This failed run is retained under `attempt-11/`.

Commit `801423c` composes the existing owner-bound `Service.Observe` after a
successful executed action **and** a validated after-image. Its response contains
the original receipt plus the new observation ID, expiry, metadata and image. A
receipt image is still never actionable. Controller, native helper, coordinate
mapping, TCC, focus, expiry and Stop gates are unchanged. This adds a read-only
capture, never an input retry. Errors, unknown/rejected outcomes, missing/corrupt
evidence and cancellation do not trigger the capture; failed post-action capture
preserves the executed receipt and available evidence and tells the model to stop.

The operation-order and failure regression tests failed against the preceding
implementation, then passed. Eight affected packages passed with `-race`:
`internal/tools/computeruse`, `internal/query`, `internal/cli`,
`internal/computeruse`, `internal/computerbackend/macos`,
`internal/computerbridge`, `desktop-v2`, and `internal/anthropic`.
The normal untagged desktop app was rebuilt and strict signature verification
passed. An initial guidance-string test failure was corrected before the final
passing suite; no failing tests were waived.

Attempt 12 used that normal build with WorkBuddy absent from the process list.
It completed only its initial Observe. The authoritative terminal event was
`agent task stream idle timeout after 2m0s`, not a native-input success or a
successful model completion. No input action was recorded. Session status and
native panel both reached stopped. The passive observer's later timeout was not
used to infer query termination; the API/event record was separately checked.
This run therefore **does not validate the new action-plus-observation path live**
or establish a same-build cold-launch/navigation pass.

### Cleanup and remaining gates

After all model runs were terminal and the native session stopped, the isolated
Wails instance and its owned server/helper processes were stopped. Only the
exact task-created directory `/Users/konglong/.go-e2e/cu-live-440467931` was
removed, including copied provider credentials, private test DB and logs. The
user's normal configuration was not modified. Requested screenshots and redacted
run evidence remain under ignored `desktop-v2/build/validation/20260927/`;
`cleanup.json` records this check. No screenshot, credential, database or private
log is included in Git.

Still open: complete cold-start plus navigation on the latest build; current-build
real-input fixture; external focus perturbation and actual OS permission
revocation; clean installation, signed upgrade and notarized distribution.
Attempt 10's independent navigation pass remains valid for its stated build,
not silently upgraded to a blanket latest-build or stable-release certification.

## Isolated latest-source fixture preparation and OS consent checkpoint

The normal desktop's read-only session API returned 32 sessions, including two in
active states. It was therefore neither quit nor overwritten. A separately built
`computeracceptance` host was packaged into an ignored side-by-side copy at
`desktop-v2/build/validation/20260927/native-fixture-build/go-e2e.app`, using the
same current source for the native stack and copied unchanged helper/server/assets.
Normal-app host/server/helper hashes were checked unchanged. The copied app has
credential-free isolated settings; no provider request is used for this fixture.

Before running the safety suite, the fixed 11-second expiry delay was found to be
invalid for the current 30-second native lifetime. Commit `e07c387` now waits for
the returned RFC3339 expiry with a bounded monotonic guard; malformed metadata
fails before input. Crash injection additionally requires an explicit app path
and the exact unique host/helper parent relationship, rechecked before signaling.
This prevents accidentally targeting the concurrently running normal desktop.

Fresh verification:

- 21 Python safety-runner regression tests passed.
- Tagged desktop/fixture/fixture-command Go race suites passed.
- 157 fake-platform native safety assertions passed (no real capture/input).
- 436 native event-construction assertions passed (no events posted).
- Side-by-side Wails build and strict ad-hoc signature verification passed.
- Its live private listener reported capture/input ready, native screenshots
  succeeded, and two bounded native keyboard actions opened/closed Spotlight.
  These setup actions are **not** a fixture-input or safety-suite pass.

A macOS consent dialog then visibly requested direct screen/system-audio access
for go-e2e. It appeared despite ready preflight metadata and successful captures;
its cause is not inferred to be a new-install, upgrade, or TCC-continuity result.
No Allow/System Settings button was clicked. Spotlight was closed so the prompt
is fully visible, and the acceptance Controller was explicitly stopped.

Evidence: `desktop-v2/build/validation/20260927/native-input/04-pending-system-permission.png`
and `native-fixture-build/checkpoint.json`. The local fixture server and isolated
app remain available for the user's action-time authorization decision; there is
no active native-input sequence. The normal desktop remains running, untouched.
**Current-build fixture inputs and safety cases have not yet run.** This new
consent checkpoint is not counted as an acceptance pass or a stable release.

## Release prerequisites checked against actual state — 2026-09-27

While the new macOS consent dialog remains awaiting user confirmation, a read-only
release audit was performed. No permission, keychain, account billing, repository
visibility or release-publication setting was changed; no secret value was read
or printed.

| Gate | Authoritative observation | Acceptance result |
| --- | --- | --- |
| Local Developer ID identity | `security find-identity -v -p codesigning` succeeded and reported zero valid identities; zero Developer ID Application identities | No local formal signing prerequisite established |
| Current normal app | `codesign` reports ad-hoc signing and no TeamIdentifier | Local integrity verification is not distribution trust |
| Gatekeeper | `spctl --assess --type execute` returned exit 3 / rejected | Current local app does not pass this distribution assessment |
| Stapled app ticket | `stapler validate` returned exit 65 / no ticket | No app-level ticket established; this does not substitute for assessing a future DMG |
| Repository signing configuration | Successful names-only query found none of the six Apple secrets referenced by the release workflow | The workflow's current all-absent branch permits unsigned/unnotarized packaging; no signed build was performed |
| Published release artifacts | GitHub release listing returned empty; repository metadata still says private | No published candidate was available for a clean-install/signed-upgrade acceptance run |
| Remote CI | CI for `46f7666` and Windows build for `801423c` failed before steps started | Not passing CI, but also not evidence of a code-test failure |

GitHub check-run annotations explain that jobs did not start because recent
account payments failed or the spending limit needs attention. The API does not
resolve which of those account conditions applies. This needs the account owner,
not a code workaround; no billing change or workflow rerun was attempted.

The new deterministic Computer Use safety-runner tests were not yet included in
the existing offline acceptance entry point. They are now wired into that gate,
using fake clocks/process tables only, with bytecode generation disabled. Local
`GO_E2E_GO_CACHE_MAINTENANCE=0 bash scripts/offline-acceptance.sh --static-only`
passed **216 checks**. This is an offline/static result, not a replacement for
remote CI, real native safety tests, or the unrun deterministic TUI scenarios.

Private raw checks are under
`desktop-v2/build/validation/20260927/release-audit/`: signing/assessment status,
CI/job/check-run annotations, names-only signing configuration summary and the
offline check log. Only this non-secret summary and the gate wiring enter Git.
Stable Computer Use release remains unaccepted. OS authorization, current-build
real-input/safety evidence, complete autonomous cold launch, and actual signed
installation/upgrade verification remain separate outstanding gates.

## Latest native-input fixture rerun — 2026-09-27

After the user reported granting the macOS dialog, native captures showed no
remaining consent prompt. The prior desktop listener subsequently disappeared;
process inspection confirmed both earlier desktop instances had exited before a
new isolated instance was launched. No input request was dispatched through the
missing socket. The reason for the application exit is not established and is
not classified as a successful crash test.

The isolated tagged host uses the current native input stack. It opened Edge
through native Spotlight actions, but initial URL setup did not establish a
connected fixture. CUA was used **only to prepare the fresh local fixture tab**;
that setup is not counted as autonomous navigation or a native-input test case.
The page then reported zero events, and its token fragment was visibly removed.

All **11 fixture cases passed** through go-e2e's own Controller -> macOS backend
-> bundled helper, with actual trusted target events, pointer-coordinate checks
and asserted final effects: click, double-click, right-click/context menu, move,
scroll, text focus, Chinese/emoji input, Command+A selection, replacement,
Backspace and ArrowLeft. No CUA/DOM-generated input was used for those cases.

The fixture snapshot contains **102 events, all trusted**, with zero server or
client drops. Final state: text `验收AB`, selection `3..3`, scrollTop `420`,
scrollLeft `0`. Native screenshot `fixture-final.png` was visually inspected and
matches the event/state assertions. This reruns the native input layer, not the
production model's action-plus-observation path.

Evidence under `desktop-v2/build/validation/20260927/native-input/`:
`fixture-results.json`, `fixture-snapshot.json`, `fixture-run.log`, each case's
before/after screenshot and redacted receipt, and `fixture-final.png`.
Safety fault injection is a separate run; its results are not implied by these
11 successful input cases.

## Current native safety rerun — seven scoped cases passed

The live run exposed a harness sequencing bug: Stop correctly terminates its
helper, so a following crash-test preflight cannot require that old helper to
still exist. A regression failed before the correction; commit `872cfcc` validates
the explicit app path, starts the crash case's new session, then selects and
rechecks that new helper before signaling. All **22** deterministic harness tests
passed afterward. Exact host/path/parent guards remain intact; there is no global
process kill. The first partial run is preserved under `safety-attempt-1/`.

The complete repeated native run then passed all seven cases:

1. Actual returned observation expiry rejection (waited past its 30-second TTL).
2. Superseded observation rejection.
3. Pause during an in-flight bounded wait; subsequent input rejected.
4. Explicit resume after that interrupted wait and fresh capture.
5. Stop during an in-flight bounded wait; subsequent input rejected.
6. SIGKILL of only the isolated host's uniquely identified helper during wait:
   outcome unknown, not executed, and later observation denied.
7. New session after helper crash successfully captures again.

The forbidden sentinel was not added to the fixture text. Pause/Stop timings,
receipts, rejection errors, and recovery screenshot are retained in
`desktop-v2/build/validation/20260927/native-input/safety-results.json`,
`safety-run.log`, and `safety-crash-recovered.png`. These are real native
Controller/helper tests, not model planning. Interruptions cover the supported
wait action, **not a held mouse button/key or drag**.

External-focus verification remains open: CUA AX Raise/click/shortcut probes did
not establish a different frontmost application PID according to the independent
macOS front-app readback. A coordinate probe returned `noWindowsAvailable` while
process inspection still showed Edge alive. No stale-focus input was sent or
claimed rejected. This is an instrumentation limitation, not proof of a product
focus defect or success. The acceptance session was explicitly stopped afterward.
Real OS permission revocation and signed clean-install/upgrade also remain open.

## Final current-build model retry checkpoint — 2026-09-27

A final isolated-copy attempt was prepared from the latest source commit and
started with WorkBuddy absent from the process list. The copy was then discarded
because macOS LaunchServices/TCC identity handling made the changed bundle
identifier require a separate permission entry; no new permission was silently
assumed. The original-bundle copy was also not used to claim a model pass.

A separate normal-app session was created only to diagnose the configured model
route. The selected `gpt-5.6-sol` request failed before the first ComputerUse
call with provider HTTP 404 `model is not found`. It therefore produced no native
input, no WorkBuddy launch, and no screenshot evidence. A gpt-5.5 provider was
selected in the UI afterward, but no new model run was started before cleanup;
this is not a pass. The normal app's existing user sessions were not quit or
modified. The exact isolated credential/config root was removed after the failed
attempt; current native fixture and safety evidence remain retained.

This does not invalidate Attempt 10's independent Assistant -> New Task pass,
which remains a genuine warm-navigation pass on its stated build. It means the
latest-source **cold launch + navigation in one model run remains unverified**.
The provider/model availability prerequisite must be fixed or explicitly routed
to an available vision-capable provider before another model attempt is useful.
Stable release remains not accepted.

## Latest-source autonomous Computer Use acceptance — 2026-09-27

The configured route was tested in an isolated, non-user configuration root:
`gpt-6-sol` through the OpenAI Responses-compatible provider, with one exact
`primary` image-input route and no fallback routes. This did not modify the
user's normal `~/.golang-cc/settings.json`.

The first latest-source run reached the provider but failed with a transient
`unexpected end of JSON input` before a ComputerUse tool call. A direct
provider tool probe then confirmed the route can return a `ComputerUse`
function call and that the Responses stream recovery path recovered a truncated
SSE attempt. The acceptance was rerun in a fresh approved session rather than
counting the failed run.

Fresh retry session evidence (`attempt-14`, ignored private build output):

- WorkBuddy was absent before the run.
- Model issued a real `ComputerUse observe` and received `observation-7`.
- Model issued eight ComputerUse calls total, with successful executed input
  receipts and fresh observations including `observation-11`, `observation-15`,
  `observation-21`, and `observation-27`.
- WorkBuddy was autonomously launched, the model clicked **助理**, then clicked
  **新建任务**.
- The final native AX tree independently shows `新建任务` selected and its
  blank composer (`今天帮你做些什么？ ...`). The final screenshot is
  `desktop-v2/build/validation/20260927/attempt-14/workbuddy-final.png`.
- The model explicitly called ComputerUse Stop and the managed session finished
  `completed`; the final response states the three observed transitions.
- No task content, login, message, settings change, points claim, or app update
  was performed.

The exact private conversation and a redacted event summary are retained under
`desktop-v2/build/validation/20260927/attempt-14/conversation.json` and
`summary.json` locally and are ignored by Git. This is now a genuine
latest-source **model-driven cold launch + navigation pass** for the stated
macOS single-display scope. It does not close the remaining external-focus,
real permission-revocation, signed clean-install/upgrade, notarization, or
release-distribution gates.

## External-focus fail-closed fix — 2026-09-27

The real tagged acceptance run exposed a focus-detection defect: a helper using
only `NSWorkspace.frontmostApplication` could retain a stale host PID while a
non-activating helper was running. In that state, a stale-observation click
could be accepted after another app became frontmost.

`native/macos/Platform.swift` now derives the focus identity from Window
Server's ordered on-screen layer-0 window list, falling back to NSWorkspace
only when the list is unavailable. This keeps the safety decision in the
native helper rather than trusting UI state.

Real verification after rebuilding the helper:

- Frontmost before observation: current acceptance host PID `39976`.
- Frontmost after perturbation: Microsoft Edge PID `48113`.
- Reusing the old observation was rejected with no after-image and no executed
  receipt; the session stayed in `needs_observation`.
- Evidence: ignored local files under
  `desktop-v2/build/validation/20260927/focus-test/`, especially
  `focus-fixed-before-1.json`, `focus-fixed-after-switch.json`, and
  `focus-fixed-snapshot.json`.

`bash native/macos/tests/run.sh` passed **157 native safety assertions**.
This closes external-focus fail-closed verification, while clean signed
installation/upgrade, notarization, stapled ticket, and actual TCC revocation
remain open release gates.
