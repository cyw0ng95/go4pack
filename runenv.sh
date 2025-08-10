#!/usr/bin/env bash
# Run environment script: starts Next.js dev server (if present) then builds & runs Go service.
# Usage: bash runenv.sh [-c] [-t]
#   -c   Clear the .runtime directory before starting
#   -t   Run 'go test ./...' first; abort if tests fail

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
VIEW_DIR="$ROOT_DIR/view"
NEXT_PID_FILE="$RUNTIME_DIR/nextjs.pid"
CLEAR_RUNTIME=0
RUN_TESTS=0

# Parse flags
while getopts ":ct" opt; do
  case "$opt" in
    c) CLEAR_RUNTIME=1 ;;
    t) RUN_TESTS=1 ;;
    *) ;;
  esac
done
shift $((OPTIND-1))

# Clear runtime if requested
if [[ $CLEAR_RUNTIME -eq 1 ]]; then
  echo "[INFO] Clearing $RUNTIME_DIR" >&2
  rm -rf "$RUNTIME_DIR" || true
fi

mkdir -p "$RUNTIME_DIR"

# Run tests early if requested
if [[ $RUN_TESTS -eq 1 ]]; then
  echo "[INFO] Running Go tests (go test ./...)" >&2
  if (cd "$ROOT_DIR" && go test ./...); then
    echo "[INFO] Tests passed." >&2
  else
    echo "[ERROR] Tests failed. Aborting start." >&2
    exit 1
  fi
fi

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

run_go() {
  echo "[INFO] Running Go service with 'go run .' (Ctrl+C to stop)" >&2
  (cd "$ROOT_DIR" && go run .)
}

start_next
run_go
