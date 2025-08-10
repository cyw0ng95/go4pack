#!/usr/bin/env bash
# Build / test / run helper (inside dev container).
# Usage:
#   build.sh -h            Show help
#   build.sh -t            Run tests with coverage (go mod vendor + go test)
#   build.sh -r            Run application (go run ./)
#   build.sh -t -r         Test then run
#   build.sh -c            Clear .runtime (if present) before other actions (optional)
# Notes:
#   Must be executed inside dev container (GO4PACK_ENV_TYPE=dev).

set -Eeuo pipefail

DO_CLEAR=0
DO_TEST=0
DO_RUN=0

usage() {
  grep '^# ' "$0" | sed 's/^# \{0,1\}//'
  exit 0
}

while getopts ":hctr" opt; do
  case "$opt" in
    h) usage ;;
    c) DO_CLEAR=1 ;;
    t) DO_TEST=1 ;;
    r) DO_RUN=1 ;;
    *) usage ;;
  esac
done
shift $((OPTIND-1))

# Show help if no action flags provided
if [[ $DO_CLEAR -eq 0 && $DO_TEST -eq 0 && $DO_RUN -eq 0 ]]; then
  usage
fi

in_container() {
  [[ -f "/.dockerenv" || -f "/run/.containerenv" ]]
}
if ! in_container; then
  echo "[ERROR] Must run inside container (use runenv.sh outside)." >&2
  exit 1
fi
if [[ "${GO4PACK_ENV_TYPE:-}" != "dev" ]]; then
  echo "[ERROR] GO4PACK_ENV_TYPE=dev required." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
DEV_DIR="$ROOT_DIR/.dev"
mkdir -p "$DEV_DIR"
chmod 0777 "$DEV_DIR" || true

if [[ $DO_CLEAR -eq 1 ]]; then
  echo "[INFO] Clearing $RUNTIME_DIR" >&2
  rm -rf "$RUNTIME_DIR" || true
fi

run_tests() {
  echo "[INFO] Ensuring vendored modules (go mod vendor)" >&2
  ( cd "$ROOT_DIR" && go mod vendor )
  echo "[INFO] Running tests with coverage" >&2
  ( cd "$ROOT_DIR" && \
    touch .dev/coverage.out .dev/coverage.html .dev/coverage.sum 2>/dev/null || true
    if go test -coverprofile=.dev/coverage.out ./...; then
      go tool cover -func=.dev/coverage.out | tail -n1 | tee .dev/coverage.sum
      go tool cover -html=.dev/coverage.out -o .dev/coverage.html
      TOTAL_PCT=$(awk '{print $3}' "$ROOT_DIR/.dev/coverage.sum" 2>/dev/null || echo "unknown")
      echo "[INFO] Tests passed. Coverage: $TOTAL_PCT" >&2
    else
      echo "[ERROR] Tests failed." >&2
      exit 1
    fi
  )
}

run_app() {
  # NEW: start frontend dev server first (background)
  if [[ -d "$ROOT_DIR/view" ]]; then
    echo "[INFO] Starting frontend (pnpm run dev) in background" >&2
    mkdir -p "$ROOT_DIR/.dev"
    (
      cd "$ROOT_DIR/view"
      # Install deps (attempt frozen first; fallback normal)
      pnpm install -f 2>&1
      pnpm run dev 2>&1 &
      echo $! > "$ROOT_DIR/.dev/frontend.pid"
    )
    echo "[INFO] Frontend PID: $(cat "$ROOT_DIR/.dev/frontend.pid" 2>/dev/null || echo '?')" >&2
  else
    echo "[WARN] No view/ directory; skipping frontend start" >&2
  fi
  echo "[INFO] Running application (go run ./) — Ctrl+C to stop" >&2
  ( cd "$ROOT_DIR" && exec go run ./ )
}

[[ $DO_TEST -eq 1 ]] && run_tests
[[ $DO_RUN  -eq 1 ]] && run_app

echo "[INFO] Done." >&2