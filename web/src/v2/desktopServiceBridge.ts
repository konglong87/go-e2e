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
