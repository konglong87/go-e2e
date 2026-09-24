import { describe, expect, it, vi } from "vitest";
import { createComputerClient } from "./client";
import type { ComputerBridge } from "./client";

const capabilities = { protocol_version: "computer-use.v1", platform: "macos", backend: "native_host", capture_readiness: "ready", input_readiness: "ready", focus_state: "focused", permission_state: "approved", coordinate_space: { origin: "top_left", unit: "pixels", width: 1, height: 1, scale_factor: 1 }, image_supported: true, supports_pause: true, supports_stop: true } as const;

describe("computer bridge client", () => {
  it("maps the Wails control surface without platform-specific APIs", async () => {
    const bridge: ComputerBridge = { GetComputerCapabilities: vi.fn().mockResolvedValue(capabilities), StartComputerSession: vi.fn().mockResolvedValue({ session_id: "s1", state: "ready", capabilities }), ObserveComputerSession: vi.fn(), PauseComputerSession: vi.fn(), ResumeComputerSession: vi.fn(), StopComputerSession: vi.fn(), GetComputerActionReceipt: vi.fn() };
    const client = createComputerClient(bridge);
    await expect(client.getCapabilities()).resolves.toEqual(capabilities);
    await client.start({ approved: true });
    expect(bridge.StartComputerSession).toHaveBeenCalledWith({ approved: true });
  });
});
