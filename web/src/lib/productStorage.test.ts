import { beforeEach, describe, expect, it } from "vitest";
import { readProductStorage } from "./productStorage";

describe("product storage migration", () => {
  beforeEach(() => {
    const storage = new Map<string, string>();
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        getItem: (key: string) => storage.get(key) ?? null,
        setItem: (key: string, value: string) => storage.set(key, value),
        removeItem: (key: string) => storage.delete(key),
        clear: () => storage.clear()
      }
    });
  });

  it("prefers the canonical key", () => {
    window.localStorage.setItem("golang-cc-webui.language.v1", "zh");
    window.localStorage.setItem("go-claude-webui.language.v1", "en");

    expect(readProductStorage("golang-cc-webui.language.v1")).toBe("zh");
  });

  it("copies a legacy value to the canonical key", () => {
    window.localStorage.setItem("go-claude-webui.language.v1", "zh");

    expect(readProductStorage("golang-cc-webui.language.v1")).toBe("zh");
    expect(window.localStorage.getItem("golang-cc-webui.language.v1")).toBe("zh");
  });
});
