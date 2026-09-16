import type { JSX } from "react";
import { Plus } from "lucide-react";
import { useI18n } from "../../lib/i18n";
import goE2E from "../assets/go-e2e-animation.svg";

type EmptyStateProps = {
  onCreateSession: () => void;
};

export function EmptyState({ onCreateSession }: EmptyStateProps): JSX.Element {
  const { t } = useI18n();
  return <section className="webui2-empty-state">
    <img alt={t("webui2.emptyState.imageAlt")} height="128" src={goE2E} width="320" />
    <div><h1>{t("webui2.emptyState.title")}</h1><p>{t("webui2.emptyState.description")}</p></div>
    <button onClick={onCreateSession} type="button"><Plus aria-hidden="true" size={16} />{t("webui2.newSession")}</button>
  </section>;
}
