import type { ReactElement } from "react";
import type { ComputerCapabilities, ComputerObservation } from "./types";

export function ComputerPreview({ observation, capabilities }: { observation: ComputerObservation | null; capabilities: ComputerCapabilities | null }): ReactElement {
  const screenshot = observation?.screenshot;
  const source = observation?.image_data ? `data:${observation.media_type || "image/png"};base64,${observation.image_data}` : screenshot?.url;
  return <section className="webui2-computer-preview" aria-label="Computer preview">
    <div className="webui2-computer-section-heading"><div><span className="webui2-computer-eyebrow">LIVE DESKTOP</span><h3>Computer Preview</h3></div><span className={`webui2-computer-status webui2-computer-status--${capabilities?.capture_readiness ?? "unknown"}`}>{capabilities?.capture_readiness ?? "not ready"}</span></div>
    {source ? <img className="webui2-computer-screenshot" src={source} alt={`Desktop observation ${observation?.id ?? ""}`} /> : <div className="webui2-computer-preview-empty"><strong>{observation ? "Screenshot reference received" : "No observation yet"}</strong><span>{observation ? screenshot?.id || "The backend did not provide a display URL." : "Start an approved session to capture the desktop."}</span></div>}
    {observation ? <div className="webui2-computer-preview-meta"><span>{observation.width} × {observation.height}</span><span>{observation.active_window?.title || capabilities?.target_window?.title || "Desktop"}</span><span>{new Date(observation.observed_at).toLocaleTimeString()}</span></div> : null}
  </section>;
}
