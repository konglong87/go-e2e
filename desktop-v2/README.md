# go-e2e Desktop

go-e2e 是基于 Wails v2 的唯一桌面 AI 工作台，复用本仓库的 WebUI 2.0、
HTTP/SSE API 和 runtime。不是独立 Go module，必须从仓库根目录构建。
legacy WebUI 仍可通过浏览器/server 入口使用，但不再提供第二个原生桌面包。

## 从源码构建

需要 Go、Node.js/npm、Wails CLI v2.10.2 和平台开发工具。发布工具链以根目录
`.tool-versions` 为准；macOS 需要 Xcode Command Line Tools，Windows 需要
WebView2 Runtime，制作安装包还需要 NSIS。首次安装前端与 Wails 依赖：

```bash
npm --prefix web ci --legacy-peer-deps
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
wails doctor
```

macOS：

```bash
scripts/build-desktop-v2.sh
open desktop-v2/build/bin/go-e2e.app
```

脚本以 `VITE_DESKTOP_UI_VERSION=2` 构建前端并复制到 `desktop-v2/frontend/dist`，
编译本地 server，再用 Wails 打包，并将 `go-e2e` 放入 app 的
`Contents/MacOS/`。不能只复制桌面可执行文件而遗漏 server。macOS 在复制
server 后会按“先签名嵌套 server，再签名外层 app”的顺序重新签名，并严格校验
整个 app；不要在构建完成后手工替换 `Contents/MacOS/go-e2e`。
脚本会重建 `web/dist`；若随后运行独立 Web server，应重新执行
`npm --prefix web run build` 生成 `/webui/` 前缀的 Web 产物。

Windows（在 Windows 上执行 PowerShell）：

```powershell
.\scripts\build-desktop-v2-windows.ps1
node scripts/test-desktop-v2-windows-packaging.mjs --require-pwsh
```

脚本目标产物为 `dist\go-e2e-setup.exe`，仅支持 Windows amd64。
自定义 Wails v2.10.2 NSIS 模板将 `go-e2e-desktop.exe` 与 `go-e2e.exe`
安装到同一目录；卸载删除两个程序和安装注册信息，保留用户配置、SQLite、
WebView2 数据及安装目录内的用户文件。npm、Go、Wails 任一步骤失败都会中止，
不会将旧安装包复制为本次产物。构建会重建前端和 `desktop-v2/build/bin`。

`Desktop Windows` workflow 已配置隔离安装、两个 EXE 的 SHA256 readback、
卸载及用户文件保留检查；本地测试使用临时 fixture，不修改已有打包产物。
无 PowerShell 时可去掉 `--require-pwsh` 仅跑静态契约检查，原生命令失败测试
会明确跳过。仅有构建配置不等于通过原生验收：此修复未触发 CI，
Windows 安装/卸载实际运行、WebView2 与 GUI 首次启动仍须在 Windows 上验收。

## 首次使用

1. 启动后自动创建并选中 `~/go-e2e-workspace/go-e2e` 作为 Default 工作区。
   该目录是 Agent 读写文件和执行命令的边界，不是数据库存储位置。
2. 如果尚未配置模型，应用会引导进入模型设置；配置自己的 provider、地址、
   凭据和模型并完成连接验证。应用不附赠模型服务或凭据。
3. 创建会话并发送消息。新建会话默认使用实际配置的主 Provider/Model；
   高级用户仍可在会话输入区手动切换。也可以随时选择其他工作区。
   后续启动会恢复上次工作区和会话首页；已有会话可从侧栏打开。
   有效深链接仍打开对应会话，无效链接显示可恢复错误。

本地 server 只绑定 `127.0.0.1` 的动态端口，使用每次启动生成的随机 token。
前端等待 health/readiness 后才访问业务接口。关闭应用会请求停止 server，
超时后结束子进程。桌面默认禁用 scheduler，不把它当作常驻渠道 worker 使用。

## 配置与数据

已实现本地 SQLite 与启动 migration，**不需要安装 MySQL**。
桌面配置和数据默认位于 `~/.golang-cc/`：

| 文件 | 用途 |
| --- | --- |
| `config-v2.json` | 工作目录选择 |
| `go-e2e.sqlite` | 本地会话、消息、提示词模板等 |
| `go-e2e-server.log` | 本地 server 诊断 |
| `go-e2e-startup.log` | 桌面启动诊断 |

desktop-v2 固定使用用户 home 下的 `.golang-cc` 目录，和平台默认的应用支持目录
无关。删除旧桌面实现后，不提供旧桌面配置或 SQLite 数据的自动迁移。

模型配置仍复用 runtime 的 `~/.golang-cc/settings.json`，可用
`GOLANG_CC_CONFIG_DIR` 重定位；它与桌面工作目录配置不是同一文件。
凭据目前由 runtime 配置机制管理，**尚未接入 Keychain/Credential Manager**。
不要提交 settings、数据库、日志或包含真实内容的验收截图。

诊断环境变量：

| 变量 | 作用 |
| --- | --- |
| `GO_E2E_SERVER_BINARY` | 指定本地 server 二进制；兼容读取 `GOLANG_CC_SERVER_BINARY` |
| `GOLANG_CC_DESKTOP_SERVER_PORT` | 指定诊断端口；默认动态选择 |
| `GOLANG_CC_DESKTOP_CONFIG_DIR` | 将 desktop-v2 数据根目录重定位到该目录下的 `golang-cc/` |
| `GOLANG_CC_CONFIG_DIR` | 重定位 runtime 模型配置目录 |

需要隔离验收数据时使用独立 OS 用户，或在 macOS/Linux 为测试进程指定独立 HOME；
设置 `GOLANG_CC_DESKTOP_CONFIG_DIR` 也会同时隔离 desktop-v2 的配置、SQLite 和日志。

## Computer Use（Phase 1，macOS Host）

desktop-v2 现在提供独立的 Computer Workspace 控制面，不修改 WebBrowser 语义：

- `GetComputerCapabilities`：读取 macOS Host 的 capture/input/focus/permission readiness。
- `StartComputerSession`：创建 session-level approval 会话；未批准不会进入 ready。
- `ObserveComputerSession`：从 native helper 获取真实桌面 PNG，并展示 Preview。
- `PauseComputerSession` / `ResumeComputerSession` / `StopComputerSession`：控制同一
  session 的输入生命周期；Stop 后不会自动恢复。
- `GetComputerActionReceipt`：读取结构化 action receipt 和 before/after observation 引用。

helper 位于 `.app/Contents/Helpers/computer-helper-macos`，使用平台无关的长度前缀 JSON
协议与 Go backend 通信。helper 环境采用白名单，不继承 provider key；协议只接收经校验的结构化动作。
不得要求模型输入密码、Token、OTP 等凭据；`type` 的普通输入在执行时仍需传给 helper，
不能把脱敏摘要误认为已有凭据识别/安全输入能力。第一阶段只支持 macOS Host，不实现
Windows/Linux/TUI/Code Execution/Replay。

真实验收命令：

```bash
bash scripts/build-desktop-v2.sh
open desktop-v2/build/bin/go-e2e.app
```

**当前仍是受限控制面，不是可用的模型自动操作闭环。** Wails 中的 Controller 尚未与
本地 server 的 Query 共享，生产请求没有设置 Computer profile/service/image capability，
因此普通聊天不会注册 ComputerUse。provider/model/fallback 图片能力解析、MediaAsset
失败保留策略和模型真实像素 E2E 仍待实现。验收状态与证据路径见
[实施与验收记录](../docs/architecture/computer_use_implementation_status.md)。

## 原生窗口控制

desktop-v2 的 host 绑定提供以下 Wails bridge 方法：

```javascript
await window.go.main.app.Maximize();
await window.go.main.app.Unmaximize();
await window.go.main.app.ToggleMaximize();
await window.go.main.app.Fullscreen();
await window.go.main.app.Unfullscreen();
await window.go.main.app.ToggleFullscreen();
const state = await window.go.main.app.GetWindowState();
```

窗口状态变化会通过 `window.runtime.EventsOn` 广播：

```javascript
window.runtime.EventsOn("go-e2e:window-state-changed", (state) => {
  console.log(state);
});
```

普通窗口的位置和尺寸、最大化状态、全屏状态会保存到 `config-v2.json`。
最大化或全屏退出时会恢复最后一次普通窗口几何；无效或过小的历史几何会回退到
`1440x900` 的默认窗口和 `1024x700` 的最小尺寸。Wails v2.10.2 在 macOS、
Windows 和 Linux 使用同一组 runtime API，不需要额外依赖。

## 验证与限制

```bash
go test ./desktop-v2 -count=1
npm --prefix web run typecheck
npm --prefix web test
cd web
npx playwright test --config playwright.desktop.config.ts
```

浏览器回归使用确定性 API fixture，不能替代原生进程、真实模型和落库验收。
当前入口修复证据见 [首次入口验收](../docs/manual_testing/desktop_v2_entry.md)，
已有模型回复与持久化记录见 [PromptPicker 验收](../docs/manual_testing/prompt_picker_ui.md)。

自动更新、系统凭据库仍未接入。Windows 安装/卸载和跨平台发布验收由
GitHub Actions 的 Release workflow 执行；本机可运行不能替代发布就绪结论。

## 正式发布

正式桌面发布由 GitHub Actions 的 `Release` workflow 统一完成。推送
`v*` tag 后会构建 macOS DMG、Windows 安装程序、Linux 桌面归档和 CLI
归档，最后生成 `SHA256SUMS` 并创建 GitHub Release。macOS tag 发布时，可以
配置 Developer ID 签名和 notarization Secrets，也可以让六个 Apple Secret
全部为空，以未签名/未公证模式发布。未签名模式下，用户首次打开需要在
“系统设置 → 隐私与安全性 → 安全性”点击“仍要打开”。具体 Secret 名称、
发布命令和验收步骤见 [Release Pipeline](../docs/deployment/release_pipeline.md)。
