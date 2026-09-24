import { afterEach, describe, expect, it, vi } from "vitest";
import { createComputerClient, getComputerBridge, type ComputerBridge } from "./client";
import { capabilities, observationResponse, snapshot } from "./testFixtures";

const bridge = (): ComputerBridge => ({
  GetComputerCapabilities: vi.fn<ComputerBridge["GetComputerCapabilities"]>().mockResolvedValue({ available: true, capabilities }),
  StartComputerSession: vi.fn<ComputerBridge["StartComputerSession"]>().mockResolvedValue(snapshot()),
  ObserveComputerSession: vi.fn<ComputerBridge["ObserveComputerSession"]>().mockResolvedValue(observationResponse),
  PauseComputerSession: vi.fn<ComputerBridge["PauseComputerSession"]>().mockResolvedValue(snapshot("paused")),
  ResumeComputerSession: vi.fn<ComputerBridge["ResumeComputerSession"]>().mockResolvedValue(snapshot()),
  StopComputerSession: vi.fn<ComputerBridge["StopComputerSession"]>().mockResolvedValue(snapshot("stopped")),
  GetComputerActionReceipt: vi.fn<ComputerBridge["GetComputerActionReceipt"]>()
});

describe("computer bridge DTO contract", () => {
  afterEach(() => { delete window.go; });
  it("preserves exact Go envelopes and control snapshots", async () => {
    const host = bridge();
    const client = createComputerClient(host);
    await expect(client.getCapabilities()).resolves.toEqual({ available: true, capabilities });
    await expect(client.start({ approved: true })).resolves.toEqual(snapshot());
    expect(host.StartComputerSession).toHaveBeenCalledWith({ approved: true });
    await expect(client.observe("s1")).resolves.toEqual(observationResponse);
    await expect(client.pause("s1")).resolves.toEqual(snapshot("paused"));
    await expect(client.resume("s1")).resolves.toEqual(snapshot());
    await expect(client.stop("s1")).resolves.toEqual(snapshot("stopped"));
    expect(host.StopComputerSession).toHaveBeenCalledWith("s1");
  });
  it("does not discard available=false or backend error details", async () => {
    const host = bridge();
    const response = { available: false, capabilities, error_code: "capability_unavailable", error_message: "helper missing" };
    host.GetComputerCapabilities = vi.fn().mockResolvedValue(response);
    await expect(createComputerClient(host).getCapabilities()).resolves.toEqual(response);
  });
  it("rejects missing and partial browser bridges", () => {
    expect(getComputerBridge()).toBeNull();
    window.go = { main: { app: { RestartLocalService: vi.fn(), GetComputerCapabilities: bridge().GetComputerCapabilities } } };
    expect(getComputerBridge()).toBeNull();
    window.go.main!.app = { RestartLocalService: vi.fn(), ...bridge() };
    expect(getComputerBridge()).not.toBeNull();
  });
});
