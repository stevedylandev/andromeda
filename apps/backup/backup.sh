#!/bin/sh
set -eu

TIMESTAMP=$(date -u +%Y-%m-%dT%H%M%SZ)
BUCKET="${R2_BUCKET:-andromeda-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

DBS="jotts:/data/jotts/jotts.sqlite sipp:/data/sipp/sipp.sqlite cellar:/data/cellar/cellar.sqlite posts:/data/posts/posts.sqlite feeds:/data/feeds/feeds.sqlite library:/data/library/library.sqlite bookmarks:/data/bookmarks/bookmarks.sqlite parcels:/data/parcels/parcels.db easel:/data/easel/easel.sqlite habbits:/data/habbits/habbits.sqlite"

for entry in $DBS; do
  name="${entry%%:*}"
  path="${entry#*:}"

  if [ ! -f "$path" ]; then
    echo "WARN: $path not found, skipping $name"
    continue
  fi

  backup_file="/tmp/${name}-${TIMESTAMP}.sqlite"
  echo "$(date -u) Backing up $name..."
  sqlite3 "$path" ".backup '$backup_file'"
  gzip "$backup_file"
  aws s3 cp "${backup_file}.gz" "s3://${BUCKET}/${name}/${TIMESTAMP}.sqlite.gz" \
    --endpoint-url "${R2_ENDPOINT}"
  rm -f "${backup_file}.gz"
  echo "$(date -u) OK: $name uploaded"
done

# Prune old backups
cutoff=$(date -u -d "-${RETENTION_DAYS} days" +%Y-%m-%d 2>/dev/null || date -u -v-${RETENTION_DAYS}d +%Y-%m-%d)
for name in jotts sipp cellar posts feeds library bookmarks parcels easel habbits; do
  aws s3 ls "s3://${BUCKET}/${name}/" --endpoint-url "${R2_ENDPOINT}" 2>/dev/null | while read -r line; do
    filedate=$(echo "$line" | awk '{print $1}')
    filename=$(echo "$line" | awk '{print $4}')
    if [ -n "$filename" ] && [ "$filedate" \< "$cutoff" ]; then
      aws s3 rm "s3://${BUCKET}/${name}/${filename}" --endpoint-url "${R2_ENDPOINT}"
      echo "$(date -u) Pruned: ${name}/${filename}"
    fi
  done
done

echo "$(date -u) Backup complete"
