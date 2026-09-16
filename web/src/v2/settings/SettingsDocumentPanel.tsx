import { useId, useState } from "react";
import { Activity, Braces, Check, Cpu, Eye, EyeOff, Plus, RotateCcw, Save, ShieldCheck, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { SETTINGS_SECRET_SENTINEL, settingsValue, type GlobalSettingsDraft, type SettingsPath } from "./globalSettingsDraft";
import { useGlobalSettingsText } from "./globalSettingsCopy";
import "./globalSettings.css";

type Props = { draft: GlobalSettingsDraft; view: "models" | "json" };
type FieldProps = { draft: GlobalSettingsDraft; path: SettingsPath; label: string; type?: "text" | "password" | "number"; options?: readonly (readonly [string, string])[]; placeholder?: string; full?: boolean };
const PRIMARY_PROVIDER = -1;
const PROTOCOL_RESPONSES = "openai-responses";
const PROVIDER_TYPES = [["", "继承默认"], ["anthropic", "Anthropic"], ["custom", "Custom API"], ["openai", "OpenAI"], ["openai-compatible", "OpenAI Compatible"]] as const;
const PROTOCOLS = [["", "自动选择"], ["anthropic-messages", "Anthropic Messages"], ["openai-chat-completions", "OpenAI Chat Completions"], [PROTOCOL_RESPONSES, "OpenAI Responses"]] as const;
const BOOL_OPTIONS = [["", "继承默认"], ["true", "启用"], ["false", "关闭"]] as const;
const IMAGE_ROOT = ["imageGeneration"] as const;
const FALLBACK_PROVIDERS = ["fallback", "providers"] as const;

function stringValue(value: unknown): string { return typeof value === "string" || typeof value === "number" ? String(value) : ""; }
function Field({ draft, path, label, type = "text", options, placeholder, full }: FieldProps) {
  const t = useGlobalSettingsText();
  const id = useId();
  const [visible, setVisible] = useState(false);
  const value = stringValue(settingsValue(draft.doc, path));
  const choices = options && (options.some(([key]) => key === value) ? options : [...options, [value, value] as const]);
  const change = (text: string) => draft.setField(path, type === "number" ? text === "" ? undefined : Number(text) : text);
  return <label htmlFor={id} className={`global-settings-field${full ? " full" : ""}`}>
    <span>{t(label)}</span>
    {choices ? <select id={id} value={value} onChange={(event) => draft.setField(path, event.target.value || undefined)}>{choices.map(([key, title]) => <option value={key} key={key}>{t(title)}</option>)}</select>
      : <span className="global-settings-input-row"><input id={id} aria-label={t(label)} type={type === "password" && visible ? "text" : type} value={value} placeholder={placeholder ? t(placeholder) : undefined} autoComplete={type === "password" ? "new-password" : "off"} spellCheck={false} onChange={(event) => change(event.target.value)} />
        {type === "password" && <button type="button" className="global-settings-icon" title={t(visible ? "隐藏密钥" : "显示密钥")} aria-label={t(visible ? "隐藏密钥" : "显示密钥")} onClick={(event) => { event.preventDefault(); setVisible(!visible); }}>{visible ? <EyeOff size={16} /> : <Eye size={16} />}</button>}
      </span>}
    {type === "password" && value === SETTINGS_SECRET_SENTINEL && <small>{t("已存储，未修改时保留")}</small>}
  </label>;
}

function BooleanField({ draft, path, label }: Pick<FieldProps, "draft" | "path" | "label">) {
  const t = useGlobalSettingsText();
  const value = settingsValue(draft.doc, path);
  return <label className="global-settings-field"><span>{t(label)}</span><select value={typeof value === "boolean" ? String(value) : ""} onChange={(event) => draft.setField(path, event.target.value === "" ? undefined : event.target.value === "true")}>{BOOL_OPTIONS.map(([key, title]) => <option key={key} value={key}>{t(title)}</option>)}</select></label>;
}

function StringListField({ draft, path, label }: Pick<FieldProps, "draft" | "path" | "label">) {
  const t = useGlobalSettingsText();
  const value = settingsValue(draft.doc, path);
  const lines = Array.isArray(value) ? value.map(stringValue).join("\n") : "";
  return <label className="global-settings-field full"><span>{t(label)}</span><textarea aria-label={t(label)} rows={3} spellCheck={false} value={lines}
    onChange={(event) => draft.setField(path, event.target.value === "" ? [] : event.target.value.split("\n"))}
    onBlur={() => { if (Array.isArray(value)) draft.setField(path, value.map(stringValue).map((item) => item.trim()).filter(Boolean)); }} /></label>;
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  const t = useGlobalSettingsText();
  return <section className="global-settings-section"><h3>{t(title)}</h3>{children}</section>;
}

function ProviderForm({ draft, selected, onRemove }: { draft: GlobalSettingsDraft; selected: number; onRemove: () => void }) {
  const t = useGlobalSettingsText();
  const primary = selected === PRIMARY_PROVIDER;
  const root: SettingsPath = primary ? [] : [...FALLBACK_PROVIDERS, selected];
  const protocolPath = [...root, primary ? "providerProtocol" : "protocol"];
  const isResponses = settingsValue(draft.doc, protocolPath) === PROTOCOL_RESPONSES;
  const credentials = primary ? ["env"] : root;
  return <div className="global-settings-provider-body">
    <div className="global-settings-provider-heading"><h2>{primary ? t("全局主模型") : stringValue(settingsValue(draft.doc, [...root, "name"])) || t("未命名供应商")}</h2>{primary ? <span className="global-settings-badge">{t("默认路由")}</span> : <button type="button" className="global-settings-icon danger" title={t("删除供应商")} aria-label={t("删除供应商")} onClick={onRemove}><Trash2 size={16} /></button>}</div>
    <div className="global-settings-grid">
      {!primary && <Field draft={draft} path={[...root, "name"]} label="供应商名称" />}
      <Field draft={draft} path={[...root, primary ? "provider" : "type"]} label="供应商类型" options={PROVIDER_TYPES} />
      <Field draft={draft} path={protocolPath} label="API 协议" options={PROTOCOLS} />
      <Field draft={draft} path={[...credentials, primary ? "ANTHROPIC_BASE_URL" : "baseURL"]} label="API 地址" placeholder="https://api.example.com/v1" full />
      <Field draft={draft} path={[...credentials, primary ? "ANTHROPIC_API_KEY" : "apiKey"]} label="API Key" type="password" full />
      <Field draft={draft} path={[...credentials, primary ? "ANTHROPIC_AUTH_TOKEN" : "authToken"]} label="Auth Token" type="password" full />
      <Field draft={draft} path={[...root, "model"]} label="默认模型" placeholder="模型标识" />
      {primary && <Field draft={draft} path={["effort"]} label="思考强度" options={[["", "继承默认"], ["off", "关闭"], ["low", "Low"], ["medium", "Medium"], ["high", "High"], ["max", "Max"]]} />}
    </div>
    {(isResponses || settingsValue(draft.doc, [...root, "responses"]) !== undefined) && <Section title="Responses"><div className="global-settings-grid">
      <Field draft={draft} path={[...root, "responses", "stateMode"]} label="状态模式" options={[["", "继承默认"], ["stateless", "Stateless"]]} />
      <BooleanField draft={draft} path={[...root, "responses", "store"]} label="服务端存储" />
    </div>{!isResponses && <button type="button" onClick={() => draft.setField([...root, "responses"], undefined)}><Trash2 size={14} />{t("移除 Responses 配置")}</button>}</Section>}
    {!primary && <details className="global-settings-advanced"><summary>{t("图片接口配置")}</summary><div className="global-settings-grid"><Field draft={draft} path={[...root, "imageProtocol"]} label="图片 API 协议" placeholder="继承供应商默认" /><StringListField draft={draft} path={[...root, "imageResultHosts"]} label="图片结果域名白名单" /></div></details>}
    {primary && <details className="global-settings-advanced"><summary>{t("环境路由与备用凭据")}</summary><div className="global-settings-grid"><Field draft={draft} path={["env", "GOLANG_CC_PROVIDER"]} label="环境供应商覆盖" /><Field draft={draft} path={["env", "CLAUDE_CODE_PROVIDER"]} label="兼容供应商覆盖" /><Field draft={draft} path={["env", "CLAUDE_CODE_AUTH_TOKEN"]} label="兼容 Auth Token" type="password" /><Field draft={draft} path={["env", "CLAUDE_CODE_OAUTH_TOKEN"]} label="OAuth Token" type="password" /></div></details>}
    <ProviderConnection draft={draft} provider={primary ? undefined : stringValue(settingsValue(draft.doc, [...root, "name"]))} />
  </div>;
}

function ProviderConnection({ draft, provider }: { draft: GlobalSettingsDraft; provider?: string }) {
  const t = useGlobalSettingsText();
  const unnamed = provider !== undefined && !provider.trim();
  const connection = draft.connection?.provider === provider ? draft.connection : null;
  return <div className="global-settings-connection"><span role="status">{unnamed ? t("请先填写供应商名称") : connection ? `${t(connection.ok ? "模型目录连接成功" : "模型目录连接失败")}: ${connection.message}` : t("尚未测试模型目录连接")}</span><button type="button" disabled={unnamed || draft.busy || !draft.doc || Boolean(draft.syntaxError)} onClick={() => void draft.testProvider(provider)}><Activity size={15} />{t(draft.operation === "test" ? "测试中…" : "测试连接")}</button></div>;
}

function ModelsView({ draft }: { draft: GlobalSettingsDraft }) {
  const t = useGlobalSettingsText();
  const [selected, setSelected] = useState(PRIMARY_PROVIDER);
  const providersValue = settingsValue(draft.doc, FALLBACK_PROVIDERS);
  const providers = Array.isArray(providersValue) ? providersValue : [];
  const selection = selected >= providers.length ? PRIMARY_PROVIDER : selected;
  const addProvider = () => {
    const names = new Set(providers.map((provider) => stringValue(settingsValue(provider, ["name"]))));
    let suffix = providers.length + 1;
    while (names.has(`provider-${suffix}`)) suffix++;
    draft.setField(FALLBACK_PROVIDERS, [...providers, { name: `provider-${suffix}`, type: "custom", protocol: "openai-chat-completions" }]);
    setSelected(providers.length);
  };
  const removeProvider = () => {
    if (!window.confirm(t("删除这个供应商？引用它的图片或路由配置也需要调整。"))) return;
    draft.setField(FALLBACK_PROVIDERS, providers.filter((_, index) => index !== selection));
    setSelected(PRIMARY_PROVIDER);
  };
  const namedProviders = providers.map((provider) => stringValue(settingsValue(provider, ["name"]))).filter(Boolean);
  const imageProviders: Array<readonly [string, string]> = [["", "继承默认"], ...namedProviders.map((name): [string, string] => [name, name])];
  return <fieldset className="global-settings-form" disabled={draft.busy || !draft.doc || Boolean(draft.syntaxError)}>
    <div className="global-settings-provider-layout">
      <nav className="global-settings-provider-list" aria-label={t("供应商列表")}>
        <button type="button" className={selection === PRIMARY_PROVIDER ? "selected" : ""} onClick={() => setSelected(PRIMARY_PROVIDER)}><Cpu size={18} /><span><strong>{t("全局主模型")}</strong><small>{stringValue(settingsValue(draft.doc, ["model"])) || t("继承默认模型")}</small></span></button>
        {providers.map((provider, index) =>
          // biome-ignore lint/suspicious/noArrayIndexKey: Draft rows may have duplicate names; positional keys never contain credentials.
          <button type="button" key={index} className={selection === index ? "selected" : ""} onClick={() => setSelected(index)}><Cpu size={18} /><span><strong>{stringValue(settingsValue(provider, ["name"])) || `${t("供应商")} ${index + 1}`}</strong><small>{stringValue(settingsValue(provider, ["model"])) || stringValue(settingsValue(provider, ["type"]))}</small></span></button>)}
        <button type="button" onClick={addProvider}><Plus size={16} />{t("添加供应商")}</button>
      </nav>
      <ProviderForm key={selection} draft={draft} selected={selection} onRemove={removeProvider} />
    </div>
    <Section title="备用路由"><div className="global-settings-grid"><BooleanField draft={draft} path={["fallback", "enabled"]} label="启用 Fallback" /></div></Section>
    <Section title="图片生成"><div className="global-settings-grid">
      <BooleanField draft={draft} path={[...IMAGE_ROOT, "enabled"]} label="启用图片生成" />
      <BooleanField draft={draft} path={[...IMAGE_ROOT, "previewInContext"]} label="模型上下文包含图片预览" />
      <Field draft={draft} path={[...IMAGE_ROOT, "defaultProvider"]} label="默认图片供应商" options={imageProviders} />
      <Field draft={draft} path={[...IMAGE_ROOT, "defaultModel"]} label="默认图片模型" />
      <Field draft={draft} path={[...IMAGE_ROOT, "size"]} label="默认尺寸" placeholder="auto" />
      <Field draft={draft} path={[...IMAGE_ROOT, "quality"]} label="图片质量" options={[["", "继承默认"], ["auto", "Auto"], ["low", "Low"], ["medium", "Medium"], ["high", "High"]]} />
    </div><details className="global-settings-advanced"><summary>{t("兼容配置与生成限制")}</summary><div className="global-settings-grid">
      <Field draft={draft} path={[...IMAGE_ROOT, "provider"]} label="兼容图片供应商" options={imageProviders} />
      <Field draft={draft} path={[...IMAGE_ROOT, "model"]} label="兼容图片模型" />
      <Field draft={draft} path={[...IMAGE_ROOT, "outputFormat"]} label="输出格式" options={[["", "继承默认"], ["png", "PNG"], ["jpeg", "JPEG"], ["webp", "WebP"]]} />
      <Field draft={draft} path={[...IMAGE_ROOT, "background"]} label="图片背景" options={[["", "继承默认"], ["auto", "Auto"], ["transparent", "透明"], ["opaque", "不透明"]]} />
      {([["maxImages", "最大图片数量"], ["timeoutSeconds", "请求超时（秒）"], ["maxConcurrent", "最大并发"], ["retentionDays", "保留天数"], ["maxPromptChars", "提示词字符上限"], ["maxInputBytes", "输入大小上限（字节）"]] as const).map(([key, label]) => <Field key={key} draft={draft} path={[...IMAGE_ROOT, key]} label={label} type="number" />)}
      <BooleanField draft={draft} path={[...IMAGE_ROOT, "asyncChannelEnabled"]} label="渠道异步图片生成" />
      <StringListField draft={draft} path={[...IMAGE_ROOT, "asyncChannelAccountKeys"]} label="异步生成渠道账号范围" />
    </div></details><details className="global-settings-advanced"><summary>{t("图片任务 Worker")}</summary><div className="global-settings-grid">
      {([["pollIntervalMs", "轮询间隔（毫秒）"], ["maxConcurrent", "Worker 最大并发"], ["maxAttempts", "最大尝试次数"], ["heartbeatSeconds", "心跳间隔（秒）"], ["leaseSeconds", "租约时长（秒）"], ["finalizeTimeoutSeconds", "完成阶段超时（秒）"], ["maxQueuedPerTenant", "每租户排队上限"], ["maxQueuedPerUser", "每用户排队上限"]] as const).map(([key, label]) => <Field key={key} draft={draft} path={[...IMAGE_ROOT, "worker", key]} label={label} type="number" />)}
    </div></details></Section>
  </fieldset>;
}

export function SettingsDocumentPanel({ draft, view }: Props) {
  const t = useGlobalSettingsText();
  const reset = () => { if (window.confirm(t("放弃未保存的修改，恢复到上次读取的配置？"))) draft.reset(); };
  const reload = () => { if (!draft.dirty || window.confirm(t("重新载入会放弃当前草稿，确定继续？"))) void draft.reload(); };
  return <div className="global-settings-panel">
    {draft.loading && <p role="status">{t("正在读取全局设置…")}</p>}
    {draft.error && <div className="global-settings-error" role="alert"><span>{t(draft.error)}</span><button type="button" disabled={draft.busy} onClick={reload}><RotateCcw size={15} />{t(draft.conflict ? "重新载入" : "重试读取")}</button></div>}
    {draft.loaded && <>
      {draft.syntaxError && <p className="global-settings-error" role="alert">{t(draft.syntaxError)}{view === "models" ? ` ${t("请在全局 Settings JSON 中修正后继续。")}` : ""}</p>}
      {view === "models" ? <ModelsView draft={draft} /> : <div className="global-settings-json"><div className="global-settings-json-toolbar"><span><Braces size={16} />settings.json</span><button type="button" className="global-settings-icon" title={t("格式化 JSON")} aria-label={t("格式化 JSON")} disabled={draft.busy || !draft.doc} onClick={() => draft.doc && draft.setRaw(JSON.stringify(draft.doc, null, 2))}><Braces size={16} /></button></div><textarea aria-label={t("全局 Settings JSON")} spellCheck={false} autoCapitalize="off" autoComplete="off" value={draft.raw} disabled={draft.busy} onChange={(event) => draft.setRaw(event.target.value)} /><div className="global-settings-json-meta"><span>{draft.raw.split("\n").length} {t("行")}</span><span>JSON · UTF-8</span></div></div>}
      {draft.validation && <div className={draft.validation.valid ? "global-settings-success" : "global-settings-error"} role="status">{draft.validation.valid ? <><Check size={16} />{t("配置校验通过")}</> : <ul>{draft.validation.issues.map((issue) => <li key={`${issue.field}-${issue.code}-${issue.message}`}><code>{issue.field || "settings"}</code>: {issue.message}</li>)}</ul>}</div>}
      {draft.saved && <p className="global-settings-success" role="status"><Check size={16} />{t("已保存并读取确认。运行中的服务需重启以加载新配置。")}</p>}
      <footer className="global-settings-footer"><span>{t(draft.dirty ? "有未保存的修改" : "与上次读取的配置一致")}</span><div><button type="button" title={t("重置草稿")} disabled={!draft.dirty || draft.busy} onClick={reset}><RotateCcw size={15} />{t("重置")}</button><button type="button" disabled={draft.busy || !draft.doc} onClick={() => void draft.validate()}><ShieldCheck size={15} />{t(draft.operation === "validate" ? "校验中…" : "校验")}</button><button type="button" className="primary" disabled={!draft.dirty || draft.busy || !draft.doc || draft.conflict} onClick={() => void draft.save()}><Save size={15} />{t(draft.operation === "save" ? "保存中…" : "保存更改")}</button></div></footer>
    </>}
  </div>;
}
