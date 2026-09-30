import type { ReactElement } from "react";
import { useI18n } from "../../lib/i18n";
import { computerReadinessMessage, computerUICopy, preferredComputerLanguage } from "./computerUICopy";
import type { ComputerCapabilities, ComputerSessionState } from "./types";

type Props = {
  state: ComputerSessionState | null;
  capabilities: ComputerCapabilities | null;
  available: boolean;
  busy: boolean;
  controlIntent: "pause" | "stop" | null;
  readOnly?: boolean;
  onStart: () => void;
  onObserve: () => void;
  onPause: () => void;
  onResume: () => void;
  onStop: () => void;
};

export function ComputerToolbar({ state, capabilities, available, busy, controlIntent, readOnly = false, onStart, onObserve, onPause, onResume, onStop }: Props): ReactElement {
  useI18n(); // Subscribe to the app language so a settings change rerenders this surface.
  const language = preferredComputerLanguage();
  const copy = computerUICopy[language];
  const active = state !== null && state !== "stopped";
  const readiness = computerReadinessMessage(available, capabilities, language);
  const observable = state === "ready" || state === "needs_observation";
  return <div className="webui2-computer-toolbar">
    <span className="webui2-computer-state">{controlIntent ? `${controlIntent === "pause" ? copy.pause : copy.stop} ${copy.requested}` : state ? copy.session[state] : copy.idle}</span>
    {!active ? <button className="webui2-computer-button" disabled={busy || Boolean(readiness)} title={readiness ?? undefined} onClick={onStart} type="button">{copy.startSession}</button> : null}
    {active && !readOnly ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={busy || !observable || Boolean(controlIntent)} onClick={onObserve} type="button">{copy.refreshScreenshot}</button> : null}
    {observable && !readOnly ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={!capabilities?.supports_pause || controlIntent === "stop"} onClick={onPause} type="button">{copy.pause}</button> : null}
    {state === "paused" && !readOnly ? <button className="webui2-computer-button webui2-computer-button--quiet" disabled={busy || Boolean(readiness) || controlIntent === "stop"} onClick={onResume} type="button">{copy.resume}</button> : null}
    {active ? <button className="webui2-computer-button webui2-computer-button--danger" disabled={!capabilities?.supports_stop} onClick={onStop} type="button">{copy.stop}</button> : null}
  </div>;
}
