import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ComputerExecutionProgress } from "./ComputerExecutionProgress";
import type { ComputerActionReceipt } from "./types";

const receipt = (id: string, summary: string, outcome: ComputerActionReceipt["outcome"] = "executed"): ComputerActionReceipt => ({
  action_id: id,
  session_id: "s1",
  platform: "macos",
  backend: "native_host",
  outcome,
  verification: outcome === "executed" ? "passed" : "failed",
  focus_before: "focused",
  focus_after: "focused",
  redacted_action_summary: summary,
  completed_at: "2026-09-24T08:30:00Z",
});

describe("ComputerExecutionProgress", () => {
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

  function render(props: Partial<Parameters<typeof ComputerExecutionProgress>[0]> = {}): void {
    act(() => root.render(<ComputerExecutionProgress state="ready" receipts={[]} {...props} />));
  }

  it("shows an expanded floating progress panel with status and recent step summaries", () => {
    render({
      loading: true,
      receipts: [receipt("a1", "Open settings"), receipt("a2", "Click permission toggle")],
    });

    expect(document.body.querySelector('[aria-label="Computer Use execution progress"]')).not.toBeNull();
    expect(document.body.textContent).toContain("Working");
    expect(document.body.textContent).toContain("2 steps completed");
    expect(document.body.textContent).toContain("Open settings");
    expect(document.body.textContent).toContain("Click permission toggle");
    expect(document.body.querySelector('[data-drag-handle="execution-progress"]')?.getAttribute("aria-label")).toBe("Drag Computer Use execution progress");
  });

  it("toggles between expanded and collapsed states and reports the change", () => {
    const onCollapsedChange = vi.fn();
    render({ onCollapsedChange });

    const collapse = document.body.querySelector<HTMLButtonElement>('[aria-label="Collapse Computer Use execution progress"]');
    expect(collapse).not.toBeNull();
    act(() => collapse?.click());
    expect(document.body.querySelector('[aria-label="Open Computer Use execution progress"]')).not.toBeNull();
    expect(onCollapsedChange).toHaveBeenCalledWith(true);

    const launcher = document.body.querySelector<HTMLButtonElement>('[aria-label="Open Computer Use execution progress"]');
    act(() => launcher?.click());
    expect(document.body.querySelector('[aria-label="Computer Use execution progress"]')).not.toBeNull();
    expect(onCollapsedChange).toHaveBeenLastCalledWith(false);
  });

  it("keeps the drag handle keyboard reachable and moves the floating panel with pointer input", () => {
    render({ receipts: [receipt("a1", "Move cursor")] });
    const panel = document.body.querySelector<HTMLElement>('[aria-label="Computer Use execution progress"]');
    const handle = document.body.querySelector<HTMLElement>('[data-drag-handle="execution-progress"]');
    expect(panel).not.toBeNull();
    expect(handle?.tagName).toBe("BUTTON");
    expect((handle as HTMLButtonElement | null)?.tabIndex).toBe(0);

    Object.defineProperty(panel, "getBoundingClientRect", {
      configurable: true,
      value: () => ({ left: 100, top: 100, width: 340, height: 220, right: 440, bottom: 320 }),
    });
    const pointer = (type: string, values: Record<string, number>): Event => {
      const event = new Event(type, { bubbles: true });
      for (const [key, value] of Object.entries(values)) Object.defineProperty(event, key, { configurable: true, value });
      return event;
    };

    act(() => handle?.dispatchEvent(pointer("pointerdown", { button: 0, pointerId: 7, clientX: 120, clientY: 120 })));
    act(() => handle?.dispatchEvent(pointer("pointermove", { pointerId: 7, clientX: 180, clientY: 155 })));
    expect(panel?.style.left).toBe("160px");
    expect(panel?.style.top).toBe("135px");
    act(() => handle?.dispatchEvent(pointer("pointerup", { pointerId: 7, clientX: 180, clientY: 155 })));
    expect(panel?.getAttribute("data-dragging")).toBe("false");
  });
});
