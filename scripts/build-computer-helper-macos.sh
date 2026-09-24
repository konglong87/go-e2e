#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${1:-$ROOT_DIR/desktop-v2/build/bin/computer-helper-macos}"
mkdir -p "$(dirname "$OUTPUT")"
swiftc -O \
  -target "${MACOS_ARCH:-$(uname -m)}-apple-macosx${MACOSX_DEPLOYMENT_TARGET:-14.0}" \
  -framework AppKit \
  -framework ApplicationServices \
  -framework CoreGraphics \
  -framework Foundation \
  -framework ImageIO \
  -framework UniformTypeIdentifiers \
  "$ROOT_DIR/native/macos/main.swift" \
  -o "$OUTPUT"
chmod 0755 "$OUTPUT"
printf '%s\n' "$OUTPUT"
