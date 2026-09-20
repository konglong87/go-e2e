#!/usr/bin/env node
// Requires Git, Node and the repository's Go toolchain; no npm install or GUI.
// Tests copy only Git-visible source into a disposable directory. Local web/dist
// and previously packaged desktop assets cannot make the fresh-clone check pass.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const desktops = ["desktop-v2"];
const go = process.env.GO ?? "go";
const run = (command, args, cwd = root, env = {}) => {
  const result = spawnSync(command, args, {
    cwd, env: { ...process.env, ...env }, encoding: "utf8", timeout: 120_000,
    maxBuffer: 10 * 1024 * 1024,
  });
  assert.ifError(result.error);
  return result;
};
const checked = (command, args, cwd = root, env = {}) => {
  const result = run(command, args, cwd, env);
  assert.equal(result.status, 0, `${command} ${args.join(" ")}\n${result.stdout}${result.stderr}`);
  return result.stdout;
};

test("only embed placeholders, not generated frontend payloads, are Git-visible", () => {
  for (const desktop of desktops) {
    const placeholder = `${desktop}/frontend/dist/.gitkeep`;
    assert.equal(run("git", ["check-ignore", "--quiet", "--no-index", placeholder]).status, 1, placeholder);
    for (const generated of ["index.html", "assets/index.js"]) {
      assert.equal(run("git", ["check-ignore", "--quiet", "--no-index", `${desktop}/frontend/dist/${generated}`]).status, 0);
    }
  }
});

test("fresh source embeds only placeholders and passes desktop build, vet and unit tests", () => {
  const fixture = mkdtempSync(path.join(tmpdir(), "go-e2e-fresh-desktop-"));
  try {
    const files = checked("git", [
      "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--",
      "go.mod", "go.sum", "third_party", ...desktops,
    ]).split("\0").filter(Boolean);
    for (const file of new Set(files)) {
      const destination = path.join(fixture, file);
      mkdirSync(path.dirname(destination), { recursive: true });
      copyFileSync(path.join(root, file), destination);
    }
    const packages = desktops.map((desktop) => `./${desktop}`);
    const env = { GOWORK: "off", GOFLAGS: "", CGO_ENABLED: "1" };
    const embedded = packages.map((pkg) =>
      JSON.parse(checked(go, ["list", "-json=EmbedFiles", pkg], fixture, env)).EmbedFiles);
    assert.deepEqual(embedded, desktops.map(() => ["frontend/dist/.gitkeep"]));
    const buildOutput = path.join(fixture, "desktop-v2-test-bin");
    checked(go, ["build", "-o", buildOutput, ...packages], fixture, env);
    checked(go, ["vet", ...packages], fixture, env);
    checked(go, ["test", ...packages, "-count=1"], fixture, env);

    // Verify the pinned Wails dependency selection, without a Linux compiler:
    // ordinary checks use no GUI libraries; production requires WebKit 4.1.
    const linuxEnv = { ...env, GOOS: "linux", GOARCH: "amd64" };
    const deps = ["-deps", "-f", '{{join .CgoPkgConfig " "}}', ...packages];
    assert.doesNotMatch(checked(go, ["list", ...deps], fixture, linuxEnv), /gtk|webkit/);
    const nativeDeps = checked(go, ["list", "-tags=production,webkit2_41", ...deps], fixture, linuxEnv);
    assert.match(nativeDeps, /gtk\+-3\.0/);
    assert.match(nativeDeps, /webkit2gtk-4\.1/);
    assert.doesNotMatch(nativeDeps, /webkit2gtk-4\.0/);

    // Negative control: reproduce the original missing embed directory failure.
    for (const desktop of desktops) {
      rmSync(path.join(fixture, desktop, "frontend/dist"), { recursive: true });
    }
    const missing = run(go, ["list", ...packages], fixture, env);
    assert.notEqual(missing.status, 0);
    assert.match(missing.stderr, /pattern all:frontend\/dist: no matching files found/);
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});

test("CI separates source-only checks from real native builds and follows main", () => {
  const ci = readFileSync(path.join(root, ".github/workflows/ci.yml"), "utf8");
  const native = ci.slice(ci.indexOf("  desktop-linux:"), ci.indexOf("\n  scripts:"));
  assert.match(native, /runs-on: ubuntu-24\.04/);
  assert.match(native, /CGO_ENABLED: '1'/);
  assert.match(native, /script: \[build-desktop-v2\.sh\]/);
  assert.match(native, /libgtk-3-dev libwebkit2gtk-4\.1-dev/);
  assert.match(native, /-tags webkit2_41/);
  assert.match(native, /npm ci --legacy-peer-deps/);
  assert.match(native, /wails@v2\.10\.2/);
  assert.match(ci, /node --test scripts\/test-desktop-ci-prerequisites\.mjs/);
  const windows = readFileSync(path.join(root, ".github/workflows/desktop-windows.yml"), "utf8");
  assert.match(windows, /branches:\s+- main/);
});
