#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="${ROOT}/desktop-v2"

VITE_DESKTOP_UI_VERSION=2 npm --prefix "${ROOT}/web" run build -- --mode desktop-v2
FRONTEND_DIST="${DESKTOP_DIR}/frontend/dist"
mkdir -p "${FRONTEND_DIST}"
# Refresh generated assets without deleting the Git-tracked embed placeholder.
find "${FRONTEND_DIST}" -mindepth 1 -maxdepth 1 ! -name ".gitkeep" -exec rm -rf -- {} +
if [[ ! -f "${FRONTEND_DIST}/.gitkeep" ]]; then
  printf '%s\n' \
    'Go embed placeholder for fresh-clone checks only; not a frontend.' \
    'Build the real desktop UI with scripts/build-desktop-v2.sh before running Wails.' \
    > "${FRONTEND_DIST}/.gitkeep"
fi
rm -f "${DESKTOP_DIR}/golang-cc"
cp -R "${ROOT}/web/dist/." "${FRONTEND_DIST}/"

LDFLAGS="-s -w" OUTPUT="${DESKTOP_DIR}/go-e2e" \
  "${ROOT}/scripts/build.sh" >/dev/null
(
  cd "${DESKTOP_DIR}"
  wails build -s "$@"
)

APP_BIN="${DESKTOP_DIR}/build/bin/go-e2e.app/Contents/MacOS"
APP_PATH="${DESKTOP_DIR}/build/bin/go-e2e.app"
# Ad-hoc signatures otherwise designate the binary by cdhash, so every local
# rebuild invalidates the user's TCC grant. A stable designated requirement
# keeps the local development identity consistent until a real Apple identity
# is supplied for release signing.
MACOS_ADHOC_DESIGNATED_REQUIREMENT="${MACOS_ADHOC_DESIGNATED_REQUIREMENT:-=designated => identifier \"com.wails.go-e2e\"}"
macos_codesign() {
  local target="$1"
  codesign --force --identifier "com.wails.go-e2e" --sign - --requirements "${MACOS_ADHOC_DESIGNATED_REQUIREMENT}" --timestamp=none "${target}"
}
if [[ -d "${APP_BIN}" ]]; then
  cp "${DESKTOP_DIR}/go-e2e" "${APP_BIN}/go-e2e"
  rm -f "${APP_BIN}/golang-cc"

  if [[ "$(uname -s)" == "Darwin" ]]; then
    helper_arch="$(uname -m)"
    for arg in "$@"; do
      case "${arg}" in
        darwin/arm64|darwin/amd64) helper_arch="${arg#darwin/}" ;;
      esac
    done
    helper_dir="${APP_PATH}/Contents/Helpers"
    # Remove the pre-bundle helper path from older builds; leaving it behind
    # would keep an unsigned/stale resource in the outer app seal.
    rm -f "${helper_dir}/computer-helper-macos"
    helper_app="${helper_dir}/ComputerHelper.app"
    mkdir -p "${helper_app}/Contents/MacOS"
    MACOS_ARCH="${helper_arch}" bash "${ROOT}/scripts/build-computer-helper-macos.sh" "${helper_app}/Contents/MacOS/computer-helper-macos" >/dev/null
    # The service/helper binaries are embedded after Wails creates its app signature.
    # Sign every executable/bundle with the same stable local requirement, then
    # seal only the outer app (without --deep re-signing nested bundles).
    macos_codesign "${APP_BIN}/go-e2e-desktop"
    macos_codesign "${APP_BIN}/go-e2e"
    macos_codesign "${helper_app}"
    # Attest the exact committed source and executable bytes before sealing.
    python3 - "${APP_PATH}" "${ROOT}" <<'PYBUILD'
import datetime, hashlib, json, pathlib, subprocess, sys
app, root = map(pathlib.Path, sys.argv[1:])
def digest(path):
    sha = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            sha.update(chunk)
    return sha.hexdigest()
commit = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
status = subprocess.check_output(["git", "-C", str(root), "status", "--porcelain=v1", "--untracked-files=all"], text=True)
paths = []
for line in status.splitlines():
    value = line[3:] if len(line) >= 4 else ""
    if " -> " in value:
        value = value.rsplit(" -> ", 1)[-1]
    if value:
        paths.append(value)
web_dirty = bool(paths) and all(path == "web" or path.startswith("web/") for path in paths)
runtime_dirty = bool(paths) and not web_dirty
resources = app / "Contents/Resources"
resources.mkdir(exist_ok=True)
(resources / "computer-use-build.json").write_text(json.dumps({
    "schema_version": "computer-use-build.v1", "source_commit": commit,
    # Web UI-only edits do not alter the Computer Use runtime/service/helper
    # bytes. Keep that fact explicit instead of mixing UI work into runtime
    # evidence or silently pretending the entire workspace is clean.
    "source_dirty": runtime_dirty, "web_source_dirty": web_dirty,
    "built_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    # Outer app sealing re-signs its main executable; a full SHA here would be circular.
    "desktop_build_id": subprocess.check_output(["go", "tool", "buildid", str(app / "Contents/MacOS/go-e2e-desktop")], text=True).strip(),
    "helper_sha256": digest(app / "Contents/Helpers/ComputerHelper.app/Contents/MacOS/computer-helper-macos"),
    "service_sha256": digest(app / "Contents/MacOS/go-e2e"),
}, indent=2) + "\n")
PYBUILD
    codesign --force --sign - --requirements "${MACOS_ADHOC_DESIGNATED_REQUIREMENT}" --timestamp=none "${APP_PATH}"
    codesign --verify --deep --strict --verbose=2 "${APP_PATH}"
  fi
  # Keep Finder's package modification time aligned with the actual build.
  touch "${APP_PATH}"
fi
