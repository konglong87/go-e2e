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
  await page.route((url) => /\/status$/.test(url.pathname), (route) => route.fulfill({ json: { ok: true, workspace: "/workspace/project", model: "fixture-model" } }));
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
  await chooseSettingsSelect(page, "Model asset", "Chinese Dragon");
  const preview = page.locator(".pet-settings-preview");
  const canvas = preview.locator("canvas");
  const scene = preview.locator("[data-render-state]");
  await expect(scene).toHaveAttribute("data-render-state", "ready");
  await expect(scene).toHaveAttribute("data-asset", "/assets/pets/chinese-dragon.glb");
  await expect(canvas).toBeVisible();
  const dragonPixels = await canvasColorRatio(canvas);
  expect(dragonPixels).toBeGreaterThan(0.1);
  expect(dragonPixels).toBeLessThan(0.8);
  const before = await canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL());
  await expect.poll(() => canvas.evaluate((node) => (node as HTMLCanvasElement).toDataURL())).not.toBe(before);
  await chooseSettingsSelect(page, "Model asset", "Go Companion");
  await expect(scene).toHaveAttribute("data-render-state", "ready");
  await expect(scene).toHaveAttribute("data-asset", "/assets/pets/go-companion.glb");
  const robotPixels = await canvasColorRatio(canvas);
  expect(robotPixels).toBeGreaterThan(0.1);
  await chooseSettingsSelect(page, "Animation", "Focus");
  await expect(scene).toHaveAttribute("data-render-state", "ready");
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

test("drags the desktop pet, persists position, and closes with per-device recovery", async ({ page }, testInfo) => {
  await page.goto("/webui/v2/settings/pet");
  await expect(page.getByRole("heading", { name: "Pet", exact: true })).toBeVisible();
  await page.getByLabel("Enable pet").check();
  await chooseSettingsSelect(page, "Model asset", "Chinese Dragon");
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.getByRole("status").filter({ hasText: "Saved and confirmed by readback." })).toBeVisible();

  await page.goto("/webui/v2");
  const pet = page.locator(".webui2-desktop-pet");
  await expect(pet).toBeVisible();
  const scene = pet.locator("[data-render-state]");
  await expect(scene).toHaveAttribute("data-render-state", "ready");
  await expect(scene).toHaveAttribute("data-asset", "/assets/pets/chinese-dragon.glb");
  expect(await canvasColorRatio(pet.locator("canvas"))).toBeGreaterThan(0.1);

  const origin = await pet.boundingBox();
  if (!origin) throw new Error("pet has no bounding box");
  const target = { x: Math.max(40, origin.x - 500), y: Math.max(40, origin.y - 350) };
  await page.mouse.move(origin.x + origin.width / 2, origin.y + origin.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x, target.y, { steps: 12 });
  await page.mouse.up();
  const dragged = await pet.boundingBox();
  if (!dragged) throw new Error("pet has no bounding box");
  expect(dragged.x).toBeLessThan(origin.x - 100);
  expect(dragged.y).toBeLessThan(origin.y - 100);
  await page.screenshot({ path: testInfo.outputPath("desktop-pet-dragged.png"), fullPage: false });

  await page.reload();
  await expect(pet).toBeVisible();
  const restored = await pet.boundingBox();
  if (!restored) throw new Error("pet has no bounding box");
  expect(Math.abs(restored.x - dragged.x)).toBeLessThan(6);
  expect(Math.abs(restored.y - dragged.y)).toBeLessThan(6);

  await pet.click();
  const menu = page.getByRole("menu", { name: "Pet options" });
  await expect(menu).toBeVisible();
  await expect(menu.getByRole("menuitem")).toHaveCount(3);
  await page.screenshot({ path: testInfo.outputPath("desktop-pet-menu.png"), fullPage: false });
  await menu.getByRole("menuitem", { name: "Close pet" }).click();
  await expect(page.locator(".webui2-desktop-pet")).toHaveCount(0);

  await page.goto("/webui/v2/settings/pet");
  await expect(page.getByText("The pet is closed on this device.")).toBeVisible();
  await page.getByRole("button", { name: "Show on this device" }).click();
  await page.goto("/webui/v2");
  await expect(page.locator(".webui2-desktop-pet")).toBeVisible();
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

async function chooseSettingsSelect(page: import("@playwright/test").Page, label: string, option: string): Promise<void> {
  await page.getByRole("button", { name: label, exact: true }).click();
  await page.getByRole("option", { name: option, exact: true }).click();
}

async function canvasColorRatio(canvas: import("@playwright/test").Locator): Promise<number> {
  return canvas.evaluate((node) => {
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
}
