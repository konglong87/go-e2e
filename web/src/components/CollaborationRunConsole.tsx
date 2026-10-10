import {
  Activity,
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  CircleDashed,
  CircleStop,
  Clock3,
  Code2,
  Eye,
  GitBranch,
  MessageSquare,
  ShieldAlert,
  Sparkles,
  UserRound,
  Users,
  XCircle
} from "lucide-react";
import { useMemo, useState, type JSX } from "react";
import type { AgentProfileRecord, AgentTeamMember, AgentTeamRecord, AgentTeamRun } from "../lib/types";
import "./collaborationRunConsole.css";

type Language = "en" | "zh";
type RunFilter = "all" | "active" | "completed" | "attention";
type NodeState = "completed" | "running" | "failed" | "cancelled" | "waiting" | "observed";

type TeamRunConsoleProps = {
  language: Language;
  team: AgentTeamRecord | null;
  members: AgentTeamMember[];
  profiles: AgentProfileRecord[];
  runs: AgentTeamRun[];
  onInspect: (run: AgentTeamRun) => void;
  onCancel: (run: AgentTeamRun) => void;
};

type TeamRunReplayConsoleProps = {
  language: Language;
  team: AgentTeamRecord | null;
  members: AgentTeamMember[];
  profiles: AgentProfileRecord[];
  run: AgentTeamRun | null;
  validation: { valid: boolean; issues?: Array<{ code: string; message: string }> } | null;
  onBack: () => void;
};

const copy = {
  en: {
    monitor: "Execution workspace",
    monitorTitle: "TeamRun monitor",
    monitorSubtitle: "Follow the team as one run: who is active, what is complete, and what needs attention.",
    replay: "Run replay",
    replaySubtitle: "A calm, readable snapshot of the pinned team and the evidence available for this run.",
    allRuns: "All runs",
    active: "Active",
    completed: "Completed",
    attention: "Needs attention",
    runs: "runs",
    members: "members",
    tokens: "tokens",
    turns: "turns",
    coordinator: "Coordinator",
    latest: "Latest",
    topology: "Team topology",
    topologyHint: "Pinned members from the published Team version.",
    execution: "Execution snapshot",
    finalResult: "Coordinator result",
    evidence: "Evidence",
    noEvidence: "No structured evidence was recorded for this run yet.",
    nodeEventsPending: "Member-level events are not available in this snapshot yet.",
    nodeEventsPendingHint: "The TeamRun is still traceable at run level; a future event stream will fill in each member's live activity.",
    runID: "Run ID",
    teamVersion: "Team version",
    status: "Status",
    started: "Started",
    finished: "Finished",
    duration: "Duration",
    coordinatorMember: "Coordinator member",
    view: "View run",
    cancel: "Stop run",
    back: "Back to runs",
    noRuns: "No TeamRun records yet",
    noRunsHint: "Once a bound Team receives a message, its durable execution will appear here.",
    noTeam: "No Team selected",
    noTeamHint: "Create or select a Team to inspect its topology and runs.",
    pinned: "Pinned",
    observed: "Included",
    role: "Role",
    profile: "Profile",
    version: "Version",
    external: "External delivery",
    internal: "Internal mailbox",
    result: "Result",
    rawEvidence: "Raw result payload",
    validation: "Validation",
    valid: "Valid",
    invalid: "Needs fixes",
    noMembers: "No members configured",
    noMembersHint: "Add published Profiles to the Team before publishing it.",
    minutesAgo: "m ago",
    seconds: "s"
  },
  zh: {
    monitor: "执行工作区",
    monitorTitle: "TeamRun 运行监控",
    monitorSubtitle: "把一次团队执行看清楚：谁在运行、什么已完成、哪里需要处理。",
    replay: "运行回放",
    replaySubtitle: "以更安静、更清晰的方式查看固定版本团队和本次运行已有的证据。",
    allRuns: "全部运行",
    active: "进行中",
    completed: "已完成",
    attention: "需关注",
    runs: "次运行",
    members: "成员",
    tokens: "tokens",
    turns: "轮",
    coordinator: "协调者",
    latest: "最近一次",
    topology: "团队拓扑",
    topologyHint: "来自已发布 Team 版本的固定成员。",
    execution: "执行快照",
    finalResult: "协调者结果",
    evidence: "执行证据",
    noEvidence: "本次运行暂未记录结构化证据。",
    nodeEventsPending: "当前快照还没有成员级事件。",
    nodeEventsPendingHint: "TeamRun 仍然可以按运行级别追踪；后续事件流会补齐每个成员的实时活动。",
    runID: "运行 ID",
    teamVersion: "团队版本",
    status: "状态",
    started: "开始时间",
    finished: "结束时间",
    duration: "耗时",
    coordinatorMember: "协调者成员",
    view: "查看运行",
    cancel: "停止运行",
    back: "返回运行记录",
    noRuns: "暂时没有 TeamRun 记录",
    noRunsHint: "绑定的 Team 收到消息后，持久化执行记录会出现在这里。",
    noTeam: "尚未选择 Team",
    noTeamHint: "请先创建或选择一个 Team，查看它的拓扑和运行记录。",
    pinned: "已固定",
    observed: "已纳入",
    role: "职责",
    profile: "Profile",
    version: "版本",
    external: "外部输出",
    internal: "内部 mailbox",
    result: "结果",
    rawEvidence: "原始结果载荷",
    validation: "校验",
    valid: "有效",
    invalid: "需要修复",
    noMembers: "尚未配置成员",
    noMembersHint: "发布 Team 前，请先加入已发布的 Profile。",
    minutesAgo: "分钟前",
    seconds: "秒"
  }
} as const;

type Copy = { [Key in keyof typeof copy.en]: string };

const roleLabel = (role: string, language: Language): string => {
  const labels: Record<string, [string, string]> = {
    coordinator: ["Coordinator", "协调者"],
    researcher: ["Researcher", "研究员"],
    writer: ["Writer", "写作者"],
    coder: ["Coder", "编码者"],
    reviewer: ["Reviewer", "审阅者"]
  };
  return labels[role]?.[language === "zh" ? 1 : 0] ?? role;
};

const statusLabel = (status: string, language: Language): string => {
  const labels: Record<string, [string, string]> = {
    queued: ["Queued", "排队中"],
    running: ["Running", "运行中"],
    waiting_input: ["Waiting for input", "等待输入"],
    completed: ["Completed", "已完成"],
    partial: ["Partial", "部分完成"],
    cancelled: ["Cancelled", "已取消"],
    failed: ["Failed", "失败"],
    timed_out: ["Timed out", "超时"]
  };
  return labels[status]?.[language === "zh" ? 1 : 0] ?? status;
};

const statusTone = (status: string): string => {
  if (status === "completed") return "success";
  if (["failed", "timed_out", "partial"].includes(status)) return "danger";
  if (["running", "queued", "waiting_input"].includes(status)) return "warning";
  return "neutral";
};

const filterMatches = (run: AgentTeamRun, filter: RunFilter): boolean => {
  if (filter === "all") return true;
  if (filter === "active") return ["queued", "running", "waiting_input"].includes(run.status);
  if (filter === "completed") return run.status === "completed";
  return ["failed", "partial", "timed_out", "cancelled"].includes(run.status);
};

const formatDate = (value: string | undefined, language: Language): string => {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(language === "zh" ? "zh-CN" : "en-US", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
};

const duration = (run: AgentTeamRun): string => {
  if (!run.started_at) return "—";
  const end = run.finished_at ? Date.parse(run.finished_at) : Date.now();
  const start = Date.parse(run.started_at);
  if (!Number.isFinite(start) || !Number.isFinite(end)) return "—";
  const seconds = Math.max(0, Math.round((end - start) / 1000));
  if (seconds < 60) return `${seconds}s`;
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
};

const profileName = (profile: AgentProfileRecord | undefined, member: AgentTeamMember, language: Language): string => {
  if (profile?.display_name) return profile.display_name;
  return member.member_key || (language === "zh" ? "未命名成员" : "Unnamed member");
};

function parseRunMembers(run: AgentTeamRun | null): Map<string, NodeState> {
  if (!run?.result_json) return new Map();
  try {
    const raw: unknown = JSON.parse(run.result_json);
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) return new Map();
    const members = (raw as { members?: unknown }).members;
    if (!Array.isArray(members)) return new Map();
    const states = new Map<string, NodeState>();
    for (const item of members) {
      if (!item || typeof item !== "object") continue;
      const member = item as { member_key?: unknown; status?: unknown };
      if (typeof member.member_key !== "string" || typeof member.status !== "string") continue;
      if (["completed", "running", "failed", "cancelled", "waiting"].includes(member.status)) states.set(member.member_key, member.status as NodeState);
    }
    return states;
  } catch {
    return new Map();
  }
}

export function TeamRunConsole({ language, team, members, profiles, runs, onInspect, onCancel }: TeamRunConsoleProps): JSX.Element {
  const labels = copy[language];
  const [filter, setFilter] = useState<RunFilter>("all");
  const filteredRuns = useMemo(() => runs.filter((run) => filterMatches(run, filter)), [filter, runs]);
  const activeCount = runs.filter((run) => filterMatches(run, "active")).length;
  const attentionCount = runs.filter((run) => filterMatches(run, "attention")).length;
  const latest = runs[0];

  return (
    <section className="collaboration-console" aria-label={labels.monitorTitle}>
      <header className="collaboration-console-header">
        <div className="collaboration-console-heading">
          <span className="collaboration-kicker"><Activity size={14} />{labels.monitor}</span>
          <h3>{labels.monitorTitle}</h3>
          <p>{labels.monitorSubtitle}</p>
        </div>
        {team ? <div className="collaboration-team-identity"><span>{team.display_name}</span><small>{team.team_key} · v{team.team_version}</small></div> : null}
      </header>

      <div className="collaboration-summary-grid">
        <SummaryMetric icon={<Activity size={16} />} label={labels.runs} value={runs.length} tone="accent" />
        <SummaryMetric icon={<CircleDashed size={16} />} label={labels.active} value={activeCount} tone="warning" />
        <SummaryMetric icon={<ShieldAlert size={16} />} label={labels.attention} value={attentionCount} tone={attentionCount ? "danger" : "neutral"} />
        <SummaryMetric icon={<Users size={16} />} label={labels.members} value={members.length} tone="neutral" />
      </div>

      <div className="collaboration-filter-row" role="tablist" aria-label={language === "zh" ? "运行筛选" : "Run filters"}>
        {(["all", "active", "completed", "attention"] as RunFilter[]).map((item) => {
          const count = item === "all" ? runs.length : item === "active" ? activeCount : item === "completed" ? runs.filter((run) => run.status === "completed").length : attentionCount;
          const label = item === "all" ? labels.allRuns : item === "active" ? labels.active : item === "completed" ? labels.completed : labels.attention;
          return <button aria-selected={filter === item} className={filter === item ? "active" : ""} key={item} onClick={() => setFilter(item)} role="tab" type="button"><span>{label}</span><strong>{count}</strong></button>;
        })}
      </div>

      {runs.length === 0 ? <EmptyRunState labels={labels} team={team} /> : filteredRuns.length === 0 ? <div className="collaboration-empty-inline">{language === "zh" ? "当前筛选没有匹配的运行记录。" : "No runs match this filter."}</div> : <div className="collaboration-run-list">{filteredRuns.map((run) => <RunCard key={run.id} language={language} labels={labels} run={run} latest={run.id === latest?.id} onInspect={onInspect} onCancel={onCancel} />)}</div>}
    </section>
  );
}

export function TeamRunReplayConsole({ language, team, members, profiles, run, validation, onBack }: TeamRunReplayConsoleProps): JSX.Element {
  const labels = copy[language];
  const memberProfiles = useMemo(() => new Map(profiles.map((profile) => [profile.id, profile])), [profiles]);
  const memberStates = useMemo(() => parseRunMembers(run), [run]);

  return (
    <section className="collaboration-console collaboration-console--replay" aria-label={labels.replay}>
      <header className="collaboration-console-header">
        <div className="collaboration-console-heading">
          <button className="collaboration-back-button" onClick={onBack} type="button"><ArrowLeft size={15} />{labels.back}</button>
          <span className="collaboration-kicker"><GitBranch size={14} />{labels.replay}</span>
          <h3>{run ? run.id : labels.noTeam}</h3>
          <p>{labels.replaySubtitle}</p>
        </div>
        {run ? <StatusBadge language={language} status={run.status} /> : null}
      </header>

      {run ? <>
        <div className="collaboration-run-hero">
          <div><span className="collaboration-run-hero-label">{labels.coordinatorMember}</span><strong>{run.coordinator_member_key || "—"}</strong></div>
          <div><span className="collaboration-run-hero-label">{labels.duration}</span><strong>{duration(run)}</strong></div>
          <div><span className="collaboration-run-hero-label">{labels.tokens}</span><strong>{(run.used_tokens || 0).toLocaleString()}</strong></div>
          <div><span className="collaboration-run-hero-label">{labels.turns}</span><strong>{(run.used_turns || 0).toLocaleString()}</strong></div>
        </div>

        <div className="collaboration-replay-grid">
          <section className="collaboration-surface collaboration-topology-card">
            <SectionHeading icon={<Users size={15} />} title={labels.topology} hint={labels.topologyHint} />
            {members.length === 0 ? <EmptyMemberState labels={labels} /> : <div className="collaboration-node-list">{members.map((member) => {
              const profile = memberProfiles.get(member.profile_id);
              const state = memberStates.get(member.member_key) ?? "observed";
              return <div className={`collaboration-node collaboration-node--${state}`} key={member.id ?? member.member_key}>
                <span className="collaboration-node-icon"><NodeIcon role={member.role} /></span>
                <div className="collaboration-node-main"><strong>{profileName(profile, member, language)}</strong><span>{roleLabel(member.role, language)} · {profile?.profile_key || "—"}{profile?.profile_version ? `@v${profile.profile_version}` : ""}</span></div>
                <span className="collaboration-node-state">{state === "observed" ? labels.observed : statusLabel(state, language)}</span>
              </div>;
            })}</div>}
            <div className="collaboration-node-note"><CircleDashed size={14} /><span>{labels.nodeEventsPending}</span></div>
            <p className="collaboration-muted-note">{labels.nodeEventsPendingHint}</p>
          </section>

          <section className="collaboration-surface collaboration-facts-card">
            <SectionHeading icon={<Clock3 size={15} />} title={labels.execution} hint={team ? `${team.team_key} · v${team.team_version}` : "—"} />
            <dl className="collaboration-facts">
              <Fact label={labels.runID} value={run.id} />
              <Fact label={labels.status} value={statusLabel(run.status, language)} />
              <Fact label={labels.started} value={formatDate(run.started_at, language)} />
              <Fact label={labels.finished} value={formatDate(run.finished_at, language)} />
              <Fact label={labels.teamVersion} value={team ? `${team.team_key} · v${team.team_version}` : "—"} />
            </dl>
            {run.error_message ? <div className="collaboration-error-callout"><XCircle size={16} /><span>{run.error_message}</span></div> : null}
            {validation ? <div className={`collaboration-validation ${validation.valid ? "valid" : "invalid"}`}><span>{labels.validation}</span><strong>{validation.valid ? labels.valid : labels.invalid}</strong></div> : null}
          </section>
        </div>

        <section className="collaboration-surface collaboration-result-card">
          <SectionHeading icon={<Sparkles size={15} />} title={labels.finalResult} hint={run.coordinator_member_key || labels.coordinator} />
          {run.result_json ? <div className="collaboration-result-copy">{run.result_json}</div> : <div className="collaboration-empty-inline">{labels.noEvidence}</div>}
          {run.result_json ? <details className="collaboration-raw-details"><summary>{labels.rawEvidence}</summary><pre>{run.result_json}</pre></details> : null}
        </section>
      </> : <div className="collaboration-empty-state"><CircleDashed size={30} /><strong>{labels.noTeam}</strong><p>{labels.noTeamHint}</p></div>}
    </section>
  );
}

function RunCard({ language, labels, run, latest, onInspect, onCancel }: { language: Language; labels: Copy; run: AgentTeamRun; latest: boolean; onInspect: (run: AgentTeamRun) => void; onCancel: (run: AgentTeamRun) => void }): JSX.Element {
  const active = ["queued", "running", "waiting_input"].includes(run.status);
  return <article className={`collaboration-run-card collaboration-run-card--${statusTone(run.status)}`}>
    <button className="collaboration-run-card-main" onClick={() => onInspect(run)} type="button">
      <span className="collaboration-run-status-mark"><StatusIcon status={run.status} /></span>
      <span className="collaboration-run-card-copy"><strong>{latest ? `${labels.latest} · ` : ""}{run.id}</strong><span>{statusLabel(run.status, language)} · {formatDate(run.created_at || run.started_at, language)}</span></span>
      <span className="collaboration-run-card-metrics"><span>{(run.used_tokens || 0).toLocaleString()} {labels.tokens}</span><span>{run.used_turns || 0} {labels.turns}</span><span>{run.member_count || 0} {labels.members}</span></span>
      <ArrowRight className="collaboration-run-card-arrow" size={17} />
    </button>
    <div className="collaboration-run-card-actions"><StatusBadge language={language} status={run.status} />{active ? <button aria-label={labels.cancel} className="collaboration-icon-button danger" onClick={() => onCancel(run)} title={labels.cancel} type="button"><CircleStop size={15} /></button> : <button aria-label={labels.view} className="collaboration-icon-button" onClick={() => onInspect(run)} title={labels.view} type="button"><Eye size={15} /></button>}</div>
  </article>;
}

function EmptyRunState({ labels, team }: { labels: Copy; team: AgentTeamRecord | null }): JSX.Element {
  return <div className="collaboration-empty-state"><span className="collaboration-empty-icon"><Activity size={20} /></span><strong>{team ? labels.noRuns : labels.noTeam}</strong><p>{team ? labels.noRunsHint : labels.noTeamHint}</p></div>;
}

function EmptyMemberState({ labels }: { labels: Copy }): JSX.Element {
  return <div className="collaboration-empty-member"><Users size={20} /><strong>{labels.noMembers}</strong><span>{labels.noMembersHint}</span></div>;
}

function SectionHeading({ icon, title, hint }: { icon: JSX.Element; title: string; hint: string }): JSX.Element {
  return <div className="collaboration-section-heading"><span className="collaboration-section-icon">{icon}</span><div><strong>{title}</strong><span>{hint}</span></div></div>;
}

function SummaryMetric({ icon, label, value, tone }: { icon: JSX.Element; label: string; value: number; tone: string }): JSX.Element {
  return <div className={`collaboration-summary-metric collaboration-summary-metric--${tone}`}><span>{icon}</span><div><small>{label}</small><strong>{value}</strong></div></div>;
}

function StatusBadge({ language, status }: { language: Language; status: string }): JSX.Element {
  return <span className={`collaboration-status-badge collaboration-status-badge--${statusTone(status)}`}><span />{statusLabel(status, language)}</span>;
}

function StatusIcon({ status }: { status: string }): JSX.Element {
  if (status === "completed") return <CheckCircle2 size={17} />;
  if (["failed", "timed_out"].includes(status)) return <XCircle size={17} />;
  if (["running", "queued", "waiting_input"].includes(status)) return <Activity size={17} />;
  return <CircleDashed size={17} />;
}

function NodeIcon({ role }: { role: string }): JSX.Element {
  if (role === "coder" || role === "writer") return <Code2 size={16} />;
  if (role === "coordinator") return <Sparkles size={16} />;
  if (role === "reviewer") return <ShieldAlert size={16} />;
  if (role === "researcher") return <MessageSquare size={16} />;
  return <UserRound size={16} />;
}

function Fact({ label, value }: { label: string; value: string }): JSX.Element {
  return <div><dt>{label}</dt><dd title={value}>{value}</dd></div>;
}
