import { Archive, Ellipsis, Filter, Folder, GripVertical, Plus, Search, Settings, Share2, Square, X } from "lucide-react";
import { useCallback, useEffect, useId, useRef, useState, type ChangeEvent, type CSSProperties, type DragEvent, type JSX, type KeyboardEvent as ReactKeyboardEvent, type PointerEvent as ReactPointerEvent } from "react";
import { useI18n } from "../../lib/i18n";
import type { IdentityConfig } from "../../lib/types";
import { webUIV2SessionPath } from "../routes";
import type { SessionListFilters, SessionRef, SessionStatus, SessionSummary } from "../types";
import { SESSION_REF_MIME_TYPE } from "./sessionContextDrag";
import "./composerExperience.css";
import brandLogo from "../assets/go-e2e-mark.svg";

type SessionSidebarProps = {
  sessions: SessionSummary[];
  filters: SessionListFilters;
  selectedRef: SessionRef | null;
  onFiltersChange: (filters: SessionListFilters) => void;
  onSelect: (ref: SessionRef) => void;
  onContextDragStart: (ref: SessionRef) => void;
  onOpenSettings: () => void;
  onCreateSession: () => void;
  onMobileClose?: () => void;
  onStop?: (ref: SessionRef) => void;
  onArchive?: (ref: SessionRef) => void;
  identity?: IdentityConfig;
};

const statuses: SessionStatus[] = ["idle", "queued", "running", "waiting_permission", "waiting_input", "blocked", "completed", "failed", "stopped", "archived"];
const sidebarWidthStorageKey = "golang-cc-webui.v2.sidebar-width.v1";
const sidebarMinWidth = 232;
const sidebarMaxWidth = 380;
const sidebarDefaultWidth = 288;
const sidebarKeyboardStep = 8;

export function SessionSidebar({ sessions, filters, selectedRef, onFiltersChange, onSelect, onContextDragStart, onOpenSettings, onCreateSession, onMobileClose, onStop, onArchive, identity }: SessionSidebarProps): JSX.Element {
  const { t } = useI18n();
  const [searchOpen, setSearchOpen] = useState(false);
  const searchButtonRef = useRef<HTMLButtonElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchID = useId();
  const [copyStatus, setCopyStatus] = useState("");
  const [sidebarWidth, setSidebarWidth] = useState(loadSidebarWidth);
  const resizeStartRef = useRef<{ pointerX: number; width: number } | null>(null);
  const query = filters.query.trim().toLowerCase();
  const visibleSessions = sessions.filter((session) => (filters.statuses.length === 0 || filters.statuses.includes(session.status)) && (query === "" || session.title.toLowerCase().includes(query) || session.ref.toLowerCase().includes(query) || session.cwd?.toLowerCase().includes(query)));
  const managed = visibleSessions.filter((session) => session.source === "tenant");
  const local = visibleSessions.filter((session) => session.source === "local");

  const updateSidebarWidth = useCallback((value: number): void => {
    const width = clampSidebarWidth(value);
    setSidebarWidth(width);
    try {
      window.localStorage.setItem(sidebarWidthStorageKey, String(width));
    } catch {
      // Width persistence is optional; the in-memory resize remains available.
    }
  }, []);

  useEffect(() => {
    function resize(event: PointerEvent): void {
      const start = resizeStartRef.current;
      if (start) updateSidebarWidth(start.width + event.clientX - start.pointerX);
    }
    function finishResize(): void {
      resizeStartRef.current = null;
    }
    window.addEventListener("pointermove", resize);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    return () => {
      window.removeEventListener("pointermove", resize);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
    };
  }, [updateSidebarWidth]);

  useEffect(() => {
    if (searchOpen) searchInputRef.current?.focus();
  }, [searchOpen]);

  function changeQuery(event: ChangeEvent<HTMLInputElement>): void {
    onFiltersChange({ ...filters, query: event.target.value });
  }

  function closeSearch(): void {
    setSearchOpen(false);
    onFiltersChange({ ...filters, query: "" });
    searchButtonRef.current?.focus();
  }

  function toggleStatus(status: SessionStatus): void {
    const current = new Set(filters.statuses);
    if (current.has(status)) current.delete(status);
    else current.add(status);
    onFiltersChange({ ...filters, statuses: statuses.filter((item) => current.has(item)) });
  }

  async function copyRef(ref: SessionRef): Promise<void> {
    try {
      await navigator.clipboard.writeText(ref);
      setCopyStatus(t("webui2.copied"));
    } catch {
      setCopyStatus(t("webui2.copyFailed"));
    }
  }

  function dragStart(event: DragEvent<HTMLElement>, ref: SessionRef): void {
    event.dataTransfer?.setData(SESSION_REF_MIME_TYPE, ref);
    event.dataTransfer?.setData("text/plain", ref);
    if (event.dataTransfer) event.dataTransfer.effectAllowed = "copy";
    onContextDragStart(ref);
  }

  function startResize(event: ReactPointerEvent<HTMLElement>): void {
    event.preventDefault();
    resizeStartRef.current = { pointerX: event.clientX, width: sidebarWidth };
  }

  function resizeWithKeyboard(event: ReactKeyboardEvent<HTMLElement>): void {
    const nextWidths: Partial<Record<string, number>> = {
      ArrowLeft: sidebarWidth - sidebarKeyboardStep,
      ArrowRight: sidebarWidth + sidebarKeyboardStep,
      Home: sidebarMinWidth,
      End: sidebarMaxWidth
    };
    const nextWidth = nextWidths[event.key];
    if (nextWidth === undefined) return;
    event.preventDefault();
    updateSidebarWidth(nextWidth);
  }

  function toggleStatusDisclosureWithKeyboard(event: ReactKeyboardEvent<HTMLElement>): void {
    if (event.key !== " ") return;
    event.preventDefault();
    const disclosure = event.currentTarget.parentElement;
    if (disclosure instanceof HTMLDetailsElement) disclosure.open = !disclosure.open;
  }

  return (
    <aside aria-label={t("webui2.sessions")} className="webui2-sidebar" id="webui2-session-sidebar" style={{ width: `${sidebarWidth}px` } as CSSProperties}>
      <div className="webui2-sidebar-controls">
        <div className="webui2-brand"><span className="webui2-brand-mark"><img alt="" src={brandLogo} />{t("webui2.brand")}</span>{onMobileClose ? <button aria-label={t("webui2.closeSessions")} className="webui2-sidebar-mobile-close" onClick={onMobileClose} title={t("webui2.closeSessions")} type="button"><X aria-hidden="true" size={17} /></button> : null}</div>
        <div className="webui2-sidebar-primary-actions">
          <button className="webui2-new-session" onClick={onCreateSession} type="button"><Plus aria-hidden="true" size={16} />{t("webui2.newSession")}</button>
          <button aria-controls={searchOpen ? searchID : undefined} aria-expanded={searchOpen} aria-label={t("webui2.searchSessions")} className="webui2-sidebar-search-toggle" onClick={() => searchOpen ? closeSearch() : setSearchOpen(true)} ref={searchButtonRef} title={t("webui2.searchSessions")} type="button"><Search aria-hidden="true" size={18} /></button>
        </div>
        {searchOpen ? <label className="webui2-search" id={searchID}>
          <Search aria-hidden="true" size={15} />
          <input aria-label={t("webui2.searchSessions")} ref={searchInputRef} onChange={changeQuery} onKeyDown={(event) => { if (event.key === "Escape") { event.preventDefault(); closeSearch(); } }} placeholder={t("webui2.searchSessions")} type="search" value={filters.query} />
        </label> : null}
        {/* Temporarily hide status filtering while retaining the existing controls. */}
        <details className="webui2-status-filter-disclosure" hidden>
          <summary aria-label={t("webui2.filterStatus")} onKeyDown={toggleStatusDisclosureWithKeyboard}>
            <Filter aria-hidden="true" size={15} />
            <span>{t("webui2.filterStatus")}</span>
            <span aria-live="polite" className="webui2-status-filter-count">{filters.statuses.length === 0 ? t("webui2.allStatuses") : t("webui2.statusFilterCount", { count: filters.statuses.length })}</span>
          </summary>
          <fieldset className="webui2-status-filters">
            <legend className="webui2-visually-hidden">{t("webui2.filterStatus")}</legend>
            {statuses.map((status) => <label key={status}><input aria-label={t(`webui2.status.${status}`)} checked={filters.statuses.includes(status)} onChange={() => toggleStatus(status)} type="checkbox" /><span>{t(`webui2.status.${status}`)}</span></label>)}
          </fieldset>
        </details>
      </div>
      <div className="webui2-session-list">
        <SessionGroup copyRef={copyRef} hideTitle onContextDragStart={dragStart} onSelect={onSelect} onStop={onStop} onArchive={onArchive} selectedRef={selectedRef} sessions={managed} title={t("webui2.managed")} />
        <SessionGroup copyRef={copyRef} onContextDragStart={dragStart} onSelect={onSelect} selectedRef={selectedRef} sessions={local} subtitle={t("webui2.localReadOnly")} title={t("webui2.local")} />
        {visibleSessions.length === 0 ? <p className="webui2-list-empty">{t("webui2.emptySessions")}</p> : null}
      </div>
      <div className="webui2-sidebar-bottom">
        {copyStatus ? <p aria-live="polite" className="webui2-copy-status">{copyStatus}</p> : null}
        <button aria-label={t("webui2.settings")} className="webui2-settings-launcher" onClick={onOpenSettings} title={t("webui2.settings")} type="button"><Settings aria-hidden="true" size={17} />{t("webui2.settings")}</button>
        {identity ? <div className="webui2-sidebar-identity"><span className="webui2-sidebar-avatar">{(identity.userId || "U").slice(0, 1).toUpperCase()}</span><div><strong>{identity.tenantKey || t("webui2.brand")}</strong><small>{identity.userId || "—"}</small></div></div> : null}
      </div>
      <hr aria-label={t("webui2.resizeSidebar")} aria-orientation="vertical" aria-valuemax={sidebarMaxWidth} aria-valuemin={sidebarMinWidth} aria-valuenow={sidebarWidth} className="webui2-sidebar-resize" onKeyDown={resizeWithKeyboard} onPointerDown={startResize} tabIndex={0} />
    </aside>
  );
}

function clampSidebarWidth(value: number): number {
  return Math.min(sidebarMaxWidth, Math.max(sidebarMinWidth, Math.round(value)));
}

function loadSidebarWidth(): number {
  try {
    const value = Number(window.localStorage.getItem(sidebarWidthStorageKey));
    return Number.isFinite(value) && value > 0 ? clampSidebarWidth(value) : sidebarDefaultWidth;
  } catch {
    return sidebarDefaultWidth;
  }
}

type SessionGroupProps = {
  sessions: SessionSummary[];
  title: string;
  subtitle?: string;
  hideTitle?: boolean;
  onStop?: (ref: SessionRef) => void;
  onArchive?: (ref: SessionRef) => void;
  selectedRef: SessionRef | null;
  onSelect: (ref: SessionRef) => void;
  onContextDragStart: (event: DragEvent<HTMLElement>, ref: SessionRef) => void;
  copyRef: (ref: SessionRef) => Promise<void>;
};

function SessionGroup({ sessions, title, subtitle, hideTitle, selectedRef, onSelect, onContextDragStart, copyRef, onStop, onArchive }: SessionGroupProps): JSX.Element | null {
  const { t } = useI18n();
  const draggedRef = useRef<SessionRef | null>(null);
  const workspaces = new Map<string, SessionSummary[]>();
  for (const session of sessions) {
    const cwd = session.cwd || "";
    workspaces.set(cwd, [...(workspaces.get(cwd) ?? []), session]);
  }
  if (sessions.length === 0) return null;
  return <section aria-label={title} className="webui2-session-group">
    {hideTitle ? null : <header><strong>{title}</strong>{subtitle ? <span>{subtitle}</span> : null}</header>}
    {[...workspaces].map(([cwd, items]) => <div className="webui2-workspace-group" key={cwd}>
    {cwd ? <div className="webui2-workspace-label" title={cwd}><Folder aria-hidden="true" size={13} /><span>{cwd.replace(/\/$/, "").split("/").pop() || cwd}</span><span>{items.length}</span></div> : null}
    {items.map((session) => <div className="webui2-session-row" key={session.ref}>
      <button aria-label={t("webui2.dragRef", { ref: session.ref })} className="webui2-drag-handle" draggable onDragStart={(event) => onContextDragStart(event, session.ref)} type="button"><GripVertical aria-hidden="true" size={14} /></button>
      <button aria-pressed={selectedRef === session.ref} className="webui2-session-select" draggable onDragStart={(event) => { draggedRef.current = session.ref; onContextDragStart(event, session.ref); }} onPointerDown={() => { draggedRef.current = null; }} onClick={(event) => { if (draggedRef.current !== session.ref || event.detail === 0) onSelect(session.ref); draggedRef.current = null; }} type="button">
        <span className="webui2-session-title">{session.title}</span>
        <span className="webui2-session-meta">
          <span className="webui2-session-status"><span aria-hidden="true" className="webui2-session-status-dot" data-status={session.status} />{t(`webui2.status.${session.status}`)}</span>
          <code>{session.shortID}</code>
          <time dateTime={session.updatedAt}>{formatTime(session.updatedAt)}</time>
        </span>
      </button>
      <SessionOverflowMenu copyRef={copyRef} session={session} onStop={onStop} onArchive={onArchive} />
    </div>)}
    </div>)}
  </section>;
}

type SessionOverflowMenuProps = {
  session: SessionSummary;
  copyRef: (ref: SessionRef) => Promise<void>;
  onStop?: (ref: SessionRef) => void;
  onArchive?: (ref: SessionRef) => void;
};

function SessionOverflowMenu({ session, copyRef, onStop, onArchive }: SessionOverflowMenuProps): JSX.Element {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuID = useId();

  useEffect(() => {
    if (!open) return;
    function closeFromOutside(event: PointerEvent): void {
      if (!menuRef.current?.contains(event.target as Node)) setOpen(false);
    }
    function closeFromEscape(event: globalThis.KeyboardEvent): void {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
    }
    document.addEventListener("pointerdown", closeFromOutside);
    document.addEventListener("keydown", closeFromEscape);
    return () => {
      document.removeEventListener("pointerdown", closeFromOutside);
      document.removeEventListener("keydown", closeFromEscape);
    };
  }, [open]);

  async function copyAndClose(): Promise<void> {
    setOpen(false);
    await copyRef(session.ref);
  }

  async function shareAndClose(): Promise<void> {
    setOpen(false);
    if (!navigator.share) { await copyRef(session.ref); return; }
    try { await navigator.share({ title: session.title, url: `${window.location.origin}${webUIV2SessionPath(session.ref)}` }); } catch { /* Cancelling the share sheet leaves the session unchanged. */ }
  }

  function mutateSession(kind: "stop" | "archive"): void {
    setOpen(false);
    if (window.confirm(t(`webui2.confirm.${kind}`))) (kind === "stop" ? onStop : onArchive)?.(session.ref);
  }

  const stoppable = session.source === "tenant" && ["queued", "running", "waiting_permission", "waiting_input"].includes(session.status);

  return <div className="webui2-session-actions" ref={menuRef}>
    <button aria-controls={menuID} aria-expanded={open} aria-haspopup="menu" aria-label={t("webui2.sessionActions", { ref: session.ref })} onClick={() => setOpen((current) => !current)} ref={triggerRef} title={t("webui2.sessionActions", { ref: session.ref })} type="button"><Ellipsis aria-hidden="true" size={16} /></button>
    {open ? <div aria-label={t("webui2.sessionActions", { ref: session.ref })} className="webui2-session-action-menu" id={menuID} role="menu">
      <button aria-label={t("webui2.copyRef", { ref: session.ref })} onClick={() => void copyAndClose()} role="menuitem" title={t("webui2.copy")} type="button">{t("webui2.copy")}</button>
      <a aria-label={t("webui2.openRef", { ref: session.ref })} href={webUIV2SessionPath(session.ref)} onClick={() => setOpen(false)} role="menuitem" title={t("webui2.openSession")}>{t("webui2.openSession")}</a>
      <button onClick={() => void shareAndClose()} role="menuitem" type="button"><Share2 size={14} />{t("webui2.share")}</button>
      {stoppable && onStop ? <button onClick={() => mutateSession("stop")} role="menuitem" type="button"><Square size={14} />{t("webui2.stop")}</button> : null}
      {session.source === "tenant" && onArchive ? <button onClick={() => mutateSession("archive")} role="menuitem" type="button"><Archive size={14} />{t("webui2.archive")}</button> : null}
    </div> : null}
  </div>;
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
