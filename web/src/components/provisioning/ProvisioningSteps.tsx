import { Check, Circle } from "lucide-react";
import type { Language } from "../../lib/i18n";

export function ProvisioningSteps({ current, labels, language, onSelect }: { current: number; labels: string[]; language: Language; onSelect: (index: number) => void }) {
  return <ol className="provisioning-steps" aria-label={language === "zh" ? "向导进度" : "Provisioning progress"}>{labels.map((label, index) => { const available = index <= current; return <li className={index === current ? "active" : index < current ? "complete" : ""} key={label}><button className="provisioning-step-button" disabled={!available} aria-current={index === current ? "step" : undefined} onClick={() => onSelect(index)} type="button"><span className="provisioning-step-marker">{index < current ? <Check size={14} /> : <Circle size={11} />}</span><span><strong>0{index + 1}</strong><small>{label}</small></span></button></li>; })}</ol>;
}
