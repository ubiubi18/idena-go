#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_FILE="$ROOT_DIR/compatibility/stack-lock.json"

"$ROOT_DIR/scripts/check-compatibility-runtime.sh"

python3 - "$LOCK_FILE" <<'PY'
import json
import re
import sys


def fail(message):
    print(f"Release compatibility gate failed: {message}", file=sys.stderr)
    raise SystemExit(1)


with open(sys.argv[1], encoding="utf-8") as handle:
    payload = json.load(handle)

if payload.get("status") != "approved":
    fail(f"stack lock status is {payload.get('status')!r}, expected 'approved'")

required = payload.get("requiredGates")
if not isinstance(required, list) or not required:
    fail("requiredGates must be a non-empty list")
if len(required) != len(set(required)):
    fail("requiredGates contains duplicates")

results = payload.get("gateResults")
if not isinstance(results, dict):
    fail("gateResults must be an object")

digest_pattern = re.compile(r"^[0-9a-f]{64}$")
for gate in required:
    result = results.get(gate)
    if not isinstance(result, dict):
        fail(f"missing result for {gate!r}")
    if result.get("status") != "passed":
        fail(f"gate {gate!r} is not marked passed")
    evidence = result.get("evidence")
    if not isinstance(evidence, str) or not evidence.startswith("https://"):
        fail(f"gate {gate!r} needs an HTTPS evidence URL")
    if not digest_pattern.fullmatch(result.get("sha256", "")):
        fail(f"gate {gate!r} needs a lowercase SHA-256 evidence digest")

unknown = sorted(set(results) - set(required))
if unknown:
    fail(f"gateResults contains unrequired entries: {', '.join(unknown)}")

print("Release compatibility evidence passed")
PY
