const canonicalPrefix = "golang-cc-webui.";
const legacyPrefix = "go-claude-webui.";

export function readProductStorage(key: string): string | null {
  const current = window.localStorage.getItem(key);
  if (current !== null || !key.startsWith(canonicalPrefix)) {
    return current;
  }
  const legacyKey = `${legacyPrefix}${key.slice(canonicalPrefix.length)}`;
  const legacy = window.localStorage.getItem(legacyKey);
  if (legacy !== null) {
    window.localStorage.setItem(key, legacy);
  }
  return legacy;
}
