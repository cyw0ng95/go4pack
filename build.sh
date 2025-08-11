#!/usr/bin/env bash
# Build / test / run helper (inside dev container).
# Usage:
#   build.sh                 Compile backend only (default if no flags)
#   build.sh -t              Compile then run tests with coverage (go mod vendor + go test)
#   build.sh -r              Compile then run application (uses compiled binary)
#   build.sh -t -r           Compile, test, then run
#   build.sh -c              Clear .runtime before other actions (optional)
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

compile_backend() {
  echo "[INFO] Compiling backend commands in parallel -> $DEV_DIR" >&2
  local CMD_ROOT="$ROOT_DIR/cmd"
  if [[ ! -d "$CMD_ROOT" ]]; then
    echo "[WARN] No cmd directory found" >&2
    return 0
  fi
  mapfile -t CMD_DIRS < <(find "$CMD_ROOT" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
  if [[ ${#CMD_DIRS[@]} -eq 0 ]]; then
    echo "[WARN] No command subdirectories under cmd/" >&2
    return 0
  fi
  mkdir -p "$DEV_DIR/build_logs"
  local pids=()
  for d in "${CMD_DIRS[@]}"; do
    local name="$(basename "$d")"
    (
      set -e
      cd "$ROOT_DIR"
      echo "[BUILD $name] start" >&2
      go build -o "$DEV_DIR/$name" "./cmd/$name" 2>&1
      echo "[BUILD $name] done size=$(stat -c %s "$DEV_DIR/$name" 2>/dev/null || echo '?')" >&2
    ) >"$DEV_DIR/build_logs/$name.log" 2>&1 &
    pids+=("$!:${name}")
  done
  local fail=0
  for pn in "${pids[@]}"; do
    local pid="${pn%%:*}"; local name="${pn##*:}"
    if ! wait "$pid"; then
      echo "[ERROR] Build failed for $name" >&2
      sed 's/^/  /' "$DEV_DIR/build_logs/$name.log" >&2 || true
      fail=1
    else
      echo "[INFO] Built $name" >&2
    fi
  done
  # Report sizes of all successfully built binaries
  echo "[INFO] Built binary sizes:" >&2
  local total=0
  for d in "${CMD_DIRS[@]}"; do
    local name="$(basename "$d")"
    if [[ -f "$DEV_DIR/$name" ]]; then
      local size
      size=$(stat -c %s "$DEV_DIR/$name" 2>/dev/null || echo '?')
      if [[ $size != '?' ]]; then total=$((total + size)); fi
      if [[ $size == '?' ]]; then
        printf '[INFO]   %-12s %s bytes\n' "$name" "$size" >&2
      else
        local mb
        mb=$(awk -v s="$size" 'BEGIN{printf "%.2f", s/1024/1024}')
        printf '[INFO]   %-12s %s bytes (%s MB)\n' "$name" "$size" "$mb" >&2
      fi
    fi
  done
  if [[ $total -gt 0 ]]; then
    local total_mb
    total_mb=$(awk -v s="$total" 'BEGIN{printf "%.2f", s/1024/1024}')
    echo "[INFO]   TOTAL        $total bytes (${total_mb} MB)" >&2
  fi
  if [[ $fail -eq 1 ]]; then
    echo "[ERROR] One or more builds failed" >&2
    return 1
  fi
}

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
  echo "[INFO] Running application ($DEV_DIR/go4pack) — Ctrl+C to stop" >&2
  ( cd "$ROOT_DIR" && exec "$DEV_DIR/go4pack" )
}

# Always compile first
compile_backend

# If no flags supplied, only compile (default behavior)
if [[ $DO_TEST -eq 0 && $DO_RUN -eq 0 ]]; then
  echo "[INFO] Compile only (no actions requested)." >&2
  exit 0
fi

[[ $DO_TEST -eq 1 ]] && run_tests
[[ $DO_RUN  -eq 1 ]] && run_app

echo "[INFO] Done." >&2