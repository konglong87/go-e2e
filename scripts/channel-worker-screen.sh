#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT}/scripts/lib/product-env.sh"
ACTION="${1:-start}"
WORKER_NAME="${GO_E2E_CHANNEL_WORKER_NAME:-feishu-e2e}"
DEFAULT_STATE_DIR="${HOME}/.golang-cc/channel-workers"
STATE_DIR="${GO_E2E_CHANNEL_WORKER_STATE_DIR:-${DEFAULT_STATE_DIR}}"
LEGACY_STATE_DIR="${GO_E2E_CHANNEL_WORKER_LEGACY_STATE_DIR:-/tmp/golang-cc-channel-workers}"
BINARY="${GO_E2E_CHANNEL_WORKER_BINARY:-${STATE_DIR}/golang-cc-channel-worker-${WORKER_NAME}}"
SCREEN_NAME="${GO_E2E_CHANNEL_WORKER_SCREEN:-golang-cc-channel-${WORKER_NAME}}"
LOG_FILE="${GO_E2E_CHANNEL_WORKER_LOG:-${STATE_DIR}/${WORKER_NAME}.log}"
ENV_FILE="${STATE_DIR}/${WORKER_NAME}.env"
WORKSPACE="${GO_E2E_CHANNEL_WORKSPACE:-${ROOT}}"
SETTINGS_FILE="${GO_E2E_CHANNEL_SETTINGS_FILE:-${GO_E2E_CHANNEL_SETTINGS:-${HOME}/.golang-cc/settings.json}}"
PROVIDER="${GO_E2E_CHANNEL_MODEL_PROVIDER:-${GO_E2E_PROVIDER:-}}"
MODEL="${GO_E2E_CHANNEL_MODEL:-${CLAUDE_CODE_MODEL:-}}"
RUNNER_FILE="${STATE_DIR}/${WORKER_NAME}.run.sh"

migrate_legacy_worker_config() {
  local legacy_env migration_file
  [[ "${STATE_DIR}" != "${LEGACY_STATE_DIR}" ]] || return 0
  [[ ! -f "${ENV_FILE}" ]] || return 0
  legacy_env="${LEGACY_STATE_DIR}/${WORKER_NAME}.env"
  [[ -f "${legacy_env}" ]] || return 0

  mkdir -p "${STATE_DIR}"
  chmod 700 "${STATE_DIR}"
  migration_file="${ENV_FILE}.migrate.$$"
  cp "${legacy_env}" "${migration_file}"
  chmod 600 "${migration_file}"
  mv "${migration_file}" "${ENV_FILE}"
  rm -f "${legacy_env}"
  echo "migrated worker config: ${ENV_FILE}"
}

migrate_legacy_worker_config

if [[ -f "${ENV_FILE}" ]]; then
  persisted_override_keys=(
    WORKSPACE
    GO_E2E_MYSQL_DSN
    GO_E2E_SQLITE_PATH
    GO_E2E_FEISHU_CREDENTIAL_FILE
    GO_E2E_CHANNEL_TENANT_ID
    GO_E2E_CHANNEL_ACCOUNT_ID
    GO_E2E_CHANNEL_ACCOUNT_KEY
    GO_E2E_CHANNEL_USER_ID
    GO_E2E_CHANNEL_PAYLOAD_KEY
    GO_E2E_CHANNEL_WORKSPACE_ROOTS
    GO_E2E_CHANNEL_PERMISSION_MODE
    GO_E2E_CHANNEL_SETTINGS
    GO_E2E_CHANNEL_SETTINGS_FILE
    GO_E2E_CONFIG_DIR
    GO_E2E_CHANNEL_MODEL_PROVIDER
    GO_E2E_CHANNEL_MODEL
    GO_E2E_CHANNEL_STREAMING
    GO_E2E_CHANNEL_STREAMING_ACCOUNT_MODE
    GO_E2E_CHANNEL_REACTIONS
    GO_E2E_CHANNEL_REDIS_ADDR
    GO_E2E_CHANNEL_REDIS_PASSWORD
    GO_E2E_CHANNEL_REDIS_PREFIX
    GO_E2E_CHANNEL_ADMIN_OPEN_IDS
    GO_E2E_CHANNEL_TOOL_DETAILS
    GO_E2E_CHANNEL_QUESTIONS
  )
  persisted_override_declarations=()
  for key in "${persisted_override_keys[@]}"; do
    if declaration="$(declare -p "${key}" 2>/dev/null)"; then
      persisted_override_declarations+=("${declaration}")
    fi
  done
  set -a
  . "${ENV_FILE}"
  set +a
  for declaration in "${persisted_override_declarations[@]}"; do
    eval "${declaration}"
  done
fi
SETTINGS_FILE="${GO_E2E_CHANNEL_SETTINGS_FILE:-${GO_E2E_CHANNEL_SETTINGS:-${HOME}/.golang-cc/settings.json}}"
PROVIDER="${GO_E2E_CHANNEL_MODEL_PROVIDER:-${GO_E2E_PROVIDER:-}}"
MODEL="${GO_E2E_CHANNEL_MODEL:-${CLAUDE_CODE_MODEL:-}}"
ACCOUNT_LOCK_ROOT="${GO_E2E_CHANNEL_ACCOUNT_LOCK_ROOT:-${STATE_DIR}/accounts}"
ACCOUNT_LOCK_DIR=""
LOCK_PERSIST=0
if [[ -n "${GO_E2E_CHANNEL_TENANT_ID:-}" && -n "${GO_E2E_CHANNEL_ACCOUNT_ID:-}" ]]; then
  ACCOUNT_LOCK_DIR="${ACCOUNT_LOCK_ROOT}/${GO_E2E_CHANNEL_TENANT_ID}-${GO_E2E_CHANNEL_ACCOUNT_ID}"
fi

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "channel-worker-screen requires $1" >&2; exit 2; }
}

require_env() {
  local key="$1"
  if [[ -z "${!key:-}" ]]; then echo "channel-worker-screen requires ${key}" >&2; exit 2; fi
}

screen_exists() {
  local sessions
  sessions="$(screen -ls 2>/dev/null || true)"
  [[ "${sessions}" == *".${SCREEN_NAME}"* ]]
}

worker_process_pids() {
  local processes
  processes="$(ps eww -axo pid=,command= 2>/dev/null || true)"
  awk -v binary="${BINARY}" '$0 ~ /channels run/ && index($0, binary) {print $1}' <<<"${processes}"
}

stop_worker() {
  if screen_exists; then
    screen -S "${SCREEN_NAME}" -X quit >/dev/null 2>&1 || true
    for _ in $(seq 1 50); do screen_exists || break; sleep 0.1; done
  fi
  rm -f "${RUNNER_FILE}"
  release_account_lock
}

account_process_pids() {
  local processes
  local marker="GO_E2E_CHANNEL_ACCOUNT_ID=${GO_E2E_CHANNEL_ACCOUNT_ID:-}"
  [[ -n "${GO_E2E_CHANNEL_ACCOUNT_ID:-}" ]] || return 0
  processes="$(ps eww -axo pid=,command= 2>/dev/null || true)"
  awk -v marker="${marker}" '$0 ~ marker && $0 ~ /channels run/ {print $1}' <<<"${processes}"
}

kill_account_processes() {
  local pids
  pids="$(account_process_pids || true)"
  [[ -z "${pids}" ]] && return 0
  while read -r pid; do
    [[ -z "${pid}" ]] || kill "${pid}" >/dev/null 2>&1 || true
  done <<<"${pids}"
  for _ in $(seq 1 50); do
    [[ -z "$(account_process_pids || true)" ]] && break
    sleep 0.1
  done
}

release_account_lock() {
  [[ -n "${ACCOUNT_LOCK_DIR}" ]] || return 0
  rm -f "${ACCOUNT_LOCK_DIR}/owner"
  rmdir "${ACCOUNT_LOCK_DIR}" >/dev/null 2>&1 || true
  LOCK_PERSIST=0
}

account_lock_is_active() {
  [[ -d "${ACCOUNT_LOCK_DIR}" ]] || return 1
  local owner_screen owner_pid owner_command
  owner_screen="$(sed -n '1p' "${ACCOUNT_LOCK_DIR}/owner" 2>/dev/null || true)"
  owner_pid="$(sed -n '3p' "${ACCOUNT_LOCK_DIR}/owner" 2>/dev/null || true)"
  if [[ -n "${owner_screen}" && "$(screen -ls 2>/dev/null || true)" == *".${owner_screen}"* ]]; then
    return 0
  fi
  if [[ "${owner_pid}" =~ ^[0-9]+$ ]] && kill -0 "${owner_pid}" >/dev/null 2>&1; then
    owner_command="$(ps -p "${owner_pid}" -o command= 2>/dev/null || true)"
    if [[ "${owner_command}" == *"channel-worker-screen.sh"* ]]; then
      return 0
    fi
  fi
  [[ -n "$(account_process_pids || true)" ]]
}

acquire_account_lock() {
  [[ -n "${ACCOUNT_LOCK_DIR}" ]] || return 0
  mkdir -p "${ACCOUNT_LOCK_ROOT}"
  if mkdir "${ACCOUNT_LOCK_DIR}" 2>/dev/null; then
    printf '%s\n%s\n%s\n' "${SCREEN_NAME}" "${WORKER_NAME}" "$$" >"${ACCOUNT_LOCK_DIR}/owner"
    return 0
  fi
  if account_lock_is_active; then
    echo "worker for account ${GO_E2E_CHANNEL_TENANT_ID}/${GO_E2E_CHANNEL_ACCOUNT_ID} already exists" >&2
    exit 1
  fi
  rm -f "${ACCOUNT_LOCK_DIR}/owner"
  rmdir "${ACCOUNT_LOCK_DIR}" >/dev/null 2>&1 || true
  mkdir "${ACCOUNT_LOCK_DIR}"
  printf '%s\n%s\n%s\n' "${SCREEN_NAME}" "${WORKER_NAME}" "$$" >"${ACCOUNT_LOCK_DIR}/owner"
}

status_worker() {
  if screen_exists; then
    local pids
    pids="$(worker_process_pids || true)"
    if [[ -z "${pids}" ]]; then
      echo "degraded: ${SCREEN_NAME} exists but worker process is not running" >&2
      return 1
    fi
    echo "running: ${SCREEN_NAME}"
    echo "binary: ${BINARY}"
    echo "log: ${LOG_FILE}"
    echo "pids: ${pids//$'\n'/ }"
    return 0
  fi
  echo "stopped: ${SCREEN_NAME}"
  return 1
}

start_worker() {
  need go
  need screen
  if [[ -z "${GO_E2E_SQLITE_PATH:-}" && -z "${GO_E2E_MYSQL_DSN:-}" ]]; then
    echo "channel-worker-screen requires GO_E2E_SQLITE_PATH or GO_E2E_MYSQL_DSN" >&2
    exit 2
  fi
  require_env GO_E2E_FEISHU_CREDENTIAL_FILE
  require_env GO_E2E_CHANNEL_TENANT_ID
  require_env GO_E2E_CHANNEL_ACCOUNT_ID
  require_env GO_E2E_CHANNEL_ACCOUNT_KEY
  require_env GO_E2E_CHANNEL_USER_ID
  mkdir -p "${STATE_DIR}"
  chmod 700 "${STATE_DIR}"
  ACCOUNT_LOCK_DIR="${ACCOUNT_LOCK_ROOT}/${GO_E2E_CHANNEL_TENANT_ID}-${GO_E2E_CHANNEL_ACCOUNT_ID}"
  if screen_exists || [[ -n "$(account_process_pids || true)" ]]; then
    echo "worker for account ${GO_E2E_CHANNEL_ACCOUNT_ID} already exists; use restart" >&2
    exit 1
  fi
  acquire_account_lock
  trap 'if [[ "${LOCK_PERSIST}" -eq 0 ]]; then release_account_lock; fi' EXIT
  go build -o "${BINARY}.tmp" ./cmd/go-e2e
  mv "${BINARY}.tmp" "${BINARY}"
  chmod 700 "${BINARY}"
  umask 077
  {
    printf 'GO_E2E_MYSQL_DSN=%q\n' "${GO_E2E_MYSQL_DSN:-}"
    printf 'GO_E2E_SQLITE_PATH=%q\n' "${GO_E2E_SQLITE_PATH:-}"
    printf 'GO_E2E_FEISHU_CREDENTIAL_FILE=%q\n' "${GO_E2E_FEISHU_CREDENTIAL_FILE}"
    printf 'GO_E2E_CHANNEL_TENANT_ID=%q\n' "${GO_E2E_CHANNEL_TENANT_ID}"
    printf 'GO_E2E_CHANNEL_ACCOUNT_ID=%q\n' "${GO_E2E_CHANNEL_ACCOUNT_ID}"
    printf 'GO_E2E_CHANNEL_ACCOUNT_KEY=%q\n' "${GO_E2E_CHANNEL_ACCOUNT_KEY}"
    printf 'GO_E2E_CHANNEL_USER_ID=%q\n' "${GO_E2E_CHANNEL_USER_ID}"
    printf 'GO_E2E_CHANNEL_PAYLOAD_KEY=%q\n' "${GO_E2E_CHANNEL_PAYLOAD_KEY}"
    printf 'WORKSPACE=%q\n' "${WORKSPACE}"
    printf 'GO_E2E_CHANNEL_WORKSPACE_ROOTS=%q\n' "${GO_E2E_CHANNEL_WORKSPACE_ROOTS:-${WORKSPACE}}"
    printf 'GO_E2E_CHANNEL_PERMISSION_MODE=%q\n' "${GO_E2E_CHANNEL_PERMISSION_MODE:-ask}"
    printf 'GO_E2E_CHANNEL_SETTINGS=%q\n' "${SETTINGS_FILE}"
    printf 'GO_E2E_CHANNEL_SETTINGS_FILE=%q\n' "${SETTINGS_FILE}"
    # Keep config-dir overrides in the screen environment so imageGeneration
    # resolution and other global settings use the same isolated profile.
    printf 'GO_E2E_CONFIG_DIR=%q\n' "${GO_E2E_CONFIG_DIR:-}"
    printf 'GO_E2E_CHANNEL_MODEL_PROVIDER=%q\n' "${PROVIDER}"
    printf 'GO_E2E_CHANNEL_MODEL=%q\n' "${MODEL}"
    printf 'GO_E2E_CHANNEL_STREAMING=%q\n' "${GO_E2E_CHANNEL_STREAMING:-}"
    printf 'GO_E2E_CHANNEL_STREAMING_ACCOUNT_MODE=%q\n' "${GO_E2E_CHANNEL_STREAMING_ACCOUNT_MODE:-}"
    printf 'GO_E2E_CHANNEL_REACTIONS=%q\n' "${GO_E2E_CHANNEL_REACTIONS:-}"
    printf 'GO_E2E_CHANNEL_REDIS_ADDR=%q\n' "${GO_E2E_CHANNEL_REDIS_ADDR:-}"
    printf 'GO_E2E_CHANNEL_REDIS_PASSWORD=%q\n' "${GO_E2E_CHANNEL_REDIS_PASSWORD:-}"
    printf 'GO_E2E_CHANNEL_REDIS_PREFIX=%q\n' "${GO_E2E_CHANNEL_REDIS_PREFIX:-}"
    printf 'GO_E2E_CHANNEL_ADMIN_OPEN_IDS=%q\n' "${GO_E2E_CHANNEL_ADMIN_OPEN_IDS:-}"
    printf 'GO_E2E_CHANNEL_TOOL_DETAILS=%q\n' "${GO_E2E_CHANNEL_TOOL_DETAILS:-}"
    printf 'GO_E2E_CHANNEL_QUESTIONS=%q\n' "${GO_E2E_CHANNEL_QUESTIONS:-}"
  } >"${ENV_FILE}"
  chmod 600 "${ENV_FILE}"
  : >"${LOG_FILE}"
  cat >"${RUNNER_FILE}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
set -a
. '${ENV_FILE}'
set +a
args=(--cwd "\${WORKSPACE}")
if [[ -n "\${GO_E2E_CHANNEL_SETTINGS:-}" && -f "\${GO_E2E_CHANNEL_SETTINGS}" ]]; then args+=(--settings "\${GO_E2E_CHANNEL_SETTINGS}"); fi
if [[ -n "\${GO_E2E_CHANNEL_MODEL_PROVIDER:-}" ]]; then args+=(--provider "\${GO_E2E_CHANNEL_MODEL_PROVIDER}"); fi
if [[ -n "\${GO_E2E_CHANNEL_MODEL:-}" ]]; then args+=(--model "\${GO_E2E_CHANNEL_MODEL}"); fi
exec '${BINARY}' "\${args[@]}" channels run >>'${LOG_FILE}' 2>&1
EOF
  chmod 700 "${RUNNER_FILE}"
  screen -dmS "${SCREEN_NAME}" "${RUNNER_FILE}"
  for _ in $(seq 1 50); do
    if screen_exists && [[ -n "$(worker_process_pids || true)" ]]; then
      break
    fi
    sleep 0.1
  done
  status_worker
  LOCK_PERSIST=1
  trap - EXIT
}

case "${ACTION}" in
  start) start_worker ;;
  stop) need screen; stop_worker; kill_account_processes; echo "stopped: ${SCREEN_NAME}" ;;
  restart) need screen; stop_worker; kill_account_processes; start_worker ;;
  status) need screen; status_worker ;;
  *) echo "usage: $0 {start|stop|restart|status}" >&2; exit 2 ;;
esac
