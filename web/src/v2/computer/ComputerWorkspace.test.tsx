import { act, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ComputerWorkspace } from "./ComputerWorkspace";
import { ComputerApprovalDialog } from "./ComputerApprovalDialog";
import type { ComputerCapabilities, ComputerObservationResponse, ComputerSessionSnapshot } from "./types";
import { capabilities, createTestClient, deferred, observationResponse, snapshot } from "./testFixtures";

describe("ComputerWorkspace", () => {
  let root: Root;
  let container: HTMLDivElement;
  let storage: Record<string, string>;
  beforeEach(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement("div");
    document.body.append(container);
    storage = { "go-e2e.computer-workspace.v1": JSON.stringify({ collapsed: false, position: null }) };
    Object.defineProperty(window, "localStorage", { configurable: true, value: {
      getItem: (key: string) => storage[key] ?? null,
      setItem: (key: string, value: string) => { storage[key] = value; },
      removeItem: (key: string) => { delete storage[key]; },
      clear: () => { storage = {}; },
    } });
    root = createRoot(container);
  });
  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    storage = {};
    delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT;
  });
  function button(label: string): HTMLButtonElement {
    const found = Array.from(document.body.querySelectorAll("button")).find((element) => element.textContent === label);
    if (!found) throw new Error(`Missing button: ${label}`);
    return found;
  }
  async function click(label: string) { await act(async () => { button(label).click(); }); }

  it("has no panel with a non-desktop / unavailable bridge", async () => {
    await act(async () => root.render(<ComputerWorkspace client={null} />));
    expect(container.innerHTML).toBe("");
  });

  it("opens as a compact launcher and expands into a draggable workspace", async () => {
    storage = {};
    const client = createTestClient();
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    expect(document.body.querySelector('[aria-label="Open Computer Use workspace"]')).not.toBeNull();
    await act(async () => { (document.body.querySelector('[aria-label="Open Computer Use workspace"]') as HTMLButtonElement).click(); });
    expect(document.body.querySelector('[aria-label="Computer workspace"]')).not.toBeNull();
    expect(document.body.querySelector('[aria-label="Collapse Computer Use workspace"]')).not.toBeNull();
  });

  it("opens the matching macOS permission page and refreshes readiness after returning", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn()
      .mockResolvedValueOnce({ available: true, capabilities: { ...capabilities, input_readiness: "permission_required", permission_state: "required" } })
      .mockResolvedValue({ available: true, capabilities });
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    expect(document.body.textContent).toContain("Open Accessibility settings");
    await click("Open Accessibility settings");
    expect(client.openPermissionSettings).toHaveBeenCalledWith("accessibility");
    await act(async () => { window.dispatchEvent(new Event("focus")); });
    expect(client.getCapabilities).toHaveBeenCalledTimes(2);
    expect(document.body.textContent).not.toContain("Open Accessibility settings");
    expect(button("Start session").disabled).toBe(false);
  });

  it("handles the actual start → observe DTO under StrictMode without resetting on rerender", async () => {
    const client = createTestClient();
    const render = () => <StrictMode><ComputerWorkspace client={client} /></StrictMode>;
    await act(async () => root.render(render()));
    expect(button("Start session").disabled).toBe(false);
    await click("Start session");
    expect(client.start).not.toHaveBeenCalled();
    await click("Approve session");
    expect(client.start).toHaveBeenCalledWith({ approved: true });
    expect(client.observe).toHaveBeenCalledWith("s1");
    expect(document.body.querySelector("img")?.getAttribute("src")).toBe(`data:image/png;base64,${observationResponse.image_data}`);
    expect(document.body.querySelector('[aria-label="Computer Use execution progress"]')).not.toBeNull();
    await act(async () => root.render(render()));
    expect(button("Stop").disabled).toBe(false);
    expect(client.start).toHaveBeenCalledTimes(1);
    expect(document.body.querySelector("img")).not.toBeNull();
  });

  it("keeps Pause and Stop clickable during capture and ignores late control/capture responses", async () => {
    const client = createTestClient();
    const capture = deferred<ComputerObservationResponse>();
    const pause = deferred<ComputerSessionSnapshot>();
    client.observe = vi.fn().mockReturnValue(capture.promise);
    client.pause = vi.fn().mockReturnValue(pause.promise);
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    await click("Start session");
    await click("Approve session");
    expect(button("Refresh screenshot").disabled).toBe(true);
    expect(button("Pause").disabled).toBe(false);
    expect(button("Stop").disabled).toBe(false);
    await click("Pause");
    expect(client.pause).toHaveBeenCalledWith("s1");
    expect(button("Stop").disabled).toBe(false);
    await click("Stop");
    expect(client.stop).toHaveBeenCalledWith("s1");
    await act(async () => { pause.resolve(snapshot("paused")); capture.resolve(observationResponse); });
    expect(document.body.querySelector(".webui2-computer-state")?.textContent).toBe("stopped");
    expect(document.body.querySelector("img")).toBeNull();
    expect(button("Start session").disabled).toBe(false);
  });

  it("surfaces capability rejection without an unhandled effect promise", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockRejectedValue(new Error("native helper unavailable"));
    await act(async () => root.render(<StrictMode><ComputerWorkspace client={client} /></StrictMode>));
    expect(document.body.querySelector('[role="alert"]')?.textContent).toBe("native helper unavailable");
    expect(button("Start session").disabled).toBe(true);
  });

  it("honors available=false even when capabilities look ready", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockResolvedValue({ available: false, capabilities, error_code: "disabled", error_message: "Computer Use disabled by host" });
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    expect(button("Start session").disabled).toBe(true);
    expect(document.body.querySelector('[role="alert"]')?.textContent).toBe("Computer Use disabled by host");
  });

  const readinessCases: Partial<ComputerCapabilities>[] = [
    { capture_readiness: "permission_required" }, { capture_readiness: "unknown" },
    { input_readiness: "failed" }, { input_readiness: "unavailable" },
    { permission_state: "required" }, { permission_state: "denied" },
    { permission_state: "unknown" }, { image_supported: false }, { supports_stop: false }
  ];
  it.each(readinessCases)("gates both Start and approval for %j", async (overrides) => {
    const blocked = { ...capabilities, ...overrides };
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockResolvedValue({ available: true, capabilities: blocked });
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    expect(button("Start session").disabled).toBe(true);
    expect(document.body.querySelector('[role="status"]')?.textContent).toBeTruthy();
    const approve = vi.fn();
    await act(async () => root.render(<ComputerApprovalDialog capabilities={blocked} available busy={false} onApprove={approve} onCancel={vi.fn()} />));
    expect(button("Approve session").disabled).toBe(true);
    await click("Approve session");
    expect(approve).not.toHaveBeenCalled();
  });
});
