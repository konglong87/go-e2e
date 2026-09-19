import { expect, test, type Page } from "@playwright/test";

const THINKING_PREFERENCE_KEY = "golang-cc-webui.v2.thinking-mode";

async function setThinkingMode(page: Page, mode: "full" | "summary" | "hidden") {
  const control = page.getByRole("combobox", { name: "Thinking", exact: true });
  if (await control.count()) {
    await control.selectOption(mode);
    return;
  }
  await page.evaluate(({ key, value }) => {
    window.localStorage.setItem(key, value);
    window.dispatchEvent(new StorageEvent("storage", { key, newValue: value }));
  }, { key: THINKING_PREFERENCE_KEY, value: mode });
}

test("long history preserves clickable folded records and scroll ownership", async ({ page }) => {
  const session = { id: 999999, ref: "tenant:message-experience", source: "tenant", title: "Renderer fixture", status: "completed", updated_at: "2026-09-06T00:00:00Z", short_id: "fixture" };
  const events: Array<{ id: number; task_id: number; event_type: string; payload_json: string; created_at: string }> = [];
  const event = (task: number, type: string, payload: object) => events.push({ id: events.length + 1, task_id: task, event_type: type, payload_json: JSON.stringify(payload), created_at: session.updated_at });
  for (let index = 1; index <= 35; index++) {
    event(index, "message", { content: `Historical request ${index}` });
    event(index, "completed", { response: `Historical response ${index}` });
  }
  event(36, "thinking_delta", { content: "Checking the source before responding." });
  event(36, "tool_call", { tool_id: "read-1", tool_name: "Read", input: { file_path: "/workspace/example.ts" } });
  event(36, "tool_result", { tool_id: "read-1", tool_name: "Read", output: "export const ready = true;" });
  event(36, "completed", { response: "Final response", model: "fixture-model" });
  const data = { schema_version: "golang-cc.session-conversation.v1", session, events, cursor: String(events.length), has_more: false };
  await page.addInitScript(() => localStorage.setItem("golang-cc-webui.language.v1", "en"));
  await page.route("**/tenant/session-control/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/conversations/stream")) return route.fulfill({ contentType: "text/event-stream", body: `event: conversation\ndata: ${JSON.stringify(data)}\n\n` });
    if (url.pathname.endsWith("/conversation")) return route.fulfill({ json: { data } });
    return route.fulfill({ json: { data: url.searchParams.get("source") === "local" ? [] : [session] } });
  });
  await page.route("**/tenant/agent-tasks/*/pending-inputs", (route) => route.fulfill({ json: { data: [] } }));
  await page.goto(`/webui/v2/sessions/${encodeURIComponent(session.ref)}?token=test-token`);
  const stream = page.locator(".webui2-conversation-stream");
  await expect(page.getByText("Final response", { exact: true })).toBeInViewport();
  await expect.poll(() => stream.evaluate((node) => node.scrollTop)).toBeGreaterThan(0);
  await stream.evaluate((node) => { node.scrollTop = 0; });
  await expect(page.getByRole("button", { name: "Jump to latest", exact: true })).toBeVisible();
  await page.getByRole("button", { name: /Show earlier messages/ }).click();
  await expect(page.getByText("Historical request 1", { exact: true })).toBeAttached();
  await setThinkingMode(page, "full");
  await expect(page.getByText("Checking the source before responding.", { exact: true })).toBeAttached();
  await setThinkingMode(page, "hidden");
  await setThinkingMode(page, "summary");
  const tool = page.locator(".webui2-tool-message");
  const summary = tool.locator("summary");
  await summary.scrollIntoViewIfNeeded();
  const geometry = await summary.evaluate((node) => {
    const rect = node.getBoundingClientRect();
    const parentRect = node.parentElement!.getBoundingClientRect();
    return { summaryHeight: rect.height, parentHeight: parentRect.height, hit: node.contains(document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2)) };
  });
  expect(geometry.parentHeight).toBeGreaterThanOrEqual(geometry.summaryHeight);
  expect(geometry.hit).toBe(true);
  await summary.click();
  await expect(tool).toHaveAttribute("open", "");
  await expect(tool.locator("pre").filter({ hasText: "export const ready" })).toBeVisible();
  await summary.click();
  await stream.evaluate((node) => { node.scrollTop = 0; });
  await page.getByRole("button", { name: "Jump to latest", exact: true }).click();
  await expect(page.getByText("Final response", { exact: true })).toBeInViewport();
  await expect.poll(() => stream.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop)).toBeLessThan(2);
});

test("conversation cards and composer share one content width without decorative rules", async ({ page }, testInfo) => {
  const session = { id: 1000000, ref: "tenant:message-width", source: "tenant", title: "Width fixture", status: "completed", updated_at: "2026-09-06T00:00:00Z", short_id: "width" };
  const events = [
    { id: 1, task_id: 1, event_type: "message", payload_json: JSON.stringify({ content: "Check the shared content width." }), created_at: session.updated_at },
    { id: 2, task_id: 1, event_type: "thinking_delta", payload_json: JSON.stringify({ content: "Thinking content stays inside the same column." }), created_at: session.updated_at },
    { id: 3, task_id: 1, event_type: "completed", payload_json: JSON.stringify({ response: "The width is aligned." }), created_at: session.updated_at }
  ];
  const data = { schema_version: "golang-cc.session-conversation.v1", session, events, cursor: String(events.length), has_more: false };
  await page.addInitScript(() => localStorage.setItem("golang-cc-webui.language.v1", "en"));
  await page.route("**/tenant/session-control/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/conversations/stream")) return route.fulfill({ contentType: "text/event-stream", body: `event: conversation\ndata: ${JSON.stringify(data)}\n\n` });
    if (url.pathname.endsWith("/conversation")) return route.fulfill({ json: { data } });
    return route.fulfill({ json: { data: url.searchParams.get("source") === "local" ? [] : [session] } });
  });
  await page.route("**/tenant/agent-tasks/*/pending-inputs", (route) => route.fulfill({ json: { data: [] } }));
  await page.goto(`/webui/v2/sessions/${encodeURIComponent(session.ref)}?token=test-token`);
  await setThinkingMode(page, "summary");
  await expect(page.locator(".webui2-thinking-surface")).toBeVisible();

  const geometry = await page.evaluate(() => {
    const rect = (selector: string) => {
      const element = document.querySelector<HTMLElement>(selector);
      if (!element) throw new Error(`missing ${selector}`);
      const box = element.getBoundingClientRect();
      const styles = getComputedStyle(element);
      return { left: box.left, right: box.right, width: box.width, borderTopStyle: styles.borderTopStyle };
    };
    return {
      thinking: rect(".webui2-thinking-surface"),
      composer: rect(".webui2-composer"),
      shell: rect(".webui2-composer-shell"),
      footer: rect(".webui2-message-footer")
    };
  });

  expect(Math.abs(geometry.thinking.left - geometry.composer.left)).toBeLessThanOrEqual(1);
  expect(Math.abs(geometry.thinking.right - geometry.composer.right)).toBeLessThanOrEqual(1);
  expect(Math.abs(geometry.composer.left - geometry.shell.left)).toBeLessThanOrEqual(1);
  expect(Math.abs(geometry.composer.right - geometry.shell.right)).toBeLessThanOrEqual(1);
  expect(geometry.composer.borderTopStyle).toBe("none");
  expect(geometry.footer.borderTopStyle).toBe("none");
  await page.screenshot({ path: testInfo.outputPath("conversation-width-aligned.png"), fullPage: false });
});
