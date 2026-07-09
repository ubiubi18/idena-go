package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunFilterAllowsFindingByOSVAndModule(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht","package":"github.com/libp2p/go-libp2p-kad-dht","function":"Provide"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht", "")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "allowed reachable GO-2024-3218 in github.com/libp2p/go-libp2p-kad-dht") {
		t.Fatalf("missing allowed finding output: %s", stderr.String())
	}
}

func TestRunFilterBlocksAllowedOSVFromUnexpectedModule(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"example.com/other","package":"example.com/other","function":"Call"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht", "")

	if code != 1 {
		t.Fatalf("expected policy failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "blocked reachable GO-2024-3218 in example.com/other") {
		t.Fatalf("missing blocked finding output: %s", stderr.String())
	}
}

func TestRunFilterBlocksUnlistedOSV(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2099-0001","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht","function":"Call"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht", "")

	if code != 1 {
		t.Fatalf("expected policy failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "blocked reachable GO-2099-0001") {
		t.Fatalf("missing blocked finding output: %s", stderr.String())
	}
}

func TestRunFilterIgnoresScopedModuleOnlyFinding(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2026-5932","trace":[{"module":"golang.org/x/crypto"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "", "GO-2026-5932@golang.org/x/crypto")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "ignored module/package-only GO-2026-5932 in golang.org/x/crypto") {
		t.Fatalf("missing ignored finding output: %s", stderr.String())
	}
}

func TestRunFilterDoesNotIgnoreReachableFinding(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2026-5932","trace":[{"module":"golang.org/x/crypto","package":"golang.org/x/crypto/openpgp","function":"ReadMessage"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "", "GO-2026-5932@golang.org/x/crypto")

	if code != 1 {
		t.Fatalf("expected policy failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "blocked reachable GO-2026-5932 in golang.org/x/crypto") {
		t.Fatalf("missing reachable finding output: %s", stderr.String())
	}
}

func TestRunFilterUsesHighestPrecisionFinding(t *testing.T) {
	input := strings.NewReader("" +
		`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht"}]}}` + "\n" +
		`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht","function":"Provide"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht", "")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "module/package-only") {
		t.Fatalf("lower precision finding should be suppressed: %s", stderr.String())
	}
}

func TestRunFilterReportsCleanScan(t *testing.T) {
	input := strings.NewReader(`{"config":{"scanner_name":"govulncheck"}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "", "")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no reachable vulnerabilities found") {
		t.Fatalf("missing clean scan output: %s", stderr.String())
	}
}

func TestRunFilterReportsMalformedJSON(t *testing.T) {
	var stderr bytes.Buffer

	code := runFilter(strings.NewReader(`{`), &stderr, "", "")

	if code != 2 {
		t.Fatalf("expected parse failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "failed to parse govulncheck JSON") {
		t.Fatalf("missing parse failure output: %s", stderr.String())
	}
}
