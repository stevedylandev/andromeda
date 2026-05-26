# blobs

Self-hosted browser, CLI, and TUI for S3-compatible blob storage. Built for Cloudflare R2 but works with any S3-compatible endpoint (AWS S3, MinIO, Backblaze B2, etc).

Single binary, three modes:

- `blobs server` — web UI (password-protected, session cookies)
- `blobs` (no args) — Yazi-style TUI for browsing buckets, copying URLs, opening files, previewing images inline
- `blobs <file>` — one-shot upload that prints the resulting URL to stdout

Features:

- Lists every bucket the credentials can see
- Folder/file navigation with breadcrumbs (web) or two-pane preview (TUI)
- Inline image previews via Kitty/Ghostty/iTerm2 graphics protocols (or `chafa` fallback)
- Presigned download URLs + optional permanent public URL map
- Upload (multi-file in web, single-file in CLI), replace, delete, create folder

## Quick start (server)

```sh
cp .env.example .env
# edit .env — set BLOBS_PASSWORD and either:
#   S3_ENDPOINT + S3_ACCESS_KEY_ID + S3_SECRET_ACCESS_KEY  (generic)
#   R2_ACCOUNT_ID + S3_ACCESS_KEY_ID + S3_SECRET_ACCESS_KEY  (R2)
go run . server
```

Visit `http://127.0.0.1:3000` and log in.

## CLI / TUI

The same binary also speaks directly to S3 (no server required):

```sh
blobs auth                          # interactive: writes ~/.config/blobs/config.toml
blobs                               # TUI — bucket picker then folder browse
blobs -b my-bucket                  # TUI — jump straight into my-bucket
blobs ./photo.png                   # upload to BLOBS_DEFAULT_BUCKET, print URL
blobs -b my-bucket --prefix imgs/ ./photo.png
```

The TUI auto-detects Kitty/Ghostty/iTerm2 graphics protocols for inline image previews and falls back to `chafa` when available. Override with `BLOBS_PREVIEW={kitty|iterm|chafa|none}`.

Key bindings in browse view: `enter`/`l` open, `h`/`esc` back, `y` copy URL (public if mapped, else presigned), `Y` force public, `K` copy S3 key, `o` open in browser, `u` upload, `d` delete, `r` refresh, `b` jump to buckets, `space` toggle preview, `?` help, `q` quit.

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
