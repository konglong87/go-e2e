import { act, StrictMode } from "react";
import { I18nProvider } from "../../lib/i18n";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SessionRef } from "../routes";
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
    storage = { "go-e2e.computer-workspace.v1": JSON.stringify({ displayMode: "expanded", collapsed: false, position: null }) };
    Object.defineProperty(window, "localStorage", { configurable: true, value: {
      getItem: (key: string) => storage[key] ?? null,
      setItem: (key: string, value: string) => { storage[key] = value; },
      removeItem: (key: string) => { delete storage[key]; },
      clear: () => { storage = {}; },
    } });
    root = createRoot(container);
    storage["golang-cc-webui.language.v1"] = "en";
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
  async function clickAria(label: string) {
    await act(async () => {
      const target = document.body.querySelector<HTMLButtonElement>(`[aria-label="${label}"]`);
      if (!target) throw new Error(`Missing aria button: ${label}`);
      target.click();
    });
  }

  it("has no panel with a non-desktop / unavailable bridge", async () => {
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={null} /></I18nProvider>));
    expect(container.innerHTML).toBe("");
  });

  it("opens as a compact launcher and expands into a draggable workspace", async () => {
    storage = { "golang-cc-webui.language.v1": "en" };
    const client = createTestClient();
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
    expect(document.body.querySelector('[aria-label="Open Computer Use workspace"]')).not.toBeNull();
    await act(async () => { (document.body.querySelector('[aria-label="Open Computer Use workspace"]') as HTMLButtonElement).click(); });
    expect(document.body.querySelector('[aria-label="Computer workspace"]')).not.toBeNull();
    expect(document.body.querySelector('[aria-label="Collapse Computer Use workspace"]')).not.toBeNull();
  });

  it("auto mode keeps idle compact, expands for a session, and collapses after Stop", async () => {
    storage = { "go-e2e.computer-workspace.v1": JSON.stringify({ displayMode: "auto", collapsed: false, position: null }), "golang-cc-webui.language.v1": "en" };
    const client = createTestClient();
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
    expect(document.body.querySelector('[aria-label="Open Computer Use workspace"]')).not.toBeNull();
    expect(document.body.querySelector('[aria-label="Computer workspace"]')).toBeNull();

    await clickAria("Open Computer Use workspace");
    await click("Start session");
    await click("Approve session");
    expect(document.body.querySelector('[aria-label="Computer workspace"]')).not.toBeNull();

    await click("Stop");
    expect(document.body.querySelector('[aria-label="Computer workspace"]')).toBeNull();
    expect(document.body.querySelector('[aria-label="Open Computer Use workspace"]')).not.toBeNull();
  });

  it.each(["compact", "expanded"] as const)("honors the %s display mode while preserving manual toggles", async (mode) => {
    storage = { "go-e2e.computer-workspace.v1": JSON.stringify({ displayMode: mode, collapsed: mode === "compact", position: null }), "golang-cc-webui.language.v1": "en" };
    const client = createTestClient();
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
    const initiallyExpanded = mode === "expanded";
    expect(document.body.querySelector('[aria-label="Computer workspace"]') !== null).toBe(initiallyExpanded);

    const toggleLabel = initiallyExpanded ? "Collapse Computer Use workspace" : "Open Computer Use workspace";
    await clickAria(toggleLabel);
    expect(document.body.querySelector('[aria-label="Computer workspace"]') !== null).toBe(!initiallyExpanded);
  });

  it("opens the matching macOS permission page and refreshes readiness after returning", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn()
      .mockResolvedValueOnce({ available: true, capabilities: { ...capabilities, input_readiness: "permission_required", permission_state: "required" } })
      .mockResolvedValue({ available: true, capabilities });
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
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
    const render = () => <I18nProvider><StrictMode><ComputerWorkspace client={client} /></StrictMode></I18nProvider>;
    await act(async () => root.render(render()));
    expect(button("Start session").disabled).toBe(false);
    await click("Start session");
    expect(client.start).not.toHaveBeenCalled();
    await click("Approve session");
    expect(client.start).toHaveBeenCalledWith({ approved: true });
    expect(client.observe).toHaveBeenCalledWith("s1");
    expect(document.body.querySelector("img")?.getAttribute("src")).toBe(`data:image/png;base64,${observationResponse.image_data}`);
    expect(document.body.querySelector('[aria-label="Computer Use progress"]')).not.toBeNull();
    await act(async () => root.render(render()));
    expect(button("Stop").disabled).toBe(false);
    expect(client.start).toHaveBeenCalledTimes(1);
    expect(document.body.querySelector("img")).not.toBeNull();
  });

  it("captures the displayed managed conversation even if selection changes before approval", async () => {
    const client = createTestClient();
    const render = (ref: SessionRef | null) => root.render(<I18nProvider><ComputerWorkspace client={client} selectedConversationRef={ref} /></I18nProvider>);
    await act(async () => render("tenant:alpha"));
    await click("Start session");
    const dialog = () => document.body.querySelector('[role="dialog"]');
    expect(dialog()?.textContent).toContain("tenant:alpha");
    expect(dialog()?.textContent).toContain("access to observe and operate your desktop");
    expect(dialog()?.textContent).toContain("does not confirm that autonomous agent execution is connected");
    await act(async () => render("tenant:beta"));
    expect(dialog()?.textContent).toContain("tenant:alpha");
    expect(dialog()?.textContent).not.toContain("tenant:beta");
    await click("Approve session");
    expect(client.start).toHaveBeenCalledExactlyOnceWith({ approved: true, conversation_ref: "tenant:alpha" });
    expect(document.body.textContent).toContain("Desktop access granted to conversation: tenant:alpha");
    expect(document.body.textContent).not.toContain("tenant:beta");
    await click("Pause");
    await act(async () => render(null));
    expect(document.body.textContent).toContain("Desktop access granted to conversation: tenant:alpha");
    expect(button("Stop").disabled).toBe(false);
    await click("Resume");
    expect(document.body.textContent).toContain("Desktop access granted to conversation: tenant:alpha");
    await click("Stop");
    expect(client.stop).toHaveBeenCalledWith("s1");
    expect(document.body.textContent).not.toContain("Desktop access granted to conversation");
  });

  it("retains the approved ref while Start is pending and when collapsed", async () => {
    const client = createTestClient();
    const pending = deferred<ComputerSessionSnapshot>();
    client.start = vi.fn().mockReturnValue(pending.promise);
    const render = (ref: SessionRef) => root.render(<I18nProvider><ComputerWorkspace client={client} selectedConversationRef={ref} /></I18nProvider>);
    await act(async () => render("tenant:alpha"));
    await click("Start session");
    await click("Approve session");
    await act(async () => render("tenant:beta"));
    expect(document.body.textContent).not.toContain("Desktop access granted to conversation");
    await act(async () => pending.resolve(snapshot()));
    expect(document.body.textContent).toContain("Desktop access granted to conversation: tenant:alpha");
    await act(async () => { (document.body.querySelector('[aria-label="Collapse Computer Use workspace"]') as HTMLButtonElement).click(); });
    expect(document.body.querySelector('[aria-label="Open Computer Use workspace"]')?.textContent).toContain("tenant:alpha");
    expect(document.body.textContent).not.toContain("tenant:beta");
  });

  it.each<SessionRef | null>([null, "local:workspace", "tenant:", "tenant:bad/ref", "tenant:bad ref", "tenant:bad:key"])("keeps %s explicitly local and never upgrades an open preview approval", async (ref) => {
    const client = createTestClient();
    const render = (selectedConversationRef: SessionRef | null) => root.render(<I18nProvider><ComputerWorkspace client={client} selectedConversationRef={selectedConversationRef} /></I18nProvider>);
    await act(async () => render(ref));
    await click("Start session");
    expect(document.body.querySelector('[role="dialog"]')?.textContent).toContain("Allow local desktop preview?");
    expect(document.body.querySelector('[role="dialog"]')?.textContent).toContain("does NOT allow any agent or conversation");
    await act(async () => render("tenant:alpha"));
    expect(document.body.querySelector('[role="dialog"]')?.textContent).not.toContain("tenant:alpha");
    await click("Approve session");
    expect(client.start).toHaveBeenCalledExactlyOnceWith({ approved: true });
    expect(document.body.textContent).toContain("Local preview only — no agent access");
    expect(document.body.textContent).not.toContain("Desktop access granted to conversation");
    expect(button("Stop").disabled).toBe(false);
  });

  it("uses a fresh approval after cancellation and Stop, including managed channel refs", async () => {
    const client = createTestClient();
    const render = (ref: SessionRef) => root.render(<I18nProvider><ComputerWorkspace client={client} selectedConversationRef={ref} /></I18nProvider>);
    await act(async () => render("tenant:alpha"));
    await click("Start session");
    await click("Cancel");
    expect(client.start).not.toHaveBeenCalled();
    await act(async () => render("tenant:channel:beta"));
    await click("Start session");
    expect(document.body.querySelector('[role="dialog"]')?.textContent).toContain("tenant:channel:beta");
    await click("Approve session");
    expect(client.start).toHaveBeenLastCalledWith({ approved: true, conversation_ref: "tenant:channel:beta" });
    await click("Stop");
    await act(async () => render("local:workspace"));
    await click("Start session");
    await click("Approve session");
    expect(client.start).toHaveBeenLastCalledWith({ approved: true });
    expect(document.body.textContent).toContain("Local preview only — no agent access");
    expect(document.body.textContent).not.toContain("tenant:channel:beta");
  });

  it("does not claim a grant or silently fall back to preview after a rejected Start", async () => {
    const client = createTestClient();
    client.start = vi.fn().mockRejectedValue(new Error("conversation cannot authorize computer use"));
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} selectedConversationRef="tenant:alpha" /></I18nProvider>));
    await click("Start session");
    await click("Approve session");
    expect(client.start).toHaveBeenCalledExactlyOnceWith({ approved: true, conversation_ref: "tenant:alpha" });
    expect(client.observe).not.toHaveBeenCalled();
    expect(document.body.querySelector('[role="alert"]')?.textContent).toContain("conversation cannot authorize computer use");
    expect(document.body.textContent).not.toContain("Desktop access granted to conversation");
    expect(button("Start session").disabled).toBe(false);
  });

  it("keeps Pause and Stop clickable during capture and ignores late control/capture responses", async () => {
    const client = createTestClient();
    const capture = deferred<ComputerObservationResponse>();
    const pause = deferred<ComputerSessionSnapshot>();
    client.observe = vi.fn().mockReturnValue(capture.promise);
    client.pause = vi.fn().mockReturnValue(pause.promise);
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
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
    await act(async () => root.render(<I18nProvider><StrictMode><ComputerWorkspace client={client} /></StrictMode></I18nProvider>));
    expect(document.body.querySelector('[role="alert"]')?.textContent).toBe("native helper unavailable");
    expect(button("Start session").disabled).toBe(true);
  });

  it("honors available=false even when capabilities look ready", async () => {
    const client = createTestClient();
    client.getCapabilities = vi.fn().mockResolvedValue({ available: false, capabilities, error_code: "disabled", error_message: "Computer Use disabled by host" });
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
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
    await act(async () => root.render(<I18nProvider><ComputerWorkspace client={client} /></I18nProvider>));
    expect(button("Start session").disabled).toBe(true);
    expect(document.body.querySelector('[role="status"]')?.textContent).toBeTruthy();
    const approve = vi.fn();
    await act(async () => root.render(<I18nProvider><ComputerApprovalDialog capabilities={blocked} available busy={false} onApprove={approve} onCancel={vi.fn()} /></I18nProvider>));
    expect(button("Approve session").disabled).toBe(true);
    await click("Approve session");
    expect(approve).not.toHaveBeenCalled();
  });
});
