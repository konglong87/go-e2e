import { expect, test } from "@playwright/test";
import { installWebUIV2Sessions } from "./fixtures/webuiV2Sessions";

test.beforeEach(async ({ page }) => {
  await installWebUIV2Sessions(page);
  await page.route("**/runtime/settings**", (route) => route.fulfill({
    status: 503, json: { error: "Settings unavailable in this navigation fixture" }
  }));
});

test("first launch returns from model setup to a valid index and survives reload", async ({ page }, testInfo) => {
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: "go-e2e", exact: true })).toBeVisible();
  await expect(page.locator(".webui2-route-error")).toHaveCount(0);
  await page.getByRole("button", { name: "Configure model first" }).click();
  await expect(page).toHaveURL(/\/webui\/v2\/settings\/models$/);
  await page.goBack();
  await expect(page.locator(".webui2-empty-state")).toBeVisible();
  await expect(page.locator(".webui2-route-error")).toHaveCount(0);
  await page.reload();
  await expect(page.locator(".webui2-empty-state")).toBeVisible();
  await expect(page.getByRole("dialog", { name: "go-e2e", exact: true })).toHaveCount(0);
  await expect(page.locator(".webui2-route-error")).toHaveCount(0);
  await expect(page.locator(".webui2-empty-state img")).toBeVisible();
  expect(await page.locator(".webui2-empty-state img").evaluate((image: HTMLImageElement) => image.naturalWidth)).toBeGreaterThan(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(page.viewportSize()!.width);
  await page.screenshot({ path: testInfo.outputPath("desktop-root.png") });
  await page.locator(".webui2-empty-state").getByRole("button", { name: "New session" }).click();
  await expect(page).toHaveURL(/\/webui\/v2\/settings\/models$/);
});

test("existing sessions and invalid deep links retain their navigation semantics", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("go-e2e.desktop.onboarding.v1", "done"));
  await page.goto("/");
  await expect(page.locator(".webui2-empty-state")).toBeVisible();
  const launcher = page.getByRole("button", { name: "Open sessions" });
  if (await launcher.isVisible()) await launcher.click();
  await page.locator(".webui2-session-select").filter({ hasText: "Design review" }).click();
  await expect(page).toHaveURL(/\/sessions\/tenant%3Abeta$/);
  await expect(page.locator(".webui2-conversation-message--assistant")).toContainText("The responsive review is complete.");
  await page.goBack();
  await expect(page.locator(".webui2-empty-state")).toBeVisible();
  await expect(page.locator(".webui2-route-error")).toHaveCount(0);
  await page.goto("/webui/v2/sessions/not-a-ref");
  await expect(page.locator(".webui2-route-error")).toContainText("This session link is invalid.");
  await page.getByRole("button", { name: "Back to sessions" }).click();
  await expect(page).toHaveURL(/\/webui\/v2$/);
  await expect(page.locator(".webui2-empty-state")).toBeVisible();
});
