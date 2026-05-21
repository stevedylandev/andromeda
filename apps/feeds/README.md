# feeds

Minimal RSS reader built on the Go standard library plus a SQLite driver and
a feed parser.

## Stack

- `net/http`
- `html/template`
- `database/sql`
- `embed`
- `modernc.org/sqlite`
- `github.com/mmcdole/gofeed`

## Run

```bash
cd apps/feeds
go run .
```

Copy `.env.example` to `.env` if you want local config.

## What it includes

- public feed list
- preview mode via `?url=` / `?urls=` (single `?url=*.opml` fetches and
  previews the OPML feed list, up to 5 items per feed)
- admin login with cookie sessions
- add/remove subscriptions and categories
- OPML import (admin form + `POST /api/import/opml`) and export
  (`/feeds?format=opml`)
- JSON API
- background polling with ETag / Last-Modified
- embedded templates and static assets
