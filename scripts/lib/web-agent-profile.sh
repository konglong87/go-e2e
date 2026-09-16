#!/usr/bin/env bash

web_agent_profile_apply() {
  local profile="${1:-${GOLANG_CC_WEB_AGENT_PROFILE:-manual}}"
  case "${profile}" in
    manual)
      : "${GOLANG_CC_WEB_AGENT_DB:=golang_cc_webui_local}"
      : "${GOLANG_CC_WEB_AGENT_TENANT:=yutang}"
      : "${GOLANG_CC_WEB_AGENT_USER:=feishu-e2e-user}"
      ;;
    e2e)
      : "${GOLANG_CC_WEB_AGENT_DB:=golang_cc_web_agent_real_e2e}"
      : "${GOLANG_CC_WEB_AGENT_TENANT:=webui-local}"
      : "${GOLANG_CC_WEB_AGENT_USER:=webui-local-user}"
      ;;
    *)
      echo "web-agent: unknown profile '${profile}'; expected manual or e2e" >&2
      return 2
      ;;
  esac

  export GOLANG_CC_WEB_AGENT_PROFILE="${profile}"
  export GOLANG_CC_WEB_AGENT_DB
  export GOLANG_CC_WEB_AGENT_TENANT
  export GOLANG_CC_WEB_AGENT_USER
}

web_agent_profile_database_from_dsn() {
  local dsn="$1"
  local database="${dsn#*/}"
  database="${database%%\?*}"
  printf '%s' "${database}"
}

web_agent_profile_validate_dsn() {
  local dsn="$1"
  local expected_database="$2"
  local actual_database
  actual_database="$(web_agent_profile_database_from_dsn "${dsn}")"
  if [[ -z "${actual_database}" || "${actual_database}" == "${dsn}" ]]; then
    echo "web-agent: cannot determine database from MySQL DSN" >&2
    return 2
  fi
  if [[ "${actual_database}" != "${expected_database}" ]]; then
    echo "web-agent: profile database mismatch: profile=${expected_database} dsn=${actual_database}; set GOLANG_CC_WEB_AGENT_MYSQL_DSN or unset GOLANG_CC_MYSQL_DSN" >&2
    return 2
  fi
}
