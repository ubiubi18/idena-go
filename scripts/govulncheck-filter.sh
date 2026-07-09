#!/usr/bin/env bash
set -euo pipefail

if go list -deps ./... | grep -Eq '^golang.org/x/crypto/openpgp($|/)'; then
  echo "govulncheck: forbidden OpenPGP package entered the dependency graph" >&2
  exit 1
fi

go tool govulncheck -format=json ./... |
  go run ./scripts/govulncheck_filter.go \
    -allow-reachable GO-2024-3218@github.com/libp2p/go-libp2p-kad-dht \
    -ignore-unreachable GO-2026-5932@golang.org/x/crypto
