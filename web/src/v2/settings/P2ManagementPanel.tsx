import { Brain, RefreshCw, Save, Sparkles } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { listLocalSkills, listMemories, listEffectiveSkills, saveMemory, saveSkill } from "../../lib/api";
import type { IdentityConfig, LocalSkillRecord, MemoryRecord, SkillRecord } from "../../lib/types";
import type { SaveMemoryRequest } from "../../lib/api";
import { useI18n } from "../../lib/i18n";

type Props = { identity: IdentityConfig; section: "memory" | "skills" };
const managementSections = ["memory", "skills"] as const;

export function P2ManagementPanel(props: Props) {
  const { identity } = props;
  return <ManagementEditors key={JSON.stringify([identity.apiBase, identity.apiToken, identity.tenantKey, identity.userId])} {...props} />;
}

function ManagementEditors({ identity, section }: Props) {
  const [visited, setVisited] = useState(() => new Set([section]));
  useEffect(() => { setVisited((previous) => previous.has(section) ? previous : new Set([...previous, section])); }, [section]);
  // Keep drafts and pending completions owned by their original section.
  return managementSections.map((value) => visited.has(value) || value === section
    ? <div key={value} hidden={value !== section}><ManagementEditor identity={identity} section={value} /></div>
    : null);
}

function ManagementEditor({ identity, section }: Props) {
  const { language } = useI18n();
  const zh = language === "zh";
  const [memories, setMemories] = useState<MemoryRecord[]>([]);
  const [skills, setSkills] = useState<SkillRecord[]>([]);
  const [localSkills, setLocalSkills] = useState<LocalSkillRecord[]>([]);
  const [localError, setLocalError] = useState("");
  const [tenantError, setTenantError] = useState("");
  const [key, setKey] = useState(section === "memory" ? "desktop.preference" : "desktop-skill");
  const [name, setName] = useState("Desktop skill");
  const [content, setContent] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);

  async function refresh() {
    setBusy(true);
    setStatus("");
    try {
      if (section === "memory") {
        const records = await listMemories(identity);
        if (active.current) setMemories(records);
      } else {
        setLocalError("");
        setTenantError("");
        await Promise.all([
          listLocalSkills(identity).then(
            (records) => { if (active.current) setLocalSkills(records); },
            (error) => { if (active.current) setLocalError(error instanceof Error ? error.message : String(error)); }
          ),
          listEffectiveSkills(identity).then(
            (records) => { if (active.current) setSkills(records); },
            (error) => { if (active.current) setTenantError(error instanceof Error ? error.message : String(error)); }
          )
        ]);
      }
    } catch (error) {
      if (active.current) setStatus(error instanceof Error ? error.message : String(error));
    } finally {
      if (active.current) setBusy(false);
    }
  }
  useEffect(() => { void refresh(); }, [identity.apiBase, identity.apiToken, identity.tenantKey, identity.userId, section]);

  async function create() {
    if (busy || !key.trim() || !content.trim()) return;
    setBusy(true);
    try {
      if (section === "memory") {
        const request: SaveMemoryRequest = { memory_key: key.trim(), content: content.trim(), category: "preference", source: "desktop" };
        await saveMemory(identity, request);
      } else {
        await saveSkill(identity, { skill_key: key.trim(), name: name.trim() || key.trim(), content_md: content.trim(), enabled: true });
      }
      if (!active.current) return;
      setContent("");
      setStatus(zh ? "已保存并重新读取。" : "Saved and reloaded.");
      await refresh();
    } catch (error) {
      if (active.current) setStatus(error instanceof Error ? error.message : String(error));
    } finally {
      if (active.current) setBusy(false);
    }
  }

  return <section className="p2-management-panel">
    <header className="p2-management-toolbar">
      <div><h2>{section === "memory" ? <><Brain size={18} />{zh ? "持久记忆" : "Persistent memory"}</> : <><Sparkles size={18} />{zh ? "可用 Skills" : "Available skills"}</>}</h2></div>
      <button className="settings-icon-button" type="button" onClick={() => void refresh()} disabled={busy} aria-label={zh ? "刷新" : "Refresh"} title={zh ? "刷新" : "Refresh"}><RefreshCw size={15} /></button>
    </header>
    {status ? <p role="status" className="p2-management-status">{status}</p> : null}
    {section === "skills" ? <section className="p2-management-group" aria-label={zh ? "本机已安装 Skills" : "Locally installed skills"}>
        <header className="p2-management-group-heading">
          <h3>{zh ? "本机已安装 Skills" : "Locally installed skills"}</h3>
          <span>{localSkills.length}</span>
        </header>
        {localError ? <p role="alert" className="settings-error">{localError}</p> : null}
        <div className="p2-management-list">
          {localSkills.map((item) => <article key={`${item.name}:${item.path}`}>
            <strong>{item.name || item.local_name || item.path}</strong>
            {item.description ? <p>{item.description}</p> : null}
            <p className="p2-skill-meta">{[item.source, item.plugin ? `plugin: ${item.plugin}` : "", item.version ? `v${item.version}` : ""].filter(Boolean).join(" · ") || (zh ? "本机发现" : "Local discovery")}</p>
            {item.legacy ? <p className="p2-skill-meta">{zh ? "兼容命令" : "Legacy command"}</p> : null}
            <code className="p2-skill-path">{item.path}</code>
          </article>)}
          {!busy && !localError && localSkills.length === 0 ? <p>{zh ? "没有发现本机 Skills。" : "No local skills discovered."}</p> : null}
        </div>
    </section> : null}
    <section className="p2-management-group" aria-label={section === "skills" ? (zh ? "租户 Skills" : "Tenant skills") : undefined}>
      {section === "skills" ? <header className="p2-management-group-heading">
        <h3>{zh ? "租户 Skills" : "Tenant skills"}</h3>
        <span>{skills.length}</span>
      </header> : null}
      {tenantError ? <p role="alert" className="settings-error">{tenantError}</p> : null}
      <div className="p2-management-create">
        <label>{section === "memory" ? (zh ? "记忆 Key" : "Memory key") : (zh ? "Skill Key" : "Skill key")}<input disabled={busy} value={key} onChange={(event) => setKey(event.target.value)} /></label>
        {section === "skills" ? <label>{zh ? "名称" : "Name"}<input disabled={busy} value={name} onChange={(event) => setName(event.target.value)} /></label> : null}
        <label className="full">{section === "memory" ? (zh ? "内容" : "Content") : "Markdown"}<textarea disabled={busy} rows={4} value={content} onChange={(event) => setContent(event.target.value)} /></label>
        <button className="primary" type="button" onClick={() => void create()} disabled={busy || !key.trim() || !content.trim()}><Save size={15} />{zh ? "保存" : "Save"}</button>
      </div>
      <div className="p2-management-list">
        {section === "memory" ? memories.map((item) => <article key={String(item.memory_key)}><strong>{item.memory_key}</strong><p>{item.content}</p></article>) : skills.map((item) => <article key={String(item.skill_key)}><strong>{item.skill_key}</strong><p>{item.name}</p></article>)}
        {!busy && !tenantError && (section === "memory" ? memories : skills).length === 0 ? <p>{zh ? "暂无记录。" : "No records yet."}</p> : null}
      </div>
    </section>
  </section>;
}
