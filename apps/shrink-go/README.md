# shrink-go

Go rewrite of [shrink](../shrink). JPEG compression + resize via stdlib `image`
plus `golang.org/x/image/draw` for Catmull-Rom scaling.

## Quickstart

```bash
cp .env.example .env
go run .
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `HOST` | `127.0.0.1` | Bind host |
| `PORT` | `3000` | Server port |

## Routes

- `GET /` — upload UI
- `POST /compress` — multipart upload (`file`, `quality` 1-100, optional `width`)
- `GET /static/*` — embedded assets
- `/assets/*` — darkmatter css/fonts

## Notes vs Rust version

JPEG EXIF metadata is preserved after recompression, with GPS data stripped to
match the Rust implementation. The 20 MB upload limit is preserved.
