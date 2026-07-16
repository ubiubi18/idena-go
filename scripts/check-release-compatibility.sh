#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_FILE="$ROOT_DIR/compatibility/stack-lock.json"

"$ROOT_DIR/scripts/check-compatibility-runtime.sh"
python3 "$ROOT_DIR/scripts/check_release_compatibility.py" "$LOCK_FILE" "$ROOT_DIR"
