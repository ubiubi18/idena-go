#!/usr/bin/env bash
set -euo pipefail

IDENA_GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_TOOLCHAIN="${IDENA_GO_GOTOOLCHAIN:-go1.26.5}"

if ! command -v go >/dev/null 2>&1; then
  echo "Go toolchain is missing." >&2
  exit 1
fi

if [[ "${IDENA_GO_DIR}" =~ [[:space:]] ]]; then
  WORKSPACE_DIR="$(cd "${IDENA_GO_DIR}/.." && pwd)"
  LINK_DIR="${TMPDIR:-/tmp}/idena-clean-fork-${UID:-user}"

  if [[ -e "${LINK_DIR}" && ! -L "${LINK_DIR}" ]]; then
    echo "Cannot create whitespace-free Go workspace symlink at ${LINK_DIR}; path exists and is not a symlink." >&2
    exit 1
  fi

  ln -sfn "${WORKSPACE_DIR}" "${LINK_DIR}"
  IDENA_GO_DIR="${LINK_DIR}/$(basename "${IDENA_GO_DIR}")"
fi

cd "${IDENA_GO_DIR}"
exec env GOTOOLCHAIN="${GO_TOOLCHAIN}" go "$@"
