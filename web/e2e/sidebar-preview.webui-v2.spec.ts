import { expect, test } from "@playwright/test";
import { installWebUIV2Sessions } from "./fixtures/webuiV2Sessions";

test.beforeEach(async ({ page }) => {
  await installWebUIV2Sessions(page);
  await page.goto("/webui/v2/sessions/tenant%3Abeta?token=test-token");
  const opener = page.getByRole("button", { name: "Open sessions", exact: true });
  if (await opener.isVisible()) await opener.click();
});

test("session rows share one surface and show metadata only in an accessible preview", async ({ page }, testInfo) => {
  const row = page.locator(".webui2-session-row").filter({ hasText: "Design review" });
  const select = row.locator(".webui2-session-select");
  await expect(row).toHaveAttribute("data-selected", "true");
  await expect(row.locator("time")).toHaveCount(0);
  await expect(row.locator(".webui2-session-meta")).toHaveText("beta");
  const geometry = await row.evaluate((node) => {
    const button = node.querySelector(".webui2-session-select")!;
    const actions = node.querySelector(".webui2-session-actions")!;
    return {
      background: getComputedStyle(node).backgroundColor,
      bottomBorder: getComputedStyle(node).borderBottomWidth,
      leftBorder: getComputedStyle(button).borderLeftWidth,
      buttonBackground: getComputedStyle(button).backgroundColor,
      actionsBackground: getComputedStyle(actions).backgroundColor
    };
  });
  expect(geometry.background).not.toBe("rgba(0, 0, 0, 0)");
  expect(geometry.bottomBorder).toBe("0px");
  expect(geometry.leftBorder).toBe("0px");
  expect(geometry.buttonBackground).toBe("rgba(0, 0, 0, 0)");
  expect(geometry.actionsBackground).toBe("rgba(0, 0, 0, 0)");

  await select.focus();
  const preview = page.getByRole("tooltip");
  await expect(preview).toBeVisible();
  await expect(preview).toContainText("Updated");
  await expect(preview.locator("time")).toHaveAttribute("datetime", "2026-09-05T00:00:00.000Z");
  const box = await preview.boundingBox();
  const viewport = page.viewportSize()!;
  expect(box!.x).toBeGreaterThanOrEqual(0);
  expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width);
  expect(box!.y + box!.height).toBeLessThanOrEqual(viewport.height);
  await page.screenshot({ path: testInfo.outputPath("sidebar-preview-light.png") });
  await page.keyboard.press("Escape");
  await expect(preview).toHaveCount(0);
  await expect(select).toBeFocused();

  await row.locator(".webui2-session-actions > button").click();
  await page.getByRole("menuitem", { name: "Session details" }).click();
  await expect(preview).toBeVisible();
  await expect(row).toHaveAttribute("data-selected", "true");
  await page.keyboard.press("Escape");
  await expect(preview).toHaveCount(0);

  if (!testInfo.project.name.includes("mobile")) {
    await page.mouse.move(700, 500);
    await select.hover();
    await expect(preview).toBeVisible();
    await preview.hover();
    await expect(preview).toBeVisible();
    await page.waitForTimeout(600);
    await expect(preview).toBeVisible();
    await page.mouse.move(700, 500);
    await expect(preview).toHaveCount(0);
  }
});

test("status motion respects preferences and compact sidebar widths in both themes", async ({ page }, testInfo) => {
  const active = page.locator('.webui2-session-row [data-status="running"] svg');
  await expect(active).toHaveCSS("animation-name", "session-status-spin");
  await page.emulateMedia({ reducedMotion: "reduce" });
  await expect(active).toHaveCSS("animation-name", "none");
  await expect(page.locator('.webui2-session-row [data-status="completed"] svg')).toHaveCSS("animation-name", "none");

  for (const width of [232, 288, 380]) {
    // Set the existing resize preference, then exercise the rendered geometry.
    await page.locator(".webui2-sidebar").evaluate((node, size) => { (node as HTMLElement).style.width = `${size}px`; }, width);
    for (const theme of ["light", "dark"]) {
      await page.locator(".webui2-page").evaluate((node, value) => node.setAttribute("data-theme", value), theme);
      const rows = page.locator(".webui2-session-row");
      for (const row of await rows.all()) {
        expect(await row.evaluate((node) => node.scrollWidth <= node.clientWidth)).toBe(true);
      }
      await page.screenshot({ path: testInfo.outputPath(`sidebar-${width}-${theme}.png`) });
    }
  }
});
