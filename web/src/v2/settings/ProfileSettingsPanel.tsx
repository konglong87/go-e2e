import { useState } from "react";
import { AgentProfilesPanel } from "../../components/AgentProfilesPanel";
import type { IdentityConfig } from "../../lib/types";
import "./profile-settings.css";

export type ProfileSettingsPanelProps = {
  identity: IdentityConfig;
  onDirtyChange?: (dirty: boolean) => void;
  onBusyChange?: (busy: boolean) => void;
  isolatedEnvironment?: boolean;
  onOpenSession?: (sessionId: number) => void;
  onDataChanged?: () => void;
};

export function ProfileSettingsPanel({ identity, onDirtyChange, onBusyChange, isolatedEnvironment, onOpenSession, onDataChanged }: ProfileSettingsPanelProps) {
  const [status, setStatus] = useState("");
  return <div className="webui2-profile-settings">
    {status ? <p className="settings-profile-status" role="status">{status}</p> : null}
    <AgentProfilesPanel identity={identity} embedded onStatus={setStatus} onDirtyChange={onDirtyChange} onBusyChange={onBusyChange} isolatedEnvironment={isolatedEnvironment} onOpenSession={onOpenSession} onDataChanged={onDataChanged} />
  </div>;
}
