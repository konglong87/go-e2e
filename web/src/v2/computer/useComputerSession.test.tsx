import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useComputerSession } from "./useComputerSession";
import type { ComputerClient } from "./client";
import type { ComputerCapabilities } from "./types";

const capabilities: ComputerCapabilities = { protocol_version: "computer-use.v1", platform: "macos", backend: "native_host", capture_readiness: "ready", input_readiness: "ready", focus_state: "focused", permission_state: "approved", coordinate_space: { origin: "top_left", unit: "pixels", width: 100, height: 100, scale_factor: 1 }, image_supported: true, supports_pause: true, supports_stop: true };

describe("useComputerSession", () => {
  let root: Root;
  let result!: ReturnType<typeof useComputerSession>;
  let client: ComputerClient;
  beforeEach(() => { (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true; root = createRoot(document.createElement("div")); client = { getCapabilities: vi.fn().mockResolvedValue(capabilities), start: vi.fn().mockResolvedValue({ session_id: "s1", state: "ready", capabilities }), observe: vi.fn().mockResolvedValue({ id: "o1", session_id: "s1", width: 100, height: 100, scale_factor: 1, screenshot: {}, cursor: { x: 1, y: 1 }, capabilities, observed_at: "2026-09-24T00:00:00Z" }), pause: vi.fn().mockResolvedValue({ session_id: "s1", state: "paused", capabilities }), resume: vi.fn(), stop: vi.fn().mockResolvedValue({ session_id: "s1", state: "stopped", capabilities }), getReceipt: vi.fn() }; });
  afterEach(() => { act(() => root.unmount()); delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT; });
  function Harness() { result = useComputerSession(client); return null; }
  it("loads capabilities and keeps an observation after start", async () => { await act(async () => root.render(<Harness />)); await act(async () => { await result.loadCapabilities(); }); await act(async () => { await result.start(); }); await act(async () => { await result.observe(); }); expect(result.capabilities?.backend).toBe("native_host"); expect(result.session?.session_id).toBe("s1"); expect(result.observation?.id).toBe("o1"); });
  it("does not let a late pause response overwrite a newer stop", async () => { let resolvePause!: (value: unknown) => void; client.pause = vi.fn().mockReturnValue(new Promise((resolve) => { resolvePause = resolve; })); await act(async () => root.render(<Harness />)); await act(async () => result.start()); let pause!: Promise<unknown>; await act(async () => { pause = result.pause(); await Promise.resolve(); }); await act(async () => { await result.stop(); }); await act(async () => { resolvePause({ session_id: "s1", state: "paused", capabilities }); await pause; }); expect(result.session?.state).toBe("stopped"); });
});
