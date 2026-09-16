#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="${ROOT}/desktop"

npm --prefix "${ROOT}/web" run build -- --mode desktop
mkdir -p "${DESKTOP_DIR}/frontend/dist"
cp -R "${ROOT}/web/dist/." "${DESKTOP_DIR}/frontend/dist/"

go build -o "${DESKTOP_DIR}/golang-cc" "${ROOT}/cmd/golang-cc"
(
  cd "${DESKTOP_DIR}"
  wails build -s "$@"
)

APP_BIN="${DESKTOP_DIR}/build/bin/golang-cc-desktop.app/Contents/MacOS"
if [[ -d "${APP_BIN}" ]]; then
  cp "${DESKTOP_DIR}/golang-cc" "${APP_BIN}/golang-cc"
fi
