export type DesktopServiceStatus = {
  state: "stopped" | "starting" | "ready" | "failed";
  pid?: number;
  port: number;
  error?: string;
};

export type DesktopServiceBridge = {
  RestartLocalService: () => Promise<void>;
  GetLocalServiceStatus?: () => Promise<DesktopServiceStatus>;
  SelectWorkspace?: () => Promise<string>;
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
