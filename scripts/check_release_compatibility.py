import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import sys


def fail(message):
    print(f"Release compatibility gate failed: {message}", file=sys.stderr)
    raise SystemExit(1)


def validate(lock_file, root_dir):
    with open(lock_file, encoding="utf-8") as handle:
        payload = json.load(handle)
    root = Path(root_dir).resolve()
    evidence_root = (root / "compatibility" / "evidence").resolve()

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

    runtime_commit = ""
    for component in payload.get("components", []):
        if component.get("name") == "idena-go":
            runtime_commit = component.get("runtimeCodeCommit", "")
            break
    if not re.fullmatch(r"[0-9a-f]{40}", runtime_commit):
        fail("idena-go runtimeCodeCommit is invalid")

    digest_pattern = re.compile(r"^[0-9a-f]{64}$")
    for gate in required:
        result = results.get(gate)
        if not isinstance(result, dict):
            fail(f"missing result for {gate!r}")
        if result.get("status") != "passed":
            fail(f"gate {gate!r} is not marked passed")
        evidence = result.get("evidence")
        if not isinstance(evidence, str):
            fail(f"gate {gate!r} needs a committed evidence path")
        relative = PurePosixPath(evidence)
        if (
            relative.is_absolute()
            or ".." in relative.parts
            or relative.parts[:2] != ("compatibility", "evidence")
            or relative.suffix != ".json"
        ):
            fail(f"gate {gate!r} evidence must be JSON below compatibility/evidence")
        evidence_path = (root / Path(*relative.parts)).resolve()
        if evidence_root not in evidence_path.parents or not evidence_path.is_file():
            fail(f"gate {gate!r} evidence file is missing or outside compatibility/evidence")

        expected_digest = result.get("sha256", "")
        if not digest_pattern.fullmatch(expected_digest):
            fail(f"gate {gate!r} needs a lowercase SHA-256 evidence digest")
        raw_evidence = evidence_path.read_bytes()
        if hashlib.sha256(raw_evidence).hexdigest() != expected_digest:
            fail(f"gate {gate!r} evidence digest does not match")
        try:
            evidence_payload = json.loads(raw_evidence)
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            fail(f"gate {gate!r} evidence is not valid JSON: {exc}")
        if evidence_payload.get("schema") != 1:
            fail(f"gate {gate!r} evidence has an unsupported schema")
        if evidence_payload.get("gate") != gate or evidence_payload.get("status") != "passed":
            fail(f"gate {gate!r} evidence does not record a passing result")
        if evidence_payload.get("testedCommit") != runtime_commit:
            fail(f"gate {gate!r} evidence targets a different node commit")
        source = evidence_payload.get("source")
        if not isinstance(source, str) or not source.startswith("https://"):
            fail(f"gate {gate!r} evidence needs an HTTPS source URL")

    unknown = sorted(set(results) - set(required))
    if unknown:
        fail(f"gateResults contains unrequired entries: {', '.join(unknown)}")

    print("Release compatibility evidence passed")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        fail("expected lock file and repository root arguments")
    validate(sys.argv[1], sys.argv[2])
