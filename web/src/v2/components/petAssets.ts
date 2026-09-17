export const PET_ASSETS = {
  "go-companion": {
    url: "/assets/pets/go-companion.glb",
    poster: "/assets/pets/go-companion.png",
    label: "Go Companion"
  }
} as const;

export function petAsset(model: string) {
  return PET_ASSETS[model as keyof typeof PET_ASSETS] ?? PET_ASSETS["go-companion"];
}
