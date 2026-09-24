import { getDesktopServiceBridge, type DesktopServiceBridge } from "../desktopServiceBridge";
import type { ComputerActionReceipt, ComputerCapabilities, ComputerCapabilitiesResponse, ComputerObservation, ComputerObservationResponse, ComputerSessionSnapshot, StartComputerSessionInput } from "./types";

export type ComputerClient = {
  getCapabilities: () => Promise<ComputerCapabilities>;
  start: (input: StartComputerSessionInput) => Promise<ComputerSessionSnapshot>;
  observe: (sessionID: string) => Promise<ComputerObservationResponse | ComputerSessionSnapshot>;
  pause: (sessionID: string) => Promise<ComputerSessionSnapshot | void>;
  resume: (sessionID: string) => Promise<ComputerSessionSnapshot | void>;
  stop: (sessionID: string) => Promise<ComputerSessionSnapshot | void>;
  getReceipt: (sessionID: string, actionID: string) => Promise<ComputerActionReceipt>;
};

export type ComputerBridge = Pick<Required<DesktopServiceBridge>, "GetComputerCapabilities" | "StartComputerSession" | "ObserveComputerSession" | "PauseComputerSession" | "ResumeComputerSession" | "StopComputerSession" | "GetComputerActionReceipt">;

export function getComputerBridge(): ComputerBridge | null {
  const bridge = getDesktopServiceBridge();
  if (!bridge?.GetComputerCapabilities || !bridge.StartComputerSession || !bridge.ObserveComputerSession || !bridge.PauseComputerSession || !bridge.ResumeComputerSession || !bridge.StopComputerSession || !bridge.GetComputerActionReceipt) return null;
  return bridge as unknown as ComputerBridge;
}

export function createComputerClient(bridge: ComputerBridge): ComputerClient {
  return {
    getCapabilities: async () => {
      const value = await bridge.GetComputerCapabilities!() as ComputerCapabilitiesResponse | ComputerCapabilities;
      return ("capabilities" in value ? value.capabilities : value);
    },
    start: async (input) => await bridge.StartComputerSession!(input) as ComputerSessionSnapshot,
    observe: async (sessionID) => await bridge.ObserveComputerSession!(sessionID) as ComputerObservationResponse | ComputerSessionSnapshot,
    pause: async (sessionID) => await bridge.PauseComputerSession!(sessionID) as ComputerSessionSnapshot | void,
    resume: async (sessionID) => await bridge.ResumeComputerSession!(sessionID) as ComputerSessionSnapshot | void,
    stop: async (sessionID) => await bridge.StopComputerSession!(sessionID) as ComputerSessionSnapshot | void,
    getReceipt: async (sessionID, actionID) => await bridge.GetComputerActionReceipt!(sessionID, actionID) as ComputerActionReceipt
  };
}
