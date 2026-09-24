import type { ComputerCapabilities } from "./types";

/** Session approval is separate from OS permission/readiness. Fail closed on unknowns. */
export function computerReadinessError(available: boolean, caps: ComputerCapabilities | null): string | null {
  if (!available) return "Computer Use is unavailable on this host.";
  if (!caps) return "Checking Computer Use capabilities…";
  if (caps.capture_readiness !== "ready") return `Screen capture is not ready: ${caps.capture_readiness}.`;
  if (caps.input_readiness !== "ready") return `Desktop input is not ready: ${caps.input_readiness}.`;
  if (caps.permission_state !== "approved") return `System permission is not approved: ${caps.permission_state}.`;
  if (!caps.image_supported) return "Desktop images are not supported.";
  if (!caps.supports_stop) return "The backend must support Stop before a session can start.";
  return null;
}
