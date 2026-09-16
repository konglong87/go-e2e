# golang-cc

> 当前项目默认首页是 [`README.md`](README.md)。本文件保留完整技术参考；[返回新版首页](README.md)。

golang-cc 是一个用 Go 实现的**通用 Agent runtime**，目标是成为一个万能的 agent —— 既能作为 coding agent，也能作为 chat agent、agent 中台等多种模式运行。它以 **golang-cc** 作为运行时身份，借鉴 Claude Code 、opencode 在工具调用、会话、权限、上下文治理和 agent loop 上的设计思想，并在此基础上建设属于 golang-cc 自己的通用 agent 架构。

本项目追求 **可靠、可观测、可扩展、可平台化**。Claude Code/openCode 都是重要参考系；兼容性用于降低迁移和理解成本，真正的产品方向是把本地 Coding Agent 扩展成 API Server、多租户、移动端 Chat、Agent 开发框架、可观测性和企业级治理底座。当前已经覆盖 CLI、query loop、tools、stream-json、session、permissions、sandbox、skills/plugins、multi-agent、OpenAI-compatible API、mobile API、多租户持久化和 telemetry 主链路；能力边界见 [当前边界](#当前边界)。

从旧名称升级时，配置、状态、环境变量和稳定协议的兼容边界见 [golang-cc 统一命名与迁移说明](docs/product_rename_golang_cc.md)。

## 工具链要求

| 工具 | 版本 | 依据 |
| --- | --- | --- |
| Go | **1.26.5 或更高** | `go.mod` 的 `go 1.25.0` 只是编译下界；`crypto/tls` 的 [GO-2026-5856](https://pkg.go.dev/vuln/GO-2026-5856) 要 1.26.5 才修好，低于此版本构建出的二进制仍带该漏洞 |
| Node.js | **^20.19.0 \|\| >=22.12.0**（CI 用 22.23.1） | `web/` 依赖 `vite@8` / `vitest@4` |

版本以 [`.tool-versions`](.tool-versions)（asdf）和 [`web/.nvmrc`](web/.nvmrc)（nvm）为准，CI 也读同一组版本号：

```bash
asdf install            # 或
nvm use                 # 在 web/ 目录下
```

如果本机 `GOSUMDB=off`（常见于配了国内 proxy 的环境），Go 会拒绝校验自动下载的 toolchain 并报
`verifying module: checksum database disabled by GOSUMDB=off`。要么装好对应版本的 Go 并放进 `PATH`，
要么只在需要下载 toolchain 时临时开启校验：

```bash
GOSUMDB=sum.golang.org GOTOOLCHAIN=go1.26.5 go version
```

`web/` 的依赖安装必须带 `--legacy-peer-deps`：`typescript@6` 与 `openapi-typescript@7.13` 声明的
peer `typescript@^5.x` 冲突，裸 `npm ci` 会 ERESOLVE 失败。

```bash
cd web && npm ci --legacy-peer-deps
```

## 安装

三条路径共用 [`scripts/build.sh`](scripts/build.sh) 这一处 Build Identity 注入，统一写入
version、Git revision、dirty 状态和 build time。所以任何一条路径装出来的二进制
`--version` 都是真实版本号，`version --json` 还能输出同一身份及 Go toolchain。

### 1. 从源码安装到 PATH（推荐）

```bash
git clone git@github.com:konglong87/golang-cc.git
cd golang-cc
scripts/install.sh

~/.local/bin/golang-cc --version    # v0.1.48-go
```

默认装到 `~/.local/bin`，用 `PREFIX=/usr/local scripts/install.sh` 换前缀（或直接给 `BIN_DIR`）。
目标目录不在 `PATH` 里时，脚本会把该加的那行 `export` 打出来；加好之后就能直接
`golang-cc` 调用。

编译前会先比 Go 版本，两个下界含义不同：低于 `go.mod` 的 `go` 指令直接失败（编译根本过不去），
低于 [`.tool-versions`](.tool-versions) pin 的版本只告警 —— 能编过，但产物仍带
[GO-2026-5856](https://pkg.go.dev/vuln/GO-2026-5856)。

### 2. 用预编译产物

推 `v*` tag 会触发 [`.github/workflows/release.yml`](.github/workflows/release.yml)：按
`.tool-versions` 的 Go 版本交叉编译 darwin / linux（amd64 + arm64）和 windows/amd64，
解开 linux 产物核对 `--version` 确实等于该 tag，再连同 `SHA256SUMS` 发到 GitHub Release。

```bash
# <tag> 形如 v0.1.48-go
shasum -a 256 -c SHA256SUMS          # Linux 上用 sha256sum -c SHA256SUMS
tar -xzf golang-cc_<tag>_darwin_arm64.tar.gz
install -m 0755 golang-cc_<tag>_darwin_arm64/golang-cc ~/.local/bin/
```

> 两个前提说清楚：本仓库当前**不是公开仓库**，Release 只有有仓库权限的人能下载；
> **这条链路落地之前打的 tag 一个产物都没有**（`git tag` 至今 40 个），要旧版本请 checkout
> 该 tag 后走路径 1 或 3。

本机复现同一套产物（`dist/` 下每个平台一个归档加一份 `SHA256SUMS`）：

```bash
scripts/release.sh                        # 全部目标
TARGETS="linux/amd64" scripts/release.sh  # 只要一个
```

### 3. 只构建，不安装

```bash
scripts/build.sh
bin/golang-cc --version
```

### 为什么没有 `go install` / `brew install` / `curl | sh`

- **`go install` 装不上**，这不是文档漏写。原先有三个阻塞项，现在只剩一个：
  module path 与仓库地址不一致这条已修（module path 现在就是
  `github.com/konglong87/go-e2e`，与仓库地址一致）。但 `go.mod` 里的
  `replace github.com/muesli/termenv => ./third_party/termenv` 仍在，而
  `go install pkg@version` 不应用 replace 指令，所以拉下来的模块解析不了 termenv。
  未传 `-ldflags` 时的版本已由统一 Build Identity 回退到 Go module build info 解决。
  记录在 [docs/todo.md](docs/todo.md) TODO-088。
- **`curl | sh` 和 `brew install` 没做**：两者都需要一个公开可下载的产物地址，而 Release 现在
  要仓库权限。现在写出来就是一条跑不通的命令，等仓库公开再补（TODO-087）。

## 快速开始

不装也能直接从源码跑：

```bash
git clone git@github.com:konglong87/golang-cc.git
cd golang-cc
export ANTHROPIC_API_KEY="your-api-key"

go run ./cmd/golang-cc --version
go run ./cmd/golang-cc doctor
go run ./cmd/golang-cc -p "查看当前目录有哪些文件"
```

常用运行：

```bash
# TUI 交互模式
go run ./cmd/golang-cc

# 普通文本
go run ./cmd/golang-cc -p "帮我总结 README.md"

# JSON 最终结果
go run ./cmd/golang-cc -p "列出项目结构" --output-format json

# 流式 JSON 事件
go run ./cmd/golang-cc -p "读取 go.mod" --output-format stream-json

# 指定工作目录
go run ./cmd/golang-cc --cwd "$PWD" -p "读取 README.md"

# 最小代码 runtime：只自动装配 Read / Edit / Bash，不做上下文和扩展自动发现
go run ./cmd/golang-cc --bare -p "读取 go.mod 并说明模块信息"

# 检查运行时指导文档中的本地坏链
go run ./cmd/golang-cc --cwd "$PWD" memory lint
```

`--bare` 也可直接启动 TUI，并保留显式 settings、MCP、add-dir、权限和 session/resume；它不会执行 hooks，也不会自动发现 memory、Git context、skills、custom agents、plugins 或 MCP。完整行为和组合限制见 [`--bare` 使用说明](docs/usage/bare.md)。

注意 `go run` 不注入版本号，`--version` 会显示 `dev`；要真实版本号走上面的 [安装](#安装)。

`go run ./cmd/golang-cc ...` 会编译运行当前源码，但项目上下文默认取进程启动时的当前目录。需要让 golang-cc 读取另一个项目的 `.golang-cc/settings*`、`golang-cc.md`、legacy `CLAUDE.md` / `.claude/*`、skills、plugins 或原生 Claude Code project memory 时，必须在目标项目目录中启动，或显式传 `--cwd <project-dir>`；否则它会按当前仓库目录加载上下文。例如在本仓库里调试另一个项目时应使用：

```bash
go run ./cmd/golang-cc --cwd /path/to/other-project -p "读取项目规则"
```

任何子命令都接受 `--help`：

```bash
go run ./cmd/golang-cc --help          # 全局用法与所有子命令
go run ./cmd/golang-cc session --help  # 单个子命令的用法
```

`memory lint` 默认检查当前 runtime 能发现的 Project、Local、Workflow 和 Claude Code project memory 物理源文件；它会检查普通 Markdown link 与 image 的本地文件目标，发现坏链时返回非零状态。`--scope all` 还会加入 Managed、User、Team 和 Auto 来源，`--json` 输出稳定 JSON：

```bash
go run ./cmd/golang-cc --cwd "$PWD" memory lint --json
go run ./cmd/golang-cc --cwd "$PWD" memory lint --scope all
```

这里的 `workspace` scope 遵循 runtime 现有发现规则，可能包含 cwd 父目录上的指导文件，因此不等于“只扫描仓库目录”；Project/Local/Workflow 的 `@include` 仍受现有 `sameTree` 边界限制，不会扩展到源文件目录树外。检查对象是完整物理源文件（frontmatter 除外），因此 Claude Code project `MEMORY.md` 第 200 行之后的坏链也会报告；它不检查远程 URL、UNC/network path、heading fragment 或缺失的 `@include` 本身，也不自动修复。该命令跳过 startup updater，不执行隐式 fetch/pull。

### 诊断日志

CLI 与 TUI 默认**不往 stderr 打诊断日志** —— 出错时你看到的就是一行人话。需要排查时用
`GOLANG_CC_LOG_LEVEL`（或 `LOG_LEVEL`）显式打开结构化日志：

```bash
GOLANG_CC_LOG_LEVEL=debug go run ./cmd/golang-cc -p "hi"
```

合法级别：`debug`、`info`、`warn`、`error`、`dpanic`、`panic`、`fatal`、`silent`。
Go 调用栈只在 `debug` 下输出。写错级别名会直接报错并列出合法值，不会静默降级。
`server` 子命令不受影响，始终输出完整的 JSON 结构化日志。

## 选择运行模式

| 你要做什么 | 推荐模式 | 入口 |
| --- | --- | --- |
| 本地交互写代码、读文件、跑工具 | TUI | `go run ./cmd/golang-cc` |
| 脚本、CI、一次性自动化 | CLI print/headless | `-p, --print` |
| 以最小工具和显式上下文运行本地/CI 任务 | CLI/TUI bare profile | `--bare` |
| 给 Web/App/后台系统调用 | API Server | `server` |
| 兼容 OpenAI SDK / Chat UI | OpenAI-compatible API | `/v1/chat/completions` |
| iOS/Android 类 ChatGPT 体验 | Mobile Chat API | `/mobile/chat/...` |
| 企业多用户、多租户、审计 | API Server + MySQL | `tenant migrate` + tenant API |
| 本地 Web Agent 真机测试 | Web Agent scripts | `scripts/web-agent-start.sh`、`scripts/web-agent-real-e2e.sh` |
| 暴露给 MCP client | MCP Server | `mcp serve` |
| 长任务后台运行和恢复 | Background | `--bg`、`/loop`、`attach`、`logs`、`kill`、`ps` |

分模式、分场景的使用说明（TUI / CLI / agent-webui，每个场景配命令和要点）见 [docs/usage/README.md](docs/usage/README.md)。完整文档入口见 [docs/README.md](docs/README.md)。运行架构和时序图见 [docs/architecture/runtime_modes.md](docs/architecture/runtime_modes.md)。`/loop` 与本地 cron scheduler 方案见 [docs/architecture/loop_scheduler_design.md](docs/architecture/loop_scheduler_design.md)。

Web Agent 本地启动和真实端到端测试：

```bash
# 真实 provider + MySQL + WebUI
scripts/web-agent-start.sh

# 真实全生命周期 E2E：API、runner、事件流、MySQL、浏览器渲染
scripts/web-agent-real-e2e.sh

# 只用于长消息滚动和打字机 UI 压测，不是真实模型验收
scripts/web-agent-scroll-smoke.sh
```

详情见 [docs/web_agent/web_agent_scripts.md](docs/web_agent/web_agent_scripts.md)。

## 最小配置

golang-cc 支持全局 `~/.golang-cc/settings.json`（或 `GOLANG_CC_CONFIG_DIR/settings.json`）、项目 `.golang-cc/settings*.json`、legacy 项目 `.claude/settings*.json` 和项目内 `config/*.yaml`。密钥优先放在 golang-cc owned global settings，项目本地覆盖推荐使用 `.golang-cc/settings.local.json` 或 `config/config.local.yaml`，避免把密钥提交进仓库。

> 可直接复制的完整 `~/.golang-cc/settings.json` 示例（含 provider / fallback / `subagentModelTiers` / `modelPricing` / permissions），以及字段逐项说明和多 provider 按 model id 路由要点，见 [docs/examples/settings_quickstart.md](docs/examples/settings_quickstart.md)（示例文件 [docs/examples/settings.example.json](docs/examples/settings.example.json)）。

```yaml
model: gpt-5.5
provider: custom
effort: high
env:
  ANTHROPIC_BASE_URL: https://ai-gateway.example.com/v1
  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
webSearch:
  endpoint: https://cn.bing.com/search
tui:
  resumeHistoryLimit: 6
  showThinking: true
  thinkingMode: summary
webAgentUI:
  thinkingMode: summary
  showThinking: true # legacy compatibility input when thinkingMode is absent
context_length: 251000
```

TUI 使用 `tui.thinkingMode: full|summary|hidden` 控制完整展示、摘要折叠或隐藏，并支持
`/thinking show <turn>` 展开指定回合；未配置时兼容旧 `tui.showThinking`。
`webAgentUI.thinkingMode: full|summary|hidden` 控制 Web Agent 完整展示、消息级摘要折叠或隐藏；页面也提供即时三态选择和“展开思考 / 收起思考”交互。未配置时兼容旧 `webAgentUI.showThinking=true/false`。两类设置都只影响 UI：
模型仍会思考，`thinking_delta` 仍会传递、持久化并保留在 Trace/审计数据中。

配置加载顺序：

```text
~/.go-claude/settings.json             # legacy product compatibility
~/.golang-cc/settings.json 或 $GOLANG_CC_CONFIG_DIR/settings.json
nearest .claude/settings.json          # legacy project compatibility
nearest .claude/settings.local.json    # legacy project compatibility
nearest .go-claude/settings.json       # legacy product compatibility
nearest .go-claude/settings.local.json # legacy product compatibility
nearest .golang-cc/settings.json       # golang-cc project, or configured identity dir
nearest .golang-cc/settings.local.json # golang-cc project local override, or configured identity dir
nearest config/config.yaml
nearest config/config.<env>.yaml
nearest config/config.local.yaml
```

模型优先级：

```text
--model 参数
config/config.yaml / 环境 YAML / 本地 YAML
CLAUDE_CODE_MODEL
settings.json
内置默认
```

TUI 交互模式支持 per-turn 配置刷新：未使用 `--model` 时，修改配置后下一条消息会重新读取模型并刷新顶部信息；使用 `--model` 时会锁定命令行模型。完整提示词与配置链路见 [docs/prompt_logic/prompt_loading.md](docs/prompt_logic/prompt_loading.md)。

可用 `--provider <name>` 从合并后的 `fallback.providers` 中按 `name` 精确选择 provider，并将其作为本次运行的 primary。例如 `--provider sensenova-deepseek-v4-flash` 会直接使用该条目的 `type`、`baseURL` 和凭证；未显式传 `--model` 时同时使用该条目的 `model`，显式 `--model` 优先。provider 名称不存在时命令直接报错并列出可用名称。该选择会随 background、loop schedule 和 goal 持久化。

上游 provider 的厂商/路由身份与 wire protocol 分开配置。老配置不写 `providerProtocol` 时行为不变：
`custom` / `openai-compatible` 仍走 Chat Completions。接入 Responses-compatible 端点必须显式声明：

```json
{
  "model": "your-responses-model",
  "provider": "custom",
  "providerProtocol": "openai-responses",
  "responses": {
    "stateMode": "stateless",
    "store": false
  },
  "env": {
    "ANTHROPIC_BASE_URL": "https://your-responses-endpoint.example.com/v1",
    "ANTHROPIC_API_KEY": "sk-REPLACE"
  }
}
```

当前 Responses 第一阶段支持 stateless 文本/图片、function calling、reasoning summary、structured
output、usage、SSE、fallback 和 encrypted reasoning continuation。固定发送 `store=false`，不发送
`previous_response_id`；`stateMode: "previous-response-id"` 和 `store: true` 会明确报未实现。兼容服务
对字段和事件的实现程度并不一致，真实端点需按能力逐项验收。

使用 `session inspect <session-id>` 可关联 transcript 与模型请求日志，查看每次请求的用途、模型、最终 provider、安全化 endpoint、attempt 和 fallback；加 `--json` 获取结构化输出，加 `--log <path>` 指定日志。新日志通过 session UUID 精确关联；旧日志仅按时间和模型推断，会明确标记 `confidence=heuristic`，不会把当前配置冒充为历史请求证据。

TUI slash commands 支持 `/loop [interval] <prompt>` 创建 recurring schedule，并立即执行一次；例如 `/loop 5m check the deploy` 或 `/loop check the deploy every 20m`。底层使用本地 cron scheduler，api-server 启动时也会带起 scheduler。通过 `ps` 查看，`logs <id>` 查看输出，`kill <id>` 停止。

常用 TUI 操作：

| 操作 | 快捷键 / 命令 |
| --- | --- |
| 停止当前响应 / 退出 | 第一次 `Ctrl+C` 停止响应，第二次 `Ctrl+C` 退出 |
| 清空输入 | `Ctrl+U` |
| 切换权限模式 | `Shift+Tab` |
| 切换鼠标模式 | `Ctrl+O`，`mouse=copy` 保护拖选复制，`mouse=scroll` 支持鼠标滚轮滚动 |
| 粘贴剪贴板图片 | `Ctrl+V` |
| 打开 session 恢复选择器 | `/resume` |
| 生成/查看会话 recap | `/recap`、`/recap show` |
| 回退代码和/或对话（非破坏，可 redo） | `/rewind` 查看候选消息，`/rewind <message-id>` 执行；支持 `--conversation-only`、`--files-only` |
| 查看/对比会话分支 | `/branches`，`/branches --compare <叶A> <叶B>`（非破坏回退产生的分支） |
| 切回被搁置的分支 | `/redo <叶id>` 默认还原文件+对话；`--conversation-only` 只回对话（叶 id 见 `/branches`） |
| 查看 slash commands | 输入 `/` |

会话恢复：

```bash
# 继续最近一次 transcript
go run ./cmd/golang-cc --continue

# 恢复指定 session
go run ./cmd/golang-cc --resume <session-id>

# 只恢复到某条 transcript entry
go run ./cmd/golang-cc --resume <session-id> --resume-session-at <entry-id>
```

TUI resume 会继续把完整 transcript 注入下一轮模型上下文，同时默认把最近 6 条用户/助手历史消息打印到终端 scrollback。可通过 `tui.resumeHistoryLimit` 调整显示条数，设为 `0` 可关闭恢复历史展示；checkpoint、rewind、tool 原始事件仍只用于恢复和观测，不作为聊天历史直接刷屏。

transcript / checkpoint / snapshot 的存储位置，以及 `session inspect`、`checkpoint`、`rewind`、`redo`、`branches`、`fork` 等命令的速查、图解和典型场景 demo，见 [docs/session_quickstart.md](docs/session_quickstart.md)。v2 消息图与非破坏 rewind/redo 的设计与实现见 [docs/transcript/README.md](docs/transcript/README.md)。

## Goal 模式

Goal 模式把一个长期目标保存为本地状态机，并在每个 goal turn 前自动创建 session checkpoint，记录事件、usage、状态和 blocker 信息。P0 本地实现使用 `~/.golang-cc/goals/goals.jsonl` 与 `~/.golang-cc/goals/events/*.jsonl`（或 configured owned global root），复用现有 query loop、session transcript、checkpoint、compact、工具、权限和 telemetry 链路。

Goal 的 slash 候选菜单、全局 `/help` 和详细 `goal help` 共用 `internal/goalcmd` 中的中文说明。命令名、参数和别名保持英文；解释、生命周期注意事项和排错入口提供中文，避免三个入口各自维护后发生漂移。

```bash
go run ./cmd/golang-cc goal help
go run ./cmd/golang-cc goal start "Implement TUI app-level selection" --turn-budget 20 --token-budget 200000
# start 返回 goal id，但不会开始执行；active 仅表示可运行
go run ./cmd/golang-cc goal run <goal-id> --once
go run ./cmd/golang-cc goal run <goal-id> --background
go run ./cmd/golang-cc goal status <goal-id>
go run ./cmd/golang-cc goal inspect <goal-id> # status 的别名
go run ./cmd/golang-cc goal logs <goal-id>
go run ./cmd/golang-cc goal stop <goal-id>
go run ./cmd/golang-cc goal resume <goal-id>
go run ./cmd/golang-cc goal run <goal-id> --background # resume 后重新启动执行
```

`goal run --background` 会创建 `kind=goal` 的后台任务并持续执行到 Goal
进入 `complete`、`blocked`、`failed` 或 `stopped`。可继续用 `ps`、`logs <id>`、
`attach <id> --wait` 和 `kill <id>` 观察或停止。

`goal logs <goal-id>` 查看 Goal 状态机事件；`logs <background-id>` 查看后台进程
输出，两者的 ID 不同。`goal start` 和 `goal resume` 都不会自动启动 worker；后者只把
Goal 恢复为 `active`，之后仍需再次 `goal run`。

TUI 支持相同的 `/goal help/start/status/inspect/list/logs/run/stop/resume/unlock` 契约。
没有活动目标时，裸 `/goal` 会显示新手帮助；有活动目标时仍作为 `status` 快捷方式。
`/goal` 是命令入口，`/goal 这个命令是做什么的` 会把中文内容当成未知子命令；自然语言
提问应直接发送，不要加 `/goal` 前缀。`Ctrl+C` 仍只停止当前响应/退出 TUI；Goal 只有
显式 `/goal stop` 才会进入 `stopped`。

## 自动压缩会话窗口

golang-cc 支持在发送模型请求前自动压缩会话上下文。该能力默认关闭；开启后会根据模型 context 大小和阈值比例判断是否触发压缩，并把较早的完整 conversation rounds 替换为结构化摘要，保留最近 rounds、当前用户输入以及成对的 tool_use/tool_result。

例如 `gpt-5.5` 的 context 是 `200000`，阈值比例是 `0.5`，当估算上下文达到 `100000` tokens 时会触发自动压缩：

```yaml
autoCompact:
  enabled: true
  defaultThresholdRatio: 0.75
  preserveRecentRounds: 6
  summaryModel: ""
  maxSummaryTokens: 8000
  cooldownTurns: 1
  maxFailures: 3
  modelContext:
    gpt-5.5: 200000
  modelThresholdRatio:
    gpt-5.5: 0.5
```

压缩摘要使用固定结构输出，并叠加运行时提取的硬事实，例如文件路径、命令、URL、工具调用、错误和用户约束。摘要会经过基础质量校验；连续失败达到 `maxFailures` 后，本会话会暂停自动压缩，避免反复失败。

## Session Recap

TUI 支持 Claude Code 风格的会话底部 `※ recap:`。`/recap` 会基于当前 session transcript 生成短总结并写入 `recap_summary` entry；`/recap show` 只展示最新 recap。恢复 session 时，TUI 会把最新 recap 作为 UI-only 消息显示，但 recap 不会进入下一轮模型上下文。

交互式 TUI 默认使用 idle-away recap：assistant 正常完成后，如果输入区保持空闲 90 秒且没有权限弹窗、picker、slash 待选或正在运行的 turn，后台生成并刷新底部 recap。想快速调短等待时间，只需要在 `config/config.yaml` 或 `~/.golang-cc/settings.json` 增加：

```yaml
recap:
  awayDelaySeconds: 10
```

也可以通过 CLI 写入全局或指定作用域的配置：

```bash
golang-cc config set recap.awayDelaySeconds 120
golang-cc config get recap.awayDelaySeconds
golang-cc config unset recap.awayDelaySeconds
```

完整配置示例：

```yaml
recap:
  enabled: true
  mode: away
  model: ""
  recentMessageWindow: 30
  maxTokens: 512
  awayDelaySeconds: 90
  includeCompactSummary: true
```

显式设置 `recap.enabled=false` 可以关闭 TUI 自动 recap；设置 `mode: manual` 会保留 `/recap` 手动生成；设置 `mode: post_turn` 会改为每轮 assistant 成功完成后异步刷新。idle-away 模式使用可测试的 TUI idle 检测，不依赖不同 terminal 支持不一致的 focus/blur 事件。

recap 生成会记录 `session.recap.started`、`session.recap.finished`、`session.recap.failed` telemetry 事件；事件只包含状态、耗时、模型和长度等元数据，不记录完整 prompt 或 recap 正文。

## 核心能力

| 方向 | 当前能力 |
| --- | --- |
| CLI / TUI | TUI 流式对话、slash commands、AskUserQuestion、权限审批、工具活动面板、usage 面板；CLI 支持 text/json/stream-json。 |
| Query loop | Anthropic Messages 流式客户端、多轮 `tool_use`、tool result、thinking/signature/citation/connector delta、prompt cache、可配置会话自动压缩。 |
| Tools | `Task`、`TaskOutput`、`AskUserQuestion`、文件读写编辑、搜索、Todo、LSP、Workflow、Worktree、WebBrowser、WebFetch、WebSearch、PowerShell、Bash、MCP resources。 |
| Session | transcript（默认 v2 消息图，append-only 树 + `parent_id`/`branch_head`）、resume（leaf-walk 当前链）、checkpoint、非破坏性 rewind/redo（带文件还原）、branches 分支查看/对比、fork、interrupted-turn 修复、自动 checkpoint、导出和 session CRUD。 |
| Skills / Plugins | 渐进式 metadata catalog、lazy `SKILL.md`、paths 过滤、runtime frontmatter、forked skill、marketplace、bundled/MCP/plugin roots。 |
| Multi-agent | `Task` sub-agent runtime，独立 transcript/tool loop/model/tools，支持 batch、priority、timeout、retry、cancel、nested progress。 |
| Permissions / Sandbox | Claude Code 兼容的 `permissions.allow` / `deny` / `ask` / `defaultMode` / `additionalDirectories`、source precedence、TUI/headless/MCP 审批、macOS Seatbelt、Linux bubblewrap/seccomp 主链路；legacy `alwaysAsk` 仍兼容读取。 |
| API Server | Gin server、`/query`、OpenAI-compatible `/v1/chat/completions` JSON/SSE、Swagger、metrics、tenant/mobile API。 |
| Mobile Chat | JWT、session CRUD、SSE、WebSocket 同步、cancel、regenerate、branch、附件 metadata、S3 presign、quota/rate limit。 |
| 多租户 | MySQL tenant/user/session/message/memory/document/profile/skill/audit/telemetry/agent-task 持久化。 |
| 可观测 | zap/slog 统一日志、trace/user/tenant 字段、audit log、telemetry events、Prometheus metrics、HTTP exporter。 |

## golang-cc 自有能力

Claude Code 提供了优秀的本地 coding agent 设计参考；golang-cc 在借鉴这些思想的基础上，重点补强服务化、平台化、多租户和 Agent 开发底座能力。

| 增强方向 | 当前能力 | 适用场景 |
| --- | --- | --- |
| API Server | 内置 Gin server，支持 `/query`、`/v1/models`、OpenAI-compatible `/v1/chat/completions`、Swagger UI。 | 前端、移动端、后台系统通过 HTTP/SSE 调用。 |
| OpenAI-compatible 网关 | 兼容 Chat Completions 请求/响应、SSE、content parts、tool_calls、response_format。 | 复用 OpenAI SDK、前端 Chat 组件、模型网关。 |
| 自定义 Agent 底座 | `Task` sub-agent runtime、agent prompt/model/tools、批量并发、priority、timeout、retry、cancel。 | 构建主 agent 分发任务、子 agent 并行处理、结果回传主会话。 |
| 多租户 SaaS 化 | tenant、user、skills、sessions、messages、audit、telemetry、agent tasks。 | 一个服务承载多个组织、多个用户和隔离数据。 |
| 移动端 Chat 后端 | JWT、SSE、WebSocket、附件、异步标题、regenerate、branch、quota/rate limit。 | iOS/Android 类 ChatGPT 移动端体验。 |
| 企业治理 | RBAC、audit log、telemetry、trace id、日志脱敏、Prometheus/HTTP exporter。 | 排查、审计、成本控制、性能分析。 |
| 模型稳定性 | YAML 多 fallback provider，OpenAI-compatible/custom provider。 | 私有网关、多模型灾备、降低 provider 故障影响。 |

## API Server 快速启动

```bash
go run ./cmd/golang-cc server \
  --host 127.0.0.1 \
  --port 8080 \
  --auth-token test-token
```

健康检查：

```bash
curl -sS http://127.0.0.1:8080/health \
  -H 'Authorization: Bearer test-token'
```

OpenAI-compatible 调用：

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Authorization: Bearer test-token' \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-5.5","messages":[{"role":"user","content":"hi"}]}'
```

Swagger UI：

```text
http://127.0.0.1:8080/swagger/index.html
```

Trace Viewer WebUI：

```text
http://127.0.0.1:8080/trace?token=test-token
```

Trace Viewer 可展示本地 TUI/CLI transcript 会话，以及 API Server / Mobile / OpenAI-compatible tenant session 的全链路 timeline、tool/model 耗时、tokens、permission、skill、sub-agent 和错误事件。

Prompt Dump Viewer：

```bash
GOLANG_CC_DUMP_PROMPT_JSON=/tmp/golang-cc-tui-prompt.jsonl \
GOLANG_CC_DUMP_PROMPT_FULL=true \
go run ./cmd/golang-cc --cwd /path/to/project

go run ./cmd/golang-cc server --host 127.0.0.1 --port 8080 --auth-token test-token
```

```text
http://127.0.0.1:8080/prompt-dump?token=test-token
```

Prompt Dump Viewer 默认读取 `/tmp/golang-cc-tui-prompt.jsonl`，也会优先使用当前 server 进程的 `GOLANG_CC_DUMP_PROMPT_JSON`。页面支持按 TUI 欢迎页里的 `sessionId` 筛选请求、查看完整 raw request、system/messages/tools/context summary，并尝试按同一 session 读取本地 Trace usage 里的 cache read/write/hit rate。完整 raw prompt 可能包含敏感上下文，不要提交 dump 文件。

Trace Viewer 的页面结构、数据来源和 `go:embed` 模板说明见 [docs/observability/trace_viewer.md](docs/observability/trace_viewer.md)。远程部署见 [docs/deployment/api_server_remote_deploy.md](docs/deployment/api_server_remote_deploy.md)，完整 API 见 [docs/api_server.md](docs/api_server.md)。

WebUI Mobile Chat Lab：

```bash
# 终端 1：启动 Go API Server
export GOLANG_CC_MOBILE_DEV_AUTH=true
go run ./cmd/golang-cc server \
  --host 127.0.0.1 \
  --port 8080 \
  --auth-token test-token

# 终端 2：启动独立前端工程
npm --prefix web install
npm --prefix web run dev
```

`web/` 是独立 Vite + React + TypeScript 前端工程，P0 用于测试 `/mobile/chat/*` 会话、SSE 聊天、memory、profile、skills 和 trace 主链路。前端维护说明见 [docs/webui/webui_frontend.md](docs/webui/webui_frontend.md)，详细技术计划见 [docs/webui/webui_mobile_chat_lab_plan.md](docs/webui/webui_mobile_chat_lab_plan.md)。

浏览器冒烟测试：

```bash
npm --prefix web run test:e2e
```

真实 API Server + MySQL 的 WebUI live E2E 需要先启动 server，再运行：

```bash
GOLANG_CC_WEBUI_LIVE_API_BASE=http://127.0.0.1:8080 \
GOLANG_CC_WEBUI_LIVE_API_TOKEN=test-token \
npm --prefix web run test:e2e:live
```

构建后也可以由 API Server 托管：

```bash
npm --prefix web run build
export GOLANG_CC_WEBUI_DIR=web/dist
go run ./cmd/golang-cc server --host 127.0.0.1 --port 8080 --auth-token test-token
```

访问 `http://127.0.0.1:8080/webui/?token=test-token`。

## 多租户与持久化

启用 MySQL：

```bash
export GOLANG_CC_MYSQL_DSN='root@tcp(127.0.0.1:3306)/golang_cc?multiStatements=true&parseTime=true'
go run ./cmd/golang-cc tenant migrate up
go run ./cmd/golang-cc tenant migrate version --json
```

请求携带 `X-Tenant-Key` 和 `X-User-Id` 时，`/query` 与 `/v1/chat/completions` 会自动创建或更新 tenant session，并写入 user/assistant messages。详细 schema 和 API 见 [docs/tenant/multi_tenant_mysql.md](docs/tenant/multi_tenant_mysql.md)。

## 提示词与项目规则

- 运行时身份是 `golang-cc, a Go-developed universal agent super assistant`。
- 设计目标是建设 golang-cc 自己的 agent runtime；Claude Code-compatible 能力用于迁移和生态兼容，不是最终产品边界。
- `golang-cc.md` 会作为默认 durable memory 注入 runtime prompt；文件名可通过 `identity.guidanceFilename` 配置。code 模式还会按当前 `cwd` 兼容加载原生 Claude Code 的 `~/.claude/projects/<sanitized-cwd>/memory/MEMORY.md` project memory index。
- 同一项目目录内，项目指导优先级为 `golang-cc.md` -> legacy `go-claude.md` -> legacy `CLAUDE.md` -> `AGENTS.md` fallback；只有该目录没有前三者时，才把同目录 `AGENTS.md` 作为低优先级 workflow fallback 注入 runtime prompt。
- prompt section、memory、output style、skills catalog、prompt cache 的完整加载链路见 [docs/prompt_logic/prompt_loading.md](docs/prompt_logic/prompt_loading.md)。

## 开发与验证

常规检查：

```bash
go test ./... -count=1
git diff --check
```

CI（[`.github/workflows/ci.yml`](.github/workflows/ci.yml)）在 push 和 PR 上跑六个 job，本地可以逐条复现：

```bash
go build ./... && go vet ./... && gofmt -l .   # build job，gofmt 输出非空即失败
go test -race ./... -count=1                   # test job，约 25 分钟
govulncheck ./...                              # govulncheck job
scripts/offline-acceptance.sh                  # scripts job，无需 API key 的验收子集
cd web && npm ci --legacy-peer-deps && npm run build && npm run test   # web job

# swagger job：生成物必须与注解一致，diff 非空即失败
swag init -g cmd/golang-cc/main.go --parseInternal --parseDependency
npm --prefix web run generate:api-types
git diff --exit-code -- docs/swagger.json docs/swagger.yaml docs/docs.go web/src/lib/generated/api-types.ts
```

发布链路是独立的 [`.github/workflows/release.yml`](.github/workflows/release.yml)，只在推 `v*` tag
或手动触发时跑，不参与每次 push 的质量门；本地等价命令是 `scripts/release.sh`。

`govulncheck` 需先安装：`go install golang.org/x/vuln/cmd/govulncheck@latest`。它按 Go 版本判定
stdlib 漏洞，所以本地务必用与 CI 相同的 Go 1.26.6，否则结论不一致。

API 修改后：

```bash
go test ./internal/server -count=1
swag init -g cmd/golang-cc/main.go --parseInternal --parseDependency
```

MySQL e2e：

```bash
export GOLANG_CC_MYSQL_E2E_DSN='root@tcp(127.0.0.1:3306)/golang_cc_e2e?multiStatements=true&parseTime=true'
go test ./internal/tenant ./internal/server -run MySQLE2E -count=1
```

真实浏览器 WebBrowser：

```bash
npm run webbrowser:self-test
go test ./internal/tools/webbrowser -count=1
```

开发规则见 [AGENTS.md](AGENTS.md)。新增 API 时要补测试、补 Swagger 注释、跑 Swagger 生成、做 curl 全链路验证，并检查数据库落库字段是否符合预期。

## 目录结构

```text
.
├── cmd/golang-cc/        CLI 入口
├── config/                        YAML 配置、环境配置和本地覆盖配置
├── docs/                          设计、API、兼容性和 Swagger 文档
├── internal/
│   ├── agentruntime/              sub-agent runtime
│   ├── agenteval/                 deterministic agent eval harness 和报告输出
│   ├── anthropic/                 model provider 适配层
│   ├── cli/                       参数解析、命令分发和 CLI golden tests
│   ├── config/                    环境变量、settings、YAML 配置
│   ├── files/                     大文件 chunk、manifest、流式读写辅助
│   ├── goal/                      Goal 模式状态机、本地 store、events、evaluator、runner
│   ├── hooks/                     Hook lifecycle 和 JSON payload
│   ├── mcp/                       MCP stdio/http client、tools/resources/prompts
│   ├── observability/             结构化日志和 trace/user/tenant 字段
│   ├── permissions/               权限策略、rule matcher、source precedence
│   ├── plugins/                   plugins/skills/agents 发现
│   ├── promptcache/               prompt cache policy 和 TTL 语义
│   ├── query/                     主对话循环、tool_use 编排、stream-json
│   ├── sandbox/                   Shell / OS sandbox runtime
│   ├── server/                    Gin API server、OpenAI-compatible、mobile API
│   ├── session/                   transcript（v2 消息图）、resume、checkpoint、rewind/redo、branches、usage
│   ├── skills/                    skills 渐进式加载、marketplace、watch、feedback
│   ├── storage/mysql/             MySQL migration runner、GORM/raw SQL repository
│   ├── telemetry/                 结构化 telemetry 事件和 sinks
│   ├── tenant/                    多租户 service 编排层
│   ├── tools/                     工具接口、注册表和各工具实现
│   └── tui/                       Bubble Tea 交互式终端 UI
├── migrations/mysql/              多租户 MySQL migrations
└── scripts/                       Playwright browser runner 等脚本
```

## 文档索引

完整文档中心见 [docs/README.md](docs/README.md)。常用入口：

| 场景 | 文档 |
| --- | --- |
| 分模式使用说明：TUI / CLI / agent-webui 场景 + 命令 | [docs/usage/README.md](docs/usage/README.md) |
| 最小代码 runtime：`--bare` 用法、行为契约和限制 | [docs/usage/bare.md](docs/usage/bare.md) |
| 运行架构、TUI/CLI/API Server 时序 | [docs/architecture/runtime_modes.md](docs/architecture/runtime_modes.md) |
| 会话存档：transcript / checkpoint / resume / inspect 图解与命令 | [docs/session_quickstart.md](docs/session_quickstart.md) |
| transcript v2 消息图 / 非破坏 rewind·redo·branches 设计与实现 | [docs/transcript/README.md](docs/transcript/README.md) |
| 全局配置 `settings.json` 快速上手：完整示例 + 字段速查 | [docs/examples/settings_quickstart.md](docs/examples/settings_quickstart.md) |
| 配置、提示词、memory、prompt cache | [docs/prompt_logic/prompt_loading.md](docs/prompt_logic/prompt_loading.md) |
| Agentic Design Patterns 智能体学习-理论和golang-cc实践 | [docs/agentic_patterns_learning/README.md](docs/agentic_patterns_learning/README.md) |
| WebSearch 国内搜索端点 | [docs/web_search.md](docs/web_search.md) |
| API Server、OpenAI-compatible、Mobile API | [docs/api_server.md](docs/api_server.md) |
| 远程部署给前端访问 | [docs/deployment/api_server_remote_deploy.md](docs/deployment/api_server_remote_deploy.md) |
| 多租户 MySQL 和 e2e 验证 | [docs/tenant/multi_tenant_mysql.md](docs/tenant/multi_tenant_mysql.md)、[docs/testing/mysql_e2e.md](docs/testing/mysql_e2e.md) |
| Web Agent 页面真机 E2E 和使用说明 | [中文](docs/web_agent/web_agent_real_e2e_usage_zh.md)、[English](docs/web_agent/web_agent_real_e2e_usage.md) |
| Web Agent session/conversation 分层模型 | [docs/web_agent/web_agent_session_conversation_model.md](docs/web_agent/web_agent_session_conversation_model.md) |
| Web Agent 生命周期、runner 接入和全链路修复计划 | [docs/web_agent/web_agent_lifecycle_runner_fix_plan.md](docs/web_agent/web_agent_lifecycle_runner_fix_plan.md) |
| Agent 拉练平台评分体系 | [docs/testing/agent_proving_ground_scoring_design.md](docs/testing/agent_proving_ground_scoring_design.md)、[docs/testing/agent_eval_harness.md](docs/testing/agent_eval_harness.md) |
| Goal 模式和 `/loop` 定时任务 | [docs/goal_mode/goal_mode_design.md](docs/goal_mode/goal_mode_design.md)、[docs/architecture/loop_scheduler_design.md](docs/architecture/loop_scheduler_design.md) |
| Skills / Plugins 渐进式加载 | [docs/skills/skills_progressive_loading.md](docs/skills/skills_progressive_loading.md) |
| 架构边界、金标、兼容性参考 | [docs/architecture/go_claude_agent_capability_boundaries.md](docs/architecture/go_claude_agent_capability_boundaries.md)、[docs/compatibility_matrix.md](docs/compatibility_matrix.md)、[docs/compatibility_deep_review.md](docs/compatibility_deep_review.md) |
| TODO、优先级、验收标准 | [docs/todo.md](docs/todo.md) |
| Swagger / OpenAPI | [docs/swagger.json](docs/swagger.json)、[docs/swagger.yaml](docs/swagger.yaml) |

## 当前边界

当前项目已经覆盖本地 Coding Agent 所需的主要 CLI、工具、权限、沙箱、会话、skills/plugins、API server、多租户和 telemetry 主链路。下面列的是当前工程边界和后续演进点，衡量标准以 golang-cc 的用户价值、工程质量和生态兼容收益为主。

| 模块 | 当前状态 |
| --- | --- |
| CLI / query / stream-json | 主链路和 stream event 已有金标覆盖，并保留兼容性参考。 |
| Goal mode | P0 Local Goal Mode 已实现：本地 JSONL store、CLI/TUI slash commands、RunOnce、自动 checkpoint、events/usage/status/blocker 记录。 |
| Prompt / cache | section registry、prompt cache scope/boundary/querySource TTL、message cache breakpoint、GrowthBook/feature gate 主链路已实现。 |
| Session | checkpoint、fork、resume-at、interrupted-turn 修复已实现；v2 消息图 + 非破坏性 rewind/redo（带文件还原）+ branches 分支查看/对比默认开启。 |
| Permissions / sandbox | macOS/Linux 主链路已接入；Windows Job Object/AppContainer、WSL、PowerShell true OS sandbox 仍是剩余项。 |
| Linux sandbox | bubblewrap/seccomp 主链路已实现；真机 CI/e2e golden 仍需要 Linux runner 验证。 |
| Web tools | HTML fallback 和 Playwright runner 均已接入；真实浏览器能力取决于 Node/Chromium 环境。 |
| 外部兼容边界 | Claude Code 非公开内部语义不作为强制追随目标；只有对 golang-cc 用户体验、迁移成本或生态兼容有价值的部分才继续吸收。 |

完整能力边界见 [docs/architecture/go_claude_agent_capability_boundaries.md](docs/architecture/go_claude_agent_capability_boundaries.md)，后续 TODO 见 [docs/todo.md](docs/todo.md)。
