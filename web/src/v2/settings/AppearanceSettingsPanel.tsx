import { Check, RotateCcw, Save, ShieldCheck } from "lucide-react";
import type { JSX } from "react";
import { useI18n } from "../../lib/i18n";
import { settingsValue, type GlobalSettingsDraft, type SettingsPath } from "./globalSettingsDraft";
import { APPEARANCE_ROOT, APPEARANCE_FIELD_LABELS, DEFAULT_APPEARANCE } from "./globalVisualSettings";

type Props = { draft: GlobalSettingsDraft };

const paths = {
  enabled: [...APPEARANCE_ROOT, "enabled"] as const,
  backgroundColor: [...APPEARANCE_ROOT, "backgroundColor"] as const,
  backgroundImage: [...APPEARANCE_ROOT, "backgroundImage"] as const,
  overlayOpacity: [...APPEARANCE_ROOT, "overlayOpacity"] as const,
  emptyBlur: [...APPEARANCE_ROOT, "emptyBlur"] as const,
  conversationBlur: [...APPEARANCE_ROOT, "conversationBlur"] as const,
  composerBlur: [...APPEARANCE_ROOT, "composerBlur"] as const,
  bubbleOpacity: [...APPEARANCE_ROOT, "bubbleOpacity"] as const
} as const;

function label(language: "en" | "zh", value: readonly [string, string]): string {
  return value[language === "zh" ? 0 : 1];
}

function toggleValue(value: unknown): boolean {
  return value === true;
}

function numberValue(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function Field({ draft, path, name, type = "text", min, max, step, fallback, placeholder }: { draft: GlobalSettingsDraft; path: SettingsPath; name: readonly [string, string]; type?: "text" | "color" | "range" | "number"; min?: number; max?: number; step?: number; fallback?: number; placeholder?: string }): JSX.Element {
  const { language } = useI18n();
  const current = settingsValue(draft.doc, path);
  const text = typeof current === "string" ? current : "";
  const value = type === "range" || type === "number" ? numberValue(current, fallback ?? min ?? 0) : text;
  return <label className={`visual-settings-field${type === "range" ? " visual-settings-range" : ""}`}>
    <span>{label(language, name)}</span>
    <input aria-label={label(language, name)} type={type} value={value} min={min} max={max} step={step} placeholder={placeholder} onChange={(event) => draft.setField(path, type === "range" || type === "number" ? Number(event.target.value) : event.target.value)} />
    {type === "range" ? <output>{typeof value === "number" ? (name === APPEARANCE_FIELD_LABELS.overlayOpacity || name === APPEARANCE_FIELD_LABELS.bubbleOpacity ? Math.round(value * 100) : value) : 0}{name === APPEARANCE_FIELD_LABELS.overlayOpacity || name === APPEARANCE_FIELD_LABELS.bubbleOpacity ? "%" : " px"}</output> : null}
  </label>;
}

function Toggle({ draft, path, name }: { draft: GlobalSettingsDraft; path: SettingsPath; name: readonly [string, string] }): JSX.Element {
  const { language } = useI18n();
  const text = label(language, name);
  return <label className="visual-settings-toggle"><span>{text}</span><input aria-label={text} role="switch" type="checkbox" checked={toggleValue(settingsValue(draft.doc, path))} onChange={(event) => draft.setField(path, event.target.checked)} /></label>;
}

export function AppearanceSettingsPanel({ draft }: Props): JSX.Element {
  const { language } = useI18n();
  const zh = language === "zh";
  const disabled = draft.busy || !draft.doc || Boolean(draft.syntaxError);
  const reset = (): void => {
    if (window.confirm(zh ? "放弃未保存的外观修改？" : "Discard unsaved appearance changes?")) draft.reset();
  };
  return <div className="visual-settings-panel">
    <section className="visual-settings-section">
      <div className="visual-settings-section-heading"><div><h2>{zh ? "页面氛围" : "Page atmosphere"}</h2><p>{zh ? "这些值通过共享全局 settings 文档保存，编辑时会在当前页面预览。" : "These values are saved in the shared global settings document and previewed on this page while editing."}</p></div><Toggle draft={draft} path={paths.enabled} name={APPEARANCE_FIELD_LABELS.enabled} /></div>
      <fieldset className="visual-settings-grid" disabled={disabled}>
        <Field draft={draft} path={paths.backgroundColor} name={APPEARANCE_FIELD_LABELS.backgroundColor} type="color" />
        <Field draft={draft} path={paths.backgroundImage} name={APPEARANCE_FIELD_LABELS.backgroundImage} placeholder="https://..." />
        <Field draft={draft} path={paths.overlayOpacity} name={APPEARANCE_FIELD_LABELS.overlayOpacity} type="range" min={0} max={1} step={0.05} fallback={DEFAULT_APPEARANCE.overlayOpacity} />
        <Field draft={draft} path={paths.bubbleOpacity} name={APPEARANCE_FIELD_LABELS.bubbleOpacity} type="range" min={0.2} max={1} step={0.05} fallback={DEFAULT_APPEARANCE.bubbleOpacity} />
      </fieldset>
    </section>
    <section className="visual-settings-section">
      <div className="visual-settings-section-heading"><div><h2>{zh ? "层次与清晰度" : "Layers and clarity"}</h2><p>{zh ? "只对对应的内容层应用模糊，不会把整页变成不可读的滤镜。" : "Blur is scoped to the corresponding surface so the whole page remains readable."}</p></div></div>
      <fieldset className="visual-settings-grid" disabled={disabled}>
        <Field draft={draft} path={paths.emptyBlur} name={APPEARANCE_FIELD_LABELS.emptyBlur} type="range" min={0} max={24} step={1} fallback={DEFAULT_APPEARANCE.emptyBlur} />
        <Field draft={draft} path={paths.conversationBlur} name={APPEARANCE_FIELD_LABELS.conversationBlur} type="range" min={0} max={24} step={1} fallback={DEFAULT_APPEARANCE.conversationBlur} />
        <Field draft={draft} path={paths.composerBlur} name={APPEARANCE_FIELD_LABELS.composerBlur} type="range" min={0} max={24} step={1} fallback={DEFAULT_APPEARANCE.composerBlur} />
      </fieldset>
    </section>
    {draft.error ? <p className="visual-settings-error" role="alert">{draft.error}</p> : null}
    {draft.validation && !draft.validation.valid ? <ul className="visual-settings-error" role="alert">{draft.validation.issues.map((issue) => <li key={`${issue.field}-${issue.code}`}>{issue.field}: {issue.message}</li>)}</ul> : null}
    {draft.saved ? <p className="visual-settings-success" role="status"><Check size={15} />{zh ? "已保存并完成读回确认。" : "Saved and confirmed by readback."}</p> : null}
    <footer className="visual-settings-footer"><span>{draft.dirty ? (zh ? "有未保存的修改" : "Unsaved changes") : (zh ? "与已保存配置一致" : "Matches saved configuration")}</span><div><button type="button" disabled={!draft.dirty || draft.busy} onClick={reset}><RotateCcw size={15} />{zh ? "重置" : "Reset"}</button><button type="button" disabled={disabled} onClick={() => void draft.validate()}><ShieldCheck size={15} />{zh ? "校验" : "Validate"}</button><button type="button" className="primary" disabled={!draft.dirty || disabled || draft.conflict} onClick={() => void draft.save()}><Save size={15} />{draft.operation === "save" ? (zh ? "保存中…" : "Saving…") : (zh ? "保存更改" : "Save changes")}</button></div></footer>
  </div>;
}
