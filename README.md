# go-e2e

面向全端的开源桌面级 Agent：覆盖桌面端、TUI、CLI 和 WebUI，
本地优先、支持私有化部署，从一句话需求出发持续执行并交付结果。

<p align="center">
  <img src="docs/web_agent/images/go-e2e-desktop-v2.png" alt="go-e2e 最新桌面端界面" width="900">
</p>

<p align="center">
  <img src="docs/web_agent/images/go-e2e-together.gif" alt="go-e2e 两端汇合的 Blender 3D 动画" width="720">
</p>

<p align="center">
  <img src="docs/web_agent/images/go-e2e-webui-v2.png" alt="go-e2e WebUI 2.0，浏览器验收示例会话" width="720">
</p>

## 六大核心亮点

**1. 开源桌面级 Agent，四端覆盖**

同时支持桌面端、TUI、CLI 和 WebUI，适合个人使用、开发调试和团队部署。

**2. 本地优先，部署方式自由**

默认支持本地运行和 SQLite，也支持私有化部署及 MySQL，数据边界和部署环境由用户掌控。

**3. 开放源码，方便二次开发**

模型、Provider、工具链、工作流、权限、界面和业务逻辑均可扩展。

**4. 从一句话到结果交付**

Agent 可以理解目标、拆解任务、调用工具、持续执行，并在同一会话中恢复上下文。

**5. Provider 自由接入，多模型扩展**

支持自定义 Provider 和兼容 API 的模型，也为多模态、画图等能力保留扩展空间。

**6. Go 内核，轻量高效**

使用 Go runtime 构建，启动和运行开销较低，适合本地长期运行与私有化部署。

## 一句话版

开源桌面级 Agent，覆盖桌面端、TUI、CLI 和 WebUI；本地优先、支持私有化部署，
能够从一句话需求出发完成任务执行与结果交付，并通过自定义 Provider 接入不同模型。

## 能力概览

- WebUI 2.0 与 `desktop-v2`：窄导航设置中心、模型、Profile、背景、3D 宠物、数据观测、飞书和 Multi-Agent。
- 会话标题支持三个点菜单和卡片右键修改，原位编辑并持久化；见[演示截图与验收记录](docs/webui/session-title-rename.md)。
- 连续评测：`--session-id` 可重复进入同一会话，追加对话并保留历史上下文；`--sessionId` 同样兼容。
- Provider-neutral 配置：模型、地址和凭据只从全局 `~/.golang-cc/settings.json` 读取，设置页面编辑的也是同一个文件。
- 数据库按部署场景选择：默认 SQLite，无需额外服务；需要多租户、审计和并发服务能力时可选 MySQL。
- 兼容外部项目指导文件格式：可读取 `go-e2e.md`、`AGENTS.md`、`CLAUDE.md` 等文件上下文，但不会读取外部模型配置或 API Key。

## 快速开始

### 工具链要求

- 源码最低要求 Go 1.25+。当前依赖图中 `x/crypto`、`x/net` 和 `x/sys` 等版本的最低要求是 Go 1.25。
- CI 与 release 默认使用 Go 1.26.6，作为包含最新安全修复的推荐构建工具链，不代表源码必须使用 Go 1.26。
- `.tool-versions` 中的 Go 版本是可复现构建 pin，不是源码最低版本。

```bash
git clone https://github.com/konglong87/go-e2e.git
cd go-e2e
go run ./cmd/go-e2e --version
```

在设置页面填写 provider、API 协议、模型、地址和凭据，或直接编辑：

```text
~/.golang-cc/settings.json
```

最小 provider-neutral 示例：

```json
{
  "provider": "custom",
  "providerProtocol": "openai-chat-completions",
  "baseURL": "https://model.example.com/v1",
  "apiKey": "replace-me",
  "model": "model-id"
}
```

开始一次任务：

```bash
go run ./cmd/go-e2e -p "检查当前项目结构"
```

继续同一会话：

```bash
go run ./cmd/go-e2e --session-id 11111111-1111-4111-8111-111111111111 -p "记住刚才的结论，并继续检查测试"
go run ./cmd/go-e2e --sessionId 11111111-1111-4111-8111-111111111111 -p "基于上一轮结果给出修复建议"
```

## 运行入口

| 场景 | 命令 |
| --- | --- |
| 终端交互 | `go run ./cmd/go-e2e` |
| 一次性任务 | `go run ./cmd/go-e2e -p "..."` |
| 服务端 | `go run ./cmd/go-e2e server` |
| 会话恢复 | `go run ./cmd/go-e2e --session-id <uuid> -p "..."` |
| WebUI 2.0 | `scripts/webui-dev.sh` |
| 桌面端 | `scripts/build-desktop-v2.sh` |

## 数据库

默认使用 SQLite，无需额外服务。需要多租户、审计、并发服务能力时，可选择 MySQL。数据库由用户根据部署场景自行选择，桌面端不会强制安装 MySQL。

## Provider 边界

运行时保留 Messages 协议和对应 SDK 适配层，作为独立的兼容 provider；产品默认配置、推荐文案和示例均保持 provider-neutral，不绑定任何单一模型厂商。

## 文档

- [详细技术说明](README.detailed.md)
- [文档索引](docs/README.md)
- [桌面端说明](desktop-v2/README.md)
- [全局设置示例](docs/examples/settings_quickstart.md)

## License

Apache-2.0
