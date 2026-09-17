#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT}/scripts/lib/web-agent-profile.sh"

assert_profile() {
  local profile="$1"
  local expected_db="$2"
  local expected_tenant="$3"
  local expected_user="$4"
  (
    unset GO_E2E_WEB_AGENT_DB GO_E2E_WEB_AGENT_TENANT GO_E2E_WEB_AGENT_USER GO_E2E_WEB_AGENT_PROFILE
    web_agent_profile_apply "${profile}"
    [[ "${GO_E2E_WEB_AGENT_PROFILE}" == "${profile}" ]]
    [[ "${GO_E2E_WEB_AGENT_DB}" == "${expected_db}" ]]
    [[ "${GO_E2E_WEB_AGENT_TENANT}" == "${expected_tenant}" ]]
    [[ "${GO_E2E_WEB_AGENT_USER}" == "${expected_user}" ]]
  )
}

assert_profile manual golang_cc_webui_local yutang feishu-e2e-user
assert_profile e2e golang_cc_web_agent_real_e2e webui-local webui-local-user

(
  GO_E2E_WEB_AGENT_DB=custom_db
  GO_E2E_WEB_AGENT_TENANT=custom-tenant
  GO_E2E_WEB_AGENT_USER=custom-user
  web_agent_profile_apply e2e
  [[ "${GO_E2E_WEB_AGENT_DB}" == "custom_db" ]]
  [[ "${GO_E2E_WEB_AGENT_TENANT}" == "custom-tenant" ]]
  [[ "${GO_E2E_WEB_AGENT_USER}" == "custom-user" ]]
)

(
  env -i \
    PATH="${PATH}" \
    HOME="${HOME}" \
    GOLANG_CC_WEB_AGENT_DB=legacy_db \
    GOLANG_CC_WEB_AGENT_TENANT=legacy-tenant \
    GOLANG_CC_WEB_AGENT_USER=legacy-user \
    bash -c '
      set -euo pipefail
      source "$1/scripts/lib/web-agent-profile.sh"
      web_agent_profile_apply e2e
      [[ "${GO_E2E_WEB_AGENT_DB}" == "legacy_db" ]]
      [[ "${GO_E2E_WEB_AGENT_TENANT}" == "legacy-tenant" ]]
      [[ "${GO_E2E_WEB_AGENT_USER}" == "legacy-user" ]]
    ' bash "${ROOT}"
)

web_agent_profile_validate_dsn 'root@tcp(127.0.0.1:3306)/golang_cc_webui_local?parseTime=true' golang_cc_webui_local
if web_agent_profile_validate_dsn 'root@tcp(127.0.0.1:3306)/golang_cc_web_agent_real_e2e?parseTime=true' golang_cc_webui_local >/dev/null 2>&1; then
  echo "expected mismatched profile DSN to fail" >&2
  exit 1
fi

if web_agent_profile_apply unsupported >/dev/null 2>&1; then
  echo "expected unsupported profile to fail" >&2
  exit 1
fi

echo "web-agent-profile tests ok"
