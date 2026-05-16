# easel-go

Go rewrite of [easel](../easel). A daily painting from the Art Institute of
Chicago, persisted to SQLite. Past days browsable; future days unavailable.

## Routes

- `GET /` — today's artwork
- `GET /day/{YYYY-MM-DD}` — specific past day
- `GET /archive` — full archive
- `GET /api/today` / `GET /api/day/{date}` / `GET /api/archive` — JSON
- `GET /feed.xml` — Atom feed

## Env

See `.env.example`. Notes: timezone uses Go's `time.LoadLocation`, which needs
the system tzdata (Debian slim base in the Dockerfile pulls `tzdata`).
