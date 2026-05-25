# Andromeda

![cover](https://files.stevedylan.dev/andromeda-cover.png)

A collection of minimal, self-hosted web apps. Each app compiles to a single
Go binary that embeds its templates and static assets and stores data in
SQLite.

## Apps

| App | Description | Deploy |
|---|---|---|
| [**Sipp**](apps/sipp) | Minimal code sharing with web UI | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Axcf_D?referralCode=JGcIp6) |
| [**Feeds**](apps/feeds) | Minimal RSS reader with OPML import/export and a JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Ezvmhx?referralCode=JGcIp6) |
| [**Jotts**](apps/jotts) | Minimal markdown notes app | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/DLhUhH?referralCode=JGcIp6) |
| [**OG**](apps/og) | Open Graph tag inspector | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/OdXBt_?referralCode=JGcIp6) |
| [**Shrink**](apps/shrink) | Image compression and resizing | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/enYUFb?referralCode=JGcIp6) |
| [**Cellar**](apps/cellar) | Minimal wine collection tracker | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/MNprVh?referralCode=JGcIp6) |
| [**Posts**](apps/posts) | Minimal CMS blog with admin interface | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/tYtJYp?referralCode=JGcIp6) |
| [**Bookmarks**](apps/bookmarks) | Minimal link saver with categories and JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/DZfr5P?referralCode=JGcIp6) |
| [**Library**](apps/library) | Minimal book tracker with Google Books search | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/tepdeI?referralCode=JGcIp6) |
| [**Easel**](apps/easel) | Daily public-domain painting from the Art Institute of Chicago | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/0DpuRE?referralCode=JGcIp6) |
| [**Blobs**](apps/blobs) | Minimal web browser for S3-compatible blob storage | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/3CH6O6?referralCode=JGcIp6) |

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
