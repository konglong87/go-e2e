import type { ChatMessage, TenantMessage } from "./types";

export function tenantMessagesToChat(messages: TenantMessage[]): ChatMessage[] {
  return [...messages]
    .sort((a, b) => (a.turn_index || 0) - (b.turn_index || 0) || (a.id || 0) - (b.id || 0))
    .map((message) => ({
      id: String(message.id),
      role: message.role === "assistant" ? "assistant" : message.role === "system" ? "system" : "user",
      content: message.content || contentFromJSON(message.content_json) || "",
      status: message.status || statusFromJSON(message.content_json),
      messageId: message.id,
      turnIndex: message.turn_index,
      createdAt: message.created_at
    }));
}

function statusFromJSON(value?: string): string {
  if (!value) {
    return "";
  }
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.status === "string") {
      return parsed.status;
    }
    if (parsed.mobile && typeof parsed.mobile === "object") {
      const mobile = parsed.mobile as Record<string, unknown>;
      if (typeof mobile.status === "string") {
        return mobile.status;
      }
    }
  } catch {
    return "";
  }
  return "";
}

function contentFromJSON(value?: string): string {
  if (!value) {
    return "";
  }
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.content === "string") {
      return parsed.content;
    }
    if (parsed.mobile && typeof parsed.mobile === "object") {
      const mobile = parsed.mobile as Record<string, unknown>;
      if (typeof mobile.content === "string") {
        return mobile.content;
      }
    }
  } catch {
    return "";
  }
  return "";
}

export function formatTokens(total: number): string {
  if (total >= 1_000_000) {
    return `${(total / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
  }
  if (total >= 1000) {
    return `${(total / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  }
  return String(total);
}

export function formatDuration(ms: number) {
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return `${minutes}m ${rest}s`;
}
