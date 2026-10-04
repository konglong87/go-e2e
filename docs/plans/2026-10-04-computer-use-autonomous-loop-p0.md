# Computer Use: macOS 自主闭环 P0 — 失败可恢复 + budget 放宽

## 目标与边界

让 macOS Computer Use 从"能跑通一次动作"进到"无需人工干预跑到任务完成"。
本方案只解决 P0 两个阻断点:**瞬态失败直接终结 run 且不可恢复**,**run budget 120s/16 轮太短**。

边界:
- 不改 Swift native helper 的安全语义(`SafetyError` 抛出逻辑、self-target 拒绝、权限预检保留)
- 不引入 unknown outcome 的自动重放(`MaxUnknownReplays` 保持 0,安全正确)
- 不动 acceptance 构建标签(它是测试适配层,不参与生产闭环)
- 只动 Go 侧(`internal/computeruse/`、`internal/computerbackend/macos/backend.go`、`internal/query/query.go`)

## 现状依据(为何是 P0)

### P0-A 瞬态失败 → 永久终态,不可恢复

当前任何瞬态失败都被映射成 run 的永久终态,run 无法继续:

- `controller.go:65 failRun()` → `Transition{To: RunStateFailed}`
- `runner.go:36 terminal()` → `failed` 是终态,**一旦进入无法转出**
- 触发点:
  - `controller.go:162-164` Observe 失败 → `Pause + failRun`
  - `controller.go:377` `receipt.Outcome == OutcomeUnknown || backendErr != nil` → `failRun`
  - `controller.go:354` receipt 非终端 → 标 unknown → failRun
  - `controller.go:393` 丢 after screenshot → `session.Pause()`(阻断但不 fail)
- backend 侧:
  - `backend.go:742` focus_changed → `quarantine()`(epoch++, paused=true, failed=true)
  - `backend.go:758` 截图尺寸/display 不符 → `invalidate()`(quarantine + `process.Abort`)
  - `backend.go:800-803` `invalidate()` = `quarantine()` + `process.Abort`(杀掉整个 helper)

**根因**:backend 把 focus_changed(瞬态,窗口还在)和真正的 helper 损坏(需重建进程)都映射成 `invalidate`(杀进程);controller 把所有 unknown/focus 都映射成 `failRun`(永久 failed)。没有"瞬态→可重试"的中间档。

### P0-B budget 硬编码且太短

- `runner.go:74 DefaultRunBudget()`: TotalDuration=120s, MaxInputActions=64, MaxModelTurns=16, MaxUnknownReplays=0
- `controller.go:44` `NewRunner(DefaultRunBudget())` —— **硬编码,无配置入口**
- `query.go:59` fast path `computerUseFastPathMaxTurns = 8`,更短
- 真实 computer use 任务(打开 app → 多步操作 → 等响应 → 验证)轻松超 120s/16 轮

## 改造方案

### Slice 1 — backend 引入"瞬态失败"语义,不再一律杀 helper

**问题**:`backend.go:742` 和 `:758` 都走 `invalidate()`(杀进程),但 focus_changed 是瞬态的(通知弹窗、Dock 动画),helper 进程本身没坏。

**改法**:区分两类失败
- **transient(瞬态)**:focus_changed、截图临时失败、窗口临时失焦 → `quarantine()` 但**不 Abort 进程**。epoch++ 撤销当前 observation 和 input 授权,但 helper 仍存活,下一次 Observe 可以重新激活窗口恢复。
- **fatal(致命)**:helper RPC 错误且 MayHaveRun、真正的传输损坏 → 保留 `invalidate()`(杀进程重建)。

**改动点**(`internal/computerbackend/macos/backend.go`):
1. 新增 `quarantineTransient()` = 现 `quarantine()` 的前半段(epoch++, paused=true, **failed=false**, observation 清空),**不调 `process.Abort`**。
2. `backend.go:742` focus_changed 分支:从 `quarantine()` 改为 `quarantineTransient()`。
3. `backend.go:758` 截图尺寸/display 不符:判断 —— 若 helper 仍响应(`response.OK != nil`)则 `quarantineTransient()`(可能只是动画中途截图),否则保留 `invalidate()`。
4. 保留 `drag`/`batch` 的 `uncertain → invalidate()`(`backend.go:700,725`)不变 —— 那是 input 已部分派遣的真 unknown,安全要求杀进程。

**验证**:
- `backend_test.go` 已有 `"focus-failed"` mode 测试(`:615,640,647,650`),扩展:断言 focus_changed 后 helper 进程**未被 Abort**、backend 处于 `paused && !failed`、下一次 Observe 可恢复。
- 新增 transient screenshot-fail 测试:helper 仍响应 → quarantineTransient,不杀进程。

### Slice 2 — controller 引入"瞬态失败→可重试 Observe"路径,不再 failRun

**问题**:`controller.go:377` 把 `OutcomeUnknown || backendErr != nil` 一律 `failRun()`。但 Slice 1 后,瞬态失败的 backend 处于 `paused && !failed`,helper 还活着,应该允许"重新 Observe 恢复"而不是终结 run。

**改法**:controller 区分 receipt 的失败类型
- **fatal receipt**(helper 损坏、真 unknown input):保留 `failRun()`。
- **transient receipt**(focus_changed、临时截图失败):**不 failRun**,改为 `session.Pause()` + runner 走 `ResetToObserved()`(`runner.go:260` 已存在,目前只供 cooperative pause 用)。run 保持非终态,模型/用户可以重新 Observe 继续。

**改动点**(`internal/computeruse/controller.go`):
1. `controller.go:162-164` Observe 失败:若 err 是 `focus_changed`/`permission_required`(瞬态,见 `:304` 已分类)→ `session.Pause()` 但**不 `failRun()`**;其余保留 `Pause + failRun`。
2. `controller.go:377` Execute 收到 unknown/focus_changed receipt:若 backend 仍 `!failed`(transient)→ `session.Pause()` + `runner.ResetToObserved()`,**不 failRun**;若 backend `failed`(fatal)→ 保留 `failRun()`。
3. `controller.go:393` 丢 after screenshot:已是 `session.Pause()`,补 `runner.ResetToObserved()` 让恢复后能重新 Observe(目前 Pause 后 runner 状态不归位)。

**关键安全约束(不变)**:
- 不自动重放同一 input(`MaxUnknownReplays` 仍 0)
- 恢复路径是"重新 Observe 当前真实状态 → 模型基于新截图决策",不是"重试刚才的动作"
- `ResetToObserved` 不清 usage(`runner.go:259` 注释已明确),budget 仍累计

**验证**:
- `runner_test.go` 新增:transient 后 `ResetToObserved` 成功、usage 不清零、可继续 Consume;fatal 后仍终态。
- `controller` 测试(fake_backend):focus_changed 后 run 不 failed、可 Observe 恢复;真 unknown input 后仍 failRun。

### Slice 3 — budget 可配置 + 默认放宽

**问题**:`controller.go:44` 硬编码 `DefaultRunBudget()`,120s/16 轮对真实任务太短。

**改法**:
1. `internal/computeruse/controller.go`:`NewControllerWithRegistry` 增加 `budget RunBudget` 参数(或新增 `NewControllerWithBudget` 构造函数,保留旧构造走默认)。
2. `runner.go:74 DefaultRunBudget()`:默认值放宽为 `TotalDuration=300s, MaxInputActions=96, MaxModelTurns=24`(保持 `MaxUnknownReplays=0`)。ObserveDuration/LaunchDuration/BindDuration 保持 10s。
3. `internal/query/query.go:59`:`computerUseFastPathMaxTurns` 从 8 提到 16(fast path 是简单任务,但仍需足够轮次完成 observe→input→observe 循环)。
4. budget 入口:wiring 到 query 层 options,允许按 prompt 复杂度选 budget(暂用放宽后的默认值,不暴露用户配置 —— 避免过度设计)。

**验证**:
- `runner_test.go` 更新 `DefaultRunBudget` 断言为新值。
- `query_test.go`(若存在)更新 fast path max turns 断言。
- `controller_test.go` 验证 `NewControllerWithBudget` 注入生效。

## 不做的事(非目标)

- 不改 Swift helper(`native/macos/`)—— 它的 `captureAfterInput` 已有 3 次截图重试(`Engine.swift:232-242`)和 Dock 点击后 app 激活等待(`:222-230`),瞬态恢复在 Go 侧做更合适
- 不加 helper 进程自动重建(P1#3,本轮不做)
- 不动 acceptance 构建标签(P2#7)
- 不加首次权限引导 UX(P2#6,macOS 约束无法自动授权)
- 不加跨全屏空间/第二屏支持(README 已标暂缓)

## 验收方案

### 单元测试(Go,必须全绿)

| 测试 | 文件 | 断言 |
|------|------|------|
| focus_changed 不杀 helper | `backend_test.go` | helper 未 Abort、`paused && !failed`、可重新 Observe |
| 瞬态截图失败不杀 helper | `backend_test.go`(新增) | helper 响应 → quarantineTransient、进程存活 |
| 真 unknown input 仍杀 helper | `backend_test.go` | drag/batch uncertain → invalidate + Abort(不变) |
| transient 后 run 不 failed | `runner_test.go`/`controller`(新增) | ResetToObserved 成功、usage 不清零、可继续 |
| fatal 后 run 仍终态 | `controller`(新增) | 真 unknown input → failRun、RunStateFailed |
| DefaultRunBudget 新值 | `runner_test.go` | 300s/96/24/0 |
| NewControllerWithBudget 注入 | `controller_test.go`(新增) | 注入的 budget 生效 |

运行:`go test ./internal/computerbackend/macos/... ./internal/computeruse/... ./internal/tools/computeruse/...`

### 集成验收(macOS 真机,带 `computeracceptance` 标签)

用现有 `scripts/computer-use-120s-acceptance.py`(脚本名是历史名,实际驱动 acceptance socket):
1. 构建:`scripts/build-desktop-v2.sh` 带 `-tags computeracceptance`(darwin)
2. 跑 acceptance:WorkBuddy 启动 → 新建会话 → 输入 1+1=2 → 点发送 → 等回复 → stop
3. **新增场景**:验收过程中模拟瞬态失焦(用 `osascript` 弹一个通知或 `open -a Notes` 抢焦点 1 秒后退出)→ 断言 run **不 failed**,自动 Pause 后能 Observe 恢复继续完成任务

### 回归保护

- 现有 acceptance 脚本的硬断言(`validate_actions`)全绿:首动作 observe、恰好 1 个 type、恰好 1 个 stop、window_id 一致、无白屏
- 最近 5 个 fix commit 修的场景(截图失败保留 trace、错误码保留、launch from frontmost)不被回归

## 改动文件清单

| Slice | 文件 | 改动 |
|-------|------|------|
| 1 | `internal/computerbackend/macos/backend.go` | 新增 `quarantineTransient`,改 `:742,:758` 分发 |
| 1 | `internal/computerbackend/macos/backend_test.go` | 扩展 focus 测试 + 新增 transient 测试 |
| 2 | `internal/computeruse/controller.go` | Observe/Execute 失败分流 transient vs fatal |
| 2 | `internal/computeruse/controller_test.go` 或 `computeruse_test.go` | 新增恢复路径测试 |
| 2 | `internal/computeruse/runner_test.go` | ResetToObserved 恢复测试 |
| 3 | `internal/computeruse/runner.go` | DefaultRunBudget 放宽 |
| 3 | `internal/computeruse/controller.go` | NewControllerWithBudget |
| 3 | `internal/query/query.go` | computerUseFastPathMaxTurns 提到 16 |

## 执行顺序

Slice 1 → 测试 → Slice 2 → 测试 → Slice 3 → 测试 → 集成验收 → 提交。
每个 Slice 独立可测、可提交。Slice 1 是 Slice 2 的前置(controller 的 transient 判断依赖 backend 的 `!failed` 状态)。
