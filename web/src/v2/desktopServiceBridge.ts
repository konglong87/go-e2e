export type DesktopServiceStatus = {
  state: "stopped" | "starting" | "ready" | "failed";
  pid?: number;
  port: number;
  error?: string;
};

export type DesktopBackgroundMode = "external" | "local";

export type DesktopBackgroundImage = {
  mode: DesktopBackgroundMode;
  enabled: boolean;
  name?: string;
  mime_type?: string;
  data_url?: string;
};

export type DesktopServiceBridge = {
  RestartLocalService: () => Promise<void>;
  GetLocalServiceStatus?: () => Promise<DesktopServiceStatus>;
  SelectWorkspace?: () => Promise<string>;
  GetSessionBackend?: () => Promise<"jsonl" | "sqlite">;
  SetSessionBackend?: (backend: "jsonl" | "sqlite") => Promise<void>;
  GetBackgroundImage?: () => Promise<DesktopBackgroundImage>;
  SaveBackgroundImage?: (dataURL: string, name: string) => Promise<DesktopBackgroundImage>;
  SetBackgroundMode?: (mode: DesktopBackgroundMode) => Promise<DesktopBackgroundImage>;
  ClearBackgroundImage?: () => Promise<void>;
  GetComputerCapabilities?: () => Promise<unknown>;
  StartComputerSession?: (input: unknown) => Promise<unknown>;
  ObserveComputerSession?: (sessionID: string) => Promise<unknown>;
  PauseComputerSession?: (sessionID: string) => Promise<unknown>;
  ResumeComputerSession?: (sessionID: string) => Promise<unknown>;
  StopComputerSession?: (sessionID: string) => Promise<unknown>;
  GetComputerActionReceipt?: (sessionID: string, actionID: string) => Promise<unknown>;
};

declare global {
  interface Window {
    go?: {
      main?: {
        app?: DesktopServiceBridge;
        App?: DesktopServiceBridge;
      };
    };
  }
}

export function getDesktopServiceBridge(): DesktopServiceBridge | null {
  if (typeof window === "undefined") {
    return null;
  }
  const app = window.go?.main?.app ?? window.go?.main?.App;
  return app?.RestartLocalService ? app : null;
}

// Computer Use methods are optional until the desktop backend is available.
// The Computer Workspace checks the complete capability set before rendering.
