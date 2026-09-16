import { expect, test } from "@playwright/test";

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
  const thinking = page.getByRole("combobox", { name: "Thinking", exact: true });
  await thinking.selectOption("full");
  await expect(page.getByText("Checking the source before responding.", { exact: true })).toBeAttached();
  await thinking.selectOption("hidden");
  await thinking.selectOption("summary");
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
