# FAQ

## go-e2e 是什么？

go-e2e 是一个用 Go 编写的本地优先开源 Coding Agent，覆盖 Desktop、CLI、TUI 和 WebUI，支持 MCP、Skills、自定义 Provider、会话持久化和持续任务执行。

## 它和普通聊天机器人有什么不同？

go-e2e 面向任务交付：Agent 可以检查项目、调用工具、修改文件、执行命令、运行测试并报告验证结果。它仍然需要你配置模型服务，具体效果取决于模型、权限和任务描述。

## 支持哪些系统？

源码路径要求 Go 1.25+。桌面 Release 目标包括 macOS Apple Silicon、macOS Intel、Windows 和 Linux x86_64；请以具体 Release 的资产和说明为准。

## 必须使用某一家模型吗？

不必须。项目支持自定义 Provider 和兼容 API，但不同服务的协议、工具调用、多模态和上下文能力可能不同。建议先用只读文本任务验证。

## 是否支持 MCP？

支持。MCP 用于连接外部工具和服务；接入时仍应遵守 go-e2e 的权限、审批、网络和数据边界。见 [MCP 扩展](mcp.md)。

## 是否支持 Skills？

支持。Skills 用于封装可复用的任务知识和工作流。请从 [使用文档](usage/README.md) 开始，区分本机文件型 Skill、租户 Skill 和 MCP 能力。

## 数据会上传到哪里？

默认本地运行和 SQLite 存储，但模型请求会发送到你配置的 Provider，远程部署时还可能使用 MySQL、渠道和外部工具。部署前请根据你的数据边界审查 Provider、网络、权限和日志配置。

## macOS 为什么提示无法验证开发者？

未配置 Apple Developer 签名和公证的构建可能触发系统提示。可以先按照系统的“隐私与安全性”提示允许打开，也可以从源码运行 CLI 验证功能。

## 如何报告 Bug？

先搜索已有 [Issues](https://github.com/konglong87/go-e2e/issues)，再使用 Bug 模板提交：版本、平台、入口、Provider、最小复现、日志和期望结果。不要提交真实凭据或敏感业务数据。

## 如何贡献 PR？

文档、示例、Provider 配置、MCP 集成和测试改进都欢迎。先看 [贡献指南](../CONTRIBUTING.md)、[路线图](roadmap.md) 和带有 `good first issue` 标签的任务。
