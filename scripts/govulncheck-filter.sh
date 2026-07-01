#!/usr/bin/env bash
set -euo pipefail

go run golang.org/x/vuln/cmd/govulncheck@latest -format=json ./... |
  go run ./scripts/govulncheck_filter.go -allow GO-2024-3218
