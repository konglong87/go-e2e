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
- 截图/动作完成前重新校验焦点，焦点变化不能被当作视觉验证成功。

提交：`c6e6188`。

## 未通过或未完成

### 1. 生产模型自动规划链路未接通

当前真实验收使用的是显式 opt-in 的 Unix socket acceptance adapter。它调用与 Wails UI 共享的 controller，但**不能证明**：

```text
模型 observation -> 模型 ComputerUse tool -> desktop controller -> receipt -> 下一轮模型 observation
```

静态审计显示 local service 启动的是独立 server 进程，生产路径没有注入 desktop-owned `computeruse.Service`。因此不能宣称已达到 Codex App 的生产级自主 Computer Use。

### 2. 窗口能力范围

当前实现：

- 单活动显示器；多显示器会被拒绝；
- 操作目标是当前桌面/前台焦点；
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

当前 `HEAD == origin/main == c6e6188`，工作区干净。

**最终发布判断：** 当前可以发布为“实验性/开发者预览验收结果”，不能按“Computer Use 完全通过、稳定版、Codex App 等价能力”发布。生产模型桥接、窗口定向/拖拽能力、clean install/upgrade/notarization 和真实焦点撤销验收仍是发布阻塞项。
