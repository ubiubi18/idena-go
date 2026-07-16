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

func TestCompatibilityRuntimeGuardUsesRuntimeCodeCommit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("compatibility runtime validation runs in bash")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}

	root := t.TempDir()
	requireTestNoError(t, os.MkdirAll(filepath.Join(root, "scripts"), 0700))
	requireTestNoError(t, os.MkdirAll(filepath.Join(root, "compatibility"), 0700))
	script, err := os.ReadFile("check-compatibility-runtime.sh")
	requireTestNoError(t, err)
	requireTestNoError(t, os.WriteFile(filepath.Join(root, "scripts", "check-compatibility-runtime.sh"), script, 0700))
	requireTestNoError(t, os.WriteFile(filepath.Join(root, "runtime.go"), []byte("package runtime\n"), 0600))
	runTestCommand(t, root, git, "init", "-q")
	runTestCommand(t, root, git, "config", "user.email", "test@example.com")
	runTestCommand(t, root, git, "config", "user.name", "Compatibility Test")
	runTestCommand(t, root, git, "add", ".")
	runTestCommand(t, root, git, "commit", "-qm", "runtime baseline")
	runtimeCommit := strings.TrimSpace(runTestCommand(t, root, git, "rev-parse", "HEAD"))

	requireTestNoError(t, os.WriteFile(filepath.Join(root, "runtime.go"), []byte("package runtime\n\nconst changed = true\n"), 0600))
	lock := map[string]any{
		"components": []map[string]any{{
			"name":              "idena-go",
			"commit":            strings.Repeat("f", 40),
			"runtimeCodeCommit": runtimeCommit,
		}},
	}
	writeTestJSON(t, filepath.Join(root, "compatibility", "stack-lock.json"), lock)
	dirtyCommand := exec.Command(bash, filepath.Join(root, "scripts", "check-compatibility-runtime.sh"))
	dirtyCommand.Dir = root
	dirtyOutput, dirtyErr := dirtyCommand.CombinedOutput()
	if dirtyErr == nil || !strings.Contains(string(dirtyOutput), "requires a clean worktree") {
		t.Fatalf("dirty runtime change was not rejected: err=%v output=%s", dirtyErr, dirtyOutput)
	}
	runTestCommand(t, root, git, "add", ".")
	runTestCommand(t, root, git, "commit", "-qm", "change runtime")

	command := exec.Command(bash, filepath.Join(root, "scripts", "check-compatibility-runtime.sh"))
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Runtime-affecting path changed") {
		t.Fatalf("runtime change after runtimeCodeCommit was not rejected: err=%v output=%s", err, output)
	}
}

func TestReleaseArtifactVerifierBindsPublishedBinariesToEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release validation runs on Linux")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}

	const (
		gate         = "independent-rebuild-digest-match"
		testedCommit = "0123456789abcdef0123456789abcdef01234567"
		releaseTag   = "v1.2.3-rc.1"
		evidencePath = "compatibility/evidence/independent-rebuild.json"
	)
	root := t.TempDir()
	buildsDir := filepath.Join(root, "builds")
	requireTestNoError(t, os.MkdirAll(filepath.Join(root, "compatibility", "evidence"), 0700))
	requireTestNoError(t, os.MkdirAll(buildsDir, 0700))

	assetNames := map[string]string{
		"linux-x64":   "idena-node-linux-1.2.3-rc.1",
		"linux-arm64": "idena-node-linux-aarch64-1.2.3-rc.1",
		"windows-x64": "idena-node-win-1.2.3-rc.1.exe",
		"macos-x64":   "idena-node-mac-1.2.3-rc.1",
		"macos-arm64": "idena-node-mac-arm64-1.2.3-rc.1",
	}
	artifacts := make([]map[string]any, 0, len(assetNames))
	for platform, assetName := range assetNames {
		content := []byte("independently rebuilt " + platform)
		digest := fmt.Sprintf("%x", sha256.Sum256(content))
		requireTestNoError(t, os.WriteFile(filepath.Join(buildsDir, assetName), content, 0700))
		requireTestNoError(t, os.WriteFile(
			filepath.Join(buildsDir, assetName+".sha256"),
			[]byte(fmt.Sprintf("%s  %s\n", digest, assetName)),
			0600,
		))
		artifacts = append(artifacts, map[string]any{"platform": platform, "sha256": digest})
	}
	evidence, err := json.Marshal(map[string]any{
		"schema":           1,
		"gate":             gate,
		"status":           "passed",
		"testedCommit":     testedCommit,
		"source":           "https://github.com/example/repository/actions/runs/123456789",
		"releaseTag":       releaseTag,
		"releaseArtifacts": artifacts,
	})
	requireTestNoError(t, err)
	requireTestNoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(evidencePath)), evidence, 0600))
	lock := map[string]any{
		"status":        "approved",
		"requiredGates": []string{gate},
		"gateResults": map[string]any{
			gate: map[string]any{
				"status":   "passed",
				"evidence": evidencePath,
				"sha256":   fmt.Sprintf("%x", sha256.Sum256(evidence)),
			},
		},
		"components": []map[string]any{{
			"name":              "idena-go",
			"runtimeCodeCommit": testedCommit,
		}},
	}
	lockPath := filepath.Join(root, "compatibility", "stack-lock.json")
	writeTestJSON(t, lockPath, lock)

	output, err := exec.Command(
		python,
		"verify_release_artifacts.py",
		lockPath,
		root,
		buildsDir,
		releaseTag,
	).CombinedOutput()
	if err != nil || !strings.Contains(string(output), "match independent rebuild evidence") {
		t.Fatalf("matching release artifacts were rejected: err=%v output=%s", err, output)
	}

	linuxAsset := filepath.Join(buildsDir, assetNames["linux-x64"])
	requireTestNoError(t, os.WriteFile(linuxAsset, []byte("substituted binary"), 0700))
	output, err = exec.Command(
		python,
		"verify_release_artifacts.py",
		lockPath,
		root,
		buildsDir,
		releaseTag,
	).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "digest does not match") {
		t.Fatalf("substituted release artifact was not rejected: err=%v output=%s", err, output)
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

func runTestCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}
