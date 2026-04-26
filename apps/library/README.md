# Library

A minimal personal book tracker

## Quickstart

```bash
git clone https://github.com/stevedylandev/andromeda.git
cd andromeda
cp apps/library/.env.example apps/library/.env
# Edit .env with your admin password
cargo run -p library
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `ADMIN_PASSWORD` | Password for admin login | `changeme` |
| `LIBRARY_DB_PATH` | SQLite database file path | `library.sqlite` |
| `GOOGLE_BOOKS_API_KEY` | Google Books API key for search | |
| `BASE_URL` | Public base URL | `http://localhost:3000` |
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |
| `LIBRARY_DISPLAY_MODE` | Public index layout: `inline` (stacked sections) or `nav` (filter buttons in header) | `inline` |

## Overview

A simple, self-hosted book tracker built with Rust. Highlights:
- Single Rust binary with embedded assets
- Password authentication with session cookies
- Track books across Read, Reading, and Want to Read (labels customizable from admin)
- Google Books search to add titles with cover art and ISBN
- Library search from the admin page (title / author / ISBN)
- Toggle between inline category sections and a filter-nav layout via `LIBRARY_DISPLAY_MODE`
- Per-book notes
- JSON API for listing and fetching books
- SQLite for persistent storage

## Structure

```
library/
├── src/
│   ├── main.rs          # App entrypoint, env vars, router
│   ├── auth.rs          # Password verification and sessions
│   ├── db.rs            # SQLite layer (books)
│   └── google_books.rs  # Google Books API client
├── templates/           # Askama HTML templates
├── static/              # Favicons, og:image, styles
├── Dockerfile           # Multi-stage build (Rust + Debian slim)
└── Cargo.toml
```

## Deployment

### Docker (recommended)

From the repo root:

```bash
cp apps/library/.env.example apps/library/.env
# Edit .env
docker compose up -d library
```

This will start Library on port `4646` with a persistent volume for the SQLite database.

### Binary

```bash
cargo build --release -p library
```

The resulting binary at `./target/release/library` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.
