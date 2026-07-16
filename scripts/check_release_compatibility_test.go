package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseCompatibilityValidatorVerifiesEvidenceDigest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release validation runs on Linux")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}

	const (
		gate         = "legacy-state-replay-differential"
		testedCommit = "0123456789abcdef0123456789abcdef01234567"
		evidencePath = "compatibility/evidence/state-replay.json"
	)
	root := t.TempDir()
	requireTestNoError(t, os.MkdirAll(filepath.Join(root, "compatibility", "evidence"), 0700))
	evidence, err := json.Marshal(map[string]any{
		"schema":       1,
		"gate":         gate,
		"status":       "passed",
		"testedCommit": testedCommit,
		"source":       "https://github.com/example/repository/actions/runs/123456789",
	})
	requireTestNoError(t, err)
	requireTestNoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(evidencePath)), evidence, 0600))
	digest := fmt.Sprintf("%x", sha256.Sum256(evidence))

	lock := map[string]any{
		"status":        "approved",
		"requiredGates": []string{gate},
		"gateResults": map[string]any{
			gate: map[string]any{
				"status":   "passed",
				"evidence": evidencePath,
				"sha256":   digest,
			},
		},
		"components": []map[string]any{
			{
				"name":              "idena-go",
				"runtimeCodeCommit": testedCommit,
			},
		},
	}
	lockPath := filepath.Join(root, "compatibility", "stack-lock.json")
	writeTestJSON(t, lockPath, lock)

	output, err := exec.Command(python, "check_release_compatibility.py", lockPath, root).CombinedOutput()
	if err != nil {
		t.Fatalf("valid evidence was rejected: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "evidence passed") {
		t.Fatalf("unexpected validator output: %s", output)
	}

	lock["gateResults"].(map[string]any)[gate].(map[string]any)["sha256"] = strings.Repeat("0", 64)
	writeTestJSON(t, lockPath, lock)
	output, err = exec.Command(python, "check_release_compatibility.py", lockPath, root).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "digest does not match") {
		t.Fatalf("mismatched digest was not rejected: err=%v output=%s", err, output)
	}
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	requireTestNoError(t, err)
	requireTestNoError(t, os.WriteFile(path, raw, 0600))
}

func requireTestNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
