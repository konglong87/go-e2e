<p align="center">
  <img src="web/src/v2/assets/go-e2e-mark.svg" alt="go-e2e logo" width="160">
</p>

<p align="center">
  <strong>go-e2e</strong>
</p>

<p align="center">
  用 Go 构建的本地优先开源 AI Coding Agent，支持 Desktop、CLI、TUI、WebUI、MCP、Skills、多智能体协作与多模型 Provider。
</p>

<p align="center">
  一句话输入 · 持续执行 · 结果验证 · 自主可控
</p>

<p align="center">
  面向高效执行：只读任务安全并行 · Skills 按需加载 · 流式刷新节流 · 长任务上下文治理
</p>

<p align="center">
  <a href="https://github.com/konglong87/go-e2e/releases/latest">下载桌面端</a> ·
  <a href="docs/getting-started.md">5 分钟上手</a> ·
  <a href="docs/use-cases.md">使用场景</a> ·
  <a href="https://github.com/konglong87/go-e2e/discussions">社区讨论</a> ·
  <a href="README.en.md">English</a>
</p>

<p align="center">
  <a href="https://github.com/konglong87/go-e2e/releases/latest">
    <img src="https://img.shields.io/github/v/release/konglong87/go-e2e?display_name=tag&sort=semver" alt="Latest release">
  </a>
  <a href="https://github.com/konglong87/go-e2e/releases">
    <img src="https://img.shields.io/github/downloads/konglong87/go-e2e/total" alt="Downloads">
  </a>
  <a href="https://github.com/konglong87/go-e2e/actions/workflows/ci.yml">
    <img src="https://github.com/konglong87/go-e2e/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI">
  </a>
  <a href="https://github.com/konglong87/go-e2e/actions/workflows/release.yml">
    <img src="https://github.com/konglong87/go-e2e/actions/workflows/release.yml/badge.svg" alt="Release">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/konglong87/go-e2e" alt="License">
  </a>
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
</p>

<p align="center">
  <img src="docs/web_agent/images/go-e2e-desktop-v2.png" alt="go-e2e desktop coding agent workspace" width="900">
</p>

## go-e2e 是什么？

go-e2e 是一个本地优先的开源 Coding Agent 工作台：你可以用自然语言描述任务，
让它理解项目、规划步骤、调用工具、修改代码、执行命令并报告验证结果。
它用 Go 编写，既可以作为个人桌面应用使用，也可以通过 CLI、TUI 或 WebUI 接入你的开发流程。

它不是只返回一段聊天文本的助手，而是围绕“任务执行和结果交付”设计的 Agent runtime。

## 适合哪些场景？

- **日常开发**：分析仓库、实现功能、修复 Bug、运行测试并总结变更。
- **终端自动化**：通过 CLI / TUI 执行一次性任务、持续会话和脚本化工作流。
- **本地优先工作台**：在自己的电脑或私有环境中保存会话、配置模型和管理权限。
- **Agent 二次开发**：通过 Provider、Skills、MCP、Tools、Profile 和 Web/API 扩展能力。
- **团队与渠道协作**：使用 WebUI、MySQL、飞书渠道和多 Agent 能力组织长期任务。

更多可复制的任务示例见 [使用场景](docs/use-cases.md)。

## 30 秒快速开始

### 方式一：下载桌面端

从 [GitHub Releases](https://github.com/konglong87/go-e2e/releases/latest) 下载对应平台的安装包。
打开应用后，在设置中填写模型 Provider、API 协议、模型、地址和凭据，然后输入第一条任务。

### 方式二：从源码运行 CLI

```bash
git clone https://github.com/konglong87/go-e2e.git
cd go-e2e
go run ./cmd/go-e2e --version
go run ./cmd/go-e2e -p "检查当前项目结构，告诉我最值得先改进的地方"
```

最低源码要求为 Go 1.26+。配置文件默认位于 `~/.golang-cc/settings.json`，最小的
OpenAI-compatible 配置示例：

```json
{
  "provider": "custom",
  "providerProtocol": "openai-chat-completions",
  "baseURL": "https://model.example.com/v1",
  "apiKey": "replace-me",
  "model": "model-id"
}
```

完整安装、配置和排障路径见 [5 分钟上手](docs/getting-started.md)。

## 为什么选择 go-e2e？

| 关注点 | go-e2e 的做法 |
| --- | --- |
| **本地优先** | 默认使用 SQLite，可在本机运行，也支持私有化部署和 MySQL。 |
| **真正执行任务** | 从需求理解、规划、工具调用到代码修改、命令执行和结果验证形成闭环。 |
| **入口自由** | Desktop、CLI、TUI、WebUI 和飞书渠道共享同一套 Runtime Core。 |
| **模型不锁定** | 支持自定义 Provider、兼容 API、多模型和多模态扩展。 |
| **便于扩展** | Skills、MCP、Tools、Profile、权限和工作流都有明确扩展边界。 |
| **Go 内核** | 启动和部署链路简单，适合本地工具、服务端和长期运行任务。 |

## 面向高效执行的设计

go-e2e 的效率优化聚焦在 Agent 自己能控制的部分：减少不必要的串行等待、重复渲染和上下文负担，同时为有副作用的操作保留安全边界。

| 效率机制 | 实际做法 | 带来的好处 |
| --- | --- | --- |
| **独立只读工具并行** | 对明确标记为并发安全的只读调用，在满足权限与运行条件时最多使用 4 个 worker 并行执行；结果仍按模型给出的顺序提交。写操作和不满足条件的调用不会因此被并行放行。 | 多项互不依赖的检查可以同时进行，同时保留可预测的副作用边界。 |
| **Skills 按需加载** | 预载 Skill 名称和精简描述，只有需要使用某项 Skill 时才加载完整内容。 | 避免每轮都把所有专业说明塞进上下文。 |
| **流式刷新节流** | 合并高频文本增量和界面刷新，减少终端对完整视图的重复重绘；任务结束时仍会刷新最终内容。 | 输出持续到达时减少本地渲染的重复工作。 |
| **长任务上下文治理** | 组合持久化 Session、Checkpoint、上下文压缩、项目指导文件与项目级 Markdown Memory。 | 支持跨进程续做，并控制历史上下文和工具输出不断膨胀。 |

> 这些机制优化的是本地调度、渲染和上下文使用；模型推理速度与端到端响应时间仍取决于所选 Provider、网络、模型和任务。本项目不以未经同条件测量的跨 Agent 速度数字作宣传。

## 入口怎么选？

| 入口 | 适合你在什么时候使用 | 快速入口 |
| --- | --- | --- |
| **Desktop** | 需要模型设置、会话管理和可视化工作台 | [下载桌面端](https://github.com/konglong87/go-e2e/releases/latest) |
| **CLI** | 一次性任务、脚本、自动化和 CI 场景 | `go run ./cmd/go-e2e -p "..."` |
| **TUI** | 希望在终端内交互式使用 Agent | `go run ./cmd/go-e2e` |
| **WebUI** | 浏览器工作台、服务端部署和团队使用 | [WebUI 文档](docs/usage/webui.md) |
| **飞书** | 把 Agent 接入渠道消息和长期 Worker | [飞书渠道文档](docs/usage/feishu.md) |

## 桌面个性化：把 Coding Agent 变成你的工作台

Desktop 不只是运行 Agent 的窗口，也是一个可以长期使用、按个人习惯调整的编码工作台。当前的“皮肤”能力以**桌面陪伴角色 + 背景/视觉外观设置**为主：

| 能力 | 当前支持 |
| --- | --- |
| **桌面宠物 / 陪伴角色** | 在 `设置 → 宠物` 选择 **Go 伙伴** 或 **中国龙**，切换待机/专注/庆祝动画，调整大小、位置、状态气泡，并可在当前设备隐藏或恢复。 |
| **背景设置** | 在 `设置 → 外观` 设置背景颜色、背景图片 URL；Desktop 还支持上传、替换和移除本地 PNG/JPEG/WebP/GIF 背景。 |
| **视觉外观** | 调整遮罩、侧栏表面色、气泡透明度，以及空状态/对话/编辑区模糊，让长时间运行的 Agent 任务更适合你的工作环境。 |

这不是皮肤商城，也不是任意 3D 角色上传系统；当前内置角色通过可扩展的资源注册表管理。桌面端的本地背景保存在本机数据目录，不会作为模型请求内容发送给 Provider。

详细设置、数据边界和当前限制见 [桌面个性化指南](docs/usage/desktop-personalization.md)。

## 公开发布边界

项目当前仍处于 `0.x` 快速迭代阶段。桌面端、Computer Use、部分多 Agent 和渠道能力会继续演进，
请在升级前阅读 [CHANGELOG](CHANGELOG.md) 和 [路线图](docs/roadmap.md)。
未配置 Apple Developer Secrets 的 macOS 构建可能需要在“系统设置 → 隐私与安全性”中手动允许打开；
运行 Agent 需要你自行配置可用的模型服务和 API 凭据。

## 产品闭环

<p align="center">
  <img src="docs/web_agent/images/go-e2e-together.gif" alt="go-e2e 端到端语义闭环动画" width="720">
</p>

<p align="center">
  从需求输入到工具执行、上下文延续和结果交付，形成可持续的 Agent 工作闭环。
</p>

## 架构总览

go-e2e 的不同入口共享同一套运行时核心：先完成鉴权、租户隔离和入口路由，
再由 Runtime Core 组合 Profile、Agent Loop、Skills、MCP、Tools、Memory、Image Tool
和 Provider，最后把结果适配回终端、桌面、WebUI、API 或飞书。

```mermaid
flowchart TB
  A["CLI / TUI / Desktop / WebUI / Feishu Worker"] --> B["鉴权 / 租户 / 路由"]
  B --> C["Runtime Core"]
  C --> D["Profile / Team"]
  C --> E["Agent Loop"]
  C --> F["Skills / MCP / Tools"]
  C --> G["Memory / Image Tool"]
  C --> H["Provider"]
  C --> I["Session / Goal"]
  C --> J["权限 / 预算 / 完成检查"]
  C --> K["SQLite / MySQL / Inbox / Outbox"]
  C --> L["Telemetry / Audit / Quota"]
  J --> M["输出适配<br/>CLI / Desktop / WebUI / Feishu / API"]
```

小白可以这样理解：**Profile** 定义“智能体怎么工作”，**Assignment** 决定“哪个入口使用
哪个 Profile”，**Feishu** 表示连接哪个飞书机器人，**Worker** 则是让这个机器人持续在线的进程。
它们不是重复设置。完整图例、分层说明和设置项边界见
[产品架构总览](docs/architecture/agent_platform_architecture.md)。

## 运行入口

| 场景 | 命令 |
| --- | --- |
| 终端交互 | `go run ./cmd/go-e2e` |
| 一次性任务 | `go run ./cmd/go-e2e -p "..."` |
| 服务端 | `go run ./cmd/go-e2e server` |
| 会话恢复 | `go run ./cmd/go-e2e --session-id <uuid> -p "..."` |
| WebUI 2.0 | `scripts/webui-dev.sh` |
| 桌面端 | `scripts/build-desktop-v2.sh` |
| 飞书 Worker | 见[飞书渠道文档](docs/README.md) |

## 部署与模型

| 主题 | 支持方式 | 适用场景 |
| --- | --- | --- |
| 本地数据库 | SQLite，默认无需额外服务 | 个人使用、桌面端和本地开发 |
| 服务化数据库 | MySQL，可选多租户、审计和并发能力 | 团队部署和服务端运行 |
| 模型接入 | 自定义 Provider、兼容 API、Messages 协议适配 | 接入不同模型服务 |
| 模型配置 | 全局 `~/.golang-cc/settings.json`，WebUI 与桌面端共用 | 统一管理 Provider、地址、模型和凭据 |

桌面端不会强制安装 MySQL；远程部署时应配置认证并限制访问范围。

## 能力全景

go-e2e 不只是一个聊天界面，而是一套可以执行任务、连接外部工具、保留上下文并持续交付结果的 Agent 工作台。

| 能力领域 | 支持内容 | 用户收益 |
| --- | --- | --- |
| Agent 执行 | 工具调用、文件操作、命令执行、代码搜索、LSP、Workflow | 真正操作项目并完成任务 |
| Skills | 项目 Skills、用户 Skills、插件 Skills、Marketplace Skills | 按需扩展 Agent 的专业能力 |
| MCP | 外部工具、数据库、知识库和企业服务接入 | 连接已有工作系统 |
| 项目上下文 | 默认 `go-e2e.md`；兼容 `AGENTS.md`、`CLAUDE.md`、`.claude/` 规则 | 遵循项目约定和工作流程 |
| 项目级 Memory | `MEMORY.md` 索引、关联 Markdown 记忆文件、项目级召回 | 持续理解项目规则和历史经验 |
| Memory 治理 | 长期记忆、自动记忆审核、团队/托管范围记忆 | 控制记忆来源和有效范围 |
| Session / Goal | 会话恢复、Checkpoint、Rewind、长期 Goal 和后台运行 | 支持跨进程和长任务持续执行 |
| Tools | Bash、文件读写、搜索、浏览器、WebFetch、WebSearch、图片生成 | 覆盖开发、研究和自动化场景 |
| Profile / Multi-Agent | Agent Profile、Teams、子 Agent 和任务编排 | 针对不同任务进行分工协作 |
| Provider | 自定义 Provider、多模型、兼容 API 和多模态扩展 | 不绑定单一模型厂商 |
| 权限与安全 | 权限审批、Sandbox、路径边界、网络策略和审计 | 控制 Agent 可以做什么 |
| 桌面体验 | 宠物选择、桌面背景、主题、窗口状态和设置中心 | 打造个性化桌面工作台 |
| 部署与观测 | SQLite、MySQL、Trace、Telemetry、Usage 和日志 | 适合本地、私有化和团队部署 |

> 部分能力需要用户配置 Provider、MCP Server、外部凭据或数据库；这里列出的是系统支持范围，不代表所有扩展默认启用。

## 模块能力实现进度

下面的表格是基于当前代码、测试和架构文档整理的能力快照，帮助读者区分“已经具备的基础能力”和“仍需要真实环境验收的部分”。

能力快照：**2026-10-19**（Computer Use）；其余模块仍以各自文档记录的审查范围为准。

| 模块 | 进度 | 已经实现 | 当前边界 | 主要代码入口 |
| --- | --- | --- | --- | --- |
| Agent 主循环 | ✅ 已实现 | 多轮对话、工具调用、结果回灌、上下文压缩、取消和预算控制 | 不同入口下的长任务仍需要持续做真实验收 | `internal/query/`、`internal/agentruntime/` |
| 上下文管理 | ✅ 较完整 | 项目规则、运行状态、工具结果、记忆、摘要和历史会话可以组合进模型上下文 | 超长任务的上下文质量仍需要更多回归样本 | `internal/query/`、`internal/memory/`、`internal/compact/` |
| Tool 工具体系 | ✅ 已实现 | 统一工具接口、工具注册、参数校验、并发边界、结果大小限制和动态工具接入 | 工具数量增加后，还需要继续优化发现和上下文成本 | `internal/tools/`、`internal/toolpolicy/` |
| Skill 技能 | ✅ 较完整 | 支持项目、用户、插件、Marketplace、MCP 和租户 Skill；支持按需加载、工具限制、独立上下文、Hooks 和路径匹配 | 需要继续补充跨 CLI、TUI、WebUI 和租户场景的真实验收 | `internal/skills/`、`internal/tools/skill/`、`internal/plugins/` |
| MCP 外部工具协议 | ✅ 较完整 | 支持 stdio 和 Streamable HTTP，可发现并调用工具、读取资源、加载提示模板，也支持回调和权限确认 | 旧版 HTTP+SSE 尚未支持；外部真实 MCP Server 的完整 E2E 仍需补充 | `internal/mcp/`、`internal/tools/mcpresources/` |
| Memory 记忆 | ✅ 已实现 | 项目规则、用户偏好、团队记忆、自动记忆和项目记忆召回，并带有范围和大小限制 | 当前重点是文档型记忆，不等同于完整的向量知识库 | `internal/memory/`、`internal/session/` |
| Planning / Goal 任务规划 | ✅ 已实现 | 目标、步骤、依赖、风险、验收条件、预算、暂停、恢复和阻塞状态 | 更复杂的跨系统计划仍需要更多真实任务样本 | `internal/goal/`、`internal/tools/planmode/`、`internal/tools/todowrite/` |
| Evidence 结果证据 | 🟡 基础能力已实现 | 保存工具轨迹、测试、命令、Git、API、数据库和产物证据，并用验收条件阻止无证据完成 | 工具成功不一定代表业务目标成功；还需要证据可信等级和更强的独立验证器 | `internal/goal/evidence.go`、`internal/goal/evaluator.go` |
| Reflection 复盘与纠错 | 🟡 部分实现 | 有结果评估、失败重试、阻塞判断、能力跟进和压缩后恢复 | 普通对话并不是每一轮都经过独立的“检查—修复—再验证”循环 | `internal/goal/evaluator.go`、`internal/repair/` |
| Multi-Agent / A2A 多智能体协作 | ✅ 较完整 | 支持创建、查询、停止和互相发消息，具备任务存储、事件、取消、进度和预算边界 | 跨组织 A2A 协议仍不是当前主路径 | `internal/agentruntime/`、`internal/agenttasks/`、`internal/tools/agent/` |
| 人机协作 | ✅ 已实现 | 用户提问、权限确认、暂停等待、恢复和交互结果回传 | 高风险动作仍需要结合具体入口做真实交互验收 | `internal/tools/askuserquestion/`、`internal/permissions/`、`internal/pendinginput/` |
| 权限与安全边界 | ✅ 较完整 | 工具审批、危险命令识别、目录和敏感路径限制、网络策略、沙箱和审计 | 安全能力的最终效果还需要持续做攻击性回归测试 | `internal/permissions/`、`internal/sandbox/`、`internal/tools/guarded.go` |
| 评测与运行观测 | 🟡 基础能力已实现 | 有测试、Golden、运行轨迹、Trace、Telemetry、行为评测和能力评分脚本 | 还需要稳定的任务集、独立验证器、Pass@k / Pass^k 和失败首因分析 | `internal/agenteval/`、`internal/observability/`、`scripts/` |
| Computer Use 电脑操作 | 🟡 macOS 单屏 Developer Preview | 原生悬浮控制面板、窗口绑定、原子 turn result、receipt、observation、screenshot、Stop 和安全拒绝路径。<br><a href="docs/usage/computer-use.md">使用与验收说明</a> | 最新 macOS tagged desktop build 已完成一次真实模型闭环，并完成 helper/bridge fault recovery 与真实截图验收；第二物理屏幕、正式签名/公证/分发和跨平台 GUI 仍未宣称完成 | `desktop-v2/computer_overlay.go`、`internal/computeruse/`、`internal/computerbackend/`、`native/macos/` |
| RAG / 知识检索 | 🟡 部分实现 | 支持文件、网页和项目上下文检索，并能把结果带回 Agent | 目前还不是完整的“向量索引—语义检索—重排—来源引用”体系 | `internal/memory/`、`internal/tools/websearch/`、`internal/tools/webfetch/` |
| 持续进化 | 🟡 部分实现 | 可以沉淀项目记忆、运行经验、Skills 和评测结果 | 尚未形成自动从轨迹生成知识、程序或模型更新的完整闭环 | `internal/memory/`、`internal/skills/`、`internal/agenteval/` |

Computer Use 行更新：**2026-10-19**；其余模块仍以以上审查快照为准。

> 表中的“已实现”表示代码和测试中已经具备对应能力，不代表所有外部 Provider、MCP Server、数据库或桌面权限默认已经配置完成。

### Skills 设置怎么用

桌面端和 WebUI 的 `设置 → Skills 管理` 会把两类 Skill 分开显示：

| 类型 | 设置页支持 | 安全边界 |
| --- | --- | --- |
| **本机 Skills** | 查看来源、路径和 `SKILL.md` 详情 | 只读，不直接删除项目、用户或插件目录中的文件 |
| **租户 Skills** | 新建/更新、查看内容、启用/停用、查看历史版本和回滚 | 只影响当前租户，历史版本保留 |

因此，“查看”在设置页内完成；租户 Skill 的“安装/更新”就是保存一份租户版本，
“卸载”对应停用。Marketplace Skill 或 Plugin 的本机文件安装/卸载仍使用 CLI，
例如 `skills install`、`plugins install` 和 `plugins remove`，避免桌面端误删用户文件。
这不是两个重复的 Skills 目录：**本机 Skill 是电脑上的文件，租户 Skill 是服务端运行时使用的版本**。

## 多端体验

### WebUI 2.0

<p align="center">
  <img src="docs/web_agent/images/go-e2e-webui-v2.png" alt="go-e2e WebUI 2.0 浏览器界面" width="760">
</p>

<p align="center">
  会话、工具、权限、模型、Profile、Memory 和运行状态在同一个 Web 工作台中管理。
</p>

<details>
<summary>展开查看扩展能力：Skills、MCP、Hooks 与 Plugins</summary>

- **Skills**：按需加载项目、用户、插件、Marketplace 和 MCP 来源的技能，减少默认上下文开销。
- **MCP**：连接外部工具和服务，并沿用 Agent 的权限审批与运行边界。
- **Hooks**：在 Agent、Tool 和任务生命周期节点注入自定义逻辑。
- **Plugins**：扩展工具、Skills、Output Style、模型和运行行为。
- **Profile**：为不同任务组合提示词、上下文、模型、工具和权限策略。
- **Multi-Agent**：拆分子任务，由多个 Agent 协同完成复杂工作。

</details>

<details>
<summary>展开查看记忆与长任务能力</summary>

- **项目指导文件**：默认读取 `go-e2e.md`，并兼容 `AGENTS.md`、`CLAUDE.md`、`.claude/CLAUDE.md` 和 `.claude/rules/*.md`。
- **项目级 Memory**：以 `MEMORY.md` 作为索引，按关联文件召回项目级记忆，并对路径、大小和召回数量设有边界。
- **持久化 Session**：跨进程继续同一会话，保留消息、工具调用和任务上下文。
- **Memory**：保存项目规则、长期偏好和可复用上下文，并支持审核与治理。
- **Goal**：创建目标、持续运行、查看状态、暂停、恢复和记录事件。
- **Checkpoint / Rewind**：在关键节点保存检查点，支持会话恢复、回退和分叉。
- **Context 管理**：通过压缩、结构化摘要和证据关联控制长任务上下文规模。

</details>

<details>
<summary>展开查看工具、权限与安全能力</summary>

- **开发工具**：文件读取、文件编辑、Glob、Grep、Bash、Workflow 和 LSP。
- **网络工具**：WebFetch、WebSearch、浏览器自动化和外部服务调用。
- **多模态工具**：图片附件、图片生成和视觉任务扩展。
- **权限控制**：工具审批、权限来源、危险操作识别和审计记录。
- **Sandbox**：工作目录、额外目录、敏感路径、网络和命令执行边界。
- **可观测性**：运行日志、Trace、Usage、Telemetry 和任务事件回放。

</details>

<details>
<summary>展开查看桌面工作台与个性化能力</summary>

- **桌面端**：基于 `desktop-v2` 的原生桌面工作台，支持本地 sidecar 和就绪检查。
- **宠物选择**：支持 Go 伙伴、中国龙等桌面形象，并可在窗口内拖拽。
- **桌面背景**：支持背景和视觉主题配置，适配不同工作环境。
- **窗口状态**：保存窗口位置、尺寸、最大化和全屏状态。
- **设置中心**：集中管理模型、Provider、Profile、背景、宠物、观测和 Multi-Agent。
- **本地数据**：默认 SQLite，无需额外服务；需要团队能力时可选 MySQL。

</details>

### 渠道会话识别

桌面端会话列表会在标题旁标识消息来源，帮助区分普通桌面会话与渠道消息会话。
当前已支持飞书渠道会话：

| 会话来源 | 列表标识 | 悬浮信息 |
| --- | --- | --- |
| 普通桌面会话 | 无渠道标签 | 来源：桌面会话 |
| 飞书渠道会话 | 飞书 | 来源：飞书渠道 |

<p align="center">
  <img src="docs/web_agent/images/desktop-feishu-session-label.png" alt="go-e2e 桌面端会话列表中的飞书渠道标签" width="900">
</p>

截图展示了桌面端会话列表中的飞书渠道标签；后续新增渠道可以沿用同一来源标识方式。

完整的产品级架构图（含渠道、租户、Runtime Core、Agent Loop、能力插件、持久化、
遥测审计和输出适配）见：

- [产品架构总览](docs/architecture/agent_platform_architecture.md)
- [Mermaid 架构图源文件](diagrams/agent-platform-architecture.mmd)

## 社区与反馈

- **飞书社群**：[点击加入 go-e2e 中文社区](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=55aq8d1d-7586-46cf-8635-a8c321ad284d)，具体加入资格以飞书页面提示为准。
- **问题反馈**：通过 [GitHub Issues](https://github.com/konglong87/go-e2e/issues) 提交 Bug 和功能建议。
- **安全问题**：请阅读 [安全策略](SECURITY.md)，不要在公开 Issue 中提交敏感细节。
- **贡献代码**：请先阅读 [贡献指南](CONTRIBUTING.md)。

## 文档

- [详细技术说明](README.detailed.md)
- [文档索引](docs/README.md)
- [桌面端说明](desktop-v2/README.md)
- [全局设置示例](docs/examples/settings_quickstart.md)

## License

Apache-2.0
