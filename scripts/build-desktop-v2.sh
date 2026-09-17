#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="${ROOT}/desktop-v2"

VITE_DESKTOP_UI_VERSION=2 npm --prefix "${ROOT}/web" run build -- --mode desktop-v2
rm -rf "${DESKTOP_DIR}/frontend/dist"
mkdir -p "${DESKTOP_DIR}/frontend/dist"
cp -R "${ROOT}/web/dist/." "${DESKTOP_DIR}/frontend/dist/"

go build -o "${DESKTOP_DIR}/go-e2e" "${ROOT}/cmd/go-e2e"
cp "${DESKTOP_DIR}/go-e2e" "${DESKTOP_DIR}/golang-cc"
(
  cd "${DESKTOP_DIR}"
  wails build -s "$@"
)

APP_BIN="${DESKTOP_DIR}/build/bin/go-e2e.app/Contents/MacOS"
if [[ -d "${APP_BIN}" ]]; then
  cp "${DESKTOP_DIR}/go-e2e" "${APP_BIN}/go-e2e"
  cp "${DESKTOP_DIR}/go-e2e" "${APP_BIN}/golang-cc"
fi
