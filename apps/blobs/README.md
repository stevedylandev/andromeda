# blobs

Single-owner web browser for S3-compatible blob storage. Built for Cloudflare R2 but works with any S3-compatible endpoint (AWS S3, MinIO, Backblaze B2, etc).

Features:

- Password login + session cookie auth
- Lists every bucket the credentials can see
- Folder/file navigation with breadcrumbs
- Inline image thumbnails in folder view
- File detail page: metadata, presigned download link, optional static public URL
- Upload (multi-file), replace, delete, create folder

## Quick start

```sh
cp .env.example .env
# edit .env — set BLOBS_PASSWORD and either:
#   S3_ENDPOINT + S3_ACCESS_KEY_ID + S3_SECRET_ACCESS_KEY  (generic)
#   R2_ACCOUNT_ID + S3_ACCESS_KEY_ID + S3_SECRET_ACCESS_KEY  (R2)
go run .
```

Visit `http://127.0.0.1:3000` and log in.

## Configuration

See `.env.example` for the full list. Notable knobs:

- `BLOBS_MAX_UPLOAD_MB` — single-shot upload cap (default 100MB)
- `BLOBS_PRESIGN_TTL_SECONDS` — presigned download URL lifetime (default 3600)
- `BLOBS_PUBLIC_URLS` — `bucket=url,bucket=url` map; when a file's bucket appears here, the detail page also surfaces a permanent public URL (e.g. an R2 public dev URL or custom domain)

## R2 setup

1. In the Cloudflare dashboard, create an R2 API token with read+write access to your bucket(s).
2. Set in `.env`:
   ```
   R2_ACCOUNT_ID=<your account id>
   S3_ACCESS_KEY_ID=<token id>
   S3_SECRET_ACCESS_KEY=<token secret>
   ```
3. (Optional) Enable a public dev URL or custom domain on the bucket and add it to `BLOBS_PUBLIC_URLS`.

## Generic S3 setup (e.g. MinIO)

```
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
```

## Routes

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/login`, POST `/login` | password form |
| POST | `/logout` | clear session |
| GET | `/buckets` | list buckets |
| GET | `/b/{bucket}/browse/{prefix...}` | folder listing |
| GET | `/b/{bucket}/object/{key...}` | file detail |
| GET | `/b/{bucket}/preview/{key...}` | proxied file stream (for inline images) |
| POST | `/b/{bucket}/upload` | multipart file upload |
| POST | `/b/{bucket}/replace` | overwrite existing key |
| POST | `/b/{bucket}/delete` | delete by key |
| POST | `/b/{bucket}/mkdir` | create zero-byte `prefix/name/` marker |
