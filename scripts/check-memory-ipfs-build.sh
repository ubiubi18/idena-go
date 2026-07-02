#!/usr/bin/env bash
set -euo pipefail

packages=(
  ./ipfs
  ./events
  ./blockchain
  ./blockchain/...
  ./core/appstate
  ./core/mempool
  ./core/state
  ./core/upgrade
  ./keystore
  ./secstore
  ./rpc
  ./vm/...
)

go test -tags=idena_memory_ipfs "${packages[@]}"

blocked_deps="$(
  go list -deps -tags=idena_memory_ipfs "${packages[@]}" |
    grep -E '^(github.com/ipfs/kubo|github.com/libp2p/go-libp2p-kad-dht)(/|$)' || true
)"

if [[ -n "${blocked_deps}" ]]; then
  echo "memory IPFS build unexpectedly imports network IPFS/DHT dependencies:" >&2
  echo "${blocked_deps}" >&2
  exit 1
fi
