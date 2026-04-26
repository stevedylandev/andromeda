# Bookmarks

Personal link saver organized by category.

## Quickstart

1. Make sure [Rust](https://www.rust-lang.org/tools/install) is installed

```bash
rustc --version
```

2. Clone and build

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda
cargo build -p bookmarks
```

3. Run the dev server

```bash
cargo run -p bookmarks
# Server running on http://localhost:3000
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `BOOKMARKS_PASSWORD` | Password for the admin panel | — |
| `BOOKMARKS_API_KEY` | API key for `POST /api/links` (omit to disable write API) | — |
| `BOOKMARKS_DB_PATH` | SQLite database path | `bookmarks.sqlite` |
| `HOST` | Bind address | `127.0.0.1` |
| `PORT` | Bind port | `3000` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

Bookmarks is a single-user link saver. Add links via the admin panel or JSON API, organize them into categories, and view them on a public index page grouped by category. A few highlights:

- Single Rust binary with embedded assets
- Local SQLite storage
- Password-protected admin panel for managing categories and links
- JSON read API (open) and write API (key-guarded)
- Dark themed UI with Commit Mono font

## Usage

### Admin Panel

Set `BOOKMARKS_PASSWORD` and visit `/login`. From the admin panel you can:

- Create and remove categories
- Add links with title, URL, and category
- Remove links

### JSON API

Read endpoints are open. Write endpoints require `x-api-key: <BOOKMARKS_API_KEY>`.

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/categories` | open | List categories |
| `GET` | `/api/links` | open | List links grouped by category. Query: `category` to filter by name |
| `POST` | `/api/links` | api key | Create link. Body: `{category, title, url}` |

Example:

```bash
curl -X POST http://localhost:3000/api/links \
  -H "x-api-key: $BOOKMARKS_API_KEY" \
  -H "content-type: application/json" \
  -d '{"category":"Reading","title":"Example","url":"https://example.com"}'
```

## Structure

```
bookmarks/
├── src/
│   ├── main.rs        # Axum server, admin routes, JSON API, static serving
│   ├── db.rs          # Schema and SQLite queries
│   └── auth.rs        # Session + API-key guards
├── templates/         # Askama HTML templates (index, login, admin)
├── static/            # Static assets embedded at compile time via rust-embed
├── Dockerfile
└── docker-compose.yml
```

## Deployment

Since Bookmarks compiles to a single binary, deployment is straightforward on any platform.

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/bookmarks
cp .env.example .env
# Edit .env with your credentials
docker compose up -d
```

Mount a volume at `BOOKMARKS_DB_PATH` to persist the SQLite database.

### Binary

```bash
cargo build --release -p bookmarks
```

The resulting binary at `./target/release/bookmarks` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](../../LICENSE)
