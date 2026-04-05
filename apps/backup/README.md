# Backup

Automated SQLite backups for Jotts, Sipp, and Cellar to Cloudflare R2. Runs every 6 hours via cron inside a Docker container and prunes backups older than 30 days.

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

1. List available backups for a service (e.g. `jotts`):

```sh
aws s3 ls s3://andromeda-backups/jotts/ --endpoint-url https://<account-id>.r2.cloudflarestorage.com
```

2. Download the backup you want to restore:

```sh
aws s3 cp s3://andromeda-backups/jotts/2026-04-04T060000Z.sqlite.gz ./restore.sqlite.gz \
  --endpoint-url https://<account-id>.r2.cloudflarestorage.com
```

3. Decompress it:

```sh
gunzip restore.sqlite.gz
```

4. Stop the target service so nothing is writing to the database:

```sh
docker compose -f /path/to/jotts/docker-compose.yml down
```

5. Copy the restored database into the volume:

```sh
docker run --rm -v jotts_jotts-data:/data -v $(pwd):/backup debian:bookworm-slim \
  cp /backup/restore.sqlite /data/jotts.sqlite
```

6. Restart the service:

```sh
docker compose -f /path/to/jotts/docker-compose.yml up -d
```

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
