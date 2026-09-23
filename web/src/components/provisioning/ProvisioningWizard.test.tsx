import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../../lib/api";
import { I18nProvider } from "../../lib/i18n";
import type { IdentityConfig, ProvisioningRecord } from "../../lib/types";
import { ProvisioningWizard } from "./ProvisioningWizard";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

vi.mock("../../lib/api", async (importOriginal) => ({
  ...await importOriginal<typeof import("../../lib/api")>(),
  getProvisioningOverview: vi.fn(),
  listAgentProfiles: vi.fn(),
  listChannelAccounts: vi.fn(),
  listProviders: vi.fn(),
  createProvisioning: vi.fn(),
  preflightProvisioning: vi.fn(),
  workerProvisioningAction: vi.fn()
}));

const identity: IdentityConfig = { apiBase: "/api", apiToken: "test", mobileJwt: "", tenantKey: "tenant", userId: "user", deviceId: "device", role: "owner", model: "test" };
const record: ProvisioningRecord = {
  id: 7,
  profile_key: "support-agent",
  account_key: "support-bot",
  supervisor: "screen",
  status: "draft",
  worker: { supervisor: "screen", account_key: "support-bot", provider: "openai", model: "gpt-5" },
  observed_worker: { state: "stopped" }
};

describe("ProvisioningWizard views", () => {
  let host: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getProvisioningOverview).mockResolvedValue({ records: [record], workers: [] });
    vi.mocked(api.listAgentProfiles).mockResolvedValue([{ id: 3, profile_key: "support-agent", profile_version: 1, status: "published", scope: "tenant_shared", display_name: "Support agent", description: "Support", config_json: "{}" }]);
    vi.mocked(api.listChannelAccounts).mockResolvedValue([{ id: 4, account_key: "support-bot", provider: "feishu", app_id: "cli_test", mode: "websocket", enabled: true, status: "active" }]);
    vi.mocked(api.listProviders).mockResolvedValue([{ name: "openai", model: "gpt-5" }]);
    vi.mocked(api.preflightProvisioning).mockResolvedValue({ ...record, status: "preflight", observed_worker: { state: "preflight" } });
    host = document.createElement("div");
    document.body.replaceChildren(host);
    root = createRoot(host);
  });

  afterEach(() => {
    act(() => root.unmount());
    vi.restoreAllMocks();
  });

  async function mount(view: "account" | "lifecycle") {
    await act(async () => {
      root.render(<I18nProvider><ProvisioningWizard identity={identity} onStatus={vi.fn()} view={view} /></I18nProvider>);
    });
  }

  async function click(label: string) {
    const button = Array.from(host.querySelectorAll<HTMLButtonElement>("button")).find((item) => item.textContent?.trim() === label);
    expect(button).toBeDefined();
    await act(async () => button?.click());
  }

  it("keeps Feishu connection setup separate from Worker lifecycle actions", async () => {
    await mount("account");
    expect(host.textContent).toContain("Feishu connection");
    expect(host.textContent).toContain("Existing account");
    expect(host.textContent).not.toContain("Start Worker");
    expect(host.textContent).not.toContain("Run preflight");
    expect(host.querySelector('input[type="password"]')).toBeNull();
  });

  it("keeps Worker runtime focused on saved drafts and lifecycle controls", async () => {
    await mount("lifecycle");
    expect(host.textContent).toContain("Worker runtime");
    expect(host.textContent).toContain("Select Worker");
    expect(host.textContent).not.toContain("App Secret");
    expect(host.textContent).not.toContain("Connect a new bot");

    await click("Continue");
    expect(host.textContent).toContain("Preflight settings");
    await click("Run preflight");
    expect(api.preflightProvisioning).toHaveBeenCalledWith(identity, record.id);
  });
});
