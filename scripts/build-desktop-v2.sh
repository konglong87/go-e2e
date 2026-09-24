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
    mkdir -p "${helper_dir}"
    MACOS_ARCH="${helper_arch}" bash "${ROOT}/scripts/build-computer-helper-macos.sh" "${helper_dir}/computer-helper-macos" >/dev/null
    # The service/helper binaries are embedded after Wails creates its app signature.
    # Sign nested binaries first, then seal the outer app bundle again.
    codesign --force --sign - --timestamp=none "${APP_BIN}/go-e2e"
    codesign --force --sign - --timestamp=none "${helper_dir}/computer-helper-macos"
    codesign --force --deep --sign - --timestamp=none "${APP_PATH}"
    codesign --verify --deep --strict --verbose=2 "${APP_PATH}"
  fi
  # Keep Finder's package modification time aligned with the actual build.
  touch "${APP_PATH}"
fi
