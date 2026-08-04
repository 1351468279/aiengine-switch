#!/usr/bin/env bash

set -Eeuo pipefail

readonly RELAY_BASE_URL="https://modelapi.aiaiaiaiai.cloud/v1"

SKIP_LOGIN=0
API_KEY_STDIN=0

usage() {
  cat <<'EOF'
Configure Codex CLI for the AIARE NewAPI relay.

Usage:
  setup-codex.sh [--skip-login] [--api-key-stdin]

Options:
  --skip-login      Only update config.toml; do not ask for an API key.
  --api-key-stdin   Read the API key from stdin instead of prompting.
  -h, --help        Show this help.

The API key may also be supplied through AIARE_CODEX_API_KEY for automation.
EOF
}

die() {
  printf 'Error: %s\n' "$1" >&2
  exit 1
}

for arg in "$@"; do
  case "$arg" in
    --skip-login)
      SKIP_LOGIN=1
      ;;
    --api-key-stdin)
      API_KEY_STDIN=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "Unknown option: $arg"
      ;;
  esac
done

if [[ "$SKIP_LOGIN" -eq 1 && "$API_KEY_STDIN" -eq 1 ]]; then
  die "--skip-login and --api-key-stdin cannot be used together"
fi

if [[ -n "${CODEX_HOME:-}" ]]; then
  CODEX_DIR="$CODEX_HOME"
elif [[ -n "${HOME:-}" ]]; then
  CODEX_DIR="$HOME/.codex"
else
  die 'HOME or CODEX_HOME must be set'
fi

[[ -n "$CODEX_DIR" && "$CODEX_DIR" != "/" ]] || die "Invalid CODEX_HOME: $CODEX_DIR"

CONFIG_PATH="$CODEX_DIR/config.toml"
mkdir -p "$CODEX_DIR"

if [[ -e "$CONFIG_PATH" && ! -f "$CONFIG_PATH" ]]; then
  die "$CONFIG_PATH exists but is not a regular file"
fi

CONFIG_EXISTED=0
if [[ -f "$CONFIG_PATH" ]]; then
  CONFIG_EXISTED=1
  BACKUP_STAMP="$(date +%Y%m%d-%H%M%S)"
  BACKUP_PATH="${CONFIG_PATH}.bak.$BACKUP_STAMP"
  BACKUP_INDEX=0
  while [[ -e "$BACKUP_PATH" ]]; do
    ((BACKUP_INDEX += 1))
    BACKUP_PATH="${CONFIG_PATH}.bak.${BACKUP_STAMP}-${BACKUP_INDEX}"
  done
  cp -p "$CONFIG_PATH" "$BACKUP_PATH"
  printf 'Backed up existing config to %s\n' "$BACKUP_PATH"
fi

TEMP_PATH=""
cleanup() {
  if [[ -n "$TEMP_PATH" ]]; then
    rm -f "$TEMP_PATH"
  fi
}
trap cleanup EXIT

TEMP_PATH="$(mktemp "$CODEX_DIR/.config.toml.aiare.XXXXXX")"

if [[ "$CONFIG_EXISTED" -eq 1 ]]; then
  awk -v relay_url="$RELAY_BASE_URL" '
    function insert_missing() {
      if (!found_provider) {
        print "model_provider = \"openai\""
        found_provider = 1
      }
      if (!found_url) {
        print "openai_base_url = \"" relay_url "\""
        found_url = 1
      }
    }

    /^[[:space:]]*\[/ {
      if (!inserted) {
        insert_missing()
        inserted = 1
      }
      in_table = 1
    }

    !in_table && $0 ~ /^[[:space:]]*model_provider[[:space:]]*=/ {
      print "model_provider = \"openai\""
      found_provider = 1
      next
    }

    !in_table && $0 ~ /^[[:space:]]*openai_base_url[[:space:]]*=/ {
      print "openai_base_url = \"" relay_url "\""
      found_url = 1
      next
    }

    { print }

    END {
      if (!inserted) {
        insert_missing()
      }
    }
  ' "$CONFIG_PATH" > "$TEMP_PATH"
else
  cat > "$TEMP_PATH" <<EOF
model_provider = "openai"
openai_base_url = "$RELAY_BASE_URL"
EOF
fi

mv "$TEMP_PATH" "$CONFIG_PATH"
TEMP_PATH=""
if [[ "$CONFIG_EXISTED" -eq 0 ]]; then
  chmod 600 "$CONFIG_PATH" 2>/dev/null || true
fi

printf 'Codex config written to %s\n' "$CONFIG_PATH"

if [[ "$SKIP_LOGIN" -eq 1 ]]; then
  printf 'Skipped API key login. Run this script again without --skip-login after installing Codex CLI.\n'
  exit 0
fi

if [[ -r /dev/tty ]] && { [[ -t 0 ]] || [[ -t 1 ]]; }; then
  exec 3</dev/tty
else
  exec 3<&0
fi

API_KEY="${AIARE_CODEX_API_KEY:-}"
if [[ -z "$API_KEY" ]]; then
  if [[ "$API_KEY_STDIN" -eq 1 ]]; then
    IFS= read -r -u 3 API_KEY || true
  elif [[ -t 3 ]]; then
    printf 'Enter your AIARE NewAPI API key (input is hidden; press Enter to skip): ' >&2
    IFS= read -r -s -u 3 API_KEY || true
    printf '\n' >&2
  else
    printf 'No interactive terminal was found; API key login was skipped.\n' >&2
    exit 0
  fi
fi

if [[ -z "$API_KEY" ]]; then
  printf 'Config updated. API key login was skipped.\n'
  exit 0
fi

if ! command -v codex >/dev/null 2>&1; then
  unset API_KEY
  die 'Codex CLI was not found. Install Codex CLI, then run this script again.'
fi

if printf '%s\n' "$API_KEY" | codex login --with-api-key; then
  printf 'Codex API key login completed.\n'
else
  unset API_KEY
  die 'Codex API key login failed; config.toml was still updated.'
fi

unset API_KEY
