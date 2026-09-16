#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
REVISION="${REVISION:-$(git rev-parse HEAD 2>/dev/null || true)}"
if [ -z "${DIRTY+x}" ]; then
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    if git diff-index --quiet HEAD -- 2>/dev/null; then
      DIRTY=false
    else
      DIRTY=true
    fi
  else
    DIRTY=""
  fi
fi
if [ -z "${BUILD_TIME:-}" ]; then
  if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    if date -u -r "$SOURCE_DATE_EPOCH" '+%Y-%m-%dT%H:%M:%SZ' >/dev/null 2>&1; then
      BUILD_TIME="$(date -u -r "$SOURCE_DATE_EPOCH" '+%Y-%m-%dT%H:%M:%SZ')"
    else
      BUILD_TIME="$(date -u -d "@$SOURCE_DATE_EPOCH" '+%Y-%m-%dT%H:%M:%SZ')"
    fi
  else
    BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  fi
fi
OUTPUT="${OUTPUT:-bin/golang-cc}"
PACKAGE="${PACKAGE:-./cmd/golang-cc}"
LDFLAGS="${LDFLAGS:-}"
BUILDINFO_PACKAGE="github.com/konglong87/go-e2e/internal/buildinfo"

mkdir -p "$(dirname "$OUTPUT")"

go build \
  -trimpath \
  -ldflags "${LDFLAGS} -X ${BUILDINFO_PACKAGE}.Version=${VERSION} -X ${BUILDINFO_PACKAGE}.Revision=${REVISION} -X ${BUILDINFO_PACKAGE}.Dirty=${DIRTY} -X ${BUILDINFO_PACKAGE}.BuildTime=${BUILD_TIME}" \
  -o "$OUTPUT" \
  "$PACKAGE"

printf 'built %s (%s)\n' "$OUTPUT" "$VERSION"
