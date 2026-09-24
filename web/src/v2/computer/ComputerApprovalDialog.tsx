import type { ReactElement } from "react";
import { computerReadinessError } from "./readiness";
import type { ComputerCapabilities } from "./types";

type Props = {
  capabilities: ComputerCapabilities | null;
  available: boolean;
  busy: boolean;
  onApprove: () => void;
  onCancel: () => void;
};

export function ComputerApprovalDialog({ capabilities, available, busy, onApprove, onCancel }: Props): ReactElement {
  const readiness = computerReadinessError(available, capabilities);
  return <div className="webui2-computer-modal-backdrop" role="presentation">
    <section className="webui2-computer-modal" role="dialog" aria-modal="true" aria-labelledby="webui2-computer-approval-title">
      <span className="webui2-computer-eyebrow">SESSION APPROVAL</span>
      <h2 id="webui2-computer-approval-title">Allow Computer Use?</h2>
      <p>This session can observe and operate the current desktop after you approve it. Do not enter passwords, tokens, or other secrets through Computer Use.</p>
      <dl>
        <div><dt>Platform</dt><dd>{capabilities?.platform || "Unknown"} · {capabilities?.backend || "Unknown backend"}</dd></div>
        <div><dt>Capture</dt><dd>{capabilities?.capture_readiness || "unknown"}</dd></div>
        <div><dt>Input</dt><dd>{capabilities?.input_readiness || "unknown"}</dd></div>
        <div><dt>Target</dt><dd>{capabilities?.target_window?.title || "Current desktop"}</dd></div>
      </dl>
      {readiness ? <p role="status">{readiness}</p> : null}
      <div className="webui2-computer-modal-actions">
        <button className="webui2-computer-button webui2-computer-button--quiet" disabled={busy} onClick={onCancel} type="button">Cancel</button>
        <button className="webui2-computer-button" disabled={busy || Boolean(readiness)} onClick={onApprove} type="button">{busy ? "Starting…" : "Approve session"}</button>
      </div>
    </section>
  </div>;
}
