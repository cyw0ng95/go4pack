#!/usr/bin/env bash
# Simple dev runner: build image then start (or replace) a single interactive dev container.

set -Eeuo pipefail

# Abort if environment variable indicates container context
if [[ -n "${GO4PACK_ENV_TYPE:-}" ]]; then
  echo "[ERROR] Detected GO4PACK_ENV_TYPE=$GO4PACK_ENV_TYPE; runenv.sh must be executed on the host (outside container)." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_TAG="go4pack:dev"
DEV_CONTAINER_NAME="go4pack_dev"
ENGINE=""
VOL_SPEC=""

detect_engine() {
  if command -v podman >/dev/null 2>&1; then
    ENGINE=podman
  elif command -v docker >/dev/null 2>&1; then
    ENGINE=docker
  else
    echo "[ERROR] podman or docker required" >&2
    exit 1
  fi
  # Volume spec (add :z only if podman + selinux)
  if [[ "$ENGINE" == "podman" ]] && command -v selinuxenabled >/dev/null 2>&1 && selinuxenabled; then
    VOL_SPEC="$ROOT_DIR:/opt:Z"
  else
    VOL_SPEC="$ROOT_DIR:/opt"
  fi
}

port_in_use() {  # NEW: generic port usage check
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null | grep -q .
  elif command -v ss >/dev/null 2>&1; then
    ss -ltnp 2>/dev/null | awk -v p=":$port$" '$4 ~ p {f=1} END{exit f?0:1}'
  else
    return 1
  fi
}

kill_port() {
  local port="$1" pids=""
  if command -v lsof >/dev/null 2>&1; then
    pids=$(lsof -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)
  elif command -v ss >/dev/null 2>&1; then
    pids=$(ss -ltnp 2>/dev/null | awk -v p=":$port\$" '$4 ~ p {for(i=1;i<=NF;i++) if($i ~ /pid=/){match($i,/pid=([0-9]+)/,a); if(a[1]) print a[1]}}' | sort -u)
  fi
  [[ -z "$pids" ]] && return 0
  echo "[INFO] Freeing port $port (PIDs: $pids)" >&2
  for pid in $pids; do
    kill "$pid" 2>/dev/null || sudo kill "$pid" 2>/dev/null || true
  done
  sleep 0.5
  # Re-check
  if port_in_use "$port"; then
    echo "[ERROR] Port $port still busy" >&2
    return 1
  fi
  return 0
}

ensure_or_skip_port() {  # NEW: attempt free port; abort optional
  local port="$1" mandatory="$2"
  if ! port_in_use "$port"; then return 0; fi
  echo "[INFO] Port $port busy; attempting to free..." >&2
  if kill_port "$port"; then
    echo "[INFO] Port $port freed." >&2
    return 0
  fi
  if [[ "$mandatory" == "true" ]]; then
    echo "[ERROR] Mandatory port $port unavailable." >&2
    exit 1
  else
    echo "[WARN] Optional port $port still busy; will skip binding." >&2
    return 1
  fi
}

build_image() {
  echo "[1/2] Building image $IMAGE_TAG" >&2
  $ENGINE build -f "$ROOT_DIR/Containerfile" -t "$IMAGE_TAG" "$ROOT_DIR"
}

run_container() {
  echo "[2/2] Starting interactive dev container $DEV_CONTAINER_NAME" >&2
  ensure_or_skip_port 8080 true
  ensure_or_skip_port 3000 true

  # Remove existing (docker) or use --replace (podman)
  if $ENGINE ps -a --format '{{.Names}}' | grep -Fxq "$DEV_CONTAINER_NAME"; then
    if [[ "$ENGINE" == "docker" ]]; then
      echo "[INFO] Removing existing container $DEV_CONTAINER_NAME" >&2
      $ENGINE rm -f "$DEV_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
  fi
  local replace_flag=""
  [[ "$ENGINE" == "podman" ]] && replace_flag="--replace"
  echo "[INFO] Launching shell (exit to stop container)" >&2
  exec $ENGINE run -it \
    -p 8080:8080 \
    -p 13000:3000 \
    -v "$VOL_SPEC" \
    $replace_flag \
    --name "$DEV_CONTAINER_NAME" \
    "$IMAGE_TAG" /bin/bash
}

main() {
  detect_engine
  build_image
  run_container
}

main "$@"