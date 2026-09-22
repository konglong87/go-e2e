import { ArrowLeft, ArrowRight, Bot, Check, CheckCircle2, KeyRound, Play, RefreshCw, Save, ShieldCheck, Square, WandSparkles } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { createProvisioning, getProvisioningOverview, listAgentProfiles, listChannelAccounts, listProviders, preflightProvisioning, publishAgentProfile, saveAgentProfile, validateAgentProfile, workerProvisioningAction } from "../../lib/api";
import { useI18n, type Language } from "../../lib/i18n";
import type { AgentProfileDocument, AgentProfileRecord, ChannelAccountRecord, IdentityConfig, ProviderOption, ProvisioningRecord, ProvisioningWorkerStatus } from "../../lib/types";
import { ProvisioningOverview } from "./ProvisioningOverview";
import { ProvisioningStatusPanel } from "./ProvisioningStatusPanel";
import { ProvisioningSteps } from "./ProvisioningSteps";

const labels: Record<Language, string[]> = {
  en: ["Profile basics", "Runtime policy", "Validate & publish", "Feishu account", "Run strategy", "Worker lifecycle", "Test & activate"],
  zh: ["Profile 基础", "运行策略", "校验与发布", "飞书账号", "运行策略", "Worker 生命周期", "测试与启用"]
};
const defaultConfig: AgentProfileDocument = {
  schema_version: 1,
  identity: { display_name: "", description: "" },
  prompt: { mode: "chat", persona: "", system_addendum: "" },
  capabilities: { tools: { allow: [], deny: [] }, skills: [], mcp_servers: [], allow_agents: false, allow_attachments: true },
  execution: { max_turns: 12, max_tokens: 4000, effort: "medium", provider: "", model: "" },
  context: { workspace: true, git: true, tenant_memory: true, user_memory: true, knowledge_base: true, session_history: true },
  safety: { permission_mode: "ask", sandbox: "workspace", allow_unsandboxed_commands: false }
};

export function ProvisioningWizard({ identity, onStatus }: { identity: IdentityConfig; onStatus: (message: string) => void }) {
  const { language } = useI18n();
  const tr = (en: string, zh: string) => language === "zh" ? zh : en;
  const stepLabels = labels[language];
  const [records, setRecords] = useState<ProvisioningRecord[]>([]);
  const [workers, setWorkers] = useState<ProvisioningWorkerStatus[]>([]);
  const [profiles, setProfiles] = useState<AgentProfileRecord[]>([]);
  const [accounts, setAccounts] = useState<ChannelAccountRecord[]>([]);
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [selected, setSelected] = useState<ProvisioningRecord | null>(null);
  const [step, setStep] = useState(0);
  const [profileKey, setProfileKey] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [persona, setPersona] = useState("");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [accountKey, setAccountKey] = useState("");
  const [appId, setAppId] = useState("");
  const [appSecret, setAppSecret] = useState("");
  const [streaming, setStreaming] = useState("on");
  const [reactions, setReactions] = useState("on");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const selectedProfile = useMemo(() => profiles.find((item) => item.profile_key === profileKey), [profiles, profileKey]);

  useEffect(() => {
    let cancelled = false;
    Promise.all([getProvisioningOverview(identity).catch(() => ({records: [], workers: []})), listAgentProfiles(identity).catch(() => []), listChannelAccounts(identity).catch(() => []), listProviders(identity).catch(() => [])])
      .then(([overview, nextProfiles, nextAccounts, nextProviders]) => {
        if (cancelled) return;
        setRecords(overview.records); setWorkers(overview.workers); setProfiles(nextProfiles); setAccounts(nextAccounts); setProviders(nextProviders);
        const first = overview.records[0];
        if (first) {
          setSelected(first);
          setProfileKey(first.profile_key);
          setAccountKey(first.account_key);
          setProvider(first.worker?.provider || "");
          setModel(first.worker?.model || "");
          // A discovered/running worker already passed profile creation,
          // validation, and publishing. Land on step 04 so an existing
          // Feishu account can be inspected and operated immediately.
          setStep(first.observed_worker?.state ? 3 : 0);
        }
      })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : String(err)); });
    return () => { cancelled = true; };
  }, [identity]);

  async function run(action: () => Promise<void>) { setBusy(true); setError(""); try { await action(); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } finally { setBusy(false); } }
  function profileConfig(): AgentProfileDocument { return { ...defaultConfig, identity: { display_name: displayName, description }, prompt: { ...defaultConfig.prompt, persona } }; }
  async function saveDraft() { await run(async () => { const item = await saveAgentProfile(identity, { profile_key: profileKey.trim(), scope: "tenant_shared", display_name: displayName.trim(), description, config: profileConfig() }); setProfiles((current) => [item, ...current.filter((profile) => profile.profile_key !== item.profile_key)]); onStatus(tr(`Saved ${item.display_name}`, `已保存 ${item.display_name}`)); setStep(1); }); }
  async function validate() { await run(async () => { const result = await validateAgentProfile(identity, { profile_key: profileKey, display_name: displayName, config: profileConfig() }, profileKey); if (!result.valid) throw new Error(result.issues?.map((issue) => issue.message).join("; ") || tr("Profile validation failed", "Profile 校验失败")); onStatus(tr("Profile validated", "Profile 校验通过")); setStep(2); }); }
  async function publish() { await run(async () => { const profile = selectedProfile || (await listAgentProfiles(identity)).find((item) => item.profile_key === profileKey); if (!profile) throw new Error(tr("Save the profile first", "请先保存 Profile")); await publishAgentProfile(identity, profile.profile_key, profile.profile_version); onStatus(tr("Profile published", "Profile 已发布")); setStep(3); }); }
  async function createWorker() { await run(async () => { if (!profileKey || !accountKey) throw new Error(tr("Profile key and account key are required", "Profile Key 和账号 Key 不能为空")); const item = await createProvisioning(identity, { profile_key: profileKey, account_key: accountKey, credential: { id: accountKey, provider: "feishu", app_id: appId, secret_value: appSecret }, worker: { supervisor: "screen", account_key: accountKey, provider, model, settings_ref: "global-settings", streaming, reactions, permission_mode: "ask" } }); setSelected(item); setRecords((current) => [item, ...current.filter((record) => record.id !== item.id)]); onStatus(tr("Provisioning draft created", "Worker 草稿已创建")); setStep(4); }); }
  async function action(kind: "preflight" | "start" | "restart" | "stop" | "status") { if (!selected) { setError(tr("Create the provisioning draft first", "请先创建 Worker 草稿")); return; } await run(async () => { const item = kind === "preflight" ? await preflightProvisioning(identity, selected.id) : await workerProvisioningAction(identity, selected.id, kind); setSelected(item); setRecords((current) => current.map((record) => record.id === item.id ? item : record)); onStatus(`${kind} ${tr("completed", "完成")}`); }); }

  return <div className="provisioning-page">
    <div className="provisioning-hero"><div><span className="eyebrow">{tr("Assignment / profile operations", "Assignment / Profile 运维")}</span><h1>{tr("Provision an agent with confidence.", "可靠地创建一个智能体。")}</h1><p>{tr("One place to shape a profile, connect its Feishu identity, and keep the local worker observable.", "在一个工作台完成 Profile 配置、飞书身份连接和本机 Worker 观测。")}</p></div><div className="provisioning-hero-mark"><WandSparkles size={30} /><span>V1 · screen</span></div></div>
    <ProvisioningOverview records={records} workers={workers} language={language} />
    <div className="provisioning-layout"><aside className="provisioning-rail"><ProvisioningSteps current={step} labels={stepLabels} language={language} onSelect={(index) => { if (!busy && index <= step) setStep(index); }} /><div className="provisioning-rail-note"><ShieldCheck size={16} /><span>{tr("Secrets stay write-only and worker actions are tenant-scoped.", "密钥仅写入，不会回显；Worker 操作受租户权限约束。")}</span></div></aside>
      <section className="provisioning-main"><ProvisioningStatusPanel record={selected} language={language} />{error ? <div className="provisioning-alert" role="alert">{error}</div> : null}<div className="provisioning-form-head"><div><span className="eyebrow">{tr("Step", "步骤")} 0{step + 1}</span><h2>{stepLabels[step]}</h2></div><span className="provisioning-draft-chip">{busy ? tr("Working...", "处理中...") : selected ? tr("Draft synced", "草稿已同步") : tr("New draft", "新建草稿")}</span></div>
        {step === 0 ? <div className="provisioning-form"><label>{tr("Profile key", "Profile Key")}<input value={profileKey} onChange={(event) => setProfileKey(event.target.value)} placeholder="copywriter" /></label><label>{tr("Display name", "显示名称")}<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="文案智能体" /></label><label className="full">{tr("Description", "描述")}<textarea value={description} onChange={(event) => setDescription(event.target.value)} placeholder={tr("Brand copy and social content assistant.", "品牌文案、社媒内容和改写助手。" )} rows={3} /></label><label className="full">Persona<textarea value={persona} onChange={(event) => setPersona(event.target.value)} placeholder={tr("Professional, concise, and decisive.", "专业、简洁、有判断力。")} rows={4} /></label></div> : null}
        {step === 1 ? <div className="provisioning-form"><label>{tr("Provider", "Provider")}<select value={provider} onChange={(event) => { setProvider(event.target.value); setModel(providers.find((item) => item.name === event.target.value)?.model || ""); }}><option value="">{tr("Select from global settings", "从全局 settings 选择")}</option>{providers.map((item) => <option value={item.name} key={item.name}>{item.name} · {item.model}</option>)}</select></label><label>{tr("Model", "模型")}<input value={model} onChange={(event) => setModel(event.target.value)} /></label><label className="full">{tr("Runtime intent", "运行意图")}<textarea value={persona} onChange={(event) => setPersona(event.target.value)} rows={5} /></label></div> : null}
        {step === 2 ? <div className="provisioning-review"><CheckCircle2 size={21} /><div><strong>{tr("Ready to publish", "可以发布")}</strong><p>{tr("The published version is the only version a worker can run.", "只有已发布版本才能被 Worker 使用。")}</p></div></div> : null}
        {step === 3 ? <div className="provisioning-form"><div className="provisioning-choice-grid"><button className={accountKey ? "provisioning-choice active" : "provisioning-choice"} onClick={() => setAccountKey(accounts[0]?.account_key || "existing-feishu")} type="button"><Bot size={19} /><span>{tr("Use existing account", "使用已有账号")}</span><small>{accounts[0]?.account_key || tr("No safe account metadata found", "没有可用的安全账号元数据")}</small></button><button className={!accountKey ? "provisioning-choice active" : "provisioning-choice"} onClick={() => setAccountKey(`${profileKey}-feishu`)} type="button"><KeyRound size={19} /><span>{tr("Connect a new bot", "连接新机器人")}</span><small>{tr("CLI or manual credentials", "CLI 或手动填写凭据")}</small></button></div><label>{tr("Account key", "账号 Key")}<input value={accountKey} onChange={(event) => setAccountKey(event.target.value)} /></label><label>App ID<input value={appId} onChange={(event) => setAppId(event.target.value)} placeholder="cli_..." /></label><label className="full">App Secret<span className="field-hint">{tr("Write-only. It is never returned after save.", "仅写入，保存后不会回显。")}</span><input type="password" value={appSecret} onChange={(event) => setAppSecret(event.target.value)} autoComplete="new-password" /></label></div> : null}
        {step === 4 ? <div className="provisioning-form"><label>Provider<select value={provider} onChange={(event) => setProvider(event.target.value)}>{providers.map((item) => <option value={item.name} key={item.name}>{item.name}</option>)}</select></label><label>{tr("Model", "模型")}<input value={model} onChange={(event) => setModel(event.target.value)} /></label><label>{tr("Streaming", "流式输出")}<select value={streaming} onChange={(event) => setStreaming(event.target.value)}><option value="on">{tr("On · Feishu card updates", "开启 · 飞书卡片实时更新")}</option><option value="off">{tr("Off", "关闭")}</option></select></label><label>{tr("Reactions", "Reaction 状态")}<select value={reactions} onChange={(event) => setReactions(event.target.value)}><option value="on">{tr("On · typing and done", "开启 · 处理中与完成")}</option><option value="off">{tr("Off", "关闭")}</option></select></label></div> : null}
        {step === 5 ? <div className="provisioning-actions"><button className="primary-button" disabled={busy || !selected} onClick={() => void action("start")} type="button"><Play size={16} /> {tr("Start worker", "启动 Worker")}</button><button className="secondary-button" disabled={busy || !selected} onClick={() => void action("restart")} type="button"><RefreshCw size={16} /> {tr("Restart", "重启")}</button><button className="secondary-button danger" disabled={busy || !selected} onClick={() => void action("stop")} type="button"><Square size={15} /> {tr("Stop", "停止")}</button></div> : null}
        {step === 6 ? <div className="provisioning-review"><CheckCircle2 size={21} /><div><strong>{tr("Run the final smoke check in Feishu", "在飞书执行最终冒烟测试")}</strong><p>{tr("Mention the bot in a group or send a DM, then refresh the observed state.", "在群里 @ 机器人或发送一条私聊消息，然后刷新状态读回。")}</p></div></div> : null}
        <div className="provisioning-form-footer"><button className="secondary-button" disabled={step === 0 || busy} onClick={() => setStep((current) => current - 1)} type="button"><ArrowLeft size={15} /> {tr("Back", "返回")}</button><div className="button-row">{step === 0 ? <button className="primary-button" disabled={busy || !profileKey || !displayName} onClick={() => void saveDraft()} type="button"><Save size={15} /> {tr("Save draft", "保存草稿")}</button> : null}{step === 1 ? <button className="primary-button" disabled={busy || !provider} onClick={() => void validate()} type="button"><ShieldCheck size={15} /> {tr("Validate", "校验")}</button> : null}{step === 2 ? <button className="primary-button" disabled={busy} onClick={() => void publish()} type="button"><Check size={15} /> {tr("Publish", "发布")}</button> : null}{step === 3 ? <button className="primary-button" disabled={busy || !accountKey} onClick={() => void createWorker()} type="button"><Bot size={15} /> {tr("Create worker draft", "创建 Worker 草稿")}</button> : null}{step === 4 ? <button className="primary-button" disabled={busy || !selected} onClick={() => void action("preflight")} type="button"><ShieldCheck size={15} /> {tr("Run preflight", "运行预检")}</button> : null}{step === 5 ? <button className="primary-button" disabled={busy || !selected} onClick={() => setStep(6)} type="button">{tr("Continue", "继续")} <ArrowRight size={15} /></button> : null}{step === 6 ? <button className="primary-button" disabled={busy || !selected} onClick={() => void action("status")} type="button"><RefreshCw size={15} /> {tr("Refresh readback", "刷新状态")}</button> : null}</div></div>
      </section></div>
  </div>;
}
