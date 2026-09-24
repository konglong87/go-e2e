import { ChevronDown, GripVertical, Monitor, Minus } from "lucide-react";
import { createPortal } from "react-dom";
import type { ReactElement, PointerEvent as ReactPointerEvent } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { ComputerApprovalDialog } from "./ComputerApprovalDialog";
import { ComputerExecutionProgress } from "./ComputerExecutionProgress";
import { ComputerPermissionGuide } from "./ComputerPermissionGuide";
import type { ComputerClient } from "./client";
import { ComputerPreview } from "./ComputerPreview";
import { ComputerTimeline } from "./ComputerTimeline";
import { ComputerToolbar } from "./ComputerToolbar";
import { clampComputerWorkspacePosition, loadComputerWorkspacePreferences, saveComputerWorkspacePreferences, type ComputerWorkspacePoint } from "./computerWorkspacePreferences";
import { computerReadinessError } from "./readiness";
import { useComputerSession } from "./useComputerSession";

const DRAG_THRESHOLD_PX = 6;
const DEFAULT_TOP_PX = 76;
const DEFAULT_RIGHT_PX = 20;

type DragState = {
  pointerId: number;
  startX: number;
  startY: number;
  originX: number;
  originY: number;
  moved: boolean;
};

function portalHost(): HTMLElement {
  return document.querySelector<HTMLElement>(".webui2-page") ?? document.body;
}

export function ComputerWorkspace({ client }: { client: ComputerClient | null }): ReactElement | null {
  const computer = useComputerSession(client);
  const [approvalOpen, setApprovalOpen] = useState(false);
  const [preferences, setPreferences] = useState(loadComputerWorkspacePreferences);
  const [position, setPosition] = useState<ComputerWorkspacePoint | null>(preferences.position);
  const [dragging, setDragging] = useState(false);
  const panelRef = useRef<HTMLElement>(null);
  const dragRef = useRef<DragState | null>(null);

  useEffect(() => {
    setApprovalOpen(false);
    void computer.loadCapabilities().catch(() => undefined); // Hook renders the error.
  }, [computer.loadCapabilities]);

  useEffect(() => {
    const onResize = (): void => {
      const rect = panelRef.current?.getBoundingClientRect();
      if (!rect || !position) return;
      const next = clampComputerWorkspacePosition(position, { x: window.innerWidth, y: window.innerHeight }, { x: rect.width, y: rect.height });
      if (next.x !== position.x || next.y !== position.y) {
        setPosition(next);
        setPreferences((current) => ({ ...current, position: next }));
        saveComputerWorkspacePreferences({ ...preferences, position: next });
      }
    };
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, [position, preferences]);

  const persist = useCallback((next: Partial<typeof preferences>): void => {
    setPreferences((current) => {
      const merged = { ...current, ...next };
      saveComputerWorkspacePreferences(merged);
      return merged;
    });
  }, []);

  if (!client) return null;

  const readiness = computerReadinessError(computer.available, computer.capabilities);
  const state = computer.session?.state ?? "idle";
  const status = readiness ? "attention" : state;
  const backend = computer.capabilities?.backend || "detecting";
  const start = async () => {
    setApprovalOpen(false);
    try {
      const session = await computer.start({ approved: true });
      if (session?.state === "ready" || session?.state === "needs_observation") await computer.observe();
    } catch { /* surfaced in the panel */ }
  };
  const action = (operation: () => Promise<unknown>) => { void operation().catch(() => undefined); };

  const toggleCollapsed = (): void => {
    const collapsed = !preferences.collapsed;
    persist({ collapsed });
  };

  const onDragStart = (event: ReactPointerEvent<HTMLDivElement>): void => {
    if (event.button !== 0 || !panelRef.current) return;
    const rect = panelRef.current.getBoundingClientRect();
    dragRef.current = { pointerId: event.pointerId, startX: event.clientX, startY: event.clientY, originX: rect.left, originY: rect.top, moved: false };
    try { event.currentTarget.setPointerCapture(event.pointerId); } catch { /* Pointer capture is unavailable in some test environments. */ }
  };

  const onDragMove = (event: ReactPointerEvent<HTMLDivElement>): void => {
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

  const onDragEnd = (event: ReactPointerEvent<HTMLDivElement>): void => {
    const current = dragRef.current;
    if (!current || current.pointerId !== event.pointerId) return;
    dragRef.current = null;
    try { event.currentTarget.releasePointerCapture(event.pointerId); } catch { /* Pointer capture is unavailable in some test environments. */ }
    if (!current.moved) return;
    setDragging(false);
    const rect = panelRef.current?.getBoundingClientRect();
    if (!rect) return;
    const next = clampComputerWorkspacePosition(
      { x: current.originX + event.clientX - current.startX, y: current.originY + event.clientY - current.startY },
      { x: window.innerWidth, y: window.innerHeight },
      { x: rect.width, y: rect.height },
    );
    setPosition(next);
    persist({ position: next });
  };

  const onDragCancel = (event: ReactPointerEvent<HTMLDivElement>): void => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    setDragging(false);
  };

  const panelStyle = position ? { left: position.x, top: position.y } : { right: DEFAULT_RIGHT_PX, top: DEFAULT_TOP_PX };
  const panel = preferences.collapsed
    ? <button
      aria-expanded={false}
      aria-label="Open Computer Use workspace"
      className="webui2-computer-launcher"
      onClick={toggleCollapsed}
      type="button"
    >
      <Monitor aria-hidden="true" size={17} strokeWidth={2.2} />
      <span className={`webui2-computer-launcher-dot webui2-computer-launcher-dot--${status}`} />
      <span className="webui2-computer-launcher-label">Computer Use</span>
      <span className="webui2-computer-launcher-backend">{backend}</span>
      <ChevronDown aria-hidden="true" size={15} />
    </button>
    : <aside
      ref={panelRef}
      aria-label="Computer workspace"
      className="webui2-computer-workspace"
      data-dragging={dragging ? "true" : "false"}
      style={panelStyle}
    >
      <header className="webui2-computer-header">
        <div
          aria-label="Drag Computer Use workspace"
          className="webui2-computer-drag-handle"
          data-drag-handle="true"
          onPointerCancel={onDragCancel}
          onPointerDown={onDragStart}
          onPointerMove={onDragMove}
          onPointerUp={onDragEnd}
          role="button"
          tabIndex={0}
        >
          <GripVertical aria-hidden="true" size={16} />
          <div><span className="webui2-computer-eyebrow">CONTROL SURFACE</span><h2>Computer Workspace</h2></div>
        </div>
        <div className="webui2-computer-header-actions">
          <span className="webui2-computer-backend">{backend}</span>
          <button aria-label="Collapse Computer Use workspace" className="webui2-computer-icon-button" onClick={toggleCollapsed} title="Collapse" type="button"><Minus aria-hidden="true" size={16} /></button>
        </div>
      </header>
      <ComputerToolbar
        state={computer.session?.state ?? null}
        capabilities={computer.capabilities}
        available={computer.available}
        busy={computer.loading}
        controlIntent={computer.controlIntent}
        onStart={() => setApprovalOpen(true)}
        onObserve={() => action(computer.observe)}
        onPause={() => action(computer.pause)}
        onResume={() => action(computer.resume)}
        onStop={() => action(computer.stop)}
      />
      {!computer.session || computer.session.state === "stopped" ? <ComputerPermissionGuide
        available={computer.available}
        capabilities={computer.capabilities}
        client={client}
        onRecheck={async () => { await computer.loadCapabilities(); }}
      /> : null}
      {!computer.error && readiness ? <p className="webui2-computer-error" role="status">{readiness}</p> : null}
      {computer.error ? <p className="webui2-computer-error" role="alert">{computer.error}</p> : null}
      <ComputerPreview observation={computer.observation} capabilities={computer.capabilities} />
      <ComputerTimeline receipts={computer.receipts} />
      {approvalOpen ? <ComputerApprovalDialog
        available={computer.available}
        capabilities={computer.capabilities}
        busy={computer.loading}
        onApprove={() => void start()}
        onCancel={() => setApprovalOpen(false)}
      /> : null}
    </aside>;

  const executionProgress = computer.session && computer.session.state !== "stopped" ? <ComputerExecutionProgress
    controlIntent={computer.controlIntent}
    error={computer.error}
    loading={computer.loading}
    receipts={computer.receipts}
    state={computer.session.state}
  /> : null;

  return createPortal(<>{panel}{executionProgress}</>, portalHost());
}
