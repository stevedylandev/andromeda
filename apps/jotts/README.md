# Jotts

![cover](https://files.stevedylan.dev/jotts-demo.png)

A minimal notes app

## Quickstart

```bash
git clone https://github.com/stevedylandev/jotts.git
cd jotts
cp .env.example .env
# Edit .env with your password
cargo build --release
./target/release/jotts
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `JOTTS_PASSWORD` | Password for login authentication | `changeme` |
| `JOTTS_DB_PATH` | SQLite database file path | `jotts.sqlite` |
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

A simple, self-hosted markdown note app built with Rust. Here's a few highlights:
- Single ~7MB Rust binary with embedded assets
- Password authentication with session cookies
- Create, edit, and delete markdown notes
- Markdown rendering with strikethrough, tables, and task lists
- Dark themed UI with Commit Mono font
- SQLite for persistent storage

## Structure

```
jotts/
├── src/
│   ├── main.rs        # App entrypoint, env vars, starts server
│   ├── server.rs      # Axum router, HTTP handlers, and templates
│   ├── auth.rs        # Password verification and session management
│   └── db.rs          # SQLite database layer (notes, sessions)
├── templates/         # Askama HTML templates
│   ├── base.html      # Base layout with header and nav
│   ├── login.html     # Login page
│   ├── index.html     # Note list
│   ├── view.html      # Single note display
│   ├── new.html       # Create note form
│   └── edit.html      # Edit note form
├── static/            # Favicons, og:image, styles, and webmanifest
├── assets/            # Commit Mono font files
├── Dockerfile         # Multi-stage build (Rust + Debian slim)
└── docker-compose.yml
```

## Deployment

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/jotts.git
cd jotts
cp .env.example .env
# Edit .env with your password
docker compose up -d
```

This will start Jotts on port `3000` with a persistent volume for the SQLite database.

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/jotts` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](LICENSE)
