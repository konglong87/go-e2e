import type { ReactElement } from "react";
import type { ComputerActionReceipt } from "./types";

export function ComputerTimeline({ receipts }: { receipts: ComputerActionReceipt[] }): ReactElement {
  return <section className="webui2-computer-timeline" aria-label="Computer action timeline"><div className="webui2-computer-section-heading"><div><span className="webui2-computer-eyebrow">AUDIT TRAIL</span><h3>Action Timeline</h3></div><span className="webui2-computer-count">{receipts.length}</span></div>{receipts.length === 0 ? <p className="webui2-computer-empty">Actions and before/after receipts will appear here.</p> : <ol>{receipts.slice().reverse().map((receipt) => <li key={receipt.action_id}><span className={`webui2-computer-dot webui2-computer-dot--${receipt.outcome}`} /><div><strong>{receipt.redacted_action_summary || "Computer action"}</strong><span>{receipt.verification} · {receipt.focus_after}</span>{receipt.error_message ? <small role="alert">{receipt.error_message}</small> : null}</div><time>{new Date(receipt.completed_at).toLocaleTimeString()}</time></li>)}</ol>}</section>;
}
