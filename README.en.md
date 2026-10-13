<p align="center">
  <img src="web/src/v2/assets/go-e2e-mark.svg" alt="go-e2e logo" width="160">
</p>

<p align="center">
  <strong>go-e2e</strong>
</p>

<p align="center">
  An open-source, local-first coding agent written in Go — Desktop, CLI, TUI, WebUI, MCP, Skills, and flexible model Providers.
</p>

<p align="center">
  One sentence in · Continuous execution · Verified delivery · You stay in control
</p>

<p align="center">
  <a href="https://github.com/konglong87/go-e2e/releases/latest">Download</a> ·
  <a href="docs/getting-started.md">Get started in 5 minutes</a> ·
  <a href="docs/use-cases.md">Use cases</a> ·
  <a href="https://github.com/konglong87/go-e2e/discussions">Community</a> ·
  <a href="README.md">简体中文</a>
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
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
</p>

<p align="center">
  <img src="docs/web_agent/images/go-e2e-desktop-v2.png" alt="go-e2e desktop coding agent workspace" width="900">
</p>

## What is go-e2e?

go-e2e is a local-first, open-source coding agent workbench. Describe a task in natural language and it can
inspect a project, plan the work, use tools, edit code, run commands, and report verified results.
It is written in Go and can be used as a desktop application or integrated through the CLI, TUI, WebUI, and Feishu channel.

It is designed around task execution and delivery, not only chat responses.

## What can you use it for?

- **Daily development**: inspect repositories, implement features, fix bugs, run tests, and summarize changes.
- **Terminal automation**: run one-shot tasks, persistent sessions, and scriptable workflows through CLI / TUI.
- **Local-first workspaces**: keep sessions, model settings, and permission boundaries in your own environment.
- **Agent extension**: add Providers, Skills, MCP servers, Tools, Profiles, workflows, and Web/API integrations.
- **Team and channel workflows**: use WebUI, MySQL, Feishu channels, and multi-agent capabilities for longer-running tasks.

See [Use cases](docs/use-cases.md) for copyable task examples.

## Get started in 30 seconds

### Option 1: Download the desktop app

Download an installer for your platform from [GitHub Releases](https://github.com/konglong87/go-e2e/releases/latest).
Open the app, configure a model Provider, protocol, model, endpoint, and credentials, then enter your first task.

### Option 2: Run the CLI from source

```bash
git clone https://github.com/konglong87/go-e2e.git
cd go-e2e
go run ./cmd/go-e2e --version
go run ./cmd/go-e2e -p "Inspect the current project and identify the highest-value improvements"
```

The source minimum is Go 1.25+. The default settings file is `~/.golang-cc/settings.json`. A minimal
OpenAI-compatible configuration looks like this:

```json
{
  "provider": "custom",
  "providerProtocol": "openai-chat-completions",
  "baseURL": "https://model.example.com/v1",
  "apiKey": "replace-me",
  "model": "model-id"
}
```

See [Get started in 5 minutes](docs/getting-started.md) for installation, configuration, and troubleshooting.

## Why go-e2e?

| Concern | How go-e2e approaches it |
| --- | --- |
| **Local first** | SQLite by default, with local execution, private deployment, and MySQL support. |
| **Task execution** | A loop from task understanding and planning to tool use, code changes, commands, and verification. |
| **Flexible entry points** | Desktop, CLI, TUI, WebUI, and Feishu share one Runtime Core. |
| **Provider flexibility** | Custom Providers, compatible APIs, multiple models, and multimodal extensions. |
| **Designed to extend** | Explicit extension boundaries for Skills, MCP, Tools, Profiles, permissions, and workflows. |
| **Go runtime** | A straightforward runtime and deployment path for local tools and long-running services. |

## Which entry point should you choose?

| Entry point | Best when you need | Quick path |
| --- | --- | --- |
| **Desktop** | Model settings, session management, and a visual workspace | [Download](https://github.com/konglong87/go-e2e/releases/latest) |
| **CLI** | One-shot tasks, scripts, automation, and CI workflows | `go run ./cmd/go-e2e -p "..."` |
| **TUI** | Interactive Agent work directly in a terminal | `go run ./cmd/go-e2e` |
| **WebUI** | A browser workspace, server deployment, or team usage | [WebUI docs](docs/usage/webui.md) |
| **Feishu** | Channel messages and long-running workers | [Feishu channel docs](docs/usage/feishu.md) |

## Public release boundaries

The project is still in fast-moving `0.x` development. Desktop, Computer Use, parts of multi-agent execution, and
channel integrations will continue to evolve; read the [CHANGELOG](CHANGELOG.md) and [roadmap](docs/roadmap.md) before upgrading.
Unsigned macOS builds may require manual approval in System Settings, and running an Agent requires a model service and API credentials that you configure yourself.

## Product Loop

<p align="center">
  <img src="docs/web_agent/images/go-e2e-together.gif" alt="go-e2e end-to-end semantic loop" width="720">
</p>

<p align="center">
  From request input to tool execution, context continuity, and result delivery,
  go-e2e forms a continuous Agent work loop.
</p>

## Architecture Overview

All go-e2e entry points share the same runtime core. The runtime handles authentication,
tenant isolation, and entry routing, then composes Profiles, the Agent Loop, Skills, MCP,
Tools, Memory, Image Tools, and Providers. Results are adapted back to the terminal, desktop,
WebUI, API, or Feishu.

```mermaid
flowchart TB
  A["CLI / TUI / Desktop / WebUI / Feishu Worker"] --> B["Auth / Tenant / Routing"]
  B --> C["Runtime Core"]
  C --> D["Profile / Team"]
  C --> E["Agent Loop"]
  C --> F["Skills / MCP / Tools"]
  C --> G["Memory / Image Tool"]
  C --> H["Provider"]
  C --> I["Session / Goal"]
  C --> J["Permissions / Budget / Completion Checks"]
  C --> K["SQLite / MySQL / Inbox / Outbox"]
  C --> L["Telemetry / Audit / Quota"]
  J --> M["Output Adapters<br/>CLI / Desktop / WebUI / Feishu / API"]
```

For a simple mental model: **Profile** defines how an Agent works, **Assignment** decides
which entry point uses which Profile, **Feishu** identifies the Feishu bot and conversation,
and **Worker** keeps that bot online. They are separate, reusable concepts. See the
[product architecture overview](docs/architecture/agent_platform_architecture.md) for the
complete diagram, layers, and configuration boundaries.

## Entry Points

| Scenario | Command |
| --- | --- |
| Interactive terminal | `go run ./cmd/go-e2e` |
| One-shot task | `go run ./cmd/go-e2e -p "..."` |
| Server | `go run ./cmd/go-e2e server` |
| Resume a session | `go run ./cmd/go-e2e --session-id <uuid> -p "..."` |
| WebUI 2.0 | `scripts/webui-dev.sh` |
| Desktop | `scripts/build-desktop-v2.sh` |

## Deployment and Models

| Topic | Supported approach | Suitable for |
| --- | --- | --- |
| Local database | SQLite, no additional service by default | Personal use, desktop, and local development |
| Service database | MySQL with optional multi-tenant, audit, and concurrency capabilities | Team deployment and server operation |
| Model integration | Custom Providers, compatible APIs, and Messages protocol adapters | Connecting different model services |
| Model configuration | Shared `~/.golang-cc/settings.json` for WebUI and Desktop | Managing providers, endpoints, models, and credentials in one place |

The desktop application does not require MySQL. For remote deployment, configure
authentication and restrict network access appropriately.

## Capability Map

go-e2e is more than a chat interface. It is an Agent workbench that executes tasks, connects
external tools, preserves context, and delivers results continuously.

| Area | Supported capabilities | User value |
| --- | --- | --- |
| Agent execution | Tool calls, file operations, command execution, code search, LSP, and Workflows | Operate on a project and complete real tasks |
| Skills | Project, user, plugin, Marketplace, and MCP Skills | Add specialized capabilities on demand |
| MCP | External tools, databases, knowledge bases, and enterprise services | Connect existing work systems |
| Project context | `go-e2e.md`; compatible with `AGENTS.md`, `CLAUDE.md`, and `.claude/` rules | Follow project conventions and workflows |
| Project Memory | `MEMORY.md` index, linked Markdown memories, and project-level recall | Keep useful project context across tasks |
| Memory governance | Long-term memory, automatic memory review, and team/managed scopes | Control memory sources and their effective scope |
| Session / Goal | Session resume, Checkpoints, Rewind, long-running Goals, and background execution | Keep long tasks running across processes |
| Tools | Bash, file I/O, search, browser, WebFetch, WebSearch, and image generation | Cover development, research, and automation |
| Profiles / Multi-Agent | Agent Profiles, Teams, sub-agents, and task orchestration | Divide complex work among specialized Agents |
| Providers | Custom Providers, multiple models, compatible APIs, and multimodal extensions | Avoid dependence on one model vendor |
| Permissions and security | Approval flows, Sandbox, path boundaries, network policy, and audit | Control what the Agent can do |
| Desktop experience | Companion selection, backgrounds, themes, window state, and settings center | Build a personalized desktop workspace |
| Deployment and observability | SQLite, MySQL, Trace, Telemetry, Usage, and logs | Support local, private, and team deployments |

> Some capabilities require a configured Provider, MCP Server, external credentials, or database.
> This table describes the supported surface, not the default enabled configuration.

## Module Capability Implementation Progress

The table below is a snapshot based on the current code, tests, and architecture documents. It distinguishes capabilities that already have a working foundation from those that still need validation in a real environment.

Snapshot: **2026-09-24** · Code baseline: `fdc3497`

| Module | Progress | Implemented today | Current boundary | Main code entry points |
| --- | --- | --- | --- | --- |
| Agent loop | ✅ Implemented | Multi-turn interaction, tool calls, result feedback, context compaction, cancellation, and budget controls | Long-running tasks still need continued real-world acceptance across entry points | `internal/query/`, `internal/agentruntime/` |
| Context management | ✅ Mature foundation | Project rules, runtime state, tool results, memory, summaries, and session history can be composed into model context | Long-task context quality still needs more regression cases | `internal/query/`, `internal/memory/`, `internal/compact/` |
| Tool system | ✅ Implemented | Common tool interface, registry, input validation, concurrency boundaries, result limits, and dynamic tool loading | Tool discovery and context cost need continued optimization as the catalog grows | `internal/tools/`, `internal/toolpolicy/` |
| Skills | ✅ Mature foundation | Project, user, plugin, Marketplace, MCP, and tenant Skills; on-demand loading, tool restrictions, isolated contexts, hooks, and path matching | More cross-surface acceptance is needed across CLI, TUI, WebUI, and tenant flows | `internal/skills/`, `internal/tools/skill/`, `internal/plugins/` |
| MCP | ✅ Mature foundation | stdio and Streamable HTTP; tool discovery and calls, resource reads, prompt loading, callbacks, and permission confirmation | Legacy HTTP+SSE is not supported; full E2E with external MCP Servers still needs more coverage | `internal/mcp/`, `internal/tools/mcpresources/` |
| Memory | ✅ Implemented | Project rules, user preferences, team memory, automatic memory, and project recall with scope and size limits | The current focus is document-based memory, not a complete vector knowledge base | `internal/memory/`, `internal/session/` |
| Planning / Goals | ✅ Implemented | Goals, steps, dependencies, risks, acceptance criteria, budgets, pause, resume, and blocked states | More complex cross-system plans need additional real-task coverage | `internal/goal/`, `internal/tools/planmode/`, `internal/tools/todowrite/` |
| Evidence | 🟡 Foundation implemented | Stores tool traces, tests, commands, Git, API, database, and artifact evidence; acceptance criteria can block unsupported completion | Tool success does not always prove business success; evidence trust levels and stronger independent verifiers are still needed | `internal/goal/evidence.go`, `internal/goal/evaluator.go` |
| Reflection and recovery | 🟡 Partially implemented | Result evaluation, retries, blocker detection, capability follow-ups, and recovery after compaction | Ordinary conversations do not yet run an independent check-repair-reverify loop on every turn | `internal/goal/evaluator.go`, `internal/repair/` |
| Multi-Agent / A2A | ✅ Mature foundation | Create, inspect, stop, and message sub-agents with task storage, events, cancellation, progress, and budget boundaries | Cross-organization A2A is not the current primary path | `internal/agentruntime/`, `internal/agenttasks/`, `internal/tools/agent/` |
| Human-in-the-loop | ✅ Implemented | User questions, permission confirmation, pause/wait, resume, and interaction results | High-risk actions still need real interactive acceptance for each entry point | `internal/tools/askuserquestion/`, `internal/permissions/`, `internal/pendinginput/` |
| Permissions and safety | ✅ Mature foundation | Tool approval, dangerous-command detection, directory and sensitive-path limits, network policy, sandboxing, and audit records | Security behavior still needs ongoing adversarial regression testing | `internal/permissions/`, `internal/sandbox/`, `internal/tools/guarded.go` |
| Evaluation and observability | 🟡 Foundation implemented | Tests, Goldens, runtime traces, Trace, Telemetry, behavior evaluation, and capability scorecard scripts | Stable task sets, independent verifiers, Pass@k / Pass^k, and first-error analysis still need to be added | `internal/agenteval/`, `internal/observability/`, `scripts/` |
| Computer Use | 🟡 macOS desktop integration available; acceptance ongoing | Native floating controls, expand/collapse/hide, latest operation-image preview, session state, permission gates, and action receipts.<br><a href="docs/web_agent/images/computer-use-native-overlay.jpg"><img src="docs/web_agent/images/computer-use-native-overlay.jpg" alt="go-e2e macOS Computer Use native floating panel and operation-image preview" width="220"></a><br><sub>Desktop screenshot · Click to enlarge</sub> | Selected native clicks and preview verified; full-screen Spaces, complete controls, and autonomous model E2E still require acceptance. Second physical display and formal signing/distribution are deferred. | `desktop-v2/computer_overlay.go`, `internal/computeruse/`, `internal/computerbackend/`, `native/macos/` |
| RAG / knowledge retrieval | 🟡 Partially implemented | File, web, and project-context retrieval can feed results back into the Agent | It is not yet a full vector-index, semantic-retrieval, reranking, and source-citation pipeline | `internal/memory/`, `internal/tools/websearch/`, `internal/tools/webfetch/` |
| Continual evolution | 🟡 Partially implemented | Project memory, runtime experience, Skills, and evaluation results can be retained | There is not yet a complete automatic loop that turns trajectories into knowledge, programs, or model updates | `internal/memory/`, `internal/skills/`, `internal/agenteval/` |

Computer Use row updated: **2026-09-30**; other modules retain the audit baseline above.

> “Implemented” means that the capability exists in the code and tests. It does not mean that every external Provider, MCP Server, database, or desktop permission is configured by default.

### Skills Management

The Desktop and WebUI `Settings -> Skills` view separates two types of Skills:

| Type | Settings support | Safety boundary |
| --- | --- | --- |
| **Local Skills** | View source, path, and `SKILL.md` details | Read-only; does not directly delete files from project, user, or plugin directories |
| **Tenant Skills** | Create/update, view, enable/disable, inspect history, and roll back | Only affects the current tenant and preserves version history |

In the settings view, “view” means inspecting the Skill there. Installing or updating a tenant
Skill saves a tenant-scoped version; uninstalling it means disabling it. Local Marketplace or
Plugin file installation/removal remains a CLI operation such as `skills install`, `plugins install`,
or `plugins remove`, which avoids accidental deletion of local user files. Local Skills are files
on the computer; tenant Skills are server-side runtime versions.

## Multi-Surface Experience

### WebUI 2.0

<p align="center">
  <img src="docs/web_agent/images/go-e2e-webui-v2.png" alt="go-e2e WebUI 2.0 browser workspace" width="760">
</p>

<p align="center">
  Sessions, tools, permissions, models, Profiles, Memory, and runtime status are managed in one Web workspace.
</p>

<details>
<summary>Extension capabilities: Skills, MCP, Hooks, and Plugins</summary>

- **Skills**: Load project, user, plugin, Marketplace, and MCP Skills on demand to reduce default context cost.
- **MCP**: Connect external tools and services while using the Agent's permission and runtime boundaries.
- **Hooks**: Inject custom logic at Agent, Tool, and task lifecycle points.
- **Plugins**: Extend tools, Skills, Output Styles, models, and runtime behavior.
- **Profiles**: Combine prompts, context, models, tools, and permission policies for different tasks.
- **Multi-Agent**: Split subtasks and coordinate multiple Agents on complex work.

</details>

<details>
<summary>Memory and long-running tasks</summary>

- **Project guidance**: Reads `go-e2e.md` by default and supports `AGENTS.md`, `CLAUDE.md`, `.claude/CLAUDE.md`, and `.claude/rules/*.md`.
- **Project Memory**: Uses `MEMORY.md` as an index, recalls linked project memories, and enforces path, size, and recall limits.
- **Persistent Sessions**: Continue the same session across processes with messages, tool calls, and task context.
- **Memory**: Store project rules, long-term preferences, and reusable context with review and governance.
- **Goals**: Create, run, inspect, pause, resume, and record long-running goals.
- **Checkpoint / Rewind**: Save checkpoints at important points and recover, rewind, or branch sessions.
- **Context management**: Control long-task context size through compression, structured summaries, and evidence links.

</details>

<details>
<summary>Tools, permissions, and safety</summary>

- **Development tools**: File reading, editing, Glob, Grep, Bash, Workflow, and LSP.
- **Network tools**: WebFetch, WebSearch, browser automation, and external service calls.
- **Multimodal tools**: Image attachments, image generation, and visual task extensions.
- **Permission control**: Tool approval, permission sources, dangerous-operation detection, and audit records.
- **Sandbox**: Workspace, extra directories, sensitive paths, network, and command execution boundaries.
- **Observability**: Runtime logs, Trace, Usage, Telemetry, and task event replay.

</details>

<details>
<summary>Desktop workspace and personalization</summary>

- **Desktop**: Native `desktop-v2` workspace with a local sidecar and readiness checks.
- **Companions**: Go Companion, Chinese Dragon, and draggable desktop characters.
- **Backgrounds**: Background and visual theme configuration for different work environments.
- **Window state**: Persist window position, size, maximized state, and fullscreen state.
- **Settings center**: Manage models, Providers, Profiles, backgrounds, companions, observability, and Multi-Agent settings.
- **Local data**: SQLite by default, with optional MySQL for team capabilities.

</details>

### Channel Session Identification

The Desktop session list labels the message source next to the title, helping distinguish
regular desktop sessions from channel sessions. Feishu channel sessions are currently supported:

| Session source | List label | Tooltip |
| --- | --- | --- |
| Regular desktop session | No channel label | Source: desktop session |
| Feishu channel session | Feishu | Source: Feishu channel |

<p align="center">
  <img src="docs/web_agent/images/desktop-feishu-session-label.png" alt="Feishu channel label in the go-e2e Desktop session list" width="900">
</p>

The screenshot shows the Feishu channel label in the Desktop session list. Future channels can
reuse the same source-identification pattern.

For the complete product architecture, including channels, tenants, Runtime Core, Agent Loop,
capability plugins, persistence, telemetry, audit, and output adapters, see:

- [Product architecture overview](docs/architecture/agent_platform_architecture.md)
- [Mermaid architecture source](diagrams/agent-platform-architecture.mmd)

## Community and Feedback

- **Feishu community**: [Join the go-e2e Chinese community](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=55aq8d1d-7586-46cf-8635-a8c321ad284d). Eligibility is subject to the access rules shown by Feishu.
- **Issues**: Use [GitHub Issues](https://github.com/konglong87/go-e2e/issues) for bugs and feature requests.
- **Security**: Read the [security policy](SECURITY.md). Do not post sensitive vulnerability details in public Issues.
- **Contributing**: Read the [contribution guide](CONTRIBUTING.md) before submitting code.

## Documentation

- [Detailed technical notes](README.detailed.md)
- [Documentation index](docs/README.md)
- [Desktop notes](desktop-v2/README.md)
- [Settings quick start](docs/examples/settings_quickstart.md)

## License

Apache-2.0
