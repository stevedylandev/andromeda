# Backup

Automated SQLite backups for Jotts, Sipp, Cellar, Posts, Feeds, Library, Bookmarks, Parcels, and Easel to Cloudflare R2. Runs every 6 hours via cron inside a Docker container and prunes backups older than 30 days.

## Setup

1. **Create an R2 bucket:**
   - Log in to the [Cloudflare dashboard](https://dash.cloudflare.com).
   - Select your account, then navigate to **R2 Object Storage** in the sidebar.
   - Click **Create bucket** and name it `andromeda-backups` (or a name of your choice).

2. **Find your account ID and endpoint:**
   - Your account ID is in the Cloudflare dashboard URL: `https://dash.cloudflare.com/<account-id>`.
   - You can also find it on the **R2 Overview** page under **Account ID**.
   - Your R2 endpoint is `https://<account-id>.r2.cloudflarestorage.com`.

3. **Generate R2 API credentials:**
   - On the **R2 Overview** page, click **Manage R2 API Tokens**.
   - Click **Create API Token**.
   - Give it a name (e.g. `andromeda-backup`).
   - Set **Permissions** to **Object Read & Write**.
   - Under **Specify bucket(s)**, select the bucket you created (or apply to all buckets).
   - Click **Create API Token**.
   - Copy the **Access Key ID** and **Secret Access Key** — these are only shown once.

4. **Configure the environment:**

```sh
cp .env.example .env
```

Fill in the values from the previous steps:

```
R2_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
AWS_ACCESS_KEY_ID=<your-r2-access-key>
AWS_SECRET_ACCESS_KEY=<your-r2-secret-key>
R2_BUCKET=andromeda-backups
```

4. If your Docker volume names differ from the defaults, set them in `.env`:

```
JOTTS_VOLUME=jotts_jotts-data
SIPP_VOLUME=sipp_sipp-data
CELLAR_VOLUME=cellar_cellar-data
POSTS_VOLUME=posts_posts-data
FEEDS_VOLUME=feeds_feeds-data
LIBRARY_VOLUME=library_library-data
BOOKMARKS_VOLUME=bookmarks_bookmarks-data
PARCELS_VOLUME=parcels_parcels_data
EASEL_VOLUME=easel_easel-data
```

Run `docker volume ls` to check the actual names on your host.

5. Start the backup container:

**Option A: Build from source**

```sh
docker compose up -d --build
```

**Option B: Use the pre-built image from GHCR**

Override the `build` directive with `image` in your compose file or use a separate override:

```sh
docker compose -f docker-compose.yml -f docker-compose.ghcr.yml up -d
```

Create a `docker-compose.ghcr.yml` override file:

```yaml
services:
  backup:
    image: ghcr.io/stevedylandev/andromeda-backup:latest
    build: !reset null
```

Or simply run the image directly:

```sh
docker run -d --restart unless-stopped \
  --env-file .env \
  -v jotts_jotts-data:/data/jotts:ro \
  -v sipp_sipp-data:/data/sipp:ro \
  -v cellar_cellar-data:/data/cellar:ro \
  -v posts_posts-data:/data/posts:ro \
  -v feeds_feeds-data:/data/feeds:ro \
  -v library_library-data:/data/library:ro \
  -v bookmarks_bookmarks-data:/data/bookmarks:ro \
  -v parcels_parcels_data:/data/parcels:ro \
  -v easel_easel-data:/data/easel:ro \
  ghcr.io/stevedylandev/andromeda-backup:latest
```

## Running a Manual Backup

```sh
docker compose exec backup /usr/local/bin/backup.sh
```

## Checking Logs

```sh
docker compose exec backup cat /var/log/backup.log
```

## Restoring from a Backup

Use `restore.sh` to download a backup from R2 and write it into the target Docker volume.
Run it on the **host** (it shells out to `docker run`; the backup container mounts the data
volumes read-only). It reads the same `.env` as the backup (`R2_ENDPOINT`, `R2_BUCKET`,
AWS keys, optional `*_VOLUME` overrides) — `source .env` first, or prefix the env inline.

```sh
restore.sh <app|all> [--timestamp <ts>] [--list] [--yes]
```

- `<app>` — one of: `jotts sipp cellar posts feeds library bookmarks parcels easel`, or `all`.
- `--timestamp <ts>` — restore a specific backup (e.g. `2026-04-04T060000Z`); the
  `.sqlite.gz` suffix is optional. Default: the latest backup. Not valid with `all`.
- `--list` — list available backups for the app(s) and exit (no restore).
- `--yes` — skip the interactive confirmation prompt.

> **Warning:** Restoring overwrites live data. Stop the target service before restoring and
> restart it after for a clean restore — `restore.sh` does **not** stop or start services.

Examples:

```sh
# Load credentials/volume overrides
set -a; . ./.env; set +a

# See what's available
restore.sh jotts --list

# Restore the latest jotts backup (prompts for confirmation)
restore.sh jotts

# Restore a specific backup, no prompt
restore.sh jotts --timestamp 2026-04-04T060000Z --yes

# Restore every app's latest backup
restore.sh all
```

For each app the script: resolves the backup key (latest or `--timestamp`), downloads and
decompresses it, runs `PRAGMA integrity_check`, confirms the target volume exists, then
copies the database to the volume root via a throwaway `debian:bookworm-slim` container.

### Manual restore (fallback)

1. List backups: `aws s3 ls s3://andromeda-backups/jotts/ --endpoint-url https://<account-id>.r2.cloudflarestorage.com`
2. Download: `aws s3 cp s3://andromeda-backups/jotts/<timestamp>.sqlite.gz ./restore.sqlite.gz --endpoint-url <endpoint>`
3. Decompress: `gunzip restore.sqlite.gz`
4. Stop the target service so nothing is writing to the database.
5. Copy into the volume: `docker run --rm -v jotts_jotts-data:/data -v $(pwd):/backup debian:bookworm-slim cp /backup/restore.sqlite /data/jotts.sqlite`
6. Restart the service.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `R2_ENDPOINT` | — | Cloudflare R2 S3-compatible endpoint |
| `AWS_ACCESS_KEY_ID` | — | R2 access key |
| `AWS_SECRET_ACCESS_KEY` | — | R2 secret key |
| `R2_BUCKET` | `andromeda-backups` | R2 bucket name |
| `RETENTION_DAYS` | `30` | Days to keep backups before pruning |
| `JOTTS_VOLUME` | `jotts_jotts-data` | Docker volume name for Jotts data |
| `SIPP_VOLUME` | `sipp_sipp-data` | Docker volume name for Sipp data |
| `CELLAR_VOLUME` | `cellar_cellar-data` | Docker volume name for Cellar data |
| `POSTS_VOLUME` | `posts_posts-data` | Docker volume name for Posts data |
| `FEEDS_VOLUME` | `feeds_feeds-data` | Docker volume name for Feeds data |
| `LIBRARY_VOLUME` | `library_library-data` | Docker volume name for Library data |
| `BOOKMARKS_VOLUME` | `bookmarks_bookmarks-data` | Docker volume name for Bookmarks data |
| `PARCELS_VOLUME` | `parcels_parcels_data` | Docker volume name for Parcels data |
| `EASEL_VOLUME` | `easel_easel-data` | Docker volume name for Easel data |
