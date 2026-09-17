import { expect, test } from "@playwright/test";
import { installWebUIV2Sessions } from "./fixtures/webuiV2Sessions";

const settingsDoc = {
  provider: "custom",
  providerProtocol: "openai-chat-completions",
  baseURL: "https://primary.example/v1",
  model: "fixture-model",
  fallback: {
    enabled: true,
    providers: [{
      name: "glm",
      type: "openai-compatible",
      protocol: "openai-chat-completions",
      baseURL: "https://glm.example/v1",
      model: "glm-5.2"
    }]
  },
  appearance: {
    enabled: false,
    backgroundColor: "#f7f8fa",
    backgroundImage: "",
    overlayOpacity: 0,
    emptyBlur: 0,
    conversationBlur: 0,
    composerBlur: 0,
    bubbleOpacity: 1
  },
  pet: {
    enabled: false,
    visible: true,
    model: "go-companion",
    scale: 1,
    right: 24,
    bottom: 20,
    animation: "idle",
    statusBubble: true,
    fontScale: 1
  }
};

test.beforeEach(async ({ page }) => {
  let document = structuredClone(settingsDoc);
  let revision = "r1";
  await installWebUIV2Sessions(page);
  await page.route("**/runtime/settings/environments", (route) => route.fulfill({ json: {
    environments: [{ id: "current", label: "Web", database: "web", tenant_key: "fixture", user_id: "operator", api_path: "", available: true }],
    global_settings_path: "/settings.json",
    global_settings_shared: true
  } }));
  await page.route("**/runtime/settings/validate", (route) => route.fulfill({ json: { valid: true, issues: [] } }));
  await page.route("**/runtime/settings", async (route) => {
    if (route.request().method() === "PUT") {
      document = route.request().postDataJSON();
      revision = "r2";
      return route.fulfill({ json: { saved: true, revision } });
    }
    return route.fulfill({ json: { doc: document, revision, path: "~/.golang-cc/settings.json", exists: true, masked: [] } });
  });
  await page.route("**/runtime/settings/promote-provider", async (route) => {
    const request = route.request().postDataJSON() as { doc: typeof settingsDoc; provider_index: number };
    const provider = request.doc.fallback.providers[request.provider_index];
    const promoted = {
      ...request.doc,
      provider: provider.type,
      providerProtocol: provider.protocol,
      baseURL: provider.baseURL,
      model: provider.model
    };
    return route.fulfill({ json: { doc: promoted, revision, masked: [] } });
  });
  await page.route("**/tenant/telemetry*", (route) => route.fulfill({ json: { data: [{ id: 1, name: "model.request.finished", status: "ok", created_at: "2026-09-17T01:02:03Z" }] } }));
  await page.route("**/tenant/usage/daily*", (route) => route.fulfill({ json: { data: [{ id: 1, usage_date: "2026-09-17", model: "fixture-model", request_count: 2, total_tokens: 128 }] } }));
  await page.route("**/tenant/usage/ledger*", (route) => route.fulfill({ json: { data: [] } }));
  await page.goto("/webui/v2/settings/appearance");
});

test("renders the compact settings registry and previews appearance changes with reset", async ({ page }, testInfo) => {
  await expect(page.getByRole("heading", { name: "Appearance", exact: true })).toBeVisible();
  const openNavigation = page.getByRole("button", { name: "Open settings navigation", exact: true });
  if (await openNavigation.isVisible()) await openNavigation.click();
  await expect(page.getByRole("navigation", { name: "Settings categories" }).getByRole("button", { name: "General", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Appearance", exact: true })).toHaveAttribute("aria-current", "page");
  const closeNavigation = page.getByRole("complementary").getByRole("button", { name: "Close settings navigation", exact: true });
  if (await closeNavigation.isVisible()) await closeNavigation.click();

  await page.getByLabel("Enable appearance").check();
  await page.getByLabel("Background color").fill("#123456");
  await expect(page.locator(".webui2-page")).toHaveAttribute("data-appearance-enabled", "true");
  await expect.poll(() => page.locator(".webui2-page").evaluate((node) => getComputedStyle(node).getPropertyValue("--webui2-visual-background-color").trim())).toBe("#123456");

  page.once("dialog", (dialog) => void dialog.accept());
  await page.getByRole("button", { name: "Reset", exact: true }).click();
  await expect(page.locator(".webui2-page")).toHaveAttribute("data-appearance-enabled", "false");
  await expect.poll(() => page.locator(".webui2-page").evaluate((node) => getComputedStyle(node).getPropertyValue("--webui2-visual-background-color").trim())).toBe("#f7f8fa");
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: testInfo.outputPath("settings-v2-appearance.png"), fullPage: false });
});

test("renders pet preview and reads actual observability records", async ({ page }, testInfo) => {
  const openNavigation = page.getByRole("button", { name: "Open settings navigation", exact: true });
  if (await openNavigation.isVisible()) await openNavigation.click();
  await page.getByRole("button", { name: "Pet", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Pet", exact: true })).toBeVisible();
  await page.getByLabel("Enable pet").check();
  const preview = page.locator(".pet-settings-preview");
  const canvas = preview.locator("canvas");
  await expect(preview.locator("[data-render-state]")).toHaveAttribute("data-render-state", "ready");
  await expect(canvas).toBeVisible();
  const pixels = await canvas.evaluate((node) => {
    const source = node as HTMLCanvasElement;
    const copy = document.createElement("canvas");
    copy.width = source.width;
    copy.height = source.height;
    const context = copy.getContext("2d")!;
    context.drawImage(source, 0, 0);
    const data = context.getImageData(0, 0, copy.width, copy.height).data;
    let colored = 0;
    for (let i = 3; i < data.length; i += 4) if (data[i] > 0) colored++;
    return colored / (copy.width * copy.height);
  });
  expect(pixels).toBeGreaterThan(0.1);
  expect(pixels).toBeLessThan(0.8);
  const before = await canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL());
  await expect.poll(() => canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL())).not.toBe(before);
  await page.getByLabel("Animation").selectOption("focus");
  await expect(preview.locator("[data-render-state]")).toHaveAttribute("data-render-state", "ready");
  const focused = await canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL());
  await canvas.click();
  await expect.poll(() => canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL())).not.toBe(focused);
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: testInfo.outputPath("settings-v2-pet.png"), fullPage: false });
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("status").filter({ hasText: "Saved and confirmed by readback." })).toBeVisible();

  const reopenNavigation = page.getByRole("button", { name: "Open settings navigation", exact: true });
  if (await reopenNavigation.isVisible()) await reopenNavigation.click();
  await page.getByRole("button", { name: "Observability", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Observability", exact: true })).toBeVisible();
  await expect(page.getByText("model.request.finished", { exact: true })).toBeVisible();
  await expect(page.getByText("128", { exact: true })).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: testInfo.outputPath("settings-v2-observability.png"), fullPage: false });
});

test("promotes a selected fallback model, saves it, and confirms the new primary route", async ({ page }, testInfo) => {
  await page.goto("/webui/v2/settings/models");
  await expect(page.getByRole("heading", { name: "Models", exact: true })).toBeVisible();
  await page.getByRole("button", { name: /glm-5\.2/ }).click();
  await expect(page.getByRole("button", { name: "Set as primary model", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Set as primary model", exact: true }).click();
  await expect(page.getByText("Primary model", { exact: true }).last()).toBeVisible();
  await expect(page.locator(".global-settings-footer-status")).toContainText("Unsaved changes");
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("status").filter({ hasText: "Saved and verified by readback." })).toBeVisible();
  await expect(page.locator(".global-settings-provider-list").getByText("glm-5.2", { exact: true }).first()).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: testInfo.outputPath("settings-v2-promote-primary.png"), fullPage: false });
});

async function expectNoHorizontalOverflow(page: import("@playwright/test").Page): Promise<void> {
  const widths = await page.evaluate(() => ({ body: document.body.scrollWidth, document: document.documentElement.scrollWidth, viewport: window.innerWidth }));
  expect(widths.body).toBeLessThanOrEqual(widths.viewport);
  expect(widths.document).toBeLessThanOrEqual(widths.viewport);
}
