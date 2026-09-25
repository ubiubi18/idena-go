package depcheck

// This guard protects a fix that lives entirely in a replaced dependency.
//
// kad-dht (via kubo) runs every query response through an IP diversity filter
// and drops all returned peers when more than the allowed number share an IP
// group, counting every advertised address. Idena nodes advertise 127.0.0.1
// and often docker addresses, so upstream kad-dht drops every response: a node
// never gets past its bootstrap peers, finds no providers, cannot fetch blocks
// from IPFS, and exits at startup. The fix ignores nonpublic addresses in that
// filter and lives in a patched fork pinned through a `replace` directive in
// go.mod.
//
// Because the fix is only a replace directive, a routine dependency update
// (e.g. `go mod tidy` or a dependabot bump of kad-dht) could silently drop the
// replace and reintroduce the startup failure with no other signal. This test
// fails loudly if that happens. It parses go.mod with the standard library
// only, so it adds no dependency of its own.

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	kadDHTModule     = "github.com/libp2p/go-libp2p-kad-dht"
	patchedKadDHTMod = "github.com/ubiubi18/go-libp2p-kad-dht"
)

// findGoMod walks up from the test's working directory to locate go.mod. `go
// test` runs a package's tests with the working directory set to that package's
// source directory, so this finds the module's go.mod without depending on
// compile-time paths (which -trimpath would rewrite).
func findGoMod(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from " + dir)
		}
		dir = parent
	}
}

// replaceTargetFor scans go.mod for a `replace <old> => <newPath> <newVersion>`
// directive, handling both the single-line form and the parenthesized block
// form. It returns the replacement path and version, and whether one was found.
func replaceTargetFor(gomod []byte, oldPath string) (newPath, newVersion string, found bool) {
	inBlock := false
	scanner := bufio.NewScanner(bytes.NewReader(gomod))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "replace ("):
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "replace "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "replace"))
		case !inBlock:
			continue
		}

		// line now looks like: <old> [oldver] => <newPath> [newVersion]
		arrow := strings.Index(line, "=>")
		if arrow < 0 {
			continue
		}
		lhs := strings.Fields(line[:arrow])
		rhs := strings.Fields(line[arrow+2:])
		if len(lhs) == 0 || len(rhs) == 0 {
			continue
		}
		if lhs[0] != oldPath {
			continue
		}
		newPath = rhs[0]
		if len(rhs) > 1 {
			newVersion = rhs[1]
		}
		return newPath, newVersion, true
	}
	return "", "", false
}

func TestKadDHTReplaceDirectivePresent(t *testing.T) {
	path := findGoMod(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	newPath, newVersion, found := replaceTargetFor(data, kadDHTModule)
	if !found {
		t.Fatalf("go.mod has no `replace %s => %s` directive; it must stay until the "+
			"nonpublic-address diversity fix is released in upstream kad-dht, otherwise "+
			"Idena nodes drop every DHT response and cannot bootstrap over IPFS",
			kadDHTModule, patchedKadDHTMod)
	}
	if newPath != patchedKadDHTMod {
		t.Fatalf("go.mod replaces %s with %q, want the patched fork %q; the DHT "+
			"response-diversity fix for Idena peer records lives only in that fork, so "+
			"redirecting or dropping this replace reintroduces the IPFS bootstrap failure",
			kadDHTModule, newPath, patchedKadDHTMod)
	}
	if newVersion == "" {
		t.Fatalf("go.mod replaces %s with %s but no version is pinned",
			kadDHTModule, patchedKadDHTMod)
	}
}
