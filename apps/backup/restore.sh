#!/bin/sh
set -eu

# Restore SQLite backups from Cloudflare R2 into Docker volumes.
#
# Usage:
#   restore.sh <app|all> [--timestamp <ts>] [--list] [--yes]
#
#   <app>            One of the registered apps, or "all".
#   --timestamp <ts> Restore a specific backup (e.g. 2026-04-04T060000Z).
#                    Default: latest. The .sqlite.gz suffix is optional.
#   --list           List available backups for the app(s) and exit.
#   --yes            Skip the interactive confirmation prompt.
#
# Env (same as backup.sh): R2_ENDPOINT, R2_BUCKET, AWS_ACCESS_KEY_ID,
# AWS_SECRET_ACCESS_KEY. Read from ./.env (or $ENV_FILE) if set; real env wins.
#
# Volume names are read ONLY from the root docker-compose.yml (the `name:`
# field of each external volume), not from .env. Override the compose file
# location with $COMPOSE_FILE. Missing volumes are created automatically.
#
# Run this on the HOST. It shells out to `docker run` to write into volumes;
# the backup container itself mounts those volumes read-only.

# Load .env from the script's directory (or $ENV_FILE) if present. Existing
# environment variables take precedence over the file.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ENV_FILE="${ENV_FILE:-$SCRIPT_DIR/.env}"
if [ -f "$ENV_FILE" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ''|\#*) continue ;;
    esac
    key="${line%%=*}"
    case "$key" in
      *[!A-Za-z0-9_]*|'') continue ;;
    esac
    # Only set if not already in the environment.
    if [ -z "$(eval "printf '%s' \"\${$key:-}\"")" ]; then
      val="${line#*=}"
      export "$key=$val"
    fi
  done < "$ENV_FILE"
fi

BUCKET="${R2_BUCKET:-andromeda-backups}"

# Root docker-compose.yml is the single source of truth for volume names.
COMPOSE_FILE="${COMPOSE_FILE:-$SCRIPT_DIR/../../docker-compose.yml}"

# app | volume-internal filename | compose volume key | fallback volume name
# The compose key is looked up in COMPOSE_FILE; the fallback is used only if
# the key (or compose file) is missing.
APPS="jotts:jotts.sqlite:jotts_data:jotts_jotts-data
sipp:sipp.sqlite:sipp_data:sipp-rust_sipp-data
cellar:cellar.sqlite:cellar_data:cellar_cellar_data
posts:posts.sqlite:posts_data:posts_posts-data
feeds:feeds.sqlite:feeds_data:feeds_feeds_data
library:library.sqlite:library_data:library_library-data
bookmarks:bookmarks.sqlite:bookmarks_data:bookmarks_bookmarks-data
parcels:parcels.db:parcels_data:parcels_parcels_data
easel:easel.sqlite:easel_data:easel_easel-data
habbits:habbits.sqlite:habbits_data:habbits_habbits-data"

RESTORE_IMAGE="debian:bookworm-slim"

usage() {
  sed -n '6,19p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

# Look up a field for an app from the APPS registry.
# $1 app name, $2 field index (2=file, 3=compose key, 4=fallback volume)
app_field() {
  echo "$APPS" | while IFS=: read -r name file key def; do
    [ "$name" = "$1" ] || continue
    case "$2" in
      2) echo "$file" ;;
      3) echo "$key" ;;
      4) echo "$def" ;;
    esac
    break
  done
}

app_exists() {
  echo "$APPS" | cut -d: -f1 | grep -qx "$1"
}

# Read the external `name:` for a volume key from the compose file's top-level
# `volumes:` block. Echoes nothing if the file or key is absent.
compose_volume_name() {
  key="$1"
  [ -f "$COMPOSE_FILE" ] || return 0
  awk -v key="$key" '
    /^volumes:/             { invol=1; next }
    invol && /^[^[:space:]]/ { invol=0 }
    invol && $0 ~ "^  "key":[[:space:]]*$" { found=1; next }
    found && /^[[:space:]]*name:[[:space:]]/ {
      sub(/^[[:space:]]*name:[[:space:]]*/, ""); print; exit
    }
    found && /^  [^[:space:]]/ { exit }   # next volume block, no name set
  ' "$COMPOSE_FILE"
}

# Resolve the Docker volume name for an app from the root compose file,
# falling back to the registry default only if compose has no entry.
resolve_volume() {
  app="$1"
  key=$(app_field "$app" 3)
  def=$(app_field "$app" 4)
  name=$(compose_volume_name "$key")
  if [ -n "$name" ]; then
    echo "$name"
  else
    echo "$def"
  fi
}

list_backups() {
  app="$1"
  echo "Backups for $app (s3://$BUCKET/$app/):"
  aws s3 ls "s3://$BUCKET/$app/" --endpoint-url "$R2_ENDPOINT" \
    | awk '{print "  " $1 " " $2 "  " $4}' \
    || echo "  (none or unreachable)"
}

# Echo the object key (filename only) to restore for an app.
resolve_key() {
  app="$1"
  if [ -n "$TIMESTAMP" ]; then
    case "$TIMESTAMP" in
      *.sqlite.gz) echo "$TIMESTAMP" ;;
      *) echo "${TIMESTAMP}.sqlite.gz" ;;
    esac
    return
  fi
  aws s3 ls "s3://$BUCKET/$app/" --endpoint-url "$R2_ENDPOINT" \
    | awk '{print $4}' | grep -v '^$' | sort | tail -1
}

restore_app() {
  app="$1"

  key=$(resolve_key "$app")
  if [ -z "$key" ]; then
    echo "WARN: no backups found for $app, skipping"
    return 1
  fi

  volume=$(resolve_volume "$app")
  internal_file=$(app_field "$app" 2)
  src="s3://$BUCKET/$app/$key"

  if ! docker volume inspect "$volume" >/dev/null 2>&1; then
    echo "$(date -u) Creating docker volume '$volume' for $app ..."
    docker volume create "$volume" >/dev/null \
      || { echo "WARN: failed to create volume '$volume' for $app, skipping"; return 1; }
  fi

  if [ "$ASSUME_YES" != "1" ]; then
    echo
    echo "About to restore:"
    echo "  app:    $app"
    echo "  source: $src"
    echo "  volume: $volume -> /$internal_file"
    printf "Overwrite live data? [y/N] "
    read -r answer
    case "$answer" in
      y|Y|yes|YES) ;;
      *) echo "Skipped $app"; return 0 ;;
    esac
  fi

  workdir=$(mktemp -d)
  trap 'rm -rf "$workdir"' EXIT INT TERM

  echo "$(date -u) Downloading $src ..."
  aws s3 cp "$src" "$workdir/$key" --endpoint-url "$R2_ENDPOINT" \
    || { echo "WARN: download failed for $app"; rm -rf "$workdir"; trap - EXIT INT TERM; return 1; }

  echo "$(date -u) Decompressing ..."
  gunzip "$workdir/$key"
  decompressed="$workdir/${key%.gz}"

  echo "$(date -u) Verifying integrity ..."
  check=$(sqlite3 "$decompressed" 'PRAGMA integrity_check;' 2>&1 || true)
  if [ "$check" != "ok" ]; then
    echo "WARN: integrity check failed for $app: $check"
    rm -rf "$workdir"; trap - EXIT INT TERM
    return 1
  fi

  echo "$(date -u) Writing into volume $volume ..."
  docker run --rm \
    -v "$volume:/data" \
    -v "$workdir:/backup:ro" \
    "$RESTORE_IMAGE" \
    cp "/backup/$(basename "$decompressed")" "/data/$internal_file"

  rm -rf "$workdir"
  trap - EXIT INT TERM

  echo "$(date -u) OK: $app restored from $key"
  echo "WARNING: stop the '$app' service before restoring and restart it after for a clean"
  echo "         restore. This script does not stop or start services."
  return 0
}

# --- arg parsing ---------------------------------------------------------------

[ $# -ge 1 ] || usage 1

TARGET=""
TIMESTAMP=""
DO_LIST=0
ASSUME_YES=0

while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help) usage 0 ;;
    --list) DO_LIST=1 ;;
    --yes|-y) ASSUME_YES=1 ;;
    --timestamp)
      shift
      [ $# -ge 1 ] || die "--timestamp requires a value"
      TIMESTAMP="$1"
      ;;
    --timestamp=*) TIMESTAMP="${1#*=}" ;;
    -*) die "unknown option: $1" ;;
    *)
      [ -z "$TARGET" ] || die "unexpected argument: $1"
      TARGET="$1"
      ;;
  esac
  shift
done

[ -n "$TARGET" ] || usage 1
: "${R2_ENDPOINT:?R2_ENDPOINT is required}"

if [ "$TARGET" = "all" ]; then
  if [ -n "$TIMESTAMP" ] && [ "$DO_LIST" != "1" ]; then
    die "--timestamp cannot be combined with 'all' (timestamps differ per app)"
  fi
  TARGETS=$(echo "$APPS" | cut -d: -f1)
else
  app_exists "$TARGET" || die "unknown app: $TARGET (valid: $(echo "$APPS" | cut -d: -f1 | tr '\n' ' '))"
  TARGETS="$TARGET"
fi

if [ "$DO_LIST" = "1" ]; then
  for app in $TARGETS; do
    list_backups "$app"
  done
  exit 0
fi

failures=0
total=0
for app in $TARGETS; do
  total=$((total + 1))
  restore_app "$app" || failures=$((failures + 1))
done

if [ "$failures" -gt 0 ] && [ "$failures" -eq "$total" ]; then
  die "all restores failed ($failures/$total)"
fi
echo "$(date -u) Restore complete ($((total - failures))/$total succeeded)"
