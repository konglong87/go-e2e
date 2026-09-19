#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="${ROOT}/desktop-v2"

VITE_DESKTOP_UI_VERSION=2 npm --prefix "${ROOT}/web" run build -- --mode desktop-v2
rm -rf "${DESKTOP_DIR}/frontend/dist"
rm -f "${DESKTOP_DIR}/golang-cc"
mkdir -p "${DESKTOP_DIR}/frontend/dist"
cp -R "${ROOT}/web/dist/." "${DESKTOP_DIR}/frontend/dist/"

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
    # The service binary is embedded after Wails creates its app signature.
    # Sign the nested binary first, then seal the outer app bundle again.
    codesign --force --sign - --timestamp=none "${APP_BIN}/go-e2e"
    codesign --force --deep --sign - --timestamp=none "${APP_PATH}"
    codesign --verify --deep --strict --verbose=2 "${APP_PATH}"
  fi
  # Keep Finder's package modification time aligned with the actual build.
  touch "${APP_PATH}"
fi
