import { describe, expect, it } from "vitest";
import {
  defaultAgentProfileDocument,
  defaultTeamPolicy,
  parseAgentProfileJSON,
  profileCapabilitySummary,
  validateAgentProfileDraft
} from "./agentProfiles";

describe("agent profile editor helpers", () => {
  it("creates safe builtin-shaped documents for chat and coder profiles", () => {
    const chat = defaultAgentProfileDocument("chat");
    const coder = defaultAgentProfileDocument("code");

    expect(chat.prompt.mode).toBe("chat");
    expect(chat.context.workspace).toBe(false);
    expect(chat.safety.sandbox).toBe("required");
    expect(coder.prompt.mode).toBe("code");
    expect(coder.context.workspace).toBe(true);
    expect(coder.capabilities.tools.allow).toContain("Bash");
  });

  it("rejects malformed and unsafe profile drafts before an API call", () => {
    expect(parseAgentProfileJSON("{\"schema_version\":1}")).toEqual({ schema_version: 1 });
    expect(() => parseAgentProfileJSON("[]")).toThrow("JSON object");

    const report = validateAgentProfileDraft({
      ...defaultAgentProfileDocument("chat"),
      prompt: { ...defaultAgentProfileDocument("chat").prompt, mode: "chat" },
      context: { ...defaultAgentProfileDocument("chat").context, workspace: true },
      safety: { ...defaultAgentProfileDocument("chat").safety, permission_mode: "bypassPermissions" }
    });

    expect(report.valid).toBe(false);
    expect(report.issues.map((issue) => issue.code)).toEqual(expect.arrayContaining(["chat_workspace_forbidden", "bypass_permission_forbidden"]));
  });

  it("summarizes capabilities without leaking raw profile content", () => {
    const summary = profileCapabilitySummary({
      ...defaultAgentProfileDocument("chat"),
      capabilities: {
        ...defaultAgentProfileDocument("chat").capabilities,
        tools: { allow: ["Read", "WebSearch"], deny: ["Bash"] },
        skills: ["copywriting"]
      }
    });

    expect(summary.tools).toBe("2 allowed / 1 denied");
    expect(summary.skills).toBe(1);
    expect(summary.context).toContain("tenant memory");
  });

  it("provides a coordinator-first team policy", () => {
    expect(defaultTeamPolicy()).toMatchObject({
      orchestration: { mode: "coordinator", max_rounds: 4, max_parallel_members: 3 },
      output: { phase_updates: "coordinator_only", post_member_cards: false }
    });
  });
});
