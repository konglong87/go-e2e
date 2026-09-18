import { GripVertical } from "lucide-react";
import { useEffect, useId, useRef, useState, type DragEvent, type ReactNode } from "react";
import { useI18n } from "../../lib/i18n";
import type { SessionRef, SessionSummary } from "../types";
import { SessionHoverPreview } from "./SessionHoverPreview";
import { SessionStatusIcon } from "./SessionStatusIcon";
import "./sessionSidebarExperience.css";

const PREVIEW_OPEN_DELAY = 450;
const PREVIEW_CLOSE_DELAY = 180;

type Props = {
  session: SessionSummary;
  selected: boolean;
  onSelect: (ref: SessionRef) => void;
  onContextDragStart: (event: DragEvent<HTMLElement>, ref: SessionRef) => void;
  actions: (showDetails: () => void) => ReactNode;
};

export function SessionRow({ session, selected, onSelect, onContextDragStart, actions }: Props) {
  const { t } = useI18n();
  const anchor = useRef<HTMLDivElement>(null);
  const dragged = useRef(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [previewOpen, setPreviewOpen] = useState(false);
  const previewID = useId();
  function clearTimer() { clearTimeout(timer.current); }
  function closePreview() { clearTimer(); setPreviewOpen(false); }
  function showPreview() { clearTimer(); setPreviewOpen(true); }
  function scheduleClose() {
    clearTimer();
    timer.current = setTimeout(() => setPreviewOpen(false), PREVIEW_CLOSE_DELAY);
  }
  useEffect(() => () => clearTimeout(timer.current), []);
  useEffect(() => {
    if (!previewOpen) return;
    function dismiss() { clearTimeout(timer.current); setPreviewOpen(false); }
    function escape(event: KeyboardEvent) { if (event.key === "Escape") dismiss(); }
    function scroll(event: Event) {
      if (!document.getElementById(previewID)?.contains(event.target as Node)) dismiss();
    }
    function outside(event: PointerEvent) {
      const target = event.target as Node;
      if (!anchor.current?.contains(target) && !document.getElementById(previewID)?.contains(target)) dismiss();
    }
    document.addEventListener("keydown", escape);
    document.addEventListener("pointerdown", outside);
    window.addEventListener("resize", dismiss);
    window.addEventListener("scroll", scroll, true);
    return () => {
      document.removeEventListener("keydown", escape);
      document.removeEventListener("pointerdown", outside);
      window.removeEventListener("resize", dismiss);
      window.removeEventListener("scroll", scroll, true);
    };
  }, [previewOpen, previewID]);
  function dragStart(event: DragEvent<HTMLElement>) {
    closePreview();
    dragged.current = true;
    onContextDragStart(event, session.ref);
  }
  return <div ref={anchor} className="webui2-session-row" data-selected={selected}
    onPointerEnter={(event) => {
      if (event.pointerType === "touch") return;
      clearTimer();
      timer.current = setTimeout(() => setPreviewOpen(true), PREVIEW_OPEN_DELAY);
    }} onPointerLeave={scheduleClose}>
    <button aria-label={t("webui2.dragRef", { ref: session.ref })} className="webui2-drag-handle"
      title={t("webui2.dragRef", { ref: session.ref })} draggable onDragStart={dragStart} type="button">
      <GripVertical aria-hidden="true" size={14} />
    </button>
    <button aria-pressed={selected} aria-describedby={previewOpen ? previewID : undefined}
      className="webui2-session-select" draggable onDragStart={dragStart}
      onPointerDown={() => { dragged.current = false; }}
      onFocus={showPreview} onBlur={scheduleClose}
      onClick={(event) => {
        closePreview();
        if (!dragged.current || event.detail === 0) onSelect(session.ref);
        dragged.current = false;
      }} type="button">
      <span className="webui2-session-title">{session.title}</span>
      <span className="webui2-session-meta"><SessionStatusIcon status={session.status} /><code>{session.shortID}</code></span>
    </button>
    <div className="webui2-session-action-slot" onPointerEnter={closePreview} onFocusCapture={closePreview}>
      {actions(showPreview)}
    </div>
    {previewOpen ? <SessionHoverPreview id={previewID} session={session} anchor={anchor} onEnter={clearTimer} onLeave={scheduleClose} /> : null}
  </div>;
}
