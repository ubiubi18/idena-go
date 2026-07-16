#!/usr/bin/env bash
set -euo pipefail

if go list -deps ./... | grep -Eq '^golang.org/x/crypto/openpgp($|/)'; then
  echo "govulncheck: forbidden OpenPGP package entered the dependency graph" >&2
  exit 1
fi

dht_module="github.com/libp2p/go-libp2p-kad-dht"
reviewed_dht_version="v0.41.0"
dht_review_deadline="2026-10-16"
if [[ "$(date -u +%F)" > "${dht_review_deadline}" ]]; then
  echo "govulncheck: GO-2024-3218 allowance expired on ${dht_review_deadline}; reassess the DHT exposure" >&2
  exit 1
fi
current_dht_version="$(go list -m -f '{{.Version}}' "${dht_module}")"
if [[ "${current_dht_version}" != "${reviewed_dht_version}" ]]; then
  echo "govulncheck: ${dht_module} changed from reviewed ${reviewed_dht_version} to ${current_dht_version}; reassess GO-2024-3218 before updating the allowance" >&2
  exit 1
fi

go tool govulncheck -format=json ./... |
  go run ./scripts/govulncheck_filter.go \
    -allow-reachable GO-2024-3218@${dht_module} \
    -ignore-unreachable GO-2026-5932@golang.org/x/crypto
