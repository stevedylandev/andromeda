# Easel

A daily painting from the [Art Institute of Chicago](https://api.artic.edu/docs/). One public-domain artwork per calendar day, persisted to SQLite. Past days browsable; future days unavailable until populated.

## Run locally

```bash
cargo run -p easel
```

Visit `http://localhost:4242`.

## Configuration

| Var | Default | Purpose |
|---|---|---|
| `HOST` | `127.0.0.1` | Bind address |
| `PORT` | `4242` | Listen port |
| `EASEL_DB_PATH` | `easel.sqlite` | SQLite file |
| `EASEL_TIMEZONE` | `UTC` | IANA TZ for day boundary |
| `EASEL_CLASSIFICATIONS` | `painting` | Comma-separated `classification_title` filter |
| `EASEL_BACKFILL_DAYS` | `0` | On boot, fill missing past N days |
| `EASEL_MAX_DEDUP_RETRIES` | `10` | Retries when picking a non-duplicate page |

## Routes

- `GET /` — today's artwork
- `GET /day/{YYYY-MM-DD}` — specific past day
- `GET /archive` — full archive
- `GET /api/today` — JSON of today
- `GET /api/day/{YYYY-MM-DD}` — JSON of specific day
- `GET /api/archive` — JSON list

## Image source

Images served from AIC's IIIF endpoint:
`https://www.artic.edu/iiif/2/{image_id}/full/843,/0/default.jpg`
