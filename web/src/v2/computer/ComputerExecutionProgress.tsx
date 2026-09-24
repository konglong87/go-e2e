import { Activity, ChevronUp, GripVertical, Minus } from "lucide-react";
import { createPortal } from "react-dom";
import type { ReactElement, PointerEvent as ReactPointerEvent } from "react";
import { useState, useRef } from "react";
import type { ComputerActionReceipt, ComputerSessionState } from "./types";
import { clampComputerWorkspacePosition, type ComputerWorkspacePoint } from "./computerWorkspacePreferences";

const DRAG_THRESHOLD_PX = 6;
const DEFAULT_RIGHT_PX = 22;
const DEFAULT_BOTTOM_PX = 22;
const MAX_VISIBLE_STEPS = 4;

type DragState = {
  pointerId: number;
  startX: number;
  startY: number;
  originX: number;
  originY: number;
  moved: boolean;
};

type ProgressTone = "active" | "ready" | "paused" | "attention";

type ProgressStatus = {
  label: string;
  tone: ProgressTone;
  summary: string;
};

export type ComputerExecutionProgressProps = {
  state: ComputerSessionState;
  receipts: ComputerActionReceipt[];
  loading?: boolean;
  controlIntent?: "pause" | "stop" | null;
  error?: string | null;
  defaultCollapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
};

function portalHost(): HTMLElement {
  return document.querySelector<HTMLElement>(".webui2-page") ?? document.body;
}

function progressStatus(state: ComputerSessionState, loading: boolean, controlIntent: "pause" | "stop" | null, error: string | null): ProgressStatus {
  if (error || state === "failed") return { label: "Needs attention", tone: "attention", summary: error ?? "The Computer Use session failed." };
  if (controlIntent === "pause") return { label: "Pausing", tone: "active", summary: "Pausing the Computer Use session…" };
  if (controlIntent === "stop") return { label: "Stopping", tone: "active", summary: "Stopping the Computer Use session…" };
  if (loading) return { label: "Working", tone: "active", summary: "Running the current Computer Use step…" };
  if (state === "paused") return { label: "Paused", tone: "paused", summary: "The session is paused. Resume from the workspace when ready." };
  if (state === "needs_observation") return { label: "Waiting", tone: "active", summary: "Waiting for the next desktop observation…" };
  if (state === "pending_approval") return { label: "Awaiting approval", tone: "paused", summary: "Approve the session to start Computer Use." };
  return { label: "Ready", tone: "ready", summary: "Computer Use is ready for the next step." };
}

function stepTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function ComputerExecutionProgress({
  state,
  receipts,
  loading = false,
  controlIntent = null,
  error = null,
  defaultCollapsed = false,
  onCollapsedChange,
}: ComputerExecutionProgressProps): ReactElement {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const [position, setPosition] = useState<ComputerWorkspacePoint | null>(null);
  const [dragging, setDragging] = useState(false);
  const panelRef = useRef<HTMLElement>(null);
  const dragRef = useRef<DragState | null>(null);
  const status = progressStatus(state, loading, controlIntent, error);
  const recentReceipts = receipts.slice(-MAX_VISIBLE_STEPS).reverse();
  const completedLabel = `${receipts.length} ${receipts.length === 1 ? "step" : "steps"} completed`;

  const toggleCollapsed = (): void => {
    const next = !collapsed;
    setCollapsed(next);
    onCollapsedChange?.(next);
  };

  const onDragStart = (event: ReactPointerEvent<HTMLButtonElement>): void => {
    if (event.button !== 0 || !panelRef.current) return;
    const rect = panelRef.current.getBoundingClientRect();
    dragRef.current = { pointerId: event.pointerId, startX: event.clientX, startY: event.clientY, originX: rect.left, originY: rect.top, moved: false };
    try { event.currentTarget.setPointerCapture?.(event.pointerId); } catch { /* Pointer capture is unavailable in some test environments. */ }
  };

  const onDragMove = (event: ReactPointerEvent<HTMLButtonElement>): void => {
    const current = dragRef.current;
    if (!current || current.pointerId !== event.pointerId || !panelRef.current) return;
    const dx = event.clientX - current.startX;
    const dy = event.clientY - current.startY;
    if (!current.moved && Math.hypot(dx, dy) < DRAG_THRESHOLD_PX) return;
    if (!current.moved) {
      current.moved = true;
      setDragging(true);
    }
    const rect = panelRef.current.getBoundingClientRect();
    setPosition(clampComputerWorkspacePosition(
      { x: current.originX + dx, y: current.originY + dy },
      { x: window.innerWidth, y: window.innerHeight },
      { x: rect.width, y: rect.height },
    ));
  };

  const onDragEnd = (event: ReactPointerEvent<HTMLButtonElement>): void => {
    const current = dragRef.current;
    if (!current || current.pointerId !== event.pointerId) return;
    dragRef.current = null;
    try { event.currentTarget.releasePointerCapture?.(event.pointerId); } catch { /* Pointer capture is unavailable in some test environments. */ }
    if (!current.moved) return;
    setDragging(false);
    const rect = panelRef.current?.getBoundingClientRect();
    if (!rect) return;
    setPosition(clampComputerWorkspacePosition(
      { x: current.originX + event.clientX - current.startX, y: current.originY + event.clientY - current.startY },
      { x: window.innerWidth, y: window.innerHeight },
      { x: rect.width, y: rect.height },
    ));
  };

  const onDragCancel = (event: ReactPointerEvent<HTMLButtonElement>): void => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    setDragging(false);
  };

  const panelStyle = position ? { left: position.x, top: position.y } : { right: DEFAULT_RIGHT_PX, bottom: DEFAULT_BOTTOM_PX };
  const launcher = <button
    aria-expanded={false}
    aria-label="Open Computer Use execution progress"
    className="webui2-computer-progress-launcher"
    onClick={toggleCollapsed}
    type="button"
  >
    <Activity aria-hidden="true" size={16} />
    <span className={`webui2-computer-progress-dot webui2-computer-progress-dot--${status.tone}`} />
    <span className="webui2-computer-progress-launcher-copy"><strong>Computer Use</strong><span>{completedLabel}</span></span>
    <ChevronUp aria-hidden="true" size={15} />
  </button>;

  const panel = <aside
    ref={panelRef}
    aria-label="Computer Use execution progress"
    className="webui2-computer-progress"
    data-dragging={dragging ? "true" : "false"}
    data-status={status.tone}
    style={panelStyle}
  >
    <header className="webui2-computer-progress-header">
      <button
        aria-label="Drag Computer Use execution progress"
        className="webui2-computer-progress-drag-handle"
        data-drag-handle="execution-progress"
        onPointerCancel={onDragCancel}
        onPointerDown={onDragStart}
        onPointerMove={onDragMove}
        onPointerUp={onDragEnd}
        type="button"
      >
        <GripVertical aria-hidden="true" size={16} />
        <span className="webui2-computer-progress-title"><span className="webui2-computer-eyebrow">LIVE EXECUTION</span><strong>Computer Use progress</strong></span>
      </button>
      <div className="webui2-computer-progress-header-actions">
        <span className={`webui2-computer-progress-status webui2-computer-progress-status--${status.tone}`}>{status.label}</span>
        <button aria-label="Collapse Computer Use execution progress" className="webui2-computer-icon-button" onClick={toggleCollapsed} title="Collapse" type="button"><Minus aria-hidden="true" size={16} /></button>
      </div>
    </header>
    <div className="webui2-computer-progress-summary" aria-live="polite">
      <div className="webui2-computer-progress-summary-row"><strong>{completedLabel}</strong><span>{status.label}</span></div>
      <p>{status.summary}</p>
    </div>
    <section aria-label="Computer Use execution steps" className="webui2-computer-progress-steps">
      <div className="webui2-computer-section-heading"><div><span className="webui2-computer-eyebrow">STEP SUMMARY</span><h3>Recent activity</h3></div><span className="webui2-computer-count">{receipts.length}</span></div>
      {recentReceipts.length === 0 ? <p className="webui2-computer-empty">No action steps have been recorded yet.</p> : <ol>
        {recentReceipts.map((receipt) => <li key={receipt.action_id}>
          <span className={`webui2-computer-progress-step-dot webui2-computer-progress-step-dot--${receipt.outcome}`} />
          <div><strong>{receipt.redacted_action_summary || "Computer action"}</strong><span>{receipt.outcome} · {receipt.verification}</span></div>
          <time dateTime={receipt.completed_at}>{stepTime(receipt.completed_at)}</time>
        </li>)}
      </ol>}
    </section>
  </aside>;

  return createPortal(collapsed ? launcher : panel, portalHost());
}
