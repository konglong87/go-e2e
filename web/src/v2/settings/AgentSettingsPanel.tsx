import { Bot, RefreshCcw, Save } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ApiError, getAgentProfileAssignment, getAgentProfileBinding, listAgentProfiles, saveAgentProfileAssignment } from "../../lib/api";
import { parseAgentProfileJSON, profileCapabilitySummary } from "../../lib/agentProfiles";
import { useI18n } from "../../lib/i18n";
import type { AgentProfileAssignment, AgentProfileChannelBinding, AgentProfileRecord } from "../../lib/types";
import type { ProfileSettingsPanelProps } from "./ProfileSettingsPanel";
import "./profile-settings.css";

const SURFACES = [
  { key: "web_chat", en: "Web chat", zh: "Web 对话" },
  { key: "mobile_chat", en: "Mobile chat", zh: "移动端对话" },
  { key: "tenant_agent", en: "Tenant agent", zh: "租户智能体" },
  { key: "channel_dm", en: "Channel direct messages", zh: "渠道私聊" },
  { key: "channel_team", en: "Channel teams", zh: "渠道团队" },
] as const;
type Surface = typeof SURFACES[number]["key"];
type Assignments = Partial<Record<Surface, AgentProfileAssignment | null>>;
type Errors = Partial<Record<Surface, string>>;
const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error);

export function AgentSettingsPanel({ identity, onDirtyChange, onBusyChange, refreshVersion = 0 }: ProfileSettingsPanelProps & { refreshVersion?: number }) {
  const { language } = useI18n();
  const zh = language === "zh";
  const [profiles, setProfiles] = useState<AgentProfileRecord[]>([]);
  const [assignments, setAssignments] = useState<Assignments>({});
  const [edits, setEdits] = useState<Partial<Record<Surface, number>>>({});
  const [errors, setErrors] = useState<Errors>({});
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [reload, setReload] = useState(0);
  const [stale, setStale] = useState(false);
  const seenRefreshVersion = useRef(refreshVersion);
  const [binding, setBinding] = useState<AgentProfileChannelBinding | null>(null);
  const [bindingError, setBindingError] = useState("");
  const [bindingLoading, setBindingLoading] = useState(false);
  const dirtyCallback = useRef(onDirtyChange);
  dirtyCallback.current = onDirtyChange;
  const changed = SURFACES.filter(({ key }) => edits[key] !== undefined && edits[key] !== (assignments[key]?.profile_id || 0));
  const dirty = changed.length > 0;
  const published = profiles.filter((profile) => profile.status === "published" && profile.id > 0);
  const current = profiles.find((profile) => profile.id === assignments.web_chat?.profile_id);

  useEffect(() => { dirtyCallback.current?.(dirty); }, [dirty]);
  useEffect(() => { onBusyChange?.(saving); }, [saving, onBusyChange]);
  useEffect(() => () => dirtyCallback.current?.(false), []);

  useEffect(() => {
    if (seenRefreshVersion.current === refreshVersion) return;
    seenRefreshVersion.current = refreshVersion;
    if (dirty || saving) setStale(true);
    else setReload((value) => value + 1);
  }, [refreshVersion, dirty, saving]);

  useEffect(() => {
    let active = true;
    setLoading(true); setError(""); setStatus(""); setStale(false);
    async function load() {
      const results = await Promise.allSettled([
        listAgentProfiles(identity),
        ...SURFACES.map(({ key }) => getAgentProfileAssignment(identity, key)),
      ]);
      if (!active) return;
      const catalog = results[0];
      if (catalog.status === "rejected") {
        setError(errorMessage(catalog.reason)); setProfiles([]);
      } else setProfiles(catalog.value as AgentProfileRecord[]);
      const next: Assignments = {};
      const nextErrors: Errors = {};
      SURFACES.forEach(({ key }, index) => {
        const result = results[index + 1];
        if (result.status === "fulfilled") next[key] = result.value as AgentProfileAssignment;
        else if (result.reason instanceof ApiError && result.reason.status === 404) next[key] = null;
        else nextErrors[key] = errorMessage(result.reason);
      });
      setAssignments(next); setErrors(nextErrors); setEdits({}); setLoading(false);
    }
    void load();
    return () => { active = false; };
  }, [identity.apiBase, identity.apiToken, identity.tenantKey, identity.userId, reload]);

  useEffect(() => {
    let active = true;
    setBinding(null); setBindingError("");
    if (!current) { setBindingLoading(false); return; }
    setBindingLoading(true);
    void getAgentProfileBinding(identity, current.profile_key, current.profile_version)
      .then((value) => { if (active) setBinding(value); })
      .catch((reason) => { if (active && !(reason instanceof ApiError && reason.status === 404)) setBindingError(errorMessage(reason)); })
      .finally(() => { if (active) setBindingLoading(false); });
    return () => { active = false; };
  }, [identity.apiBase, identity.apiToken, identity.tenantKey, identity.userId, current?.id, current?.profile_key, current?.profile_version]);

  async function save() {
    if (saving || !dirty) return;
    setSaving(true); setError(""); setStatus("");
    try {
      // Read back each completed write so partial failures retain only pending edits.
      for (const { key } of changed) {
        const profileId = edits[key];
        if (!profileId || !published.some((profile) => profile.id === profileId)) throw new Error(zh ? "请选择已发布的 Profile" : "Select a published profile");
        await saveAgentProfileAssignment(identity, { surface: key, profile_id: profileId });
        const confirmed = await getAgentProfileAssignment(identity, key);
        if (confirmed.profile_id !== profileId) throw new Error(zh ? "分配回读不一致，请刷新后重试" : "Assignment readback differs; refresh and retry");
        setAssignments((previous) => ({ ...previous, [key]: confirmed }));
        setEdits((previous) => { const next = { ...previous }; delete next[key]; return next; });
      }
      setStatus(zh ? "入口分配已保存" : "Assignments saved");
    } catch (reason) { setError(errorMessage(reason)); }
    finally { setSaving(false); }
  }

  let summary: ReturnType<typeof profileCapabilitySummary> | null = null;
  let mode = "";
  if (current) {
    try { const config = parseAgentProfileJSON(current.config_json); summary = profileCapabilitySummary(config); mode = config.prompt.mode; } catch { /* Catalog metadata remains available for malformed historical documents. */ }
  }
  return <div className="webui2-agent-settings" aria-busy={loading || saving}>
    <div className="settings-profile-toolbar">
      <h2>{zh ? "入口分配" : "Surface assignments"}</h2>
      <div className="button-row">
        <button type="button" className="icon-button" title={zh ? "刷新分配" : "Refresh assignments"} aria-label={zh ? "刷新分配" : "Refresh assignments"} disabled={loading || saving} onClick={() => { if (!dirty || window.confirm(zh ? "放弃尚未保存的修改？" : "Discard unsaved changes?")) setReload((value) => value + 1); }}><RefreshCcw size={16} /></button>
        <button type="button" className="primary-button" disabled={loading || saving || !dirty} onClick={() => void save()}><Save size={15} />{saving ? (zh ? "保存中..." : "Saving...") : (zh ? "保存分配" : "Save assignments")}</button>
      </div>
    </div>
    {error ? <p role="alert" className="settings-profile-error">{error}</p> : null}
    {status ? <p role="status" className="settings-profile-status">{status}</p> : null}
    {stale ? <p role="status" className="settings-profile-stale">{zh ? "Profile 已更新。当前未保存的分配已保留，刷新后可选择最新发布版本。" : "Profiles changed. Unsaved assignments are preserved; refresh to load newly published versions."}</p> : null}
    {loading ? <p role="status">{zh ? "正在读取入口分配..." : "Loading assignments..."}</p> : <>
      <section className="settings-agent-overview" aria-label={zh ? "web_chat 入口分配" : "web_chat surface assignment"}>
        <Bot size={30} />
        <div><h3>{current?.display_name || (errors.web_chat ? (zh ? "Web 对话分配读取失败" : "Web chat assignment unavailable") : assignments.web_chat?.profile_id ? `Profile #${assignments.web_chat.profile_id}` : (zh ? "Web 对话尚未分配 Profile" : "No Web chat profile assigned"))}</h3><p>{current ? `${current.profile_key} · v${current.profile_version} · ${mode}` : (zh ? "当前租户与用户" : "Current tenant and user")}</p></div>
      </section>
      {current ? <dl className="settings-agent-facts">
        <div><dt>{zh ? "工具策略" : "Tool policy"}</dt><dd>{summary?.tools || (zh ? "配置不可读" : "Configuration unavailable")}</dd></div>
        <div><dt>{zh ? "技能" : "Skills"}</dt><dd>{summary?.skills ?? "-"}</dd></div>
        <div><dt>{zh ? "渠道绑定" : "Channel binding"}</dt><dd>{bindingLoading ? (zh ? "读取中..." : "Loading...") : bindingError ? (zh ? "读取失败" : "Unavailable") : binding ? `${binding.binding_key} · ${binding.provider}` : (zh ? "未绑定" : "Not bound")}</dd></div>
      </dl> : null}
      {bindingError ? <p role="alert" className="settings-profile-error">{bindingError}</p> : null}
      <p className="settings-profile-boundary">{zh ? "分配仅用于已接入 Profile 的入口。WebUI v2 托管会话暂不读取这些分配，当前 Run 不受影响。" : "Assignments apply to Profile-aware entrypoints. WebUI v2 managed sessions do not consume these assignments; current Runs are unchanged."}</p>
      <div className="settings-assignment-list">
        {SURFACES.map(({ key, en, zh: label }) => {
          const assignedId = assignments[key]?.profile_id || 0;
          const assignedProfile = profiles.find((profile) => profile.id === assignedId);
          const value = edits[key] ?? assignedId;
          return <div className="settings-assignment-row" key={key}>
            <div><label htmlFor={`assignment-${key}`}>{zh ? label : en}</label><small>{key}</small></div>
            <div><select id={`assignment-${key}`} value={value} disabled={saving || Boolean(errors[key]) || Boolean(error && profiles.length === 0)} onChange={(event) => { setEdits((previous) => ({ ...previous, [key]: Number(event.target.value) })); setStatus(""); }}>
              <option value={0} disabled>{zh ? "选择已发布 Profile" : "Select a published profile"}</option>
              {assignedId && !published.some((profile) => profile.id === assignedId) ? <option value={assignedId} disabled>{assignedProfile?.display_name || `Profile #${assignedId}`} · {zh ? "当前分配" : "Current assignment"}</option> : null}
              {published.map((profile) => <option key={profile.id} value={profile.id}>{profile.display_name} · v{profile.profile_version}</option>)}
            </select><small>{errors[key] ? errors[key] : assignedProfile ? `${zh ? "当前" : "Current"}: ${assignedProfile.display_name} · v${assignedProfile.profile_version}` : assignedId ? `Profile #${assignedId}` : (zh ? "尚未分配" : "Not assigned")}</small></div>
            <span className="settings-assignment-state">{edits[key] !== undefined && edits[key] !== assignedId ? (zh ? "未保存" : "Unsaved") : errors[key] ? (zh ? "读取失败" : "Load failed") : assignedId ? (zh ? "已分配" : "Assigned") : "-"}</span>
          </div>;
        })}
      </div>
      {!published.length && !error ? <p>{zh ? "暂无已发布版本" : "No published versions available"}</p> : null}
    </>}
  </div>;
}
