package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunFilterAllowsFindingByOSVAndModule(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht","version":"v0.41.0"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "allowed GO-2024-3218 in github.com/libp2p/go-libp2p-kad-dht") {
		t.Fatalf("missing allowed finding output: %s", stderr.String())
	}
}

func TestRunFilterBlocksAllowedOSVFromUnexpectedModule(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2024-3218","trace":[{"module":"example.com/other"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht")

	if code != 1 {
		t.Fatalf("expected policy failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "blocked GO-2024-3218 in example.com/other") {
		t.Fatalf("missing blocked finding output: %s", stderr.String())
	}
}

func TestRunFilterBlocksUnlistedOSV(t *testing.T) {
	input := strings.NewReader(`{"finding":{"osv":"GO-2099-0001","trace":[{"module":"github.com/libp2p/go-libp2p-kad-dht"}]}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht")

	if code != 1 {
		t.Fatalf("expected policy failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "blocked GO-2099-0001") {
		t.Fatalf("missing blocked finding output: %s", stderr.String())
	}
}

func TestRunFilterReportsCleanScan(t *testing.T) {
	input := strings.NewReader(`{"config":{"scanner_name":"govulncheck"}}`)
	var stderr bytes.Buffer

	code := runFilter(input, &stderr, "")

	if code != 0 {
		t.Fatalf("expected success, got exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no reachable vulnerabilities found") {
		t.Fatalf("missing clean scan output: %s", stderr.String())
	}
}

func TestRunFilterReportsMalformedJSON(t *testing.T) {
	var stderr bytes.Buffer

	code := runFilter(strings.NewReader(`{`), &stderr, "")

	if code != 2 {
		t.Fatalf("expected parse failure, got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "failed to parse govulncheck JSON") {
		t.Fatalf("missing parse failure output: %s", stderr.String())
	}
}
