import { AlertTriangle, CheckCircle2, CircleDashed, LoaderCircle } from "lucide-react";
import type { ProvisioningRecord } from "../../lib/types";
import type { Language } from "../../lib/i18n";

export function ProvisioningStatusPanel({ record, language }: { record: ProvisioningRecord | null; language: Language }) {
  const tr = (en: string, zh: string) => language === "zh" ? zh : en;
  if (!record) return <div className="provisioning-empty"><CircleDashed size={20} /><span>{tr("Create a profile agent to begin the lifecycle.", "创建一个 Profile 智能体开始生命周期操作。")}</span></div>;
  const state = record.observed_worker?.state || record.status;
  const icon = state === "running" || state === "preflight" ? <CheckCircle2 size={17} /> : state === "failed" || state === "degraded" ? <AlertTriangle size={17} /> : <LoaderCircle size={17} className="spin" />;
  return <div className={`provisioning-status-card ${state}`}><div className="provisioning-status-icon">{icon}</div><div><span>{tr("Current state", "当前状态")}</span><strong>{state}</strong><small>{record.account_key} · {record.worker?.provider || tr("provider pending", "等待 Provider")}</small></div><div className="provisioning-status-meta"><span>screen</span><strong>{record.observed_worker?.screen || record.supervisor || "screen"}</strong></div></div>;
}
