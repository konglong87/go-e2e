import { useState, type ReactElement } from "react";
import type { ComputerClient } from "./client";
import { computerPermissionGuideActions } from "./readiness";
import type { ComputerPermissionTarget, ComputerCapabilities } from "./types";

type Props = {
  capabilities: ComputerCapabilities | null;
  available: boolean;
  client: Pick<ComputerClient, "openPermissionSettings">;
  onRecheck: () => void | Promise<void>;
};

const TITLE_ID = "webui2-computer-permission-guide-title";

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  return "Unable to open Computer Use permission settings.";
}

export function ComputerPermissionGuide({ capabilities, available, client, onRecheck }: Props): ReactElement | null {
  const actions = computerPermissionGuideActions(available, capabilities);
  const [openingTarget, setOpeningTarget] = useState<ComputerPermissionTarget | null>(null);
  const [rechecking, setRechecking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (actions.length === 0) return null;

  const openSettings = async (target: ComputerPermissionTarget) => {
    setError(null);
    setOpeningTarget(target);
    try {
      await client.openPermissionSettings(target);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setOpeningTarget(null);
    }
  };

  const recheck = async () => {
    setError(null);
    setRechecking(true);
    try {
      await onRecheck();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setRechecking(false);
    }
  };

  return <section className="webui2-computer-permission-guide" aria-labelledby={TITLE_ID}>
    <span className="webui2-computer-eyebrow">SYSTEM PERMISSIONS</span>
    <h3 id={TITLE_ID}>Computer Use needs system permissions</h3>
    <p>Open the relevant macOS settings, enable access for Computer Use, then recheck permissions.</p>
    <div className="webui2-computer-permission-actions">
      {actions.map((action) => <div className="webui2-computer-permission-action" key={action.target}>
        <div>
          <strong>{action.label}</strong>
          <span>{action.description}</span>
        </div>
        <button
          className="webui2-computer-button webui2-computer-button--quiet"
          disabled={openingTarget !== null || rechecking}
          onClick={() => void openSettings(action.target)}
          type="button"
        >{openingTarget === action.target ? "Opening settings…" : `Open ${action.label} settings`}</button>
      </div>)}
    </div>
    {error ? <p className="webui2-computer-error" role="alert">{error}</p> : null}
    <button className="webui2-computer-button webui2-computer-button--quiet" disabled={openingTarget !== null || rechecking} onClick={() => void recheck()} type="button">
      {rechecking ? "Checking permissions…" : "Recheck permissions"}
    </button>
  </section>;
}
