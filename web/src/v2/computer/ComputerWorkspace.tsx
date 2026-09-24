import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { ComputerApprovalDialog } from "./ComputerApprovalDialog";
import { type ComputerClient } from "./client";
import { ComputerPreview } from "./ComputerPreview";
import { ComputerTimeline } from "./ComputerTimeline";
import { ComputerToolbar } from "./ComputerToolbar";
import { useComputerSession } from "./useComputerSession";

export function ComputerWorkspace({ client }: { client: ComputerClient | null }): ReactElement | null {
  const computer = useComputerSession(client);
  const [approvalOpen, setApprovalOpen] = useState(false);
  useEffect(() => { if (client) void computer.loadCapabilities(); }, [client]);
  if (!client) return null;
  const state = computer.session?.state ?? null;
  const start = async () => { setApprovalOpen(false); try { await computer.start({ approved: true }); await computer.observe(); } catch { /* surfaced in the panel */ } };
  const action = (operation: () => Promise<unknown>) => { void operation().catch(() => undefined); };
  return <aside className="webui2-computer-workspace" aria-label="Computer workspace"><header className="webui2-computer-header"><div><span className="webui2-computer-eyebrow">CONTROL SURFACE</span><h2>Computer Workspace</h2></div><span className="webui2-computer-backend">{computer.capabilities?.backend || "detecting backend"}</span></header><ComputerToolbar state={state} capabilities={computer.capabilities} busy={computer.loading} onStart={() => setApprovalOpen(true)} onObserve={() => action(computer.observe)} onPause={() => action(computer.pause)} onResume={() => action(computer.resume)} onStop={() => action(computer.stop)} />{computer.error ? <p className="webui2-computer-error" role="alert">{computer.error}</p> : null}<ComputerPreview observation={computer.observation} capabilities={computer.capabilities} /><ComputerTimeline receipts={computer.receipts} />{approvalOpen ? <ComputerApprovalDialog capabilities={computer.capabilities} busy={computer.loading} onApprove={() => void start()} onCancel={() => setApprovalOpen(false)} /> : null}</aside>;
}
