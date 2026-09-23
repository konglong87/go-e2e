import { ArrowLeft, ArrowRight, Bot, CheckCircle2, KeyRound, Play, RefreshCw, ShieldCheck, Square, WandSparkles } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { createProvisioning, getProvisioningOverview, listAgentProfiles, listChannelAccounts, listProviders, preflightProvisioning, workerProvisioningAction } from "../../lib/api";
import { useI18n, type Language } from "../../lib/i18n";
import type { AgentProfileRecord, ChannelAccountRecord, IdentityConfig, ProviderOption, ProvisioningRecord, ProvisioningWorkerStatus } from "../../lib/types";
import { ProvisioningOverview } from "./ProvisioningOverview";
import { ProvisioningStatusPanel } from "./ProvisioningStatusPanel";
import { ProvisioningSteps } from "./ProvisioningSteps";

export type ProvisioningView = "account" | "lifecycle";

const labels: Record<ProvisioningView, Record<Language, string[]>> = {
  account: {
    en: ["Select agent", "Connect Feishu", "Connection draft"],
    zh: ["选择智能体", "连接飞书", "连接草稿"]
  },
  lifecycle: {
    en: ["Select Worker", "Preflight settings", "Worker lifecycle", "Test and refresh"],
    zh: ["选择 Worker", "运行预检", "Worker 生命周期", "测试与刷新"]
  }
};

type Props = {
  identity: IdentityConfig;
  onStatus: (message: string) => void;
  view?: ProvisioningView;
};

export function ProvisioningWizard({ identity, onStatus, view = "account" }: Props) {
  const { language } = useI18n();
  const tr = (en: string, zh: string) => language === "zh" ? zh : en;
  const stepLabels = labels[view][language];
  const [records, setRecords] = useState<ProvisioningRecord[]>([]);
  const [workers, setWorkers] = useState<ProvisioningWorkerStatus[]>([]);
  const [profiles, setProfiles] = useState<AgentProfileRecord[]>([]);
  const [accounts, setAccounts] = useState<ChannelAccountRecord[]>([]);
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [selected, setSelected] = useState<ProvisioningRecord | null>(null);
  const [step, setStep] = useState(0);
  const [profileKey, setProfileKey] = useState("");
  const [accountKey, setAccountKey] = useState("");
  const [appId, setAppId] = useState("");
  const [appSecret, setAppSecret] = useState("");
  const [usingExistingAccount, setUsingExistingAccount] = useState(true);
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [streaming, setStreaming] = useState("on");
  const [reactions, setReactions] = useState("on");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const publishedProfiles = useMemo(() => profiles.filter((item) => item.status === "published"), [profiles]);
  const selectedProfile = useMemo(() => profiles.find((item) => item.profile_key === profileKey), [profiles, profileKey]);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      getProvisioningOverview(identity).catch(() => ({ records: [], workers: [] })),
      listAgentProfiles(identity).catch(() => []),
      listChannelAccounts(identity).catch(() => []),
      listProviders(identity).catch(() => [])
    ]).then(([overview, nextProfiles, nextAccounts, nextProviders]) => {
      if (cancelled) return;
      setRecords(overview.records);
      setWorkers(overview.workers);
      setProfiles(nextProfiles);
      setAccounts(nextAccounts);
      setProviders(nextProviders);
      const first = overview.records[0];
      if (!first) return;
      setSelected(first);
      setProfileKey(first.profile_key);
      setAccountKey(first.account_key);
      setUsingExistingAccount(true);
      setProvider(first.worker?.provider || "");
      setModel(first.worker?.model || "");
      setStep(view === "account" ? 1 : 0);
    }).catch((reason) => {
      if (!cancelled) setError(reason instanceof Error ? reason.message : String(reason));
    });
    return () => { cancelled = true; };
  }, [identity.apiBase, identity.apiToken, identity.tenantKey, identity.userId, view]);

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  }

  async function createConnectionDraft() {
    await run(async () => {
      if (!profileKey || !accountKey) throw new Error(tr("Select a published agent and enter an account key.", "请选择已发布智能体并填写账号 Key。"));
      const item = await createProvisioning(identity, {
        profile_key: profileKey,
        account_key: accountKey,
        credential: { id: accountKey, provider: "feishu", app_id: appId, secret_value: appSecret },
        worker: { supervisor: "screen", account_key: accountKey, provider, model, settings_ref: "global-settings", streaming, reactions, permission_mode: "ask" }
      });
      setSelected(item);
      setRecords((current) => [item, ...current.filter((record) => record.id !== item.id)]);
      onStatus(tr("Feishu connection draft saved; Worker is not started.", "飞书连接草稿已保存；Worker 尚未启动。"));
      setStep(2);
    });
  }

  async function action(kind: "preflight" | "start" | "restart" | "stop" | "status") {
    if (!selected) {
      setError(tr("Select a Worker first.", "请先选择 Worker。"));
      return;
    }
    await run(async () => {
      const item = kind === "preflight"
        ? await preflightProvisioning(identity, selected.id)
        : await workerProvisioningAction(identity, selected.id, kind);
      setSelected(item);
      setRecords((current) => current.map((record) => record.id === item.id ? item : record));
      onStatus(`${kind} ${tr("completed", "完成")}`);
    });
  }

  function selectRecord(record: ProvisioningRecord) {
    if (busy) return;
    setSelected(record);
    setProfileKey(record.profile_key);
    setAccountKey(record.account_key);
    setProvider(record.worker?.provider || "");
    setModel(record.worker?.model || "");
  }

  const title = view === "account" ? tr("Feishu connection", "飞书连接") : tr("Worker runtime", "Worker 运行");
  const description = view === "account"
    ? tr("Connect a Feishu bot to a published agent definition. This creates a Worker draft, but does not start the Worker.", "把飞书机器人连接到已发布的智能体定义。这里会创建 Worker 草稿，但不会启动 Worker。")
    : tr("Operate an existing channel Worker. Profile editing and Feishu credentials stay in their own settings pages.", "管理已有渠道 Worker。Profile 编辑和飞书凭据分别在各自设置页面完成。");

  return <div className="provisioning-page">
    <div className="provisioning-hero">
      <div><span className="eyebrow">{view === "account" ? tr("Account and binding", "账号与绑定") : tr("Lifecycle operations", "生命周期运维")}</span><h1>{title}</h1><p>{description}</p></div>
      <div className="provisioning-hero-mark"><WandSparkles size={30} /><span>{view === "account" ? "Feishu" : "Worker"}</span></div>
    </div>
    {view === "lifecycle" ? <ProvisioningOverview records={records} workers={workers} language={language} /> : null}
    <div className="provisioning-scope-note"><strong>{tr("What belongs here", "这里负责什么")}</strong><span>{view === "account" ? tr("Choose a published agent, connect a bot account, and save a connection draft. Start and stop controls are on Worker runtime.", "选择已发布智能体、连接机器人账号并保存连接草稿。启动和停止请前往“Worker 运行”。") : tr("Choose a saved Worker, run preflight, then start, restart, stop or refresh its observed state. Account credentials are not edited here.", "选择已保存的 Worker，执行预检，然后启动、重启、停止或刷新状态。账号凭据不在这里编辑。")}</span></div>
    <div className="provisioning-layout">
      <aside className="provisioning-rail">
        <ProvisioningSteps current={step} labels={stepLabels} language={language} onSelect={(index) => { if (!busy && index <= step) setStep(index); }} />
        <div className="provisioning-rail-note"><ShieldCheck size={16} /><span>{tr("Secrets stay write-only and worker actions are tenant-scoped.", "密钥仅写入，不会回显；Worker 操作受租户权限约束。")}</span></div>
      </aside>
      <section className="provisioning-main">
        {view === "lifecycle" ? <ProvisioningStatusPanel record={selected} language={language} /> : selected ? <ProvisioningStatusPanel record={selected} language={language} /> : null}
        {error ? <div className="provisioning-alert" role="alert">{error}</div> : null}
        <div className="provisioning-form-head"><div><span className="eyebrow">{tr("Step", "步骤")} 0{step + 1}</span><h2>{stepLabels[step]}</h2></div><span className="provisioning-draft-chip">{busy ? tr("Working...", "处理中...") : selected ? tr("Synced", "已同步") : tr("No selection", "未选择")}</span></div>

        {view === "account" && step === 0 ? <div className="provisioning-form">
          <label className="full">{tr("Published agent definition", "已发布智能体定义")}<select value={profileKey} onChange={(event) => setProfileKey(event.target.value)}><option value="">{tr("Select a published agent", "选择已发布智能体")}</option>{publishedProfiles.map((profile) => <option value={profile.profile_key} key={`${profile.profile_key}-${profile.profile_version}`}>{profile.display_name} · v{profile.profile_version}</option>)}</select></label>
          <div className="provisioning-review full"><CheckCircle2 size={21} /><div><strong>{tr("Profile editing lives in Agent definitions", "Profile 编辑请前往“智能体定义”")}</strong><p>{tr("This page only chooses which published definition the Feishu bot will use.", "本页面只选择飞书机器人使用哪个已发布定义。")}</p></div></div>
        </div> : null}

        {view === "account" && step === 1 ? <div className="provisioning-form">
          <div className="provisioning-choice-grid"><button className={usingExistingAccount ? "provisioning-choice active" : "provisioning-choice"} onClick={() => { setUsingExistingAccount(true); setAccountKey(accounts[0]?.account_key || accountKey); }} type="button"><Bot size={19} /><span>{tr("Use existing account", "使用已有账号")}</span><small>{accounts[0]?.account_key || tr("No safe account metadata found", "没有可用的安全账号元数据")}</small></button><button className={!usingExistingAccount ? "provisioning-choice active" : "provisioning-choice"} onClick={() => { setUsingExistingAccount(false); setAccountKey(accountKey || `${profileKey}-feishu`); }} type="button"><KeyRound size={19} /><span>{tr("Connect a new bot", "连接新机器人")}</span><small>{tr("The secret is write-only.", "密钥只写入，不会回显。")}</small></button></div>
          {usingExistingAccount ? <label className="full">{tr("Existing account", "已有账号")}<select value={accountKey} onChange={(event) => setAccountKey(event.target.value)}><option value="">{tr("Select an account", "选择账号")}</option>{accounts.map((account) => <option value={account.account_key} key={account.id}>{account.account_key} · {account.provider}</option>)}</select></label> : <><label>{tr("Account key", "账号 Key")}<input value={accountKey} onChange={(event) => setAccountKey(event.target.value)} /></label><label>{tr("App ID", "App ID")}<input value={appId} onChange={(event) => setAppId(event.target.value)} placeholder="cli_..." /></label><label className="full">{tr("App Secret", "App Secret")}<span className="field-hint">{tr("Write-only. It is never returned after save.", "仅写入，不会回显。")}</span><input type="password" value={appSecret} onChange={(event) => setAppSecret(event.target.value)} autoComplete="new-password" /></label></>}
        </div> : null}

        {view === "account" && step === 2 ? <div className="provisioning-review"><CheckCircle2 size={21} /><div><strong>{tr("Connection draft saved", "连接草稿已保存")}</strong><p>{tr(`${selectedProfile?.display_name || profileKey} is connected to ${accountKey}. Go to Worker runtime when you are ready to preflight and start it.`, `${selectedProfile?.display_name || profileKey} 已连接到 ${accountKey}。准备好后前往“Worker 运行”执行预检并启动。`)}</p></div></div> : null}

        {view === "lifecycle" && step === 0 ? <div className="provisioning-target-list">
          {records.length ? records.map((record) => <button className={`provisioning-target${selected?.id === record.id ? " active" : ""}`} key={record.id} onClick={() => selectRecord(record)} type="button"><span><strong>{record.profile_key}</strong><small>{record.account_key}</small></span><span className="provisioning-target-state">{record.observed_worker?.state || record.status}</span></button>) : <div className="provisioning-empty"><Bot size={20} /><span>{tr("No Worker drafts yet. First connect an account in Feishu connections.", "还没有 Worker 草稿。请先到“飞书连接”完成账号连接。")}</span></div>}
        </div> : null}

        {view === "lifecycle" && step === 1 ? <div className="provisioning-form">
          <div className="provisioning-review full"><ShieldCheck size={21} /><div><strong>{tr("Preflight uses the saved Worker draft", "预检使用已保存的 Worker 草稿")}</strong><p>{tr("Adjust runtime options here, then ask the server to check provider and Feishu readiness.", "在这里调整运行选项，然后让服务端检查 Provider 和飞书是否就绪。")}</p></div></div>
          <label>{tr("Provider", "Provider")}<select value={provider} onChange={(event) => { setProvider(event.target.value); setModel(providers.find((item) => item.name === event.target.value)?.model || ""); }}>{providers.map((item) => <option value={item.name} key={item.name}>{item.name} · {item.model}</option>)}</select></label>
          <label>{tr("Model", "模型")}<input value={model} onChange={(event) => setModel(event.target.value)} /></label>
          <label>{tr("Streaming", "流式输出")}<select value={streaming} onChange={(event) => setStreaming(event.target.value)}><option value="on">{tr("On · Feishu card updates", "开启 · 飞书卡片实时更新")}</option><option value="off">{tr("Off", "关闭")}</option></select></label>
          <label>{tr("Reactions", "Reaction 状态")}<select value={reactions} onChange={(event) => setReactions(event.target.value)}><option value="on">{tr("On · typing and done", "开启 · 处理中与完成")}</option><option value="off">{tr("Off", "关闭")}</option></select></label>
        </div> : null}

        {view === "lifecycle" && step === 2 ? <div className="provisioning-actions"><button className="primary-button" disabled={busy || !selected} onClick={() => void action("start")} type="button"><Play size={16} /> {tr("Start Worker", "启动 Worker")}</button><button className="secondary-button" disabled={busy || !selected} onClick={() => void action("restart")} type="button"><RefreshCw size={16} /> {tr("Restart", "重启")}</button><button className="secondary-button danger" disabled={busy || !selected} onClick={() => void action("stop")} type="button"><Square size={15} /> {tr("Stop", "停止")}</button></div> : null}
        {view === "lifecycle" && step === 3 ? <div className="provisioning-review"><CheckCircle2 size={21} /><div><strong>{tr("Run the final smoke check in Feishu", "在飞书执行最终冒烟测试")}</strong><p>{tr("Mention the bot in a group or send a DM, then refresh the observed state here.", "在群里 @ 机器人或发送一条私聊消息，然后在这里刷新状态。")}</p></div></div> : null}

        <div className="provisioning-form-footer">
          <button className="secondary-button" disabled={step === 0 || busy} onClick={() => setStep((current) => current - 1)} type="button"><ArrowLeft size={15} /> {tr("Back", "返回")}</button>
          <div className="button-row">
            {view === "account" && step === 0 ? <button className="primary-button" disabled={busy || !profileKey} onClick={() => setStep(1)} type="button">{tr("Continue", "继续")} <ArrowRight size={15} /></button> : null}
            {view === "account" && step === 1 ? <button className="primary-button" disabled={busy || !accountKey} onClick={() => void createConnectionDraft()} type="button"><Bot size={15} /> {tr("Save connection draft", "保存连接草稿")}</button> : null}
            {view === "lifecycle" && step === 0 ? <button className="primary-button" disabled={busy || !selected} onClick={() => setStep(1)} type="button">{tr("Continue", "继续")} <ArrowRight size={15} /></button> : null}
            {view === "lifecycle" && step === 1 ? <button className="primary-button" disabled={busy || !selected} onClick={() => void action("preflight")} type="button"><ShieldCheck size={15} /> {tr("Run preflight", "运行预检")}</button> : null}
            {view === "lifecycle" && step === 2 ? <button className="primary-button" disabled={busy || !selected} onClick={() => setStep(3)} type="button">{tr("Continue", "继续")} <ArrowRight size={15} /></button> : null}
            {view === "lifecycle" && step === 3 ? <button className="primary-button" disabled={busy || !selected} onClick={() => void action("status")} type="button"><RefreshCw size={16} /> {tr("Refresh readback", "刷新状态")}</button> : null}
          </div>
        </div>
      </section>
    </div>
  </div>;
}
