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
  beforeEach(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });
  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT;
  });
  function button(label: string): HTMLButtonElement {
    const found = Array.from(container.querySelectorAll("button")).find((element) => element.textContent === label);
    if (!found) throw new Error(`Missing button: ${label}`);
    return found;
  }
  async function click(label: string) { await act(async () => { button(label).click(); }); }

  it("has no panel with a non-desktop / unavailable bridge", async () => {
    await act(async () => root.render(<ComputerWorkspace client={null} />));
    expect(container.innerHTML).toBe("");
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
    expect(container.querySelector("img")?.getAttribute("src")).toBe(`data:image/png;base64,${observationResponse.image_data}`);
    await act(async () => root.render(render()));
    expect(button("Stop").disabled).toBe(false);
    expect(client.start).toHaveBeenCalledTimes(1);
    expect(container.querySelector("img")).not.toBeNull();
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
    expect(container.querySelector(".webui2-computer-state")?.textContent).toBe("stopped");
    expect(container.querySelector("img")).toBeNull();
    expect(button("Start session").disabled).toBe(false);
  });

  it("surfaces capability rejection without an unhandled effect promise", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockRejectedValue(new Error("native helper unavailable"));
    await act(async () => root.render(<StrictMode><ComputerWorkspace client={client} /></StrictMode>));
    expect(container.querySelector('[role="alert"]')?.textContent).toBe("native helper unavailable");
    expect(button("Start session").disabled).toBe(true);
  });

  it("honors available=false even when capabilities look ready", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockResolvedValue({ available: false, capabilities, error_code: "disabled", error_message: "Computer Use disabled by host" });
    await act(async () => root.render(<ComputerWorkspace client={client} />));
    expect(button("Start session").disabled).toBe(true);
    expect(container.querySelector('[role="alert"]')?.textContent).toBe("Computer Use disabled by host");
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
    expect(container.querySelector('[role="status"]')?.textContent).toBeTruthy();
    const approve = vi.fn();
    await act(async () => root.render(<ComputerApprovalDialog capabilities={blocked} available busy={false} onApprove={approve} onCancel={vi.fn()} />));
    expect(button("Approve session").disabled).toBe(true);
    await click("Approve session");
    expect(approve).not.toHaveBeenCalled();
  });
});
