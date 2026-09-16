#!/usr/bin/env bash
# Build golang-cc from this checkout and put it on PATH.
#
# Compilation is delegated to scripts/build.sh, so the installed binary reports
# the same `git describe` version a local build does. Installing must not be the
# one path where `--version` prints `dev`.
#
# It builds rather than downloads on purpose. There is no `curl | sh` variant
# because that needs a publicly downloadable artifact and this repository is not
# public; and `go install` still cannot work, even though the module path now
# matches the repository (github.com/konglong87/go-e2e), because go.mod keeps
# `replace github.com/muesli/termenv => ./third_party/termenv` and
# `go install pkg@version` ignores replace directives. Both are written up in
# README.md 安装.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="${BIN_DIR:-$PREFIX/bin}"

usage() {
  cat <<'EOF'
Usage: scripts/install.sh [-h|--help]

Builds golang-cc from this checkout (via scripts/build.sh) and installs
it as $BIN_DIR/golang-cc.

Environment:
  PREFIX   install prefix, binary goes to $PREFIX/bin (default: $HOME/.local)
  BIN_DIR  target directory, overrides $PREFIX/bin
  VERSION  version to inject (default: git describe --tags --always --dirty)
EOF
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
  "") ;;
  *)
    echo "install.sh: unknown argument: $1" >&2
    usage >&2
    exit 2
    ;;
esac

# Two Go floors, and they do not mean the same thing:
#   go.mod's `go` directive is the hard one — below it the compile itself fails.
#   .tool-versions is the security floor (crypto/tls GO-2026-5856) and what CI
#   pins, so falling short of it only warns: the build succeeds, the binary just
#   carries a known stdlib CVE. Both are read from their files so this script
#   never becomes a third place where a Go version is written down.
min_version="$(awk '$1 == "go" { print $2; exit }' go.mod)"
pinned_version="$(awk '$1 == "golang" { print $2; exit }' .tool-versions)"

# True when $1 <= $2, comparing dotted versions field by field.
version_le() {
  [ "$(printf '%s\n%s\n' "$1" "$2" | sort -t. -k1,1n -k2,2n -k3,3n | head -1)" = "$1" ]
}

if ! command -v go >/dev/null 2>&1; then
  echo "install failed: no 'go' on PATH." >&2
  echo "  golang-cc is built from source; install Go $pinned_version or newer and re-run." >&2
  exit 1
fi

go_version="$(go env GOVERSION | sed -e 's/^go//' -e 's/[^0-9.].*$//')"
if [ -z "$go_version" ]; then
  # Say what we actually saw. An empty version in the message below would read as
  # a bug in this script rather than an unusual toolchain (a devel build, say).
  echo "install failed: no version number in 'go env GOVERSION' ($(go env GOVERSION))." >&2
  exit 1
fi
if ! version_le "$min_version" "$go_version"; then
  echo "install failed: Go $go_version is below go.mod's minimum $min_version." >&2
  exit 1
fi
if ! version_le "$pinned_version" "$go_version"; then
  echo "warning: Go $go_version is older than the pinned $pinned_version (.tool-versions)." >&2
  echo "         The build will succeed, but crypto/tls GO-2026-5856 is only fixed in $pinned_version," >&2
  echo "         so the binary you install here still carries it." >&2
fi

# Checked before building, not after: the build takes tens of seconds, and
# finding out then that the target directory is read-only wastes all of it.
if ! mkdir -p "$BIN_DIR" 2>/dev/null || [ ! -w "$BIN_DIR" ]; then
  echo "install failed: $BIN_DIR is not writable." >&2
  echo "  Set PREFIX or BIN_DIR to a writable location and re-run." >&2
  exit 1
fi

"$ROOT_DIR/scripts/build.sh"

installed="$BIN_DIR/golang-cc"
install -m 0755 "$ROOT_DIR/bin/golang-cc" "$installed"

printf 'installed %s\n          %s\n' "$installed" "$("$installed" --version)"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo
    echo "$BIN_DIR is not on your PATH. Add it, for example:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac
