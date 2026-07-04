#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
  echo "This script is for macOS arm64 only." >&2
  exit 1
fi

IDENA_GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE_DIR="$(cd "${IDENA_GO_DIR}/.." && pwd)"
WASM_BINDING_DIR="${WORKSPACE_DIR}/idena-wasm-binding"
WASM_SRC_DIR="${WORKSPACE_DIR}/idena-wasm"
OUTPUT_BIN="${1:-$HOME/Library/Application Support/Idena/node/idena-go}"
GO_RUNNER="${IDENA_GO_DIR}/scripts/run-go-toolchain.sh"

if ! command -v cargo >/dev/null 2>&1 || ! command -v rustc >/dev/null 2>&1; then
  echo "Rust toolchain is missing. Install rustup first:" >&2
  echo "brew install rustup-init" >&2
  echo "rustup-init -y --profile minimal" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go toolchain is missing." >&2
  exit 1
fi

if [[ ! -d "${WASM_SRC_DIR}" ]]; then
  echo "idena-wasm directory not found at ${WASM_SRC_DIR}" >&2
  echo "Clone the idena-wasm fork beside idena-go before running this script." >&2
  exit 1
fi

if [[ ! -d "${WASM_BINDING_DIR}" ]]; then
  echo "idena-wasm-binding directory not found at ${WASM_BINDING_DIR}" >&2
  echo "Clone the idena-wasm-binding fork beside idena-go before running this script." >&2
  exit 1
fi

echo "Building libidena_wasm for aarch64-apple-darwin..."
(
  cd "${WASM_SRC_DIR}"
  cargo build --release --target aarch64-apple-darwin
)

mkdir -p "${WASM_BINDING_DIR}/lib"
cp "${WASM_SRC_DIR}/target/aarch64-apple-darwin/release/libidena_wasm.a" \
  "${WASM_BINDING_DIR}/lib/libidena_wasm_darwin_arm64.a"

echo "Building idena-go..."
mkdir -p "$(dirname "${OUTPUT_BIN}")"
(
  cd "${IDENA_GO_DIR}"
  "${GO_RUNNER}" build -ldflags "-X main.version=1.1.2" -o "${OUTPUT_BIN}" .
)
chmod 755 "${OUTPUT_BIN}"

"${OUTPUT_BIN}" --version

echo "Done. Node binary written to: ${OUTPUT_BIN}"
