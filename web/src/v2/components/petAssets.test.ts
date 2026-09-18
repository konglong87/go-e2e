import { describe, expect, it } from "vitest";
import { PET_ASSETS, petAsset, petAssetOptions } from "./petAssets";

describe("pet assets", () => {
  it("registers the robot and Chinese dragon with reproducible GLB and poster assets", () => {
    expect(Object.keys(PET_ASSETS)).toEqual(["go-companion", "chinese-dragon"]);
    expect(PET_ASSETS["chinese-dragon"]).toMatchObject({
      url: "/assets/pets/chinese-dragon.glb",
      poster: "/assets/pets/chinese-dragon.png",
      label: { en: "Chinese Dragon", zh: "中国龙" }
    });
  });

  it("falls back to the default model and localizes selection options", () => {
    expect(petAsset("missing")).toBe(PET_ASSETS["go-companion"]);
    expect(petAssetOptions("zh")).toEqual([
      { value: "go-companion", label: "Go 伙伴" },
      { value: "chinese-dragon", label: "中国龙" }
    ]);
  });
});

