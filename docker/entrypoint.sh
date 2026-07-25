#!/bin/sh
set -eu

DATA_ROOT="${FRANTIC_POSTR_DATA_DIR:-/data}"
APP_BIN="/app/frantic-postr"

seed_if_empty() {
  src="$1"
  dest="$2"

  mkdir -p "$dest"
  if [ ! -d "$src" ]; then
    return 0
  fi

  if [ -z "$(find "$dest" -mindepth 1 -maxdepth 1 2>/dev/null)" ]; then
    cp -a "$src"/. "$dest"/ 2>/dev/null || true
  fi
}

mkdir -p "$DATA_ROOT"
seed_if_empty "/seed/config" "$DATA_ROOT/config"
seed_if_empty "/seed/templates" "$DATA_ROOT/templates"
seed_if_empty "/seed/fonts" "$DATA_ROOT/fonts"
seed_if_empty "/seed/output" "$DATA_ROOT/output"
seed_if_empty "/seed/backups" "$DATA_ROOT/backups"
seed_if_empty "/seed/logs" "$DATA_ROOT/logs"

if [ ! -f "$DATA_ROOT/config/config.toml" ] && [ -f "$DATA_ROOT/config/config.example.toml" ]; then
  cp "$DATA_ROOT/config/config.example.toml" "$DATA_ROOT/config/config.toml"
fi

if [ "$#" -eq 0 ]; then
  set -- "$APP_BIN" -config "$DATA_ROOT/config/config.toml" -web -port "${FRANTIC_POSTR_PORT:-8080}"
elif [ "${1#-}" != "$1" ]; then
  set -- "$APP_BIN" "$@"
fi

exec "$@"
