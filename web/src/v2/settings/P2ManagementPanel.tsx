import { Brain, Plus, RefreshCw, Save, Sparkles } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { listMemories, listEffectiveSkills, saveMemory, saveSkill } from "../../lib/api";
import type { IdentityConfig, MemoryRecord, SkillRecord } from "../../lib/types";
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
        const records = await listEffectiveSkills(identity);
        if (active.current) setSkills(records);
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
      <div><h2>{section === "memory" ? <><Brain size={18} />{zh ? "持久记忆" : "Persistent memory"}</> : <><Sparkles size={18} />{zh ? "可用 Skills" : "Available skills"}</>}</h2><p>{section === "memory" ? (zh ? "保存后会进入桌面端运行时上下文。" : "Saved records can be used by the desktop runtime.") : (zh ? "Profile 引用的技能会在对话中注入。" : "Skills referenced by Profiles are injected into conversations.")}</p></div>
      <button type="button" onClick={() => void refresh()} disabled={busy} title={zh ? "刷新" : "Refresh"}><RefreshCw size={15} />{zh ? "刷新" : "Refresh"}</button>
    </header>
    <div className="p2-management-create">
      <label>{section === "memory" ? (zh ? "记忆 Key" : "Memory key") : (zh ? "Skill Key" : "Skill key")}<input disabled={busy} value={key} onChange={(event) => setKey(event.target.value)} /></label>
      {section === "skills" ? <label>{zh ? "名称" : "Name"}<input disabled={busy} value={name} onChange={(event) => setName(event.target.value)} /></label> : null}
      <label className="full">{section === "memory" ? (zh ? "内容" : "Content") : "Markdown"}<textarea disabled={busy} rows={4} value={content} onChange={(event) => setContent(event.target.value)} /></label>
      <button className="primary" type="button" onClick={() => void create()} disabled={busy || !key.trim() || !content.trim()}><Plus size={15} /><Save size={15} />{zh ? "保存" : "Save"}</button>
    </div>
    {status ? <p role="status" className="p2-management-status">{status}</p> : null}
    <div className="p2-management-list">
      {section === "memory" ? memories.map((item) => <article key={String(item.memory_key)}><strong>{item.memory_key}</strong><p>{item.content}</p></article>) : skills.map((item) => <article key={String(item.skill_key)}><strong>{item.skill_key}</strong><p>{item.name}</p></article>)}
      {!busy && (section === "memory" ? memories : skills).length === 0 ? <p>{zh ? "暂无记录。" : "No records yet."}</p> : null}
    </div>
  </section>;
}
