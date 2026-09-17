import { Check, RotateCcw, Save, ShieldCheck } from "lucide-react";
import type { JSX } from "react";
import { useI18n } from "../../lib/i18n";
import { settingsValue, type GlobalSettingsDraft, type SettingsPath } from "./globalSettingsDraft";
import { DEFAULT_PET, PET_FIELD_LABELS, PET_ROOT, readVisualSettings } from "./globalVisualSettings";
import { PET_ASSETS } from "../components/petAssets";
import { PetScene } from "../components/PetScene";

type Props = { draft: GlobalSettingsDraft };

const paths = {
  enabled: [...PET_ROOT, "enabled"] as const,
  visible: [...PET_ROOT, "visible"] as const,
  model: [...PET_ROOT, "model"] as const,
  scale: [...PET_ROOT, "scale"] as const,
  right: [...PET_ROOT, "right"] as const,
  bottom: [...PET_ROOT, "bottom"] as const,
  animation: [...PET_ROOT, "animation"] as const,
  statusBubble: [...PET_ROOT, "statusBubble"] as const,
  fontScale: [...PET_ROOT, "fontScale"] as const
} as const;

function text(language: "en" | "zh", value: readonly [string, string]): string {
  return value[language === "zh" ? 0 : 1];
}

function bool(value: unknown, fallback = false): boolean {
  return typeof value === "boolean" ? value : fallback;
}

function number(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function NumberField({ draft, path, name, min, max, step = 1, fallback }: { draft: GlobalSettingsDraft; path: SettingsPath; name: readonly [string, string]; min: number; max: number; step?: number; fallback: number }): JSX.Element {
  const { language } = useI18n();
  const title = text(language, name);
  const value = number(settingsValue(draft.doc, path), fallback);
  return <label className="visual-settings-field visual-settings-range"><span>{title}</span><input aria-label={title} type="range" min={min} max={max} step={step} value={value} onChange={(event) => draft.setField(path, Number(event.target.value))} /><output>{value}</output></label>;
}

function Toggle({ draft, path, name, fallback = false }: { draft: GlobalSettingsDraft; path: SettingsPath; name: readonly [string, string]; fallback?: boolean }): JSX.Element {
  const { language } = useI18n();
  const title = text(language, name);
  return <label className="visual-settings-toggle"><span>{title}</span><input aria-label={title} role="switch" type="checkbox" disabled={draft.busy || !draft.doc || Boolean(draft.syntaxError)} checked={bool(settingsValue(draft.doc, path), fallback)} onChange={(event) => draft.setField(path, event.target.checked)} /></label>;
}

export function PetSettingsPanel({ draft }: Props): JSX.Element {
  const { language } = useI18n();
  const zh = language === "zh";
  const disabled = draft.busy || !draft.doc || Boolean(draft.syntaxError);
  const previewSettings = {
    enabled: bool(settingsValue(draft.doc, paths.enabled)),
    visible: bool(settingsValue(draft.doc, paths.visible), true),
    model: typeof settingsValue(draft.doc, paths.model) === "string" ? String(settingsValue(draft.doc, paths.model)) : "go-companion",
    scale: number(settingsValue(draft.doc, paths.scale), 1),
    right: number(settingsValue(draft.doc, paths.right), 24),
    bottom: number(settingsValue(draft.doc, paths.bottom), 20),
    animation: typeof settingsValue(draft.doc, paths.animation) === "string" ? String(settingsValue(draft.doc, paths.animation)) : "idle",
    statusBubble: bool(settingsValue(draft.doc, paths.statusBubble), true),
    fontScale: number(settingsValue(draft.doc, paths.fontScale), 1)
  };
  const reset = (): void => {
    if (window.confirm(zh ? "放弃未保存的宠物修改？" : "Discard unsaved pet changes?")) draft.reset();
  };
  return <div className="visual-settings-panel">
    <section className="visual-settings-section pet-settings-section">
      <div className="visual-settings-section-heading"><h2>{zh ? "陪伴角色" : "Companion character"}</h2><Toggle draft={draft} path={paths.enabled} name={PET_FIELD_LABELS.enabled} /></div>
      <div className="pet-settings-layout">
        <fieldset className="visual-settings-grid" disabled={disabled}>
          <Toggle draft={draft} path={paths.visible} name={PET_FIELD_LABELS.visible} fallback />
          <label className="visual-settings-field"><span>{text(language, PET_FIELD_LABELS.model)}</span><select aria-label={text(language, PET_FIELD_LABELS.model)} value={previewSettings.model} onChange={(event) => draft.setField(paths.model, event.target.value)}>{Object.entries(PET_ASSETS).map(([key, asset]) => <option value={key} key={key}>{asset.label}</option>)}</select></label>
          <label className="visual-settings-field"><span>{text(language, PET_FIELD_LABELS.animation)}</span><select aria-label={text(language, PET_FIELD_LABELS.animation)} value={previewSettings.animation} onChange={(event) => draft.setField(paths.animation, event.target.value)}><option value="idle">{zh ? "待机" : "Idle"}</option><option value="focus">{zh ? "专注" : "Focus"}</option><option value="celebrate">{zh ? "庆祝" : "Celebrate"}</option></select></label>
          <NumberField draft={draft} path={paths.scale} name={PET_FIELD_LABELS.scale} min={0.5} max={2} step={0.05} fallback={DEFAULT_PET.scale} />
          <NumberField draft={draft} path={paths.right} name={PET_FIELD_LABELS.right} min={0} max={160} fallback={DEFAULT_PET.right} />
          <NumberField draft={draft} path={paths.bottom} name={PET_FIELD_LABELS.bottom} min={0} max={160} fallback={DEFAULT_PET.bottom} />
          <Toggle draft={draft} path={paths.statusBubble} name={PET_FIELD_LABELS.statusBubble} fallback />
          <NumberField draft={draft} path={paths.fontScale} name={PET_FIELD_LABELS.fontScale} min={0.75} max={1.5} step={0.05} fallback={DEFAULT_PET.fontScale} />
        </fieldset>
        <div className="pet-settings-preview" aria-label={zh ? "宠物预览" : "Pet preview"}><PetScene settings={readVisualSettings(draft.doc).pet} status="running" /></div>
      </div>
    </section>
    {draft.error ? <p className="visual-settings-error" role="alert">{draft.error}</p> : null}
    {draft.validation && !draft.validation.valid ? <ul className="visual-settings-error" role="alert">{draft.validation.issues.map((issue) => <li key={`${issue.field}-${issue.code}`}>{issue.field}: {issue.message}</li>)}</ul> : null}
    {draft.saved ? <p className="visual-settings-success" role="status"><Check size={15} />{zh ? "已保存并完成读回确认。" : "Saved and confirmed by readback."}</p> : null}
    <footer className="visual-settings-footer"><span>{draft.dirty ? (zh ? "有未保存的修改" : "Unsaved changes") : (zh ? "与已保存配置一致" : "Matches saved configuration")}</span><div><button type="button" disabled={!draft.dirty || draft.busy} onClick={reset}><RotateCcw size={15} />{zh ? "重置" : "Reset"}</button><button type="button" disabled={disabled} onClick={() => void draft.validate()}><ShieldCheck size={15} />{zh ? "校验" : "Validate"}</button><button type="button" className="primary" disabled={!draft.dirty || disabled || draft.conflict} onClick={() => void draft.save()}><Save size={15} />{draft.operation === "save" ? (zh ? "保存中…" : "Saving…") : (zh ? "保存更改" : "Save changes")}</button></div></footer>
  </div>;
}
