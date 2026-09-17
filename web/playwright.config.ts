import { defineConfig, devices } from "@playwright/test";

const productEnv = (suffix: string): string | undefined =>
  process.env[`GO_E2E_${suffix}`] ?? process.env[`GOLANG_CC_${suffix}`] ?? process.env[`GOLANG_CLAUDE_CODE_${suffix}`];

const liveApiTarget = productEnv("WEBUI_LIVE_API_BASE");
const viteEnv = liveApiTarget ? `VITE_GOLANG_CC_API_TARGET=${liveApiTarget} ` : "";
const webPort = productEnv("WEBUI_E2E_PORT") || "5174";
const webURL = `http://127.0.0.1:${webPort}`;
const reuseExistingServer = productEnv("WEBUI_E2E_REUSE_SERVER") === "1";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  reporter: "list",
  use: {
    baseURL: webURL,
    trace: "retain-on-failure"
  },
  webServer: {
    command: `${viteEnv}npm run dev -- --host 127.0.0.1 --port ${webPort} --strictPort`,
    url: webURL,
    reuseExistingServer,
    timeout: 120_000
  },
  projects: [
    {
      name: "chromium-mock",
      testMatch: /webui\.spec\.ts/,
      use: { ...devices["Desktop Chrome"] }
    },
    {
      name: "chromium-webui-v2-desktop",
      testMatch: /(?:webui-v2|settings-v2)\.spec\.ts/,
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 960 } }
    },
    {
      name: "chromium-webui-v2-narrow",
      testMatch: /(?:webui-v2|settings-v2)\.spec\.ts/,
      use: { ...devices["Desktop Chrome"], viewport: { width: 1024, height: 768 } }
    },
    {
      name: "chromium-webui-v2-mobile",
      testMatch: /(?:webui-v2|settings-v2)\.spec\.ts/,
      use: { ...devices["Pixel 5"] }
    },
    {
      name: "chromium-smoke-desktop",
      testMatch: /navigation-smoke\.spec\.ts/,
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 960 } }
    },
    {
      name: "chromium-smoke-mobile",
      testMatch: /navigation-smoke\.spec\.ts/,
      use: { ...devices["Pixel 5"] }
    },
    {
      name: "chromium-live",
      testMatch: /live-api\.spec\.ts/,
      use: { ...devices["Desktop Chrome"] }
    }
  ]
});
