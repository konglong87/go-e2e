import { act, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "../../lib/i18n";
import type { SessionListFilters, SessionRef, SessionSummary } from "../types";
import { SessionSidebar } from "./SessionSidebar";
import { SESSION_REF_MIME_TYPE } from "./sessionContextDrag";

const sessions: SessionSummary[] = [
  { ref: "tenant:alpha", source: "tenant", title: "Release coordination", status: "running", updatedAt: "2026-09-05T00:00:00.000Z", shortID: "alpha" },
  { ref: "tenant:beta", source: "tenant", title: "Design review", status: "completed", updatedAt: "2026-09-04T00:00:00.000Z", shortID: "beta" },
  { ref: "local:workspace", source: "local", title: "Local workspace", status: "idle", updatedAt: "2026-09-03T00:00:00.000Z", shortID: "workspace" }
];

describe("SessionSidebar", () => {
  let host: HTMLDivElement;
  let root: Root;
  let filters: SessionListFilters;
  let selected: SessionRef | null;
  let storage: Map<string, string>;
  let select = vi.fn<(ref: SessionRef) => void>();
  let drag = vi.fn<(ref: SessionRef) => void>();

  beforeEach(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    host = document.createElement("div");
    document.body.replaceChildren(host);
    root = createRoot(host);
    storage = new Map();
    Object.defineProperty(window, "localStorage", { configurable: true, value: { getItem: (key: string) => storage.get(key) ?? null, setItem: (key: string, value: string) => storage.set(key, value), removeItem: (key: string) => storage.delete(key), clear: () => storage.clear() } });
    filters = { query: "", statuses: [] };
    selected = "tenant:alpha";
    select = vi.fn<(ref: SessionRef) => void>();
    drag = vi.fn<(ref: SessionRef) => void>();
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
  });

  afterEach(() => {
    act(() => root.unmount());
    delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT;
  });

  function render(items = sessions) {
    act(() => root.render(<I18nProvider><SessionSidebar sessions={items} filters={filters} selectedRef={selected} onCreateSession={vi.fn()} onFiltersChange={(next) => { filters = next; }} onSelect={select} onContextDragStart={drag} onOpenSettings={vi.fn()} onOpenSearch={vi.fn()} onHideSidebar={vi.fn()} /></I18nProvider>));
  }

  function renderControlled() {
    function Harness() {
      const [currentFilters, setCurrentFilters] = useState<SessionListFilters>({ query: "", statuses: [] });
      return <SessionSidebar sessions={sessions} filters={currentFilters} selectedRef={selected} onCreateSession={vi.fn()} onFiltersChange={setCurrentFilters} onSelect={select} onContextDragStart={drag} onOpenSettings={vi.fn()} onOpenSearch={vi.fn()} onHideSidebar={vi.fn()} />;
    }
    act(() => root.render(<I18nProvider><Harness /></I18nProvider>));
  }

  it("groups managed and read-only local sessions with their metadata", () => {
    render();
    expect(host.querySelector('section[aria-label="Managed"]')).not.toBeNull();
    expect(host.textContent).not.toContain("Managed");
    expect(host.textContent).toContain("Local - read only");
    expect(host.textContent).toContain("Release coordination");
    expect(host.textContent).toContain("alpha");
    expect(host.textContent).toContain("running");
    expect(host.querySelector(".webui2-session-status-dot[data-status=\"running\"]")).not.toBeNull();
    expect(host.querySelector("code")?.textContent).toBe("alpha");
    expect(host.querySelector("time")?.getAttribute("dateTime")).toBe("2026-09-05T00:00:00.000Z");
    expect(host.textContent).toContain("Local sessions are read only");
  });

  it("groups by full workspace path and includes that path in search", () => {
    const items = sessions.map((session, index) => ({ ...session, cwd: index === 0 ? "/work/service" : "/other/service" }));
    render(items);
    expect(host.querySelectorAll('.webui2-workspace-toggle[title="/work/service"]')).toHaveLength(1);
    filters = { query: "/work/", statuses: [] };
    render(items);
    expect(host.querySelectorAll(".webui2-session-row")).toHaveLength(1);
    expect(host.textContent).toContain("Release coordination");
  });

  it("expands and collapses workspaces and persists the disclosure state", () => {
    const items = sessions.map((session) => ({ ...session, cwd: "/work/service" }));
    render(items);
    const toggle = host.querySelector<HTMLButtonElement>('.webui2-workspace-toggle[title="/work/service"]');
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
    expect(host.querySelectorAll(".webui2-session-row")).toHaveLength(3);

    act(() => toggle?.click());
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    expect(host.querySelectorAll(".webui2-session-row")).toHaveLength(1);
    expect(window.localStorage.getItem("golang-cc-webui.v2.workspace-collapse.v1")).toContain("tenant:%2Fwork%2Fservice");

    act(() => toggle?.click());
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
    expect(host.querySelectorAll(".webui2-session-row")).toHaveLength(3);
  });

  it("automatically reopens the selected workspace after it was collapsed", () => {
    const items = sessions.map((session, index) => ({ ...session, cwd: index === 0 ? "/work/service" : "/other/service" }));
    render(items);
    const firstToggle = host.querySelector<HTMLButtonElement>('.webui2-workspace-toggle[title="/work/service"]');
    act(() => firstToggle?.click());
    expect(firstToggle?.getAttribute("aria-expanded")).toBe("false");

    selected = "tenant:beta";
    render(items);
    expect(host.querySelector<HTMLButtonElement>('.webui2-workspace-toggle[title="/work/service"]')?.getAttribute("aria-expanded")).toBe("false");
    selected = "tenant:alpha";
    render(items);
    expect(host.querySelector<HTMLButtonElement>('.webui2-workspace-toggle[title="/work/service"]')?.getAttribute("aria-expanded")).toBe("true");
    expect(host.querySelectorAll(".webui2-session-row")).toHaveLength(3);
  });

  it("temporarily hides the retained status filter controls", () => {
    renderControlled();
    const disclosure = host.querySelector<HTMLDetailsElement>(".webui2-status-filter-disclosure");
    const summary = disclosure?.querySelector<HTMLElement>("summary");

    expect(disclosure?.open).toBe(false);
    expect(disclosure?.hidden).toBe(true);
    expect(summary?.getAttribute("aria-label")).toBe("Filter status");
    expect(summary?.textContent).toContain("All statuses");

  });

  it("keeps search as a dedicated toolbar action", () => {
    renderControlled();
    const toggle = host.querySelector<HTMLButtonElement>('button[aria-label="Search sessions"]');
    expect(host.querySelectorAll('button[aria-label="Search sessions"]')).toHaveLength(1);
    expect(host.querySelector(".webui2-sidebar-toolbar")?.contains(toggle)).toBe(true);
    expect(host.querySelector(".webui2-new-session")).not.toBeNull();
    expect(toggle?.textContent).toBe("");
    expect(toggle?.title).toBe("Search sessions");
    expect(host.querySelector('.webui2-sidebar-bottom button[aria-label="Search sessions"]')).toBeNull();
  });

  it("exposes the hidden-sidebar action without changing session filters", () => {
    renderControlled();
    expect(host.querySelector('button[aria-label="Hide sidebar"]')).not.toBeNull();
    expect(host.querySelector('button[aria-label="Search sessions"]')).not.toBeNull();
  });

  it("resizes between 232 and 380 pixels with keyboard controls and persists width", () => {
    render();
    const sidebar = host.querySelector<HTMLElement>(".webui2-sidebar");
    const handle = host.querySelector<HTMLElement>(".webui2-sidebar-resize");

    expect(handle?.getAttribute("aria-valuemin")).toBe("232");
    expect(handle?.getAttribute("aria-valuemax")).toBe("380");
    act(() => handle?.dispatchEvent(new KeyboardEvent("keydown", { key: "End", bubbles: true })));
    expect(sidebar?.style.width).toBe("380px");
    expect(window.localStorage.getItem("golang-cc-webui.v2.sidebar-width.v1")).toBe("380");
    act(() => handle?.dispatchEvent(new KeyboardEvent("keydown", { key: "Home", bubbles: true })));
    expect(sidebar?.style.width).toBe("232px");

    const pointerDown = new Event("pointerdown", { bubbles: true });
    const pointerMove = new Event("pointermove", { bubbles: true });
    Object.defineProperty(pointerDown, "clientX", { value: 200 });
    Object.defineProperty(pointerMove, "clientX", { value: 260 });
    act(() => handle?.dispatchEvent(pointerDown));
    act(() => window.dispatchEvent(pointerMove));
    expect(sidebar?.style.width).toBe("292px");
  });

  it("opens session commands from an accessible overflow menu and closes it naturally", async () => {
    render();
    const trigger = host.querySelector<HTMLButtonElement>('button[aria-label="Actions for tenant:alpha"]');
    expect(trigger?.getAttribute("aria-expanded")).toBe("false");
    act(() => trigger?.click());
    expect(trigger?.getAttribute("aria-expanded")).toBe("true");

    const copy = host.querySelector<HTMLButtonElement>('button[role="menuitem"][aria-label="Copy tenant:alpha"]');
    await act(async () => copy?.click());
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("tenant:alpha");
    expect(host.querySelector('[role="menu"]')).toBeNull();

    act(() => trigger?.click());
    const link = host.querySelector<HTMLAnchorElement>('a[role="menuitem"][aria-label="Open tenant:alpha"]');
    expect(link?.getAttribute("href")).toBe("/webui/v2/sessions/tenant%3Aalpha");
    act(() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    expect(host.querySelector('[role="menu"]')).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("preserves selection and drag behavior while the overflow menu is closed", () => {
    render();
    const row = host.querySelector<HTMLElement>(".webui2-session-row");
    const selectButton = row?.querySelector<HTMLButtonElement>(".webui2-session-select");
    act(() => selectButton?.click());
    expect(select).toHaveBeenCalledWith("tenant:alpha");

    const handle = host.querySelector<HTMLElement>('[aria-label="Drag tenant:alpha as context"]');
    act(() => handle?.dispatchEvent(new Event("dragstart", { bubbles: true })));
    expect(drag).toHaveBeenCalledWith("tenant:alpha");
  });

  it("drags from the session title without selecting it and keeps normal clicks working", () => {
    render();
    const title = host.querySelector<HTMLElement>(".webui2-session-title");
    const button = host.querySelector<HTMLButtonElement>(".webui2-session-select");
    const setData = vi.fn();
    const transfer = { setData, effectAllowed: "none" };
    const event = new Event("dragstart", { bubbles: true });
    Object.defineProperty(event, "dataTransfer", { value: transfer });
    act(() => title?.dispatchEvent(event));
    expect(button?.draggable).toBe(true);
    expect(setData).toHaveBeenCalledWith(SESSION_REF_MIME_TYPE, "tenant:alpha");
    expect(transfer.effectAllowed).toBe("copy");
    act(() => button?.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 })));
    expect(select).not.toHaveBeenCalled();
    act(() => button?.dispatchEvent(new Event("pointerdown", { bubbles: true })));
    act(() => button?.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 })));
    expect(select).toHaveBeenCalledWith("tenant:alpha");
  });
});
