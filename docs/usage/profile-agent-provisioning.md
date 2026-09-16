# Profile-Agent 一键向导使用说明

## 适用范围

第一版向导面向本机部署：WebUI、MySQL、全局 `settings.json`、独立 Go worker binary 和 `screen`。它不改变默认 code 模式，也不会把 Profile 变成常驻进程。Profile、Feishu Bot Account 和 worker 是三个可以独立复用的资源。

## 使用步骤

1. 打开 WebUI，进入 **Agent Profiles -> Provisioning**。
2. 在 Profile basics 填写稳定的 `profile_key`、显示名称、描述和 persona，保存草稿。
3. 在 Runtime policy 选择全局 settings 解析出的 provider/model，点击 Validate。
4. 发布已验证版本。只有 published Profile 才能被 worker/Team 使用。
5. 选择已有 Feishu account，或填写新 Bot 的 account key、App ID 和 App Secret。App Secret 只写入受保护凭据文件，页面不会回显。
6. 选择 streaming 和 reactions，运行 Preflight。token、机器人身份、能力检查会分项显示。
7. 启动 worker，在 Feishu DM 或群内 `@` Bot 发一条测试消息；回到向导点击 Refresh readback，确认 screen、PID、Inbox/Outbox 和最终状态。

## CLI 与账号选择

向导优先复用已有 `golang-cc channels onboard feishu` 和本机 `lark-cli` 能力；CLI 不存在时仍可选择已有 channel account 或由管理员手动录入凭据。向导不会自动创建第二个同账号 worker，worker 生命周期按 `(tenant_id, account_id)` 加锁。

## Provider 选择

页面只调用 `GET /v1/providers`，返回 provider 名称和默认 model。后端继续使用统一 settings loader、`config.ConfiguredProviders` 和 `Config.SelectProvider`；页面不能提交任意 endpoint、API key 或 settings 路径。

## 运行和重启

V1 supervisor 是 `screen`。重启只复用原有 payload key，不能重新生成，否则历史 Inbox/Interaction 无法解密。建议通过向导的 Restart 操作，不要手工启动第二个同账号 worker。未来可以在同一 `WorkerSupervisor` 端口下增加 systemd、Docker、Kubernetes 实现。

## 运行统计口径

向导概览中的“运行中 Worker”和“健康读回”来自实时 screen worker inventory，会扫描本机现有的 worker env、screen session 和子进程 PID。因此，先通过 CLI/脚本启动、尚未创建 provisioning session 的旧 worker 也会被统计；“向导管理 Profile”则只统计已经纳入 WebUI 生命周期的 provisioning 记录，两者是有意分开的指标。

## 常见故障

- `provider not found`：检查全局 settings 是否存在该 named provider，重新加载 provider 下拉后再 Preflight。
- `reaction scope missing`：在 Feishu 开发者后台开通 `im:message` 或 `im:message.reactions:write_only`，重新授权后再测试。
- screen 存在但状态 degraded：先查看向导中的 log path，确认子进程 PID 和 `GOLANG_CC_MYSQL_DSN`、credential file 等运行环境。
- 群聊无回复：确认 Bot 已加入群、消息里 `@` 了正确 Bot，并检查 Team binding 的 external chat id 与 trigger policy。

## 安全边界

Secret 不写入 Profile JSON、普通日志、Swagger 或前端 localStorage。所有 create/publish/start/restart/stop 操作需要 tenant owner/admin 权限，并应在审计日志中关联 trace id。
