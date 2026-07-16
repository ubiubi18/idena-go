#!/usr/bin/env bash
set -euo pipefail

go-bindata -o keywords_gen.go -pkg=keywords keywords.json
