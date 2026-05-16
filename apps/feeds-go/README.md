# Feeds Go

A Go rewrite of `apps/feeds` using mostly the Go standard library plus a SQLite driver and a feed parser.

## Stack

- `net/http`
- `html/template`
- `database/sql`
- `embed`
- `modernc.org/sqlite`
- `github.com/mmcdole/gofeed`

## Run

```bash
cd apps/feeds-go
go run .
```

Copy `.env.example` to `.env` if you want local config.

## What it includes

- public feed list
- preview mode via `?url=` / `?urls=`
- admin login with cookie sessions
- add/remove subscriptions and categories
- OPML import
- JSON API
- background polling with ETag / Last-Modified
- embedded templates and static assets
