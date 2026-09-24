import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { ComputerApprovalDialog } from "./ComputerApprovalDialog";
import type { ComputerClient } from "./client";
import { ComputerPreview } from "./ComputerPreview";
import { ComputerTimeline } from "./ComputerTimeline";
import { ComputerToolbar } from "./ComputerToolbar";
import { computerReadinessError } from "./readiness";
import { useComputerSession } from "./useComputerSession";

export function ComputerWorkspace({ client }: { client: ComputerClient | null }): ReactElement | null {
  const computer = useComputerSession(client);
  const [approvalOpen, setApprovalOpen] = useState(false);
  useEffect(() => {
    setApprovalOpen(false);
    void computer.loadCapabilities().catch(() => undefined); // Hook renders the error.
  }, [computer.loadCapabilities]);
  if (!client) return null;

  const readiness = computerReadinessError(computer.available, computer.capabilities);
  const start = async () => {
    setApprovalOpen(false);
    try {
      const session = await computer.start({ approved: true });
      // A superseded request returns null. Do not observe after cancellation/replacement.
      if (session?.state === "ready" || session?.state === "needs_observation") await computer.observe();
    } catch { /* surfaced in the panel */ }
  };
  const action = (operation: () => Promise<unknown>) => { void operation().catch(() => undefined); };

  return <aside className="webui2-computer-workspace" aria-label="Computer workspace">
    <header className="webui2-computer-header">
      <div><span className="webui2-computer-eyebrow">CONTROL SURFACE</span><h2>Computer Workspace</h2></div>
      <span className="webui2-computer-backend">{computer.capabilities?.backend || "detecting backend"}</span>
    </header>
    <ComputerToolbar
      state={computer.session?.state ?? null}
      capabilities={computer.capabilities}
      available={computer.available}
      busy={computer.loading}
      controlIntent={computer.controlIntent}
      onStart={() => setApprovalOpen(true)}
      onObserve={() => action(computer.observe)}
      onPause={() => action(computer.pause)}
      onResume={() => action(computer.resume)}
      onStop={() => action(computer.stop)}
    />
    {!computer.error && readiness ? <p className="webui2-computer-error" role="status">{readiness}</p> : null}
    {computer.error ? <p className="webui2-computer-error" role="alert">{computer.error}</p> : null}
    <ComputerPreview observation={computer.observation} capabilities={computer.capabilities} />
    <ComputerTimeline receipts={computer.receipts} />
    {approvalOpen ? <ComputerApprovalDialog
      available={computer.available}
      capabilities={computer.capabilities}
      busy={computer.loading}
      onApprove={() => void start()}
      onCancel={() => setApprovalOpen(false)}
    /> : null}
  </aside>;
}
