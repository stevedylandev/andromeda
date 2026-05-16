# jotts-go

Go port of [jotts](../jotts): minimal markdown notes app.

## Stack

- Go stdlib `net/http` + `html/template`
- `modernc.org/sqlite` (pure-Go SQLite, no CGO)
- `github.com/yuin/goldmark` (markdown rendering w/ strikethrough, tables, tasklists)
- Bubble Tea/Lip Gloss/Glamour for the TUI editor
- `github.com/pkg/browser` and `github.com/atotto/clipboard` for TUI browser/copy actions

## Quickstart

```bash
cp .env.example .env
# edit .env with your password
go run .
```

## Environment variables

| Variable | Description | Default |
|---|---|---|
| `JOTTS_PASSWORD` | Login password | `changeme` |
| `JOTTS_DB_PATH` | SQLite file path | `jotts.sqlite` |
| `HOST` | Bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | HTTPS-only cookies | `false` |
| `JOTTS_API_KEY` | API key for `/api/notes` (unset = API disabled) | _(unset)_ |

## Structure

```
jotts-go/
├── main.go           # entrypoint
├── app.go            # App struct + page data types
├── db.go             # SQLite schema + queries (notes, sessions)
├── routes.go         # http.ServeMux routes
├── middleware.go     # session + API key middleware, cookies
├── handlers_web.go   # HTML form handlers
├── handlers_api.go   # JSON API handlers
├── markdown.go       # goldmark rendering
├── web.go            # template render, JSON, embedded static
├── util.go           # env, dotenv, short IDs, session tokens
├── templates/        # html/template pages
├── static/           # favicons, styles, og image
├── assets/           # darkmatter.css + Commit Mono fonts
├── Dockerfile
└── docker-compose.yml
```

## API

All endpoints require `x-api-key: $JOTTS_API_KEY` header.

- `GET /api/notes` — list notes
- `POST /api/notes` — create `{title, content}`
- `GET /api/notes/{short_id}`
- `PUT /api/notes/{short_id}` — update `{title, content}`
- `DELETE /api/notes/{short_id}`

## Build

```bash
CGO_ENABLED=0 go build -o jotts-go .
```

Single ~10MB self-contained binary with all assets embedded.

## Docker

```bash
docker compose up -d
```
