import { Code, MessageSquare, X } from "lucide-react";
import { useEffect, useId, useRef, useState, type FormEvent, type JSX, type KeyboardEvent } from "react";
import { useI18n } from "../../lib/i18n";
import type { IdentityConfig } from "../../lib/types";
import { sessionControlErrorCode } from "../api/sessionControlClient";
import { useCreateSession } from "../api/sessionControlQueries";
import { validateAgentWorkspace } from "../../lib/api";
import { useRuntimeCatalog } from "../api/useSessionRuntime";
import type { OperationResult } from "../types";

type NewSessionDialogProps = {
  identity: IdentityConfig;
  open: boolean;
  defaultCWD?: string;
  onClose: () => void;
  onCreated: (result: OperationResult) => void;
};

const focusableSelector = 'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]):not([disabled])';
type CreateSubmission = { fingerprint: string; idempotencyKey: string };

export function NewSessionDialog({ identity, open, defaultCWD, onClose, onCreated }: NewSessionDialogProps): JSX.Element | null {
  const { t, language } = useI18n();
  const create = useCreateSession(identity);
  const [title, setTitle] = useState("");
  const [initialText, setInitialText] = useState("");
  const [provider, setProvider] = useState(identity.provider ?? "");
  const [model, setModel] = useState(identity.model);
  const modelEditedRef = useRef(false);
  const wasOpenRef = useRef(false);
  const [cwd, setCWD] = useState<string | null>(null);
  const [promptMode, setPromptMode] = useState("code");
  const [validating, setValidating] = useState(false);
  const [workspaceError, setWorkspaceError] = useState("");
  const { providers, models, status } = useRuntimeCatalog(identity, open);
  const defaultModel = status.data?.model?.trim() || identity.model;
  const workspace = cwd ?? defaultCWD ?? status.data?.workspace ?? "";
  const busy = create.isPending || validating;
  const [errorCode, setErrorCode] = useState("");
  const dialogRef = useRef<HTMLElement>(null);
  const titleRef = useRef<HTMLInputElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const submissionRef = useRef<CreateSubmission | null>(null);
  const titleID = useId();

  useEffect(() => {
    if (!open) {
      wasOpenRef.current = false;
      return;
    }
    if (wasOpenRef.current) return;
    wasOpenRef.current = true;
    modelEditedRef.current = false;
    setProvider(identity.provider ?? "");
    setModel(identity.model);
  }, [identity.model, identity.provider, open]);

  useEffect(() => {
    if (open && !provider && !modelEditedRef.current) {
      setModel(defaultModel);
    }
  }, [defaultModel, open, provider]);

  useEffect(() => {
    if (!open) return;
    previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    titleRef.current?.focus();
    return () => previousFocusRef.current?.focus();
  }, [open]);

  if (!open) return null;

  function close(): void {
    if (busy) return;
    setErrorCode("");
    onClose();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLElement>): void {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(focusableSelector) ?? []);
    const first = focusable[0];
    const last = focusable.at(-1);
    if (!first || !last) return;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (busy) return;
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setErrorCode("title_required");
      titleRef.current?.focus();
      return;
    }
    setErrorCode("");
    setWorkspaceError("");
    try {
      let validatedCWD = workspace.trim();
      if (validatedCWD) {
        setValidating(true);
        try {
          const result = await validateAgentWorkspace(identity, validatedCWD);
          if (!result.exists || !result.is_dir) throw new Error("invalid_workspace");
          validatedCWD = result.cwd;
        } catch {
          setWorkspaceError(language === "zh" ? "无法访问该工作目录，请检查路径和权限。" : "Workspace is unavailable. Check its path and permissions.");
          return;
        } finally { setValidating(false); }
      }
      const trimmedInitialText = initialText.trim();
      const fingerprint = JSON.stringify({ title: trimmedTitle, initialText: trimmedInitialText, provider, model, cwd: validatedCWD, promptMode });
      const submission = submissionRef.current?.fingerprint === fingerprint ? submissionRef.current : { fingerprint, idempotencyKey: crypto.randomUUID() };
      submissionRef.current = submission;
      const result = await create.mutateAsync({ title: trimmedTitle, ...(trimmedInitialText ? { initialText: trimmedInitialText } : {}), provider, model, cwd: validatedCWD, promptMode, idempotencyKey: submission.idempotencyKey });
      submissionRef.current = null;
      setTitle("");
      setInitialText("");
      setCWD(null);
      onCreated(result);
    } catch (error) {
      setErrorCode(sessionControlErrorCode(error));
    }
  }

  return <>
    <button aria-label={t("webui2.closeNewSession")} className="webui2-new-session-backdrop" disabled={busy} onClick={close} tabIndex={-1} type="button" />
    <aside aria-labelledby={titleID} aria-modal="true" className="webui2-new-session-dialog" onKeyDown={handleKeyDown} ref={dialogRef} role="dialog">
      <header><h2 id={titleID}>{t("webui2.newSession")}</h2><button aria-label={t("webui2.closeNewSession")} disabled={busy} onClick={close} type="button"><X aria-hidden="true" size={18} /></button></header>
      <form onSubmit={(event) => void submit(event)}>
        <fieldset aria-label={language === "zh" ? "对话模式" : "Conversation mode"} className="webui2-session-mode">
          <button aria-pressed={promptMode === "chat"} disabled={busy} onClick={() => setPromptMode("chat")} type="button"><MessageSquare size={15} />Chat</button>
          <button aria-pressed={promptMode === "code"} disabled={busy} onClick={() => setPromptMode("code")} type="button"><Code size={15} />Code</button>
        </fieldset>
        <label>{t("webui2.sessionTitle")}<input aria-label={t("webui2.sessionTitle")} autoComplete="off" disabled={busy} onChange={(event) => setTitle(event.target.value)} ref={titleRef} value={title} /></label>
        <div className="webui2-session-routing">
          <label>Provider<select aria-label="Provider" disabled={busy} value={provider} onChange={(event) => { const nextProvider = event.target.value; setProvider(nextProvider); modelEditedRef.current = false; const selected = providers.data?.find((item) => item.name === nextProvider); if (selected?.model) setModel(selected.model); else if (!nextProvider) setModel(defaultModel); }}><option value="">{t("webui2.provider.default")}</option>{providers.data?.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>
          <label>Model<input aria-label="Model" list={`${titleID}-models`} disabled={busy} value={model} onChange={(event) => { modelEditedRef.current = true; setModel(event.target.value); }} /><datalist id={`${titleID}-models`}>{[...new Set([...(models.data ?? []), ...(providers.data ?? []).map((item) => item.model)])].map((item) => <option key={item} value={item} />)}</datalist></label>
        </div>
        <label>{t("webui2.workspaceSection")}<input aria-label="cwd" disabled={busy} value={workspace} onChange={(event) => { setCWD(event.target.value); setWorkspaceError(""); }} /></label>
        {workspaceError ? <p role="alert">{workspaceError}</p> : null}
        {providers.isError ? <p role="alert">{t("webui2.error.network_unavailable")}</p> : null}
        <label>{t("webui2.initialInstruction")}<textarea aria-label={t("webui2.initialInstruction")} disabled={busy} onChange={(event) => setInitialText(event.target.value)} rows={3} value={initialText} /></label>
        {errorCode ? <p role="alert">{t(`webui2.error.${errorCode}`)}</p> : null}
        <footer><button disabled={busy} onClick={close} type="button">{t("webui2.cancel")}</button><button disabled={busy} type="submit">{busy ? t("webui2.creatingSession") : t("webui2.createSession")}</button></footer>
      </form>
    </aside>
  </>;
}
