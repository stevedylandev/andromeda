# Andromeda

![cover](https://files.stevedylan.dev/andromeda-cover.png)

A collection of minimal, self-hosted web apps. Each app compiles to a single
Go binary that embeds its templates and static assets and stores data in
SQLite.

## Apps

| App | Description | Deploy |
|---|---|---|
| [**Sipp**](apps/sipp) | Minimal code sharing with web UI | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-sipp) |
| [**Feeds**](apps/feeds) | Minimal RSS reader with OPML import/export and a JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-feeds) |
| [**Jotts**](apps/jotts) | Minimal markdown notes app | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-jotts) |
| [**OG**](apps/og) | Open Graph tag inspector | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-og) |
| [**Shrink**](apps/shrink) | Image compression and resizing | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-shrink) |
| [**Cellar**](apps/cellar) | Minimal wine collection tracker | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-cellar) |
| [**Posts**](apps/posts) | Minimal CMS blog with admin interface | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-posts) |
| [**Bookmarks**](apps/bookmarks) | Minimal link saver with categories and JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-bookmarks) |
| [**Library**](apps/library) | Minimal book tracker with Google Books search | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-library) |
| [**Easel**](apps/easel) | Daily public-domain painting from the Art Institute of Chicago | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-easel) |
| [**Blobs**](apps/blobs) | Minimal web browser for S3-compatible blob storage | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-blobs) |
| [**Quotes**](apps/quotes) | Minimal quote-a-day site seeded from classic literature | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/andromeda-quotes) |

## Shared packages

Under `pkg/`, each its own Go module:

| Package | Description |
|---|---|
| `pkg/web` | HTTP helpers (embedded assets, JSON, render, redirect) |
| `pkg/auth` | Sessions store, password/api-key verification, short-id |
| `pkg/config` | env + `.env` loading helpers |
| `pkg/darkmatter` | Embedded CSS + fonts, mountable on any `http.ServeMux` |

Each app references these via `replace` directives in its `go.mod`, so the
source tree is fully self-contained.

## Stack

Stdlib `net/http` + `modernc.org/sqlite` (pure Go, no cgo) + `html/template`
+ `embed.FS`. Permitted extras: `goldmark` (markdown), `gofeed` (RSS),
`golang.org/x/net/html` (HTML parsing), `golang.org/x/image/draw` (image
resize), `alecthomas/chroma` (highlight), `golang.org/x/crypto/bcrypt`
(passwords).

## Getting Started

```bash
cd apps/feeds && cp .env.example .env && go run .
cd apps/posts && cp .env.example .env && go run .
cd apps/sipp && go run . server --port 3000
```

Each app has its own README with detailed setup, environment variables, and
deployment instructions.

## License

[MIT](LICENSE)
