export type DesktopServiceBridge = {
  RestartLocalService: () => Promise<void>;
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
