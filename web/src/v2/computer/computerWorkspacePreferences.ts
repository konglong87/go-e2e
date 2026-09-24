export const COMPUTER_WORKSPACE_PREFERENCES_KEY = "go-e2e.computer-workspace.v1";
export const COMPUTER_WORKSPACE_VIEWPORT_MARGIN = 12;

export type ComputerWorkspacePoint = { x: number; y: number };
export type ComputerWorkspacePreferences = {
  collapsed: boolean;
  position: ComputerWorkspacePoint | null;
};

const DEFAULT_PREFERENCES: ComputerWorkspacePreferences = { collapsed: true, position: null };

function finitePoint(value: unknown): value is ComputerWorkspacePoint {
  if (!value || typeof value !== "object") return false;
  const point = value as Record<string, unknown>;
  return typeof point.x === "number" && Number.isFinite(point.x)
    && typeof point.y === "number" && Number.isFinite(point.y);
}

export function clampComputerWorkspacePosition(
  point: ComputerWorkspacePoint,
  viewport: ComputerWorkspacePoint,
  size: ComputerWorkspacePoint,
): ComputerWorkspacePoint {
  const maxX = Math.max(COMPUTER_WORKSPACE_VIEWPORT_MARGIN, viewport.x - size.x - COMPUTER_WORKSPACE_VIEWPORT_MARGIN);
  const maxY = Math.max(COMPUTER_WORKSPACE_VIEWPORT_MARGIN, viewport.y - size.y - COMPUTER_WORKSPACE_VIEWPORT_MARGIN);
  return {
    x: Math.min(maxX, Math.max(COMPUTER_WORKSPACE_VIEWPORT_MARGIN, point.x)),
    y: Math.min(maxY, Math.max(COMPUTER_WORKSPACE_VIEWPORT_MARGIN, point.y)),
  };
}

export function loadComputerWorkspacePreferences(): ComputerWorkspacePreferences {
  try {
    const raw = window.localStorage.getItem(COMPUTER_WORKSPACE_PREFERENCES_KEY);
    if (!raw) return { ...DEFAULT_PREFERENCES };
    const value = JSON.parse(raw) as Record<string, unknown>;
    return {
      collapsed: typeof value.collapsed === "boolean" ? value.collapsed : DEFAULT_PREFERENCES.collapsed,
      position: finitePoint(value.position) ? value.position : null,
    };
  } catch {
    return { ...DEFAULT_PREFERENCES };
  }
}

export function saveComputerWorkspacePreferences(preferences: ComputerWorkspacePreferences): void {
  try {
    window.localStorage.setItem(COMPUTER_WORKSPACE_PREFERENCES_KEY, JSON.stringify(preferences));
  } catch {
    // Floating controls remain usable when storage is unavailable.
  }
}
