import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "../../lib/i18n";
import type { IdentityConfig } from "../../lib/types";
import { SessionControlClientProvider, SessionControlError, type SessionControlClient } from "../api/sessionControlClient";
import type { OperationResult, SessionDetail } from "../types";
import { NewSessionDialog } from "./NewSessionDialog";
import { getStatus, validateAgentWorkspace } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  listProviders: vi.fn(async () => [{ name: "custom", model: "model" }]),
  listModels: vi.fn(async () => ["model"]),
  getStatus: vi.fn(async () => ({})),
  validateAgentWorkspace: vi.fn()
}));

const identity: IdentityConfig = { apiBase: "/api", apiToken: "token", mobileJwt: "", tenantKey: "tenant", userId: "user", deviceId: "device", model: "model" };
const session: SessionDetail = { ref: "tenant:created", source: "tenant", title: "Launch plan", status: "idle", updatedAt: "2026-09-05T00:00:00.000Z", shortID: "created", messages: [], activity: [], context: [], changes: [], runs: [] };

describe("NewSessionDialog", () => {
  let host: HTMLDivElement;
  let root: Root;
  let client: SessionControlClient;

  beforeEach(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    host = document.createElement("div");
    document.body.replaceChildren(host);
    root = createRoot(host);
    const result: OperationResult = { operation: { id: "operation-create", kind: "create", status: "completed", title: "", detail: "", createdAt: session.updatedAt }, session, replayed: false };
    client = {
      list: vi.fn(async () => []), get: vi.fn(async () => session),
      create: vi.fn(async (): Promise<OperationResult> => result),
      send: vi.fn(), stop: vi.fn(), archive: vi.fn()
    };
  });

  afterEach(() => {
    act(() => root.unmount());
    delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT;
  });

  function render(onCreated = vi.fn<(result: OperationResult) => void>(), defaultCWD?: string): ReturnType<typeof vi.fn> {
    const onClose = vi.fn();
    act(() => root.render(<I18nProvider><QueryClientProvider client={new QueryClient()}><SessionControlClientProvider client={client}><NewSessionDialog identity={identity} defaultCWD={defaultCWD} onClose={onClose} onCreated={onCreated} open /></SessionControlClientProvider></QueryClientProvider></I18nProvider>));
    return onClose;
  }

  function setValue(selector: string, value: string): void {
    const input = host.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector);
    const prototype = input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    const setter = Object.getOwnPropertyDescriptor(prototype, "value")?.set;
    act(() => {
      setter?.call(input, value);
      input?.dispatchEvent(new Event("input", { bubbles: true }));
    });
  }

  it("uses the saved primary model for the default provider", async () => {
    vi.mocked(getStatus).mockResolvedValueOnce({ model: "gpt-5.6-sol" });
    render();

    await vi.waitFor(() => expect(host.querySelector<HTMLInputElement>('input[aria-label="Model"]')?.value).toBe("gpt-5.6-sol"));
    expect(host.querySelector<HTMLSelectElement>('select[aria-label="Provider"]')?.value).toBe("");

    setValue('input[aria-label="Session title"]', "Primary model test");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());

    expect(client.create).toHaveBeenCalledWith(identity, expect.objectContaining({ provider: "", model: "gpt-5.6-sol" }));
  });

  it("focuses its title, rejects an empty title, and creates once with the optional instruction", async () => {
    const onCreated = vi.fn<(result: OperationResult) => void>();
    render(onCreated);
    const dialog = host.querySelector<HTMLElement>('[role="dialog"][aria-modal="true"]');
    const title = host.querySelector<HTMLInputElement>('input[aria-label="Session title"]');

    expect(dialog).not.toBeNull();
    expect(document.activeElement).toBe(title);
    act(() => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    expect(client.create).not.toHaveBeenCalled();
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("Enter a session title");

    setValue('input[aria-label="Session title"]', "  Launch plan  ");
    setValue('textarea[aria-label="Initial instruction"]', "  Prepare the rollout  ");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());

    expect(client.create).toHaveBeenCalledTimes(1);
    expect(client.create).toHaveBeenCalledWith(identity, expect.objectContaining({ title: "Launch plan", initialText: "Prepare the rollout", idempotencyKey: expect.any(String) }));
    expect(onCreated).toHaveBeenCalledWith(expect.objectContaining({ session: expect.objectContaining({ ref: "tenant:created" }) }));
  });

  it("reuses the idempotency key for an unchanged draft after an ambiguous failure", async () => {
    const result: OperationResult = { operation: { id: "operation-create", kind: "create", status: "completed", title: "", detail: "", createdAt: session.updatedAt }, session, replayed: true };
    const create = vi.fn<SessionControlClient["create"]>()
      .mockRejectedValueOnce(new SessionControlError("network_unavailable"))
      .mockResolvedValueOnce(result)
      .mockResolvedValueOnce({ ...result, replayed: false });
    client = { ...client, create };
    render();
    setValue('input[aria-label="Session title"]', "Launch plan");
    setValue('textarea[aria-label="Initial instruction"]', "Prepare the rollout");

    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());

    expect(create).toHaveBeenCalledTimes(2);
    expect(create.mock.calls[1][1].idempotencyKey).toBe(create.mock.calls[0][1].idempotencyKey);

    setValue('input[aria-label="Session title"]', "Launch plan");
    setValue('textarea[aria-label="Initial instruction"]', "Prepare the rollout");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    expect(create.mock.calls[2][1].idempotencyKey).not.toBe(create.mock.calls[1][1].idempotencyKey);
  });

  it("rotates the idempotency key after the create draft changes", async () => {
    const create = vi.fn<SessionControlClient["create"]>().mockRejectedValue(new SessionControlError("network_unavailable"));
    client = { ...client, create };
    render();
    setValue('input[aria-label="Session title"]', "First title");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    setValue('input[aria-label="Session title"]', "Changed title");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());

    expect(create).toHaveBeenCalledTimes(2);
    expect(create.mock.calls[1][1].idempotencyKey).not.toBe(create.mock.calls[0][1].idempotencyKey);
  });

  it("validates the inherited workspace and persists its canonical path with Chat mode", async () => {
    vi.mocked(validateAgentWorkspace).mockResolvedValueOnce({ cwd: "/work/project", workspace_name: "project", exists: true, is_dir: true, is_git_repo: true });
    render(vi.fn(), "/work/project/../project");
    setValue('input[aria-label="Session title"]', "Chat test");
    act(() => [...host.querySelectorAll("button")].find((button) => button.textContent === "Chat")?.click());
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    expect(validateAgentWorkspace).toHaveBeenCalledWith(identity, "/work/project/../project");
    expect(client.create).toHaveBeenCalledWith(identity, expect.objectContaining({ cwd: "/work/project", promptMode: "chat" }));
  });

  it("keeps the draft and prevents creation when workspace validation fails", async () => {
    vi.mocked(validateAgentWorkspace).mockRejectedValueOnce(new Error("not found"));
    render(vi.fn(), "/missing");
    setValue('input[aria-label="Session title"]', "Keep this draft");
    await act(async () => host.querySelector<HTMLButtonElement>('button[type="submit"]')?.click());
    expect(client.create).not.toHaveBeenCalled();
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("Workspace is unavailable");
    expect(host.querySelector<HTMLInputElement>('input[aria-label="Session title"]')?.value).toBe("Keep this draft");
  });
});
