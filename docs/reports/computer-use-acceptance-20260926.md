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
