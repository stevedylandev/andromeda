# Andromeda

![cover](https://files.stevedylan.dev/andromeda-cover.png)

A collection of minimal, self-hosted web apps. Each app compiles to a single
binary. The original implementation is a Rust workspace (Axum + Askama). A
parallel Go port lives alongside, sharing the same SQLite schemas and routes
so either implementation can serve the same data.

## Apps

| App | Description | Deploy |
|---|---|---|
| [**Sipp**](apps/sipp) | Minimal code sharing with web UI and TUI | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Axcf_D?referralCode=JGcIp6) |
| [**Feeds**](apps/feeds) | Minimal RSS reader with OPML import/export and a JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Ezvmhx?referralCode=JGcIp6) |
| [**Parcels**](apps/parcels) | Minimal package tracking (USPS) | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/HNQUs4?referralCode=JGcIp6) |
| [**Jotts**](apps/jotts) | Minimal markdown notes app | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/DLhUhH?referralCode=JGcIp6) |
| [**OG**](apps/og) | Open Graph tag inspector | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/OdXBt_?referralCode=JGcIp6) |
| [**Shrink**](apps/shrink) | Image compression and resizing | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/enYUFb?referralCode=JGcIp6) |
| [**Cellar**](apps/cellar) | Minimal wine collection tracker | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/MNprVh?referralCode=JGcIp6) |
| [**Posts**](apps/posts) | Minimal CMS blog with admin interface | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/tYtJYp?referralCode=JGcIp6) |
| [**Bookmarks**](apps/bookmarks) | Minimal link saver with categories and JSON API | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/DZfr5P?referralCode=JGcIp6) |
| [**Library**](apps/library) | Minimal book tracker with Google Books search | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/tepdeI?referralCode=JGcIp6) |
| [**Easel**](apps/easel) | Daily public-domain painting from the Art Institute of Chicago | [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/0DpuRE?referralCode=JGcIp6) |

## Go ports

The same apps are being rewritten in Go under `apps/<name>-go/`. Each one is
a separate Go module that embeds its templates and static assets and uses
shared packages from `crates-go/`. Status:

| Go app | Notes |
|---|---|
| `apps/feeds-go` | full parity |
| `apps/jotts-go` | full parity |
| `apps/og-go` | full parity |
| `apps/shrink-go` | EXIF reinjection dropped |
| `apps/bookmarks-go` | full parity |
| `apps/library-go` | full parity |
| `apps/easel-go` | full parity |
| `apps/cellar-go` | EXIF orientation auto-rotate dropped |
| `apps/posts-go` | local FS only (no R2/S3) |
| `apps/sipp-go` | server + CLI; interactive TUI not ported |

`apps/parcels-go` is intentionally not built (USPS API access has changed).

Each Go app references the shared `crates-go/` packages via `replace`
directives in its `go.mod`, so the source tree is fully self-contained.

## Shared crates

Rust:

| Crate | Description |
|---|---|
| [`andromeda-auth`](crates/auth) | Session-based password authentication |
| [`andromeda-db`](crates/db) | Shared database types and session management |
| [`andromeda-darkmatter-css`](crates/darkmatter-css) | Shared CSS + fonts |

Go (each is its own module under `crates-go/`):

| Package | Description |
|---|---|
| `crates-go/web` | HTTP helpers (embedded assets, JSON, render, redirect) |
| `crates-go/auth` | Sessions store, password/api-key verification, short-id |
| `crates-go/config` | env + `.env` loading helpers |
| `crates-go/darkmatter` | Embedded CSS + fonts, mountable on any `http.ServeMux` |

## Stack

Rust apps: Axum + rusqlite + Askama + rust-embed + tokio.
Go apps: stdlib `net/http` + `modernc.org/sqlite` (pure Go, no cgo) +
`html/template` + `embed.FS`. Permitted extras: `goldmark` (markdown),
`gofeed` (RSS), `golang.org/x/net/html` (HTML parsing),
`golang.org/x/image/draw` (image resize), `alecthomas/chroma` (highlight),
`golang.org/x/crypto/bcrypt` (passwords).

## Getting Started

Rust:

```bash
# Build all apps
cargo build --release

# Run a specific app
cargo run -p sipp -- server --port 3000
cargo run -p feeds
cargo run -p jotts
cargo run -p og
cargo run -p shrink
```

Go:

```bash
cd apps/feeds-go && cp .env.example .env && go run .
cd apps/posts-go && cp .env.example .env && go run .
# sipp-go has two binaries:
cd apps/sipp-go && go run ./cmd/server
```

Each app has its own README with detailed setup, environment variables, and deployment instructions.

## License

[MIT](LICENSE)
