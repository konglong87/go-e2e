import type { ReactElement } from "react";
import { computerReadinessError } from "./readiness";
import type { ComputerCapabilities, ComputerSessionState } from "./types";

type Props = {
  state: ComputerSessionState | null;
  capabilities: ComputerCapabilities | null;
  available: boolean;
  busy: boolean;
  controlIntent: "pause" | "stop" | null;
  onStart: () => void;
  onObserve: () => void;
  onPause: () => void;
  onResume: () => void;
  onStop: () => void;
};

export function ComputerToolbar({ state, capabilities, available, busy, controlIntent, onStart, onObserve, onPause, onResume, onStop }: Props): ReactElement {
  const active = state !== null && state !== "stopped";
  const readiness = computerReadinessError(available, capabilities);
  const observable = state === "ready" || state === "needs_observation";
  return <div className="webui2-computer-toolbar">
    <span className="webui2-computer-state">{controlIntent ? `${controlIntent} requested` : state || "idle"}</span>
    {!active ? <button className="webui2-computer-button" disabled={busy || Boolean(readiness)} title={readiness ?? undefined} onClick={onStart} type="button">Start session</button> : null}
    {active ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={busy || !observable || Boolean(controlIntent)} onClick={onObserve} type="button">Refresh screenshot</button> : null}
    {observable ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={!capabilities?.supports_pause || controlIntent === "stop"} onClick={onPause} type="button">Pause</button> : null}
    {state === "paused" ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={busy || Boolean(readiness) || controlIntent === "stop"} onClick={onResume} type="button">Resume</button> : null}
    {active ? <button className="webui2-computer-button webui2-computer-button--danger" disabled={!capabilities?.supports_stop} onClick={onStop} type="button">Stop</button> : null}
  </div>;
}
