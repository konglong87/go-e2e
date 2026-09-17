# go-e2e Desktop

go-e2e 是基于 Wails v2 的桌面 AI 工作台，复用本仓库的 WebUI 2.0、
HTTP/SSE API 和 runtime。不是独立 Go module，必须从仓库根目录构建；
旧版桌面在 `desktop/`，不要混用两个构建脚本。

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
`Contents/MacOS/`。不能只复制桌面可执行文件而遗漏 server。
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

1. 启动后选择一个工作目录。该目录是 Agent 读写文件和执行命令的工作区，
   不是数据库存储位置；请先使用不含敏感数据的测试目录。
2. 欢迎页进入模型设置，配置自己的 provider、地址、凭据和模型，
   保存并完成连接验证。应用不附赠模型服务或凭据。
3. 创建会话并发送消息。后续启动进入会话首页；已有会话可从侧栏打开。
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
无关。旧版桌面使用独立的 `config.json` 与 `desktop.sqlite`，两版不会自动迁移
或同步数据。

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

## 原生窗口控制

desktop-v2 的 host 绑定提供以下 Wails bridge 方法：

```javascript
await window.go.main.App.Maximize();
await window.go.main.App.Unmaximize();
await window.go.main.App.ToggleMaximize();
await window.go.main.App.Fullscreen();
await window.go.main.App.Unfullscreen();
await window.go.main.App.ToggleFullscreen();
const state = await window.go.main.App.GetWindowState();
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

当前定位是开发预览版：macOS 签名、公证、自动更新、系统凭据库、完整
Windows 安装/卸载验收仍待完成。首次公开还必须通过
[开源发布检查](../docs/deployment/open_source_release_checklist.md)，
不能以本机可运行替代发布就绪结论。
