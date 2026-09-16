# 全局配置快速上手 · `~/.golang-cc/settings.json`

golang-cc 的全局配置放在 `~/.golang-cc/settings.json`（或 `$GOLANG_CC_CONFIG_DIR/settings.json`）。
本目录的 [settings.example.json](settings.example.json) 是可直接复制的完整示例。

```bash
mkdir -p ~/.golang-cc
cp docs/examples/settings.example.json ~/.golang-cc/settings.json
# 然后编辑，替换所有 "REPLACE" / "example.com" / "your-*" 占位值
```

> ⚠️ **不要把真实密钥提交进仓库。** 密钥放在全局 `~/.golang-cc/settings.json`（仓库外），或用环境变量 `${VAR}` 引用；本次调用的额外配置使用显式 `--settings <文件路径或JSON>`，不会自动扫描项目配置。

## 默认读取与显式覆盖（2026-09-09）

默认只读取 `~/.golang-cc/settings.json`；设置 `GOLANG_CC_CONFIG_DIR` 后只读取该目录的 `settings.json`。工作目录不参与默认文件搜索。原有显式目录/身份环境变量机制保留，本轮不调整环境变量优先级。

旧全局 `.go-claude/settings.json`、项目 `.claude` / `.go-claude` / `.golang-cc` / 自定义目录中的 settings/local 文件以及 `config/config*.yaml` 不再自动加载。旧文件、解析器和路径代码仍保留；缺失或损坏全局文件不会触发旧文件回退，错误与默认值处理沿用既有实现。

`--settings` JSON 文件/JSON 字符串、显式命令行参数和进程环境变量保持原有处理。需要复用额外 JSON 文件可明确传入（CLI 不接受 YAML）：

```bash
golang-cc --settings ./.golang-cc/settings.local.json -p "检查项目"
golang-cc --settings '{"effort":"medium"}' -p "检查项目"
```

项目配置写入命令仍保留，但写出的文件只有显式加载时才生效。WebUI 2.0 的模型表单、全局 JSON、文件生效预览均读取同一全局文件；启动快照、会话/Run 覆盖仍单独展示，保存不自动重启或改变活跃 Run。

## 最小配置（单一 Anthropic / 兼容网关）

```json
{
  "model": "claude-sonnet-4-6",
  "env": {
    "ANTHROPIC_API_KEY": "sk-REPLACE-WITH-YOUR-KEY"
  }
}
```

官方 Anthropic 无需写 `ANTHROPIC_BASE_URL`；用自建/第三方 Anthropic 兼容网关时才需要。

## OpenAI Responses-compatible 端点

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
    "ANTHROPIC_API_KEY": "sk-REPLACE-WITH-YOUR-KEY"
  }
}
```

- `baseURL` 填 API 根路径，client 会请求其下的 `/responses`；不要直接填完整 `/responses` URL。
- 必须显式设置 `providerProtocol: "openai-responses"`。不设置时，老 `custom` provider 仍请求
  `/chat/completions`，不会探测或自动切换协议。
- 第一阶段只支持 `stateMode: "stateless"` 且 `store: false`。reasoning 模型的 encrypted
  continuation 会作为不可见的版本化 transcript entry 本地保存和重放。
- `previous-response-id` 已预留配置枚举、capability 和 store 接口，但本期不会发送
  `previous_response_id`；配置启用会明确失败。

## 开启模型思考（extended thinking / reasoning）

```json
{
  "model": "glm-5.1",
  "provider": "custom",
  "effort": "medium"
}
```

- 默认使用 `high`。`low/medium/high/max` 对应 1024/2048/4096/8192 思考预算；显式设置 `off` 或 `none` 可关闭。
- Anthropic 协议映射为 `thinking` 参数；OpenAI 兼容协议映射为 `reasoning_effort`
  （low/medium/high 三档），且只在启用思考时才出现在请求体里。
- 带 `effort` 声明的 skill 优先于该设置；skill 写 `inherit` 则继承此值。
- 临时覆盖：`GOLANG_CC_EFFORT=high`（或 `CLAUDE_CODE_EFFORT`）。
- 会话 transcript 的 usage 记录含 `reasoning_output_tokens`，可用于确认思考已生效
  （OpenAI 兼容网关需在 usage 里返回 `completion_tokens_details.reasoning_tokens`）。

## 字段说明（对照示例文件）

| 字段 | 作用 |
|------|------|
| `model` | 会话主模型（模型 id）。留空时按 provider 顺序取第一个已配置模型。 |
| `provider` | provider 类型：`anthropic`（默认）/ `custom`（OpenAI 兼容（chat/completions）网关）/ `openai`（OpenAI 兼容）。 |
| `providerProtocol` | 上游 wire protocol：`anthropic-messages` / `openai-chat-completions` / `openai-responses`。缺省时按原 provider 类型推导，保持旧行为。 |
| `responses` | 仅用于 `openai-responses`；当前接受 `{stateMode: "stateless", store: false}`。 |
| `env.ANTHROPIC_BASE_URL` | Anthropic 兼容端点。官方无需填。 |
| `env.ANTHROPIC_API_KEY` | 主密钥（字面量；此处不做 `${VAR}` 展开——要用环境变量请放到 provider 的 `apiKey`）。 |
| `fallback.providers[]` | 备用 provider 列表。每项可配置 `name`/`type`/`protocol`/`baseURL`/`apiKey`/`model`/`responses`；`apiKey` 支持 `${ENV_VAR}` 展开。请求按 **model id** 路由到匹配的 provider。 |
| `subagentModelTiers` | 子代理档位别名 → 具体模型。非 Anthropic provider 下，`haiku`/`sonnet`/`opus` 会映射到这里配置的模型；未配置则继承会话模型。详见 [subagent_model_tiering_plan.md](../subagent_multiagent/subagent_model_tiering_plan.md)。 |
| `modelPricing` | 每个模型的每百万 token 单价 `{input, output}`，用于成本可观测性（批量 Task 的 `cost_usd` / `total_cost_usd`）。非 Anthropic 模型不配则成本为「未知」。可选的 `cacheRead` / `cacheWrite5m` / `cacheWrite1h` 用于 prompt cache 分档单价：**只有 Anthropic 形状的 provider（`anthropic` / `anthropic-compatible`）会按 `read 0.1× / write5m 1.25× / write1h 2×` 从 `input` 推导**，`custom` / `openai*` 各家折扣比例差异极大（DeepSeek V4 的 cache hit 约 0.02×），不配这三项则含 cache token 的轮次报「未知定价」而不是套用 Anthropic 倍率。详见 [provider_neutrality_plan.md](../provider_neutrality_plan.md)。 |
| `permissions` | 工具权限：`defaultMode`（`ask`/`acceptEdits`/`bypassPermissions` 等）、`allow`/`ask`/`deny` 规则列表。 |
| `language` | 界面/回复语言偏好，如 `zh-CN`。 |

## 多 provider 路由要点

- `subagentModelTiers` 和 `fallback.providers[].model` 里填的都是 **model id**（不是 provider 的 `name`）——客户端按 model id 匹配 provider 并覆盖请求模型。
- 例：档位 `haiku` 想走某网关的 `your-cheap-fast-model`，就把该 model id 同时出现在 `subagentModelTiers.haiku` 和某个 `fallback.providers[].model` 里，且该 provider 配好 `baseURL`/`apiKey`。

## 常用可选项（未在示例中，按需添加）

- `modelOptions`: 模型选择器候选列表。
- `outputStyle` / `contextLength`。
- `autoCompact` / `recap` / `tui` / `multimodal` / `webSearch`：各子系统配置。
- `nextSteps`：turn 结束后生成下一步 prompt 建议的子系统。**目前仅 TUI 生效**：TUI 默认开启，且每轮都会额外调用一次模型（计费）；Web Agent UI 的生成路径暂时关闭（等客户端事件交付修复后再打开），因此 web 用户既看不到建议也不会产生这笔费用。字段：`enabled`（`*bool`，留空视为开启，设为 `false` 可关闭）、`model`（显式指定模型 id，默认按档位推断）、`count`（建议条数，默认 `3`，范围 `1`-`5`）。非 Anthropic 部署若未配置 `subagentModelTiers.haiku`，本功能会**继承主模型**（可能远比预期贵），建议显式设置 `nextSteps.model`。
- `mcpServers`: MCP server 声明。
- `hooks`: 生命周期钩子。

完整字段以 `internal/config` 的 `Settings` 结构为准。
