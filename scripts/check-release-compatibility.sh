#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_FILE="$ROOT_DIR/compatibility/stack-lock.json"

"$ROOT_DIR/scripts/check-compatibility-runtime.sh"
args=("$LOCK_FILE" "$ROOT_DIR")
if [[ -n "${GITHUB_REF_NAME:-}" ]]; then
  args+=("$GITHUB_REF_NAME")
fi
python3 "$ROOT_DIR/scripts/check_release_compatibility.py" "${args[@]}"
