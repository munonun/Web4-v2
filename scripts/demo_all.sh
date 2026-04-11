#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

printf '=== BASIC ===\n'
go run ./cmd/web4 demo basic

printf '\n=== CONFLICT ===\n'
go run ./cmd/web4 demo conflict

printf '\n=== PROOF ===\n'
go run ./cmd/web4 demo proof
