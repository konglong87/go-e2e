#!/usr/bin/env bash

# Promote deprecated product environment variables to the canonical prefix.
# Canonical values always win, and are mirrored back so Go packages that have
# not yet moved off the compatibility prefix still work in direct test runs.
golang_cc_promote_env() {
  local legacy_prefix="GOLANG_CLAUDE_CODE_"
  local canonical_prefix="GOLANG_CC_"
  local legacy_name canonical_name value

  while IFS= read -r legacy_name; do
    canonical_name="${canonical_prefix}${legacy_name#${legacy_prefix}}"
    if ! declare -p "${canonical_name}" >/dev/null 2>&1; then
      printf -v "${canonical_name}" '%s' "${!legacy_name}"
      export "${canonical_name}"
    fi
  done < <(compgen -A variable "${legacy_prefix}")

  while IFS= read -r canonical_name; do
    legacy_name="${legacy_prefix}${canonical_name#${canonical_prefix}}"
    value="${!canonical_name}"
    printf -v "${legacy_name}" '%s' "${value}"
    export "${legacy_name}"
  done < <(compgen -A variable "${canonical_prefix}")
}

golang_cc_promote_env
