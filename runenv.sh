#!/usr/bin/env bash
# Run environment script: starts Next.js dev server (if present) then builds & runs Go service.
# Usage: bash runenv.sh

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
VIEW_DIR="$ROOT_DIR/view"
BIN_DIR="$ROOT_DIR/bin"
GO_BINARY="$BIN_DIR/go4pack"
NEXT_PID_FILE="$RUNTIME_DIR/nextjs.pid"

mkdir -p "$RUNTIME_DIR" "$BIN_DIR"

cleanup() {
  echo "[INFO] Caught exit signal. Cleaning up..." >&2
  if [[ -f "$NEXT_PID_FILE" ]]; then
    NEXT_PID="$(cat "$NEXT_PID_FILE" || true)"
    if [[ -n "${NEXT_PID}" ]] && ps -p "$NEXT_PID" > /dev/null 2>&1; then
      echo "[INFO] Stopping Next.js dev server (PID $NEXT_PID)" >&2
      kill "$NEXT_PID" 2>/dev/null || true
      wait "$NEXT_PID" 2>/dev/null || true
    fi
    rm -f "$NEXT_PID_FILE"
  fi
  echo "[INFO] Cleanup complete." >&2
}
trap cleanup INT TERM EXIT

start_next() {
  if [[ -d "$VIEW_DIR" && -f "$VIEW_DIR/package.json" ]]; then
    echo "[INFO] Starting Next.js dev server from $VIEW_DIR" >&2
    (
      cd "$VIEW_DIR"
      if [[ ! -d node_modules ]]; then
        echo "[INFO] Installing view dependencies (npm install)" >&2
        npm install
      fi
      # Prefer pnpm / yarn if lockfiles present
      if [[ -f pnpm-lock.yaml ]]; then
        (command -v pnpm >/dev/null 2>&1) || npm install -g pnpm
        pnpm dev &
      elif [[ -f yarn.lock ]]; then
        (command -v yarn >/dev/null 2>&1) || npm install -g yarn
        yarn dev &
      else
        npm run dev &
      fi
      echo $! > "$NEXT_PID_FILE"
    )
    echo "[INFO] Next.js dev server PID $(cat "$NEXT_PID_FILE")" >&2
  else
    echo "[INFO] No Next.js project detected (missing view/ or package.json). Skipping front-end." >&2
  fi
}

build_go() {
  echo "[INFO] Building Go module..." >&2
  go build -o "$GO_BINARY" .
  echo "[INFO] Go binary at $GO_BINARY" >&2
}

run_go() {
  echo "[INFO] Running Go service... (Ctrl+C to stop)" >&2
  "$GO_BINARY"
}

start_next
build_go
run_go
