#!/usr/bin/env bash

# Promote product environment variables to the canonical prefix.
# Canonical values always win, and are mirrored back so Go packages that have
# not yet moved off the compatibility prefix still work in direct script runs.
golang_cc_promote_env() {
  local canonical_prefix="GO_E2E_"
  local legacy_prefix legacy_name canonical_name value
  local -a compatibility_prefixes=(
    "GOLANG_CC_"
    "GOLANG_CLAUDE_CODE_"
    "GO_CLAUDE_CODE_"
    "GO_CLAUDE_"
  )

  for legacy_prefix in "${compatibility_prefixes[@]}"; do
    while IFS= read -r legacy_name; do
      if [[ "${legacy_prefix}" == "GO_CLAUDE_" && "${legacy_name}" == GO_CLAUDE_CODE_* ]]; then
        continue
      fi
      canonical_name="${canonical_prefix}${legacy_name#${legacy_prefix}}"
      if ! declare -p "${canonical_name}" >/dev/null 2>&1; then
        printf -v "${canonical_name}" '%s' "${!legacy_name}"
        export "${canonical_name}"
      fi
    done < <(compgen -A variable "${legacy_prefix}")
  done

  while IFS= read -r canonical_name; do
    value="${!canonical_name}"
    for legacy_prefix in "${compatibility_prefixes[@]}"; do
      legacy_name="${legacy_prefix}${canonical_name#${canonical_prefix}}"
      printf -v "${legacy_name}" '%s' "${value}"
      export "${legacy_name}"
    done
  done < <(compgen -A variable "${canonical_prefix}")
}

golang_cc_promote_env
